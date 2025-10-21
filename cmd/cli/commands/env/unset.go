package env

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func unsetCommand() *cli.Command {
	return &cli.Command{
		Name:      "unset",
		Usage:     "Remove environment variable(s)",
		ArgsUsage: "APP KEY [KEY2...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation",
			},
		},
		Action: unsetAction,
	}
}

func unsetAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return cli.Exit(ui.ErrorString("Usage: env unset APP KEY [KEY2...]"), 1)
	}

	appName := cmd.Args().Get(0)
	keys := cmd.Args().Slice()[1:]

	// Load existing variables to compute and display diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Remove specified keys from the variables
	newVars := make(map[string]string)
	for k, v := range oldVars {
		newVars[k] = v
	}
	for _, key := range keys {
		delete(newVars, key)
	}

	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		ui.PrintWarning("No variables to remove")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	// Confirm unless --force flag is set
	if !cmd.Bool("force") {
		fmt.Print("\nApply these changes? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Println("\nCancelled")
			return nil
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("Removed %d variable(s) from %s", len(keys), appName))
	return nil
}
