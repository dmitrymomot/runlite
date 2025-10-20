package main

import (
	"context"

	"github.com/urfave/cli/v3"
)

// deployCommand deploys an application
func deployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy an application",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ref",
				Usage: "Git reference to deploy (commit, tag, or branch)",
				Value: "HEAD",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// rollbackCommand rolls back to previous deployment
func rollbackCommand() *cli.Command {
	return &cli.Command{
		Name:      "rollback",
		Usage:     "Rollback to previous deployment",
		ArgsUsage: "APP",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// restartCommand restarts an application
func restartCommand() *cli.Command {
	return &cli.Command{
		Name:      "restart",
		Usage:     "Restart an application",
		ArgsUsage: "APP",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}
