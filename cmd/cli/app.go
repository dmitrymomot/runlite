package main

import (
	"context"

	"github.com/urfave/cli/v3"
)

// appCreateCommand creates a new application
func appCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "app:create",
		Usage:     "Create a new application",
		ArgsUsage: "NAME",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// appListCommand lists all applications
func appListCommand() *cli.Command {
	return &cli.Command{
		Name:  "app:list",
		Usage: "List all applications",
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

// appInfoCommand shows detailed application information
func appInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "app:info",
		Usage:     "Show detailed application information",
		ArgsUsage: "NAME",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}
