package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/cmd/cli/stubs"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
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
		Action: initAction,
	}
}

func initAction(ctx context.Context, cmd *cli.Command) error {
	const configFileName = "runlite.yml"

	// Check if runlite.yml already exists and reject unless --force flag is set
	if _, err := os.Stat(configFileName); err == nil {
		if !cmd.Bool("force") {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("%s already exists. Use --force to overwrite.", configFileName)), 1)
		}
	}

	// Resolve app name: use flag > arg > current directory name
	appName := cmd.String("name")
	if appName == "" && cmd.Args().Len() > 0 {
		appName = cmd.Args().Get(0)
	}
	if appName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		appName = filepath.Base(cwd)
	}

	projectType := detectProjectType()
	ui.PrintInfo(fmt.Sprintf("Detected project type: %s", projectType))
	ui.PrintInfo(fmt.Sprintf("Generating %s for app: %s", configFileName, appName))

	config := generateConfig(appName, projectType)

	if err := os.WriteFile(configFileName, []byte(config), 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Created %s", configFileName))

	// Display next steps from template
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

// detectProjectType detects the project type by checking for language-specific files (go.mod, package.json).
func detectProjectType() string {
	if _, err := os.Stat("go.mod"); err == nil {
		return "go"
	}

	if _, err := os.Stat("package.json"); err == nil {
		return "node"
	}

	return "unknown"
}

// generateConfig returns the appropriate runlite.yml template based on detected project type.
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

// generateGoConfig generates a Go project configuration template.
func generateGoConfig(appName string) string {
	config, err := renderTemplate("go.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render go template: %v", err))
	}
	return config
}

// generateNodeConfig generates a Node.js project configuration template.
func generateNodeConfig(appName string) string {
	config, err := renderTemplate("nodejs.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render nodejs template: %v", err))
	}
	return config
}

// generateGenericConfig generates a minimal generic configuration template.
func generateGenericConfig(appName string) string {
	config, err := renderTemplate("generic.yml.tmpl", map[string]string{"AppName": appName})
	if err != nil {
		panic(fmt.Sprintf("render generic template: %v", err))
	}
	return config
}

// renderTemplate loads a template from the embedded filesystem and renders it with the provided data.
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
