package deploy

import "github.com/urfave/cli/v3"

// Command returns the deploy parent command with all subcommands.
func Command() *cli.Command {
	return &cli.Command{
		Name:     "deploy",
		Category: "Deployment",
		Usage:    "Deploy and manage application deployments",
		Commands: []*cli.Command{
			startCommand(),
			rollbackCommand(),
			restartCommand(),
		},
	}
}
