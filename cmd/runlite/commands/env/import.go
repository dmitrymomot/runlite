package env

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
)

func newImportCommand() *cobra.Command {
	var merge bool

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import environment variables from a file",
		Long: `Import environment variables from a file.

This command reads variables from a source file and writes them to the
application's environment file at:
  /var/lib/runlite/apps/<app-name>/env

Source file format:
  KEY=VALUE
  ANOTHER_KEY=another value

Behavior:
  • By default, REPLACES all existing variables
  • Use --merge to keep existing variables and add new ones
  • Validates all variables before writing
  • Creates app environment file if it doesn't exist

Use cases:
  • Import production settings from a file
  • Restore environment from a backup
  • Clone settings from another environment

Examples:
  # Import and replace all variables
  runlite env myapp import .env.production

  # Import and merge with existing variables
  runlite env myapp import .env.production --merge

  # Import from a custom location
  runlite env myapp import /path/to/config/.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app name from parent command
			appName := cmd.Parent().Flags().Args()[0]
			sourceFile := args[0]
			return runImport(cmd, appName, sourceFile, merge)
		},
	}

	cmd.Flags().BoolVar(&merge, "merge", false, "Merge with existing variables instead of replacing")

	return cmd
}

func runImport(cmd *cobra.Command, appName string, sourceFile string, merge bool) error {
	// Validate app exists
	if err := validateAppExists(appName); err != nil {
		return err
	}

	// Read source file
	newVars, err := readEnvFile(sourceFile)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Check if source is empty
	if len(newVars) == 0 {
		return fmt.Errorf("source file is empty or contains no valid variables")
	}

	// Read current environment
	envPath := getEnvFilePath(appName)
	currentVars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}

	// Determine final variables
	var finalVars map[string]string
	var added, updated, kept int

	if merge {
		// Merge mode: keep existing + add new
		finalVars = make(map[string]string)

		// Copy existing
		for k, v := range currentVars {
			finalVars[k] = v
		}

		// Add/update from source
		for k, v := range newVars {
			if oldValue, exists := finalVars[k]; exists {
				if oldValue != v {
					updated++
				} else {
					kept++
				}
			} else {
				added++
			}
			finalVars[k] = v
		}
	} else {
		// Replace mode: use source as-is
		finalVars = newVars
		added = len(newVars)
	}

	// Show preview and confirm
	cmd.Println()
	cmd.Println(ui.SubtitleStyle.Render("Import Preview"))
	cmd.Println()

	if merge {
		if added > 0 {
			cmd.Printf("  %s %d variable(s) will be added\n", ui.SuccessStyle.Render("+"), added)
		}
		if updated > 0 {
			cmd.Printf("  %s %d variable(s) will be updated\n", ui.InfoStyle.Render("~"), updated)
		}
		if kept > 0 {
			cmd.Printf("  %s %d variable(s) kept unchanged\n", ui.MutedStyle.Render("="), kept)
		}
		cmd.Println()
		cmd.Printf("Total after merge: %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d variable(s)", len(finalVars))))
	} else {
		cmd.Printf("  %s Current variables will be replaced\n", ui.WarningStyle.Render("!"))
		cmd.Printf("  %s %d variable(s) in source file\n", ui.InfoStyle.Render("→"), len(newVars))
		cmd.Println()
		if len(currentVars) > 0 {
			cmd.Println(ui.WarningStyle.Render("  ⚠ All existing variables will be lost"))
		}
	}

	cmd.Println()

	// Confirm
	var confirmed bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Import these variables?").
				Value(&confirmed).
				Affirmative("Import").
				Negative("Cancel"),
		),
	)

	if err := confirmForm.Run(); err != nil || !confirmed {
		return fmt.Errorf("import cancelled")
	}

	// Write the environment
	if err := writeEnvFile(envPath, finalVars); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}

	// Show success message
	cmd.Println()
	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s Environment imported successfully", ui.IconSuccess)))
	cmd.Println()
	cmd.Printf("Total: %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d variable(s)", len(finalVars))))

	return nil
}
