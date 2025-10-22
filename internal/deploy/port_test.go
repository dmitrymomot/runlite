package deploy_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/internal/deploy"
)

func TestCheckPortFree(t *testing.T) {
	t.Run("returns true for available port", func(t *testing.T) {
		// Find a port that should be available
		port := 8123 // Arbitrary port in the deployment range
		free := deploy.CheckPortFree(port)
		assert.True(t, free, "Port %d should be available", port)
	})

	t.Run("returns false for port in use", func(t *testing.T) {
		// Start a listener on a port
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err, "Failed to start test listener")
		defer listener.Close()

		// Get the port number
		addr := listener.Addr().(*net.TCPAddr)
		port := addr.Port

		// Check if it's reported as in use
		free := deploy.CheckPortFree(port)
		assert.False(t, free, "Port %d should be reported as in use", port)
	})

	t.Run("returns false for invalid port numbers", func(t *testing.T) {
		// Test invalid ports
		testCases := []int{
			-1,    // Negative
			65536, // Out of range
			99999, // Out of range
		}

		for _, port := range testCases {
			free := deploy.CheckPortFree(port)
			assert.False(t, free, "Port %d should be reported as unavailable", port)
		}
	})

	t.Run("port becomes free after listener closes", func(t *testing.T) {
		// Start a listener
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		addr := listener.Addr().(*net.TCPAddr)
		port := addr.Port

		// Verify it's in use
		assert.False(t, deploy.CheckPortFree(port))

		// Close listener
		listener.Close()

		// Verify it's free (might need a small delay in some systems)
		assert.True(t, deploy.CheckPortFree(port))
	})
}

func TestAllocateFreePort(t *testing.T) {
	t.Run("allocates port in valid range", func(t *testing.T) {
		port, err := deploy.AllocateFreePort("test-app")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, deploy.MinPort, "Port should be >= MinPort")
		assert.LessOrEqual(t, port, deploy.MaxPort, "Port should be <= MaxPort")
	})

	t.Run("allocated port is actually free", func(t *testing.T) {
		port, err := deploy.AllocateFreePort("test-app")
		require.NoError(t, err)

		// Verify we can bind to the allocated port
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err, "Should be able to bind to allocated port")
		listener.Close()
	})

	t.Run("allocates different ports on subsequent calls", func(t *testing.T) {
		// Allocate multiple ports and verify they're different
		// Note: Due to randomness, we might occasionally get duplicates,
		// but with 1000 ports in range, probability is low for small samples
		ports := make(map[int]bool)
		iterations := 10

		for i := 0; i < iterations; i++ {
			port, err := deploy.AllocateFreePort("test-app")
			require.NoError(t, err)
			ports[port] = true
		}

		// We should have gotten at least some different ports
		// (with 1000 port range and 10 tries, getting all the same is extremely unlikely)
		assert.Greater(t, len(ports), 1, "Should allocate different ports")
	})

	t.Run("skips ports that are in use", func(t *testing.T) {
		// Start listeners on several ports in the range
		var listeners []net.Listener
		occupiedPorts := make(map[int]bool)

		// Occupy a few ports
		for i := 0; i < 5; i++ {
			port := deploy.MinPort + i*100 // Spread them out
			listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err == nil {
				listeners = append(listeners, listener)
				occupiedPorts[port] = true
			}
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		// Allocate a port - should not be one of the occupied ones
		port, err := deploy.AllocateFreePort("test-app")
		require.NoError(t, err)
		assert.NotContains(t, occupiedPorts, port, "Should not allocate an occupied port")
	})

	t.Run("handles high port occupation gracefully", func(t *testing.T) {
		// This test verifies the function can handle when many ports are occupied
		// We'll occupy a reasonable number and verify allocation still works
		var listeners []net.Listener
		occupiedCount := 0

		// Try to occupy up to 50 ports (should still leave plenty free)
		for i := deploy.MinPort; i < deploy.MinPort+50 && i <= deploy.MaxPort; i++ {
			listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", i))
			if err == nil {
				listeners = append(listeners, listener)
				occupiedCount++
			}
		}
		defer func() {
			for _, l := range listeners {
				l.Close()
			}
		}()

		t.Logf("Occupied %d ports", occupiedCount)

		// Should still be able to allocate
		port, err := deploy.AllocateFreePort("test-app")
		require.NoError(t, err)

		// Verify it's in range and free
		assert.GreaterOrEqual(t, port, deploy.MinPort)
		assert.LessOrEqual(t, port, deploy.MaxPort)
		assert.True(t, deploy.CheckPortFree(port))
	})

	t.Run("returns error when all ports exhausted", func(t *testing.T) {
		// This test is impractical to run (would need to occupy 1000+ ports)
		// Instead, we verify the error message format by checking the function
		// can theoretically return an error

		// We'll just verify the function signature and trust the implementation
		// In a real scenario where all ports are occupied, we'd expect:
		// _, err := deploy.AllocateFreePort("test-app")
		// require.Error(t, err)
		// assert.Contains(t, err.Error(), "no free ports available")

		t.Skip("Skipping exhaustive port occupation test - impractical to run")
	})

	t.Run("multiple concurrent allocations", func(t *testing.T) {
		// Verify that multiple goroutines can allocate ports concurrently
		iterations := 20
		results := make(chan int, iterations)
		errors := make(chan error, iterations)

		for i := 0; i < iterations; i++ {
			go func() {
				port, err := deploy.AllocateFreePort("test-app")
				if err != nil {
					errors <- err
					return
				}
				results <- port
			}()
		}

		// Collect results
		ports := make(map[int]bool)
		for i := 0; i < iterations; i++ {
			select {
			case port := <-results:
				ports[port] = true
			case err := <-errors:
				t.Fatalf("Allocation failed: %v", err)
			}
		}

		// All allocations should have succeeded and returned valid ports
		// Note: Due to random selection, some ports may be allocated multiple times
		// We just verify we got at least some diversity (not all the same port)
		assert.GreaterOrEqual(t, len(ports), 10, "Should have at least 10 different ports allocated")

		// Verify all ports are in valid range
		for port := range ports {
			assert.GreaterOrEqual(t, port, deploy.MinPort)
			assert.LessOrEqual(t, port, deploy.MaxPort)
		}
	})
}

func TestPortConstants(t *testing.T) {
	t.Run("port range constants are valid", func(t *testing.T) {
		assert.Greater(t, deploy.MinPort, 0, "MinPort should be positive")
		assert.Less(t, deploy.MinPort, 65536, "MinPort should be valid port number")
		assert.Greater(t, deploy.MaxPort, deploy.MinPort, "MaxPort should be greater than MinPort")
		assert.Less(t, deploy.MaxPort, 65536, "MaxPort should be valid port number")
	})

	t.Run("port range provides reasonable capacity", func(t *testing.T) {
		capacity := deploy.MaxPort - deploy.MinPort + 1
		assert.GreaterOrEqual(t, capacity, 100, "Should have at least 100 ports available")
	})
}
