package env

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the 'env' command for managing environment variables.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env <app-name> [command]",
		Short: "Manage environment variables for applications",
		Long: `Manage environment variables for your applications.

Environment variables are stored in plain text at:
  /var/lib/runlite/apps/<app-name>/env

The env file format is simple:
  KEY=VALUE
  ANOTHER_KEY=another value

Keys must contain only uppercase letters, numbers, and underscores (A-Z, 0-9, _).

MAIN EDITING METHOD:
  Use 'env edit' for managing multiple variables with a multiline text editor.
  This is the recommended way to make changes.

QUICK OPERATIONS:
  Other commands (set, unset, get) are provided for quick single operations
  and scripting purposes.

Examples:
  # View all variables (default action)
  runlite env myapp
  runlite env myapp list

  # Edit variables in multiline text editor (RECOMMENDED)
  runlite env myapp edit

  # Quick operations
  runlite env myapp set PORT=8000 DB_HOST=localhost
  runlite env myapp get PORT
  runlite env myapp unset OLD_KEY UNUSED_VAR

  # Import/Export for backups
  runlite env myapp import .env.production
  runlite env myapp export .env.backup`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to list when no subcommand is provided
			appName := args[0]
			return runList(cmd, appName)
		},
	}

	// Add subcommands
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newEditCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newUnsetCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newImportCommand())
	cmd.AddCommand(newExportCommand())

	return cmd
}
