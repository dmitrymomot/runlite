package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

// Color functions for different message types
var (
	ColorSuccess = color.New(color.FgGreen).SprintFunc()
	ColorError   = color.New(color.FgRed).SprintFunc()
	ColorWarning = color.New(color.FgYellow).SprintFunc()
	ColorInfo    = color.New(color.FgCyan).SprintFunc()
	ColorGray    = color.New(color.FgHiBlack).SprintFunc()
	ColorBold    = color.New(color.Bold).SprintFunc()
)

// PrintSuccess prints a success message with a checkmark in green.
func PrintSuccess(msg string) {
	color.Green("✓ %s", msg)
}

// PrintError prints an error message with an X mark in red.
func PrintError(msg string) {
	color.Red("✗ %s", msg)
}

// PrintWarning prints a warning message with a warning symbol in yellow.
func PrintWarning(msg string) {
	color.Yellow("⚠ %s", msg)
}

// PrintInfo prints an info message with an arrow in cyan.
func PrintInfo(msg string) {
	color.Cyan("→ %s", msg)
}

// ErrorString returns a formatted error message string suitable for cli.Exit.
func ErrorString(msg string) string {
	return color.RedString("✗ %s", msg)
}

// SuccessString returns a formatted success message string.
func SuccessString(msg string) string {
	return color.GreenString("✓ %s", msg)
}

// WarningString returns a formatted warning message string.
func WarningString(msg string) string {
	return color.YellowString("⚠ %s", msg)
}

// InfoString returns a formatted info message string.
func InfoString(msg string) string {
	return color.CyanString("→ %s", msg)
}

// OutputJSON encodes data as JSON with indentation and writes to stdout.
func OutputJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// FormatStatus maps status strings to display strings with visual indicators.
func FormatStatus(status string) string {
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

// FormatTimeAgo returns a human-readable "time ago" string (e.g., "5m ago", "2h ago").
// Returns "-" for zero time values.
func FormatTimeAgo(t time.Time) string {
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

// FormatTime returns a formatted time string in "YYYY-MM-DD HH:MM:SS" format.
// Returns "-" for zero time values.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatBytes returns a human-readable byte size string (e.g., "1.5 MB", "256 KB").
func FormatBytes(bytes int64) string {
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
