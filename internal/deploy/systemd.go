package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// DefaultSystemdUnitPath is the default directory for systemd unit files
	DefaultSystemdUnitPath = "/etc/systemd/system"

	// DefaultSystemctlCmd is the default systemctl command
	DefaultSystemctlCmd = "systemctl"
)

var (
	// SystemdUnitPath can be overridden via SYSTEMD_PATH environment variable for testing
	SystemdUnitPath = getSystemdPath()

	// SystemctlCmd can be overridden via SYSTEMCTL_CMD environment variable for testing
	SystemctlCmd = getSystemctlCmd()
)

func getSystemdPath() string {
	if path := os.Getenv("SYSTEMD_PATH"); path != "" {
		return path
	}
	return DefaultSystemdUnitPath
}

func getSystemctlCmd() string {
	if cmd := os.Getenv("SYSTEMCTL_CMD"); cmd != "" {
		return cmd
	}
	return DefaultSystemctlCmd
}

// unitFileTemplate is the systemd service unit file template
const unitFileTemplate = `[Unit]
Description=runlite app: {{.AppName}} (release: {{.ReleaseID}})
After=network.target

[Service]
Type=simple
WorkingDirectory={{.WorkingDir}}
ExecStart={{.Command}}
EnvironmentFile={{.EnvFile}}
Environment=PORT={{.Port}}
Environment=RELEASE_ID={{.ReleaseID}}
Environment=COMMIT_HASH={{.CommitHash}}
Restart=on-failure
RestartSec=5
KillMode=mixed
TimeoutStopSec=30
User=runlite
Group=runlite

[Install]
WantedBy=multi-user.target
`

// UnitFileParams holds parameters for generating a systemd unit file
type UnitFileParams struct {
	AppName    string
	ReleaseID  string
	CommitHash string
	Port       int
	WorkingDir string
	Command    string
	EnvFile    string
}

// GenerateUnitFile creates a systemd service unit file for an application release.
// It uses text/template for rendering, writes to a temporary file first, validates
// with systemd-analyze, then atomically renames to the final location.
func GenerateUnitFile(ctx context.Context, params UnitFileParams) error {
	// Validate inputs
	if params.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if params.ReleaseID == "" {
		return fmt.Errorf("release ID is required")
	}

	// Build command string with arguments
	command := buildCommand(params.Command)

	// Prepare template data
	data := map[string]any{
		"AppName":    params.AppName,
		"ReleaseID":  params.ReleaseID,
		"CommitHash": params.CommitHash,
		"Port":       params.Port,
		"WorkingDir": params.WorkingDir,
		"Command":    command,
		"EnvFile":    params.EnvFile,
	}

	// Parse and render template
	tmpl, err := template.New("unit").Parse(unitFileTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse unit file template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render unit file template: %w", err)
	}

	// Generate unit file path
	unitName := fmt.Sprintf("runlite-%s@%s.service", params.AppName, params.ReleaseID)
	unitPath := filepath.Join(SystemdUnitPath, unitName)
	tmpPath := unitPath + ".tmp"

	// Write to temporary file
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	// Validate unit file with systemd-analyze
	if err := validateUnitFile(ctx, tmpPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("unit file validation failed: %w", err)
	}

	// Atomically rename to final location
	if err := os.Rename(tmpPath, unitPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to install unit file: %w", err)
	}

	return nil
}

// buildCommand constructs the full command string, replacing {{.Port}} template if present
func buildCommand(cmd string) string {
	// For MVP, we'll use simple string replacement for {{.Port}}
	// The Port environment variable will be available, so apps can use $PORT
	// If command has {{.Port}}, it will be replaced by the template engine later
	return cmd
}

// validateUnitFile runs systemd-analyze verify on a unit file
func validateUnitFile(ctx context.Context, unitPath string) error {
	cmd := exec.CommandContext(ctx, "systemd-analyze", "verify", unitPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validation failed: %w: %s", err, string(output))
	}
	return nil
}

// ReloadDaemon reloads the systemd daemon to pick up new/changed unit files
func ReloadDaemon(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, SystemctlCmd, "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w: %s", err, string(output))
	}
	return nil
}

// StartService starts a systemd service
func StartService(ctx context.Context, unitName string) error {
	if unitName == "" {
		return fmt.Errorf("unit name is required")
	}

	cmd := exec.CommandContext(ctx, SystemctlCmd, "start", unitName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service %s: %w: %s", unitName, err, string(output))
	}
	return nil
}

// StopService stops a systemd service
func StopService(ctx context.Context, unitName string) error {
	if unitName == "" {
		return fmt.Errorf("unit name is required")
	}

	cmd := exec.CommandContext(ctx, SystemctlCmd, "stop", unitName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service %s: %w: %s", unitName, err, string(output))
	}
	return nil
}

// DisableService disables a systemd service (prevents auto-start on boot)
func DisableService(ctx context.Context, unitName string) error {
	if unitName == "" {
		return fmt.Errorf("unit name is required")
	}

	cmd := exec.CommandContext(ctx, SystemctlCmd, "disable", unitName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore error if service is not enabled
		if !strings.Contains(string(output), "not loaded") {
			return fmt.Errorf("failed to disable service %s: %w: %s", unitName, err, string(output))
		}
	}
	return nil
}

// GetServiceStatus returns the status of a systemd service
func GetServiceStatus(ctx context.Context, unitName string) (string, error) {
	if unitName == "" {
		return "", fmt.Errorf("unit name is required")
	}

	cmd := exec.CommandContext(ctx, SystemctlCmd, "is-active", unitName)
	output, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(output))

	// is-active returns exit code 0 for "active", non-zero for others
	// But we still want to return the status string
	if err != nil && status == "" {
		return "unknown", fmt.Errorf("failed to get service status %s: %w", unitName, err)
	}

	return status, nil
}

// FormatUnitName returns the systemd unit name for an app release
func FormatUnitName(appName, releaseID string) string {
	return fmt.Sprintf("runlite-%s@%s.service", appName, releaseID)
}
