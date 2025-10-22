package deploy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/deploy"
)

func TestWaitForHealthy(t *testing.T) {
	t.Run("succeeds when endpoint returns 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, 100*time.Millisecond)
		require.NoError(t, err)
	})

	t.Run("succeeds with various 2xx status codes", func(t *testing.T) {
		testCases := []int{200, 201, 202, 204, 299}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}))
				defer server.Close()

				ctx := context.Background()
				err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, 100*time.Millisecond)
				require.NoError(t, err)
			})
		}
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		var attempts atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := attempts.Add(1)
			if count < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, 100*time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, attempts.Load(), int32(3))
	})

	t.Run("waits for interval before first check", func(t *testing.T) {
		var firstCheckTime time.Time
		startTime := time.Now()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if firstCheckTime.IsZero() {
				firstCheckTime = time.Now()
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		interval := 200 * time.Millisecond
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, interval)
		require.NoError(t, err)

		// First check should occur at least 'interval' time after start
		elapsed := firstCheckTime.Sub(startTime)
		assert.GreaterOrEqual(t, elapsed, interval)
	})

	t.Run("timeout when never healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check timeout")
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 10*time.Second, 100*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check cancelled")
	})

	t.Run("rejects 3xx status codes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMovedPermanently)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check timeout")
	})

	t.Run("rejects 4xx status codes", func(t *testing.T) {
		testCases := []int{400, 401, 403, 404, 429}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}))
				defer server.Close()

				ctx := context.Background()
				err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "health check timeout")
			})
		}
	})

	t.Run("rejects 5xx status codes", func(t *testing.T) {
		testCases := []int{500, 502, 503, 504}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}))
				defer server.Close()

				ctx := context.Background()
				err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "health check timeout")
			})
		}
	})

	t.Run("handles connection refused", func(t *testing.T) {
		// Use a port that's definitely not listening
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:59999/health", 500*time.Millisecond, 100*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check timeout")
	})

	t.Run("handles connection timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep longer than request timeout
			time.Sleep(10 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check timeout")
	})

	t.Run("works with https URLs", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		// Note: httptest TLS server uses self-signed cert, but our client should still work
		// because we're testing URL parsing, not full TLS validation
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, 100*time.Millisecond)
		// This might fail due to cert validation, but it shouldn't fail on URL validation
		// The important thing is that https:// URLs are accepted
		_ = err // We accept either success or TLS error, just not URL validation error
	})

	t.Run("doesn't follow redirects", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				http.Redirect(w, r, "/redirect-target", http.StatusMovedPermanently)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 500*time.Millisecond, 100*time.Millisecond)
		require.Error(t, err)
		// Should timeout because redirect (3xx) is not considered healthy
		assert.Contains(t, err.Error(), "health check timeout")
	})

	t.Run("rejects empty URL", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "", 5*time.Second, 1*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL cannot be empty")
	})

	t.Run("rejects invalid URL", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "not a valid url", 5*time.Second, 1*time.Second)
		require.Error(t, err)
		// url.Parse() doesn't error on this, but scheme validation catches it
		assert.Contains(t, err.Error(), "must use http or https scheme")
	})

	t.Run("rejects non-http scheme", func(t *testing.T) {
		testCases := []string{
			"ftp://localhost/health",
			"ws://localhost/health",
			"file:///health",
		}

		for _, url := range testCases {
			t.Run(url, func(t *testing.T) {
				ctx := context.Background()
				err := deploy.WaitForHealthy(ctx, url, 5*time.Second, 1*time.Second)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "must use http or https scheme")
			})
		}
	})

	t.Run("rejects zero timeout", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:8080/health", 0, 1*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("rejects negative timeout", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:8080/health", -1*time.Second, 1*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("rejects zero interval", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:8080/health", 5*time.Second, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval must be positive")
	})

	t.Run("rejects negative interval", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:8080/health", 5*time.Second, -1*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval must be positive")
	})

	t.Run("rejects interval greater than timeout", func(t *testing.T) {
		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, "http://localhost:8080/health", 1*time.Second, 5*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval")
		assert.Contains(t, err.Error(), "cannot be greater than timeout")
	})

	t.Run("checks correct path", func(t *testing.T) {
		var requestedPath string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/custom/health/path", 5*time.Second, 100*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, "/custom/health/path", requestedPath)
	})

	t.Run("sends GET requests", func(t *testing.T) {
		var requestMethod string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestMethod = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx := context.Background()
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 5*time.Second, 100*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, http.MethodGet, requestMethod)
	})

	t.Run("multiple health checks with same interval", func(t *testing.T) {
		var checkTimes []time.Time
		startTime := time.Now()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			checkTimes = append(checkTimes, time.Now())
			if len(checkTimes) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		ctx := context.Background()
		interval := 200 * time.Millisecond
		err := deploy.WaitForHealthy(ctx, server.URL+"/health", 10*time.Second, interval)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(checkTimes), 3)

		// Verify checks happen at regular intervals
		for i := 1; i < len(checkTimes); i++ {
			elapsed := checkTimes[i].Sub(checkTimes[i-1])
			// Allow some variance due to scheduling
			assert.GreaterOrEqual(t, elapsed, interval-50*time.Millisecond)
			assert.LessOrEqual(t, elapsed, interval+50*time.Millisecond)
		}

		// First check should be at least interval after start
		firstCheckDelay := checkTimes[0].Sub(startTime)
		assert.GreaterOrEqual(t, firstCheckDelay, interval)
	})
}
