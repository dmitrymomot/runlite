package deploy

import (
	"context"

	"github.com/urfave/cli/v3"
)

func restartCommand() *cli.Command {
	return &cli.Command{
		Name:      "restart",
		Usage:     "Restart an application",
		ArgsUsage: "APP",
		Action:    restartAction,
	}
}

func restartAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
