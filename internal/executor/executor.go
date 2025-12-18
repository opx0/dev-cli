package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Result holds the output of a command execution
type Result struct {
	Command   string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Timestamp time.Time
	Shell     string
}

// getShell returns the user's preferred shell
func getShell() string {
	// Check SHELL env var
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	// Default to zsh, fallback to bash, then sh
	for _, shell := range []string{"/bin/zsh", "/usr/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	return "/bin/sh"
}

// Execute runs a shell command using the user's configured shell (with aliases, functions, etc.)
func Execute(command string) Result {
	return ExecuteWithTimeout(command, 60*time.Second)
}

// ExecuteWithTimeout runs a shell command with a timeout
func ExecuteWithTimeout(command string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecuteWithContext(ctx, command)
}

// ExecuteWithContext runs a shell command with context for cancellation
// Sources user config explicitly without interactive mode to avoid TTY conflicts
func ExecuteWithContext(ctx context.Context, command string) Result {
	start := time.Now()
	shell := getShell()

	// Don't use -i (interactive) as it conflicts with bubbletea's terminal control
	// Instead, source the config file explicitly for aliases
	var cmd *exec.Cmd
	var wrappedCmd string
	if strings.HasSuffix(shell, "zsh") {
		// Source .zshrc for aliases, then run command
		wrappedCmd = fmt.Sprintf("source ~/.zshrc 2>/dev/null; %s", command)
		cmd = exec.CommandContext(ctx, shell, "-c", wrappedCmd)
	} else if strings.HasSuffix(shell, "bash") {
		wrappedCmd = fmt.Sprintf("source ~/.bashrc 2>/dev/null; %s", command)
		cmd = exec.CommandContext(ctx, shell, "-c", wrappedCmd)
	} else {
		wrappedCmd = command
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set working directory to current directory
	cwd, _ := os.Getwd()
	cmd.Dir = cwd

	// Inherit environment - this preserves PATH, HOME, etc.
	cmd.Env = os.Environ()

	// Add TERM if not set (some programs need it)
	hasTermEnv := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "TERM=") {
			hasTermEnv = true
			break
		}
	}
	if !hasTermEnv {
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	}

	err := cmd.Run()

	duration := time.Since(start)

	// Combine stdout and stderr
	output := stdout.String()
	stderrStr := stderr.String()

	// Filter out common zsh startup noise
	stderrStr = filterShellNoise(stderrStr)

	if stderrStr != "" {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderrStr
	}

	// Clean up trailing newline
	output = strings.TrimSuffix(output, "\n")

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
			if output == "" {
				output = err.Error()
			}
		}
	}

	return Result{
		Command:   command,
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     shell,
	}
}

// filterShellNoise removes common shell startup messages that aren't useful
func filterShellNoise(stderr string) string {
	lines := strings.Split(stderr, "\n")
	var filtered []string

	for _, line := range lines {
		// Skip common zinit/zsh startup messages
		if strings.Contains(line, "compinit") ||
			strings.Contains(line, "compdef") ||
			strings.Contains(line, "zinit") ||
			strings.Contains(line, "Loading") ||
			strings.Contains(line, "Loaded") ||
			strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// ExecuteSimple runs a command without loading user config (faster, for internal use)
func ExecuteSimple(command string) Result {
	start := time.Now()

	cmd := exec.Command("sh", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cwd, _ := os.Getwd()
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	err := cmd.Run()

	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	output = strings.TrimSuffix(output, "\n")

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
			if output == "" {
				output = err.Error()
			}
		}
	}

	return Result{
		Command:   command,
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     "sh",
	}
}

// IsAIQuery checks if input is an AI query (starts with ? or @)
func IsAIQuery(input string) bool {
	input = strings.TrimSpace(input)
	return strings.HasPrefix(input, "?") || strings.HasPrefix(input, "@")
}

// ParseAIQuery extracts the query from AI input
func ParseAIQuery(input string) (queryType string, query string) {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "?") {
		return "question", strings.TrimPrefix(input, "?")
	}

	if strings.HasPrefix(input, "@fix") {
		return "fix", strings.TrimSpace(strings.TrimPrefix(input, "@fix"))
	}

	if strings.HasPrefix(input, "@explain") {
		return "explain", strings.TrimSpace(strings.TrimPrefix(input, "@explain"))
	}

	if strings.HasPrefix(input, "@") {
		// Generic @ command
		parts := strings.SplitN(strings.TrimPrefix(input, "@"), " ", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], ""
	}

	return "", input
}
