package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

// CreateSystemUser creates a system user if it doesn't exist.
// System users are created with --system flag and no home directory.
func CreateSystemUser(ctx context.Context, username string) error {
	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		// User exists, skip creation
		return nil
	}

	// Create system user
	cmd := exec.CommandContext(ctx, "useradd",
		"--system",              // System user
		"--no-create-home",      // No home directory
		"--shell", "/bin/false", // No login shell
		username,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create user %s: %w\n%s", username, err, output)
	}

	return nil
}

// CreateDirectory creates a single directory with specified permissions.
func CreateDirectory(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	return nil
}

// SetOwner sets the owner and group of a path.
// username can be "user" or "user:group" format.
func SetOwner(ctx context.Context, path, username string) error {
	cmd := exec.CommandContext(ctx, "chown", "-R", username, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown %s %s: %w\n%s", username, path, err, output)
	}
	return nil
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
