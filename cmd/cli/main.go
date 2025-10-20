package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create CLI app
	app := &cli.Command{
		Name:  "runlite",
		Usage: "Lightweight deployment platform for Go applications",
		Commands: []*cli.Command{
			// App management
			appCreateCommand(),
			appListCommand(),
			appInfoCommand(),

			// Environment variables
			envSetCommand(),
			envListCommand(),
			envUnsetCommand(),

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
	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

// versionCommand is an example command demonstrating the pattern
func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			slog.Info("runlite version", "version", "0.1.0-dev")
			return nil
		},
	}
}
