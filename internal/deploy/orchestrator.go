package deploy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/caddy"
	"github.com/dmitrymomot/runlite/internal/config"
	"github.com/dmitrymomot/runlite/internal/spec"
)

type Orchestrator struct {
	appName   string
	releaseID string
	caddy     *caddy.Client
	logger    *slog.Logger
}

func NewOrchestrator(appName, releaseID string) (*Orchestrator, error) {
	if appName == "" {
		return nil, errors.New("app name is required")
	}
	if releaseID == "" {
		return nil, errors.New("release ID is required")
	}

	return &Orchestrator{
		appName:   appName,
		releaseID: releaseID,
		caddy:     caddy.NewClient(""),
		logger:    slog.Default().With("app", appName, "release", releaseID),
	}, nil
}

func Deploy(ctx context.Context, appName, commitHash string, appSpec *spec.AppSpec, checkoutDir string) error {
	releaseID := GenerateReleaseID(commitHash)

	if err := AcquireLock(ctx, appName, releaseID); err != nil {
		return fmt.Errorf("failed to acquire deployment lock: %w", err)
	}
	defer func() {
		if err := ReleaseLock(ctx, appName); err != nil {
			slog.Error("failed to release deployment lock", "app", appName, "error", err)
		}
	}()

	orch, err := NewOrchestrator(appName, releaseID)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	mgr := app.NewManager(config.GetDataDir())
	activeDeployment, err := mgr.GetActiveDeployment(appName)
	if err != nil && !errors.Is(err, app.ErrNoActiveDeployment) {
		return fmt.Errorf("failed to get active deployment: %w", err)
	}

	if activeDeployment == nil {
		return orch.deployFirst(ctx, commitHash, appSpec, checkoutDir)
	}

	return orch.deployBlueGreen(ctx, commitHash, appSpec, checkoutDir, activeDeployment)
}

func (o *Orchestrator) deployFirst(ctx context.Context, commitHash string, appSpec *spec.AppSpec, checkoutDir string) error {
	o.logger.Info("starting first deployment")

	releaseDir := GetReleaseDir(o.appName, o.releaseID)
	if err := EnsureReleaseDir(o.appName, o.releaseID); err != nil {
		return fmt.Errorf("failed to create release directory: %w", err)
	}

	o.logger.Info("building application", "checkout_dir", checkoutDir, "release_dir", releaseDir)
	if err := Build(ctx, appSpec, checkoutDir, releaseDir); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("build failed: %w", err)
	}

	port, err := AllocateFreePort(o.appName)
	if err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	o.logger.Info("generating systemd unit file", "port", port)
	params := UnitFileParams{
		AppName:    o.appName,
		ReleaseID:  o.releaseID,
		CommitHash: commitHash,
		Port:       port,
		WorkingDir: releaseDir,
		Command:    appSpec.Run.Command,
		EnvFile:    config.GetAppEnvPath(o.appName),
	}
	if err := GenerateUnitFile(ctx, params); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to generate systemd unit: %w", err)
	}

	if err := ReloadDaemon(ctx); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	unitName := FormatUnitName(o.appName, o.releaseID)
	o.logger.Info("starting service", "unit", unitName)
	if err := StartService(ctx, unitName); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to start service: %w", err)
	}

	mgr := app.NewManager(config.GetDataDir())
	deploymentTime := time.Now()

	healthPath := "/health"
	healthTimeout := 60 * time.Second
	healthInterval := 5 * time.Second
	if appSpec.Health != nil {
		if appSpec.Health.Path != "" {
			healthPath = appSpec.Health.Path
		}
		if appSpec.Health.Timeout > 0 {
			healthTimeout = appSpec.Health.Timeout
		}
		if appSpec.Health.Interval > 0 {
			healthInterval = appSpec.Health.Interval
		}
	}
	healthURL := fmt.Sprintf("http://localhost:%d%s", port, healthPath)

	o.logger.Info("waiting for health check", "url", healthURL, "timeout", healthTimeout)
	if err := WaitForHealthy(ctx, healthURL, healthTimeout, healthInterval); err != nil {
		o.stopService(ctx, unitName)
		o.markFailed(mgr)
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("health check failed: %w", err)
	}

	o.logger.Info("adding caddy route", "port", port, "domains", appSpec.Domains)
	if err := o.caddy.AddAppRoute(o.appName, appSpec.Domains, port); err != nil {
		o.stopService(ctx, unitName)
		o.markFailed(mgr)
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to add caddy route: %w", err)
	}

	// Add deployment as active (Manager will handle state transitions)
	activeDeployment := app.Deployment{
		ReleaseID:  o.releaseID,
		Commit:     commitHash,
		Port:       port,
		Status:     "active",
		DeployedAt: deploymentTime,
	}
	if err := mgr.AddDeployment(o.appName, activeDeployment); err != nil {
		return fmt.Errorf("failed to mark deployment as active: %w", err)
	}

	o.logger.Info("first deployment completed successfully")

	if appSpec.Deploy != nil && appSpec.Deploy.KeepReleases != nil && *appSpec.Deploy.KeepReleases > 0 {
		if err := CleanupOldReleases(ctx, o.appName, *appSpec.Deploy.KeepReleases); err != nil {
			o.logger.Error("failed to cleanup old releases", "error", err)
		}
	}

	return nil
}

func (o *Orchestrator) deployBlueGreen(ctx context.Context, commitHash string, appSpec *spec.AppSpec, checkoutDir string, oldDeployment *app.Deployment) error {
	o.logger.Info("starting blue-green deployment", "old_release", oldDeployment.ReleaseID)

	releaseDir := GetReleaseDir(o.appName, o.releaseID)
	if err := EnsureReleaseDir(o.appName, o.releaseID); err != nil {
		return fmt.Errorf("failed to create release directory: %w", err)
	}

	o.logger.Info("building application", "checkout_dir", checkoutDir, "release_dir", releaseDir)
	if err := Build(ctx, appSpec, checkoutDir, releaseDir); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("build failed: %w", err)
	}

	port, err := AllocateFreePort(o.appName)
	if err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	o.logger.Info("generating systemd unit file", "port", port)
	params := UnitFileParams{
		AppName:    o.appName,
		ReleaseID:  o.releaseID,
		CommitHash: commitHash,
		Port:       port,
		WorkingDir: releaseDir,
		Command:    appSpec.Run.Command,
		EnvFile:    config.GetAppEnvPath(o.appName),
	}
	if err := GenerateUnitFile(ctx, params); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to generate systemd unit: %w", err)
	}

	if err := ReloadDaemon(ctx); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	unitName := FormatUnitName(o.appName, o.releaseID)
	o.logger.Info("starting service", "unit", unitName)
	if err := StartService(ctx, unitName); err != nil {
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to start service: %w", err)
	}

	mgr := app.NewManager(config.GetDataDir())
	deploymentTime := time.Now()

	healthPath := "/health"
	healthTimeout := 60 * time.Second
	healthInterval := 5 * time.Second
	if appSpec.Health != nil {
		if appSpec.Health.Path != "" {
			healthPath = appSpec.Health.Path
		}
		if appSpec.Health.Timeout > 0 {
			healthTimeout = appSpec.Health.Timeout
		}
		if appSpec.Health.Interval > 0 {
			healthInterval = appSpec.Health.Interval
		}
	}
	healthURL := fmt.Sprintf("http://localhost:%d%s", port, healthPath)

	o.logger.Info("waiting for health check", "url", healthURL, "timeout", healthTimeout)
	if err := WaitForHealthy(ctx, healthURL, healthTimeout, healthInterval); err != nil {
		o.stopService(ctx, unitName)
		o.markFailed(mgr)
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("health check failed: %w", err)
	}

	o.logger.Info("updating caddy route", "new_port", port, "old_port", oldDeployment.Port)
	if err := o.caddy.UpdateAppRoute(o.appName, port); err != nil {
		o.stopService(ctx, unitName)
		o.markFailed(mgr)
		if appSpec.Deploy != nil && appSpec.Deploy.CleanupOnFailure {
			o.cleanup(ctx, releaseDir)
		}
		return fmt.Errorf("failed to update caddy route: %w", err)
	}

	// Add deployment as active (Manager will handle transitioning old active to standby)
	activeDeployment := app.Deployment{
		ReleaseID:  o.releaseID,
		Commit:     commitHash,
		Port:       port,
		Status:     "active",
		DeployedAt: deploymentTime,
	}
	if err := mgr.AddDeployment(o.appName, activeDeployment); err != nil {
		return fmt.Errorf("failed to mark new deployment as active: %w", err)
	}

	drainPeriod := 30 * time.Second
	if appSpec.Deploy != nil && appSpec.Deploy.DrainPeriod > 0 {
		drainPeriod = appSpec.Deploy.DrainPeriod
	}
	o.logger.Info("draining old deployment", "duration", drainPeriod, "old_release", oldDeployment.ReleaseID)
	select {
	case <-time.After(drainPeriod):
	case <-ctx.Done():
		return ctx.Err()
	}

	oldUnitName := FormatUnitName(o.appName, oldDeployment.ReleaseID)
	o.logger.Info("stopping old service", "unit", oldUnitName)
	if err := StopService(ctx, oldUnitName); err != nil {
		o.logger.Error("failed to stop old service", "unit", oldUnitName, "error", err)
	}

	if err := DisableService(ctx, oldUnitName); err != nil {
		o.logger.Error("failed to disable old service", "unit", oldUnitName, "error", err)
	}

	if err := mgr.MarkDeploymentStopped(o.appName, oldDeployment.ReleaseID); err != nil {
		o.logger.Error("failed to mark old deployment as stopped", "error", err)
	}

	o.logger.Info("blue-green deployment completed successfully")

	if appSpec.Deploy != nil && appSpec.Deploy.KeepReleases != nil && *appSpec.Deploy.KeepReleases > 0 {
		if err := CleanupOldReleases(ctx, o.appName, *appSpec.Deploy.KeepReleases); err != nil {
			o.logger.Error("failed to cleanup old releases", "error", err)
		}
	}

	return nil
}

func (o *Orchestrator) stopService(ctx context.Context, unitName string) {
	if err := StopService(ctx, unitName); err != nil {
		o.logger.Error("failed to stop service", "unit", unitName, "error", err)
	}
	if err := DisableService(ctx, unitName); err != nil {
		o.logger.Error("failed to disable service", "unit", unitName, "error", err)
	}
}

func (o *Orchestrator) markFailed(mgr *app.Manager) {
	failedDeployment := app.Deployment{
		ReleaseID:  o.releaseID,
		Status:     "failed",
		DeployedAt: time.Now(),
	}
	if err := mgr.AddDeployment(o.appName, failedDeployment); err != nil {
		o.logger.Error("failed to mark deployment as failed", "error", err)
	}
}

func (o *Orchestrator) cleanup(ctx context.Context, releaseDir string) {
	o.logger.Info("cleaning up release directory", "dir", releaseDir)
	if err := os.RemoveAll(releaseDir); err != nil {
		o.logger.Error("failed to cleanup release directory", "dir", releaseDir, "error", err)
	}
}
