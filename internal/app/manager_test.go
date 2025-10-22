package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/app"
)

// TestNewManager verifies manager construction.
func TestNewManager(t *testing.T) {
	t.Parallel()

	manager := app.NewManager("/test/path")
	require.NotNil(t, manager)
}

// TestCreate tests initial metadata creation.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates metadata file with correct structure", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("myapp")
		require.NoError(t, err)

		// Verify file exists
		metaPath := filepath.Join(dataDir, "apps", "myapp", "app.json")
		require.FileExists(t, metaPath)

		// Verify JSON structure
		data, err := os.ReadFile(metaPath)
		require.NoError(t, err)

		var meta app.Metadata
		err = json.Unmarshal(data, &meta)
		require.NoError(t, err)

		assert.Equal(t, "myapp", meta.Name)
		assert.NotZero(t, meta.CreatedAt)
		assert.NotZero(t, meta.UpdatedAt)
		assert.Empty(t, meta.Deployments)
	})

	t.Run("creates app directory if missing", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("newapp")
		require.NoError(t, err)

		appDir := filepath.Join(dataDir, "apps", "newapp")
		stat, err := os.Stat(appDir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())
	})

	t.Run("fails when metadata already exists", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("duplicate")
		require.NoError(t, err)

		// Second create should fail
		err = manager.Create("duplicate")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// TestExists verifies metadata existence checking.
func TestExists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing metadata", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("existing")
		require.NoError(t, err)

		assert.True(t, manager.Exists("existing"))
	})

	t.Run("returns false for non-existent metadata", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		assert.False(t, manager.Exists("nonexistent"))
	})
}

// TestLoadAndSave tests metadata persistence.
func TestLoadAndSave(t *testing.T) {
	t.Parallel()

	t.Run("load returns created metadata", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("loadtest")
		require.NoError(t, err)

		meta, err := manager.Load("loadtest")
		require.NoError(t, err)
		assert.Equal(t, "loadtest", meta.Name)
	})

	t.Run("load fails for non-existent app", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		_, err := manager.Load("missing")
		require.Error(t, err)
	})

	t.Run("save updates UpdatedAt timestamp", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("timestamptest")
		require.NoError(t, err)

		meta1, err := manager.Load("timestamptest")
		require.NoError(t, err)
		originalUpdated := meta1.UpdatedAt

		time.Sleep(10 * time.Millisecond) // Ensure time difference

		err = manager.Save(meta1)
		require.NoError(t, err)

		meta2, err := manager.Load("timestamptest")
		require.NoError(t, err)

		assert.True(t, meta2.UpdatedAt.After(originalUpdated))
	})

	t.Run("save performs atomic write", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("atomictest")
		require.NoError(t, err)

		meta, err := manager.Load("atomictest")
		require.NoError(t, err)

		// Add deployment
		meta.Deployments = []app.Deployment{
			{
				ReleaseID: "rel-1",
				Status:    app.DeploymentStatusActive,
			},
		}

		err = manager.Save(meta)
		require.NoError(t, err)

		// Verify temp file doesn't exist
		tmpPath := filepath.Join(dataDir, "apps", "atomictest", "app.json.tmp")
		_, err = os.Stat(tmpPath)
		assert.True(t, os.IsNotExist(err))

		// Verify actual file has new data
		reloaded, err := manager.Load("atomictest")
		require.NoError(t, err)
		assert.Len(t, reloaded.Deployments, 1)
	})

	t.Run("load fails on corrupted JSON", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		// Create app directory and write corrupted JSON
		appDir := filepath.Join(dataDir, "apps", "corrupt")
		err := os.MkdirAll(appDir, 0755)
		require.NoError(t, err)

		metaPath := filepath.Join(appDir, "app.json")
		err = os.WriteFile(metaPath, []byte("{invalid json}"), 0644)
		require.NoError(t, err)

		_, err = manager.Load("corrupt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")
	})
}

// TestAddDeployment tests deployment addition and state transitions.
func TestAddDeployment(t *testing.T) {
	t.Parallel()

	t.Run("adds first active deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		deployment := app.Deployment{
			ReleaseID:         "rel-1",
			Commit:            "abc123",
			Port:              8080,
			Status:            app.DeploymentStatusActive,
			BinaryPath:        "/path/to/binary",
			DeployedAt:        time.Now(),
			HealthCheckPassed: true,
		}

		err = manager.AddDeployment("app1", deployment)
		require.NoError(t, err)

		meta, err := manager.Load("app1")
		require.NoError(t, err)

		require.Len(t, meta.Deployments, 1)
		assert.Equal(t, "rel-1", meta.Deployments[0].ReleaseID)
		assert.Equal(t, app.DeploymentStatusActive, meta.Deployments[0].Status)
	})

	t.Run("transitions active to standby when adding new active", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		// Add first active deployment
		dep1 := app.Deployment{
			ReleaseID:  "rel-1",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app2", dep1)
		require.NoError(t, err)

		// Add second active deployment
		dep2 := app.Deployment{
			ReleaseID:  "rel-2",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app2", dep2)
		require.NoError(t, err)

		meta, err := manager.Load("app2")
		require.NoError(t, err)

		require.Len(t, meta.Deployments, 2)
		assert.Equal(t, "rel-2", meta.Deployments[0].ReleaseID)
		assert.Equal(t, app.DeploymentStatusActive, meta.Deployments[0].Status)
		assert.Equal(t, "rel-1", meta.Deployments[1].ReleaseID)
		assert.Equal(t, app.DeploymentStatusStandby, meta.Deployments[1].Status)
	})

	t.Run("transitions standby to stopped when adding new active", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app3")
		require.NoError(t, err)

		// Add three active deployments to test full transition chain
		for i := 1; i <= 3; i++ {
			dep := app.Deployment{
				ReleaseID:  "rel-" + string(rune('0'+i)),
				Status:     app.DeploymentStatusActive,
				DeployedAt: time.Now(),
			}
			err = manager.AddDeployment("app3", dep)
			require.NoError(t, err)
			time.Sleep(1 * time.Millisecond) // Ensure ordering
		}

		meta, err := manager.Load("app3")
		require.NoError(t, err)

		require.Len(t, meta.Deployments, 3)
		assert.Equal(t, app.DeploymentStatusActive, meta.Deployments[0].Status)  // rel-3
		assert.Equal(t, app.DeploymentStatusStandby, meta.Deployments[1].Status) // rel-2
		assert.Equal(t, app.DeploymentStatusStopped, meta.Deployments[2].Status) // rel-1
		assert.NotNil(t, meta.Deployments[2].StoppedAt)
	})

	t.Run("adds failed deployment without transitions", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app4")
		require.NoError(t, err)

		// Add active deployment
		dep1 := app.Deployment{
			ReleaseID:  "rel-1",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app4", dep1)
		require.NoError(t, err)

		// Add failed deployment
		dep2 := app.Deployment{
			ReleaseID:     "rel-2",
			Status:        app.DeploymentStatusFailed,
			DeployedAt:    time.Now(),
			FailureReason: "health check timeout",
		}
		err = manager.AddDeployment("app4", dep2)
		require.NoError(t, err)

		meta, err := manager.Load("app4")
		require.NoError(t, err)

		require.Len(t, meta.Deployments, 2)
		assert.Equal(t, app.DeploymentStatusFailed, meta.Deployments[0].Status)
		assert.Equal(t, app.DeploymentStatusActive, meta.Deployments[1].Status) // Unchanged
	})

	t.Run("rejects invalid deployment status", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app5")
		require.NoError(t, err)

		dep := app.Deployment{
			ReleaseID:  "rel-1",
			Status:     app.DeploymentStatusStandby, // Invalid for new deployment
			DeployedAt: time.Now(),
		}

		err = manager.AddDeployment("app5", dep)
		require.Error(t, err)
		assert.ErrorIs(t, err, app.ErrInvalidStatus)
	})

	t.Run("enforces max deployment history", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app6")
		require.NoError(t, err)

		// Add 15 deployments (max is 10)
		for i := 1; i <= 15; i++ {
			dep := app.Deployment{
				ReleaseID:  "rel-" + string(rune('0'+i)),
				Status:     app.DeploymentStatusActive,
				DeployedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			err = manager.AddDeployment("app6", dep)
			require.NoError(t, err)
		}

		meta, err := manager.Load("app6")
		require.NoError(t, err)

		// Should only keep last 10
		assert.Len(t, meta.Deployments, app.MaxDeploymentHistory)
		// Newest should be first
		assert.Equal(t, "rel-?", meta.Deployments[0].ReleaseID) // rel-15
	})
}

// TestGetActiveDeployment tests active deployment retrieval.
func TestGetActiveDeployment(t *testing.T) {
	t.Parallel()

	t.Run("returns active deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		dep := app.Deployment{
			ReleaseID:  "active-1",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app1", dep)
		require.NoError(t, err)

		active, err := manager.GetActiveDeployment("app1")
		require.NoError(t, err)
		assert.Equal(t, "active-1", active.ReleaseID)
	})

	t.Run("returns error when no active deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		_, err = manager.GetActiveDeployment("app2")
		require.Error(t, err)
		assert.ErrorIs(t, err, app.ErrNoActiveDeployment)
	})

	t.Run("returns error for non-existent app", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		_, err := manager.GetActiveDeployment("missing")
		require.Error(t, err)
	})
}

// TestGetStandbyDeployment tests standby deployment retrieval.
func TestGetStandbyDeployment(t *testing.T) {
	t.Parallel()

	t.Run("returns standby deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add two active deployments to create standby
		dep1 := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{ReleaseID: "rel-2", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep2)
		require.NoError(t, err)

		standby, err := manager.GetStandbyDeployment("app1")
		require.NoError(t, err)
		assert.Equal(t, "rel-1", standby.ReleaseID)
	})

	t.Run("returns error when no standby deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		// Only add one active deployment
		dep := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app2", dep)
		require.NoError(t, err)

		_, err = manager.GetStandbyDeployment("app2")
		require.Error(t, err)
		assert.ErrorIs(t, err, app.ErrNoStandbyDeployment)
	})
}

// TestRollback tests deployment rollback logic.
func TestRollback(t *testing.T) {
	t.Parallel()

	t.Run("swaps active and standby deployments", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add two deployments
		dep1 := app.Deployment{ReleaseID: "rel-1", Port: 8080, Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{ReleaseID: "rel-2", Port: 8081, Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep2)
		require.NoError(t, err)

		// Rollback
		newActive, err := manager.Rollback("app1")
		require.NoError(t, err)
		assert.Equal(t, "rel-1", newActive.ReleaseID)
		assert.Equal(t, 8080, newActive.Port)

		// Verify states
		active, err := manager.GetActiveDeployment("app1")
		require.NoError(t, err)
		assert.Equal(t, "rel-1", active.ReleaseID)

		standby, err := manager.GetStandbyDeployment("app1")
		require.NoError(t, err)
		assert.Equal(t, "rel-2", standby.ReleaseID)
	})

	t.Run("moves new active to front of deployments list", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		// Add two deployments
		dep1 := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app2", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{ReleaseID: "rel-2", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app2", dep2)
		require.NoError(t, err)

		// Rollback
		_, err = manager.Rollback("app2")
		require.NoError(t, err)

		// Verify ordering
		deployments, err := manager.ListDeployments("app2")
		require.NoError(t, err)
		assert.Equal(t, "rel-1", deployments[0].ReleaseID) // New active at front
		assert.Equal(t, app.DeploymentStatusActive, deployments[0].Status)
	})

	t.Run("fails when no active deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app3")
		require.NoError(t, err)

		_, err = manager.Rollback("app3")
		require.Error(t, err)
		assert.ErrorIs(t, err, app.ErrNoActiveDeployment)
	})

	t.Run("fails when no standby deployment", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app4")
		require.NoError(t, err)

		// Add only one deployment
		dep := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app4", dep)
		require.NoError(t, err)

		_, err = manager.Rollback("app4")
		require.Error(t, err)
		assert.ErrorIs(t, err, app.ErrNoStandbyDeployment)
	})
}

// TestListDeployments tests deployment listing.
func TestListDeployments(t *testing.T) {
	t.Parallel()

	t.Run("returns deployments sorted newest first", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add deployments with explicit timing
		baseTime := time.Now()
		for i := 1; i <= 3; i++ {
			dep := app.Deployment{
				ReleaseID:  "rel-" + string(rune('0'+i)),
				Status:     app.DeploymentStatusActive,
				DeployedAt: baseTime.Add(time.Duration(i) * time.Second),
			}
			err = manager.AddDeployment("app1", dep)
			require.NoError(t, err)
		}

		deployments, err := manager.ListDeployments("app1")
		require.NoError(t, err)

		require.Len(t, deployments, 3)
		// Newest first (rel-3 added last)
		assert.Equal(t, "rel-3", deployments[0].ReleaseID)
		assert.Equal(t, "rel-1", deployments[2].ReleaseID)
	})

	t.Run("returns empty slice when no deployments", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		deployments, err := manager.ListDeployments("app2")
		require.NoError(t, err)
		assert.Empty(t, deployments)
	})

	t.Run("fails for non-existent app", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		_, err := manager.ListDeployments("missing")
		require.Error(t, err)
	})
}

// TestMarkDeploymentStopped tests marking deployments as stopped.
func TestMarkDeploymentStopped(t *testing.T) {
	t.Parallel()

	t.Run("marks deployment as stopped with timestamp", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		dep := app.Deployment{
			ReleaseID:  "rel-1",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app1", dep)
		require.NoError(t, err)

		before := time.Now()
		err = manager.MarkDeploymentStopped("app1", "rel-1")
		require.NoError(t, err)
		after := time.Now()

		stopped, err := manager.GetDeploymentByReleaseID("app1", "rel-1")
		require.NoError(t, err)
		assert.Equal(t, app.DeploymentStatusStopped, stopped.Status)
		require.NotNil(t, stopped.StoppedAt)
		assert.True(t, stopped.StoppedAt.After(before) || stopped.StoppedAt.Equal(before))
		assert.True(t, stopped.StoppedAt.Before(after) || stopped.StoppedAt.Equal(after))
	})

	t.Run("fails for non-existent release ID", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		err = manager.MarkDeploymentStopped("app2", "missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestGetDeploymentByReleaseID tests deployment lookup by ID.
func TestGetDeploymentByReleaseID(t *testing.T) {
	t.Parallel()

	t.Run("returns deployment by release ID", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		dep := app.Deployment{
			ReleaseID:  "target-release",
			Commit:     "abc123",
			Port:       9000,
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app1", dep)
		require.NoError(t, err)

		found, err := manager.GetDeploymentByReleaseID("app1", "target-release")
		require.NoError(t, err)
		assert.Equal(t, "target-release", found.ReleaseID)
		assert.Equal(t, "abc123", found.Commit)
		assert.Equal(t, 9000, found.Port)
	})

	t.Run("fails for non-existent release ID", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		_, err = manager.GetDeploymentByReleaseID("app2", "missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestGetDeploymentsByStatus tests filtering deployments by status.
func TestGetDeploymentsByStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns deployments matching status", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add multiple deployments with different statuses
		dep1 := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{ReleaseID: "rel-2", Status: app.DeploymentStatusFailed, DeployedAt: time.Now(), FailureReason: "timeout"}
		err = manager.AddDeployment("app1", dep2)
		require.NoError(t, err)

		dep3 := app.Deployment{ReleaseID: "rel-3", Status: app.DeploymentStatusFailed, DeployedAt: time.Now(), FailureReason: "crash"}
		err = manager.AddDeployment("app1", dep3)
		require.NoError(t, err)

		failed, err := manager.GetDeploymentsByStatus("app1", app.DeploymentStatusFailed)
		require.NoError(t, err)
		assert.Len(t, failed, 2)
	})

	t.Run("returns empty slice when no matches", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		stopped, err := manager.GetDeploymentsByStatus("app2", app.DeploymentStatusStopped)
		require.NoError(t, err)
		assert.Empty(t, stopped)
	})
}

// TestCleanupOldDeployments tests deployment cleanup logic.
func TestCleanupOldDeployments(t *testing.T) {
	t.Parallel()

	t.Run("removes stopped deployments older than threshold", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add old stopped deployment
		oldDep := app.Deployment{
			ReleaseID:  "old-stopped",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now().Add(-48 * time.Hour),
		}
		err = manager.AddDeployment("app1", oldDep)
		require.NoError(t, err)

		// Manually mark as stopped (simulating old deployment)
		err = manager.MarkDeploymentStopped("app1", "old-stopped")
		require.NoError(t, err)

		// Update deployed_at to be old
		meta, _ := manager.Load("app1")
		for i := range meta.Deployments {
			if meta.Deployments[i].ReleaseID == "old-stopped" {
				meta.Deployments[i].DeployedAt = time.Now().Add(-48 * time.Hour)
			}
		}
		_ = manager.Save(meta)

		// Add recent deployment
		newDep := app.Deployment{
			ReleaseID:  "new-active",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app1", newDep)
		require.NoError(t, err)

		// Cleanup deployments older than 24 hours
		err = manager.CleanupOldDeployments("app1", 24*time.Hour)
		require.NoError(t, err)

		deployments, err := manager.ListDeployments("app1")
		require.NoError(t, err)

		// Old stopped should be removed, new active should remain
		assert.Len(t, deployments, 1)
		assert.Equal(t, "new-active", deployments[0].ReleaseID)
	})

	t.Run("keeps active and standby deployments regardless of age", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		// Add old deployments
		dep1 := app.Deployment{
			ReleaseID:  "old-1",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now().Add(-72 * time.Hour),
		}
		err = manager.AddDeployment("app2", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{
			ReleaseID:  "old-2",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now().Add(-72 * time.Hour),
		}
		err = manager.AddDeployment("app2", dep2)
		require.NoError(t, err)

		// Manually set old deployed_at
		meta, _ := manager.Load("app2")
		for i := range meta.Deployments {
			meta.Deployments[i].DeployedAt = time.Now().Add(-72 * time.Hour)
		}
		_ = manager.Save(meta)

		// Cleanup
		err = manager.CleanupOldDeployments("app2", 24*time.Hour)
		require.NoError(t, err)

		deployments, err := manager.ListDeployments("app2")
		require.NoError(t, err)

		// Both active and standby should remain despite age
		assert.Len(t, deployments, 2)
	})

	t.Run("keeps recent stopped and failed deployments", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app3")
		require.NoError(t, err)

		// Add recent failed deployment
		failedDep := app.Deployment{
			ReleaseID:     "recent-failed",
			Status:        app.DeploymentStatusFailed,
			DeployedAt:    time.Now().Add(-12 * time.Hour),
			FailureReason: "health check failed",
		}
		err = manager.AddDeployment("app3", failedDep)
		require.NoError(t, err)

		// Cleanup deployments older than 24 hours
		err = manager.CleanupOldDeployments("app3", 24*time.Hour)
		require.NoError(t, err)

		deployments, err := manager.ListDeployments("app3")
		require.NoError(t, err)

		// Recent failed should be kept
		assert.Len(t, deployments, 1)
		assert.Equal(t, "recent-failed", deployments[0].ReleaseID)
	})

	t.Run("removes multiple old deployments", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app4")
		require.NoError(t, err)

		// Add current active
		activeDep := app.Deployment{
			ReleaseID:  "current",
			Status:     app.DeploymentStatusActive,
			DeployedAt: time.Now(),
		}
		err = manager.AddDeployment("app4", activeDep)
		require.NoError(t, err)

		// Manually add old stopped deployments
		meta, _ := manager.Load("app4")
		for i := 1; i <= 5; i++ {
			meta.Deployments = append(meta.Deployments, app.Deployment{
				ReleaseID:  "old-" + string(rune('0'+i)),
				Status:     app.DeploymentStatusStopped,
				DeployedAt: time.Now().Add(-48 * time.Hour),
			})
		}
		_ = manager.Save(meta)

		// Cleanup
		err = manager.CleanupOldDeployments("app4", 24*time.Hour)
		require.NoError(t, err)

		deployments, err := manager.ListDeployments("app4")
		require.NoError(t, err)

		// Only current active should remain
		assert.Len(t, deployments, 1)
		assert.Equal(t, "current", deployments[0].ReleaseID)
	})
}

// TestGetStats tests deployment statistics.
func TestGetStats(t *testing.T) {
	t.Parallel()

	t.Run("returns accurate statistics", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app1")
		require.NoError(t, err)

		// Add deployments with various statuses
		dep1 := app.Deployment{ReleaseID: "rel-1", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep1)
		require.NoError(t, err)

		dep2 := app.Deployment{ReleaseID: "rel-2", Status: app.DeploymentStatusActive, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep2)
		require.NoError(t, err)

		dep3 := app.Deployment{ReleaseID: "rel-3", Status: app.DeploymentStatusFailed, DeployedAt: time.Now()}
		err = manager.AddDeployment("app1", dep3)
		require.NoError(t, err)

		// Manually add stopped deployment
		meta, _ := manager.Load("app1")
		meta.Deployments = append(meta.Deployments, app.Deployment{
			ReleaseID:  "stopped-1",
			Status:     app.DeploymentStatusStopped,
			DeployedAt: time.Now(),
		})
		_ = manager.Save(meta)

		total, active, standby, failed, stopped, err := manager.GetStats("app1")
		require.NoError(t, err)

		assert.Equal(t, 4, total)
		assert.Equal(t, 1, active)  // rel-2
		assert.Equal(t, 1, standby) // rel-1
		assert.Equal(t, 1, failed)  // rel-3
		assert.Equal(t, 1, stopped) // stopped-1
	})

	t.Run("returns zeros for app with no deployments", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		err := manager.Create("app2")
		require.NoError(t, err)

		total, active, standby, failed, stopped, err := manager.GetStats("app2")
		require.NoError(t, err)

		assert.Equal(t, 0, total)
		assert.Equal(t, 0, active)
		assert.Equal(t, 0, standby)
		assert.Equal(t, 0, failed)
		assert.Equal(t, 0, stopped)
	})

	t.Run("fails for non-existent app", func(t *testing.T) {
		t.Parallel()

		dataDir := t.TempDir()
		manager := app.NewManager(dataDir)

		_, _, _, _, _, err := manager.GetStats("missing")
		require.Error(t, err)
	})
}

// TestDeploymentStatusConstants verifies status constants are correct.
func TestDeploymentStatusConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "active", app.DeploymentStatusActive)
	assert.Equal(t, "standby", app.DeploymentStatusStandby)
	assert.Equal(t, "failed", app.DeploymentStatusFailed)
	assert.Equal(t, "stopped", app.DeploymentStatusStopped)
}

// TestMaxDeploymentHistory verifies the history limit constant.
func TestMaxDeploymentHistory(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 10, app.MaxDeploymentHistory)
}

// TestErrorConstants verifies error values.
func TestErrorConstants(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, app.ErrNoActiveDeployment, "no active deployment found")
	assert.EqualError(t, app.ErrNoStandbyDeployment, "no standby deployment found")
	assert.EqualError(t, app.ErrInvalidStatus, "invalid deployment status")
}
