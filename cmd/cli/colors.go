package main

import "github.com/fatih/color"

//nolint:unused // Will be used by command implementations
var (
	// Color functions for different message types
	colorSuccess = color.New(color.FgGreen).SprintFunc()
	colorError   = color.New(color.FgRed).SprintFunc()
	colorWarning = color.New(color.FgYellow).SprintFunc()
	colorInfo    = color.New(color.FgCyan).SprintFunc()
	colorGray    = color.New(color.FgHiBlack).SprintFunc()
	colorBold    = color.New(color.Bold).SprintFunc()
)

// printSuccess prints a success message with green color
//
//nolint:unused // Will be used by command implementations
func printSuccess(msg string) {
	color.Green("✓ %s", msg)
}

// printError prints an error message with red color
//
//nolint:unused // Will be used by command implementations
func printError(msg string) {
	color.Red("✗ %s", msg)
}

// printWarning prints a warning message with yellow color
//
//nolint:unused // Will be used by command implementations
func printWarning(msg string) {
	color.Yellow("⚠ %s", msg)
}

// printInfo prints an info message with cyan color
//
//nolint:unused // Will be used by command implementations
func printInfo(msg string) {
	color.Cyan("→ %s", msg)
}
