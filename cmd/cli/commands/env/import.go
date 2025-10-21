package env

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func importCommand() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Usage:     "Import environment variables from file",
		ArgsUsage: "APP FILE",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "merge",
				Usage: "Merge with existing variables (don't replace)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation",
			},
		},
		Action: importAction,
	}
}

func importAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit(ui.ErrorString("Usage: env import APP FILE"), 1)
	}

	appName := cmd.Args().Get(0)
	filePath := cmd.Args().Get(1)
	merge := cmd.Bool("merge")

	// Load existing variables to compute and display diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Read import file (support stdin with "-" filename)
	var content []byte
	if filePath == "-" {
		content, err = os.ReadFile("/dev/stdin")
	} else {
		content, err = os.ReadFile(filePath)
	}
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to read import file: %v", err)), 1)
	}

	importedVars, err := env.Parse(string(content))
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to parse import file: %v", err)), 1)
	}

	// Either merge with existing variables or replace all
	var newVars map[string]string
	if merge {
		newVars = make(map[string]string)
		for k, v := range oldVars {
			newVars[k] = v
		}
		for k, v := range importedVars {
			newVars[k] = v
		}
	} else {
		newVars = importedVars
	}

	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		ui.PrintInfo("No changes")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	// Confirm unless --force flag is set
	if !cmd.Bool("force") {
		fmt.Print("\nImport these changes? [y/N]: ")
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

	ui.PrintSuccess(fmt.Sprintf("Imported %d variable(s) for %s", len(importedVars), appName))
	return nil
}
