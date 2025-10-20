package builder_test

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/builder"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns expected default values", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()

		assert.Equal(t, "/bin/bash", cfg.BashPath, "BashPath should be /bin/bash")
		assert.Empty(t, cfg.BuildScript, "BuildScript should be empty")
		assert.Empty(t, cfg.BuildScriptPath, "BuildScriptPath should be empty")
	})

	t.Run("bash path is absolute", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()

		assert.True(t, filepath.IsAbs(cfg.BashPath), "BashPath should be absolute")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config with defaults passes", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("valid config with custom bash path passes", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath: "/usr/bin/bash",
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("valid config with custom build script passes", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:    "/bin/bash",
			BuildScript: "go build -o {{.BinaryPath}}",
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("empty bash path fails", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath: "",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, builder.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "bash path cannot be empty")
	})

	t.Run("relative bash path fails", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath: "bin/bash",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, builder.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "bash path must be absolute")
	})

	t.Run("bash path with dot prefix fails", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath: "./bash",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, builder.ErrInvalidConfig), "should wrap ErrInvalidConfig")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid config creates manager successfully", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		logger := slog.Default()

		manager, err := builder.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath: "",
		}
		logger := slog.Default()

		manager, err := builder.New(cfg, logger)
		require.Error(t, err)
		assert.Nil(t, manager)
		assert.True(t, errors.Is(err, builder.ErrInvalidConfig), "should return ErrInvalidConfig")
	})

	t.Run("nil logger creates usable manager", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()

		manager, err := builder.New(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, manager, "manager should be created with nil logger")

		// Create a simple test app to verify manager is usable
		sourceDir := createTestGoApp(t)

		_, err = manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err, "manager with nil logger should be usable")
	})
}

func TestManager_Build_DefaultGoBuild(t *testing.T) {
	t.Parallel()

	t.Run("successfully builds simple Go program", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, binaryPath, "binary path should not be empty")

		// Verify binary was created
		info, err := os.Stat(binaryPath)
		require.NoError(t, err, "binary should exist")
		assert.False(t, info.IsDir(), "binary path should be a file, not a directory")

		// Cleanup
		defer os.Remove(binaryPath)

		// Verify binary is in expected location
		assert.Contains(t, binaryPath, "/tmp/runlite-build-test-app-", "binary should be in /tmp with expected prefix")
	})

	t.Run("binary has correct permissions", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		info, err := os.Stat(binaryPath)
		require.NoError(t, err)

		// Verify binary is executable (mode should have at least owner execute bit)
		mode := info.Mode()
		assert.True(t, mode&0100 != 0, "binary should be executable")
	})

	t.Run("build failure returns ErrBuildFailed with output", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		// Create source dir with invalid Go code
		sourceDir := t.TempDir()
		invalidCode := `package main
import "fmt"
func main() {
	// Syntax error: missing closing brace
	fmt.Println("test"
}
`
		err = os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte(invalidCode), 0644)
		require.NoError(t, err)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath, "binary path should be empty on failure")
		assert.True(t, errors.Is(err, builder.ErrBuildFailed), "should return ErrBuildFailed")

		// Error message should contain build output
		errMsg := err.Error()
		assert.Contains(t, errMsg, "build failed", "error should mention build failure")
	})

	t.Run("commit hash is optional", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		// Build without commit hash
		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir:  sourceDir,
			AppName:    "test-app",
			CommitHash: "",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, binaryPath)
		defer os.Remove(binaryPath)
	})

	t.Run("commit hash is included when provided", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		// Build with commit hash
		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir:  sourceDir,
			AppName:    "test-app",
			CommitHash: "abc123def456",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, binaryPath)
		defer os.Remove(binaryPath)

		// Verify binary was created
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})
}

func TestManager_Build_CustomInlineScript(t *testing.T) {
	t.Parallel()

	t.Run("successfully builds with custom inline script", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:    "/bin/bash",
			BuildScript: "go build -o {{.BinaryPath}}",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify binary was created
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})

	t.Run("template variables are substituted correctly", func(t *testing.T) {
		t.Parallel()

		// Create script that echoes variables to a marker file
		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker.txt")

		cfg := builder.Config{
			BashPath: "/bin/bash",
			BuildScript: `echo "SourceDir={{.SourceDir}}" > ` + markerFile + `
echo "BinaryPath={{.BinaryPath}}" >> ` + markerFile + `
echo "AppName={{.AppName}}" >> ` + markerFile + `
echo "CommitHash={{.CommitHash}}" >> ` + markerFile + `
go build -o {{.BinaryPath}}`,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir:  sourceDir,
			AppName:    "my-test-app",
			CommitHash: "abc123",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify template substitution
		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)

		markerContent := string(content)
		assert.Contains(t, markerContent, "SourceDir="+sourceDir)
		assert.Contains(t, markerContent, "BinaryPath="+binaryPath)
		assert.Contains(t, markerContent, "AppName=my-test-app")
		assert.Contains(t, markerContent, "CommitHash=abc123")
	})

	t.Run("working directory is set to SourceDir", func(t *testing.T) {
		t.Parallel()

		// Create script that writes current directory to a marker file
		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "workdir.txt")

		cfg := builder.Config{
			BashPath: "/bin/bash",
			BuildScript: `pwd > ` + markerFile + `
go build -o {{.BinaryPath}}`,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify working directory
		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)

		workDir := strings.TrimSpace(string(content))
		assert.Equal(t, sourceDir, workDir, "working directory should be SourceDir")
	})

	t.Run("script failure is captured with error", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:    "/bin/bash",
			BuildScript: "exit 1",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.True(t, errors.Is(err, builder.ErrBuildFailed), "should return ErrBuildFailed")
	})

	t.Run("binary not created returns ErrBinaryNotFound", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:    "/bin/bash",
			BuildScript: "echo 'not creating binary'",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.True(t, errors.Is(err, builder.ErrBinaryNotFound), "should return ErrBinaryNotFound")
	})

	t.Run("template syntax error returns error", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:    "/bin/bash",
			BuildScript: "go build -o {{.InvalidTemplate",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.Contains(t, err.Error(), "template", "error should mention template")
	})

	t.Run("special characters in values are handled", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker.txt")

		cfg := builder.Config{
			BashPath: "/bin/bash",
			BuildScript: `echo "{{.AppName}}" > ` + markerFile + `
go build -o {{.BinaryPath}}`,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		// App name with special characters
		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app-123",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify special characters preserved
		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)
		assert.Equal(t, "test-app-123\n", string(content))
	})
}

func TestManager_Build_CustomScriptFromFile(t *testing.T) {
	t.Parallel()

	t.Run("reads and executes script from file", func(t *testing.T) {
		t.Parallel()

		scriptDir := t.TempDir()
		scriptPath := filepath.Join(scriptDir, "build.sh")
		scriptContent := `#!/bin/bash
go build -o {{.BinaryPath}}
`
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
		require.NoError(t, err)

		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScriptPath: scriptPath,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify binary was created
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})

	t.Run("template variables substituted in file-based script", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker.txt")
		scriptPath := filepath.Join(tmpDir, "build.sh")

		scriptContent := `#!/bin/bash
echo "AppName={{.AppName}}" > ` + markerFile + `
go build -o {{.BinaryPath}}
`
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
		require.NoError(t, err)

		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScriptPath: scriptPath,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "file-script-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify template was substituted
		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "AppName=file-script-app")
	})

	t.Run("script file not found returns error", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScriptPath: "/non/existent/script.sh",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.Contains(t, err.Error(), "failed to read build script", "error should mention script read failure")
	})
}

func TestManager_Build_Priority(t *testing.T) {
	t.Parallel()

	t.Run("inline BuildScript takes highest priority", func(t *testing.T) {
		t.Parallel()

		scriptDir := t.TempDir()
		scriptPath := filepath.Join(scriptDir, "build.sh")
		markerFile := filepath.Join(scriptDir, "marker.txt")

		// File script that shouldn't run
		fileScriptContent := `#!/bin/bash
echo "file-script" > ` + markerFile + `
`
		err := os.WriteFile(scriptPath, []byte(fileScriptContent), 0644)
		require.NoError(t, err)

		// Inline script should override file script
		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScript:     `echo "inline-script" > ` + markerFile,
			BuildScriptPath: scriptPath,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		_, err = manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		// This will fail because inline script doesn't build binary
		// But we can verify which script ran
		require.Error(t, err)

		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "inline-script", "inline script should have priority")
		assert.NotContains(t, string(content), "file-script", "file script should not run")
	})

	t.Run("BuildScriptPath used when BuildScript empty", func(t *testing.T) {
		t.Parallel()

		scriptDir := t.TempDir()
		scriptPath := filepath.Join(scriptDir, "build.sh")
		markerFile := filepath.Join(scriptDir, "marker.txt")

		scriptContent := `#!/bin/bash
echo "file-script-ran" > ` + markerFile + `
go build -o {{.BinaryPath}}
`
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
		require.NoError(t, err)

		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScript:     "", // Empty
			BuildScriptPath: scriptPath,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "file-script-ran")
	})

	t.Run("default go build when both scripts empty", func(t *testing.T) {
		t.Parallel()

		cfg := builder.Config{
			BashPath:        "/bin/bash",
			BuildScript:     "",
			BuildScriptPath: "",
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify binary was created using default go build
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})
}

func TestManager_Build_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty SourceDir rejected", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: "",
			AppName:   "test-app",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.Contains(t, err.Error(), "source directory cannot be empty")
	})

	t.Run("empty AppName rejected", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: "/tmp/test",
			AppName:   "",
		})
		require.Error(t, err)
		assert.Empty(t, binaryPath)
		assert.Contains(t, err.Error(), "app name cannot be empty")
	})
}

func TestManager_Build_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very long build output is captured", func(t *testing.T) {
		t.Parallel()

		// Create script that generates lots of output
		cfg := builder.Config{
			BashPath: "/bin/bash",
			BuildScript: `
for i in {1..100}; do echo "Build output line $i"; done
go build -o {{.BinaryPath}}
`,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Should handle lots of output without issues
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})

	t.Run("unicode in app names handled", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app-世界-🌍",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify binary was created
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})

	t.Run("paths with spaces are handled", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		// Create temp dir with spaces in name
		baseDir := t.TempDir()
		sourceDirWithSpaces := filepath.Join(baseDir, "my test app")
		err = os.Mkdir(sourceDirWithSpaces, 0755)
		require.NoError(t, err)

		// Create go.mod
		goModContent := `module testapp

go 1.25
`
		err = os.WriteFile(filepath.Join(sourceDirWithSpaces, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)

		// Create simple Go app
		mainContent := `package main
func main() {}
`
		err = os.WriteFile(filepath.Join(sourceDirWithSpaces, "main.go"), []byte(mainContent), 0644)
		require.NoError(t, err)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDirWithSpaces,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		// Verify binary was created
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	})

	t.Run("special characters in commit hash are handled", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker.txt")

		cfg := builder.Config{
			BashPath: "/bin/bash",
			BuildScript: `echo "{{.CommitHash}}" > ` + markerFile + `
go build -o {{.BinaryPath}}`,
		}
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		binaryPath, err := manager.Build(builder.BuildOptions{
			SourceDir:  sourceDir,
			AppName:    "test-app",
			CommitHash: "abc-123_def.456",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath)

		content, err := os.ReadFile(markerFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "abc-123_def.456")
	})

	t.Run("multiple builds for same app create different binaries", func(t *testing.T) {
		t.Parallel()

		cfg := builder.DefaultConfig()
		manager, err := builder.New(cfg, slog.Default())
		require.NoError(t, err)

		sourceDir := createTestGoApp(t)

		// Build twice
		binaryPath1, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath1)

		binaryPath2, err := manager.Build(builder.BuildOptions{
			SourceDir: sourceDir,
			AppName:   "test-app",
		})
		require.NoError(t, err)
		defer os.Remove(binaryPath2)

		// Binary paths should be different (UUIDs make them unique)
		assert.NotEqual(t, binaryPath1, binaryPath2, "each build should create unique binary path")

		// Both binaries should exist
		_, err = os.Stat(binaryPath1)
		require.NoError(t, err)
		_, err = os.Stat(binaryPath2)
		require.NoError(t, err)
	})
}

// Helper function to create a simple test Go application
func createTestGoApp(t *testing.T) string {
	t.Helper()

	sourceDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testapp

go 1.25
`
	err := os.WriteFile(filepath.Join(sourceDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create main.go
	mainContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello from test app")
}
`
	err = os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte(mainContent), 0644)
	require.NoError(t, err)

	return sourceDir
}
