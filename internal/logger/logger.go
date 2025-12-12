package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// QueryOpts - options for querying log entries
type QueryOpts struct {
	Limit  int           // Max entries to return (0 = no limit)
	Filter string        // Command keyword filter
	Since  time.Duration // Time window (0 = no time filter)
}

// GetFailures - query failed commands from log with filters
func (l *Logger) GetFailures(opts QueryOpts) ([]LogEntry, error) {
	data, err := os.ReadFile(l.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No log file yet
		}
		return nil, fmt.Errorf("failed to read log: %w", err)
	}

	var entries []LogEntry
	lines := strings.Split(string(data), "\n")

	// Parse all entries (newest last in file)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed entries
		}

		// Only failures (exit_code != 0)
		if entry.ExitCode == 0 {
			continue
		}

		// Apply keyword filter
		if opts.Filter != "" && !strings.Contains(strings.ToLower(entry.Command), strings.ToLower(opts.Filter)) {
			continue
		}

		// Apply time filter
		if opts.Since > 0 {
			ts, err := time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				continue
			}
			if time.Since(ts) > opts.Since {
				continue
			}
		}

		entries = append(entries, entry)
	}

	// Reverse to get newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Apply limit
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}

	return entries, nil
}
