package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/google/uuid"
)

var (
	// ErrInvalidConfig is returned when configuration validation fails
	ErrInvalidConfig = errors.New("invalid builder configuration")
	// ErrBuildFailed is returned when build command fails
	ErrBuildFailed = errors.New("build failed")
	// ErrBinaryNotFound is returned when binary not created after build
	ErrBinaryNotFound = errors.New("binary not found after build")
	// ErrGoNotInstalled is returned when go command not found
	ErrGoNotInstalled = errors.New("go is not installed")
)

// Config holds the configuration for the builder manager
type Config struct {
	// BuildScript is custom build script as string (optional, highest priority)
	BuildScript string
	// BuildScriptPath is path to build script file (optional, second priority)
	BuildScriptPath string
	// BashPath is path to bash (default: /bin/bash)
	BashPath string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		BashPath: "/bin/bash",
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.BashPath == "" {
		return fmt.Errorf("%w: bash path cannot be empty", ErrInvalidConfig)
	}
	if !filepath.IsAbs(c.BashPath) {
		return fmt.Errorf("%w: bash path must be absolute", ErrInvalidConfig)
	}
	return nil
}

// BuildOptions holds options for building an application
type BuildOptions struct {
	// SourceDir is the source directory (required)
	SourceDir string
	// AppName is the application name (required)
	AppName string
	// CommitHash is the git commit hash (optional)
	CommitHash string
}

// Manager manages application builds
type Manager struct {
	config Config
	logger *slog.Logger
}

// New creates a new Manager with the given configuration
func New(config Config, logger *slog.Logger) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Manager{
		config: config,
		logger: logger.With("component", "builder"),
	}, nil
}

// Build builds an application and returns the binary path
func (m *Manager) Build(opts BuildOptions) (string, error) {
	if opts.SourceDir == "" {
		return "", fmt.Errorf("source directory cannot be empty")
	}
	if opts.AppName == "" {
		return "", fmt.Errorf("app name cannot be empty")
	}

	// Create temp binary path
	binaryPath := filepath.Join("/tmp", fmt.Sprintf("runlite-build-%s-%s", opts.AppName, uuid.New().String()))

	m.logger.Debug("starting build",
		"app", opts.AppName,
		"source_dir", opts.SourceDir,
		"binary_path", binaryPath,
		"commit_hash", opts.CommitHash)

	// Determine build script to use
	var script string
	var err error

	if m.config.BuildScript != "" {
		// Priority 1: Use BuildScript string directly
		script = m.config.BuildScript
		m.logger.Debug("using inline build script")
	} else if m.config.BuildScriptPath != "" {
		// Priority 2: Read BuildScriptPath
		script, err = m.readScriptFile(m.config.BuildScriptPath)
		if err != nil {
			return "", err
		}
		m.logger.Debug("using build script from file", "path", m.config.BuildScriptPath)
	} else {
		// Priority 3: Default go build
		if err := m.checkGoInstalled(); err != nil {
			return "", err
		}
		script = fmt.Sprintf("go build -o %s", binaryPath)
		m.logger.Debug("using default go build")
	}

	// Execute template substitution
	processedScript, err := m.processTemplate(script, opts, binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to process template: %w", err)
	}

	// Execute build script
	output, err := m.executeScript(processedScript, opts.SourceDir)
	if err != nil {
		m.logger.Error("build failed",
			"app", opts.AppName,
			"error", err,
			"output", string(output))
		return "", fmt.Errorf("%w: %s", ErrBuildFailed, string(output))
	}

	m.logger.Debug("build command executed", "output", string(output))

	// Verify binary was created
	if _, err := os.Stat(binaryPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.logger.Error("binary not found after build",
				"app", opts.AppName,
				"binary_path", binaryPath)
			return "", ErrBinaryNotFound
		}
		return "", fmt.Errorf("failed to check binary: %w", err)
	}

	m.logger.Info("build completed successfully",
		"app", opts.AppName,
		"binary_path", binaryPath)

	return binaryPath, nil
}

// readScriptFile reads a build script from file
func (m *Manager) readScriptFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		m.logger.Error("failed to read build script file",
			"path", path,
			"error", err)
		return "", fmt.Errorf("failed to read build script: %w", err)
	}
	return string(data), nil
}

// checkGoInstalled checks if Go is installed
func (m *Manager) checkGoInstalled() error {
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		m.logger.Error("go command not found")
		return ErrGoNotInstalled
	}
	return nil
}

// processTemplate processes template variables in the script
func (m *Manager) processTemplate(script string, opts BuildOptions, binaryPath string) (string, error) {
	tmpl, err := template.New("build").Parse(script)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]string{
		"SourceDir":  opts.SourceDir,
		"BinaryPath": binaryPath,
		"AppName":    opts.AppName,
		"CommitHash": opts.CommitHash,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// executeScript executes a script via bash with working directory set to sourceDir
func (m *Manager) executeScript(script, sourceDir string) ([]byte, error) {
	cmd := exec.Command(m.config.BashPath, "-c", script)
	cmd.Dir = sourceDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	m.logger.Debug("executing build script",
		"working_dir", sourceDir,
		"bash_path", m.config.BashPath)

	err := cmd.Run()
	return output.Bytes(), err
}
