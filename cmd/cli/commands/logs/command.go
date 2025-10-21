package logs

import (
	"context"

	"github.com/urfave/cli/v3"
)

// Command returns the logs command for displaying application logs.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Category:  "Deployment",
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
		Action: logsAction,
	}
}

func logsAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
