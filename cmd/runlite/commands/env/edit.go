package env

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit environment variables in a text editor",
		Long: `Edit environment variables using a multiline text editor.

This is the MAIN and RECOMMENDED way to manage environment variables.

Format:
  KEY=VALUE
  ANOTHER_KEY=another value

Rules:
  • One variable per line
  • Format: KEY=VALUE
  • Keys must be uppercase letters, numbers, and underscores only
  • Empty lines and lines starting with # are ignored
  • No spaces in keys

The editor supports:
  • Multiple lines of text
  • Copy/paste
  • Standard keyboard navigation

Examples:
  # Edit variables for an app
  runlite env myapp edit

  # After opening, you can edit the full environment:
  PORT=8000
  DB_HOST=localhost
  DB_NAME=myapp_production
  API_KEY=abc123def456`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			return runEdit(cmd, appName)
		},
	}

	return cmd
}

func runEdit(cmd *cobra.Command, appName string) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Read current environment
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Format current variables as text
	content := formatEnvVars(vars)

	// Create interactive text editor
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title(fmt.Sprintf("Edit Environment Variables: %s", appName)).
				Description("Format: KEY=VALUE (one per line). Empty lines and # comments are ignored.").
				CharLimit(100000).
				Value(&content).
				Lines(20),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return fmt.Errorf("edit cancelled")
	}

	// Parse and validate the content
	newVars, err := parseEnvContent(content)
	if err != nil {
		return fmt.Errorf("invalid environment format: %w", err)
	}

	// Write the updated environment
	if err := writeEnvFile(envPath, newVars); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}

	// Show success message
	cmd.Println()
	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s Environment updated successfully", ui.IconSuccess)))
	cmd.Println()

	// Show summary of changes
	addedCount := 0
	removedCount := 0
	updatedCount := 0

	// Count added and updated
	for key := range newVars {
		if oldValue, exists := vars[key]; exists {
			if oldValue != newVars[key] {
				updatedCount++
			}
		} else {
			addedCount++
		}
	}

	// Count removed
	for key := range vars {
		if _, exists := newVars[key]; !exists {
			removedCount++
		}
	}

	// Print summary
	if addedCount > 0 || removedCount > 0 || updatedCount > 0 {
		if addedCount > 0 {
			cmd.Printf("  %s %d variable(s) added\n", ui.SuccessStyle.Render("+"), addedCount)
		}
		if updatedCount > 0 {
			cmd.Printf("  %s %d variable(s) updated\n", ui.InfoStyle.Render("~"), updatedCount)
		}
		if removedCount > 0 {
			cmd.Printf("  %s %d variable(s) removed\n", ui.WarningStyle.Render("-"), removedCount)
		}
		cmd.Println()
	}

	cmd.Printf("Total: %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d variable(s)", len(newVars))))

	return nil
}
