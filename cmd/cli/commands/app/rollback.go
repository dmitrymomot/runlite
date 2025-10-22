package app

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
)

func rollbackCommand() *cli.Command {
	return &cli.Command{
		Name:      "rollback",
		Usage:     "Rollback to a previous release",
		ArgsUsage: "APP [RELEASE_ID]",
		Action:    rollbackAction,
	}
}

func rollbackAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)
	var targetReleaseID string
	if cmd.Args().Len() > 1 {
		targetReleaseID = cmd.Args().Get(1)
	}

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	manager := app.NewManager(config.GetDataDir())
	if !manager.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' not found", appName)), 1)
	}

	activeDeployment, err := manager.GetActiveDeployment(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to get active deployment: %v", err)), 1)
	}

	var targetDeployment *app.Deployment

	if targetReleaseID == "" {
		targetDeployment, err = manager.GetStandbyDeployment(appName)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("no standby deployment available: %v", err)), 1)
		}
	} else {
		targetDeployment, err = manager.GetDeploymentByReleaseID(appName, targetReleaseID)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("deployment not found: %v", err)), 1)
		}

		if targetDeployment.Status == app.DeploymentStatusActive {
			return cli.Exit(ui.ErrorString("target release is already active"), 1)
		}

		if targetDeployment.Status == app.DeploymentStatusFailed {
			return cli.Exit(ui.ErrorString("cannot rollback to a failed deployment"), 1)
		}
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Rolling back app '%s':", appName))
	fmt.Printf("  From: %s (port %d)\n", activeDeployment.ReleaseID, activeDeployment.Port)
	fmt.Printf("  To:   %s (port %d)\n", targetDeployment.ReleaseID, targetDeployment.Port)
	fmt.Println()

	ui.PrintInfo("Switching traffic to target release...")

	caddyClient := caddy.NewClient("")
	if err := caddyClient.UpdateAppRoute(appName, targetDeployment.Port); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to update Caddy route: %v", err)), 1)
	}

	ui.PrintInfo("Updating deployment statuses...")

	if targetReleaseID == "" {
		if _, err := manager.Rollback(appName); err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to update metadata: %v", err)), 1)
		}
	} else {
		meta, err := manager.Load(appName)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to load metadata: %v", err)), 1)
		}

		for i := range meta.Deployments {
			if meta.Deployments[i].ReleaseID == activeDeployment.ReleaseID {
				meta.Deployments[i].Status = app.DeploymentStatusStandby
			}
			if meta.Deployments[i].ReleaseID == targetDeployment.ReleaseID {
				meta.Deployments[i].Status = app.DeploymentStatusActive
			}
		}

		if err := manager.Save(meta); err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to save metadata: %v", err)), 1)
		}
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Rollback complete - now running release %s", targetDeployment.ReleaseID))

	return nil
}
