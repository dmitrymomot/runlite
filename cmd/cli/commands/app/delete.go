package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm", "remove"},
		Usage:     "Delete an application and all its data",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: deleteAction,
	}
}

func deleteAction(ctx context.Context, cmd *cli.Command) error {
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

	if !cmd.Bool("force") {
		fmt.Println()
		ui.PrintWarning("WARNING: This will permanently delete:")
		fmt.Printf("  - Application '%s'\n", appName)
		fmt.Println("  - All deployments and releases")
		fmt.Println("  - Git repository")
		fmt.Println("  - Environment variables")
		fmt.Println("  - All configuration")
		fmt.Println()
		fmt.Println("This action cannot be undone.")
		fmt.Println()
		fmt.Print("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to read input: %v", err)), 1)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			ui.PrintInfo("Deletion cancelled")
			return nil
		}
	}

	fmt.Println()
	ui.PrintInfo("Deleting application...")

	meta, err := manager.Load(appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to load metadata: %v", err)), 1)
	}

	for _, deployment := range meta.Deployments {
		if deployment.Status == app.DeploymentStatusActive || deployment.Status == app.DeploymentStatusStandby {
			unitName := deploy.FormatUnitName(appName, deployment.ReleaseID)

			ui.PrintInfo(fmt.Sprintf("Stopping service: %s", unitName))
			if err := deploy.StopService(ctx, unitName); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to stop service %s: %v", unitName, err))
			}

			if err := deploy.DisableService(ctx, unitName); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to disable service %s: %v", unitName, err))
			}
		}
	}

	ui.PrintInfo("Removing Caddy route...")
	caddyClient := caddy.NewClient("")
	if err := caddyClient.DeleteAppRoute(appName); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to delete Caddy route: %v", err))
	}

	ui.PrintInfo("Deleting app directory...")
	appDir := config.GetAppDir(appName)
	if err := os.RemoveAll(appDir); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to delete app directory: %v", err)), 1)
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Application '%s' deleted successfully", appName))

	return nil
}
