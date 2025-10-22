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
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func cleanupCommand() *cli.Command {
	return &cli.Command{
		Name:      "cleanup",
		Usage:     "Clean up old releases",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "keep",
				Usage:    "Number of successful releases to keep",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "yes",
				Usage: "Skip confirmation prompt",
			},
		},
		Action: cleanupAction,
	}
}

func cleanupAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)
	keepCount := cmd.Int("keep")

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	manager := app.NewManager(config.GetDataDir())
	if !manager.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' not found", appName)), 1)
	}

	if keepCount < 1 {
		return cli.Exit(ui.ErrorString("keep count must be at least 1"), 1)
	}

	releases, err := deploy.ListReleases(ctx, appName)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to list releases: %v", err)), 1)
	}

	var successfulReleases []string
	var failedReleases []string

	for _, releaseID := range releases {
		deployment, err := manager.GetDeploymentByReleaseID(appName, releaseID)
		if err != nil {
			continue
		}

		if deployment.Status != app.DeploymentStatusFailed {
			successfulReleases = append(successfulReleases, releaseID)
		} else {
			failedReleases = append(failedReleases, releaseID)
		}
	}

	if len(successfulReleases) <= keepCount {
		ui.PrintInfo(fmt.Sprintf("No cleanup needed - only %d successful release(s) found", len(successfulReleases)))
		return nil
	}

	toDelete := successfulReleases[keepCount:]

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Will delete %d old release(s), keeping %d most recent:", len(toDelete), keepCount))
	fmt.Println()

	for _, releaseID := range toDelete {
		fmt.Printf("  • %s\n", releaseID)
	}

	fmt.Println()
	if len(failedReleases) > 0 {
		ui.PrintInfo(fmt.Sprintf("Will keep %d failed release(s) for debugging:", len(failedReleases)))
		fmt.Println()
		for _, releaseID := range failedReleases {
			fmt.Printf("  • %s (failed)\n", releaseID)
		}
		fmt.Println()
	}

	if !cmd.Bool("yes") {
		fmt.Print("Continue? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to read input: %v", err)), 1)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			ui.PrintInfo("Cleanup cancelled")
			return nil
		}
	}

	fmt.Println()
	ui.PrintInfo("Starting cleanup...")

	if err := deploy.CleanupOldReleases(ctx, appName, keepCount); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("cleanup failed: %v", err)), 1)
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Cleanup complete - deleted %d release(s), kept %d", len(toDelete), keepCount))

	return nil
}
