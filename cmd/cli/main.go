package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// version can be set at build time using:
// go build -ldflags "-X main.version=v1.0.0" -o runlite cmd/cli/main.go
var version = "dev"

func main() {
	// Create CLI app
	app := &cli.Command{
		Name:  "runlite",
		Usage: "Lightweight deployment platform for Go applications",
		Commands: []*cli.Command{
			// App management
			appCreateCommand(),
			appListCommand(),
			appInfoCommand(),
			appInitCommand(),

			// Environment variables
			envSetCommand(),
			envGetCommand(),
			envListCommand(),
			envUnsetCommand(),
			envEditCommand(),
			envImportCommand(),
			envExportCommand(),

			// Deployment
			deployCommand(),
			rollbackCommand(),
			restartCommand(),

			// Logs
			logsCommand(),

			// Utility
			versionCommand(),
		},
	}

	// Run CLI app
	// Commands print their own error messages, so just exit on error
	if err := app.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}

// versionCommand shows version information
func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("runlite version %s\n", version)
			return nil
		},
	}
}
