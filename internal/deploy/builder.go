package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dmitrymomot/runlite/internal/spec"
)

const (
	// DefaultBuildTimeout is the default timeout for build operations
	DefaultBuildTimeout = 15 * time.Minute
)

// Build executes the build script and copies artifacts to the release directory.
// It validates artifacts both in the checkout directory (after build) and in the
// release directory (after copy).
func Build(ctx context.Context, s *spec.AppSpec, checkoutDir, releaseDir string) error {
	if s.Build == nil {
		return fmt.Errorf("no build configuration in spec")
	}

	if s.Build.Script == "" {
		return fmt.Errorf("no build script specified in spec")
	}

	if len(s.Build.Artifacts) == 0 {
		return fmt.Errorf("no artifacts specified in spec")
	}

	// Apply build timeout from spec or use default
	timeout := DefaultBuildTimeout
	if s.Build.Timeout > 0 {
		timeout = s.Build.Timeout
	}

	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1. Execute build script
	slog.InfoContext(ctx, "executing build script",
		"script", s.Build.Script,
		"workdir", checkoutDir,
		"timeout", timeout)

	if err := executeScript(buildCtx, s.Build.Script, checkoutDir); err != nil {
		return fmt.Errorf("build script failed: %w", err)
	}

	// 2. Validate artifacts in checkout dir (check source paths)
	slog.InfoContext(ctx, "validating artifacts in checkout directory",
		"artifacts", len(s.Build.Artifacts))

	if err := validateSourceArtifacts(checkoutDir, s.Build.Artifacts); err != nil {
		return fmt.Errorf("build validation failed: %w", err)
	}

	// 3. Copy artifacts to release dir
	slog.InfoContext(ctx, "copying artifacts to release directory",
		"dest", releaseDir)

	if err := copyArtifacts(checkoutDir, releaseDir, s.Build.Artifacts); err != nil {
		return fmt.Errorf("artifact copy failed: %w", err)
	}

	// 4. Validate artifacts in release dir (check dest paths)
	slog.InfoContext(ctx, "validating artifacts in release directory")

	if err := validateDestArtifacts(releaseDir, s.Build.Artifacts); err != nil {
		return fmt.Errorf("artifact validation in release dir failed: %w", err)
	}

	slog.InfoContext(ctx, "build completed successfully")
	return nil
}

// executeScript runs the build script in a bash shell with streaming output
func executeScript(ctx context.Context, script, workDir string) error {
	// Run script in bash shell
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Dir = workDir

	// Capture output for error messages while also streaming to logger
	var outputBuf bytes.Buffer
	multiWriter := io.MultiWriter(&outputBuf, newLogWriter(slog.LevelInfo))

	cmd.Stdout = multiWriter
	cmd.Stderr = newLogWriter(slog.LevelError)

	if err := cmd.Run(); err != nil {
		// Include captured output in error message
		output := strings.TrimSpace(outputBuf.String())
		if output != "" {
			return fmt.Errorf("%w\nOutput:\n%s", err, output)
		}
		return err
	}

	return nil
}

// logWriter is an io.Writer that writes to slog
type logWriter struct {
	level slog.Level
	buf   []byte
}

func newLogWriter(level slog.Level) *logWriter {
	return &logWriter{level: level}
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	// Accumulate bytes until we have a complete line
	w.buf = append(w.buf, p...)

	// Process complete lines
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}

		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]

		if line != "" {
			slog.Log(context.Background(), w.level, line)
		}
	}

	return len(p), nil
}

// validateSourceArtifacts checks that all source artifacts exist in the checkout directory
func validateSourceArtifacts(baseDir string, artifacts []spec.Artifact) error {
	for _, artifact := range artifacts {
		// Normalize path (remove trailing slash)
		source := strings.TrimSuffix(artifact.Source, "/")
		path := filepath.Join(baseDir, source)

		// Check if artifact exists
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("artifact %q not found in %s", source, baseDir)
			}
			return fmt.Errorf("failed to check artifact %q: %w", source, err)
		}
	}
	return nil
}

// validateDestArtifacts checks that all dest artifacts exist in the release directory
func validateDestArtifacts(baseDir string, artifacts []spec.Artifact) error {
	for _, artifact := range artifacts {
		// Normalize path (remove trailing slash)
		dest := strings.TrimSuffix(artifact.Dest, "/")
		path := filepath.Join(baseDir, dest)

		// Check if artifact exists
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("artifact %q not found in %s", dest, baseDir)
			}
			return fmt.Errorf("failed to check artifact %q: %w", dest, err)
		}
	}
	return nil
}

// copyArtifacts copies all artifacts from source to destination directory
func copyArtifacts(srcDir, dstDir string, artifacts []spec.Artifact) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, artifact := range artifacts {
		// Normalize paths (remove trailing slash for consistent handling)
		source := strings.TrimSuffix(artifact.Source, "/")
		dest := strings.TrimSuffix(artifact.Dest, "/")

		srcPath := filepath.Join(srcDir, source)
		dstPath := filepath.Join(dstDir, dest)

		// Get source info
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("failed to stat artifact %q: %w", source, err)
		}

		// Copy file or directory
		if srcInfo.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy directory %q: %w", source, err)
			}
		} else {
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory for %q: %w", dest, err)
			}

			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file %q: %w", source, err)
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Ensure data is written to disk
	return dstFile.Sync()
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
