package app

import "github.com/urfave/cli/v3"

// Command returns the app parent command with all subcommands.
func Command() *cli.Command {
	return &cli.Command{
		Name:     "app",
		Category: "Application Management",
		Usage:    "Manage applications",
		Commands: []*cli.Command{
			createCommand(),
			listCommand(),
			infoCommand(),
			initCommand(),
		},
	}
}
