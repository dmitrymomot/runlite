package commands

import (
	"github.com/spf13/cobra"
)

// Root returns the root command for the installer CLI.
// This will later become a subcommand when integrated into the main runlite CLI.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installer",
		Short: "Install and manage runlite setup templates",
		Long: `Installer manages runlite setup templates and dependencies.

Install complete templates or individual components.
Supports automated configuration with minimal manual intervention.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands
	cmd.AddCommand(NewInstallCommand())

	return cmd
}
