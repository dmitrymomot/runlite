package workflow

import (
	"context"

	"github.com/dmitrymomot/runlite/db/repository"
)

// Handler processes a deployment at a specific status.
// Handlers must be non-blocking and return quickly (typically within seconds).
// For long-running operations, start the operation asynchronously and return nil,
// then check its status on the next poll cycle.
//
// The Engine executes handlers sequentially as deployments progress through their lifecycle.
// A handler returning nil transitions the deployment to the OnSuccess status.
// A handler returning an error transitions the deployment to the OnFailure status.
type Handler interface {
	// Handle processes the deployment at its current status.
	// Returns nil on success (transition to OnSuccess status).
	// Returns error on failure (transition to OnFailure status).
	Handle(ctx context.Context, deployment *repository.Deployment) error
}
