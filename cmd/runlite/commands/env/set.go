package env

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set KEY=VALUE [KEY2=VALUE2 ...]",
		Short: "Set one or more environment variables",
		Long: `Set one or more environment variables quickly.

This is a convenience command for quick operations and scripting.
For editing multiple variables, use 'runlite env <app> edit' instead.

Format:
  KEY=VALUE

Rules:
  • Keys must be uppercase letters, numbers, and underscores only
  • No spaces in keys
  • Values can contain any characters
  • Multiple variables can be set at once

Behavior:
  • Creates new variables if they don't exist
  • Updates existing variables with new values
  • Other variables remain unchanged

Examples:
  # Set a single variable
  runlite env myapp set PORT=8000

  # Set multiple variables at once
  runlite env myapp set PORT=8000 DB_HOST=localhost DB_NAME=myapp

  # Update an existing variable
  runlite env myapp set API_KEY=new_key_value`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			return runSet(cmd, appName, args)
		},
	}

	return cmd
}

func runSet(cmd *cobra.Command, appName string, pairs []string) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Parse KEY=VALUE pairs
	varsToSet := make(map[string]string)
	for _, pair := range pairs {
		key, value, err := parseEnvLine(pair)
		if err != nil {
			return fmt.Errorf("invalid format '%s': %w", pair, err)
		}

		if key == "" {
			return fmt.Errorf("invalid format '%s': expected KEY=VALUE", pair)
		}

		varsToSet[key] = value
	}

	// Read current environment
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Track what's being added vs updated
	added := []string{}
	updated := []string{}

	// Update or add variables
	for key, value := range varsToSet {
		if _, exists := vars[key]; exists {
			updated = append(updated, key)
		} else {
			added = append(added, key)
		}
		vars[key] = value
	}

	// Write updated environment
	if err := writeEnvFile(envPath, vars); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}

	// Show success message
	cmd.Println()
	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s Environment updated", ui.IconSuccess)))
	cmd.Println()

	// Show what was changed
	if len(added) > 0 {
		cmd.Printf("  %s Added:   %s\n", ui.SuccessStyle.Render("+"), strings.Join(added, ", "))
	}
	if len(updated) > 0 {
		cmd.Printf("  %s Updated: %s\n", ui.InfoStyle.Render("~"), strings.Join(updated, ", "))
	}

	return nil
}
