package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dmitrymomot/runlite/internal/config"
)

const (
	lockFileName       = ".deploy.lock"
	defaultLockTimeout = 30 * time.Minute
)

var ErrDeploymentInProgress = errors.New("deployment already in progress")

type DeployLock struct {
	PID       int       `json:"pid"`
	ReleaseID string    `json:"release_id"`
	StartTime time.Time `json:"start_time"`
}

func AcquireLock(ctx context.Context, appName, releaseID string) error {
	lockPath := filepath.Join(config.GetAppDir(appName), lockFileName)
	timeout := getLockTimeout()

	lock := DeployLock{
		PID:       os.Getpid(),
		ReleaseID: releaseID,
		StartTime: time.Now(),
	}

	// Try to create lock file exclusively (atomic operation)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create lock file: %w", err)
		}

		// Lock file exists, check if it's stale
		if err := checkStaleLock(lockPath, timeout); err != nil {
			return err
		}

		// Stale lock was removed, try again
		f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to create lock file after stale removal: %w", err)
		}
	}
	defer f.Close()

	// Write lock data
	if err := json.NewEncoder(f).Encode(lock); err != nil {
		os.Remove(lockPath)
		return fmt.Errorf("failed to write lock data: %w", err)
	}

	return nil
}

func checkStaleLock(lockPath string, timeout time.Duration) error {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("failed to read existing lock: %w", err)
	}

	var existing DeployLock
	if err := json.Unmarshal(data, &existing); err != nil {
		// Corrupted lock file, remove it
		os.Remove(lockPath)
		return nil
	}

	// Check if lock is still valid
	if isProcessAlive(existing.PID) && time.Since(existing.StartTime) < timeout {
		return fmt.Errorf("%w: release %s started at %s (PID %d)",
			ErrDeploymentInProgress, existing.ReleaseID,
			existing.StartTime.Format(time.RFC3339), existing.PID)
	}

	// Lock is stale, remove it
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove stale lock: %w", err)
	}

	return nil
}

func ReleaseLock(ctx context.Context, appName string) error {
	lockPath := filepath.Join(config.GetAppDir(appName), lockFileName)
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	return nil
}

func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func getLockTimeout() time.Duration {
	if val := os.Getenv("RUNLITE_DEPLOY_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultLockTimeout
}
