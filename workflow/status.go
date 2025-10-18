package workflow

// Status represents the deployment lifecycle state
type Status string

// Deployment status constants
const (
	StatusPending         Status = "pending"
	StatusBuilding        Status = "building"
	StatusStarting        Status = "starting"
	StatusHealthChecking  Status = "health_checking"
	StatusReady           Status = "ready"
	StatusActivating      Status = "activating"
	StatusActive          Status = "active"
	StatusFailed          Status = "failed"
	StatusCancelling      Status = "cancelling"
	StatusCancelled       Status = "cancelled"
	StatusReadyToShutdown Status = "ready_to_shutdown"
	StatusStopped         Status = "stopped"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsTerminal() bool {
	switch s {
	case StatusActive, StatusFailed, StatusCancelled, StatusStopped:
		return true
	default:
		return false
	}
}

// Transition defines the next status after handler execution.
// OnSuccess is used when the handler completes without error.
// OnFailure is used when the handler returns an error.
type Transition struct {
	OnSuccess Status
	OnFailure Status
}

// TransitionMap defines all valid status transitions in the deployment workflow.
// The Engine uses this map to determine the next status based on handler success/failure.
// Terminal states (Active, Failed, Cancelled, Stopped) have empty transitions.
var TransitionMap = map[Status]Transition{
	StatusPending: {
		OnSuccess: StatusBuilding,
		OnFailure: StatusFailed,
	},
	StatusBuilding: {
		OnSuccess: StatusStarting,
		OnFailure: StatusFailed,
	},
	StatusStarting: {
		OnSuccess: StatusHealthChecking,
		OnFailure: StatusFailed,
	},
	StatusHealthChecking: {
		OnSuccess: StatusReady,
		OnFailure: StatusFailed,
	},
	StatusReady: {
		OnSuccess: StatusActivating,
		OnFailure: StatusFailed,
	},
	StatusActivating: {
		OnSuccess: StatusActive,
		OnFailure: StatusFailed,
	},
	StatusCancelling: {
		OnSuccess: StatusCancelled,
		OnFailure: StatusFailed,
	},
	StatusReadyToShutdown: {
		OnSuccess: StatusStopped,
		OnFailure: StatusFailed,
	},
	StatusActive:    {},
	StatusFailed:    {},
	StatusCancelled: {},
	StatusStopped:   {},
}

// GetTransition returns the transition configuration for a given status.
// Returns (transition, true) if the status has a transition defined.
// Returns (empty transition, false) if the status is not in the map.
func GetTransition(status Status) (Transition, bool) {
	transition, ok := TransitionMap[status]
	return transition, ok
}
