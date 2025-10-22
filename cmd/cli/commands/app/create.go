package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new application",
		ArgsUsage: "NAME",
		Action:    createAction,
	}
}

func createAction(ctx context.Context, cmd *cli.Command) error {
	// TODO(deployment): Caddy and Litestream integration
	// This command creates the app structure and git repository.
	// Caddy reverse proxy configuration and Litestream backup setup
	// will be handled during the first deployment (git push hook).
	// Assumptions:
	// - Caddy is installed and running on the server
	// - Litestream is installed and running on the server
	// - Git post-receive hook will read runlite.yml and configure services

	// Get app name from arguments
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)

	// Validate app name
	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	// Create metadata manager
	mgr := app.NewManager(config.GetDataDir())

	// Check if app already exists
	if mgr.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' already exists", appName)), 1)
	}

	ui.PrintInfo(fmt.Sprintf("Creating app '%s'...", appName))

	// Create app directory structure and metadata
	if err := mgr.Create(appName); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to create app: %v", err)), 1)
	}

	// Initialize bare git repository
	appDir := config.GetAppDir(appName)
	gitRepoPath := filepath.Join(appDir, "repo.git")
	gitCmd := exec.CommandContext(ctx, "git", "init", "--bare", gitRepoPath)
	if output, err := gitCmd.CombinedOutput(); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to initialize git repository: %v\n%s", err, output)), 1)
	}

	// Install pre-receive hook
	hookScript, err := renderTemplate("pre-receive.tmpl", map[string]string{
		"AppName": appName,
		"DataDir": config.GetDataDir(),
	})
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to render hook template: %v", err)), 1)
	}

	hookPath := filepath.Join(gitRepoPath, "hooks", "pre-receive")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to write pre-receive hook: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("App '%s' created successfully", appName))
	fmt.Println()
	ui.PrintInfo("Next steps:")
	fmt.Printf("  1. Add git remote:\n     git remote add runlite %s\n\n", gitRepoPath)
	fmt.Printf("  2. Push your code:\n     git push runlite main\n\n")
	ui.PrintWarning("Note: Ensure your project has a runlite.yml configuration file")

	return nil
}
