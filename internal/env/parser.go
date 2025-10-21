package env

import (
	"fmt"
	"strings"
)

// Parse parses .env file content into a map of key-value pairs
func Parse(content string) (map[string]string, error) {
	vars := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentKey string
	var currentValue strings.Builder
	inMultiLine := false

	for i, line := range lines {
		// Skip empty lines when not in multi-line mode
		if !inMultiLine && strings.TrimSpace(line) == "" {
			continue
		}

		// Skip comments when not in multi-line mode
		if !inMultiLine && strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Check for KEY=value start
		if !inMultiLine && strings.Contains(line, "=") {
			// Save previous multi-line value if any
			if currentKey != "" {
				vars[currentKey] = strings.TrimRight(currentValue.String(), "\n")
				currentValue.Reset()
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 0 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}

			// Validate key
			if key == "" {
				return nil, fmt.Errorf("line %d: empty key", i+1)
			}

			currentKey = key

			// Check if value starts a multi-line block
			if isMultiLineStart(value) {
				inMultiLine = true
				currentValue.WriteString(value)
				currentValue.WriteString("\n")
			} else {
				// Single line value - remove quotes if present
				vars[currentKey] = unquote(value)
				currentKey = ""
			}
		} else if inMultiLine {
			// Continue multi-line value
			currentValue.WriteString(line)
			currentValue.WriteString("\n")

			// Check for end of multi-line block
			if isMultiLineEnd(line) {
				inMultiLine = false
				vars[currentKey] = strings.TrimRight(currentValue.String(), "\n")
				currentKey = ""
				currentValue.Reset()
			}
		}
	}

	// Save final multi-line value if any
	if currentKey != "" {
		vars[currentKey] = strings.TrimRight(currentValue.String(), "\n")
	}

	return vars, nil
}

// Format formats a map of environment variables into .env file content
func Format(vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	var builder strings.Builder

	// Sort keys for consistent output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	// Simple alphabetical sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := vars[key]
		if isMultiLineValue(value) {
			// Multi-line value - write as-is
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		} else if needsQuotes(value) {
			// Single line with special chars - quote it
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(quote(value))
			builder.WriteString("\n")
		} else {
			// Simple value
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// isMultiLineStart detects if a value starts a multi-line block
func isMultiLineStart(value string) bool {
	trimmed := strings.TrimSpace(value)
	// Detect common multi-line patterns
	return strings.HasPrefix(trimmed, "-----BEGIN") ||
		strings.HasPrefix(trimmed, "{") ||
		strings.HasPrefix(trimmed, "[") ||
		strings.HasPrefix(trimmed, "MII") // Base64 cert/key
}

// isMultiLineEnd detects if a line ends a multi-line block
func isMultiLineEnd(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Detect end markers
	return strings.HasPrefix(trimmed, "-----END") ||
		trimmed == "}" ||
		trimmed == "]"
}

// isMultiLineValue checks if a value contains newlines
func isMultiLineValue(value string) bool {
	return strings.Contains(value, "\n")
}

// needsQuotes checks if a value needs to be quoted
func needsQuotes(value string) bool {
	// Quote if contains spaces or special shell characters
	return strings.ContainsAny(value, " \t\"'$`\\!*?&|;<>()[]{}#")
}

// quote adds quotes to a value
func quote(value string) string {
	// Use single quotes to avoid most escaping issues
	// Escape single quotes within the value
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return "'" + escaped + "'"
}

// unquote removes surrounding quotes from a value
func unquote(value string) string {
	value = strings.TrimSpace(value)

	// Remove double quotes
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}

	// Remove single quotes
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}
