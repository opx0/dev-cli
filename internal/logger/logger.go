package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxOutputSize = 10 * 1024

type LogEntry struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output,omitempty"`
	Cwd        string `json:"cwd"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
}

type Logger struct {
	logPath string
}

func New() (*Logger, error) {
	var logDir string

	if envDir := os.Getenv("DEV_CLI_LOG_DIR"); envDir != "" {
		logDir = envDir
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		logDir = filepath.Join(home, ".devlogs")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	return &Logger{logPath: filepath.Join(logDir, "history.jsonl")}, nil
}

func (l *Logger) LogEvent(command string, exitCode int, cwd string, durationMs int64, output string) error {
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
		return fmt.Errorf("marshal entry: %w", err)
	}

	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write log: %w", err)
	}

	return nil
}

type QueryOpts struct {
	Limit  int
	Filter string
	Since  time.Duration
}

func (l *Logger) GetFailures(opts QueryOpts) ([]LogEntry, error) {
	data, err := os.ReadFile(l.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read log: %w", err)
	}

	var entries []LogEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.ExitCode == 0 {
			continue
		}

		if opts.Filter != "" && !strings.Contains(strings.ToLower(entry.Command), strings.ToLower(opts.Filter)) {
			continue
		}

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

	// Reverse (newest first)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}

	return entries, nil
}
