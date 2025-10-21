package caddy

import "fmt"

// Route represents a Caddy route configuration.
type Route struct {
	ID     string      `json:"@id,omitempty"`
	Match  []MatchHost `json:"match"`
	Handle []Handler   `json:"handle"`
}

// MatchHost represents a host matcher for routing.
type MatchHost struct {
	Host []string `json:"host"`
}

// Handler represents a route handler.
type Handler struct {
	Handler   string     `json:"handler"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
}

// Upstream represents a reverse proxy upstream.
type Upstream struct {
	Dial string `json:"dial"`
}

// ErrorResponse represents an error response from Caddy API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewAppRoute creates a new route configuration for an app with a single domain.
func NewAppRoute(appName, domain string, upstreamPort int) *Route {
	return NewAppRouteMulti(appName, []string{domain}, upstreamPort)
}

// NewAppRouteMulti creates a new route configuration for an app with multiple domains.
func NewAppRouteMulti(appName string, domains []string, upstreamPort int) *Route {
	return &Route{
		ID: "runlite-" + appName,
		Match: []MatchHost{
			{
				Host: domains,
			},
		},
		Handle: []Handler{
			{
				Handler: "reverse_proxy",
				Upstreams: []Upstream{
					{
						Dial: formatUpstream(upstreamPort),
					},
				},
			},
		},
	}
}

// formatUpstream formats the upstream address for localhost.
func formatUpstream(port int) string {
	return fmt.Sprintf("localhost:%d", port)
}
