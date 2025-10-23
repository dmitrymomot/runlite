package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dmitrymomot/runlite/internal/installer"
	"github.com/dmitrymomot/runlite/internal/installer/helpers"
)

// InstallConfig holds all configuration values for template installation.
type InstallConfig struct {
	Domains     []string
	Port        int
	DataDir     string
	DBPath      string // Optional: only if provided via flag
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
}

// NewInstallCommand creates the main 'install' command with template support.
func NewInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install infrastructure components from a template",
		Long: `Install infrastructure components from a template.

Installs all components defined in the template manifest with
automated configuration from command-line flags.`,
		RunE: runInstallTemplate,
	}

	// Template flags
	cmd.Flags().String("template", "", "Template to install (e.g., 'simple')")

	// Configuration flags
	cmd.Flags().StringSlice("domains", nil, "Domain names (comma-separated)")
	cmd.Flags().Int("port", 8000, "Internal application port")
	cmd.Flags().String("data-dir", "/var/lib/runlite", "Data directory")
	cmd.Flags().String("db-path", "", "Database path for backups (optional)")

	// S3 backup flags
	cmd.Flags().String("s3-bucket", "", "S3 bucket for backups (optional)")
	cmd.Flags().String("s3-region", "us-east-1", "S3 region")
	cmd.Flags().String("s3-access-key-id", "", "S3 access key ID (optional)")
	cmd.Flags().String("s3-secret-access-key", "", "S3 secret access key (optional)")

	return cmd
}

// runInstallTemplate orchestrates template installation.
func runInstallTemplate(cmd *cobra.Command, args []string) error {
	// Get flags
	template, _ := cmd.Flags().GetString("template")
	if template == "" {
		return cmd.Help()
	}

	// Build config from flags
	domains, _ := cmd.Flags().GetStringSlice("domains")
	port, _ := cmd.Flags().GetInt("port")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	dbPath, _ := cmd.Flags().GetString("db-path")
	s3Bucket, _ := cmd.Flags().GetString("s3-bucket")
	s3Region, _ := cmd.Flags().GetString("s3-region")
	s3AccessKey, _ := cmd.Flags().GetString("s3-access-key-id")
	s3SecretKey, _ := cmd.Flags().GetString("s3-secret-access-key")

	config := InstallConfig{
		Domains:     domains,
		Port:        port,
		DataDir:     dataDir,
		DBPath:      dbPath,
		S3Bucket:    s3Bucket,
		S3Region:    s3Region,
		S3AccessKey: s3AccessKey,
		S3SecretKey: s3SecretKey,
	}

	// 1. Load template manifest
	manifestPath := filepath.Join("stubs", template, "manifest.yml")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	cmd.Printf("🎯 Installing template: %s\n", manifest.Name)
	cmd.Printf("📋 Description: %s\n", manifest.Description)
	cmd.Println()

	// 2. Validate prerequisites
	if err := validatePrerequisites(manifest.Requires); err != nil {
		return fmt.Errorf("prerequisites not met: %w", err)
	}

	// 3. Run installers in order
	ctx := cmd.Context()
	for i, installerName := range manifest.Installers {
		cmd.Printf("[%d/%d] Installing %s...\n", i+1, len(manifest.Installers), installerName)

		if err := runInstaller(ctx, installerName, manifest, config); err != nil {
			return fmt.Errorf("install %s: %w", installerName, err)
		}

		cmd.Printf("✓ %s installed\n\n", installerName)
	}

	// 4. Print post-install message
	if manifest.PostInstall != "" {
		cmd.Println(manifest.PostInstall)
	}

	return nil
}

// runInstaller calls the generic installer with universal template data.
func runInstaller(ctx context.Context, name string, manifest *installer.Manifest, config InstallConfig) error {
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
func validatePrerequisites(reqs installer.Prerequisites) error {
	// Check OS
	if reqs.OS != "" && runtime.GOOS != reqs.OS {
		return fmt.Errorf("unsupported OS: %s (required: %s)", runtime.GOOS, reqs.OS)
	}

	// Check architecture
	if len(reqs.Arch) > 0 {
		archSupported := slices.Contains(reqs.Arch, runtime.GOARCH)
		if !archSupported {
			return fmt.Errorf("unsupported architecture: %s (required: %v)", runtime.GOARCH, reqs.Arch)
		}
	}

	// Check systemd availability
	if reqs.Systemd {
		if err := checkSystemd(); err != nil {
			return fmt.Errorf("systemd required but not available: %w", err)
		}
	}

	// Check root privileges
	if reqs.Root {
		if os.Geteuid() != 0 {
			return fmt.Errorf("root privileges required (run with sudo)")
		}
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
