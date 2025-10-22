package app

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func stopCommand() *cli.Command {
	return &cli.Command{
		Name:      "stop",
		Usage:     "Stop the active deployment",
		ArgsUsage: "APP",
		Action:    stopAction,
	}
}

func stopAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	manager := app.NewManager(config.GetDataDir())
	if !manager.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' not found", appName)), 1)
	}

	activeDeployment, err := manager.GetActiveDeployment(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("no active deployment: %v", err)), 1)
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Stopping app '%s':", appName))
	fmt.Printf("  Release ID: %s\n", activeDeployment.ReleaseID)
	fmt.Printf("  Port: %d\n", activeDeployment.Port)
	fmt.Println()

	unitName := deploy.FormatUnitName(appName, activeDeployment.ReleaseID)

	ui.PrintInfo(fmt.Sprintf("Stopping service: %s", unitName))
	if err := deploy.StopService(ctx, unitName); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to stop service: %v", err)), 1)
	}

	ui.PrintInfo("Removing Caddy route...")
	caddyClient := caddy.NewClient("")
	if err := caddyClient.DeleteAppRoute(appName); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to delete Caddy route: %v", err))
	}

	ui.PrintInfo("Updating deployment status...")
	if err := manager.MarkDeploymentStopped(appName, activeDeployment.ReleaseID); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to update metadata: %v", err)), 1)
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Stopped app '%s' (release: %s)", appName, activeDeployment.ReleaseID))

	return nil
}
