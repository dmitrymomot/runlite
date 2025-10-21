package env

import (
	"fmt"
	"strings"
)

// Parse parses .env file content into a map of key-value pairs.
// Supports single-line values and multi-line values (certificates, JSON, etc.).
// Multi-line detection uses markers like "-----BEGIN", "{", "[", or "-----END".
func Parse(content string) (map[string]string, error) {
	vars := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentKey string
	var currentValue strings.Builder
	inMultiLine := false

	for i, line := range lines {
		// Skip empty lines and comments outside of multi-line values
		if !inMultiLine && strings.TrimSpace(line) == "" {
			continue
		}

		if !inMultiLine && strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Process new KEY=value line
		if !inMultiLine && strings.Contains(line, "=") {
			// Save any previous multi-line value before starting new one
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

			if key == "" {
				return nil, fmt.Errorf("line %d: empty key", i+1)
			}

			currentKey = key

			// Detect if this value starts a multi-line block (certificates, JSON, etc.)
			if isMultiLineStart(value) {
				inMultiLine = true
				currentValue.WriteString(value)
				currentValue.WriteString("\n")
			} else {
				// Single-line value: remove surrounding quotes if present
				vars[currentKey] = unquote(value)
				currentKey = ""
			}
		} else if inMultiLine {
			// Accumulate lines for current multi-line value
			currentValue.WriteString(line)
			currentValue.WriteString("\n")

			// Check for end markers (-----END, }, ])
			if isMultiLineEnd(line) {
				inMultiLine = false
				vars[currentKey] = strings.TrimRight(currentValue.String(), "\n")
				currentKey = ""
				currentValue.Reset()
			}
		}
	}

	// Save final value if file ends while parsing multi-line
	if currentKey != "" {
		vars[currentKey] = strings.TrimRight(currentValue.String(), "\n")
	}

	return vars, nil
}

// Format formats a map of environment variables into .env file content.
// Multi-line values are preserved as-is. Single-line values with special characters are quoted.
// Keys are sorted alphabetically for consistent output.
func Format(vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	var builder strings.Builder

	// Sort keys alphabetically for reproducible output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
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
			// Multi-line value: preserve as-is (for certificates, JSON, etc.)
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		} else if needsQuotes(value) {
			// Single-line with special chars: quote to preserve literal content
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(quote(value))
			builder.WriteString("\n")
		} else {
			// Simple alphanumeric value: no quoting needed
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// isMultiLineStart detects if a value starts a multi-line block by checking for
// markers like "-----BEGIN" (PEM certs), "{" or "[" (JSON), or "MII" (Base64).
func isMultiLineStart(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "-----BEGIN") ||
		strings.HasPrefix(trimmed, "{") ||
		strings.HasPrefix(trimmed, "[") ||
		strings.HasPrefix(trimmed, "MII")
}

// isMultiLineEnd detects the end of a multi-line block by checking for
// markers like "-----END" (PEM certs) or closing braces/brackets.
func isMultiLineEnd(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "-----END") ||
		trimmed == "}" ||
		trimmed == "]"
}

// isMultiLineValue reports whether a value contains newlines (multi-line values).
func isMultiLineValue(value string) bool {
	return strings.Contains(value, "\n")
}

// needsQuotes reports whether a value contains special shell characters
// that require quoting for safe preservation.
func needsQuotes(value string) bool {
	return strings.ContainsAny(value, " \t\"'$`\\!*?&|;<>()[]{}#")
}

// quote wraps a value in single quotes and escapes internal single quotes.
// Single quotes are preferred to avoid shell expansions.
func quote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return "'" + escaped + "'"
}

// unquote removes surrounding single or double quotes from a value.
func unquote(value string) string {
	value = strings.TrimSpace(value)

	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}

	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}
