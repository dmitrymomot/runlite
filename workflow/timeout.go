package workflow

import "time"

// StatusTimeouts defines timeout durations for each deployment status
type StatusTimeouts struct {
	durations map[Status]time.Duration
}

// NewStatusTimeouts creates a new timeout configuration with default values
func NewStatusTimeouts() *StatusTimeouts {
	return &StatusTimeouts{
		durations: map[Status]time.Duration{
			StatusBuilding:        10 * time.Minute,
			StatusStarting:        2 * time.Minute,
			StatusHealthChecking:  5 * time.Minute,
			StatusActivating:      1 * time.Minute,
			StatusCancelling:      2 * time.Minute,
			StatusReadyToShutdown: 1 * time.Minute,
		},
	}
}

// GetTimeout returns the timeout duration for a given status.
// Returns (duration, true) if timeout is configured for the status.
// Returns (0, false) if no timeout is configured.
func (st *StatusTimeouts) GetTimeout(status Status) (time.Duration, bool) {
	timeout, ok := st.durations[status]
	return timeout, ok
}

// SetTimeout sets the timeout duration for a specific status
func (st *StatusTimeouts) SetTimeout(status Status, duration time.Duration) {
	st.durations[status] = duration
}

// HasTimeout returns true if a timeout is configured for the given status
func (st *StatusTimeouts) HasTimeout(status Status) bool {
	_, ok := st.durations[status]
	return ok
}
