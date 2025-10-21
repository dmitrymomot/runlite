package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/stubs"
)

// appCreateCommand creates a new application
func appCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "app:create",
		Usage:     "Create a new application",
		ArgsUsage: "NAME",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// appListCommand lists all applications
func appListCommand() *cli.Command {
	return &cli.Command{
		Name:  "app:list",
		Usage: "List all applications",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// appInfoCommand shows detailed application information
func appInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "app:info",
		Usage:     "Show detailed application information",
		ArgsUsage: "NAME",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			panic("not implemented")
		},
	}
}

// appInitCommand generates a runlite.yml configuration file
func appInitCommand() *cli.Command {
	return &cli.Command{
		Name:      "app:init",
		Usage:     "Generate a runlite.yml configuration file",
		ArgsUsage: "[NAME]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Application name (default: current directory name)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing runlite.yml",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runAppInit(cmd)
		},
	}
}

// runAppInit implements the app:init command logic
func runAppInit(cmd *cli.Command) error {
	const configFileName = "runlite.yml"

	// Check if runlite.yml already exists
	if _, err := os.Stat(configFileName); err == nil {
		if !cmd.Bool("force") {
			return cli.Exit(errorString(fmt.Sprintf("%s already exists. Use --force to overwrite.", configFileName)), 1)
		}
	}

	// Get app name from flag, arg, or current directory
	appName := cmd.String("name")
	if appName == "" && cmd.Args().Len() > 0 {
		appName = cmd.Args().Get(0)
	}
	if appName == "" {
		// Use current directory name
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		appName = filepath.Base(cwd)
	}

	// Detect project type
	projectType := detectProjectType()

	printInfo(fmt.Sprintf("Detected project type: %s", projectType))
	printInfo(fmt.Sprintf("Generating %s for app: %s", configFileName, appName))

	// Generate config based on project type
	config := generateConfig(appName, projectType)

	// Write config file
	if err := os.WriteFile(configFileName, []byte(config), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	printSuccess(fmt.Sprintf("Created %s", configFileName))

	// Print next steps
	nextSteps, err := renderTemplate("next-steps.txt.tmpl", map[string]string{
		"ConfigFileName": configFileName,
		"AppName":        appName,
	})
	if err != nil {
		return fmt.Errorf("render next steps: %w", err)
	}
	fmt.Print(nextSteps)

	return nil
}

// detectProjectType detects the project type based on files in the current directory
func detectProjectType() string {
	// Check for Go project
	if _, err := os.Stat("go.mod"); err == nil {
		return "go"
	}

	// Check for Node.js project
	if _, err := os.Stat("package.json"); err == nil {
		return "node"
	}

	// Default to Go
	return "go"
}

// generateConfig generates the runlite.yml content based on project type
func generateConfig(appName, projectType string) string {
	switch projectType {
	case "go":
		return generateGoConfig(appName)
	case "node":
		return generateNodeConfig(appName)
	default:
		return generateGenericConfig(appName)
	}
}

// generateGoConfig generates a Go-specific configuration
func generateGoConfig(appName string) string {
	config, err := renderTemplate("go.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render go template: %v", err))
	}
	return config
}

// generateNodeConfig generates a Node.js-specific configuration
func generateNodeConfig(appName string) string {
	config, err := renderTemplate("nodejs.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render nodejs template: %v", err))
	}
	return config
}

// generateGenericConfig generates a minimal generic configuration
func generateGenericConfig(appName string) string {
	config, err := renderTemplate("generic.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render generic template: %v", err))
	}
	return config
}

// renderTemplate loads and renders a template from the embedded filesystem
func renderTemplate(name string, data any) (string, error) {
	tmplContent, err := stubs.FS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
