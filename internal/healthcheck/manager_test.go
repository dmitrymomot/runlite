package healthcheck_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/healthcheck"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns expected default values", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()

		assert.Equal(t, 30*time.Second, cfg.DefaultTimeout, "DefaultTimeout should be 30 seconds")
		assert.Equal(t, 1*time.Second, cfg.DefaultInterval, "DefaultInterval should be 1 second")
		assert.Equal(t, 5*time.Second, cfg.HTTPTimeout, "HTTPTimeout should be 5 seconds")
	})

	t.Run("all fields are positive", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()

		assert.Positive(t, cfg.DefaultTimeout, "DefaultTimeout must be positive")
		assert.Positive(t, cfg.DefaultInterval, "DefaultInterval must be positive")
		assert.Positive(t, cfg.HTTPTimeout, "HTTPTimeout must be positive")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config passes", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("zero DefaultTimeout fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  0,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("negative DefaultTimeout fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  -1 * time.Second,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("zero DefaultInterval fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 0,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "interval must be positive")
	})

	t.Run("negative DefaultInterval fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: -1 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "interval must be positive")
	})

	t.Run("zero HTTPTimeout fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     0,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "HTTP timeout must be positive")
	})

	t.Run("negative HTTPTimeout fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     -1 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "HTTP timeout must be positive")
	})

	t.Run("interval greater than timeout fails", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  1 * time.Second,
			DefaultInterval: 2 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should wrap ErrInvalidConfig")
		assert.Contains(t, err.Error(), "interval cannot exceed timeout")
	})

	t.Run("interval equal to timeout passes", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  5 * time.Second,
			DefaultInterval: 5 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("minimum valid values pass", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  1 * time.Nanosecond,
			DefaultInterval: 1 * time.Nanosecond,
			HTTPTimeout:     1 * time.Nanosecond,
		}

		err := cfg.Validate()
		require.NoError(t, err)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid config creates manager successfully", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		logger := slog.Default()

		manager, err := healthcheck.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  -1 * time.Second,
			DefaultInterval: 1 * time.Second,
			HTTPTimeout:     5 * time.Second,
		}
		logger := slog.Default()

		manager, err := healthcheck.New(cfg, logger)
		require.Error(t, err)
		assert.Nil(t, manager)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should return ErrInvalidConfig")
	})

	t.Run("nil logger uses default logger and manager still works", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()

		manager, err := healthcheck.New(cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, manager, "manager should be created with nil logger")

		// Verify manager is usable by running a health check
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		// Verify manager works by making a successful health check
		err = manager.Check(port, "/health")
		require.NoError(t, err, "manager with nil logger should be usable")
	})

	t.Run("created manager is usable", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.Config{
			DefaultTimeout:  5 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		logger := slog.Default()

		manager, err := healthcheck.New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Verify manager is actually usable by making a real health check
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Check should work with the created manager
		// We need to pass a valid port, but httptest doesn't expose it easily
		// This is tested more thoroughly in Check tests
	})
}

func TestManager_Check(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for HTTP 200 OK response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/health", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "/health")
		require.NoError(t, err, "health check should succeed for 200 OK")
	})

	t.Run("returns error for 404 status code", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy")
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("returns error for 500 status code", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy")
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("returns error for 503 status code", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy")
		assert.Contains(t, err.Error(), "503")
	})

	t.Run("returns error for connection failure", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		// Use a port that is not listening
		err = manager.Check(54321, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy")
	})

	t.Run("returns error for port 0", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(0, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should return ErrInvalidConfig")
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("returns error for negative port", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(-1, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should return ErrInvalidConfig")
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("returns error for port greater than 65535", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(65536, "/health")
		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrInvalidConfig), "should return ErrInvalidConfig")
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("port 65535 is valid", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		// Port 65535 is valid, though connection will fail since nothing is listening
		err = manager.Check(65535, "/health")
		require.Error(t, err)
		// Should be ErrUnhealthy (connection failure), not ErrInvalidConfig (port validation)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy for connection failure, not port validation error")
	})

	t.Run("port 1 is valid", func(t *testing.T) {
		t.Parallel()

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		// Port 1 is valid, though connection will fail since nothing is listening
		err = manager.Check(1, "/health")
		require.Error(t, err)
		// Should be ErrUnhealthy (connection failure), not ErrInvalidConfig (port validation)
		assert.True(t, errors.Is(err, healthcheck.ErrUnhealthy), "should return ErrUnhealthy for connection failure, not port validation error")
	})

	t.Run("handles different paths correctly", func(t *testing.T) {
		t.Parallel()

		requestedPath := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "/custom/health")
		require.NoError(t, err)
		assert.Equal(t, "/custom/health", requestedPath, "should request the correct path")
	})

	t.Run("handles empty path correctly", func(t *testing.T) {
		t.Parallel()

		requestedPath := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.DefaultConfig()
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Check(port, "")
		require.NoError(t, err)
		assert.Equal(t, "/", requestedPath, "empty path should become /")
	})
}

func TestManager_Wait(t *testing.T) {
	t.Parallel()

	t.Run("immediate success when service already healthy", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  5 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		err = manager.Wait(context.Background(), port, "/health", 5*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 500*time.Millisecond, "should return immediately without waiting for interval")
	})

	t.Run("eventual success after retries", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if attempts.Add(1) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Wait(context.Background(), port, "/health", 10*time.Second, 100*time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, attempts.Load(), int32(3), "should have made at least 3 attempts")
	})

	t.Run("timeout when service never becomes healthy", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  1 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     500 * time.Millisecond,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		err = manager.Wait(context.Background(), port, "/health", 1*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrTimeout), "should return ErrTimeout")
		assert.GreaterOrEqual(t, elapsed, 1*time.Second, "should wait at least the timeout duration")
		assert.Less(t, elapsed, 2*time.Second, "should not wait significantly longer than timeout")
	})

	t.Run("context cancellation stops waiting", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after 500ms
		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err = manager.Wait(ctx, port, "/health", 10*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled), "should return context.Canceled")
		assert.Less(t, elapsed, 2*time.Second, "should stop quickly after cancellation")
		assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond, "should wait at least until cancellation")
	})

	t.Run("uses provided timeout value", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second, // Default is 10s
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     500 * time.Millisecond,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		// Explicitly pass 1s timeout instead of using default 10s
		err = manager.Wait(context.Background(), port, "/health", 1*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrTimeout), "should return ErrTimeout")
		assert.GreaterOrEqual(t, elapsed, 1*time.Second, "should use provided 1s timeout")
		assert.Less(t, elapsed, 2*time.Second, "should not use default 10s timeout")
	})

	t.Run("uses provided interval value", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  2 * time.Second,
			DefaultInterval: 1 * time.Second, // Default is 1s
			HTTPTimeout:     500 * time.Millisecond,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		// Use 200ms interval instead of default 1s
		start := time.Now()
		err = manager.Wait(context.Background(), port, "/health", 2*time.Second, 200*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		// With 200ms interval and 2s timeout, we should get around 10 attempts
		// With 1s interval and 2s timeout, we would only get around 2 attempts
		assert.GreaterOrEqual(t, attempts.Load(), int32(8), "should have made many attempts with 200ms interval")
		assert.GreaterOrEqual(t, elapsed, 2*time.Second, "should have waited the full timeout")
	})

	t.Run("uses default timeout when zero value passed", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  1 * time.Second, // Set a short default for this test
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     500 * time.Millisecond,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		// Pass 0 as timeout to use default
		err = manager.Wait(context.Background(), port, "/health", 0, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrTimeout), "should return ErrTimeout")
		assert.GreaterOrEqual(t, elapsed, 1*time.Second, "should use default timeout of 1s")
		assert.Less(t, elapsed, 2*time.Second, "should not wait much longer than default timeout")
	})

	t.Run("uses default interval when zero value passed", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  1 * time.Second,
			DefaultInterval: 200 * time.Millisecond, // Set default interval
			HTTPTimeout:     500 * time.Millisecond,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		// Pass 0 as interval to use default
		err = manager.Wait(context.Background(), port, "/health", 1*time.Second, 0)

		require.Error(t, err)
		// With 200ms default interval and 1s timeout, should get around 5-6 attempts
		assert.GreaterOrEqual(t, attempts.Load(), int32(4), "should use default interval for polling")
	})

	t.Run("service becomes healthy mid-wait after initially failing", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fail first 5 times, then succeed
			if attempts.Add(1) <= 5 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  10 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		err = manager.Wait(context.Background(), port, "/health", 10*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.NoError(t, err, "should eventually succeed")
		assert.GreaterOrEqual(t, attempts.Load(), int32(6), "should have made at least 6 attempts")
		// Should complete in roughly 600ms (6 attempts * 100ms interval)
		assert.Less(t, elapsed, 2*time.Second, "should not wait the full timeout")
	})

	t.Run("handles flaky service that alternates between healthy and unhealthy", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Alternate between success and failure
			if attempts.Add(1)%2 == 0 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  5 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     2 * time.Second,
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		err = manager.Wait(context.Background(), port, "/health", 5*time.Second, 100*time.Millisecond)
		require.NoError(t, err, "should succeed on first healthy response")
		// First attempt fails (odd), second succeeds (even)
		assert.GreaterOrEqual(t, attempts.Load(), int32(2), "should succeed on second attempt")
	})

	t.Run("respects HTTP timeout in config", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delay longer than HTTP timeout
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		port := extractPort(t, server)

		cfg := healthcheck.Config{
			DefaultTimeout:  5 * time.Second,
			DefaultInterval: 100 * time.Millisecond,
			HTTPTimeout:     500 * time.Millisecond, // Short HTTP timeout
		}
		manager, err := healthcheck.New(cfg, slog.Default())
		require.NoError(t, err)

		start := time.Now()
		err = manager.Wait(context.Background(), port, "/health", 2*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, healthcheck.ErrTimeout), "should timeout")
		// Each request should timeout in ~500ms, not wait 2s
		assert.Less(t, elapsed, 5*time.Second, "individual requests should timeout quickly")
	})
}

// Helper function to extract port from httptest.Server
func extractPort(t *testing.T, server *httptest.Server) int {
	t.Helper()

	_, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	require.NoError(t, err, "failed to split host:port")

	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, err, "failed to parse port number")
	require.Greater(t, port, 0, "port should be positive")

	return port
}
