package env

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset KEY [KEY2 KEY3 ...]",
		Short: "Remove one or more environment variables",
		Long: `Remove one or more environment variables.

This is a convenience command for quick operations and scripting.
For editing multiple variables, use 'runlite env <app> edit' instead.

Behavior:
  • Removes specified variables from the environment
  • Silently ignores variables that don't exist
  • Other variables remain unchanged

Examples:
  # Remove a single variable
  runlite env myapp unset OLD_KEY

  # Remove multiple variables at once
  runlite env myapp unset OLD_KEY UNUSED_VAR DEPRECATED_SETTING

  # Remove a variable (even if it doesn't exist)
  runlite env myapp unset MAYBE_EXISTS`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			return runUnset(cmd, appName, args)
		},
	}

	return cmd
}

func runUnset(cmd *cobra.Command, appName string, keys []string) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Validate all keys
	for _, key := range keys {
		if err := validateEnvKey(key); err != nil {
			return fmt.Errorf("invalid key '%s': %w", key, err)
		}
	}

	// Read current environment
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Track what was actually removed
	removed := []string{}

	// Remove variables
	for _, key := range keys {
		if _, exists := vars[key]; exists {
			delete(vars, key)
			removed = append(removed, key)
		}
	}

	// Check if anything was actually removed
	if len(removed) == 0 {
		cmd.Println()
		cmd.Println(ui.WarningStyle.Render(fmt.Sprintf("%s No variables were removed", ui.IconWarning)))
		cmd.Println()
		cmd.Println(ui.MutedStyle.Render("The specified variables do not exist in the environment"))
		return nil
	}

	// Write updated environment
	if err := writeEnvFile(envPath, vars); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}

	// Show success message
	cmd.Println()
	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s Environment updated", ui.IconSuccess)))
	cmd.Println()
	cmd.Printf("  %s Removed: %s\n", ui.WarningStyle.Render("-"), strings.Join(removed, ", "))
	cmd.Println()
	cmd.Printf("Remaining: %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d variable(s)", len(vars))))

	return nil
}
