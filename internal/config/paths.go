package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultDataDir is the default data directory for runlite
	DefaultDataDir = "/var/lib/runlite"
)

// GetDataDir returns the configured data directory
// Uses RUNLITE_DATA_DIR env var, falls back to DefaultDataDir
func GetDataDir() string {
	if dir := os.Getenv("RUNLITE_DATA_DIR"); dir != "" {
		return dir
	}
	return DefaultDataDir
}

// GetAppDir returns the app's data directory
func GetAppDir(appName string) string {
	return filepath.Join(GetDataDir(), "apps", appName)
}

// GetAppEnvPath returns the path to the app's environment file
func GetAppEnvPath(appName string) string {
	return filepath.Join(GetAppDir(appName), "env")
}

// EnsureAppDir ensures the app directory exists
func EnsureAppDir(appName string) error {
	appDir := GetAppDir(appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	return nil
}

// ValidateAppName validates that an app name is valid
func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	// Check for path traversal
	if filepath.Clean(name) != name || filepath.IsAbs(name) {
		return fmt.Errorf("invalid app name: %s", name)
	}
	return nil
}
