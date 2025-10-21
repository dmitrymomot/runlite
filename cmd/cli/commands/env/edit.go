package env

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/env"
)

func editCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "Edit environment variables in $EDITOR",
		ArgsUsage: "APP",
		Action:    editAction,
	}
}

func editAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(ui.ErrorString("Usage: env edit APP"), 1)
	}

	appName := cmd.Args().Get(0)

	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Create temporary file for editor
	tmpFile, err := os.CreateTemp("", "runlite-env-*.env")
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to create temp file: %v", err)), 1)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Format current variables and write to temp file
	content := env.Format(oldVars)
	if _, err := tmpFile.WriteString(content); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to write temp file: %v", err)), 1)
	}
	tmpFile.Close()

	// Launch editor (prefer $EDITOR env var, fall back to vim)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to run editor: %v", err)), 1)
	}

	// Parse edited file
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to read edited file: %v", err)), 1)
	}

	newVars, err := env.Parse(string(editedContent))
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to parse edited file: %v", err)), 1)
	}

	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		ui.PrintInfo("No changes")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	// Require confirmation before saving
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

	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("Updated environment for %s", appName))
	return nil
}
