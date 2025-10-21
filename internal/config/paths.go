package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultDataDir is the default data directory for runlite.
	DefaultDataDir = "/var/lib/runlite"
)

// GetDataDir returns the configured data directory, preferring the RUNLITE_DATA_DIR
// environment variable if set, otherwise using DefaultDataDir.
func GetDataDir() string {
	if dir := os.Getenv("RUNLITE_DATA_DIR"); dir != "" {
		return dir
	}
	return DefaultDataDir
}

// GetAppDir returns the directory path where app-specific data is stored.
func GetAppDir(appName string) string {
	return filepath.Join(GetDataDir(), "apps", appName)
}

// GetAppEnvPath returns the path to the app's environment variables file.
func GetAppEnvPath(appName string) string {
	return filepath.Join(GetAppDir(appName), "env")
}

// EnsureAppDir creates the app directory if it doesn't exist.
func EnsureAppDir(appName string) error {
	appDir := GetAppDir(appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	return nil
}

// ValidateAppName checks that an app name is safe: non-empty, not absolute, and no path traversal attempts.
func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	// Reject absolute paths and path traversal attempts
	if filepath.Clean(name) != name || filepath.IsAbs(name) {
		return fmt.Errorf("invalid app name: %s", name)
	}
	return nil
}
