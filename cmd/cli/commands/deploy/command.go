package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
	"github.com/dmitrymomot/runlite/internal/spec"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Category:  "Deployment",
		Usage:     "Deploy an application from git repository",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ref",
				Usage: "Git reference to deploy (commit, tag, or branch)",
				Value: "HEAD",
			},
		},
		Action: deployAction,
		Commands: []*cli.Command{
			logsCommand(),
		},
	}
}

func deployAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)
	gitRef := cmd.String("ref")

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	mgr := app.NewManager(config.GetDataDir())
	if !mgr.Exists(appName) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("app '%s' not found", appName)), 1)
	}

	gitRepoPath := filepath.Join(config.GetAppDir(appName), "repo.git")
	if _, err := os.Stat(gitRepoPath); os.IsNotExist(err) {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("git repository not found for app '%s'", appName)), 1)
	}

	ui.PrintInfo(fmt.Sprintf("Deploying app '%s' from ref '%s'...", appName, gitRef))

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("runlite-deploy-%s-*", appName))
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to create temp directory: %v", err)), 1)
	}
	defer os.RemoveAll(tempDir)

	checkoutCmd := exec.CommandContext(ctx, "git",
		"--git-dir="+gitRepoPath,
		"--work-tree="+tempDir,
		"checkout",
		"--force",
		gitRef,
	)
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to checkout ref '%s': %v\n%s", gitRef, err, output)), 1)
	}

	specPath := filepath.Join(tempDir, "runlite.yml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		return cli.Exit(ui.ErrorString("runlite.yml not found in repository"), 1)
	}

	appSpec, err := spec.ParseFile(specPath)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to parse runlite.yml: %v", err)), 1)
	}

	revParseCmd := exec.CommandContext(ctx, "git",
		"--git-dir="+gitRepoPath,
		"rev-parse",
		gitRef,
	)
	commitHashOutput, err := revParseCmd.Output()
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to resolve commit hash: %v", err)), 1)
	}
	commitHash := strings.TrimSpace(string(commitHashOutput))

	ui.PrintInfo(fmt.Sprintf("Starting deployment of commit %s...", commitHash[:7]))

	if err := deploy.Deploy(ctx, appName, commitHash, appSpec, tempDir); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("deployment failed: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("App '%s' deployed successfully", appName))

	return nil
}
