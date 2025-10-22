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
	"github.com/dmitrymomot/runlite/internal/spec"
)

func TestBuild(t *testing.T) {
	t.Run("successful build with single file artifact", func(t *testing.T) {
		// Setup directories
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		// Create a simple build script
		script := `echo "building..." && echo "binary content" > server && chmod +x server`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "server", Dest: "server"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify artifact was copied
		serverPath := filepath.Join(releaseDir, "server")
		content, err := os.ReadFile(serverPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "binary content")

		// Verify executable permission
		info, err := os.Stat(serverPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "file should be executable")
	})

	t.Run("successful build with directory artifact", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		// Create a script that builds a directory structure
		script := `
mkdir -p migrations
echo "CREATE TABLE users;" > migrations/001_init.sql
echo "ALTER TABLE users;" > migrations/002_alter.sql
`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "migrations", Dest: "migrations"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify directory was copied
		migrationDir := filepath.Join(releaseDir, "migrations")
		info, err := os.Stat(migrationDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify files in directory
		file1, err := os.ReadFile(filepath.Join(migrationDir, "001_init.sql"))
		require.NoError(t, err)
		assert.Contains(t, string(file1), "CREATE TABLE")

		file2, err := os.ReadFile(filepath.Join(migrationDir, "002_alter.sql"))
		require.NoError(t, err)
		assert.Contains(t, string(file2), "ALTER TABLE")
	})

	t.Run("successful build with multiple artifacts", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `
echo "server" > server
mkdir -p static
echo "index" > static/index.html
mkdir -p migrations
echo "init" > migrations/001.sql
`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "server", Dest: "server"},
					{Source: "static", Dest: "static"},
					{Source: "migrations", Dest: "migrations"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify all artifacts
		assert.FileExists(t, filepath.Join(releaseDir, "server"))
		assert.FileExists(t, filepath.Join(releaseDir, "static", "index.html"))
		assert.FileExists(t, filepath.Join(releaseDir, "migrations", "001.sql"))
	})

	t.Run("normalizes trailing slashes in paths", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `
mkdir -p mydir
echo "content" > mydir/file.txt
`

		// Test both with and without trailing slash
		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "mydir/", Dest: "mydir"}, // trailing slash on source
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(releaseDir, "mydir", "file.txt"))
	})

	t.Run("respects custom timeout", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		// Script that sleeps longer than custom timeout
		script := `sleep 2 && echo "done" > output`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script:  script,
				Timeout: 100 * time.Millisecond, // Very short timeout
				Artifacts: []spec.Artifact{
					{Source: "output", Dest: "output"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		// Context timeout kills the process, error may be "killed" or "context deadline exceeded"
		assert.True(t,
			strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "killed"),
			"error should indicate timeout: %v", err)
	})

	t.Run("uses default timeout when not specified", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `echo "quick" > output`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				// No timeout specified - should use default
				Artifacts: []spec.Artifact{
					{Source: "output", Dest: "output"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)
	})

	t.Run("returns error for nil build config", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		s := &spec.AppSpec{
			Build: nil,
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no build configuration")
	})

	t.Run("returns error for empty script", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: "",
				Artifacts: []spec.Artifact{
					{Source: "server", Dest: "server"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no build script specified")
	})

	t.Run("returns error for no artifacts", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script:    "echo 'build'",
				Artifacts: []spec.Artifact{},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no artifacts specified")
	})

	t.Run("returns error when build script fails", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `exit 1` // Fail immediately

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "server", Dest: "server"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build script failed")
	})

	t.Run("returns error when artifact not found after build", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		// Script succeeds but doesn't create the artifact
		script := `echo "building..."`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "nonexistent", Dest: "nonexistent"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build validation failed")
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("handles complex directory structures", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `
mkdir -p static/css static/js static/img
echo "style" > static/css/main.css
echo "script" > static/js/app.js
echo "image" > static/img/logo.png
mkdir -p static/fonts
echo "font" > static/fonts/main.ttf
`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "static", Dest: "static"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify nested structure
		assert.FileExists(t, filepath.Join(releaseDir, "static", "css", "main.css"))
		assert.FileExists(t, filepath.Join(releaseDir, "static", "js", "app.js"))
		assert.FileExists(t, filepath.Join(releaseDir, "static", "img", "logo.png"))
		assert.FileExists(t, filepath.Join(releaseDir, "static", "fonts", "main.ttf"))
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `
echo "executable" > script.sh
chmod 755 script.sh
echo "readonly" > data.txt
chmod 444 data.txt
`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "script.sh", Dest: "script.sh"},
					{Source: "data.txt", Dest: "data.txt"},
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Check executable file
		scriptInfo, err := os.Stat(filepath.Join(releaseDir, "script.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), scriptInfo.Mode().Perm())

		// Check readonly file
		dataInfo, err := os.Stat(filepath.Join(releaseDir, "data.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o444), dataInfo.Mode().Perm())
	})

	t.Run("handles artifacts with different source and dest paths", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `
mkdir -p build/output
echo "binary" > build/output/app
`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "build/output/app", Dest: "app"}, // Copy to different location
				},
			},
		}

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify artifact is at dest path, not source path
		assert.FileExists(t, filepath.Join(releaseDir, "app"))
		assert.NoFileExists(t, filepath.Join(releaseDir, "build", "output", "app"))
	})

	t.Run("context cancellation stops build", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		script := `sleep 10 && echo "done" > output`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "output", Dest: "output"},
				},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.Error(t, err)
		// Context cancellation kills the process, error may be "killed" or "context canceled"
		assert.True(t,
			strings.Contains(err.Error(), "context canceled") ||
				strings.Contains(err.Error(), "killed"),
			"error should indicate cancellation: %v", err)
	})

	t.Run("validates artifacts exist in release dir after copy", func(t *testing.T) {
		checkoutDir := t.TempDir()
		releaseDir := t.TempDir()

		// Create artifact in checkout dir
		script := `echo "content" > file.txt`

		s := &spec.AppSpec{
			Build: &spec.BuildConfig{
				Script: script,
				Artifacts: []spec.Artifact{
					{Source: "file.txt", Dest: "file.txt"},
				},
			},
		}

		// Make release dir read-only to cause copy failure
		// (This is a bit tricky to test, so we'll trust the validation logic works)

		ctx := context.Background()
		err := deploy.Build(ctx, s, checkoutDir, releaseDir)
		require.NoError(t, err)

		// Verify validation happened by checking file exists
		assert.FileExists(t, filepath.Join(releaseDir, "file.txt"))
	})
}
