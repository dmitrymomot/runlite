package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

func CleanupOldReleases(ctx context.Context, appName string, keepCount int) error {
	releases, err := ListReleases(ctx, appName)
	if err != nil {
		return fmt.Errorf("list releases: %w", err)
	}

	manager := app.NewManager(config.GetDataDir())

	var successfulReleases []string
	for _, releaseID := range releases {
		deployment, err := manager.GetDeploymentByReleaseID(appName, releaseID)
		if err != nil {
			slog.WarnContext(ctx, "deployment not found in metadata, skipping",
				"app", appName,
				"release", releaseID,
				"error", err)
			continue
		}

		if deployment.Status != app.DeploymentStatusFailed {
			successfulReleases = append(successfulReleases, releaseID)
		}
	}

	if len(successfulReleases) <= keepCount {
		slog.InfoContext(ctx, "no releases to clean up",
			"app", appName,
			"total_successful", len(successfulReleases),
			"keep_count", keepCount)
		return nil
	}

	toDelete := successfulReleases[keepCount:]

	for _, releaseID := range toDelete {
		if err := cleanupRelease(ctx, appName, releaseID); err != nil {
			slog.ErrorContext(ctx, "failed to cleanup release",
				"app", appName,
				"release", releaseID,
				"error", err)
			continue
		}

		slog.InfoContext(ctx, "cleaned up release",
			"app", appName,
			"release", releaseID)
	}

	return nil
}

func cleanupRelease(ctx context.Context, appName, releaseID string) error {
	unitName := fmt.Sprintf("runlite-%s@%s", appName, releaseID)

	if err := StopService(ctx, unitName); err != nil {
		slog.WarnContext(ctx, "failed to stop service",
			"unit", unitName,
			"error", err)
	}

	if err := DisableService(ctx, unitName); err != nil {
		slog.WarnContext(ctx, "failed to disable service",
			"unit", unitName,
			"error", err)
	}

	releaseDir := GetReleaseDir(appName, releaseID)
	if err := os.RemoveAll(releaseDir); err != nil {
		return fmt.Errorf("remove release directory: %w", err)
	}

	return nil
}
