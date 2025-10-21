package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultRegistryURL = "http://127.0.0.1:3000"

type client struct {
	baseURL    string
	httpClient *http.Client
}

type domainInfo struct {
	Name     string `json:"name"`
	AppName  string `json:"app_name"`
	Verified bool   `json:"verified"`
}

type addDomainRequest struct {
	Domain string `json:"domain"`
}

type removeDomainRequest struct {
	Domain string `json:"domain"`
}

type verifyDomainRequest struct {
	Domain string `json:"domain"`
}

type verifyDomainResponse struct {
	Verified bool `json:"verified"`
}

func newClient() *client {
	baseURL := os.Getenv("RUNLITE_REGISTRY_URL")
	if baseURL == "" {
		baseURL = defaultRegistryURL
	}

	return &client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *client) addDomain(ctx context.Context, domain string) error {
	payload := addDomainRequest{Domain: domain}
	return c.doRequest(ctx, "POST", "/api/domains", payload, nil)
}

func (c *client) removeDomain(ctx context.Context, domain string) error {
	payload := removeDomainRequest{Domain: domain}
	return c.doRequest(ctx, "DELETE", "/api/domains", payload, nil)
}

func (c *client) listDomains(ctx context.Context) ([]domainInfo, error) {
	var domains []domainInfo
	if err := c.doRequest(ctx, "GET", "/api/domains", nil, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (c *client) verifyDomain(ctx context.Context, domain string) (bool, error) {
	payload := verifyDomainRequest{Domain: domain}
	var resp verifyDomainResponse
	if err := c.doRequest(ctx, "POST", "/api/domains/verify", payload, &resp); err != nil {
		return false, err
	}
	return resp.Verified, nil
}

func (c *client) doRequest(ctx context.Context, method, path string, payload, result any) error {
	url := c.baseURL + path

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
