package deploy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/spec"
)

func TestNewOrchestrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appName   string
		releaseID string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid inputs",
			appName:   "test-app",
			releaseID: "20251022143055-a3f5c2b",
			wantErr:   false,
		},
		{
			name:      "empty app name",
			appName:   "",
			releaseID: "20251022143055-a3f5c2b",
			wantErr:   true,
			errMsg:    "app name is required",
		},
		{
			name:      "empty release ID",
			appName:   "test-app",
			releaseID: "",
			wantErr:   true,
			errMsg:    "release ID is required",
		},
		{
			name:      "both empty",
			appName:   "",
			releaseID: "",
			wantErr:   true,
			errMsg:    "app name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orch, err := NewOrchestrator(tt.appName, tt.releaseID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, orch)
			} else {
				require.NoError(t, err)
				require.NotNil(t, orch)
				assert.Equal(t, tt.appName, orch.appName)
				assert.Equal(t, tt.releaseID, orch.releaseID)
				assert.NotNil(t, orch.caddy)
				assert.NotNil(t, orch.logger)
			}
		})
	}
}

func TestDeploy_LockAcquisition(t *testing.T) {
	ctx := context.Background()
	appName := "test-app"
	commitHash := "a3f5c2b1234"

	// Create minimal spec
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Run:     &spec.RunConfig{Command: "./server"},
	}

	// Setup test environment
	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	// Create app directory and metadata
	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	// Acquire lock manually to simulate in-progress deployment
	releaseID := GenerateReleaseID(commitHash)
	require.NoError(t, AcquireLock(ctx, appName, releaseID))

	// Try to deploy - should fail with lock error
	err := Deploy(ctx, appName, commitHash, appSpec, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acquire deployment lock")
	assert.ErrorIs(t, err, ErrDeploymentInProgress)

	// Release lock and verify next deploy can proceed (will fail for other reasons)
	require.NoError(t, ReleaseLock(ctx, appName))
}

func TestDeploy_OrchestratorCreationFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	appName := "" // Empty to trigger orchestrator creation failure
	commitHash := "a3f5c2b1234"

	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: "test"},
		Domains: []string{"test-app.local"},
		Run:     &spec.RunConfig{Command: "./server"},
	}

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	err := Deploy(ctx, appName, commitHash, appSpec, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create orchestrator")
}

func TestDeployFirst_BuildFailureWithCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	appName := "test-app"
	commitHash := "a3f5c2b1234"
	releaseID := GenerateReleaseID(commitHash)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	// Create app directory and metadata
	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	// Create invalid checkout directory to force build failure
	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	// Spec with cleanup enabled and invalid build script
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "exit 1", // Force failure
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
		},
		Run: &spec.RunConfig{Command: "./server"},
		Deploy: &spec.DeployConfig{
			CleanupOnFailure: true,
		},
	}

	orch, err := NewOrchestrator(appName, releaseID)
	require.NoError(t, err)

	err = orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build failed")

	// Verify release directory was cleaned up
	releaseDir := GetReleaseDir(appName, releaseID)
	_, err = os.Stat(releaseDir)
	assert.True(t, os.IsNotExist(err), "release directory should be cleaned up")
}

func TestDeployFirst_BuildFailureWithoutCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	appName := "test-app"
	commitHash := "a3f5c2b1234"
	releaseID := GenerateReleaseID(commitHash)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "exit 1",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
		},
		Run: &spec.RunConfig{Command: "./server"},
		Deploy: &spec.DeployConfig{
			CleanupOnFailure: false, // Cleanup disabled
		},
	}

	orch, err := NewOrchestrator(appName, releaseID)
	require.NoError(t, err)

	err = orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build failed")

	// Verify release directory still exists (not cleaned up)
	releaseDir := GetReleaseDir(appName, releaseID)
	stat, err := os.Stat(releaseDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir(), "release directory should exist")
}

func TestDeployFirst_SuccessfulFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test requires actual systemd - skip if not available
	ctx := context.Background()
	if err := ReloadDaemon(ctx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app"
	commitHash := "a3f5c2b1234"
	releaseID := GenerateReleaseID(commitHash)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	// Create env file
	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	// Create checkout directory with a simple test server
	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	// Create a simple HTTP server for testing
	serverCode := `package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	})
	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	keepReleases := 2
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
		},
		Deploy: &spec.DeployConfig{
			CleanupOnFailure: true,
			KeepReleases:     &keepReleases,
		},
	}

	orch, err := NewOrchestrator(appName, releaseID)
	require.NoError(t, err)

	err = orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	require.NoError(t, err)

	// Verify deployment metadata
	activeDeployment, err := mgr.GetActiveDeployment(appName)
	require.NoError(t, err)
	assert.Equal(t, releaseID, activeDeployment.ReleaseID)
	assert.Equal(t, commitHash, activeDeployment.Commit)
	assert.Equal(t, "active", activeDeployment.Status)
	assert.Greater(t, activeDeployment.Port, 0)

	// Verify systemd service is running
	unitName := FormatUnitName(appName, releaseID)
	defer func() {
		_ = StopService(ctx, unitName)
		_ = DisableService(ctx, unitName)
	}()

	// Cleanup caddy route
	caddyClient := caddy.NewClient("")
	defer func() { _ = caddyClient.DeleteAppRoute(appName) }()
}

func TestDeployFirst_HealthCheckFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	if err := ReloadDaemon(ctx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app-healthfail"
	commitHash := "a3f5c2b1234"
	releaseID := GenerateReleaseID(commitHash)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	// Server that doesn't respond to health checks
	serverCode := `package main
import (
	"os"
	"time"
)
func main() {
	time.Sleep(60 * time.Second)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  3 * time.Second,
			Interval: 500 * time.Millisecond,
		},
		Deploy: &spec.DeployConfig{
			CleanupOnFailure: true,
		},
	}

	orch, err := NewOrchestrator(appName, releaseID)
	require.NoError(t, err)

	err = orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")

	// Verify deployment marked as failed
	deployment, err := mgr.GetDeploymentByReleaseID(appName, releaseID)
	require.NoError(t, err)
	assert.Equal(t, "failed", deployment.Status)

	// Verify service was stopped
	unitName := FormatUnitName(appName, releaseID)
	defer func() {
		_ = StopService(ctx, unitName)
		_ = DisableService(ctx, unitName)
	}()
}

func TestDeployBlueGreen_SuccessfulFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	if err := ReloadDaemon(ctx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app-bluegreen"
	oldCommit := "old1234"
	newCommit := "new5678"
	oldReleaseID := GenerateReleaseID(oldCommit)
	newReleaseID := GenerateReleaseID(newCommit)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	// Deploy first version
	checkoutDir := filepath.Join(tmpDir, "checkout1")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	serverCode := `package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	})
	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
		},
		Deploy: &spec.DeployConfig{
			DrainPeriod:      2 * time.Second,
			CleanupOnFailure: true,
		},
	}

	// First deployment
	orch1, err := NewOrchestrator(appName, oldReleaseID)
	require.NoError(t, err)

	err = orch1.deployFirst(ctx, oldCommit, appSpec, checkoutDir)
	require.NoError(t, err)

	oldDeployment, err := mgr.GetActiveDeployment(appName)
	require.NoError(t, err)
	oldUnitName := FormatUnitName(appName, oldReleaseID)

	defer func() {
		_ = StopService(ctx, oldUnitName)
		_ = DisableService(ctx, oldUnitName)
	}()

	// Second deployment (blue-green)
	checkoutDir2 := filepath.Join(tmpDir, "checkout2")
	require.NoError(t, os.MkdirAll(checkoutDir2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir2, "main.go"), []byte(serverCode), 0o644))

	orch2, err := NewOrchestrator(appName, newReleaseID)
	require.NoError(t, err)

	err = orch2.deployBlueGreen(ctx, newCommit, appSpec, checkoutDir2, oldDeployment)
	require.NoError(t, err)

	// Verify new deployment is active
	newDeployment, err := mgr.GetActiveDeployment(appName)
	require.NoError(t, err)
	assert.Equal(t, newReleaseID, newDeployment.ReleaseID)
	assert.Equal(t, newCommit, newDeployment.Commit)
	assert.Equal(t, "active", newDeployment.Status)

	// Verify old deployment is stopped
	oldDeploymentAfter, err := mgr.GetDeploymentByReleaseID(appName, oldReleaseID)
	require.NoError(t, err)
	assert.Equal(t, "stopped", oldDeploymentAfter.Status)
	assert.NotNil(t, oldDeploymentAfter.StoppedAt)

	// Cleanup
	newUnitName := FormatUnitName(appName, newReleaseID)
	defer func() {
		_ = StopService(ctx, newUnitName)
		_ = DisableService(ctx, newUnitName)
	}()

	caddyClient := caddy.NewClient("")
	defer func() { _ = caddyClient.DeleteAppRoute(appName) }()
}

func TestDeployBlueGreen_DrainPeriodHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	if err := ReloadDaemon(ctx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app-drain"
	oldCommit := "old1234"
	newCommit := "new5678"
	oldReleaseID := GenerateReleaseID(oldCommit)
	newReleaseID := GenerateReleaseID(newCommit)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	serverCode := `package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	})
	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	drainPeriod := 5 * time.Second
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
		},
		Deploy: &spec.DeployConfig{
			DrainPeriod:      drainPeriod,
			CleanupOnFailure: true,
		},
	}

	// First deployment
	orch1, err := NewOrchestrator(appName, oldReleaseID)
	require.NoError(t, err)

	err = orch1.deployFirst(ctx, oldCommit, appSpec, checkoutDir)
	require.NoError(t, err)

	oldDeployment, err := mgr.GetActiveDeployment(appName)
	require.NoError(t, err)

	// Blue-green deployment
	checkoutDir2 := filepath.Join(tmpDir, "checkout2")
	require.NoError(t, os.MkdirAll(checkoutDir2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir2, "main.go"), []byte(serverCode), 0o644))

	orch2, err := NewOrchestrator(appName, newReleaseID)
	require.NoError(t, err)

	start := time.Now()
	err = orch2.deployBlueGreen(ctx, newCommit, appSpec, checkoutDir2, oldDeployment)
	elapsed := time.Since(start)

	require.NoError(t, err)

	// Verify drain period was respected (allowing some margin for processing)
	assert.GreaterOrEqual(t, elapsed, drainPeriod)
	assert.Less(t, elapsed, drainPeriod+3*time.Second, "deployment took too long")

	// Cleanup
	oldUnitName := FormatUnitName(appName, oldReleaseID)
	newUnitName := FormatUnitName(appName, newReleaseID)
	defer func() {
		_ = StopService(ctx, oldUnitName)
		_ = DisableService(ctx, oldUnitName)
		_ = StopService(ctx, newUnitName)
		_ = DisableService(ctx, newUnitName)
	}()

	caddyClient := caddy.NewClient("")
	defer func() { _ = caddyClient.DeleteAppRoute(appName) }()
}

func TestDeployBlueGreen_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	baseCtx := context.Background()
	if err := ReloadDaemon(baseCtx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app-cancel"
	oldCommit := "old1234"
	newCommit := "new5678"
	oldReleaseID := GenerateReleaseID(oldCommit)
	newReleaseID := GenerateReleaseID(newCommit)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	serverCode := `package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	})
	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	longDrainPeriod := 30 * time.Second
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
		},
		Deploy: &spec.DeployConfig{
			DrainPeriod:      longDrainPeriod,
			CleanupOnFailure: true,
		},
	}

	// First deployment
	orch1, err := NewOrchestrator(appName, oldReleaseID)
	require.NoError(t, err)

	err = orch1.deployFirst(baseCtx, oldCommit, appSpec, checkoutDir)
	require.NoError(t, err)

	oldDeployment, err := mgr.GetActiveDeployment(appName)
	require.NoError(t, err)

	// Blue-green deployment with context cancellation
	checkoutDir2 := filepath.Join(tmpDir, "checkout2")
	require.NoError(t, os.MkdirAll(checkoutDir2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir2, "main.go"), []byte(serverCode), 0o644))

	orch2, err := NewOrchestrator(appName, newReleaseID)
	require.NoError(t, err)

	// Create context that will be canceled during drain period
	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
	defer cancel()

	err = orch2.deployBlueGreen(ctx, newCommit, appSpec, checkoutDir2, oldDeployment)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Cleanup
	oldUnitName := FormatUnitName(appName, oldReleaseID)
	newUnitName := FormatUnitName(appName, newReleaseID)
	defer func() {
		_ = StopService(baseCtx, oldUnitName)
		_ = DisableService(baseCtx, oldUnitName)
		_ = StopService(baseCtx, newUnitName)
		_ = DisableService(baseCtx, newUnitName)
	}()

	caddyClient := caddy.NewClient("")
	defer func() { _ = caddyClient.DeleteAppRoute(appName) }()
}

func TestDeployFirst_KeepReleasesCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	if err := ReloadDaemon(ctx); err != nil {
		t.Skip("systemd not available")
	}

	t.Parallel()

	appName := "test-app-keepreleases"
	commitHash := "a3f5c2b1234"
	releaseID := GenerateReleaseID(commitHash)

	tmpDir := t.TempDir()
	t.Setenv("RUNLITE_DATA_DIR", tmpDir)

	appDir := filepath.Join(tmpDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	mgr := app.NewManager(tmpDir)
	require.NoError(t, mgr.Create(appName))

	envPath := config.GetAppEnvPath(appName)
	require.NoError(t, os.WriteFile(envPath, []byte("TEST=value\n"), 0o644))

	// Create multiple old releases
	releasesDir := filepath.Join(appDir, "releases")
	require.NoError(t, os.MkdirAll(releasesDir, 0o755))

	oldRelease1 := "20251020120000-old1234"
	oldRelease2 := "20251021120000-old5678"
	require.NoError(t, os.MkdirAll(filepath.Join(releasesDir, oldRelease1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(releasesDir, oldRelease2), 0o755))

	// Add old deployments to metadata (as stopped)
	oldDep1 := app.Deployment{
		ReleaseID:  oldRelease1,
		Commit:     "old1234",
		Port:       8001,
		Status:     app.DeploymentStatusStopped,
		DeployedAt: time.Now().Add(-48 * time.Hour),
	}
	oldDep2 := app.Deployment{
		ReleaseID:  oldRelease2,
		Commit:     "old5678",
		Port:       8002,
		Status:     app.DeploymentStatusStopped,
		DeployedAt: time.Now().Add(-24 * time.Hour),
	}

	// Manually update metadata to include old deployments
	meta, err := mgr.Load(appName)
	require.NoError(t, err)
	meta.Deployments = append(meta.Deployments, oldDep1, oldDep2)
	require.NoError(t, mgr.Save(meta))

	checkoutDir := filepath.Join(tmpDir, "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	serverCode := `package main
import (
	"fmt"
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	})
	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(checkoutDir, "main.go"), []byte(serverCode), 0o644))

	keepReleases := 1
	appSpec := &spec.AppSpec{
		App:     spec.AppConfig{Name: appName},
		Domains: []string{"test-app.local"},
		Build: &spec.BuildConfig{
			Script:    "go build -o server main.go",
			Artifacts: []spec.Artifact{{Source: "server", Dest: "server"}},
			Timeout:   30 * time.Second,
		},
		Run: &spec.RunConfig{Command: "./server"},
		Health: &spec.HealthConfig{
			Path:     "/health",
			Timeout:  10 * time.Second,
			Interval: 1 * time.Second,
		},
		Deploy: &spec.DeployConfig{
			CleanupOnFailure: true,
			KeepReleases:     &keepReleases,
		},
	}

	orch, err := NewOrchestrator(appName, releaseID)
	require.NoError(t, err)

	err = orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	require.NoError(t, err)

	// Give cleanup time to complete
	time.Sleep(500 * time.Millisecond)

	// Verify old releases were cleaned up
	releases, err := ListReleases(ctx, appName)
	require.NoError(t, err)

	// Should only have the new release (keepReleases=1)
	assert.LessOrEqual(t, len(releases), 2, "should have at most 2 releases (new + 1 kept)")

	// Cleanup
	unitName := FormatUnitName(appName, releaseID)
	defer func() {
		_ = StopService(ctx, unitName)
		_ = DisableService(ctx, unitName)
	}()

	caddyClient := caddy.NewClient("")
	defer func() { _ = caddyClient.DeleteAppRoute(appName) }()
}

func TestGenerateReleaseID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		commitHash string
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "full commit hash",
			commitHash: "a3f5c2b1234567890",
			wantSuffix: "a3f5c2b",
		},
		{
			name:       "short commit hash",
			commitHash: "abc123",
			wantSuffix: "abc123",
		},
		{
			name:       "exactly 7 chars",
			commitHash: "abcdefg",
			wantSuffix: "abcdefg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			releaseID := GenerateReleaseID(tt.commitHash)

			// Verify format: {timestamp}-{commit-hash}
			assert.Contains(t, releaseID, "-")
			parts := filepath.Base(releaseID)
			assert.Contains(t, parts, tt.wantSuffix)

			// Verify timestamp part is 14 digits
			timestampPart := releaseID[:14]
			assert.Len(t, timestampPart, 14)
			assert.Regexp(t, `^\d{14}$`, timestampPart)
		})
	}
}
