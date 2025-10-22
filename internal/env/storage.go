package env

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/dmitrymomot/runlite/internal/config"
)

// Load reads environment variables from the app's env file.
// Returns an empty map if the file doesn't exist.
func Load(appName string) (map[string]string, error) {
	if err := config.ValidateAppName(appName); err != nil {
		return nil, err
	}

	envPath := config.GetAppEnvPath(appName)

	// File doesn't exist: return empty map (not an error)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	vars, err := Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse env file: %w", err)
	}

	return vars, nil
}

// Save writes environment variables to the app's env file atomically using a temp file + rename strategy.
// This ensures consistency and prevents corruption during writes.
func Save(appName string, vars map[string]string) error {
	if err := config.ValidateAppName(appName); err != nil {
		return err
	}

	if err := config.EnsureAppDir(appName); err != nil {
		return err
	}

	envPath := config.GetAppEnvPath(appName)
	content := Format(vars)

	// Write to temp file first, then atomically rename to target location
	tmpFile := envPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write temp env file: %w", err)
	}

	if err := os.Rename(tmpFile, envPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("rename temp env file: %w", err)
	}

	return nil
}

// Get retrieves a single environment variable value by key.
func Get(appName, key string) (string, error) {
	vars, err := Load(appName)
	if err != nil {
		return "", err
	}

	value, ok := vars[key]
	if !ok {
		return "", fmt.Errorf("variable %q not found", key)
	}

	return value, nil
}

// Set merges the provided updates into the existing environment variables and saves them.
func Set(appName string, updates map[string]string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	maps.Copy(vars, updates)

	return Save(appName, vars)
}

// Unset removes the specified keys from the environment variables and saves them.
func Unset(appName string, keys []string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	for _, key := range keys {
		delete(vars, key)
	}

	return Save(appName, vars)
}

// Import loads variables from a file and either merges or replaces existing variables.
// The filePath "-" means read from stdin.
func Import(appName string, filePath string, merge bool) error {
	var content []byte
	var err error

	if filePath == "-" {
		content, err = os.ReadFile("/dev/stdin")
	} else {
		content, err = os.ReadFile(filePath)
	}

	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}

	importedVars, err := Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse import file: %w", err)
	}

	if merge {
		return Set(appName, importedVars)
	}

	return Save(appName, importedVars)
}

// Export writes environment variables to a file or stdout.
// The filePath "" or "-" means write to stdout.
func Export(appName string, filePath string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	content := Format(vars)

	if filePath == "" || filePath == "-" {
		fmt.Print(content)
		return nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	return nil
}
