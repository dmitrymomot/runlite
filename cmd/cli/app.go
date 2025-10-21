package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
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
	fmt.Printf(`
Next steps:
  1. Review and customize %s
  2. Commit the file to your repository:
     git add %s && git commit -m "Add runlite config"
  3. Add your RunLite server as a remote:
     git remote add runlite git@server:/home/git/apps/%s.git
  4. Deploy:
     git push runlite main
`, configFileName, configFileName, appName)

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
	return fmt.Sprintf(`# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json
# RunLite Configuration
# See: https://github.com/dmitrymomot/runlite/blob/main/docs/config-file.md

app:
  name: %s

build:
  # Build script (optional - defaults to: go build -o ./app .)
  script: |
    go build -o ./bin/server .

  # Copy only necessary files to release directory
  artifacts:
    - ./bin/server

run:
  # Command to execute (relative to release directory)
  command: ./bin/server

  # Optional: Command arguments
  # args:
  #   - --port
  #   - "{{.Port}}"

health:
  # Health check endpoint
  path: /health
  timeout: 30s
  interval: 1s

# Optional: Database configuration
# database:
#   path: ./databases

# Optional: Static files
# static:
#   path: ./public

# Optional: Deployment settings
# deploy:
#   drain_period: 30s
`, appName)
}

// generateNodeConfig generates a Node.js-specific configuration
func generateNodeConfig(appName string) string {
	return fmt.Sprintf(`# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json
# RunLite Configuration
# See: https://github.com/dmitrymomot/runlite/blob/main/docs/config-file.md

app:
  name: %s

build:
  script: |
    npm ci
    npm run build
    npm prune --production

  # Copy build output and dependencies
  artifacts:
    - ./dist/
    - ./node_modules/
    - ./package.json

run:
  # Command to execute (relative to release directory)
  command: node dist/index.js

health:
  path: /health
  timeout: 60s
  interval: 2s

# Optional: Static files
# static:
#   path: ./public

# Optional: Deployment settings
# deploy:
#   drain_period: 30s
`, appName)
}

// generateGenericConfig generates a minimal generic configuration
func generateGenericConfig(appName string) string {
	return fmt.Sprintf(`# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json
# RunLite Configuration
# See: https://github.com/dmitrymomot/runlite/blob/main/docs/config-file.md

app:
  name: %s

# Optional: Build configuration
# build:
#   script: |
#     # Your build commands here
#     make build
#
#   artifacts:
#     - ./bin/app

# Optional: Run configuration
# run:
#   command: ./bin/app

# Optional: Health check
# health:
#   path: /health
#   timeout: 30s
`, appName)
}
