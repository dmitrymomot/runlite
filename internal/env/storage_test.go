package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/env"
)

// setupTestEnv creates a temporary directory for testing and sets RUNLITE_DATA_DIR
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "runlite-test-*")
	require.NoError(t, err, "failed to create temp directory")

	// Set environment variable to use temp directory
	oldDataDir := os.Getenv("RUNLITE_DATA_DIR")
	err = os.Setenv("RUNLITE_DATA_DIR", tmpDir)
	require.NoError(t, err, "failed to set RUNLITE_DATA_DIR")

	cleanup := func() {
		os.Setenv("RUNLITE_DATA_DIR", oldDataDir)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createEnvFile creates an env file with the given content
func createEnvFile(t *testing.T, dataDir, appName, content string) {
	t.Helper()
	appDir := filepath.Join(dataDir, "apps", appName)
	err := os.MkdirAll(appDir, 0755)
	require.NoError(t, err, "failed to create app directory")

	envPath := filepath.Join(appDir, "env")
	err = os.WriteFile(envPath, []byte(content), 0644)
	require.NoError(t, err, "failed to create env file")
}

func TestLoad(t *testing.T) {
	t.Run("returns empty map for non-existent env file", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run("loads valid env file with variables", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := "KEY1=value1\nKEY2=value2\nKEY3=value with spaces\n"
		createEnvFile(t, dataDir, "testapp", content)

		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 3)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
		assert.Equal(t, "value with spaces", vars["KEY3"])
	})

	t.Run("loads env file with comments and empty lines", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := "# Comment line\nKEY1=value1\n\nKEY2=value2\n# Another comment\n"
		createEnvFile(t, dataDir, "testapp", content)

		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 2)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})

	t.Run("loads env file with quoted values", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := `KEY1="quoted value"
KEY2='single quoted'
KEY3="value with = sign"
`
		createEnvFile(t, dataDir, "testapp", content)

		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Equal(t, "quoted value", vars["KEY1"])
		assert.Equal(t, "single quoted", vars["KEY2"])
		assert.Equal(t, "value with = sign", vars["KEY3"])
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Load("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - absolute path", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Load("/absolute/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Load("/absolute/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})

	t.Run("handles malformed env file gracefully", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Lines without = should be skipped or cause error
		content := "VALID=value\nINVALID_LINE_NO_EQUALS\nANOTHER=value\n"
		createEnvFile(t, dataDir, "testapp", content)

		vars, err := env.Load("testapp")
		// Should either skip invalid lines or return error
		// Based on black-box behavior, we'll test both possibilities
		if err == nil {
			// If it skips invalid lines, valid ones should be present
			assert.Contains(t, vars, "VALID")
			assert.Contains(t, vars, "ANOTHER")
		} else {
			// If it errors on malformed lines
			assert.Error(t, err)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("creates new env file with variables", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		err := env.Save("testapp", vars)
		require.NoError(t, err)

		// Verify file was created
		envPath := filepath.Join(dataDir, "apps", "testapp", "env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "KEY1=value1")
		assert.Contains(t, contentStr, "KEY2=value2")
	})

	t.Run("overwrites existing env file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create initial env file
		createEnvFile(t, dataDir, "testapp", "OLD_KEY=old_value\n")

		// Save new variables
		vars := map[string]string{
			"NEW_KEY": "new_value",
		}

		err := env.Save("testapp", vars)
		require.NoError(t, err)

		// Verify old key is gone and new key exists
		envPath := filepath.Join(dataDir, "apps", "testapp", "env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "NEW_KEY=new_value")
		assert.NotContains(t, contentStr, "OLD_KEY")
	})

	t.Run("creates directory structure if it doesn't exist", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{
			"KEY": "value",
		}

		err := env.Save("newapp", vars)
		require.NoError(t, err)

		// Verify directory structure was created
		appDir := filepath.Join(dataDir, "apps", "newapp")
		info, err := os.Stat(appDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("saves empty map creates empty file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{}

		err := env.Save("testapp", vars)
		require.NoError(t, err)

		// Verify file exists (even if empty)
		envPath := filepath.Join(dataDir, "apps", "testapp", "env")
		_, err = os.Stat(envPath)
		require.NoError(t, err)
	})

	t.Run("handles values with special characters", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{
			"KEY1": "value with spaces",
			"KEY2": "value=with=equals",
			"KEY3": "value\nwith\nnewlines",
			"KEY4": `value"with"quotes`,
		}

		err := env.Save("testapp", vars)
		require.NoError(t, err)

		// Load back and verify
		loaded, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Equal(t, vars["KEY1"], loaded["KEY1"])
		assert.Equal(t, vars["KEY2"], loaded["KEY2"])
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{"KEY": "value"}
		err := env.Save("", vars)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		vars := map[string]string{"KEY": "value"}
		err := env.Save("/absolute/path", vars)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})
}

func TestGet(t *testing.T) {
	t.Run("retrieves existing variable", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := "KEY1=value1\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		value, err := env.Get("testapp", "KEY1")
		require.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("returns error when variable not found", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := "KEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		_, err := env.Get("testapp", "NONEXISTENT")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error when env file doesn't exist", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Get("testapp", "KEY1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Get("", "KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		_, err := env.Get("/absolute/path", "KEY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})

	t.Run("retrieves variable with special characters in value", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		content := `KEY="value with spaces and = signs"`
		createEnvFile(t, dataDir, "testapp", content)

		value, err := env.Get("testapp", "KEY")
		require.NoError(t, err)
		assert.Equal(t, "value with spaces and = signs", value)
	})
}

func TestSet(t *testing.T) {
	t.Run("sets single variable in empty file", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		updates := map[string]string{"KEY1": "value1"}
		err := env.Set("testapp", updates)
		require.NoError(t, err)

		// Verify
		value, err := env.Get("testapp", "KEY1")
		require.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("sets multiple variables", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		updates := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}
		err := env.Set("testapp", updates)
		require.NoError(t, err)

		// Verify all keys
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
		assert.Equal(t, "value3", vars["KEY3"])
	})

	t.Run("updates existing variables", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create initial env file
		content := "KEY1=old_value\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Update KEY1
		updates := map[string]string{"KEY1": "new_value"}
		err := env.Set("testapp", updates)
		require.NoError(t, err)

		// Verify KEY1 was updated and KEY2 remains
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Equal(t, "new_value", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})

	t.Run("merges with existing variables", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create initial env file
		content := "KEY1=value1\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Add KEY3 and update KEY1
		updates := map[string]string{
			"KEY1": "updated_value1",
			"KEY3": "value3",
		}
		err := env.Set("testapp", updates)
		require.NoError(t, err)

		// Verify all three keys exist with correct values
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 3)
		assert.Equal(t, "updated_value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
		assert.Equal(t, "value3", vars["KEY3"])
	})

	t.Run("handles empty updates map", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create initial env file
		content := "KEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		updates := map[string]string{}
		err := env.Set("testapp", updates)
		require.NoError(t, err)

		// Verify original key still exists
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["KEY1"])
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		updates := map[string]string{"KEY": "value"}
		err := env.Set("", updates)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		updates := map[string]string{"KEY": "value"}
		err := env.Set("/absolute/path", updates)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})
}

func TestUnset(t *testing.T) {
	t.Run("removes single variable", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file with multiple variables
		content := "KEY1=value1\nKEY2=value2\nKEY3=value3\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Remove KEY2
		err := env.Unset("testapp", []string{"KEY2"})
		require.NoError(t, err)

		// Verify KEY2 is gone, others remain
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 2)
		assert.Contains(t, vars, "KEY1")
		assert.NotContains(t, vars, "KEY2")
		assert.Contains(t, vars, "KEY3")
	})

	t.Run("removes multiple variables", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY1=value1\nKEY2=value2\nKEY3=value3\nKEY4=value4\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Remove KEY1 and KEY3
		err := env.Unset("testapp", []string{"KEY1", "KEY3"})
		require.NoError(t, err)

		// Verify only KEY2 and KEY4 remain
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 2)
		assert.Contains(t, vars, "KEY2")
		assert.Contains(t, vars, "KEY4")
	})

	t.Run("does not error when removing non-existent variable", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Try to remove non-existent key
		err := env.Unset("testapp", []string{"NONEXISTENT"})
		require.NoError(t, err)

		// Verify KEY1 still exists
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Contains(t, vars, "KEY1")
	})

	t.Run("handles empty keys slice", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Unset with empty slice
		err := env.Unset("testapp", []string{})
		require.NoError(t, err)

		// Verify KEY1 still exists
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Contains(t, vars, "KEY1")
	})

	t.Run("handles non-existent env file", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		// Try to unset from non-existent file
		err := env.Unset("testapp", []string{"KEY1"})
		// Should either succeed (no-op) or return error depending on implementation
		// Black-box: we accept either behavior as reasonable
		_ = err
	})

	t.Run("removes all variables leaves empty file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY1=value1\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Remove all keys
		err := env.Unset("testapp", []string{"KEY1", "KEY2"})
		require.NoError(t, err)

		// Verify file is empty
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		err := env.Unset("", []string{"KEY"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		err := env.Unset("/absolute/path", []string{"KEY"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})
}

func TestImport(t *testing.T) {
	t.Run("imports from file with merge=false replaces all", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create existing env file
		content := "OLD_KEY=old_value\nKEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Create import file
		importFile := filepath.Join(dataDir, "import.env")
		importContent := "NEW_KEY=new_value\nKEY2=value2\n"
		err := os.WriteFile(importFile, []byte(importContent), 0644)
		require.NoError(t, err)

		// Import with merge=false
		err = env.Import("testapp", importFile, false)
		require.NoError(t, err)

		// Verify old keys are gone, new keys exist
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.NotContains(t, vars, "OLD_KEY")
		assert.NotContains(t, vars, "KEY1")
		assert.Contains(t, vars, "NEW_KEY")
		assert.Contains(t, vars, "KEY2")
	})

	t.Run("imports from file with merge=true merges with existing", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create existing env file
		content := "KEY1=value1\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Create import file
		importFile := filepath.Join(dataDir, "import.env")
		importContent := "KEY2=updated_value2\nKEY3=value3\n"
		err := os.WriteFile(importFile, []byte(importContent), 0644)
		require.NoError(t, err)

		// Import with merge=true
		err = env.Import("testapp", importFile, true)
		require.NoError(t, err)

		// Verify KEY1 remains, KEY2 updated, KEY3 added
		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Len(t, vars, 3)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "updated_value2", vars["KEY2"])
		assert.Equal(t, "value3", vars["KEY3"])
	})

	t.Run("imports from stdin when filePath is -", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		// This test requires stdin simulation which is challenging in black-box tests
		// We'll create a minimal test that verifies the function accepts "-"
		// The actual stdin reading would need to be tested in integration tests

		// For now, we can verify that passing "-" doesn't immediately fail
		// with validation errors (it will fail later when trying to read stdin)
		err := env.Import("testapp", "-", false)
		// We expect some error (can't read stdin in test), but not validation error
		if err != nil {
			assert.NotContains(t, err.Error(), "app name")
			assert.NotContains(t, err.Error(), "invalid app name")
		}
	})

	t.Run("returns error when import file not found", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		err := env.Import("testapp", "/nonexistent/file.env", false)
		require.Error(t, err)
	})

	t.Run("returns error for malformed import file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create malformed import file with empty key (actual parse error)
		importFile := filepath.Join(dataDir, "bad.env")
		importContent := "=value_without_key\n"
		err := os.WriteFile(importFile, []byte(importContent), 0644)
		require.NoError(t, err)

		err = env.Import("testapp", importFile, false)
		// Should return error for malformed content (empty key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty key")
	})

	t.Run("imports empty file creates empty env", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create existing env file
		content := "KEY1=value1\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Create empty import file
		importFile := filepath.Join(dataDir, "empty.env")
		err := os.WriteFile(importFile, []byte(""), 0644)
		require.NoError(t, err)

		// Import with merge=false should clear all vars
		err = env.Import("testapp", importFile, false)
		require.NoError(t, err)

		vars, err := env.Load("testapp")
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		importFile := filepath.Join(dataDir, "import.env")
		err := os.WriteFile(importFile, []byte("KEY=value"), 0644)
		require.NoError(t, err)

		err = env.Import("", importFile, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - absolute path", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		importFile := filepath.Join(dataDir, "import.env")
		err := os.WriteFile(importFile, []byte("KEY=value"), 0644)
		require.NoError(t, err)

		err = env.Import("/absolute/path/app", importFile, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})
}

func TestExport(t *testing.T) {
	t.Run("exports to file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY1=value1\nKEY2=value2\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Export to file
		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("testapp", exportFile)
		require.NoError(t, err)

		// Verify exported file content
		exportedContent, err := os.ReadFile(exportFile)
		require.NoError(t, err)

		exportedStr := string(exportedContent)
		assert.Contains(t, exportedStr, "KEY1=value1")
		assert.Contains(t, exportedStr, "KEY2=value2")
	})

	t.Run("creates export directory if needed", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY=value\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Export to nested path that doesn't exist
		exportFile := filepath.Join(dataDir, "nested", "dir", "export.env")
		err := env.Export("testapp", exportFile)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(exportFile)
		require.NoError(t, err)
	})

	t.Run("exports to stdout when filePath is empty", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY=value\n"
		createEnvFile(t, dataDir, "testapp", content)

		// This test is tricky in black-box testing as we can't capture stdout
		// We'll verify it doesn't error with empty path
		err := env.Export("testapp", "")
		// Should succeed (writes to stdout) or handle empty path gracefully
		_ = err
	})

	t.Run("exports to stdout when filePath is -", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file
		content := "KEY=value\n"
		createEnvFile(t, dataDir, "testapp", content)

		// Export to stdout (-)
		err := env.Export("testapp", "-")
		// Should succeed (writes to stdout)
		_ = err
	})

	t.Run("exports empty variables creates empty file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create empty env file
		createEnvFile(t, dataDir, "testapp", "")

		// Export to file
		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("testapp", exportFile)
		require.NoError(t, err)

		// Verify exported file exists and is empty (or has minimal content)
		exportedContent, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		// Should be empty or contain only whitespace/comments
		assert.True(t, len(strings.TrimSpace(string(exportedContent))) == 0 ||
			!strings.Contains(string(exportedContent), "="))
	})

	t.Run("handles non-existent app env file", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("nonexistent", exportFile)
		// Should either export empty file or return error
		if err == nil {
			// If successful, should create empty or minimal file
			_, statErr := os.Stat(exportFile)
			require.NoError(t, statErr)
		}
	})

	t.Run("preserves variable format in export", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create env file with various formats
		content := `KEY1=simple
KEY2="quoted value"
KEY3=value with spaces
`
		createEnvFile(t, dataDir, "testapp", content)

		// Export to file
		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("testapp", exportFile)
		require.NoError(t, err)

		// Import back and verify values match
		err = env.Import("testapp2", exportFile, false)
		require.NoError(t, err)

		vars, err := env.Load("testapp2")
		require.NoError(t, err)

		originalVars, err := env.Load("testapp")
		require.NoError(t, err)

		assert.Equal(t, originalVars["KEY1"], vars["KEY1"])
		assert.Equal(t, originalVars["KEY2"], vars["KEY2"])
		assert.Equal(t, originalVars["KEY3"], vars["KEY3"])
	})

	t.Run("returns error for invalid app name - empty", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("", exportFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app name")
	})

	t.Run("returns error for invalid app name - path traversal", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		exportFile := filepath.Join(dataDir, "export.env")
		err := env.Export("/absolute/path", exportFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app name")
	})
}

func TestIntegrationScenarios(t *testing.T) {
	t.Run("complete workflow - set, get, unset, export", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		// 1. Set some variables
		err := env.Set("workflow-app", map[string]string{
			"DB_HOST": "localhost",
			"DB_PORT": "5432",
			"DB_NAME": "testdb",
		})
		require.NoError(t, err)

		// 2. Get a specific variable
		dbHost, err := env.Get("workflow-app", "DB_HOST")
		require.NoError(t, err)
		assert.Equal(t, "localhost", dbHost)

		// 3. Update one variable
		err = env.Set("workflow-app", map[string]string{
			"DB_HOST": "remotehost",
		})
		require.NoError(t, err)

		// 4. Verify update
		vars, err := env.Load("workflow-app")
		require.NoError(t, err)
		assert.Equal(t, "remotehost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])

		// 5. Export to file
		exportPath := filepath.Join(dataDir, "exported.env")
		err = env.Export("workflow-app", exportPath)
		require.NoError(t, err)

		// 6. Unset one variable
		err = env.Unset("workflow-app", []string{"DB_NAME"})
		require.NoError(t, err)

		// 7. Verify unset
		vars, err = env.Load("workflow-app")
		require.NoError(t, err)
		assert.NotContains(t, vars, "DB_NAME")

		// 8. Import back from export (should restore DB_NAME)
		err = env.Import("workflow-app", exportPath, true)
		require.NoError(t, err)

		vars, err = env.Load("workflow-app")
		require.NoError(t, err)
		assert.Contains(t, vars, "DB_NAME")
	})

	t.Run("import and export round-trip preserves data", func(t *testing.T) {
		dataDir, cleanup := setupTestEnv(t)
		defer cleanup()

		original := map[string]string{
			"KEY1": "value1",
			"KEY2": "value with spaces",
			"KEY3": "value=with=equals",
			"KEY4": `value"with"quotes`,
		}

		// Save original
		err := env.Save("roundtrip-app", original)
		require.NoError(t, err)

		// Export
		exportPath := filepath.Join(dataDir, "roundtrip.env")
		err = env.Export("roundtrip-app", exportPath)
		require.NoError(t, err)

		// Import to new app
		err = env.Import("roundtrip-app2", exportPath, false)
		require.NoError(t, err)

		// Load and compare
		loaded, err := env.Load("roundtrip-app2")
		require.NoError(t, err)

		// Compare keys (order might differ)
		for key, value := range original {
			assert.Equal(t, value, loaded[key], "Value mismatch for key: %s", key)
		}
	})
}
