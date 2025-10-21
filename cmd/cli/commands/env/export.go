package env

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export environment variables to file",
		ArgsUsage: "APP [FILE]",
		Action:    exportAction,
	}
}

func exportAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 || cmd.Args().Len() > 2 {
		return cli.Exit(ui.ErrorString("Usage: env export APP [FILE]"), 1)
	}

	appName := cmd.Args().Get(0)
	filePath := ""
	if cmd.Args().Len() == 2 {
		filePath = cmd.Args().Get(1)
	}

	if err := env.Export(appName, filePath); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to export env: %v", err)), 1)
	}

	if filePath != "" && filePath != "-" {
		ui.PrintSuccess(fmt.Sprintf("Exported environment to %s", filePath))
	}

	return nil
}
