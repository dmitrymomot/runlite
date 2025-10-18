package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name:     "pending status",
			status:   StatusPending,
			expected: "pending",
		},
		{
			name:     "building status",
			status:   StatusBuilding,
			expected: "building",
		},
		{
			name:     "starting status",
			status:   StatusStarting,
			expected: "starting",
		},
		{
			name:     "health checking status",
			status:   StatusHealthChecking,
			expected: "health_checking",
		},
		{
			name:     "ready status",
			status:   StatusReady,
			expected: "ready",
		},
		{
			name:     "activating status",
			status:   StatusActivating,
			expected: "activating",
		},
		{
			name:     "active status",
			status:   StatusActive,
			expected: "active",
		},
		{
			name:     "failed status",
			status:   StatusFailed,
			expected: "failed",
		},
		{
			name:     "cancelling status",
			status:   StatusCancelling,
			expected: "cancelling",
		},
		{
			name:     "cancelled status",
			status:   StatusCancelled,
			expected: "cancelled",
		},
		{
			name:     "ready to shutdown status",
			status:   StatusReadyToShutdown,
			expected: "ready_to_shutdown",
		},
		{
			name:     "stopped status",
			status:   StatusStopped,
			expected: "stopped",
		},
		{
			name:     "empty status",
			status:   Status(""),
			expected: "",
		},
		{
			name:     "invalid status",
			status:   Status("invalid_status"),
			expected: "invalid_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.status.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     Status
		isTerminal bool
	}{
		{
			name:       "active is terminal",
			status:     StatusActive,
			isTerminal: true,
		},
		{
			name:       "failed is terminal",
			status:     StatusFailed,
			isTerminal: true,
		},
		{
			name:       "cancelled is terminal",
			status:     StatusCancelled,
			isTerminal: true,
		},
		{
			name:       "stopped is terminal",
			status:     StatusStopped,
			isTerminal: true,
		},
		{
			name:       "pending is not terminal",
			status:     StatusPending,
			isTerminal: false,
		},
		{
			name:       "building is not terminal",
			status:     StatusBuilding,
			isTerminal: false,
		},
		{
			name:       "starting is not terminal",
			status:     StatusStarting,
			isTerminal: false,
		},
		{
			name:       "health checking is not terminal",
			status:     StatusHealthChecking,
			isTerminal: false,
		},
		{
			name:       "ready is not terminal",
			status:     StatusReady,
			isTerminal: false,
		},
		{
			name:       "activating is not terminal",
			status:     StatusActivating,
			isTerminal: false,
		},
		{
			name:       "cancelling is not terminal",
			status:     StatusCancelling,
			isTerminal: false,
		},
		{
			name:       "ready to shutdown is not terminal",
			status:     StatusReadyToShutdown,
			isTerminal: false,
		},
		{
			name:       "empty status is not terminal",
			status:     Status(""),
			isTerminal: false,
		},
		{
			name:       "invalid status is not terminal",
			status:     Status("unknown_status"),
			isTerminal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.status.IsTerminal()
			assert.Equal(t, tt.isTerminal, result)
		})
	}
}

func TestGetTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		status            Status
		expectedTranstion Transition
		expectedOK        bool
	}{
		{
			name:   "pending status transitions",
			status: StatusPending,
			expectedTranstion: Transition{
				OnSuccess: StatusBuilding,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "building status transitions",
			status: StatusBuilding,
			expectedTranstion: Transition{
				OnSuccess: StatusStarting,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "starting status transitions",
			status: StatusStarting,
			expectedTranstion: Transition{
				OnSuccess: StatusHealthChecking,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "health checking status transitions",
			status: StatusHealthChecking,
			expectedTranstion: Transition{
				OnSuccess: StatusReady,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "ready status transitions",
			status: StatusReady,
			expectedTranstion: Transition{
				OnSuccess: StatusActivating,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "activating status transitions",
			status: StatusActivating,
			expectedTranstion: Transition{
				OnSuccess: StatusActive,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "cancelling status transitions",
			status: StatusCancelling,
			expectedTranstion: Transition{
				OnSuccess: StatusCancelled,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:   "ready to shutdown status transitions",
			status: StatusReadyToShutdown,
			expectedTranstion: Transition{
				OnSuccess: StatusStopped,
				OnFailure: StatusFailed,
			},
			expectedOK: true,
		},
		{
			name:              "active status has empty transition",
			status:            StatusActive,
			expectedTranstion: Transition{},
			expectedOK:        true,
		},
		{
			name:              "failed status has empty transition",
			status:            StatusFailed,
			expectedTranstion: Transition{},
			expectedOK:        true,
		},
		{
			name:              "cancelled status has empty transition",
			status:            StatusCancelled,
			expectedTranstion: Transition{},
			expectedOK:        true,
		},
		{
			name:              "stopped status has empty transition",
			status:            StatusStopped,
			expectedTranstion: Transition{},
			expectedOK:        true,
		},
		{
			name:              "empty status not found",
			status:            Status(""),
			expectedTranstion: Transition{},
			expectedOK:        false,
		},
		{
			name:              "invalid status not found",
			status:            Status("invalid_status"),
			expectedTranstion: Transition{},
			expectedOK:        false,
		},
		{
			name:              "unknown status not found",
			status:            Status("deploying"),
			expectedTranstion: Transition{},
			expectedOK:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transition, ok := GetTransition(tt.status)
			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedTranstion, transition)
		})
	}
}

func TestTransitionMap_Completeness(t *testing.T) {
	t.Parallel()

	// All defined status constants
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

	t.Run("all statuses exist in transition map", func(t *testing.T) {
		t.Parallel()

		for _, status := range allStatuses {
			_, exists := TransitionMap[status]
			assert.True(t, exists, "status %s should exist in TransitionMap", status)
		}
	})

	t.Run("non-terminal states have valid transitions", func(t *testing.T) {
		t.Parallel()

		for _, status := range allStatuses {
			if status.IsTerminal() {
				transition := TransitionMap[status]
				assert.Empty(t, transition.OnSuccess, "terminal status %s should have empty OnSuccess", status)
				assert.Empty(t, transition.OnFailure, "terminal status %s should have empty OnFailure", status)
			} else {
				transition := TransitionMap[status]
				assert.NotEmpty(t, transition.OnSuccess, "non-terminal status %s should have OnSuccess defined", status)
			}
		}
	})

	t.Run("all transition targets are valid statuses", func(t *testing.T) {
		t.Parallel()

		validStatuses := make(map[Status]bool)
		for _, status := range allStatuses {
			validStatuses[status] = true
		}

		for status, transition := range TransitionMap {
			if transition.OnSuccess != "" {
				assert.True(t, validStatuses[transition.OnSuccess],
					"status %s has invalid OnSuccess transition to %s", status, transition.OnSuccess)
			}
			if transition.OnFailure != "" {
				assert.True(t, validStatuses[transition.OnFailure],
					"status %s has invalid OnFailure transition to %s", status, transition.OnFailure)
			}
		}
	})

	t.Run("failure transitions lead to terminal states", func(t *testing.T) {
		t.Parallel()

		for status, transition := range TransitionMap {
			if transition.OnFailure != "" {
				assert.True(t, transition.OnFailure.IsTerminal(),
					"status %s has OnFailure transition to non-terminal state %s", status, transition.OnFailure)
			}
		}
	})

	t.Run("no circular transitions to same status", func(t *testing.T) {
		t.Parallel()

		for status, transition := range TransitionMap {
			if transition.OnSuccess != "" {
				assert.NotEqual(t, status, transition.OnSuccess,
					"status %s has circular OnSuccess transition to itself", status)
			}
			if transition.OnFailure != "" {
				assert.NotEqual(t, status, transition.OnFailure,
					"status %s has circular OnFailure transition to itself", status)
			}
		}
	})
}

func TestTransitionMap_WorkflowIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("happy path from pending to active", func(t *testing.T) {
		t.Parallel()

		expectedPath := []Status{
			StatusPending,
			StatusBuilding,
			StatusStarting,
			StatusHealthChecking,
			StatusReady,
			StatusActivating,
			StatusActive,
		}

		current := StatusPending
		for i := 0; i < len(expectedPath)-1; i++ {
			require.Equal(t, expectedPath[i], current, "unexpected status at step %d", i)

			transition, ok := GetTransition(current)
			require.True(t, ok, "transition should exist for status %s", current)
			require.NotEmpty(t, transition.OnSuccess, "OnSuccess should be defined for %s", current)

			current = transition.OnSuccess
		}

		require.Equal(t, StatusActive, current)
		require.True(t, current.IsTerminal())
	})

	t.Run("failure path from any status leads to terminal state", func(t *testing.T) {
		t.Parallel()

		nonTerminalStatuses := []Status{
			StatusPending,
			StatusBuilding,
			StatusStarting,
			StatusHealthChecking,
			StatusReady,
			StatusActivating,
			StatusCancelling,
			StatusReadyToShutdown,
		}

		for _, status := range nonTerminalStatuses {
			transition, ok := GetTransition(status)
			require.True(t, ok, "transition should exist for %s", status)

			if transition.OnFailure != "" {
				assert.True(t, transition.OnFailure.IsTerminal(),
					"failure from %s should lead to terminal state, got %s", status, transition.OnFailure)
			}
		}
	})

	t.Run("cancelling workflow leads to cancelled", func(t *testing.T) {
		t.Parallel()

		transition, ok := GetTransition(StatusCancelling)
		require.True(t, ok)
		assert.Equal(t, StatusCancelled, transition.OnSuccess)
		assert.True(t, transition.OnSuccess.IsTerminal())
	})

	t.Run("shutdown workflow leads to stopped", func(t *testing.T) {
		t.Parallel()

		transition, ok := GetTransition(StatusReadyToShutdown)
		require.True(t, ok)
		assert.Equal(t, StatusStopped, transition.OnSuccess)
		assert.True(t, transition.OnSuccess.IsTerminal())
	})
}

func TestStatus_TypeSafety(t *testing.T) {
	t.Parallel()

	t.Run("status is string type", func(t *testing.T) {
		t.Parallel()

		var s Status = "test"
		str := s.String()
		assert.Equal(t, "test", str)
		assert.IsType(t, "", str)
	})

	t.Run("status constants are unique", func(t *testing.T) {
		t.Parallel()

		statuses := []Status{
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

		seen := make(map[string]bool)
		for _, status := range statuses {
			str := status.String()
			assert.False(t, seen[str], "duplicate status value found: %s", str)
			seen[str] = true
		}

		assert.Equal(t, len(statuses), len(seen), "all status constants should be unique")
	})
}
