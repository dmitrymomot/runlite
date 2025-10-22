package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func logsCommand() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Usage:     "Show deployment logs",
		ArgsUsage: "APP [RELEASE_ID]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
			&cli.IntFlag{
				Name:    "tail",
				Aliases: []string{"n"},
				Usage:   "Show last N lines",
				Value:   0,
			},
		},
		Action: logsAction,
	}
}

func logsAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit(ui.ErrorString("app name is required"), 1)
	}

	appName := cmd.Args().Get(0)
	var releaseID string

	if err := config.ValidateAppName(appName); err != nil {
		return cli.Exit(ui.ErrorString(err.Error()), 1)
	}

	if cmd.Args().Len() > 1 {
		releaseID = cmd.Args().Get(1)
	} else {
		releases, err := deploy.ListReleases(ctx, appName)
		if err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to list releases: %v", err)), 1)
		}

		if len(releases) == 0 {
			return cli.Exit(ui.ErrorString("no releases found for this app"), 1)
		}

		releaseID = releases[0]
	}

	logPath := deploy.GetReleaseDir(appName, releaseID) + "/deploy.log"

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("no deployment logs found for release %s", releaseID)), 1)
		}
		return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to open log file: %v", err)), 1)
	}
	defer file.Close()

	tailLines := cmd.Int("tail")
	followMode := cmd.Bool("follow")

	if tailLines > 0 {
		if err := printTailLines(file, tailLines); err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to read logs: %v", err)), 1)
		}
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("error reading logs: %v", err)), 1)
		}
	}

	if followMode {
		if err := followLogs(ctx, logPath); err != nil {
			return cli.Exit(ui.ErrorString(fmt.Sprintf("failed to follow logs: %v", err)), 1)
		}
	}

	return nil
}

func printTailLines(file *os.File, n int) error {
	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

func followLogs(ctx context.Context, logPath string) error {
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek to end: %w", err)
	}

	scanner := bufio.NewScanner(file)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
		}
	}
}
