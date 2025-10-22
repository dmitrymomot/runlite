package deploy

import (
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/dmitrymomot/runlite/internal/app"
	"github.com/dmitrymomot/runlite/internal/config"
)

const (
	// Port range for application deployments
	MinPort = 8000
	MaxPort = 9000

	// Maximum attempts to find a free port
	maxPortAttempts = 100
)

// CheckPortFree tests if a port is available by attempting to bind to it.
// Returns true if the port is free, false if it's in use or there's an error.
func CheckPortFree(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false // Port in use or error
	}
	defer listener.Close()
	return true
}

// AllocateFreePort finds an available port in the valid range (8000-9000).
// It checks both system port availability and existing deployment ports to avoid conflicts.
// Returns an error if no free port is found after maxPortAttempts tries.
func AllocateFreePort(appName string) (int, error) {
	// Get ports already used by this app's deployments
	usedPorts := make(map[int]bool)
	mgr := app.NewManager(config.GetDataDir())
	deployments, err := mgr.ListDeployments(appName)
	if err == nil {
		for _, d := range deployments {
			usedPorts[d.Port] = true
		}
	}

	portRange := MaxPort - MinPort + 1

	for range maxPortAttempts {
		port := rand.IntN(portRange) + MinPort

		// Skip if port is already used by an existing deployment
		if usedPorts[port] {
			continue
		}

		// Check if port is actually free on the system
		if CheckPortFree(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free ports available in range %d-%d after %d attempts", MinPort, MaxPort, maxPortAttempts)
}
