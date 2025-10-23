package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmitrymomot/runlite/internal/installer"
)

// Install performs a complete installation following a standard flow:
// 1. Download binary (direct or from archive)
// 2. Extract if needed
// 3. Create system user if specified
// 4. Create config directories
// 5. Render config template
// 6. Set ownership if specified
// 7. Install systemd service
// 8. Enable and start service
func Install(ctx context.Context, cfg installer.InstallerConfig, templateData any) error {
	// 1. Download binary
	fmt.Println("📦 Downloading binary...")
	if err := downloadAndInstallBinary(ctx, cfg); err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	fmt.Println("✓ Binary installed")

	// 2. Create system user if specified
	if cfg.SystemUser != "" {
		fmt.Printf("👤 Creating %s user...\n", cfg.SystemUser)
		if err := CreateSystemUser(ctx, cfg.SystemUser); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		fmt.Println("✓ User created")
	}

	// 3. Create config directory
	fmt.Println("📁 Creating config directory...")
	configDir := filepath.Dir(cfg.ConfigPath)
	if err := CreateDirectory(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// 4. Render config template
	fmt.Println("📝 Generating configuration...")
	if err := RenderTemplate(cfg.ConfigTemplate, cfg.ConfigPath, templateData); err != nil {
		return fmt.Errorf("render config: %w", err)
	}
	fmt.Println("✓ Configuration generated")

	// 5. Set ownership if specified
	if cfg.Owner != "" {
		if err := SetOwner(ctx, configDir, cfg.Owner); err != nil {
			return fmt.Errorf("set ownership: %w", err)
		}
	}

	// 6. Install systemd service
	fmt.Println("🔧 Installing systemd service...")
	serviceName := extractServiceName(cfg.ServiceFile)
	if err := InstallService(ctx, cfg.ServiceFile, serviceName); err != nil {
		return fmt.Errorf("install service: %w", err)
	}

	// 7. Enable and start service
	fmt.Printf("🚀 Starting %s...\n", serviceName)
	if err := EnableAndStart(ctx, serviceName); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	fmt.Printf("✓ %s installed and running\n", serviceName)
	return nil
}

// downloadAndInstallBinary handles both direct binary downloads and archive extraction.
func downloadAndInstallBinary(ctx context.Context, cfg installer.InstallerConfig) error {
	// Check if this is an archive that needs extraction
	if cfg.ExtractFile != "" {
		// Download to temp location
		archivePath := "/tmp/runlite-download.tar.gz"
		if err := DownloadBinary(ctx, cfg.DownloadURL, archivePath); err != nil {
			return fmt.Errorf("download archive: %w", err)
		}
		defer os.Remove(archivePath)

		// Extract specific file from archive
		if err := ExtractFromTarGz(archivePath, cfg.ExtractFile, cfg.BinaryPath); err != nil {
			return fmt.Errorf("extract binary: %w", err)
		}
	} else {
		// Direct binary download
		if err := DownloadBinary(ctx, cfg.DownloadURL, cfg.BinaryPath); err != nil {
			return fmt.Errorf("download binary: %w", err)
		}
	}

	return nil
}

// extractServiceName extracts service name from file path.
// E.g., "stubs/simple/systemd/caddy.service" -> "caddy"
func extractServiceName(servicePath string) string {
	base := filepath.Base(servicePath)
	return strings.TrimSuffix(base, ".service")
}
