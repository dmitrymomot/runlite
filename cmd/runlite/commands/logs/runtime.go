package logs

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/logger"
	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

func newRuntimeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime [release-id]",
		Short: "View runtime logs from systemd",
		Long: `View runtime logs from systemd journal (journalctl).

Shows application output, errors, and stdout/stderr from the running service.

If no release ID is provided, shows logs for the currently active release.

Examples:
  # View active runtime logs
  runlite logs myapp runtime

  # View specific release runtime logs
  runlite logs myapp runtime 20251022143055-a3f5c2b

  # Follow runtime logs in real-time
  runlite logs myapp runtime --follow

  # Show last 100 lines and follow
  runlite logs myapp runtime --tail 100 --follow`,
		Args: cobra.RangeArgs(0, 1),
		RunE: runRuntimeLogs,
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().IntP("tail", "n", 50, "Show last N lines (default: 50)")

	return cmd
}

func runRuntimeLogs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get app name from parent command
	appName := cmd.Parent().Flags().Args()[0]

	// Validate app name
	if err := validateAppName(appName); err != nil {
		return err
	}

	// Get or determine release ID
	var releaseID string
	if len(args) > 0 {
		releaseID = args[0]
	} else {
		// Use active deployment by default
		mgr := app.NewManager(config.GetDataDir())
		activeDeployment, err := mgr.GetActiveDeployment(appName)
		if err != nil {
			return fmt.Errorf("%s failed to get active deployment: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}

		releaseID = activeDeployment.ReleaseID
	}

	// Construct systemd unit name
	unitName := fmt.Sprintf("runlite-%s@%s.service", appName, releaseID)

	// Get flags
	tailLines, _ := cmd.Flags().GetInt("tail")
	followMode, _ := cmd.Flags().GetBool("follow")

	// Create logger for output
	log := logger.NewWithPrefix("runtime")

	// Print header
	createLogHeader(log, appName, releaseID, "Runtime")
	log.Info(ui.MutedStyle.Render(fmt.Sprintf("Service: %s", unitName)))
	log.Info(ui.MutedStyle.Render(strings.Repeat("─", 60)))

	// Build journalctl command
	journalArgs := []string{
		"-u", unitName,
		"--no-pager",
		"--output=short-iso",
	}

	if tailLines > 0 {
		journalArgs = append(journalArgs, "-n", fmt.Sprintf("%d", tailLines))
	} else {
		journalArgs = append(journalArgs, "--no-tail")
	}

	if followMode {
		journalArgs = append(journalArgs, "-f")
	}

	// Execute journalctl
	journalCmd := exec.CommandContext(ctx, "journalctl", journalArgs...)

	stdout, err := journalCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%s failed to create stdout pipe: %w",
			ui.ErrorStyle.Render("Error:"), err)
	}

	if err := journalCmd.Start(); err != nil {
		return fmt.Errorf("%s failed to start journalctl: %w",
			ui.ErrorStyle.Render("Error:"), err)
	}

	// Read and format output
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		formatLogLine(scanner.Text(), log)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s error reading logs: %w",
			ui.ErrorStyle.Render("Error:"), err)
	}

	// Wait for command to finish
	if err := journalCmd.Wait(); err != nil {
		// Ignore exit errors from Ctrl+C in follow mode
		if !followMode {
			return fmt.Errorf("%s journalctl failed: %w",
				ui.ErrorStyle.Render("Error:"), err)
		}
	}

	return nil
}
