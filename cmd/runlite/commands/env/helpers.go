package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dmitrymomot/runlite/internal/config"
)

// EnvVar represents a single environment variable
type EnvVar struct {
	Key   string
	Value string
}

var (
	// envKeyPattern validates environment variable keys (A-Z, 0-9, _ only)
	envKeyPattern = regexp.MustCompile(`^[A-Z0-9_]+$`)
)

// getEnvFilePath returns the path to an app's environment file
func getEnvFilePath(appName string) string {
	return config.GetAppEnvPath(appName)
}

// validateEnvKey validates an environment variable key
func validateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if strings.Contains(key, " ") {
		return fmt.Errorf("key cannot contain spaces: %s", key)
	}

	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("key must contain only uppercase letters, numbers, and underscores: %s", key)
	}

	return nil
}

// parseEnvLine parses a single KEY=VALUE line
func parseEnvLine(line string) (key, value string, err error) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", nil
	}

	// Split on first =
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format (expected KEY=VALUE): %s", line)
	}

	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])

	// Validate key
	if err := validateEnvKey(key); err != nil {
		return "", "", err
	}

	return key, value, nil
}

// readEnvFile reads environment variables from a file
func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil // Empty env is valid
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		key, value, err := parseEnvLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// Skip empty lines and comments
		if key == "" {
			continue
		}

		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return vars, nil
}

// writeEnvFile writes environment variables to a file
func writeEnvFile(path string, vars map[string]string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Create temp file for atomic write
	tempPath := path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	// Write variables (sorted by key for consistency)
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	// Sort keys for deterministic output
	// (using simple bubble sort to avoid importing sort package)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, vars[key]); err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("write variable: %w", err)
		}
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// formatEnvVars formats environment variables as a string (KEY=VALUE per line)
func formatEnvVars(vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	// Sort keys
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Build output
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, vars[key]))
	}

	return builder.String()
}

// parseEnvContent parses multiline KEY=VALUE content
func parseEnvContent(content string) (map[string]string, error) {
	vars := make(map[string]string)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		key, value, err := parseEnvLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}

		// Skip empty lines and comments
		if key == "" {
			continue
		}

		vars[key] = value
	}

	return vars, nil
}

// validateAppExists checks if an app exists
func validateAppExists(appName string) error {
	appDir := config.GetAppDir(appName)

	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return fmt.Errorf("app '%s' does not exist", appName)
	}

	return nil
}
