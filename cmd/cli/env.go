package main

import (
	"context"

	"github.com/urfave/cli/v3"
)

// envSetCommand sets an environment variable
func envSetCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:set",
		Usage:     "Set environment variable",
		ArgsUsage: "APP KEY=VALUE",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// envListCommand lists environment variables
func envListCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:list",
		Usage:     "List environment variables",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// envUnsetCommand removes an environment variable
func envUnsetCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:unset",
		Usage:     "Remove environment variable",
		ArgsUsage: "APP KEY",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}
