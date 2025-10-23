package installer

// Manifest defines the template metadata and configuration.
// Based on SETUP_TEMPLATES.md specification.
type Manifest struct {
	Name            string                     `yaml:"name"`
	Version         string                     `yaml:"version"`
	Description     string                     `yaml:"description"`
	Author          string                     `yaml:"author"`
	Requires        Prerequisites              `yaml:"requires"`
	Installers      []string                   `yaml:"installers"`       // List of installer names to run
	InstallerConfig map[string]InstallerConfig `yaml:"installer_config"` // Configuration for each installer
	PostInstall     string                     `yaml:"post_install"`
}

// InstallerConfig contains paths and URLs for a specific installer.
type InstallerConfig struct {
	DownloadURL    string `yaml:"download_url"`    // Download URL (static, no templates)
	ExtractFile    string `yaml:"extract_file"`    // File to extract from archive (optional, for .tar.gz)
	BinaryPath     string `yaml:"binary_path"`     // Where to install binary
	ConfigTemplate string `yaml:"config_template"` // Path to config template file
	ConfigPath     string `yaml:"config_path"`     // Where to write rendered config
	ServiceFile    string `yaml:"service_file"`    // Path to systemd service file
	SystemUser     string `yaml:"system_user"`     // System user to create (empty = skip)
	Owner          string `yaml:"owner"`           // Owner for config files (e.g., "caddy:caddy", empty = skip)
}

// Prerequisites defines system requirements for a template.
type Prerequisites struct {
	OS      string   `yaml:"os"`      // e.g., "linux"
	Arch    []string `yaml:"arch"`    // e.g., ["amd64", "arm64"]
	Systemd bool     `yaml:"systemd"` // Requires systemd
	Root    bool     `yaml:"root"`    // Requires root/sudo
}
