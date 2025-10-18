package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewStatusTimeouts(t *testing.T) {
	t.Parallel()

	t.Run("creates config with default timeouts", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		require.NotNil(t, config)
		require.NotNil(t, config.durations)

		expectedDefaults := map[Status]time.Duration{
			StatusBuilding:        10 * time.Minute,
			StatusStarting:        2 * time.Minute,
			StatusHealthChecking:  5 * time.Minute,
			StatusActivating:      1 * time.Minute,
			StatusCancelling:      2 * time.Minute,
			StatusReadyToShutdown: 1 * time.Minute,
		}

		for status, expectedDuration := range expectedDefaults {
			timeout, ok := config.GetTimeout(status)
			require.True(t, ok, "timeout should be configured for status %s", status)
			require.Equal(t, expectedDuration, timeout, "unexpected timeout for status %s", status)
		}
	})

	t.Run("does not configure timeouts for non-timeout statuses", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		nonTimeoutStatuses := []Status{
			StatusPending,
			StatusReady,
			StatusActive,
			StatusFailed,
			StatusCancelled,
			StatusStopped,
		}

		for _, status := range nonTimeoutStatuses {
			_, ok := config.GetTimeout(status)
			require.False(t, ok, "timeout should not be configured for status %s", status)
		}
	})

	t.Run("creates independent instances", func(t *testing.T) {
		t.Parallel()

		config1 := NewStatusTimeouts()
		config2 := NewStatusTimeouts()

		config1.SetTimeout(StatusBuilding, 20*time.Minute)

		timeout, ok := config2.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, 10*time.Minute, timeout, "config2 should have original default value")
	})
}

func TestStatusTimeouts_GetTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         Status
		expectedExists bool
		expectedValue  time.Duration
	}{
		{
			name:           "configured status - building",
			status:         StatusBuilding,
			expectedExists: true,
			expectedValue:  10 * time.Minute,
		},
		{
			name:           "configured status - starting",
			status:         StatusStarting,
			expectedExists: true,
			expectedValue:  2 * time.Minute,
		},
		{
			name:           "configured status - health checking",
			status:         StatusHealthChecking,
			expectedExists: true,
			expectedValue:  5 * time.Minute,
		},
		{
			name:           "configured status - activating",
			status:         StatusActivating,
			expectedExists: true,
			expectedValue:  1 * time.Minute,
		},
		{
			name:           "configured status - cancelling",
			status:         StatusCancelling,
			expectedExists: true,
			expectedValue:  2 * time.Minute,
		},
		{
			name:           "configured status - ready to shutdown",
			status:         StatusReadyToShutdown,
			expectedExists: true,
			expectedValue:  1 * time.Minute,
		},
		{
			name:           "unconfigured status - pending",
			status:         StatusPending,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unconfigured status - ready",
			status:         StatusReady,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unconfigured status - active",
			status:         StatusActive,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unconfigured status - failed",
			status:         StatusFailed,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unconfigured status - cancelled",
			status:         StatusCancelled,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unconfigured status - stopped",
			status:         StatusStopped,
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "empty status string",
			status:         Status(""),
			expectedExists: false,
			expectedValue:  0,
		},
		{
			name:           "unknown status string",
			status:         Status("unknown_status"),
			expectedExists: false,
			expectedValue:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := NewStatusTimeouts()
			timeout, ok := config.GetTimeout(tt.status)

			require.Equal(t, tt.expectedExists, ok, "existence check mismatch")
			require.Equal(t, tt.expectedValue, timeout, "timeout value mismatch")
		})
	}

	t.Run("returns custom timeout after SetTimeout", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		customDuration := 15 * time.Minute

		config.SetTimeout(StatusBuilding, customDuration)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, customDuration, timeout)
	})
}

func TestStatusTimeouts_SetTimeout(t *testing.T) {
	t.Parallel()

	t.Run("updates existing timeout", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		newDuration := 20 * time.Minute

		config.SetTimeout(StatusBuilding, newDuration)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, newDuration, timeout)
	})

	t.Run("adds timeout for unconfigured status", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		customDuration := 3 * time.Minute

		_, ok := config.GetTimeout(StatusPending)
		require.False(t, ok)

		config.SetTimeout(StatusPending, customDuration)

		timeout, ok := config.GetTimeout(StatusPending)
		require.True(t, ok)
		require.Equal(t, customDuration, timeout)
	})

	t.Run("handles zero duration", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		config.SetTimeout(StatusBuilding, 0)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok, "timeout should still be configured even with zero duration")
		require.Equal(t, time.Duration(0), timeout)
	})

	t.Run("handles negative duration", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		negativeDuration := -5 * time.Minute

		config.SetTimeout(StatusBuilding, negativeDuration)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, negativeDuration, timeout)
	})

	t.Run("handles very large duration", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		largeDuration := 365 * 24 * time.Hour // 1 year

		config.SetTimeout(StatusBuilding, largeDuration)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, largeDuration, timeout)
	})

	t.Run("does not affect other timeouts", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()
		originalStarting := 2 * time.Minute

		config.SetTimeout(StatusBuilding, 30*time.Minute)

		timeout, ok := config.GetTimeout(StatusStarting)
		require.True(t, ok)
		require.Equal(t, originalStarting, timeout)
	})

	t.Run("allows multiple updates to same status", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		config.SetTimeout(StatusBuilding, 15*time.Minute)
		config.SetTimeout(StatusBuilding, 25*time.Minute)
		config.SetTimeout(StatusBuilding, 35*time.Minute)

		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, 35*time.Minute, timeout, "should have the last set value")
	})
}

func TestStatusTimeouts_HasTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{
			name:     "has timeout - building",
			status:   StatusBuilding,
			expected: true,
		},
		{
			name:     "has timeout - starting",
			status:   StatusStarting,
			expected: true,
		},
		{
			name:     "has timeout - health checking",
			status:   StatusHealthChecking,
			expected: true,
		},
		{
			name:     "has timeout - activating",
			status:   StatusActivating,
			expected: true,
		},
		{
			name:     "has timeout - cancelling",
			status:   StatusCancelling,
			expected: true,
		},
		{
			name:     "has timeout - ready to shutdown",
			status:   StatusReadyToShutdown,
			expected: true,
		},
		{
			name:     "no timeout - pending",
			status:   StatusPending,
			expected: false,
		},
		{
			name:     "no timeout - ready",
			status:   StatusReady,
			expected: false,
		},
		{
			name:     "no timeout - active",
			status:   StatusActive,
			expected: false,
		},
		{
			name:     "no timeout - failed",
			status:   StatusFailed,
			expected: false,
		},
		{
			name:     "no timeout - cancelled",
			status:   StatusCancelled,
			expected: false,
		},
		{
			name:     "no timeout - stopped",
			status:   StatusStopped,
			expected: false,
		},
		{
			name:     "no timeout - empty status",
			status:   Status(""),
			expected: false,
		},
		{
			name:     "no timeout - unknown status",
			status:   Status("unknown_status"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := NewStatusTimeouts()
			result := config.HasTimeout(tt.status)

			require.Equal(t, tt.expected, result)
		})
	}

	t.Run("returns true after SetTimeout on unconfigured status", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		require.False(t, config.HasTimeout(StatusPending))

		config.SetTimeout(StatusPending, 5*time.Minute)

		require.True(t, config.HasTimeout(StatusPending))
	})

	t.Run("returns true for zero duration timeout", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		config.SetTimeout(StatusPending, 0)

		require.True(t, config.HasTimeout(StatusPending), "should return true even for zero duration")
	})

	t.Run("consistent with GetTimeout existence check", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		allStatuses := []Status{
			StatusPending,
			StatusBuilding,
			StatusStarting,
			StatusHealthChecking,
			StatusReady,
			StatusActivating,
			StatusActive,
			StatusFailed,
			StatusCancelling,
			StatusCancelled,
			StatusReadyToShutdown,
			StatusStopped,
		}

		for _, status := range allStatuses {
			hasTimeout := config.HasTimeout(status)
			_, getTimeoutExists := config.GetTimeout(status)

			require.Equal(t, getTimeoutExists, hasTimeout,
				"HasTimeout and GetTimeout existence check should be consistent for status %s", status)
		}
	})
}

func TestStatusTimeouts_Integration(t *testing.T) {
	t.Parallel()

	t.Run("complete workflow - query, modify, verify", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		require.True(t, config.HasTimeout(StatusBuilding))
		timeout, ok := config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, 10*time.Minute, timeout)

		config.SetTimeout(StatusBuilding, 15*time.Minute)

		require.True(t, config.HasTimeout(StatusBuilding))
		timeout, ok = config.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, 15*time.Minute, timeout)
	})

	t.Run("add custom timeout for terminal status", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		require.False(t, config.HasTimeout(StatusFailed))

		config.SetTimeout(StatusFailed, 30*time.Second)

		require.True(t, config.HasTimeout(StatusFailed))
		timeout, ok := config.GetTimeout(StatusFailed)
		require.True(t, ok)
		require.Equal(t, 30*time.Second, timeout)
	})

	t.Run("all configured statuses have non-zero timeouts", func(t *testing.T) {
		t.Parallel()

		config := NewStatusTimeouts()

		defaultConfiguredStatuses := []Status{
			StatusBuilding,
			StatusStarting,
			StatusHealthChecking,
			StatusActivating,
			StatusCancelling,
			StatusReadyToShutdown,
		}

		for _, status := range defaultConfiguredStatuses {
			timeout, ok := config.GetTimeout(status)
			require.True(t, ok, "status %s should have timeout configured", status)
			require.Greater(t, timeout, time.Duration(0), "status %s should have positive timeout", status)
		}
	})
}
