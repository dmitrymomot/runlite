package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Metadata represents the metadata for an application.
type Metadata struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"` // created, deployed, stopped, failed
}

// Save writes the app metadata to app.json in the app directory.
func (m *Metadata) Save(appDir string) error {
	metadataPath := filepath.Join(appDir, "app.json")

	m.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

// Load reads the app metadata from app.json in the app directory.
func Load(appDir string) (*Metadata, error) {
	metadataPath := filepath.Join(appDir, "app.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// Exists checks if metadata file exists for the given app directory.
func Exists(appDir string) bool {
	metadataPath := filepath.Join(appDir, "app.json")
	_, err := os.Stat(metadataPath)
	return err == nil
}
