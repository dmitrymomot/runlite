package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/dmitrymomot/runlite/internal/config"
)

func GenerateReleaseID(commitHash string) string {
	timestamp := time.Now().Format("20060102150405")
	if len(commitHash) > 7 {
		commitHash = commitHash[:7]
	}
	return fmt.Sprintf("%s-%s", timestamp, commitHash)
}

func GetReleaseDir(appName, releaseID string) string {
	return filepath.Join(config.GetAppDir(appName), "releases", releaseID)
}

func EnsureReleaseDir(appName, releaseID string) error {
	releaseDir := GetReleaseDir(appName, releaseID)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return fmt.Errorf("create release directory: %w", err)
	}
	return nil
}

func ListReleases(ctx context.Context, appName string) ([]string, error) {
	releasesDir := filepath.Join(config.GetAppDir(appName), "releases")

	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read releases directory: %w", err)
	}

	var releases []string
	for _, entry := range entries {
		if entry.IsDir() {
			releases = append(releases, entry.Name())
		}
	}

	slices.Sort(releases)
	slices.Reverse(releases)

	return releases, nil
}
