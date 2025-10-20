package internal

import "time"

// AppInfo represents basic application information
type AppInfo struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Port       int       `json:"port"`
	Release    string    `json:"release"`
	Commit     string    `json:"commit"`
	DeployedAt time.Time `json:"deployed_at"`
	Domains    []string  `json:"domains"`
	Health     string    `json:"health"`
}

// AppDetail represents detailed application information
type AppDetail struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Current   DeploymentInfo `json:"current"`
	Previous  DeploymentInfo `json:"previous"`
	Domains   []string       `json:"domains"`
	Databases []DatabaseInfo `json:"databases"`
}

// DeploymentInfo represents deployment information
type DeploymentInfo struct {
	Release    string    `json:"release"`
	Commit     string    `json:"commit"`
	Port       int       `json:"port"`
	BinaryPath string    `json:"binary_path"`
	DeployedAt time.Time `json:"deployed_at"`
}

// DatabaseInfo represents database information
type DatabaseInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastBackup   time.Time `json:"last_backup"`
	BackupStatus string    `json:"backup_status"`
}

// DeployProgress represents deployment progress updates
type DeployProgress struct {
	Stage       string `json:"stage"` // "cloning", "building", "starting", etc.
	Message     string `json:"message"`
	Percent     int    `json:"percent"`
	BuildOutput string `json:"build_output"` // Verbose build logs
	Error       error  `json:"error,omitempty"`

	// Completion info
	Release string   `json:"release,omitempty"`
	Port    int      `json:"port,omitempty"`
	Domains []string `json:"domains,omitempty"`
}

// CreateAppResult represents the result of creating an app
type CreateAppResult struct {
	Name       string    `json:"name"`
	GitRemote  string    `json:"git_remote"`
	ConfigPath string    `json:"config_path"`
	CreatedAt  time.Time `json:"created_at"`
}

// RollbackResult represents the result of a rollback
type RollbackResult struct {
	Release    string    `json:"release"`
	Port       int       `json:"port"`
	DeployedAt time.Time `json:"deployed_at"`
}
