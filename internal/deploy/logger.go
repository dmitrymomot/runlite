package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DeployLogger struct {
	logFile   *os.File
	appName   string
	releaseID string
}

type logEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message"`
	App     string `json:"app"`
	Release string `json:"release"`
	Error   string `json:"error,omitempty"`
}

func NewDeployLogger(appName, releaseID string) (*DeployLogger, error) {
	releaseDir := GetReleaseDir(appName, releaseID)
	logPath := filepath.Join(releaseDir, "deploy.log")

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open deploy log: %w", err)
	}

	return &DeployLogger{
		logFile:   file,
		appName:   appName,
		releaseID: releaseID,
	}, nil
}

func (l *DeployLogger) Info(phase, message string, args ...any) {
	formattedMsg := fmt.Sprintf(message, args...)

	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   "info",
		Phase:   phase,
		Message: formattedMsg,
		App:     l.appName,
		Release: l.releaseID,
	}

	l.writeJSON(entry)
	fmt.Printf("→ %s\n", formattedMsg)
}

func (l *DeployLogger) Error(phase, message string, err error) {
	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   "error",
		Phase:   phase,
		Message: message,
		App:     l.appName,
		Release: l.releaseID,
		Error:   err.Error(),
	}

	l.writeJSON(entry)
	fmt.Printf("❌ %s: %v\n", message, err)
}

func (l *DeployLogger) Success(message string) {
	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   "success",
		Message: message,
		App:     l.appName,
		Release: l.releaseID,
	}

	l.writeJSON(entry)
	fmt.Printf("✅ %s\n", message)
}

func (l *DeployLogger) writeJSON(entry logEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = l.logFile.Write(data)
	_, _ = l.logFile.WriteString("\n")
}

func (l *DeployLogger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
