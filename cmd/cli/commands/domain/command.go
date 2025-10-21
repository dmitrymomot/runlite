package domain

import "github.com/urfave/cli/v3"

func Command() *cli.Command {
	return &cli.Command{
		Name:     "domain",
		Category: "Domain Management",
		Usage:    "Manage domains",
		Commands: []*cli.Command{
			addCommand(),
			removeCommand(),
			listCommand(),
			verifyCommand(),
		},
	}
}
