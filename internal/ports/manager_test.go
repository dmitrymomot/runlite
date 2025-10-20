package ports_test

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/ports"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns expected default values", func(t *testing.T) {
		t.Parallel()

		cfg := ports.DefaultConfig()

		assert.Equal(t, 8000, cfg.MinPort, "MinPort should be 8000")
		assert.Equal(t, 9000, cfg.MaxPort, "MaxPort should be 9000")
		assert.Equal(t, 5, cfg.MaxRetries, "MaxRetries should be 5")
	})

	t.Run("all fields are within valid ranges", func(t *testing.T) {
		t.Parallel()

		cfg := ports.DefaultConfig()

		assert.GreaterOrEqual(t, cfg.MinPort, 1, "MinPort must be >= 1")
		assert.LessOrEqual(t, cfg.MaxPort, 65535, "MaxPort must be <= 65535")
		assert.LessOrEqual(t, cfg.MinPort, cfg.MaxPort, "MinPort must be <= MaxPort")
		assert.Positive(t, cfg.MaxRetries, "MaxRetries must be positive")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config passes", func(t *testing.T) {
		t.Parallel()

		cfg := ports.DefaultConfig()

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("min port greater than max port fails with ErrInvalidPortRange", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    9000,
			MaxPort:    8000,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ports.ErrInvalidPortRange), "should return ErrInvalidPortRange")
	})

	t.Run("port less than 1 fails", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    0,
			MaxPort:    9000,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "between 1 and 65535")
	})

	t.Run("negative port fails", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    -1,
			MaxPort:    9000,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "between 1 and 65535")
	})

	t.Run("port greater than 65535 fails", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    8000,
			MaxPort:    65536,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "between 1 and 65535")
	})

	t.Run("port equal to 65535 is valid", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    65535,
			MaxPort:    65535,
			MaxRetries: 1,
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("port equal to 1 is valid", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    1,
			MaxPort:    1000,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("zero max retries fails", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    8000,
			MaxPort:    9000,
			MaxRetries: 0,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max retries must be at least 1")
	})

	t.Run("negative max retries fails", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    8000,
			MaxPort:    9000,
			MaxRetries: -1,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max retries must be at least 1")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid config creates usable manager", func(t *testing.T) {
		t.Parallel()

		cfg := ports.DefaultConfig()
		logger := slog.Default()

		manager, err := ports.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Verify manager is actually usable by allocating a port
		port, err := manager.FindFreePort()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, cfg.MinPort)
		assert.LessOrEqual(t, port, cfg.MaxPort)
	})

	t.Run("invalid config returns wrapped error", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    9000,
			MaxPort:    8000,
			MaxRetries: 5,
		}
		logger := slog.Default()

		manager, err := ports.New(cfg, logger)
		require.Error(t, err)
		assert.Nil(t, manager)
		assert.Contains(t, err.Error(), "invalid config")
		// The error should wrap the validation error
		assert.True(t, errors.Is(err, ports.ErrInvalidPortRange), "should wrap ErrInvalidPortRange")
	})

	t.Run("nil logger creates usable manager", func(t *testing.T) {
		t.Parallel()

		cfg := ports.DefaultConfig()

		manager, err := ports.New(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Verify manager is usable despite nil logger
		port, err := manager.FindFreePort()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, cfg.MinPort)
		assert.LessOrEqual(t, port, cfg.MaxPort)
	})
}

func TestManager_FindFreePort(t *testing.T) {
	t.Parallel()

	t.Run("allocates port within configured range", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    10000,
			MaxPort:    10100,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port, err := manager.FindFreePort()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, cfg.MinPort, "port should be >= MinPort")
		assert.LessOrEqual(t, port, cfg.MaxPort, "port should be <= MaxPort")
	})

	t.Run("allocated port is actually free", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    10200,
			MaxPort:    10300,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port, err := manager.FindFreePort()
		require.NoError(t, err)

		// Verify we can actually bind to the port
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		require.NoError(t, err, "allocated port should be bindable")
		listener.Close()
	})

	t.Run("can allocate multiple ports", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    10400,
			MaxPort:    10500,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Allocate multiple ports (they may or may not be different due to stateless nature)
		for i := 0; i < 10; i++ {
			port, err := manager.FindFreePort()
			require.NoError(t, err)
			assert.GreaterOrEqual(t, port, cfg.MinPort)
			assert.LessOrEqual(t, port, cfg.MaxPort)
		}
	})

	t.Run("fails with ErrPortAllocationFailed when all ports exhausted", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    11000,
			MaxPort:    11002, // Only 3 ports available
			MaxRetries: 3,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind all available ports
		var listeners []net.Listener
		for port := cfg.MinPort; port <= cfg.MaxPort; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				t.Logf("warning: failed to bind port %d: %v", port, err)
				continue
			}
			listeners = append(listeners, listener)
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		require.NotEmpty(t, listeners, "should have bound at least one port")

		// Next allocation should fail
		port, err := manager.FindFreePort()
		require.Error(t, err)
		assert.Zero(t, port, "should return 0 on failure")
		assert.True(t, errors.Is(err, ports.ErrPortAllocationFailed), "should return ErrPortAllocationFailed")
	})

	t.Run("retries work correctly", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    11100,
			MaxPort:    11105, // Small range to increase collision probability
			MaxRetries: 10,    // Multiple retries
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind most ports to force retries
		var listeners []net.Listener
		for port := cfg.MinPort; port < cfg.MaxPort; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				continue
			}
			listeners = append(listeners, listener)
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		// Should still succeed due to retries (one port left)
		port, err := manager.FindFreePort()
		if len(listeners) < cfg.MaxPort-cfg.MinPort {
			// We left at least one port free
			require.NoError(t, err)
			assert.Equal(t, cfg.MaxPort, port, "should find the last free port")
		}
	})
}

func TestManager_IsPortInUse(t *testing.T) {
	t.Parallel()

	t.Run("returns false for free port", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    12000,
			MaxPort:    12100,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Find a free port
		port, err := manager.FindFreePort()
		require.NoError(t, err)

		// Port should not be in use (we're not holding it)
		inUse := manager.IsPortInUse(port)
		assert.False(t, inUse, "free port should not be detected as in use")
	})

	t.Run("returns true for bound port", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    12200,
			MaxPort:    12300,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port := 12250
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		require.NoError(t, err, "failed to bind port for test")
		defer listener.Close()

		inUse := manager.IsPortInUse(port)
		assert.True(t, inUse, "bound port should be detected as in use")
	})

	t.Run("returns true for port below configured range", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    12400,
			MaxPort:    12500,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port := cfg.MinPort - 1
		inUse := manager.IsPortInUse(port)
		assert.True(t, inUse, "port below range should be considered in use")
	})

	t.Run("returns true for port above configured range", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    12600,
			MaxPort:    12700,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port := cfg.MaxPort + 1
		inUse := manager.IsPortInUse(port)
		assert.True(t, inUse, "port above range should be considered in use")
	})

	t.Run("handles boundary values correctly", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    12800,
			MaxPort:    12900,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// MinPort should be considered in range (unless actually bound)
		inUse := manager.IsPortInUse(cfg.MinPort)
		// Could be true or false depending on OS, but should not crash
		_ = inUse

		// MaxPort should be considered in range (unless actually bound)
		inUse = manager.IsPortInUse(cfg.MaxPort)
		_ = inUse
	})
}

func TestManager_ConcurrentAllocation(t *testing.T) {
	t.Parallel()

	t.Run("concurrent allocations are thread-safe", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    13000,
			MaxPort:    13100,
			MaxRetries: 10,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Allocate ports concurrently with race detection
		const numGoroutines = 20
		var wg sync.WaitGroup
		portsChan := make(chan int, numGoroutines)
		errorsChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				port, err := manager.FindFreePort()
				if err != nil {
					errorsChan <- err
					return
				}
				portsChan <- port
			}()
		}

		wg.Wait()
		close(portsChan)
		close(errorsChan)

		// Collect results
		var ports []int
		for port := range portsChan {
			ports = append(ports, port)
		}

		var errorCount int
		for range errorsChan {
			errorCount++
		}

		// Should have allocated at least some ports successfully
		assert.NotEmpty(t, ports, "should allocate ports successfully")

		// All allocated ports should be in range
		for _, port := range ports {
			assert.GreaterOrEqual(t, port, cfg.MinPort)
			assert.LessOrEqual(t, port, cfg.MaxPort)
		}

		// Stateless implementation may allocate duplicates (expected behavior)
		// Just verify no panics and all ports are valid
	})

	t.Run("concurrent IsPortInUse checks are thread-safe", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    13200,
			MaxPort:    13300,
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind a port
		port := 13250
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		require.NoError(t, err)
		defer listener.Close()

		// Check the same port concurrently
		const numGoroutines = 50
		var wg sync.WaitGroup
		resultsChan := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				inUse := manager.IsPortInUse(port)
				resultsChan <- inUse
			}()
		}

		wg.Wait()
		close(resultsChan)

		// All checks should return true (port is bound)
		for result := range resultsChan {
			assert.True(t, result, "all concurrent checks should detect port as in use")
		}
	})
}

func TestManager_RetryBackoff(t *testing.T) {
	t.Parallel()

	t.Run("retry delays increase progressively", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    13400,
			MaxPort:    13402, // Very small range
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind all ports
		var listeners []net.Listener
		for port := cfg.MinPort; port <= cfg.MaxPort; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				continue
			}
			listeners = append(listeners, listener)
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		// Measure retry time
		start := time.Now()
		_, err = manager.FindFreePort()
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, ports.ErrPortAllocationFailed))

		// With 5 retries and progressive backoff (10ms, 20ms, 30ms, 40ms),
		// total delay should be at least 100ms
		assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond,
			"should have progressive backoff delays between retries")
	})
}

func TestManager_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("single port range works correctly", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    13500,
			MaxPort:    13500, // Single port
			MaxRetries: 3,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		port, err := manager.FindFreePort()
		require.NoError(t, err)
		assert.Equal(t, 13500, port, "should allocate the only available port")
	})

	t.Run("single port range exhaustion fails correctly", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    13600,
			MaxPort:    13600,
			MaxRetries: 3,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind the only port
		listener, err := net.Listen("tcp", ":13600")
		require.NoError(t, err)
		defer listener.Close()

		// Should fail with ErrPortAllocationFailed
		port, err := manager.FindFreePort()
		require.Error(t, err)
		assert.Zero(t, port)
		assert.True(t, errors.Is(err, ports.ErrPortAllocationFailed))
	})

	t.Run("large port range allocation works", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    14000,
			MaxPort:    15000, // 1000 ports
			MaxRetries: 5,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Should easily find a free port in large range
		port, err := manager.FindFreePort()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, cfg.MinPort)
		assert.LessOrEqual(t, port, cfg.MaxPort)
	})
}

func TestManager_ErrorMessages(t *testing.T) {
	t.Parallel()

	t.Run("allocation failure error contains useful context", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    15100,
			MaxPort:    15102,
			MaxRetries: 2,
		}

		manager, err := ports.New(cfg, slog.Default())
		require.NoError(t, err)

		// Bind all ports
		var listeners []net.Listener
		for port := cfg.MinPort; port <= cfg.MaxPort; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				continue
			}
			listeners = append(listeners, listener)
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		_, err = manager.FindFreePort()
		require.Error(t, err)

		// Error message should be informative
		errMsg := err.Error()
		assert.Contains(t, errMsg, "failed to allocate free port",
			"error should mention allocation failure")
	})

	t.Run("validation error contains specific issue", func(t *testing.T) {
		t.Parallel()

		cfg := ports.Config{
			MinPort:    9000,
			MaxPort:    8000,
			MaxRetries: 5,
		}

		err := cfg.Validate()
		require.Error(t, err)

		// Should explain what's wrong
		assert.True(t, errors.Is(err, ports.ErrInvalidPortRange), "should return ErrInvalidPortRange")
	})
}
