package env

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func setCommand() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Set environment variable(s)",
		ArgsUsage: "APP KEY=VALUE [KEY2=VALUE2...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation",
			},
		},
		Action: setAction,
	}
}

func setAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return cli.Exit(ui.ErrorString("Usage: env set APP KEY=VALUE [KEY2=VALUE2...]"), 1)
	}

	appName := cmd.Args().Get(0)

	// Parse command-line KEY=VALUE arguments
	updates := make(map[string]string)
	for i := 1; i < cmd.Args().Len(); i++ {
		arg := cmd.Args().Get(i)
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("Invalid format: %s (expected KEY=VALUE)", arg)), 1)
		}
		key := strings.TrimSpace(parts[0])
		value := parts[1]
		if key == "" {
			return cli.Exit(ui.ErrorString("Key cannot be empty"), 1)
		}
		updates[key] = value
	}

	// Load existing variables to compute and display diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Merge updates into existing variables
	newVars := make(map[string]string)
	for k, v := range oldVars {
		newVars[k] = v
	}
	for k, v := range updates {
		newVars[k] = v
	}

	// Show diff and require confirmation unless --force flag is set
	diff := env.Diff(oldVars, newVars)
	if !diff.IsEmpty() {
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
	}

	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("Set %d variable(s) for %s", len(updates), appName))
	return nil
}
