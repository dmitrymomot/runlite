package env

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get KEY",
		Short: "Get the value of an environment variable",
		Long: `Get the value of a single environment variable.

This command is useful for:
  • Checking variable values quickly
  • Using in shell scripts
  • Debugging environment configuration

Output:
  • Shows only the variable value (no formatting)
  • Exit code 0 if variable exists
  • Exit code 1 if variable doesn't exist

Examples:
  # Get a variable value
  runlite env myapp get PORT

  # Use in a script
  PORT=$(runlite env myapp get PORT)
  echo "App is running on port $PORT"

  # Check if a variable exists
  if runlite env myapp get API_KEY > /dev/null 2>&1; then
    echo "API_KEY is configured"
  fi`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			key := args[0]
			return runGet(cmd, appName, key)
		},
	}

	return cmd
}

func runGet(cmd *cobra.Command, appName string, key string) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Validate key format
	if err := validateEnvKey(key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	// Read environment
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Check if variable exists
	value, exists := vars[key]
	if !exists {
		return fmt.Errorf("%s variable '%s' not found", ui.IconError, key)
	}

	// Print just the value (for scripting)
	cmd.Println(value)

	return nil
}
