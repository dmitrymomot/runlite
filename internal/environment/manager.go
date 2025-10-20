package environment

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrInvalidConfig is returned when configuration validation fails
	ErrInvalidConfig = errors.New("invalid environment configuration")
	// ErrInvalidAppName is returned when app name contains invalid characters
	ErrInvalidAppName = errors.New("invalid app name")
	// ErrInvalidKey is returned when environment variable key is invalid
	ErrInvalidKey = errors.New("invalid environment variable key")
	// ErrEnvFileNotFound is returned when environment file does not exist
	ErrEnvFileNotFound = errors.New("environment file not found")
)

var (
	// appNamePattern validates app names (alphanumeric, dash, underscore)
	appNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// keyPattern validates environment variable keys (alphanumeric and underscore only)
	keyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// Config holds the configuration for the environment manager
type Config struct {
	// BasePath is the base directory for app environment files (default: /var/lib/runlite/apps)
	BasePath string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		BasePath: "/var/lib/runlite/apps",
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("%w: base path cannot be empty", ErrInvalidConfig)
	}
	if !filepath.IsAbs(c.BasePath) {
		return fmt.Errorf("%w: base path must be absolute", ErrInvalidConfig)
	}
	return nil
}

// Manager manages environment variables for applications
// It stores variables in systemd EnvironmentFile format
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
		logger: logger.With("component", "environment"),
	}, nil
}

// WriteEnvFile writes environment variables to the app's environment file
// Uses atomic write (temp file + rename) and sets permissions to 0600
func (m *Manager) WriteEnvFile(appName string, vars map[string]string) error {
	if err := validateAppName(appName); err != nil {
		return err
	}

	for key := range vars {
		if err := validateKey(key); err != nil {
			return fmt.Errorf("invalid key %q: %w", key, err)
		}
	}

	envDir := filepath.Join(m.config.BasePath, appName)
	envFile := filepath.Join(envDir, "env")

	m.logger.Debug("writing environment file",
		"app", appName,
		"path", envFile,
		"vars_count", len(vars))

	// Ensure directory exists
	if err := os.MkdirAll(envDir, 0700); err != nil {
		m.logger.Error("failed to create environment directory",
			"app", appName,
			"path", envDir,
			"error", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Build content
	var content strings.Builder
	for key, value := range vars {
		content.WriteString(key)
		content.WriteString("=")
		content.WriteString(escapeValue(value))
		content.WriteString("\n")
	}

	// Atomic write: temp file + rename
	tempFile := envFile + ".tmp"
	if err := os.WriteFile(tempFile, []byte(content.String()), 0600); err != nil {
		m.logger.Error("failed to write temp file",
			"app", appName,
			"path", tempFile,
			"error", err)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, envFile); err != nil {
		os.Remove(tempFile)
		m.logger.Error("failed to rename temp file",
			"app", appName,
			"temp", tempFile,
			"target", envFile,
			"error", err)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	m.logger.Info("environment file written",
		"app", appName,
		"path", envFile,
		"vars_count", len(vars))

	return nil
}

// ReadEnvFile reads environment variables from the app's environment file
// Returns empty map if file doesn't exist
func (m *Manager) ReadEnvFile(appName string) (map[string]string, error) {
	if err := validateAppName(appName); err != nil {
		return nil, err
	}

	envFile := filepath.Join(m.config.BasePath, appName, "env")

	m.logger.Debug("reading environment file",
		"app", appName,
		"path", envFile)

	file, err := os.Open(envFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.logger.Debug("environment file not found, returning empty map",
				"app", appName,
				"path", envFile)
			return make(map[string]string), nil
		}
		m.logger.Error("failed to open environment file",
			"app", appName,
			"path", envFile,
			"error", err)
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			m.logger.Warn("skipping invalid line in environment file",
				"app", appName,
				"line", lineNum,
				"content", line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := unescapeValue(parts[1])

		if err := validateKey(key); err != nil {
			m.logger.Warn("skipping line with invalid key",
				"app", appName,
				"line", lineNum,
				"key", key)
			continue
		}

		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		m.logger.Error("error reading environment file",
			"app", appName,
			"path", envFile,
			"error", err)
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	m.logger.Debug("environment file read",
		"app", appName,
		"vars_count", len(vars))

	return vars, nil
}

// SetVar sets a single environment variable for the app
func (m *Manager) SetVar(appName, key, value string) error {
	if err := validateAppName(appName); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}

	m.logger.Debug("setting environment variable",
		"app", appName,
		"key", key)

	// Read existing variables
	vars, err := m.ReadEnvFile(appName)
	if err != nil {
		return err
	}

	// Update variable
	vars[key] = value

	// Write back
	return m.WriteEnvFile(appName, vars)
}

// UnsetVar removes a single environment variable for the app
func (m *Manager) UnsetVar(appName, key string) error {
	if err := validateAppName(appName); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}

	m.logger.Debug("unsetting environment variable",
		"app", appName,
		"key", key)

	// Read existing variables
	vars, err := m.ReadEnvFile(appName)
	if err != nil {
		return err
	}

	// Remove variable
	delete(vars, key)

	// Write back
	return m.WriteEnvFile(appName, vars)
}

// validateAppName checks if app name is valid and safe (prevents path traversal)
func validateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidAppName)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("%w: name contains path separators or parent directory references", ErrInvalidAppName)
	}
	if !appNamePattern.MatchString(name) {
		return fmt.Errorf("%w: name must contain only alphanumeric characters, dashes, and underscores", ErrInvalidAppName)
	}
	return nil
}

// validateKey checks if environment variable key is valid
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrInvalidKey)
	}
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: key must start with letter or underscore and contain only uppercase letters, digits, and underscores", ErrInvalidKey)
	}
	return nil
}

// escapeValue escapes a value for systemd EnvironmentFile format
// Values with spaces or special characters are quoted, quotes and backslashes are escaped
func escapeValue(value string) string {
	// If value contains spaces, tabs, newlines, or quotes, it needs to be quoted
	needsQuoting := strings.ContainsAny(value, " \t\n\r\"'")

	if needsQuoting {
		// Escape backslashes and double quotes
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return "\"" + escaped + "\""
	}

	return value
}

// unescapeValue unescapes a value from systemd EnvironmentFile format
func unescapeValue(value string) string {
	value = strings.TrimSpace(value)

	// Handle quoted values
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		// Remove surrounding quotes
		value = value[1 : len(value)-1]
		// Unescape sequences
		value = strings.ReplaceAll(value, "\\\"", "\"")
		value = strings.ReplaceAll(value, "\\\\", "\\")
	}

	return value
}
