package commands

import (
	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/commands/env"
	"github.com/dmitrymomot/runlite/cmd/runlite/commands/install"
	"github.com/dmitrymomot/runlite/cmd/runlite/commands/logs"
)

// Root returns the root command for the runlite CLI.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runlite",
		Short: "Self-hosted PaaS for Go applications and static websites",
		Long: `runlite is a lightweight, self-hosted Platform-as-a-Service (PaaS)
for deploying and managing Go applications and static websites.

Features:
  • Interactive setup with beautiful CLI
  • Git-based deployments with zero-downtime
  • Automatic SSL with Caddy
  • SQLite backups with Litestream
  • Systemd-based process management

Get started:
  runlite install          # Interactive setup wizard
  runlite app create       # Create your first app
  runlite deploy           # Deploy your application`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands
	cmd.AddCommand(install.NewCommand())
	cmd.AddCommand(env.NewCommand())
	cmd.AddCommand(logs.NewCommand())

	return cmd
}
