package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmitrymomot/runlite/internal/config"
)

// Load reads environment variables from the app's env file
func Load(appName string) (map[string]string, error) {
	if err := config.ValidateAppName(appName); err != nil {
		return nil, err
	}

	envPath := config.GetAppEnvPath(appName)

	// If file doesn't exist, return empty map
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

// Save writes environment variables to the app's env file atomically
func Save(appName string, vars map[string]string) error {
	if err := config.ValidateAppName(appName); err != nil {
		return err
	}

	// Ensure app directory exists
	if err := config.EnsureAppDir(appName); err != nil {
		return err
	}

	envPath := config.GetAppEnvPath(appName)
	content := Format(vars)

	// Write atomically using temp file + rename
	tmpFile := envPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("write temp env file: %w", err)
	}

	if err := os.Rename(tmpFile, envPath); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("rename temp env file: %w", err)
	}

	return nil
}

// Get retrieves a single environment variable value
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

// Set sets one or more environment variables
func Set(appName string, updates map[string]string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	// Merge updates
	for key, value := range updates {
		vars[key] = value
	}

	return Save(appName, vars)
}

// Unset removes one or more environment variables
func Unset(appName string, keys []string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	// Remove keys
	for _, key := range keys {
		delete(vars, key)
	}

	return Save(appName, vars)
}

// Import imports environment variables from a file
func Import(appName string, filePath string, merge bool) error {
	// Read import file
	var content []byte
	var err error

	if filePath == "-" {
		// Read from stdin
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
		// Merge with existing vars
		return Set(appName, importedVars)
	}

	// Replace all vars
	return Save(appName, importedVars)
}

// Export exports environment variables to a file
func Export(appName string, filePath string) error {
	vars, err := Load(appName)
	if err != nil {
		return err
	}

	content := Format(vars)

	if filePath == "" || filePath == "-" {
		// Write to stdout
		fmt.Print(content)
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	return nil
}
