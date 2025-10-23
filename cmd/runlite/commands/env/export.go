package env

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newExportCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "export <file>",
		Short: "Export environment variables to a file",
		Long: `Export environment variables to a file.

This command reads variables from the application's environment file at:
  /var/lib/runlite/apps/<app-name>/env

And writes them to the specified destination file.

Output format:
  KEY=VALUE
  ANOTHER_KEY=another value

Behavior:
  • Creates the destination file if it doesn't exist
  • By default, fails if the destination file already exists
  • Use --force to overwrite existing files
  • Output is sorted by key name

Use cases:
  • Backup environment configuration
  • Share configuration with team members
  • Migrate settings to another environment
  • Create template .env files

Examples:
  # Export to a file
  runlite env myapp export .env.backup

  # Overwrite existing file
  runlite env myapp export .env.backup --force

  # Export to a custom location
  runlite env myapp export /backups/myapp-env-$(date +%Y%m%d).env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			destFile := args[0]
			return runExport(cmd, appName, destFile, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite destination file if it exists")

	return cmd
}

func runExport(cmd *cobra.Command, appName string, destFile string, force bool) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Read environment
	envPath := getEnvFilePath(appName)
	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Check if environment is empty
	if len(vars) == 0 {
		return fmt.Errorf("environment is empty, nothing to export")
	}

	// Check if destination exists
	if _, err := os.Stat(destFile); err == nil && !force {
		return fmt.Errorf("destination file already exists: %s (use --force to overwrite)", destFile)
	}

	// Show preview
	cmd.Println()
	cmd.Println(ui.SubtitleStyle.Render("Export Preview"))
	cmd.Println()
	cmd.Printf("  %s Source: %s\n", ui.IconFile, envPath)
	cmd.Printf("  %s Destination: %s\n", ui.IconFile, destFile)
	cmd.Printf("  %s Variables: %d\n", ui.IconPackage, len(vars))
	cmd.Println()

	// Show sample of variables (first 5)
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	// Sort keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	cmd.Println(ui.MutedStyle.Render("  Sample:"))
	sampleCount := 5
	if len(keys) < sampleCount {
		sampleCount = len(keys)
	}
	for i := 0; i < sampleCount; i++ {
		key := keys[i]
		value := vars[key]
		if len(value) > 40 {
			value = value[:37] + "..."
		}
		cmd.Printf("    %s=%s\n", ui.BoldStyle.Render(key), ui.MutedStyle.Render(value))
	}
	if len(keys) > sampleCount {
		cmd.Printf("    %s\n", ui.MutedStyle.Render(fmt.Sprintf("... and %d more", len(keys)-sampleCount)))
	}

	cmd.Println()

	// Confirm
	var confirmed bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Export to this file?").
				Value(&confirmed).
				Affirmative("Export").
				Negative("Cancel"),
		),
	)

	if err := confirmForm.Run(); err != nil || !confirmed {
		return fmt.Errorf("export cancelled")
	}

	// Write to destination
	if err := writeEnvFile(destFile, vars); err != nil {
		return fmt.Errorf("write destination file: %w", err)
	}

	// Show success message
	cmd.Println()
	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s Environment exported successfully", ui.IconSuccess)))
	cmd.Println()
	cmd.Printf("Exported %s to:\n  %s\n",
		ui.BoldStyle.Render(fmt.Sprintf("%d variable(s)", len(vars))),
		ui.InfoStyle.Render(destFile))

	return nil
}
