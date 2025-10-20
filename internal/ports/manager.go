package ports

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"time"
)

var (
	// ErrInvalidPortRange is returned when min port is greater than max port
	ErrInvalidPortRange = errors.New("minimum port must be less than or equal to maximum port")
	// ErrPortAllocationFailed is returned when all retry attempts are exhausted
	ErrPortAllocationFailed = errors.New("failed to allocate free port after maximum retries")
)

// Config holds the configuration for the port manager
type Config struct {
	// MinPort is the minimum port number to allocate (default: 8000)
	MinPort int
	// MaxPort is the maximum port number to allocate (default: 9000)
	MaxPort int
	// MaxRetries is the maximum number of retry attempts (default: 5)
	MaxRetries int
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		MinPort:    8000,
		MaxPort:    9000,
		MaxRetries: 5,
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.MinPort > c.MaxPort {
		return ErrInvalidPortRange
	}
	if c.MinPort < 1 || c.MaxPort > 65535 {
		return errors.New("ports must be between 1 and 65535")
	}
	if c.MaxRetries < 1 {
		return errors.New("max retries must be at least 1")
	}
	return nil
}

// Manager manages dynamic port allocation using OS-level checks
// It is stateless and thread-safe
type Manager struct {
	config Config
	logger *slog.Logger
}

// New creates a new Manager with the given configuration
func New(config Config, logger *slog.Logger) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Manager{
		config: config,
		logger: logger.With("component", "ports"),
	}, nil
}

// FindFreePort finds and allocates a free port within the configured range
// It retries up to MaxRetries times, relying on OS-level atomic port binding
func (m *Manager) FindFreePort() (int, error) {
	var lastErr error

	for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
		port, err := m.tryAllocatePort()
		if err == nil {
			m.logger.Debug("allocated port",
				"port", port,
				"attempt", attempt+1)
			return port, nil
		}
		lastErr = err

		// Small delay between retries to reduce contention
		if attempt < m.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
		}
	}

	m.logger.Error("failed to allocate port after retries",
		"max_retries", m.config.MaxRetries,
		"error", lastErr)
	return 0, fmt.Errorf("%w: %v", ErrPortAllocationFailed, lastErr)
}

// tryAllocatePort attempts to allocate a single port
func (m *Manager) tryAllocatePort() (int, error) {
	// Generate random port in range
	port := m.config.MinPort + rand.IntN(m.config.MaxPort-m.config.MinPort+1)

	// Verify port is available at OS level
	if m.IsPortInUse(port) {
		return 0, fmt.Errorf("port %d is in use", port)
	}

	return port, nil
}

// IsPortInUse checks if a port is currently in use by attempting an OS-level bind
func (m *Manager) IsPortInUse(port int) bool {
	// Validate port is in range
	if port < m.config.MinPort || port > m.config.MaxPort {
		m.logger.Warn("port out of configured range",
			"port", port,
			"min", m.config.MinPort,
			"max", m.config.MaxPort)
		return true // Consider out-of-range ports as "in use"
	}

	// Try to bind to the port
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use
		return true
	}

	// Port is available, close the listener
	listener.Close()
	return false
}
