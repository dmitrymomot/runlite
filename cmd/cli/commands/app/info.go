package app

import (
	"context"

	"github.com/urfave/cli/v3"
)

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Show detailed application information",
		ArgsUsage: "NAME",
		Action:    infoAction,
	}
}

func infoAction(ctx context.Context, cmd *cli.Command) error {
	panic("not implemented")
}
