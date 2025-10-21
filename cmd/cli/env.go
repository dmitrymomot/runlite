package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/internal/env"
)

// envSetCommand sets environment variables
func envSetCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:set",
		Usage:     "Set environment variable(s)",
		ArgsUsage: "APP KEY=VALUE [KEY2=VALUE2...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvSet(cmd)
		},
	}
}

// envGetCommand gets a single environment variable
func envGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:get",
		Usage:     "Get environment variable value",
		ArgsUsage: "APP KEY",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvGet(cmd)
		},
	}
}

// envListCommand lists environment variables
func envListCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:list",
		Usage:     "List environment variables",
		ArgsUsage: "APP",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvList(cmd)
		},
	}
}

// envUnsetCommand removes environment variables
func envUnsetCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:unset",
		Usage:     "Remove environment variable(s)",
		ArgsUsage: "APP KEY [KEY2...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvUnset(cmd)
		},
	}
}

// envEditCommand opens environment variables in editor
func envEditCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:edit",
		Usage:     "Edit environment variables in $EDITOR",
		ArgsUsage: "APP",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvEdit(cmd)
		},
	}
}

// envImportCommand imports environment variables from a file
func envImportCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:import",
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvImport(cmd)
		},
	}
}

// envExportCommand exports environment variables to a file
func envExportCommand() *cli.Command {
	return &cli.Command{
		Name:      "env:export",
		Usage:     "Export environment variables to file",
		ArgsUsage: "APP [FILE]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnvExport(cmd)
		},
	}
}

// runEnvSet implements the env:set command
func runEnvSet(cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return cli.Exit(errorString("Usage: env:set APP KEY=VALUE [KEY2=VALUE2...]"), 1)
	}

	appName := cmd.Args().Get(0)

	// Parse KEY=VALUE pairs
	updates := make(map[string]string)
	for i := 1; i < cmd.Args().Len(); i++ {
		arg := cmd.Args().Get(i)
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return cli.Exit(errorString(fmt.Sprintf("Invalid format: %s (expected KEY=VALUE)", arg)), 1)
		}
		key := strings.TrimSpace(parts[0])
		value := parts[1]
		if key == "" {
			return cli.Exit(errorString("Key cannot be empty"), 1)
		}
		updates[key] = value
	}

	// Load existing vars to show diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Create new vars map
	newVars := make(map[string]string)
	for k, v := range oldVars {
		newVars[k] = v
	}
	for k, v := range updates {
		newVars[k] = v
	}

	// Show diff
	diff := env.Diff(oldVars, newVars)
	if !diff.IsEmpty() {
		fmt.Println("Changes:")
		fmt.Print(diff.Format())

		// Confirm unless --force
		if !cmd.Bool("force") {
			fmt.Print("\nApply these changes? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				// Treat scan error as cancellation
				fmt.Println("\nCancelled")
				return nil
			}
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Cancelled")
				return nil
			}
		}
	}

	// Save
	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	printSuccess(fmt.Sprintf("Set %d variable(s) for %s", len(updates), appName))
	return nil
}

// runEnvGet implements the env:get command
func runEnvGet(cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit(errorString("Usage: env:get APP KEY"), 1)
	}

	appName := cmd.Args().Get(0)
	key := cmd.Args().Get(1)

	value, err := env.Get(appName, key)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to get variable: %v", err)), 1)
	}

	fmt.Println(value)
	return nil
}

// runEnvList implements the env:list command
func runEnvList(cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(errorString("Usage: env:list APP"), 1)
	}

	appName := cmd.Args().Get(0)

	vars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	if len(vars) == 0 {
		printInfo(fmt.Sprintf("No environment variables set for %s", appName))
		return nil
	}

	// Sort keys
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

	// Print
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

// runEnvUnset implements the env:unset command
func runEnvUnset(cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return cli.Exit(errorString("Usage: env:unset APP KEY [KEY2...]"), 1)
	}

	appName := cmd.Args().Get(0)
	keys := cmd.Args().Slice()[1:]

	// Load existing vars to show diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Create new vars map
	newVars := make(map[string]string)
	for k, v := range oldVars {
		newVars[k] = v
	}
	for _, key := range keys {
		delete(newVars, key)
	}

	// Show diff
	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		printWarning("No variables to remove")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	// Confirm unless --force
	if !cmd.Bool("force") {
		fmt.Print("\nApply these changes? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Treat scan error as cancellation
			fmt.Println("\nCancelled")
			return nil
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Save
	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	printSuccess(fmt.Sprintf("Removed %d variable(s) from %s", len(keys), appName))
	return nil
}

// runEnvEdit implements the env:edit command
func runEnvEdit(cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(errorString("Usage: env:edit APP"), 1)
	}

	appName := cmd.Args().Get(0)

	// Load existing vars
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "runlite-env-*.env")
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to create temp file: %v", err)), 1)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write current vars to temp file
	content := env.Format(oldVars)
	if _, err := tmpFile.WriteString(content); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to write temp file: %v", err)), 1)
	}
	tmpFile.Close()

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Open editor
	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to run editor: %v", err)), 1)
	}

	// Read edited content
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to read edited file: %v", err)), 1)
	}

	// Parse edited vars
	newVars, err := env.Parse(string(editedContent))
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to parse edited file: %v", err)), 1)
	}

	// Show diff
	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		printInfo("No changes")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	fmt.Print("\nApply these changes? [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// Treat scan error as cancellation
		fmt.Println("\nCancelled")
		return nil
	}
	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	// Save
	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	printSuccess(fmt.Sprintf("Updated environment for %s", appName))
	return nil
}

// runEnvImport implements the env:import command
func runEnvImport(cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit(errorString("Usage: env:import APP FILE"), 1)
	}

	appName := cmd.Args().Get(0)
	filePath := cmd.Args().Get(1)
	merge := cmd.Bool("merge")

	// Load existing vars to show diff
	oldVars, err := env.Load(appName)
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to load env: %v", err)), 1)
	}

	// Read import file
	var content []byte
	if filePath == "-" {
		content, err = os.ReadFile("/dev/stdin")
	} else {
		content, err = os.ReadFile(filePath)
	}
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to read import file: %v", err)), 1)
	}

	importedVars, err := env.Parse(string(content))
	if err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to parse import file: %v", err)), 1)
	}

	// Create new vars
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

	// Show diff
	diff := env.Diff(oldVars, newVars)
	if diff.IsEmpty() {
		printInfo("No changes")
		return nil
	}

	fmt.Println("Changes:")
	fmt.Print(diff.Format())

	// Confirm unless --force
	if !cmd.Bool("force") {
		fmt.Print("\nImport these changes? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Treat scan error as cancellation
			fmt.Println("\nCancelled")
			return nil
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Save
	if err := env.Save(appName, newVars); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to save env: %v", err)), 1)
	}

	printSuccess(fmt.Sprintf("Imported %d variable(s) for %s", len(importedVars), appName))
	return nil
}

// runEnvExport implements the env:export command
func runEnvExport(cmd *cli.Command) error {
	if cmd.Args().Len() < 1 || cmd.Args().Len() > 2 {
		return cli.Exit(errorString("Usage: env:export APP [FILE]"), 1)
	}

	appName := cmd.Args().Get(0)
	filePath := ""
	if cmd.Args().Len() == 2 {
		filePath = cmd.Args().Get(1)
	}

	if err := env.Export(appName, filePath); err != nil {
		return cli.Exit(errorString(fmt.Sprintf("Failed to export env: %v", err)), 1)
	}

	if filePath != "" && filePath != "-" {
		printSuccess(fmt.Sprintf("Exported environment to %s", filePath))
	}

	return nil
}
