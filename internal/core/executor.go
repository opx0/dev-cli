package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

type ExecResult struct {
	Command   string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Timestamp time.Time
	Shell     string
	Cwd       string
}

type EventPublisher interface {
	PublishCommandEvent(command string, exitCode int, duration time.Duration, output string)
}

var globalDB *sql.DB

func SetDatabase(db *sql.DB) {
	globalDB = db
}

func ExecuteAndLog(command string, publisher EventPublisher) ExecResult {
	return ExecuteAndLogWithTimeout(command, 60*time.Second, publisher)
}

func ExecuteAndLogWithTimeout(command string, timeout time.Duration, publisher EventPublisher) ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := ExecuteWithContext(ctx, command)

	if globalDB != nil {
		logEntry := LogEntry{
			Command:    result.Command,
			ExitCode:   result.ExitCode,
			Output:     TruncateOutput(result.Output, 10240),
			Cwd:        result.Cwd,
			DurationMs: result.Duration.Milliseconds(),
			Timestamp:  result.Timestamp.Format(time.RFC3339),
		}
		if err := SaveCommand(globalDB, logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log command: %v\n", err)
		}
	}

	if publisher != nil {
		publisher.PublishCommandEvent(result.Command, result.ExitCode, result.Duration, result.Output)
	}

	return result
}

func getShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	for _, shell := range []string{"/bin/zsh", "/usr/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	return "/bin/sh"
}

func Execute(command string) ExecResult {
	return ExecuteWithTimeout(command, 60*time.Second)
}

func ExecuteWithTimeout(command string, timeout time.Duration) ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecuteWithContext(ctx, command)
}

func ExecuteWithContext(ctx context.Context, command string) ExecResult {
	start := time.Now()
	shell := getShell()
	cwd, _ := os.Getwd()

	var cmd *exec.Cmd
	var wrappedCmd string
	if strings.HasSuffix(shell, "zsh") {
		wrappedCmd = fmt.Sprintf("source ~/.zshrc 2>/dev/null; %s", command)
		cmd = exec.CommandContext(ctx, shell, "-c", wrappedCmd)
	} else if strings.HasSuffix(shell, "bash") {
		wrappedCmd = fmt.Sprintf("source ~/.bashrc 2>/dev/null; %s", command)
		cmd = exec.CommandContext(ctx, shell, "-c", wrappedCmd)
	} else {
		wrappedCmd = command
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Dir = cwd
	cmd.Env = os.Environ()

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

	output := stdout.String()
	stderrStr := stderr.String()

	stderrStr = filterShellNoise(stderrStr)

	if stderrStr != "" {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderrStr
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

	return ExecResult{
		Command:   command,
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     shell,
		Cwd:       cwd,
	}
}

func filterShellNoise(stderr string) string {
	lines := strings.Split(stderr, "\n")
	var filtered []string

	for _, line := range lines {
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

func ExecuteSimple(command string) ExecResult {
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

	return ExecResult{
		Command:   command,
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     "sh",
	}
}

func IsAIQuery(input string) bool {
	input = strings.TrimSpace(input)
	return strings.HasPrefix(input, "?") || strings.HasPrefix(input, "@")
}

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
		parts := strings.SplitN(strings.TrimPrefix(input, "@"), " ", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], ""
	}

	return "", input
}

func ExecutePTY(command string) ExecResult {
	return ExecutePTYWithTimeout(command, 60*time.Second)
}

func ExecutePTYWithTimeout(command string, timeout time.Duration) ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecutePTYWithContext(ctx, command)
}

func ExecutePTYWithContext(ctx context.Context, command string) ExecResult {
	start := time.Now()
	shell := getShell()

	var cmd *exec.Cmd
	if strings.HasSuffix(shell, "zsh") {
		cmd = exec.CommandContext(ctx, shell, "-i", "-c", command)
	} else if strings.HasSuffix(shell, "bash") {
		cmd = exec.CommandContext(ctx, shell, "-i", "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	cwd, _ := os.Getwd()
	cmd.Dir = cwd

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return ExecResult{
			Command:   command,
			Output:    "Failed to start PTY: " + err.Error(),
			ExitCode:  1,
			Duration:  time.Since(start),
			Timestamp: start,
			Shell:     shell,
		}
	}
	defer ptmx.Close()

	var output bytes.Buffer
	done := make(chan error, 1)

	go func() {
		io.Copy(&output, ptmx)
		done <- cmd.Wait()
	}()

	select {
	case err = <-done:
	case <-ctx.Done():
		cmd.Process.Kill()
		err = ctx.Err()
	}

	duration := time.Since(start)
	outputStr := cleanPTYOutput(output.String())

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else if err == context.DeadlineExceeded {
			exitCode = 124
			outputStr = "Command timed out"
		} else {
			exitCode = 1
			if outputStr == "" {
				outputStr = err.Error()
			}
		}
	}

	return ExecResult{
		Command:   command,
		Output:    outputStr,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     shell,
	}
}

func cleanPTYOutput(output string) string {
	output = stripANSI(output)

	lines := strings.Split(output, "\n")
	var cleaned []string

	for _, line := range lines {
		if len(cleaned) == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "compinit") ||
			strings.Contains(line, "zinit") ||
			strings.Contains(line, "Loading") ||
			strings.HasPrefix(strings.TrimSpace(line), "❯") ||
			strings.HasPrefix(strings.TrimSpace(line), "$") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	result := strings.Join(cleaned, "\n")
	return strings.TrimSpace(result)
}

func stripANSI(str string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(str); i++ {
		if str[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (str[i] >= 'a' && str[i] <= 'z') || (str[i] >= 'A' && str[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		if str[i] < 32 && str[i] != '\n' && str[i] != '\t' && str[i] != '\r' {
			continue
		}
		result.WriteByte(str[i])
	}

	return result.String()
}
