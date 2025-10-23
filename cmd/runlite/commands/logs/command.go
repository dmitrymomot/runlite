package logs

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the 'logs' command for viewing application logs.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <app-name> [command]",
		Short: "View application logs",
		Long: `View logs from your applications.

Two types of logs are available:
  • Deployment logs: Build output and deployment steps (deploy.log files)
  • Runtime logs: Application output from systemd journal (journalctl)

The default action shows deployment logs if no subcommand is specified.

Examples:
  # View deployment logs for latest release (default)
  runlite logs myapp
  runlite logs myapp deploy

  # View deployment logs for specific release
  runlite logs myapp deploy 20251022143055-a3f5c2b

  # Follow deployment logs in real-time
  runlite logs myapp deploy --follow

  # View runtime logs from systemd
  runlite logs myapp runtime

  # Follow runtime logs with last 50 lines
  runlite logs myapp runtime --tail 50 --follow`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to deploy logs when no subcommand is provided
			// Create a temporary command context to call deploy with proper args
			return runDeployLogsDefault(cmd, args)
		},
	}

	// Add subcommands
	cmd.AddCommand(newDeployCommand())
	cmd.AddCommand(newRuntimeCommand())

	return cmd
}
