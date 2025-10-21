package env_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/env"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("no changes when maps are identical", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}
		new := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Modified)
		assert.Empty(t, result.Removed)
		assert.True(t, result.IsEmpty())
	})

	t.Run("added variables only", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
		}
		new := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Len(t, result.Added, 2)
		assert.Equal(t, "value2", result.Added["KEY2"])
		assert.Equal(t, "value3", result.Added["KEY3"])
		assert.Empty(t, result.Modified)
		assert.Empty(t, result.Removed)
		assert.False(t, result.IsEmpty())
	})

	t.Run("modified variables only", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}
		new := map[string]string{
			"KEY1": "value1",
			"KEY2": "changed_value",
			"KEY3": "another_change",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Len(t, result.Modified, 2)
		assert.Equal(t, "changed_value", result.Modified["KEY2"])
		assert.Equal(t, "another_change", result.Modified["KEY3"])
		assert.Empty(t, result.Removed)
		assert.False(t, result.IsEmpty())
	})

	t.Run("removed variables only", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}
		new := map[string]string{
			"KEY1": "value1",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Modified)
		assert.Len(t, result.Removed, 2)
		assert.Equal(t, "value2", result.Removed["KEY2"])
		assert.Equal(t, "value3", result.Removed["KEY3"])
		assert.False(t, result.IsEmpty())
	})

	t.Run("combination of added, modified, and removed", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1":    "value1",
			"KEY2":    "value2",
			"REMOVED": "old_value",
		}
		new := map[string]string{
			"KEY1":  "value1",
			"KEY2":  "modified_value",
			"ADDED": "new_value",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Len(t, result.Added, 1)
		assert.Equal(t, "new_value", result.Added["ADDED"])
		assert.Len(t, result.Modified, 1)
		assert.Equal(t, "modified_value", result.Modified["KEY2"])
		assert.Len(t, result.Removed, 1)
		assert.Equal(t, "old_value", result.Removed["REMOVED"])
		assert.False(t, result.IsEmpty())
	})

	t.Run("empty old map - all new vars are added", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{}
		new := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Len(t, result.Added, 3)
		assert.Equal(t, "value1", result.Added["KEY1"])
		assert.Equal(t, "value2", result.Added["KEY2"])
		assert.Equal(t, "value3", result.Added["KEY3"])
		assert.Empty(t, result.Modified)
		assert.Empty(t, result.Removed)
		assert.False(t, result.IsEmpty())
	})

	t.Run("empty new map - all old vars are removed", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}
		new := map[string]string{}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Modified)
		assert.Len(t, result.Removed, 3)
		assert.Equal(t, "value1", result.Removed["KEY1"])
		assert.Equal(t, "value2", result.Removed["KEY2"])
		assert.Equal(t, "value3", result.Removed["KEY3"])
		assert.False(t, result.IsEmpty())
	})

	t.Run("both maps empty", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{}
		new := map[string]string{}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Modified)
		assert.Empty(t, result.Removed)
		assert.True(t, result.IsEmpty())
	})

	t.Run("nil maps are treated as empty", func(t *testing.T) {
		t.Parallel()

		result := env.Diff(nil, nil)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Modified)
		assert.Empty(t, result.Removed)
		assert.True(t, result.IsEmpty())
	})

	t.Run("value changed to empty string", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "value1",
		}
		new := map[string]string{
			"KEY1": "",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Len(t, result.Modified, 1)
		assert.Equal(t, "", result.Modified["KEY1"])
		assert.Empty(t, result.Removed)
	})

	t.Run("value changed from empty string", func(t *testing.T) {
		t.Parallel()

		old := map[string]string{
			"KEY1": "",
		}
		new := map[string]string{
			"KEY1": "value1",
		}

		result := env.Diff(old, new)

		require.NotNil(t, result)
		assert.Empty(t, result.Added)
		assert.Len(t, result.Modified, 1)
		assert.Equal(t, "value1", result.Modified["KEY1"])
		assert.Empty(t, result.Removed)
	})
}

func TestDiffResult_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty diff result", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added:    map[string]string{},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		assert.True(t, result.IsEmpty())
	})

	t.Run("non-empty diff result with added", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"KEY1": "value1",
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		assert.False(t, result.IsEmpty())
	})

	t.Run("non-empty diff result with modified", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{},
			Modified: map[string]string{
				"KEY1": "value1",
			},
			Removed: map[string]string{},
		}

		assert.False(t, result.IsEmpty())
	})

	t.Run("non-empty diff result with removed", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added:    map[string]string{},
			Modified: map[string]string{},
			Removed: map[string]string{
				"KEY1": "value1",
			},
		}

		assert.False(t, result.IsEmpty())
	})

	t.Run("non-empty diff result with all categories", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"ADDED": "value",
			},
			Modified: map[string]string{
				"MODIFIED": "value",
			},
			Removed: map[string]string{
				"REMOVED": "value",
			},
		}

		assert.False(t, result.IsEmpty())
	})
}

func TestDiffResult_Format(t *testing.T) {
	t.Parallel()

	t.Run("empty diff shows no changes", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added:    map[string]string{},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		assert.Equal(t, "No changes", output)
	})

	t.Run("formatted output for added vars with + prefix", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"NEW_VAR": "new_value",
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		assert.Contains(t, output, "+ NEW_VAR=new_value")
	})

	t.Run("formatted output for modified vars with ~ prefix", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{},
			Modified: map[string]string{
				"CHANGED_VAR": "new_value",
			},
			Removed: map[string]string{},
		}

		output := result.Format()

		assert.Contains(t, output, "~ CHANGED_VAR=new_value")
	})

	t.Run("formatted output for removed vars with - prefix", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added:    map[string]string{},
			Modified: map[string]string{},
			Removed: map[string]string{
				"DELETED_VAR": "old_value",
			},
		}

		output := result.Format()

		assert.Contains(t, output, "- DELETED_VAR")
		// Removed vars should not show the value
		assert.NotContains(t, output, "old_value")
	})

	t.Run("alphabetical sorting of keys", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"ZEBRA":  "value",
				"ALPHA":  "value",
				"MIDDLE": "value",
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		// Find positions of each key in output
		alphaPos := strings.Index(output, "ALPHA")
		middlePos := strings.Index(output, "MIDDLE")
		zebraPos := strings.Index(output, "ZEBRA")

		assert.True(t, alphaPos < middlePos, "ALPHA should come before MIDDLE")
		assert.True(t, middlePos < zebraPos, "MIDDLE should come before ZEBRA")
	})

	t.Run("long value truncation", func(t *testing.T) {
		t.Parallel()

		longValue := strings.Repeat("a", 100)
		result := &env.DiffResult{
			Added: map[string]string{
				"LONG_VAR": longValue,
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		// Should be truncated with ... at the end
		assert.Contains(t, output, "...")
		// Should not contain the full value
		assert.NotContains(t, output, longValue)
	})

	t.Run("short value not truncated", func(t *testing.T) {
		t.Parallel()

		shortValue := "short_value"
		result := &env.DiffResult{
			Added: map[string]string{
				"SHORT_VAR": shortValue,
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		assert.Contains(t, output, shortValue)
	})

	t.Run("multi-line value handling", func(t *testing.T) {
		t.Parallel()

		multiLineValue := "line1\nline2\nline3"
		result := &env.DiffResult{
			Added: map[string]string{
				"MULTILINE_VAR": multiLineValue,
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		// Should show first line with indicator of multiple lines
		assert.Contains(t, output, "line1")
		assert.Contains(t, output, "lines)")
		// Should not contain all lines
		assert.NotContains(t, output, "line2\nline3")
	})

	t.Run("all categories combined with proper formatting", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"ADDED_VAR": "added_value",
			},
			Modified: map[string]string{
				"MODIFIED_VAR": "modified_value",
			},
			Removed: map[string]string{
				"REMOVED_VAR": "removed_value",
			},
		}

		output := result.Format()

		// Check all categories are present with proper prefixes
		assert.Contains(t, output, "ADDED_VAR=added_value")
		assert.Contains(t, output, "+")
		assert.Contains(t, output, "MODIFIED_VAR=modified_value")
		assert.Contains(t, output, "~")
		assert.Contains(t, output, "REMOVED_VAR")
		assert.Contains(t, output, "-")

		// Output should end with newline
		assert.True(t, strings.HasSuffix(output, "\n"))
	})

	t.Run("multiple keys sorted within each category", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"Z_ADDED": "value",
				"A_ADDED": "value",
			},
			Modified: map[string]string{
				"Z_MODIFIED": "value",
				"A_MODIFIED": "value",
			},
			Removed: map[string]string{
				"Z_REMOVED": "value",
				"A_REMOVED": "value",
			},
		}

		output := result.Format()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Find positions
		findLineIndex := func(prefix, key string) int {
			for i, line := range lines {
				if strings.Contains(line, prefix) && strings.Contains(line, key) {
					return i
				}
			}
			return -1
		}

		// Added should come first, then modified, then removed
		aAddedIdx := findLineIndex("+", "A_ADDED")
		zAddedIdx := findLineIndex("+", "Z_ADDED")
		aModifiedIdx := findLineIndex("~", "A_MODIFIED")
		zModifiedIdx := findLineIndex("~", "Z_MODIFIED")
		aRemovedIdx := findLineIndex("-", "A_REMOVED")
		zRemovedIdx := findLineIndex("-", "Z_REMOVED")

		// Within each category, keys should be sorted
		assert.True(t, aAddedIdx < zAddedIdx, "A_ADDED should come before Z_ADDED")
		assert.True(t, aModifiedIdx < zModifiedIdx, "A_MODIFIED should come before Z_MODIFIED")
		assert.True(t, aRemovedIdx < zRemovedIdx, "A_REMOVED should come before Z_REMOVED")

		// Categories should be in order: added, modified, removed
		assert.True(t, zAddedIdx < aModifiedIdx, "Added should come before Modified")
		assert.True(t, zModifiedIdx < aRemovedIdx, "Modified should come before Removed")
	})

	t.Run("empty string value is shown", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"EMPTY_VAR": "",
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		assert.Contains(t, output, "+ EMPTY_VAR=")
	})

	t.Run("special characters in values", func(t *testing.T) {
		t.Parallel()

		result := &env.DiffResult{
			Added: map[string]string{
				"SPECIAL_VAR": "value=with=equals&special*chars",
			},
			Modified: map[string]string{},
			Removed:  map[string]string{},
		}

		output := result.Format()

		assert.Contains(t, output, "value=with=equals&special*chars")
	})
}
