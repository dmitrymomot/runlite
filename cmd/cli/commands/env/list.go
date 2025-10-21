package env

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List environment variables",
		ArgsUsage: "APP",
		Action:    listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(ui.ErrorString("Usage: env list APP"), 1)
	}

	appName := cmd.Args().Get(0)

	vars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	if len(vars) == 0 {
		ui.PrintInfo(fmt.Sprintf("No environment variables set for %s", appName))
		return nil
	}

	// Sort keys alphabetically for consistent output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Display variables, truncating multi-line values for readability
	for _, key := range keys {
		value := vars[key]
		if strings.Contains(value, "\n") {
			lines := strings.Split(value, "\n")
			if len(lines) > 0 {
				fmt.Printf("%s=%s... (%d lines)\n", key, lines[0], len(lines))
			} else {
				fmt.Printf("%s=%s\n", key, value)
			}
		} else {
			fmt.Printf("%s=%s\n", key, value)
		}
	}

	return nil
}
