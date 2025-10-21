package app

import (
	"context"

	"github.com/urfave/cli/v3"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all applications",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
