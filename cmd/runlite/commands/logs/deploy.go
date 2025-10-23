package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/logger"
	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
	"github.com/dmitrymomot/runlite/internal/deploy"
)

func newDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [release-id]",
		Short: "View deployment logs",
		Long: `View deployment logs (deploy.log) from application releases.

Shows build output, deployment steps, and any errors that occurred during deployment.

If no release ID is provided, shows logs for the most recent release.

Examples:
  # View latest deployment logs
  runlite logs myapp deploy

  # View specific release deployment logs
  runlite logs myapp deploy 20251022143055-a3f5c2b

  # Follow deployment logs in real-time
  runlite logs myapp deploy --follow

  # Show last 50 lines and follow
  runlite logs myapp deploy --tail 50 --follow`,
		Args: cobra.RangeArgs(0, 1),
		RunE: runDeployLogs,
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().IntP("tail", "n", 0, "Show last N lines (0 = all)")

	return cmd
}

// runDeployLogsDefault is called when logs is invoked without a subcommand (e.g., `runlite logs myapp`)
func runDeployLogsDefault(cmd *cobra.Command, args []string) error {
	// In this case, args[0] is the app name
	appName := args[0]

	// Create a pseudo-args array for runDeployLogs (remaining args after app name)
	var releaseArgs []string
	if len(args) > 1 {
		releaseArgs = args[1:]
	}

	return runDeployLogsWithAppName(cmd, appName, releaseArgs)
}

func runDeployLogs(cmd *cobra.Command, args []string) error {
	// Get app name from parent command when called as a subcommand
	appName := cmd.Parent().Flags().Args()[0]
	return runDeployLogsWithAppName(cmd, appName, args)
}

func runDeployLogsWithAppName(cmd *cobra.Command, appName string, args []string) error {
	ctx := cmd.Context()

	// Validate app name
	if err := validateAppName(appName); err != nil {
		return err
	}

	// Get or determine release ID
	var releaseID string
	if len(args) > 0 {
		releaseID = args[0]
	} else {
		releases, err := deploy.ListReleases(ctx, appName)
		if err != nil {
			return fmt.Errorf("%s failed to list releases: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}

		if len(releases) == 0 {
			return fmt.Errorf("%s no releases found for app '%s'",
				ui.ErrorStyle.Render("Error:"), appName)
		}

		releaseID = releases[0]
	}

	// Construct log file path
	logPath := filepath.Join(deploy.GetReleaseDir(appName, releaseID), "deploy.log")

	// Open log file
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s no deployment logs found for release %s",
				ui.ErrorStyle.Render("Error:"), releaseID)
		}
		return fmt.Errorf("%s failed to open log file: %w",
			ui.ErrorStyle.Render("Error:"), err)
	}
	defer file.Close()

	// Create logger for output
	log := logger.NewWithPrefix("deploy")

	// Print header
	createLogHeader(log, appName, releaseID, "Deployment")

	// Get flags
	tailLines, _ := cmd.Flags().GetInt("tail")
	followMode, _ := cmd.Flags().GetBool("follow")

	// Print logs based on flags
	if tailLines > 0 {
		if err := printTailLines(file, tailLines, log); err != nil {
			return fmt.Errorf("%s failed to read logs: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}
	} else {
		// Print all lines
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			formatLogLine(scanner.Text(), log)
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("%s error reading logs: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}
	}

	// Follow mode if requested
	if followMode {
		log.Info(ui.MutedStyle.Render("\n--- Following logs (Ctrl+C to exit) ---\n"))
		if err := followLogs(ctx, logPath, log); err != nil {
			return fmt.Errorf("%s failed to follow logs: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}
	}

	return nil
}
