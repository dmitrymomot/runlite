package workflow

import "time"

// TimeoutConfig defines timeout durations for each deployment status
type TimeoutConfig struct {
	defaults map[Status]time.Duration
}

// NewTimeoutConfig creates a new timeout configuration with default values
func NewTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		defaults: map[Status]time.Duration{
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
func (tc *TimeoutConfig) GetTimeout(status Status) (time.Duration, bool) {
	timeout, ok := tc.defaults[status]
	return timeout, ok
}

// SetTimeout sets the timeout duration for a specific status
func (tc *TimeoutConfig) SetTimeout(status Status, duration time.Duration) {
	tc.defaults[status] = duration
}

// HasTimeout returns true if a timeout is configured for the given status
func (tc *TimeoutConfig) HasTimeout(status Status) bool {
	_, ok := tc.defaults[status]
	return ok
}
