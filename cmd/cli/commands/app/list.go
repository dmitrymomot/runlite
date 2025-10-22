package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all applications",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: listAction,
	}
}

type appListItem struct {
	Name             string    `json:"name"`
	ActiveRelease    string    `json:"active_release"`
	Port             int       `json:"port"`
	DeploymentsCount int       `json:"deployments_count"`
	CreatedAt        time.Time `json:"created_at"`
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	dataDir := config.GetDataDir()
	appsDir := filepath.Join(dataDir, "apps")

	entries, err := os.ReadDir(appsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No applications found")
			return nil
		}
		return fmt.Errorf("failed to read apps directory: %w", err)
	}

	manager := app.NewManager(dataDir)

	var items []appListItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()

		metadata, err := manager.Load(appName)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to load metadata for app %q: %v", appName, err))
			continue
		}

		item := appListItem{
			Name:             appName,
			ActiveRelease:    "none",
			Port:             0,
			DeploymentsCount: len(metadata.Deployments),
			CreatedAt:        metadata.CreatedAt,
		}

		activeDep, err := manager.GetActiveDeployment(appName)
		if err == nil && activeDep != nil {
			item.ActiveRelease = activeDep.ReleaseID
			item.Port = activeDep.Port
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		fmt.Println("No applications found")
		return nil
	}

	if cmd.Bool("json") {
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%-20s %-25s %-8s %-15s %s\n", "NAME", "ACTIVE RELEASE", "PORT", "DEPLOYMENTS", "CREATED")
	for _, item := range items {
		port := "-"
		if item.Port > 0 {
			port = fmt.Sprintf("%d", item.Port)
		}
		created := item.CreatedAt.Format("2006-01-02")
		fmt.Printf("%-20s %-25s %-8s %-15d %s\n",
			item.Name,
			item.ActiveRelease,
			port,
			item.DeploymentsCount,
			created,
		)
	}

	return nil
}
