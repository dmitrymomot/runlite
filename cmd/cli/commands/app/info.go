package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Show detailed application information",
		ArgsUsage: "NAME",
		Action:    infoAction,
	}
}

func infoAction(ctx context.Context, cmd *cli.Command) error {
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

	meta, err := manager.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to load app metadata: %v", err)), 1)
	}

	activeDeployment, err := manager.GetActiveDeployment(appName)
	hasActiveDeployment := err == nil

	total, active, standby, failed, stopped, err := manager.GetStats(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to get deployment stats: %v", err)), 1)
	}

	deployments, err := manager.ListDeployments(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to list deployments: %v", err)), 1)
	}

	fmt.Println()
	fmt.Printf("%s %s\n", ui.ColorBold("Application:"), appName)

	if hasActiveDeployment {
		fmt.Printf("%s %s\n", ui.ColorBold("Status:     "), ui.ColorSuccess("Active"))
	} else if total > 0 {
		fmt.Printf("%s %s\n", ui.ColorBold("Status:     "), ui.ColorWarning("No active deployment"))
	} else {
		fmt.Printf("%s %s\n", ui.ColorBold("Status:     "), ui.ColorGray("No deployments"))
	}

	fmt.Printf("%s %s\n", ui.ColorBold("Created:    "), ui.FormatTime(meta.CreatedAt))
	fmt.Println()

	if hasActiveDeployment && activeDeployment != nil {
		fmt.Println(ui.ColorBold("Active Deployment:"))
		fmt.Printf("  Release ID:    %s\n", activeDeployment.ReleaseID)
		fmt.Printf("  Commit:        %s\n", activeDeployment.Commit)
		fmt.Printf("  Port:          %d\n", activeDeployment.Port)
		fmt.Printf("  Status:        %s\n", ui.ColorSuccess(activeDeployment.Status))
		fmt.Printf("  Deployed:      %s\n", ui.FormatTime(activeDeployment.DeployedAt))
		fmt.Println()
	}

	if total > 0 {
		fmt.Println(ui.ColorBold("Deployment Statistics:"))
		fmt.Printf("  Total:         %d\n", total)
		fmt.Printf("  Active:        %d\n", active)
		fmt.Printf("  Standby:       %d\n", standby)
		fmt.Printf("  Stopped:       %d\n", stopped)
		fmt.Printf("  Failed:        %d\n", failed)
		fmt.Println()
	}

	if len(deployments) > 0 {
		fmt.Println(ui.ColorBold("Recent Deployments:"))
		limit := min(5, len(deployments))
		for i := range limit {
			d := deployments[i]
			statusStr := formatDeploymentStatus(d.Status)
			fmt.Printf("  %s  %-8s  %4d  %s\n",
				d.ReleaseID,
				statusStr,
				d.Port,
				ui.FormatTime(d.DeployedAt),
			)
		}
		fmt.Println()
	}

	appDir := config.GetAppDir(appName)
	fmt.Println(ui.ColorBold("Paths:"))
	fmt.Printf("  App Directory:    %s\n", appDir)
	fmt.Printf("  Git Repository:   %s\n", filepath.Join(appDir, "repo.git"))
	fmt.Printf("  Environment:      %s\n", config.GetAppEnvPath(appName))
	fmt.Printf("  Releases:         %s\n", filepath.Join(appDir, "releases"))
	fmt.Println()

	return nil
}

func formatDeploymentStatus(status string) string {
	switch status {
	case app.DeploymentStatusActive:
		return ui.ColorSuccess(status)
	case app.DeploymentStatusStandby:
		return ui.ColorInfo(status)
	case app.DeploymentStatusFailed:
		return ui.ColorError(status)
	case app.DeploymentStatusStopped:
		return ui.ColorGray(status)
	default:
		return status
	}
}
