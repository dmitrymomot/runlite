package deploy

import (
	"context"

	"github.com/urfave/cli/v3"
)

func startCommand() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Deploy an application",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ref",
				Usage: "Git reference to deploy (commit, tag, or branch)",
				Value: "HEAD",
			},
		},
		Action: startAction,
	}
}

func startAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
