package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

var (
	// ErrInvalidConfig is returned when configuration validation fails
	ErrInvalidConfig = errors.New("invalid health check configuration")
	// ErrUnhealthy is returned when health check fails
	ErrUnhealthy = errors.New("health check failed")
	// ErrTimeout is returned when health check times out
	ErrTimeout = errors.New("health check timeout exceeded")
)

// Config holds the configuration for the health checker
type Config struct {
	// DefaultTimeout is the default timeout for health checks (default: 30s)
	DefaultTimeout time.Duration
	// DefaultInterval is the default polling interval (default: 1s)
	DefaultInterval time.Duration
	// HTTPTimeout is the timeout for individual HTTP requests (default: 5s)
	HTTPTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		DefaultTimeout:  30 * time.Second,
		DefaultInterval: 1 * time.Second,
		HTTPTimeout:     5 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidConfig)
	}
	if c.DefaultInterval <= 0 {
		return fmt.Errorf("%w: interval must be positive", ErrInvalidConfig)
	}
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("%w: HTTP timeout must be positive", ErrInvalidConfig)
	}
	if c.DefaultInterval > c.DefaultTimeout {
		return fmt.Errorf("%w: interval cannot exceed timeout", ErrInvalidConfig)
	}
	return nil
}

// Manager manages health checks for application deployments
// It is stateless and thread-safe
type Manager struct {
	config Config
	logger *slog.Logger
	client *http.Client
}

// New creates a new Manager with the given configuration
func New(config Config, logger *slog.Logger) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Manager{
		config: config,
		logger: logger.With("component", "healthcheck"),
		client: &http.Client{
			Timeout: config.HTTPTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}, nil
}

// Check performs a single health check against the specified port and path
// Returns nil if the service is healthy (HTTP 200), error otherwise
func (m *Manager) Check(port int, path string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidConfig, port)
	}

	url := fmt.Sprintf("http://localhost:%d%s", port, path)

	m.logger.Debug("performing health check",
		"url", url)

	resp, err := m.client.Get(url)
	if err != nil || resp == nil {
		m.logger.Debug("health check request failed",
			"url", url,
			"error", err)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnhealthy, err)
		}
		return fmt.Errorf("%w: nil response", ErrUnhealthy)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Debug("health check returned non-200 status",
			"url", url,
			"status", resp.StatusCode)
		return fmt.Errorf("%w: status code %d", ErrUnhealthy, resp.StatusCode)
	}

	m.logger.Debug("health check passed",
		"url", url)
	return nil
}

// Wait polls the health endpoint until it becomes healthy, the context is cancelled,
// or the timeout is exceeded. Returns nil when healthy, error otherwise.
func (m *Manager) Wait(ctx context.Context, port int, path string, timeout, interval time.Duration) error {
	if timeout <= 0 {
		timeout = m.config.DefaultTimeout
	}
	if interval <= 0 {
		interval = m.config.DefaultInterval
	}

	m.logger.Info("waiting for service to become healthy",
		"port", port,
		"path", path,
		"timeout", timeout,
		"interval", interval)

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Try immediately first
	if err := m.Check(port, path); err == nil {
		m.logger.Info("service became healthy immediately",
			"port", port,
			"path", path)
		return nil
	}

	// Then poll at intervals
	attempt := 1
	for {
		select {
		case <-timeoutCtx.Done():
			// Check if it was a timeout or cancellation
			if errors.Is(ctx.Err(), context.Canceled) {
				m.logger.Info("health check cancelled",
					"port", port,
					"path", path,
					"attempts", attempt)
				return ctx.Err()
			}
			m.logger.Warn("health check timeout exceeded",
				"port", port,
				"path", path,
				"timeout", timeout,
				"attempts", attempt)
			return fmt.Errorf("%w after %d attempts", ErrTimeout, attempt)

		case <-ticker.C:
			attempt++
			if err := m.Check(port, path); err == nil {
				m.logger.Info("service became healthy",
					"port", port,
					"path", path,
					"attempts", attempt)
				return nil
			}
			m.logger.Debug("health check attempt failed, retrying",
				"port", port,
				"path", path,
				"attempt", attempt)
		}
	}
}
