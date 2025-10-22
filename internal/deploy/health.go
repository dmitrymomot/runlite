package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	// healthCheckRequestTimeout is the per-request timeout for health checks
	healthCheckRequestTimeout = 5 * time.Second
)

// WaitForHealthy performs health checks against the given URL until it returns
// a 2xx status code or the timeout expires. It is stateless and universal,
// working with both localhost and external URLs.
//
// The first check occurs after waiting for 'interval' duration to allow apps
// time to initialize (e.g., run migrations).
//
// Parameters:
//   - ctx: Context for cancellation
//   - healthURL: Full health check URL (e.g., "http://localhost:8080/health" or "https://myapp.com/health")
//   - timeout: Total time to wait before giving up
//   - interval: Time to wait between checks
func WaitForHealthy(ctx context.Context, healthURL string, timeout, interval time.Duration) error {
	// Validate URL
	if healthURL == "" {
		return fmt.Errorf("health check URL cannot be empty")
	}

	parsedURL, err := url.Parse(healthURL)
	if err != nil {
		return fmt.Errorf("invalid health check URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("health check URL must use http or https scheme, got %q", parsedURL.Scheme)
	}

	// Validate durations
	if timeout <= 0 {
		return fmt.Errorf("health check timeout must be positive, got %v", timeout)
	}

	if interval <= 0 {
		return fmt.Errorf("health check interval must be positive, got %v", interval)
	}

	if interval > timeout {
		return fmt.Errorf("health check interval (%v) cannot be greater than timeout (%v)", interval, timeout)
	}

	// Create HTTP client
	client := createHealthCheckClient()

	// Start polling
	return pollHealth(ctx, client, healthURL, timeout, interval)
}

// createHealthCheckClient creates an HTTP client configured for health checks
func createHealthCheckClient() *http.Client {
	return &http.Client{
		Timeout: healthCheckRequestTimeout,
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			DisableCompression:  true,
			MaxIdleConns:        1,
			IdleConnTimeout:     1 * time.Second,
			TLSHandshakeTimeout: 3 * time.Second,
		},
		// Don't follow redirects - they indicate misconfiguration
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// pollHealth polls the health endpoint until it's healthy or timeout occurs
func pollHealth(ctx context.Context, client *http.Client, healthURL string, timeout, interval time.Duration) error {
	slog.InfoContext(ctx, "starting health check",
		"url", healthURL,
		"timeout", timeout,
		"interval", interval)

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check cancelled: %w", ctx.Err())

		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("health check timeout after %v", timeout)
			}

			if err := checkHealth(client, healthURL); err == nil {
				slog.InfoContext(ctx, "health check passed", "url", healthURL)
				return nil
			} else {
				slog.DebugContext(ctx, "health check failed, retrying",
					"url", healthURL,
					"error", err,
					"next_check_in", interval)
			}
		}
	}
}

// checkHealth performs a single health check request
func checkHealth(client *http.Client, healthURL string) error {
	resp, err := client.Get(healthURL)
	if err != nil {
		return err
	}

	// Guard against nil response
	if resp == nil {
		return fmt.Errorf("nil response from health check")
	}

	defer resp.Body.Close()

	// Only 2xx status codes are considered healthy
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}
