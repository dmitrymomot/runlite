package deploy

import (
	"context"

	"github.com/urfave/cli/v3"
)

func rollbackCommand() *cli.Command {
	return &cli.Command{
		Name:      "rollback",
		Usage:     "Rollback to previous deployment",
		ArgsUsage: "APP",
		Action:    rollbackAction,
	}
}

func rollbackAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
