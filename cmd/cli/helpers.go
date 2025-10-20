package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// outputJSON outputs data as JSON to stdout
//
//nolint:unused // Will be used by future commands
func outputJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// formatStatus formats a status string with visual indicators
//
//nolint:unused // Will be used by future commands
func formatStatus(status string) string {
	switch status {
	case "running":
		return "● Running"
	case "stopped":
		return "○ Stopped"
	case "deploying":
		return "⏳ Deploying"
	case "failed":
		return "✗ Failed"
	default:
		return status
	}
}

// formatTimeAgo formats a time as a relative "time ago" string
//
//nolint:unused // Will be used by future commands
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}

// formatTime formats a time as a readable string
//
//nolint:unused // Will be used by future commands
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatBytes formats bytes as a human-readable size
//
//nolint:unused // Will be used by future commands
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
