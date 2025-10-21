package env

import "github.com/urfave/cli/v3"

// Command returns the env parent command with all subcommands.
func Command() *cli.Command {
	return &cli.Command{
		Name:     "env",
		Category: "Environment Variables",
		Usage:    "Manage environment variables",
		Commands: []*cli.Command{
			setCommand(),
			getCommand(),
			listCommand(),
			unsetCommand(),
			editCommand(),
			importCommand(),
			exportCommand(),
		},
	}
}
