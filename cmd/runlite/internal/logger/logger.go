package logger

import (
	"io"
	"os"

	"github.com/charmbracelet/log"
)

// Config holds logger configuration options
type Config struct {
	// Level is the minimum log level to display (default: InfoLevel)
	Level log.Level

	// ReportTimestamp controls whether to show timestamps in logs (default: false for CLI)
	ReportTimestamp bool

	// ReportCaller controls whether to show caller information (default: false)
	ReportCaller bool

	// Output is where logs should be written (default: os.Stderr)
	Output io.Writer

	// Prefix is an optional prefix for all log messages
	Prefix string
}

// DefaultConfig returns the default logger configuration for CLI usage
func DefaultConfig() Config {
	return Config{
		Level:           log.InfoLevel,
		ReportTimestamp: false, // Clean output for CLI
		ReportCaller:    false,
		Output:          os.Stderr,
		Prefix:          "",
	}
}

// New creates a new charm logger with the given configuration
func New(cfg Config) *log.Logger {
	// Set defaults if not provided
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	logger := log.NewWithOptions(cfg.Output, log.Options{
		Level:           cfg.Level,
		ReportTimestamp: cfg.ReportTimestamp,
		ReportCaller:    cfg.ReportCaller,
		Prefix:          cfg.Prefix,
	})

	return logger
}

// NewDefault creates a new charm logger with default CLI configuration
func NewDefault() *log.Logger {
	return New(DefaultConfig())
}

// NewDebug creates a new charm logger configured for debug mode
func NewDebug() *log.Logger {
	cfg := DefaultConfig()
	cfg.Level = log.DebugLevel
	cfg.ReportCaller = true
	return New(cfg)
}

// NewSilent creates a new charm logger that only shows warnings and errors
func NewSilent() *log.Logger {
	cfg := DefaultConfig()
	cfg.Level = log.WarnLevel
	return New(cfg)
}

// NewWithPrefix creates a new charm logger with the specified prefix
func NewWithPrefix(prefix string) *log.Logger {
	cfg := DefaultConfig()
	cfg.Prefix = prefix
	return New(cfg)
}
