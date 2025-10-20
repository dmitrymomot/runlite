package main

import (
	"context"

	"github.com/urfave/cli/v3"
)

// logsCommand shows application logs
func logsCommand() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Usage:     "Show application logs",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show",
				Value:   100,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}
