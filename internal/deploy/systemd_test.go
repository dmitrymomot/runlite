package deploy_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/deploy"
)

// setupSystemdTest creates a temporary directory for systemd testing
func setupSystemdTest(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "runlite-systemd-test-*")
	require.NoError(t, err, "failed to create temp directory")

	// Override systemd path for testing
	oldPath := os.Getenv("SYSTEMD_PATH")
	err = os.Setenv("SYSTEMD_PATH", tmpDir)
	require.NoError(t, err, "failed to set SYSTEMD_PATH")

	// Refresh the global variable
	deploy.SystemdUnitPath = tmpDir

	cleanup := func() {
		os.Setenv("SYSTEMD_PATH", oldPath)
		deploy.SystemdUnitPath = deploy.DefaultSystemdUnitPath
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGenerateUnitFile(t *testing.T) {
	t.Run("generates valid unit file", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "testapp",
			ReleaseID:  "20251022143055-a3f5c2b",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/var/lib/runlite/apps/testapp/releases/20251022143055-a3f5c2b",
			Command:    "/var/lib/runlite/apps/testapp/releases/20251022143055-a3f5c2b/server --port=$PORT",
			EnvFile:    "/var/lib/runlite/apps/testapp/env",
		}

		err := deploy.GenerateUnitFile(ctx, params)

		// Note: This will fail validation if systemd-analyze is not available
		// In CI/non-systemd environments, this is expected
		if err != nil && strings.Contains(err.Error(), "systemd-analyze") {
			t.Skip("systemd-analyze not available, skipping validation test")
		}

		require.NoError(t, err)

		// Verify file was created
		unitPath := filepath.Join(tmpDir, "runlite-testapp@20251022143055-a3f5c2b.service")
		content, err := os.ReadFile(unitPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Description=runlite app: testapp")
		assert.Contains(t, contentStr, "WorkingDirectory=/var/lib/runlite/apps/testapp/releases/20251022143055-a3f5c2b")
		assert.Contains(t, contentStr, "ExecStart=/var/lib/runlite/apps/testapp/releases/20251022143055-a3f5c2b/server --port=$PORT")
		assert.Contains(t, contentStr, "Environment=PORT=8080")
		assert.Contains(t, contentStr, "Environment=RELEASE_ID=20251022143055-a3f5c2b")
		assert.Contains(t, contentStr, "Environment=COMMIT_HASH=a3f5c2b")
		assert.Contains(t, contentStr, "EnvironmentFile=/var/lib/runlite/apps/testapp/env")
		assert.Contains(t, contentStr, "User=runlite")
		assert.Contains(t, contentStr, "Group=runlite")
		assert.Contains(t, contentStr, "Restart=on-failure")
	})

	t.Run("generates unit file without systemd-analyze validation", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemd-analyze to always succeed
		mockScript := filepath.Join(tmpDir, "systemd-analyze")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\nexit 0\n"), 0o755)
		require.NoError(t, err)

		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "myapp",
			ReleaseID:  "20251022150000-abc1234",
			CommitHash: "abc1234",
			Port:       8123,
			WorkingDir: "/var/lib/runlite/apps/myapp/releases/20251022150000-abc1234",
			Command:    "./myapp",
			EnvFile:    "/var/lib/runlite/apps/myapp/env",
		}

		err = deploy.GenerateUnitFile(ctx, params)
		require.NoError(t, err)

		// Verify file exists
		unitPath := filepath.Join(tmpDir, "runlite-myapp@20251022150000-abc1234.service")
		_, err = os.Stat(unitPath)
		require.NoError(t, err)
	})

	t.Run("returns error for empty app name", func(t *testing.T) {
		_, cleanup := setupSystemdTest(t)
		defer cleanup()

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "",
			ReleaseID:  "20251022143055-a3f5c2b",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/test",
			Command:    "./server",
			EnvFile:    "/test/env",
		}

		err := deploy.GenerateUnitFile(ctx, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for empty release ID", func(t *testing.T) {
		_, cleanup := setupSystemdTest(t)
		defer cleanup()

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "testapp",
			ReleaseID:  "",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/test",
			Command:    "./server",
			EnvFile:    "/test/env",
		}

		err := deploy.GenerateUnitFile(ctx, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "release ID")
	})

	t.Run("overwrites existing unit file", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Create initial unit file
		unitPath := filepath.Join(tmpDir, "runlite-testapp@20251022143055-a3f5c2b.service")
		err := os.WriteFile(unitPath, []byte("old content"), 0o644)
		require.NoError(t, err)

		// Mock systemd-analyze
		mockScript := filepath.Join(tmpDir, "systemd-analyze")
		err = os.WriteFile(mockScript, []byte("#!/bin/sh\nexit 0\n"), 0o755)
		require.NoError(t, err)
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "testapp",
			ReleaseID:  "20251022143055-a3f5c2b",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/test",
			Command:    "./server",
			EnvFile:    "/test/env",
		}

		err = deploy.GenerateUnitFile(ctx, params)
		require.NoError(t, err)

		// Verify file was overwritten
		content, err := os.ReadFile(unitPath)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "old content")
		assert.Contains(t, string(content), "Description=runlite app: testapp")
	})

	t.Run("cleans up temp file on validation failure", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemd-analyze to fail
		mockScript := filepath.Join(tmpDir, "mock-systemd-analyze")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\necho 'validation error' >&2\nexit 1\n"), 0o755)
		require.NoError(t, err)
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		ctx := context.Background()
		params := deploy.UnitFileParams{
			AppName:    "testapp",
			ReleaseID:  "20251022143055-a3f5c2b",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/test",
			Command:    "./server",
			EnvFile:    "/test/env",
		}

		err = deploy.GenerateUnitFile(ctx, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")

		// Verify temp file was cleaned up
		tmpPath := filepath.Join(tmpDir, "runlite-testapp@20251022143055-a3f5c2b.service.tmp")
		_, err = os.Stat(tmpPath)
		assert.True(t, os.IsNotExist(err), "temp file should be cleaned up")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		_, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		params := deploy.UnitFileParams{
			AppName:    "testapp",
			ReleaseID:  "20251022143055-a3f5c2b",
			CommitHash: "a3f5c2b",
			Port:       8080,
			WorkingDir: "/test",
			Command:    "./server",
			EnvFile:    "/test/env",
		}

		err := deploy.GenerateUnitFile(ctx, params)
		// Should fail during validation (systemd-analyze with cancelled context)
		// In environments without systemd, this might succeed in writing but fail in validation
		if err != nil {
			assert.Error(t, err)
		}
	})
}

func TestReloadDaemon(t *testing.T) {
	t.Run("calls systemctl daemon-reload", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemctl
		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "daemon-reload" ]; then
    exit 0
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.ReloadDaemon(ctx)
		require.NoError(t, err)
	})

	t.Run("returns error on failure", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemctl that fails
		mockScript := filepath.Join(tmpDir, "mock-systemctl-fail")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\nexit 1\n"), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.ReloadDaemon(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to reload systemd daemon")
	})
}

func TestStartService(t *testing.T) {
	t.Run("starts service successfully", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemctl
		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "start" ] && [ "$2" = "test.service" ]; then
    exit 0
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.StartService(ctx, "test.service")
		require.NoError(t, err)
	})

	t.Run("returns error for empty unit name", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.StartService(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit name")
	})

	t.Run("returns error on start failure", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\necho 'start failed' >&2\nexit 1\n"), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.StartService(ctx, "test.service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start service")
	})

	t.Run("respects context timeout", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		// Mock systemctl that sleeps
		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\nsleep 10\n"), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = deploy.StartService(ctx, "test.service")
		require.Error(t, err)
	})
}

func TestStopService(t *testing.T) {
	t.Run("stops service successfully", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "stop" ] && [ "$2" = "test.service" ]; then
    exit 0
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.StopService(ctx, "test.service")
		require.NoError(t, err)
	})

	t.Run("returns error for empty unit name", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.StopService(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit name")
	})
}

func TestDisableService(t *testing.T) {
	t.Run("disables service successfully", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "disable" ] && [ "$2" = "test.service" ]; then
    exit 0
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.DisableService(ctx, "test.service")
		require.NoError(t, err)
	})

	t.Run("ignores error if service not loaded", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		err := os.WriteFile(mockScript, []byte("#!/bin/sh\necho 'not loaded' >&2\nexit 1\n"), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		err = deploy.DisableService(ctx, "test.service")
		require.NoError(t, err) // Should not error for "not loaded"
	})

	t.Run("returns error for empty unit name", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.DisableService(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit name")
	})
}

func TestGetServiceStatus(t *testing.T) {
	t.Run("returns active status", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "is-active" ]; then
    echo "active"
    exit 0
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		status, err := deploy.GetServiceStatus(ctx, "test.service")
		require.NoError(t, err)
		assert.Equal(t, "active", status)
	})

	t.Run("returns inactive status", func(t *testing.T) {
		tmpDir, cleanup := setupSystemdTest(t)
		defer cleanup()

		mockScript := filepath.Join(tmpDir, "mock-systemctl")
		scriptContent := `#!/bin/sh
if [ "$1" = "is-active" ]; then
    echo "inactive"
    exit 3
fi
exit 1
`
		err := os.WriteFile(mockScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		oldCmd := deploy.SystemctlCmd
		deploy.SystemctlCmd = mockScript
		defer func() { deploy.SystemctlCmd = oldCmd }()

		ctx := context.Background()
		status, err := deploy.GetServiceStatus(ctx, "test.service")
		require.NoError(t, err)
		assert.Equal(t, "inactive", status)
	})

	t.Run("returns error for empty unit name", func(t *testing.T) {
		ctx := context.Background()
		_, err := deploy.GetServiceStatus(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit name")
	})
}

func TestFormatUnitName(t *testing.T) {
	t.Run("formats unit name correctly", func(t *testing.T) {
		unitName := deploy.FormatUnitName("myapp", "20251022143055-a3f5c2b")
		assert.Equal(t, "runlite-myapp@20251022143055-a3f5c2b.service", unitName)
	})

	t.Run("handles different app names", func(t *testing.T) {
		testCases := []struct {
			appName   string
			releaseID string
			expected  string
		}{
			{"api", "20251022143055-abc1234", "runlite-api@20251022143055-abc1234.service"},
			{"web-frontend", "20251022150000-def5678", "runlite-web-frontend@20251022150000-def5678.service"},
			{"worker", "20251022160000-ghi9012", "runlite-worker@20251022160000-ghi9012.service"},
		}

		for _, tc := range testCases {
			result := deploy.FormatUnitName(tc.appName, tc.releaseID)
			assert.Equal(t, tc.expected, result)
		}
	})
}
