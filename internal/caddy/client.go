package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the default Caddy admin API endpoint.
	DefaultBaseURL = "http://localhost:2019"

	// DefaultServerName is the Caddy HTTP server name used by runlite.
	// For single-instance PaaS deployments, one server handles all app routes.
	// This can be changed via Client.ServerName if a different server name is needed.
	DefaultServerName = "srv0"
)

// Client represents a Caddy admin API client.
type Client struct {
	BaseURL    string
	ServerName string
	httpClient *http.Client
}

// NewClient creates a new Caddy admin API client.
// If baseURL is empty, DefaultBaseURL is used.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL:    baseURL,
		ServerName: DefaultServerName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AddAppRoute adds a new route for an application with multiple domains.
func (c *Client) AddAppRoute(appName string, domains []string, upstreamPort int) error {
	route := NewAppRouteMulti(appName, domains, upstreamPort)

	url := fmt.Sprintf("%s/config/apps/http/servers/%s/routes", c.BaseURL, c.ServerName)

	return c.doRequest("POST", url, route)
}

// UpdateAppRoute updates an existing app route's upstream port.
func (c *Client) UpdateAppRoute(appName string, upstreamPort int) error {
	upstream := Upstream{
		Dial: formatUpstream(upstreamPort),
	}

	url := fmt.Sprintf("%s/id/runlite-%s/handle/0/upstreams/0", c.BaseURL, appName)

	return c.doRequest("PATCH", url, upstream)
}

// DeleteAppRoute deletes an app route.
func (c *Client) DeleteAppRoute(appName string) error {
	url := fmt.Sprintf("%s/id/runlite-%s", c.BaseURL, appName)

	return c.doRequest("DELETE", url, nil)
}

// GetConfig retrieves configuration at the specified path.
func (c *Client) GetConfig(path string) ([]byte, error) {
	url := fmt.Sprintf("%s/config/%s", c.BaseURL, path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response from server")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp.StatusCode, body)
	}

	return body, nil
}

// doRequest performs an HTTP request to the Caddy API.
func (c *Client) doRequest(method, url string, payload any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("nil response from server")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Success status codes for different methods
	successCodes := map[string]int{
		"POST":   http.StatusOK,
		"PATCH":  http.StatusOK,
		"DELETE": http.StatusOK,
		"PUT":    http.StatusOK,
	}

	expectedStatus := successCodes[method]
	if resp.StatusCode != expectedStatus {
		return c.parseError(resp.StatusCode, respBody)
	}

	return nil
}

// AddDomainToApp appends a single domain to an existing app route.
func (c *Client) AddDomainToApp(appName, domain string) error {
	// Step 1: Get current domains
	path := fmt.Sprintf("id/runlite-%s/match/0/host", appName)
	currentDomainsJSON, err := c.GetConfig(path)
	if err != nil {
		return fmt.Errorf("get current domains: %w", err)
	}

	// Step 2: Parse current array
	var currentDomains []string
	if err := json.Unmarshal(currentDomainsJSON, &currentDomains); err != nil {
		return fmt.Errorf("parse domains: %w", err)
	}

	// Step 3: Check if domain already exists
	for _, d := range currentDomains {
		if d == domain {
			return nil // Already exists, no-op
		}
	}

	// Step 4: Append new domain
	currentDomains = append(currentDomains, domain)

	// Step 5: PATCH with updated array
	url := fmt.Sprintf("%s/id/runlite-%s/match/0/host", c.BaseURL, appName)
	return c.doRequest("PATCH", url, currentDomains)
}

// RemoveDomainFromApp removes a single domain from an app route.
func (c *Client) RemoveDomainFromApp(appName, domain string) error {
	// Step 1: Get current domains
	path := fmt.Sprintf("id/runlite-%s/match/0/host", appName)
	currentDomainsJSON, err := c.GetConfig(path)
	if err != nil {
		return fmt.Errorf("get current domains: %w", err)
	}

	// Step 2: Parse current array
	var currentDomains []string
	if err := json.Unmarshal(currentDomainsJSON, &currentDomains); err != nil {
		return fmt.Errorf("parse domains: %w", err)
	}

	// Step 3: Filter out the domain
	filtered := make([]string, 0, len(currentDomains))
	found := false
	for _, d := range currentDomains {
		if d != domain {
			filtered = append(filtered, d)
		} else {
			found = true
		}
	}

	if !found {
		return nil // Domain not found, no-op
	}

	// Step 4: PATCH with updated array
	url := fmt.Sprintf("%s/id/runlite-%s/match/0/host", c.BaseURL, appName)
	return c.doRequest("PATCH", url, filtered)
}

// GetAppDomains retrieves all domains for an app.
func (c *Client) GetAppDomains(appName string) ([]string, error) {
	path := fmt.Sprintf("id/runlite-%s/match/0/host", appName)
	domainsJSON, err := c.GetConfig(path)
	if err != nil {
		return nil, err
	}

	var domains []string
	if err := json.Unmarshal(domainsJSON, &domains); err != nil {
		return nil, fmt.Errorf("parse domains: %w", err)
	}

	return domains, nil
}

// ConfigureOnDemandTLS sets up global on-demand TLS with validation endpoint.
func (c *Client) ConfigureOnDemandTLS(askEndpoint string) error {
	config := map[string]any{
		"apps": map[string]any{
			"tls": map[string]any{
				"automation": map[string]any{
					"on_demand": map[string]any{
						"ask": askEndpoint,
						"rate_limit": map[string]any{
							"interval": "2m",
							"burst":    5,
						},
					},
					"policies": []map[string]any{
						{"on_demand": true},
					},
				},
			},
		},
	}

	return c.doRequest("POST", c.BaseURL+"/load", config)
}

// parseError parses an error response from Caddy API.
func (c *Client) parseError(statusCode int, body []byte) error {
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("caddy api error (status %d): %s", statusCode, string(body))
	}
	return fmt.Errorf("caddy api error (status %d): %s", statusCode, errResp.Error)
}
