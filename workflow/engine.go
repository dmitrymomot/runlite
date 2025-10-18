package workflow

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/dmitrymomot/runlite/db/repository"
)

const (
	// defaultStaleDeploymentWindow is the time window for checking stale deployments
	defaultStaleDeploymentWindow = 30 * time.Minute
)

var (
	// ErrHandlerAlreadyRegistered is returned when trying to register a handler for a status that already has one
	ErrHandlerAlreadyRegistered = errors.New("handler already registered for status")
	// ErrNoHandler is returned when no handler is registered for a status
	ErrNoHandler = errors.New("no handler registered for status")
)

// EngineConfig holds configuration for the workflow engine
type EngineConfig struct {
	PollInterval          time.Duration
	StaleDeploymentWindow time.Duration
	StatusTimeouts        *StatusTimeouts
	Logger                *slog.Logger
}

// Engine orchestrates the deployment workflow.
// It polls for deployments needing work and executes registered handlers.
// Engine is safe for concurrent use; RegisterHandler and SetTimeout can be called
// concurrently with ProcessDeployment.
type Engine struct {
	repo                  repository.Querier
	handlers              map[Status]Handler
	timeouts              *StatusTimeouts
	pollInterval          time.Duration
	staleDeploymentWindow time.Duration
	logger                *slog.Logger
	mu                    sync.RWMutex // protects handlers and timeouts
}

// NewEngine creates a new workflow engine with the provided configuration.
// If config.Logger is nil, a no-op logger will be used.
// If config.StatusTimeouts is nil, default timeouts will be used.
// If config.StaleDeploymentWindow is 0, the default window will be used.
func NewEngine(repo repository.Querier, config EngineConfig) *Engine {
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if config.StatusTimeouts == nil {
		config.StatusTimeouts = NewStatusTimeouts()
	}
	if config.StaleDeploymentWindow == 0 {
		config.StaleDeploymentWindow = defaultStaleDeploymentWindow
	}

	return &Engine{
		repo:                  repo,
		handlers:              map[Status]Handler{},
		timeouts:              config.StatusTimeouts,
		pollInterval:          config.PollInterval,
		staleDeploymentWindow: config.StaleDeploymentWindow,
		logger:                config.Logger,
	}
}

// RegisterHandler registers a handler for a specific status.
// Only one handler can be registered per status.
// This method is safe for concurrent use.
func (e *Engine) RegisterHandler(status Status, handler Handler) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.handlers[status]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, status)
	}
	e.handlers[status] = handler
	e.logger.Info("handler registered", "status", status.String())
	return nil
}

// SetTimeout sets the timeout duration for a specific status.
// This method is safe for concurrent use.
func (e *Engine) SetTimeout(status Status, duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.timeouts.SetTimeout(status, duration)
}

// Run starts the background polling worker.
// It processes work immediately on start, then continues polling at the configured interval.
// Processing errors are logged but do not stop the engine to ensure resilience.
func (e *Engine) Run(ctx context.Context) error {
	e.logger.Info("workflow engine started", "poll_interval", e.pollInterval)

	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	if err := e.processWorkBatch(ctx); err != nil {
		e.logger.Error("initial batch processing failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("workflow engine stopping", "reason", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			if err := e.processWorkBatch(ctx); err != nil {
				e.logger.Error("batch processing failed", "error", err)
			}
		}
	}
}

// processWorkBatch processes all deployments needing work in a single batch.
// First handles any timed-out deployments, then processes ready deployments.
// Continues processing remaining deployments even if individual ones fail.
func (e *Engine) processWorkBatch(ctx context.Context) error {
	if err := e.handleTimeouts(ctx); err != nil {
		return err
	}

	deployments, err := e.repo.GetDeploymentsNeedingWork(ctx)
	if err != nil {
		return err
	}

	if len(deployments) == 0 {
		return nil
	}

	e.logger.Debug("processing deployments", "count", len(deployments))

	for _, dep := range deployments {
		if err := e.ProcessDeployment(ctx, dep.ID); err != nil {
			e.logger.Error("deployment processing failed",
				"deployment_id", dep.ID,
				"status", dep.Status,
				"error", err)
		}
	}

	return nil
}

// ProcessDeployment processes a single deployment by ID.
// This method can also be called externally for on-demand processing.
// It looks up the handler for the deployment's current status and executes it,
// then transitions to the next status based on success/failure.
func (e *Engine) ProcessDeployment(ctx context.Context, deploymentID string) error {
	dep, err := e.repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return err
	}

	status := Status(dep.Status)

	if status.IsTerminal() {
		return nil
	}

	e.mu.RLock()
	handler, exists := e.handlers[status]
	e.mu.RUnlock()

	if !exists {
		// No handler registered - some statuses transition externally
		e.logger.Debug("no handler for status", "deployment_id", deploymentID, "status", status)
		return nil
	}

	e.logger.Debug("executing handler",
		"deployment_id", deploymentID,
		"status", status)

	err = handler.Handle(ctx, &dep)

	transition, ok := GetTransition(status)
	if !ok {
		return nil
	}

	var nextStatus Status
	if err != nil {
		nextStatus = transition.OnFailure
		if logErr := e.logError(ctx, deploymentID, status.String(), err); logErr != nil {
			return fmt.Errorf("handler failed and logging failed: handler error: %w, log error: %v", err, logErr)
		}
		e.logger.Warn("handler failed",
			"deployment_id", deploymentID,
			"status", status,
			"next_status", nextStatus,
			"error", err)
	} else {
		nextStatus = transition.OnSuccess
		if logErr := e.logSuccess(ctx, deploymentID, status.String()); logErr != nil {
			return fmt.Errorf("handler succeeded but logging failed: %w", logErr)
		}
		e.logger.Info("handler succeeded",
			"deployment_id", deploymentID,
			"status", status,
			"next_status", nextStatus)
	}

	if nextStatus != "" {
		return e.updateStatus(ctx, deploymentID, nextStatus)
	}

	return nil
}

// handleTimeouts checks for deployments that have exceeded their timeout.
// Fetches deployments created within the stale deployment window to balance performance vs. coverage.
// Marks timed-out deployments as failed with a log entry.
func (e *Engine) handleTimeouts(ctx context.Context) error {
	threshold := time.Now().Add(-e.staleDeploymentWindow)
	staleDeployments, err := e.repo.GetStaleDeployments(ctx, threshold)
	if err != nil {
		return err
	}

	for _, dep := range staleDeployments {
		status := Status(dep.Status)

		if status.IsTerminal() {
			continue
		}

		e.mu.RLock()
		timeout, ok := e.timeouts.GetTimeout(status)
		e.mu.RUnlock()

		if !ok {
			continue
		}

		age := time.Since(dep.CreatedAt)
		if age > timeout {
			e.logger.Warn("deployment timeout",
				"deployment_id", dep.ID,
				"status", status,
				"age", age,
				"timeout", timeout)

			if err := e.updateStatus(ctx, dep.ID, StatusFailed); err != nil {
				e.logger.Error("failed to mark deployment as failed",
					"deployment_id", dep.ID,
					"error", err)
				continue
			}

			timeoutErr := errors.New("deployment exceeded timeout")
			if logErr := e.logError(ctx, dep.ID, "timeout", timeoutErr); logErr != nil {
				e.logger.Error("failed to log timeout",
					"deployment_id", dep.ID,
					"error", logErr)
			}
		}
	}

	return nil
}

// updateStatus updates the deployment status with appropriate timestamps.
// Terminal statuses (Active, Stopped, Cancelled, Failed) get an activated_at or stopped_at timestamp.
// Intermediate statuses are updated without special timestamps.
func (e *Engine) updateStatus(ctx context.Context, deploymentID string, status Status) error {
	statusStr := status.String()

	switch status {
	case StatusActive:
		return e.repo.UpdateDeploymentStatusWithActivatedAt(ctx, repository.UpdateDeploymentStatusWithActivatedAtParams{
			Status: statusStr,
			ID:     deploymentID,
		})
	case StatusStopped, StatusCancelled, StatusFailed:
		return e.repo.UpdateDeploymentStatusWithStoppedAt(ctx, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: statusStr,
			ID:     deploymentID,
		})
	default:
		return e.repo.UpdateDeploymentStatus(ctx, repository.UpdateDeploymentStatusParams{
			Status: statusStr,
			ID:     deploymentID,
		})
	}
}

// logSuccess records a successful event in the deployment log.
// Returns an error if log creation fails.
func (e *Engine) logSuccess(ctx context.Context, deploymentID, event string) error {
	msg := "completed successfully"
	if err := e.repo.CreateDeploymentLog(ctx, repository.CreateDeploymentLogParams{
		ID:           generateID(),
		DeploymentID: deploymentID,
		Event:        event,
		Message:      &msg,
		Error:        nil,
	}); err != nil {
		e.logger.Error("failed to create log",
			"deployment_id", deploymentID,
			"error", err)
		return fmt.Errorf("failed to create success log: %w", err)
	}
	return nil
}

// logError records a failed event in the deployment log.
// Returns an error if log creation fails.
func (e *Engine) logError(ctx context.Context, deploymentID, event string, err error) error {
	errMsg := err.Error()
	if logErr := e.repo.CreateDeploymentLog(ctx, repository.CreateDeploymentLogParams{
		ID:           generateID(),
		DeploymentID: deploymentID,
		Event:        event,
		Message:      nil,
		Error:        &errMsg,
	}); logErr != nil {
		e.logger.Error("failed to create log",
			"deployment_id", deploymentID,
			"error", logErr)
		return fmt.Errorf("failed to create error log: %w", logErr)
	}
	return nil
}

// generateID generates a unique ULID for log entries.
// ULIDs are sortable, unique, and URL-safe.
func generateID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
