package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const MaxOutputSize = 10 * 1024 // 10KB limit

// LogEntry - single command record (fields ordered by priority)
type LogEntry struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output,omitempty"`
	Cwd        string `json:"cwd"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
}

// Logger - writes to log directory
type Logger struct {
	logPath string
}

// New - creates Logger
// Uses DEV_CLI_LOG_DIR env var if set, otherwise ~/.devlogs/history.jsonl
func New() (*Logger, error) {
	var logDir string

	// Check for dev mode override
	if envDir := os.Getenv("DEV_CLI_LOG_DIR"); envDir != "" {
		logDir = envDir
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		logDir = filepath.Join(home, ".devlogs")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &Logger{
		logPath: filepath.Join(logDir, "history.jsonl"),
	}, nil
}

// LogEvent - appends entry to log
func (l *Logger) LogEvent(command string, exitCode int, cwd string, durationMs int64, output string) error {
	// Truncate output if exceeds max size
	if len(output) > MaxOutputSize {
		output = output[len(output)-MaxOutputSize:]
	}

	entry := LogEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Command:    command,
		ExitCode:   exitCode,
		Cwd:        cwd,
		DurationMs: durationMs,
		Output:     output,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}
