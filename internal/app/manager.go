package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager handles application metadata operations with controlled access.
type Manager struct {
	dataDir string // Base data directory (/var/lib/runlite)
}

// NewManager creates a new metadata manager.
func NewManager(dataDir string) *Manager {
	return &Manager{dataDir: dataDir}
}

// appDir returns the directory path for the given app.
func (m *Manager) appDir(appName string) string {
	return filepath.Join(m.dataDir, "apps", appName)
}

// metadataPath returns the metadata file path for the given app.
func (m *Manager) metadataPath(appName string) string {
	return filepath.Join(m.appDir(appName), "app.json")
}

// Load reads the app metadata from app.json.
func (m *Manager) Load(appName string) (*Metadata, error) {
	path := m.metadataPath(appName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// Save writes the app metadata to app.json atomically.
func (m *Manager) Save(meta *Metadata) error {
	meta.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	path := m.metadataPath(meta.Name)

	// Atomic write: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp metadata: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // cleanup on failure
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

// Exists checks if metadata file exists for the given app.
func (m *Manager) Exists(appName string) bool {
	_, err := os.Stat(m.metadataPath(appName))
	return err == nil
}

// Create creates initial metadata for a new app.
func (m *Manager) Create(appName string) error {
	if m.Exists(appName) {
		return fmt.Errorf("metadata already exists for app: %s", appName)
	}

	// Ensure app directory exists
	appDir := m.appDir(appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}

	meta := &Metadata{
		Name:        appName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Deployments: []Deployment{},
	}

	return m.Save(meta)
}

// AddDeployment adds a new deployment record to the metadata.
// This should be called AFTER a successful deployment.
func (m *Manager) AddDeployment(appName string, deployment Deployment) error {
	meta, err := m.Load(appName)
	if err != nil {
		return err
	}

	// Validate status
	if deployment.Status != DeploymentStatusActive &&
		deployment.Status != DeploymentStatusFailed {
		return fmt.Errorf("%w: new deployment must be active or failed", ErrInvalidStatus)
	}

	// If this is a new active deployment, transition existing deployments
	if deployment.Status == DeploymentStatusActive {
		for i := range meta.Deployments {
			switch meta.Deployments[i].Status {
			case DeploymentStatusActive:
				// Current active becomes standby
				meta.Deployments[i].Status = DeploymentStatusStandby
			case DeploymentStatusStandby:
				// Previous standby becomes stopped
				meta.Deployments[i].Status = DeploymentStatusStopped
				now := time.Now()
				meta.Deployments[i].StoppedAt = &now
			}
		}
	}

	// Add new deployment at the front
	meta.Deployments = append([]Deployment{deployment}, meta.Deployments...)

	// Cleanup: keep only last N deployments
	if len(meta.Deployments) > MaxDeploymentHistory {
		meta.Deployments = meta.Deployments[:MaxDeploymentHistory]
	}

	return m.Save(meta)
}

// GetActiveDeployment returns the currently active deployment.
func (m *Manager) GetActiveDeployment(appName string) (*Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	for i := range meta.Deployments {
		if meta.Deployments[i].Status == DeploymentStatusActive {
			return &meta.Deployments[i], nil
		}
	}

	return nil, ErrNoActiveDeployment
}

// GetStandbyDeployment returns the standby deployment (rollback target).
func (m *Manager) GetStandbyDeployment(appName string) (*Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	for i := range meta.Deployments {
		if meta.Deployments[i].Status == DeploymentStatusStandby {
			return &meta.Deployments[i], nil
		}
	}

	return nil, ErrNoStandbyDeployment
}

// ListDeployments returns all deployments for an app, sorted by deployed_at descending.
func (m *Manager) ListDeployments(appName string) ([]Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	// Already sorted by insertion order (newest first)
	return meta.Deployments, nil
}

// Rollback switches active and standby deployments.
// Returns the new active deployment (previously standby).
func (m *Manager) Rollback(appName string) (*Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	// Find active and standby indices
	activeIdx := -1
	standbyIdx := -1

	for i, d := range meta.Deployments {
		if d.Status == DeploymentStatusActive {
			activeIdx = i
		}
		if d.Status == DeploymentStatusStandby {
			standbyIdx = i
		}
	}

	if activeIdx == -1 {
		return nil, ErrNoActiveDeployment
	}
	if standbyIdx == -1 {
		return nil, ErrNoStandbyDeployment
	}

	// Swap statuses
	meta.Deployments[activeIdx].Status = DeploymentStatusStandby
	meta.Deployments[standbyIdx].Status = DeploymentStatusActive

	// Move new active to front (for consistency)
	newActive := meta.Deployments[standbyIdx]
	meta.Deployments = append(
		[]Deployment{newActive},
		append(
			meta.Deployments[:standbyIdx],
			meta.Deployments[standbyIdx+1:]...,
		)...,
	)

	if err := m.Save(meta); err != nil {
		return nil, err
	}

	return &newActive, nil
}

// MarkDeploymentStopped marks a specific deployment as stopped.
func (m *Manager) MarkDeploymentStopped(appName, releaseID string) error {
	meta, err := m.Load(appName)
	if err != nil {
		return err
	}

	found := false
	for i := range meta.Deployments {
		if meta.Deployments[i].ReleaseID == releaseID {
			meta.Deployments[i].Status = DeploymentStatusStopped
			now := time.Now()
			meta.Deployments[i].StoppedAt = &now
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("deployment not found: %s", releaseID)
	}

	return m.Save(meta)
}

// GetDeploymentByReleaseID returns a specific deployment by release ID.
func (m *Manager) GetDeploymentByReleaseID(appName, releaseID string) (*Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	for i := range meta.Deployments {
		if meta.Deployments[i].ReleaseID == releaseID {
			return &meta.Deployments[i], nil
		}
	}

	return nil, fmt.Errorf("deployment not found: %s", releaseID)
}

// GetDeploymentsByStatus returns all deployments with the given status.
func (m *Manager) GetDeploymentsByStatus(appName, status string) ([]Deployment, error) {
	meta, err := m.Load(appName)
	if err != nil {
		return nil, err
	}

	var result []Deployment
	for _, d := range meta.Deployments {
		if d.Status == status {
			result = append(result, d)
		}
	}

	return result, nil
}

// CleanupOldDeployments removes deployments older than the specified duration.
// Only removes deployments with status "stopped" or "failed".
func (m *Manager) CleanupOldDeployments(appName string, olderThan time.Duration) error {
	meta, err := m.Load(appName)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)
	var kept []Deployment

	for _, d := range meta.Deployments {
		// Always keep active and standby
		if d.Status == DeploymentStatusActive || d.Status == DeploymentStatusStandby {
			kept = append(kept, d)
			continue
		}

		// Keep recent stopped/failed deployments
		if d.DeployedAt.After(cutoff) {
			kept = append(kept, d)
			continue
		}

		// Remove old stopped/failed (not added to kept)
	}

	meta.Deployments = kept
	return m.Save(meta)
}

// GetStats returns deployment statistics for an app.
func (m *Manager) GetStats(appName string) (total, active, standby, failed, stopped int, err error) {
	meta, err := m.Load(appName)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	total = len(meta.Deployments)
	for _, d := range meta.Deployments {
		switch d.Status {
		case DeploymentStatusActive:
			active++
		case DeploymentStatusStandby:
			standby++
		case DeploymentStatusFailed:
			failed++
		case DeploymentStatusStopped:
			stopped++
		}
	}

	return total, active, standby, failed, stopped, nil
}
