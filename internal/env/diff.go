package env

import (
	"fmt"
	"strings"
)

// DiffResult represents the differences between two sets of environment variables.
// Added, Modified, and Removed track new, changed, and deleted variables respectively.
type DiffResult struct {
	Added    map[string]string
	Modified map[string]string
	Removed  map[string]string
}

// Diff compares two sets of environment variables and returns a DiffResult with categorized changes.
func Diff(old, new map[string]string) *DiffResult {
	result := &DiffResult{
		Added:    make(map[string]string),
		Modified: make(map[string]string),
		Removed:  make(map[string]string),
	}

	// Categorize new/modified variables
	for key, newValue := range new {
		if oldValue, exists := old[key]; exists {
			if oldValue != newValue {
				result.Modified[key] = newValue
			}
		} else {
			result.Added[key] = newValue
		}
	}

	// Categorize removed variables
	for key, value := range old {
		if _, exists := new[key]; !exists {
			result.Removed[key] = value
		}
	}

	return result
}

// IsEmpty reports whether the diff contains any changes.
func (d *DiffResult) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Modified) == 0 && len(d.Removed) == 0
}

// Format returns a human-readable string representation of the diff.
// Added variables are prefixed with "+", modified with "~", and removed with "-".
// Long values are truncated for readability.
func (d *DiffResult) Format() string {
	if d.IsEmpty() {
		return "No changes"
	}

	var builder strings.Builder

	// Helper to sort keys alphabetically
	sortKeys := func(m map[string]string) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		return keys
	}

	// Format added variables with "+" prefix
	if len(d.Added) > 0 {
		for _, key := range sortKeys(d.Added) {
			value := d.Added[key]
			builder.WriteString(fmt.Sprintf("  + %s=%s\n", key, formatValue(value)))
		}
	}

	// Format modified variables with "~" prefix
	if len(d.Modified) > 0 {
		for _, key := range sortKeys(d.Modified) {
			value := d.Modified[key]
			builder.WriteString(fmt.Sprintf("  ~ %s=%s\n", key, formatValue(value)))
		}
	}

	// Format removed variables with "-" prefix
	if len(d.Removed) > 0 {
		for _, key := range sortKeys(d.Removed) {
			builder.WriteString(fmt.Sprintf("  - %s\n", key))
		}
	}

	return builder.String()
}

// formatValue returns a display-friendly value string, truncating long values and
// summarizing multi-line values to prevent verbose output.
func formatValue(value string) string {
	const maxLen = 60

	// Multi-line values: show first line and line count
	if strings.Contains(value, "\n") {
		lines := strings.Split(value, "\n")
		if len(lines) > 1 {
			return fmt.Sprintf("%s... (%d lines)", lines[0], len(lines))
		}
	}

	// Single-line values: truncate if too long
	if len(value) > maxLen {
		return value[:maxLen] + "..."
	}

	return value
}
