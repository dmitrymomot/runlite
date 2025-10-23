package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/dmitrymomot/runlite/cmd/runlite/internal/ui"
	"github.com/dmitrymomot/runlite/internal/config"
)

var (
	// Regular expressions for detecting log levels
	errorRegex   = regexp.MustCompile(`(?i)\b(error|err|fatal|panic|failed)\b`)
	warningRegex = regexp.MustCompile(`(?i)\b(warn|warning)\b`)
	infoRegex    = regexp.MustCompile(`(?i)\b(info|information)\b`)
	debugRegex   = regexp.MustCompile(`(?i)\b(debug|trace)\b`)
)

// logLevel represents the detected log level
type logLevel int

const (
	logLevelUnknown logLevel = iota
	logLevelDebug
	logLevelInfo
	logLevelWarning
	logLevelError
)

// printTailLines reads and prints the last N lines from a file
func printTailLines(file *os.File, n int, logger *log.Logger) error {
	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	for _, line := range lines {
		formatLogLine(line, logger)
	}

	return nil
}

// followLogs tails a log file in follow mode (like tail -f)
func followLogs(ctx context.Context, logPath string, logger *log.Logger) error {
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Seek to end of file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek to end: %w", err)
	}

	scanner := bufio.NewScanner(file)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			for scanner.Scan() {
				formatLogLine(scanner.Text(), logger)
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
		}
	}
}

// formatLogLine formats and prints a log line with appropriate styling
func formatLogLine(line string, logger *log.Logger) {
	if strings.TrimSpace(line) == "" {
		return
	}

	level := detectLogLevel(line)

	switch level {
	case logLevelError:
		logger.Error(line)
	case logLevelWarning:
		logger.Warn(line)
	case logLevelInfo:
		logger.Info(line)
	case logLevelDebug:
		logger.Debug(line)
	default:
		// Unknown level - just print with info style
		logger.Info(line)
	}
}

// detectLogLevel attempts to detect the log level from the line content
func detectLogLevel(line string) logLevel {
	switch {
	case errorRegex.MatchString(line):
		return logLevelError
	case warningRegex.MatchString(line):
		return logLevelWarning
	case infoRegex.MatchString(line):
		return logLevelInfo
	case debugRegex.MatchString(line):
		return logLevelDebug
	default:
		return logLevelUnknown
	}
}

// validateAppName validates the app name and returns a friendly error message
func validateAppName(appName string) error {
	if err := config.ValidateAppName(appName); err != nil {
		return fmt.Errorf("%s %s",
			ui.ErrorStyle.Render("Invalid app name:"),
			err.Error())
	}
	return nil
}

// createLogHeader prints a styled header for log output
func createLogHeader(logger *log.Logger, appName, releaseID, logType string) {
	logger.Info(ui.SubtitleStyle.Render(fmt.Sprintf("%s %s Logs - Release: %s",
		ui.IconRocket, appName, releaseID)))
	logger.Info(ui.MutedStyle.Render(fmt.Sprintf("Log type: %s", logType)))
	logger.Info(ui.MutedStyle.Render(strings.Repeat("─", 60)))
}
