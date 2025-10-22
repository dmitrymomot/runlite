package logs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Category:  "Deployment",
		Usage:     "Show application logs",
		ArgsUsage: "APP",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show",
				Value:   100,
			},
		},
		Action: logsAction,
	}
}

func logsAction(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Args().Get(0)
	if appName == "" {
		return fmt.Errorf("app name is required")
	}

	if err := config.ValidateAppName(appName); err != nil {
		return err
	}

	dataDir := config.GetDataDir()
	mgr := app.NewManager(dataDir)

	if !mgr.Exists(appName) {
		return fmt.Errorf("app does not exist: %s", appName)
	}

	activeDeployment, err := mgr.GetActiveDeployment(appName)
	if err != nil {
		if err == app.ErrNoActiveDeployment {
			return fmt.Errorf("no active deployment for app '%s'", appName)
		}
		return fmt.Errorf("failed to get active deployment: %w", err)
	}

	unitName := deploy.FormatUnitName(appName, activeDeployment.ReleaseID)

	follow := cmd.Bool("follow")
	lines := cmd.Int("lines")

	args := []string{
		"-u", unitName,
		"--no-pager",
		"--output=cat",
	}

	if follow {
		args = append(args, "-f")
	} else {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	execCtx := ctx
	if follow {
		var cancel context.CancelFunc
		execCtx, cancel = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer cancel()
	}

	journalctl := exec.CommandContext(execCtx, "journalctl", args...)
	journalctl.Stdout = os.Stdout
	journalctl.Stderr = os.Stderr

	if err := journalctl.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 127 {
				return fmt.Errorf("journalctl command not found (systemd required)")
			}
		}
		if execCtx.Err() == context.Canceled {
			return nil
		}
		return fmt.Errorf("failed to read logs: %w", err)
	}

	return nil
}
