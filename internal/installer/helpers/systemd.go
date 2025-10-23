package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallService copies a systemd service file to /etc/systemd/system/.
// sourcePath is the path to the .service file in the template.
// serviceName is the name of the service (e.g., "caddy", "litestream").
func InstallService(ctx context.Context, sourcePath, serviceName string) error {
	destPath := filepath.Join("/etc/systemd/system", serviceName+".service")

	// Read source file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read service file: %w", err)
	}

	// Write to destination (atomic)
	tmpFile := destPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	if err := os.Rename(tmpFile, destPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("install service file: %w", err)
	}

	return nil
}

// DaemonReload runs systemctl daemon-reload to pick up new service files.
func DaemonReload(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %w\n%s", err, output)
	}
	return nil
}

// EnableService enables a systemd service to start on boot.
func EnableService(ctx context.Context, serviceName string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "enable", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("enable service %s failed: %w\n%s", serviceName, err, output)
	}
	return nil
}

// StartService starts a systemd service.
func StartService(ctx context.Context, serviceName string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "start", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start service %s failed: %w\n%s", serviceName, err, output)
	}
	return nil
}

// EnableAndStart is a convenience function that daemon-reloads, enables, and starts a service.
func EnableAndStart(ctx context.Context, serviceName string) error {
	if err := DaemonReload(ctx); err != nil {
		return err
	}

	if err := EnableService(ctx, serviceName); err != nil {
		return err
	}

	if err := StartService(ctx, serviceName); err != nil {
		return err
	}

	return nil
}

// ServiceStatus checks if a service is running.
func ServiceStatus(ctx context.Context, serviceName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", serviceName)
	err := cmd.Run()
	if err != nil {
		// Service is not active
		return false, nil
	}
	return true, nil
}
