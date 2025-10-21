package env

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func getCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get environment variable value",
		ArgsUsage: "APP KEY",
		Action:    getAction,
	}
}

func getAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit(ui.ErrorString("Usage: env get APP KEY"), 1)
	}

	appName := cmd.Args().Get(0)
	key := cmd.Args().Get(1)

	value, err := env.Get(appName, key)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to get variable: %v", err)), 1)
	}

	fmt.Println(value)
	return nil
}
