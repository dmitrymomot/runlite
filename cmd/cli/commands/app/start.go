package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
	"github.com/dmitrymomot/runlite/internal/spec"
)

func startCommand() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Start a stopped deployment or restart the active one",
		ArgsUsage: "APP [RELEASE_ID]",
		Action:    startAction,
	}
}

func startAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)
	var releaseID string
	if cmd.Args().Len() > 1 {
		releaseID = cmd.Args().Get(1)
	}

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	manager := app.NewManager(config.GetDataDir())
	if !manager.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' not found", appName)), 1)
	}

	meta, err := manager.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to load metadata: %v", err)), 1)
	}

	var targetDeployment *app.Deployment

	if releaseID != "" {
		targetDeployment, err = manager.GetDeploymentByReleaseID(appName, releaseID)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("deployment not found: %v", err)), 1)
		}

		if targetDeployment.Status == app.DeploymentStatusFailed {
			return cli.Exit(ui.ErrorString("cannot start a failed deployment"), 1)
		}

		if targetDeployment.Status == app.DeploymentStatusActive {
			return cli.Exit(ui.ErrorString("deployment is already active"), 1)
		}

		if targetDeployment.Status != app.DeploymentStatusStopped {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("deployment must be stopped to start (current status: %s)", targetDeployment.Status)), 1)
		}
	} else {
		targetDeployment, err = manager.GetActiveDeployment(appName)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("no active deployment found: %v", err)), 1)
		}

		if targetDeployment.Status != app.DeploymentStatusStopped {
			return cli.Exit(ui.ErrorString("active deployment is already running"), 1)
		}
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Starting app '%s':", appName))
	fmt.Printf("  Release ID: %s\n", targetDeployment.ReleaseID)
	fmt.Printf("  Port: %d\n", targetDeployment.Port)
	fmt.Println()

	unitName := deploy.FormatUnitName(appName, targetDeployment.ReleaseID)

	ui.PrintInfo(fmt.Sprintf("Starting service: %s", unitName))
	if err := deploy.StartService(ctx, unitName); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to start service: %v", err)), 1)
	}

	releaseDir := filepath.Join(config.GetDataDir(), "apps", appName, "releases", targetDeployment.ReleaseID)
	specPath := filepath.Join(releaseDir, "runlite.yml")

	appSpec, err := spec.ParseFile(specPath)
	if err != nil {
		if stopErr := deploy.StopService(ctx, unitName); stopErr != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to stop service: %v", stopErr))
		}
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to load runlite.yml: %v", err)), 1)
	}

	var healthPassed bool
	if appSpec.Health != nil {
		healthPath := "/health"
		healthTimeout := 60 * time.Second
		healthInterval := 5 * time.Second

		if appSpec.Health.Path != "" {
			healthPath = appSpec.Health.Path
		}
		if appSpec.Health.Timeout > 0 {
			healthTimeout = appSpec.Health.Timeout
		}
		if appSpec.Health.Interval > 0 {
			healthInterval = appSpec.Health.Interval
		}

		healthURL := fmt.Sprintf("http://localhost:%d%s", targetDeployment.Port, healthPath)

		ui.PrintInfo("Performing health check...")
		if err := deploy.WaitForHealthy(ctx, healthURL, healthTimeout, healthInterval); err != nil {
			if stopErr := deploy.StopService(ctx, unitName); stopErr != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to stop service: %v", stopErr))
			}
			return cli.Exit(ui.ErrorString(fmt.Sprintf("health check failed: %v", err)), 1)
		}
		healthPassed = true
	}

	ui.PrintInfo("Updating Caddy route...")
	caddyClient := caddy.NewClient("")

	if err := caddyClient.UpdateAppRoute(appName, targetDeployment.Port); err != nil {
		if err := caddyClient.AddAppRoute(appName, appSpec.Domains, targetDeployment.Port); err != nil {
			if stopErr := deploy.StopService(ctx, unitName); stopErr != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to stop service: %v", stopErr))
			}
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to update Caddy route: %v", err)), 1)
		}
	}

	ui.PrintInfo("Updating deployment status...")

	for i := range meta.Deployments {
		if meta.Deployments[i].ReleaseID == targetDeployment.ReleaseID {
			meta.Deployments[i].Status = app.DeploymentStatusActive
			meta.Deployments[i].HealthCheckPassed = healthPassed
		} else if meta.Deployments[i].Status == app.DeploymentStatusActive {
			meta.Deployments[i].Status = app.DeploymentStatusStandby
		}
	}
	meta.UpdatedAt = time.Now()

	if err := manager.Save(meta); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to save metadata: %v", err)), 1)
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Started app '%s' (release: %s)", appName, targetDeployment.ReleaseID))
	if healthPassed {
		fmt.Println("  Health check: passed")
	}
	fmt.Println("  Caddy route: updated")

	return nil
}
