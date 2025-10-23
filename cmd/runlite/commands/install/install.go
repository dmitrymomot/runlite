package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/logger"
	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
	"github.com/dmitrymomot/runlite/internal/installer"
	"github.com/dmitrymomot/runlite/internal/installer/helpers"
)

// InstallConfig holds all configuration values for template installation.
type InstallConfig struct {
	Template    string
	Domains     []string
	Port        int
	DataDir     string
	DBPath      string // Optional: only if provided
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
}

// NewCommand creates the 'install' command with interactive forms.
func NewCommand() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "install [template]",
		Short: "Install infrastructure components from a template",
		Long: `Install infrastructure components from a template with an interactive setup wizard.

The installer will guide you through configuration with beautiful forms.
For CI/CD or scripting, use flags with --non-interactive mode.

Examples:
  # Interactive mode (recommended)
  runlite install

  # Non-interactive mode for automation
  runlite install --template simple --domains example.com --non-interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallTemplate(cmd, args, nonInteractive)
		},
	}

	// Template selection
	cmd.Flags().String("template", "simple", "Template to install (e.g., 'simple')")

	// Configuration flags (for non-interactive mode)
	cmd.Flags().StringSlice("domains", nil, "Domain names (comma-separated)")
	cmd.Flags().Int("port", 8000, "Internal application port")
	cmd.Flags().String("data-dir", "/var/lib/runlite", "Data directory")
	cmd.Flags().String("db-path", "", "Database path for backups (optional)")

	// S3 backup flags
	cmd.Flags().String("s3-bucket", "", "S3 bucket for backups (optional)")
	cmd.Flags().String("s3-region", "us-east-1", "S3 region")
	cmd.Flags().String("s3-access-key-id", "", "S3 access key ID (optional)")
	cmd.Flags().String("s3-secret-access-key", "", "S3 secret access key (optional)")

	// Mode flags
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode using flags")

	return cmd
}

// runInstallTemplate orchestrates template installation.
func runInstallTemplate(cmd *cobra.Command, args []string, nonInteractive bool) error {
	// Create logger
	logger := logger.NewDefault()

	// Show welcome banner
	if !nonInteractive {
		showWelcomeBanner(cmd)
	}

	// Get configuration
	var config InstallConfig
	var err error

	if nonInteractive {
		// Use flags
		config, err = getConfigFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	} else {
		// Interactive mode
		config, err = collectInteractiveConfig(cmd)
		if err != nil {
			return fmt.Errorf("configuration cancelled: %w", err)
		}
	}

	// Validate template exists
	manifestPath := filepath.Join("stubs", config.Template, "manifest.yml")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("%s template not found: %w", config.Template, err)
	}

	// Load template manifest
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Show template info
	if !nonInteractive {
		showTemplateInfo(cmd, manifest)
	} else {
		logger.Info("installing template",
			"name", manifest.Name,
			"version", manifest.Version,
		)
	}

	// Validate prerequisites
	cmd.Println()
	cmd.Println(ui.SubtitleStyle.Render("Validating Prerequisites"))
	cmd.Println()

	if err := validatePrerequisites(cmd, manifest.Requires, logger); err != nil {
		return fmt.Errorf("%s prerequisites not met: %w",
			ui.IconError, err)
	}

	cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s All prerequisites validated", ui.IconSuccess)))
	cmd.Println()

	// Run installers
	ctx := cmd.Context()
	cmd.Println(ui.SubtitleStyle.Render("Installing Components"))
	cmd.Println()

	for i, installerName := range manifest.Installers {
		progress := fmt.Sprintf("[%d/%d]", i+1, len(manifest.Installers))
		cmd.Printf("%s %s Installing %s...\n",
			ui.InfoStyle.Render(progress),
			ui.IconDownload,
			ui.BoldStyle.Render(installerName))

		if err := runInstaller(ctx, installerName, manifest, config, logger); err != nil {
			errMsg := fmt.Sprintf("%s Failed to install %s", ui.IconError, installerName)
			cmd.Println(ui.ErrorStyle.Render(errMsg))
			return fmt.Errorf("install %s: %w", installerName, err)
		}

		cmd.Println(ui.SuccessStyle.Render(fmt.Sprintf("%s %s installed successfully", ui.IconSuccess, installerName)))
		cmd.Println()
	}

	// Show success message
	showSuccessMessage(cmd, manifest)

	return nil
}

// showWelcomeBanner displays the welcome banner
func showWelcomeBanner(cmd *cobra.Command) {
	banner := ui.TitleStyle.Render(fmt.Sprintf("%s runlite Setup Wizard", ui.IconRocket))
	subtitle := ui.MutedStyle.Render("Self-hosted PaaS for Go applications")

	cmd.Println()
	cmd.Println(banner)
	cmd.Println(subtitle)
	cmd.Println()
}

// showTemplateInfo displays template information in a beautiful box
func showTemplateInfo(cmd *cobra.Command, manifest *installer.Manifest) {
	content := fmt.Sprintf(`%s Template Information

Name:        %s
Version:     %s
Description: %s
Author:      %s

Components:  %s`,
		ui.IconPackage,
		ui.BoldStyle.Render(manifest.Name),
		manifest.Version,
		manifest.Description,
		manifest.Author,
		strings.Join(manifest.Installers, ", "),
	)

	cmd.Println(ui.BoxStyle.Render(content))
}

// showSuccessMessage displays the post-install success message
func showSuccessMessage(cmd *cobra.Command, manifest *installer.Manifest) {
	cmd.Println()

	// Success header
	successHeader := ui.SuccessStyle.Render(fmt.Sprintf("%s Installation Complete!", ui.IconSuccess))
	cmd.Println(successHeader)
	cmd.Println()

	// Post-install message in a box
	if manifest.PostInstall != "" {
		cmd.Println(ui.SuccessBoxStyle.Render(manifest.PostInstall))
	}
}

// collectInteractiveConfig collects configuration via interactive forms
func collectInteractiveConfig(cmd *cobra.Command) (InstallConfig, error) {
	var config InstallConfig

	// Variables for form inputs
	var (
		domainsStr  string
		portStr     = "8000"
		dataDir     = "/var/lib/runlite"
		configureDB bool
		dbPath      string
		configureS3 bool
		s3Bucket    string
		s3Region    = "us-east-1"
		s3AccessKey string
		s3SecretKey string
	)

	// Get template from flag if provided, otherwise use default
	template, _ := cmd.Flags().GetString("template")
	if template == "" {
		template = "simple"
	}
	config.Template = template

	// Create interactive form
	form := huh.NewForm(
		// Basic Configuration
		huh.NewGroup(
			huh.NewNote().
				Title("Basic Configuration").
				Description("Configure basic settings for your runlite installation"),

			huh.NewInput().
				Title("Domain names").
				Description("Enter domain names separated by commas (e.g., example.com,app.example.com)").
				Placeholder("example.com,api.example.com").
				Value(&domainsStr).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("at least one domain is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Internal application port").
				Description("Port for the application to listen on").
				Value(&portStr).
				Placeholder("8000").
				Validate(func(s string) error {
					port, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("must be a valid number")
					}
					if port < 1024 || port > 65535 {
						return fmt.Errorf("must be between 1024 and 65535")
					}
					return nil
				}),

			huh.NewInput().
				Title("Data directory").
				Description("Directory where runlite will store application data").
				Value(&dataDir).
				Placeholder("/var/lib/runlite"),
		),

		// Optional Database Path
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure custom database path?").
				Description("By default, databases are stored in the data directory").
				Value(&configureDB),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Database path").
				Description("Custom path for database files").
				Value(&dbPath).
				Placeholder("/var/lib/runlite/db"),
		).WithHideFunc(func() bool { return !configureDB }),

		// Optional S3 Backups
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure S3 backups?").
				Description("Enable automatic backups to AWS S3 or compatible storage").
				Value(&configureS3),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("S3 Bucket").
				Description("Name of the S3 bucket for backups").
				Value(&s3Bucket).
				Placeholder("my-runlite-backups").
				Validate(func(s string) error {
					if configureS3 && strings.TrimSpace(s) == "" {
						return fmt.Errorf("S3 bucket is required when S3 backups are enabled")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("S3 Region").
				Description("AWS region where your S3 bucket is located").
				Options(
					huh.NewOption("US East (N. Virginia)", "us-east-1"),
					huh.NewOption("US East (Ohio)", "us-east-2"),
					huh.NewOption("US West (N. California)", "us-west-1"),
					huh.NewOption("US West (Oregon)", "us-west-2"),
					huh.NewOption("Europe (Ireland)", "eu-west-1"),
					huh.NewOption("Europe (London)", "eu-west-2"),
					huh.NewOption("Europe (Frankfurt)", "eu-central-1"),
					huh.NewOption("Asia Pacific (Singapore)", "ap-southeast-1"),
					huh.NewOption("Asia Pacific (Tokyo)", "ap-northeast-1"),
				).
				Value(&s3Region),

			huh.NewInput().
				Title("S3 Access Key ID").
				Description("AWS access key for S3 authentication").
				Value(&s3AccessKey).
				Placeholder("AKIAIOSFODNN7EXAMPLE"),

			huh.NewInput().
				Title("S3 Secret Access Key").
				Description("AWS secret key for S3 authentication").
				Value(&s3SecretKey).
				EchoMode(huh.EchoModePassword).
				Placeholder("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
		).WithHideFunc(func() bool { return !configureS3 }),

		// Confirmation
		huh.NewGroup(
			huh.NewConfirm().
				Title("Ready to install?").
				Description("This will install the selected components with the configured settings").
				Affirmative("Install").
				Negative("Cancel"),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return config, err
	}

	// Parse and build config
	config.Domains = parseDomainsString(domainsStr)
	config.Port, _ = strconv.Atoi(portStr)
	config.DataDir = dataDir

	if configureDB {
		config.DBPath = dbPath
	}

	if configureS3 {
		config.S3Bucket = s3Bucket
		config.S3Region = s3Region
		config.S3AccessKey = s3AccessKey
		config.S3SecretKey = s3SecretKey
	}

	return config, nil
}

// getConfigFromFlags extracts configuration from command flags
func getConfigFromFlags(cmd *cobra.Command) (InstallConfig, error) {
	template, _ := cmd.Flags().GetString("template")
	domains, _ := cmd.Flags().GetStringSlice("domains")
	port, _ := cmd.Flags().GetInt("port")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	dbPath, _ := cmd.Flags().GetString("db-path")
	s3Bucket, _ := cmd.Flags().GetString("s3-bucket")
	s3Region, _ := cmd.Flags().GetString("s3-region")
	s3AccessKey, _ := cmd.Flags().GetString("s3-access-key-id")
	s3SecretKey, _ := cmd.Flags().GetString("s3-secret-access-key")

	if template == "" {
		return InstallConfig{}, fmt.Errorf("--template is required in non-interactive mode")
	}

	if len(domains) == 0 {
		return InstallConfig{}, fmt.Errorf("--domains is required in non-interactive mode")
	}

	return InstallConfig{
		Template:    template,
		Domains:     domains,
		Port:        port,
		DataDir:     dataDir,
		DBPath:      dbPath,
		S3Bucket:    s3Bucket,
		S3Region:    s3Region,
		S3AccessKey: s3AccessKey,
		S3SecretKey: s3SecretKey,
	}, nil
}

// parseDomainsString parses comma-separated domains into a slice
func parseDomainsString(s string) []string {
	parts := strings.Split(s, ",")
	domains := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			domains = append(domains, trimmed)
		}
	}
	return domains
}

// runInstaller calls the generic installer with universal template data.
func runInstaller(ctx context.Context, name string, manifest *installer.Manifest, config InstallConfig, logger *log.Logger) error {
	// Get installer config from manifest
	installerCfg, ok := manifest.InstallerConfig[name]
	if !ok {
		return fmt.Errorf("no config found for installer: %s", name)
	}

	// Build universal template data map (all installers receive same data)
	templateData := map[string]any{
		"Domains":     config.Domains,
		"Port":        config.Port,
		"DataDir":     config.DataDir,
		"BackupPath":  filepath.Join(config.DataDir, "backups"),
		"S3Bucket":    config.S3Bucket,
		"S3Region":    config.S3Region,
		"S3AccessKey": config.S3AccessKey,
		"S3SecretKey": config.S3SecretKey,
	}

	// Only include DBPath if explicitly provided
	if config.DBPath != "" {
		templateData["DBPath"] = config.DBPath
	}

	// Log installation
	logger.Info("installing component",
		"name", name,
		"data_dir", config.DataDir,
	)

	// Run generic installer
	return helpers.Install(ctx, installerCfg, templateData)
}

// loadManifest loads and parses a manifest.yml file.
func loadManifest(path string) (*installer.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest installer.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

// validatePrerequisites checks if system meets template requirements.
func validatePrerequisites(cmd *cobra.Command, reqs installer.Prerequisites, logger *log.Logger) error {
	// Check OS
	if reqs.OS != "" && runtime.GOOS != reqs.OS {
		return fmt.Errorf("unsupported OS: %s (required: %s)", runtime.GOOS, reqs.OS)
	}
	cmd.Printf("  %s Operating system: %s\n",
		ui.SuccessStyle.Render(ui.IconCheck),
		ui.BoldStyle.Render(runtime.GOOS))

	// Check architecture
	if len(reqs.Arch) > 0 {
		archSupported := slices.Contains(reqs.Arch, runtime.GOARCH)
		if !archSupported {
			return fmt.Errorf("unsupported architecture: %s (required: %v)", runtime.GOARCH, reqs.Arch)
		}
	}
	cmd.Printf("  %s Architecture: %s\n",
		ui.SuccessStyle.Render(ui.IconCheck),
		ui.BoldStyle.Render(runtime.GOARCH))

	// Check systemd availability
	if reqs.Systemd {
		if err := checkSystemd(); err != nil {
			return fmt.Errorf("systemd required but not available: %w", err)
		}
		cmd.Printf("  %s Systemd: %s\n",
			ui.SuccessStyle.Render(ui.IconCheck),
			ui.SuccessStyle.Render("available"))
	}

	// Check root privileges
	if reqs.Root {
		if os.Geteuid() != 0 {
			return fmt.Errorf("root privileges required (run with sudo)")
		}
		cmd.Printf("  %s Privileges: %s\n",
			ui.SuccessStyle.Render(ui.IconCheck),
			ui.SuccessStyle.Render("root"))
	}

	return nil
}

// checkSystemd verifies systemd is available on the system.
func checkSystemd() error {
	// Method 1: Check if systemd runtime directory exists
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return nil
	}

	// Method 2: Try running systemctl
	cmd := exec.Command("systemctl", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl not found or not working")
	}

	return nil
}
