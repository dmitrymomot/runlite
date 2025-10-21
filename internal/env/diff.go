package env

import (
	"fmt"
	"strings"
)

// DiffResult represents the differences between two sets of environment variables
type DiffResult struct {
	Added    map[string]string
	Modified map[string]string
	Removed  map[string]string
}

// Diff compares two sets of environment variables and returns the differences
func Diff(old, new map[string]string) *DiffResult {
	result := &DiffResult{
		Added:    make(map[string]string),
		Modified: make(map[string]string),
		Removed:  make(map[string]string),
	}

	// Find added and modified
	for key, newValue := range new {
		if oldValue, exists := old[key]; exists {
			if oldValue != newValue {
				result.Modified[key] = newValue
			}
		} else {
			result.Added[key] = newValue
		}
	}

	// Find removed
	for key, value := range old {
		if _, exists := new[key]; !exists {
			result.Removed[key] = value
		}
	}

	return result
}

// IsEmpty returns true if there are no changes
func (d *DiffResult) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Modified) == 0 && len(d.Removed) == 0
}

// Format formats the diff result as a human-readable string
func (d *DiffResult) Format() string {
	if d.IsEmpty() {
		return "No changes"
	}

	var builder strings.Builder

	// Sort keys for consistent output
	sortKeys := func(m map[string]string) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
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
		return keys
	}

	// Added variables
	if len(d.Added) > 0 {
		for _, key := range sortKeys(d.Added) {
			value := d.Added[key]
			builder.WriteString(fmt.Sprintf("  + %s=%s\n", key, formatValue(value)))
		}
	}

	// Modified variables
	if len(d.Modified) > 0 {
		for _, key := range sortKeys(d.Modified) {
			value := d.Modified[key]
			builder.WriteString(fmt.Sprintf("  ~ %s=%s\n", key, formatValue(value)))
		}
	}

	// Removed variables
	if len(d.Removed) > 0 {
		for _, key := range sortKeys(d.Removed) {
			builder.WriteString(fmt.Sprintf("  - %s\n", key))
		}
	}

	return builder.String()
}

// formatValue formats a value for display (truncate long values)
func formatValue(value string) string {
	const maxLen = 60

	// Handle multi-line values
	if strings.Contains(value, "\n") {
		lines := strings.Split(value, "\n")
		if len(lines) > 1 {
			return fmt.Sprintf("%s... (%d lines)", lines[0], len(lines))
		}
	}

	// Truncate long single-line values
	if len(value) > maxLen {
		return value[:maxLen] + "..."
	}

	return value
}
