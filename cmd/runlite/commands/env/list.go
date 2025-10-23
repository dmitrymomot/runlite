package env

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all environment variables",
		Long: `List all environment variables for an application.

Shows variables in a compact, colored format (KEY=VALUE).

The output can be copied directly to a .env file or used in scripts.

Examples:
  # List all variables
  runlite env myapp list

  # Also works as default action
  runlite env myapp`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			return runList(cmd, appName)
		},
	}

	return cmd
}

func runList(cmd *cobra.Command, appName string) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Read environment file
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Handle empty environment
	if len(vars) == 0 {
		cmd.Println(ui.MutedStyle.Render("No environment variables set"))
		cmd.Println()
		cmd.Println(ui.InfoStyle.Render(fmt.Sprintf("Use 'runlite env %s edit' to add variables", appName)))
		return nil
	}

	// Sort and display variables
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	// Sort keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Print variables with colored keys
	cmd.Println()
	for _, key := range keys {
		keyStyled := ui.BoldStyle.Foreground(ui.ColorPrimary).Render(key)
		cmd.Printf("%s=%s\n", keyStyled, vars[key])
	}

	// Print summary
	cmd.Println()
	count := len(vars)
	var countText string
	if count == 1 {
		countText = "1 variable"
	} else {
		countText = fmt.Sprintf("%d variables", count)
	}
	cmd.Println(ui.MutedStyle.Render(countText))

	return nil
}
