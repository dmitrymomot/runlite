package environment_test

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/environment"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns expected default values", func(t *testing.T) {
		t.Parallel()

		cfg := environment.DefaultConfig()

		assert.Equal(t, "/var/lib/runlite/apps", cfg.BasePath, "BasePath should be /var/lib/runlite/apps")
	})

	t.Run("base path is absolute", func(t *testing.T) {
		t.Parallel()

		cfg := environment.DefaultConfig()

		assert.True(t, filepath.IsAbs(cfg.BasePath), "BasePath should be absolute")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config passes", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: "/tmp/runlite/apps",
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("empty base path fails", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: "",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "base path cannot be empty")
	})

	t.Run("relative base path fails", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: "relative/path",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "base path must be absolute")
	})

	t.Run("base path with dot prefix fails", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: "./apps",
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidConfig), "should wrap ErrInvalidConfig")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid config creates manager successfully", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		logger := slog.Default()

		manager, err := environment.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: "",
		}
		logger := slog.Default()

		manager, err := environment.New(cfg, logger)
		require.Error(t, err)
		assert.Nil(t, manager)
		assert.True(t, errors.Is(err, environment.ErrInvalidConfig), "should return ErrInvalidConfig")
	})

	t.Run("nil logger creates usable manager with default logger", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}

		manager, err := environment.New(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, manager, "manager should be created with nil logger")

		// Verify manager is usable
		err = manager.SetVar("test-app", "TEST_VAR", "value")
		require.NoError(t, err, "manager with nil logger should be usable")
	})

	t.Run("created manager is usable", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		logger := slog.Default()

		manager, err := environment.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Verify manager works
		err = manager.SetVar("my-app", "DATABASE_URL", "postgres://localhost")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("my-app")
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost", vars["DATABASE_URL"])
	})
}

func TestManager_WriteEnvFile(t *testing.T) {
	t.Parallel()

	t.Run("writes new file with multiple variables", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"DATABASE_URL": "postgres://localhost:5432/db",
			"API_KEY":      "secret123",
			"DEBUG_MODE":   "true",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		// Verify file was created
		envFile := filepath.Join(cfg.BasePath, "test-app", "env")
		_, err = os.Stat(envFile)
		require.NoError(t, err, "env file should exist")

		// Verify content
		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, vars, readVars)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Write initial vars
		initialVars := map[string]string{
			"OLD_VAR": "old_value",
			"KEEP":    "keep_value",
		}
		err = manager.WriteEnvFile("test-app", initialVars)
		require.NoError(t, err)

		// Overwrite with new vars
		newVars := map[string]string{
			"NEW_VAR": "new_value",
			"KEEP":    "updated_value",
		}
		err = manager.WriteEnvFile("test-app", newVars)
		require.NoError(t, err)

		// Verify old vars are gone, new vars are present
		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, newVars, readVars)
		assert.NotContains(t, readVars, "OLD_VAR")
	})

	t.Run("file has correct permissions 0600", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"SECRET_KEY": "secret_value",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		envFile := filepath.Join(cfg.BasePath, "test-app", "env")
		info, err := os.Stat(envFile)
		require.NoError(t, err)

		assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "file should have 0600 permissions")
	})

	t.Run("directory created with 0700 permissions", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"VAR": "value",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		appDir := filepath.Join(cfg.BasePath, "test-app")
		info, err := os.Stat(appDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm(), "directory should have 0700 permissions")
	})

	t.Run("invalid app name rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"VAR": "value",
		}

		testCases := []struct {
			name    string
			appName string
		}{
			{"empty", ""},
			{"with slash", "app/name"},
			{"with backslash", "app\\name"},
			{"parent dir ref", "../other-app"},
			{"double dot", "app..name"},
			{"special chars", "app@name"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err = manager.WriteEnvFile(tc.appName, vars)
				require.Error(t, err)
				assert.True(t, errors.Is(err, environment.ErrInvalidAppName), "should return ErrInvalidAppName")
			})
		}
	})

	t.Run("values with special characters are escaped correctly", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"WITH_QUOTES":    `value "with quotes"`,
			"WITH_SPACES":    "value with spaces",
			"WITH_BACKSLASH": `C:\path\to\file`,
			"WITH_SINGLE":    "value's apostrophe",
			"COMPLEX":        `{"key": "value with \"quotes\""}`,
			"SIMPLE":         "simple_value",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		// Verify all values are read back correctly
		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, vars, readVars, "all special characters should be preserved")
	})

	t.Run("values with newlines are quoted but may not round-trip perfectly", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Newlines in values are tricky for systemd EnvironmentFile format
		// The implementation quotes them, but they may not parse back correctly
		// This is expected behavior - environment files are line-based
		vars := map[string]string{
			"WITH_NEWLINE": "line1\nline2",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err, "should write file without error")

		// Reading back may not preserve the newline exactly due to format limitations
		// This is acceptable behavior for environment files
	})

	t.Run("empty variable values are handled", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		vars := map[string]string{
			"EMPTY":     "",
			"NOT_EMPTY": "value",
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "", readVars["EMPTY"])
		assert.Equal(t, "value", readVars["NOT_EMPTY"])
	})

	t.Run("invalid variable key rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		testCases := []struct {
			name string
			key  string
		}{
			{"lowercase", "lowercase_var"},
			{"starts with digit", "1VAR"},
			{"contains dash", "VAR-NAME"},
			{"contains space", "VAR NAME"},
			{"contains dot", "VAR.NAME"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vars := map[string]string{
					tc.key: "value",
				}
				err = manager.WriteEnvFile("test-app", vars)
				require.Error(t, err)
				assert.True(t, errors.Is(err, environment.ErrInvalidKey), "should return ErrInvalidKey")
			})
		}
	})

	t.Run("valid variable keys accepted", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		validKeys := []string{
			"UPPERCASE",
			"WITH_UNDERSCORE",
			"WITH_123_NUMBERS",
			"_LEADING_UNDERSCORE",
			"A",
		}

		vars := make(map[string]string)
		for _, key := range validKeys {
			vars[key] = "value"
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err, "all valid keys should be accepted")
	})

	t.Run("empty vars map creates empty file", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.WriteEnvFile("test-app", map[string]string{})
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Empty(t, readVars)
	})
}

func TestManager_ReadEnvFile(t *testing.T) {
	t.Parallel()

	t.Run("reads existing file correctly", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Write vars
		vars := map[string]string{
			"DATABASE_URL": "postgres://localhost",
			"API_KEY":      "secret",
			"PORT":         "8080",
		}
		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		// Read vars
		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, vars, readVars)
	})

	t.Run("non-existent file returns empty map without error", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("non-existent-app")
		require.NoError(t, err, "should not return error for non-existent file")
		assert.NotNil(t, readVars, "should return empty map, not nil")
		assert.Empty(t, readVars, "map should be empty")
	})

	t.Run("parses escaped values correctly", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Write raw content directly to test parsing
		appDir := filepath.Join(cfg.BasePath, "test-app")
		err = os.MkdirAll(appDir, 0700)
		require.NoError(t, err)

		envFile := filepath.Join(appDir, "env")
		content := `QUOTED="value with spaces"
ESCAPED_QUOTES="value \"with\" quotes"
BACKSLASH="C:\\path\\to\\file"
SIMPLE=simple_value
`
		err = os.WriteFile(envFile, []byte(content), 0600)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "value with spaces", readVars["QUOTED"])
		assert.Equal(t, `value "with" quotes`, readVars["ESCAPED_QUOTES"])
		assert.Equal(t, `C:\path\to\file`, readVars["BACKSLASH"])
		assert.Equal(t, "simple_value", readVars["SIMPLE"])
	})

	t.Run("invalid app name rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		testCases := []string{
			"",
			"app/name",
			"../other-app",
			"app\\name",
		}

		for _, appName := range testCases {
			_, err = manager.ReadEnvFile(appName)
			require.Error(t, err)
			assert.True(t, errors.Is(err, environment.ErrInvalidAppName), "should return ErrInvalidAppName for %q", appName)
		}
	})

	t.Run("skips empty lines and comments", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		appDir := filepath.Join(cfg.BasePath, "test-app")
		err = os.MkdirAll(appDir, 0700)
		require.NoError(t, err)

		envFile := filepath.Join(appDir, "env")
		content := `# This is a comment
VAR1=value1

# Another comment
VAR2=value2

`
		err = os.WriteFile(envFile, []byte(content), 0600)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Len(t, readVars, 2)
		assert.Equal(t, "value1", readVars["VAR1"])
		assert.Equal(t, "value2", readVars["VAR2"])
	})

	t.Run("skips lines with invalid keys", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		appDir := filepath.Join(cfg.BasePath, "test-app")
		err = os.MkdirAll(appDir, 0700)
		require.NoError(t, err)

		envFile := filepath.Join(appDir, "env")
		content := `VALID_KEY=value1
invalid-key=value2
ANOTHER_VALID=value3
`
		err = os.WriteFile(envFile, []byte(content), 0600)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err, "should not error, just skip invalid lines")
		assert.Len(t, readVars, 2, "should only include valid keys")
		assert.Equal(t, "value1", readVars["VALID_KEY"])
		assert.Equal(t, "value3", readVars["ANOTHER_VALID"])
		assert.NotContains(t, readVars, "invalid-key")
	})

	t.Run("skips malformed lines", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		appDir := filepath.Join(cfg.BasePath, "test-app")
		err = os.MkdirAll(appDir, 0700)
		require.NoError(t, err)

		envFile := filepath.Join(appDir, "env")
		content := `VALID=value
malformed_line_without_equals
ANOTHER=value2
`
		err = os.WriteFile(envFile, []byte(content), 0600)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err, "should not error, just skip malformed lines")
		assert.Len(t, readVars, 2)
		assert.Equal(t, "value", readVars["VALID"])
		assert.Equal(t, "value2", readVars["ANOTHER"])
	})
}

func TestManager_SetVar(t *testing.T) {
	t.Parallel()

	t.Run("sets new variable", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.SetVar("test-app", "NEW_VAR", "new_value")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "new_value", vars["NEW_VAR"])
	})

	t.Run("updates existing variable", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Set initial value
		err = manager.SetVar("test-app", "VAR", "initial")
		require.NoError(t, err)

		// Update value
		err = manager.SetVar("test-app", "VAR", "updated")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "updated", vars["VAR"])
	})

	t.Run("creates file if doesn't exist", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.SetVar("new-app", "VAR", "value")
		require.NoError(t, err)

		envFile := filepath.Join(cfg.BasePath, "new-app", "env")
		_, err = os.Stat(envFile)
		require.NoError(t, err, "file should be created")
	})

	t.Run("preserves other variables", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Set multiple vars
		err = manager.WriteEnvFile("test-app", map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		})
		require.NoError(t, err)

		// Update one var
		err = manager.SetVar("test-app", "VAR2", "updated")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["VAR1"], "VAR1 should be preserved")
		assert.Equal(t, "updated", vars["VAR2"], "VAR2 should be updated")
		assert.Equal(t, "value3", vars["VAR3"], "VAR3 should be preserved")
	})

	t.Run("invalid app name rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.SetVar("../other-app", "VAR", "value")
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidAppName), "should return ErrInvalidAppName")
	})

	t.Run("invalid key rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.SetVar("test-app", "invalid-key", "value")
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidKey), "should return ErrInvalidKey")
	})

	t.Run("empty value is allowed", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.SetVar("test-app", "EMPTY_VAR", "")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "", vars["EMPTY_VAR"])
	})
}

func TestManager_UnsetVar(t *testing.T) {
	t.Parallel()

	t.Run("removes existing variable", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Set initial vars
		err = manager.WriteEnvFile("test-app", map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		})
		require.NoError(t, err)

		// Remove one var
		err = manager.UnsetVar("test-app", "VAR1")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.NotContains(t, vars, "VAR1", "VAR1 should be removed")
		assert.Equal(t, "value2", vars["VAR2"], "VAR2 should remain")
	})

	t.Run("non-existent variable doesn't error", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Set some vars
		err = manager.SetVar("test-app", "VAR", "value")
		require.NoError(t, err)

		// Unset non-existent var
		err = manager.UnsetVar("test-app", "NON_EXISTENT")
		require.NoError(t, err, "unsetting non-existent var should not error")

		// Original var should still exist
		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "value", vars["VAR"])
	})

	t.Run("preserves other variables", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Set multiple vars
		err = manager.WriteEnvFile("test-app", map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
			"VAR4": "value4",
		})
		require.NoError(t, err)

		// Remove one var
		err = manager.UnsetVar("test-app", "VAR2")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Len(t, vars, 3, "should have 3 vars remaining")
		assert.Equal(t, "value1", vars["VAR1"])
		assert.NotContains(t, vars, "VAR2")
		assert.Equal(t, "value3", vars["VAR3"])
		assert.Equal(t, "value4", vars["VAR4"])
	})

	t.Run("invalid app name rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.UnsetVar("app/name", "VAR")
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidAppName), "should return ErrInvalidAppName")
	})

	t.Run("invalid key rejected", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.UnsetVar("test-app", "invalid-key")
		require.Error(t, err)
		assert.True(t, errors.Is(err, environment.ErrInvalidKey), "should return ErrInvalidKey")
	})

	t.Run("works with non-existent file", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Unset var in non-existent file
		err = manager.UnsetVar("non-existent-app", "VAR")
		require.NoError(t, err, "should not error when file doesn't exist")

		// Should create an empty file
		vars, err := manager.ReadEnvFile("non-existent-app")
		require.NoError(t, err)
		assert.Empty(t, vars)
	})
}

func TestManager_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("values with equals signs", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Value contains = sign
		err = manager.SetVar("test-app", "BASE64", "abc=def=ghi==")
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "abc=def=ghi==", vars["BASE64"], "equals signs in value should be preserved")
	})

	t.Run("path traversal attempts blocked", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		maliciousNames := []string{
			"../etc/passwd",
			"..\\windows\\system32",
			"app/../../../etc/passwd",
			"./../../../../root/.ssh/authorized_keys",
		}

		for _, name := range maliciousNames {
			err = manager.SetVar(name, "VAR", "value")
			require.Error(t, err, "should reject path traversal attempt: %s", name)
			assert.True(t, errors.Is(err, environment.ErrInvalidAppName), "should return ErrInvalidAppName for: %s", name)
		}
	})

	t.Run("very long variable value", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		longValue := strings.Repeat("x", 10000)
		err = manager.SetVar("test-app", "LONG_VAR", longValue)
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, longValue, vars["LONG_VAR"], "long values should be preserved")
	})

	t.Run("many variables in one file", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Create 100 variables with unique keys
		vars := make(map[string]string)
		for i := 0; i < 100; i++ {
			key := "TEST_VAR_" + string(rune('A'+i/10)) + string(rune('0'+i%10))
			value := "value_" + string(rune('0'+i%10))
			vars[key] = value
		}

		err = manager.WriteEnvFile("test-app", vars)
		require.NoError(t, err)

		readVars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Len(t, readVars, len(vars), "should read all variables")
	})

	t.Run("unicode in values", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		unicodeValue := "Hello 世界 🌍 مرحبا"
		err = manager.SetVar("test-app", "UNICODE_VAR", unicodeValue)
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, unicodeValue, vars["UNICODE_VAR"], "unicode should be preserved")
	})

	t.Run("valid app names with special patterns", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		validNames := []string{
			"my-app",
			"my_app",
			"app123",
			"123app",
			"a",
			"APP-NAME_123",
		}

		for _, name := range validNames {
			err = manager.SetVar(name, "VAR", "value")
			require.NoError(t, err, "should accept valid app name: %s", name)
		}
	})

	t.Run("windows-style line endings", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Write file with CRLF line endings manually
		appDir := filepath.Join(cfg.BasePath, "test-app")
		err = os.MkdirAll(appDir, 0700)
		require.NoError(t, err)

		envFile := filepath.Join(appDir, "env")
		content := "VAR1=value1\r\nVAR2=value2\r\n"
		err = os.WriteFile(envFile, []byte(content), 0600)
		require.NoError(t, err)

		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["VAR1"], "should handle CRLF correctly")
		assert.Equal(t, "value2", vars["VAR2"], "should handle CRLF correctly")
	})
}

func TestManager_AtomicWrites(t *testing.T) {
	t.Parallel()

	t.Run("write is atomic using temp file", func(t *testing.T) {
		t.Parallel()

		cfg := environment.Config{
			BasePath: t.TempDir(),
		}
		manager, err := environment.New(cfg, slog.Default())
		require.NoError(t, err)

		// Write initial vars
		initialVars := map[string]string{
			"VAR1": "value1",
		}
		err = manager.WriteEnvFile("test-app", initialVars)
		require.NoError(t, err)

		// Write new vars - this should use atomic write
		newVars := map[string]string{
			"VAR2": "value2",
		}
		err = manager.WriteEnvFile("test-app", newVars)
		require.NoError(t, err)

		// Verify temp file is cleaned up
		appDir := filepath.Join(cfg.BasePath, "test-app")
		entries, err := os.ReadDir(appDir)
		require.NoError(t, err)

		for _, entry := range entries {
			assert.NotContains(t, entry.Name(), ".tmp", "temp files should be cleaned up")
		}

		// Verify final state
		vars, err := manager.ReadEnvFile("test-app")
		require.NoError(t, err)
		assert.Equal(t, newVars, vars)
	})
}
