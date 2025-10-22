package app

import (
	"errors"
	"time"
)

const (
	// Maximum number of deployment records to keep in history
	MaxDeploymentHistory = 10

	// Deployment status values
	DeploymentStatusActive  = "active"  // Currently serving traffic
	DeploymentStatusStandby = "standby" // Last successful, ready for rollback
	DeploymentStatusFailed  = "failed"  // Health check failed, never went active
	DeploymentStatusStopped = "stopped" // Old release, cleaned up
)

var (
	ErrNoActiveDeployment  = errors.New("no active deployment found")
	ErrNoStandbyDeployment = errors.New("no standby deployment found")
	ErrInvalidStatus       = errors.New("invalid deployment status")
)

// Metadata represents the metadata for an application.
type Metadata struct {
	Name        string       `json:"name"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Deployments []Deployment `json:"deployments,omitempty"`
}

// Deployment represents a single deployment record.
type Deployment struct {
	ReleaseID         string     `json:"release_id"`
	Commit            string     `json:"commit"`
	Port              int        `json:"port"`
	Status            string     `json:"status"` // active, standby, failed, stopped
	BinaryPath        string     `json:"binary_path"`
	DeployedAt        time.Time  `json:"deployed_at"`
	StoppedAt         *time.Time `json:"stopped_at,omitempty"`
	HealthCheckPassed bool       `json:"health_check_passed"`
	FailureReason     string     `json:"failure_reason,omitempty"`
}
