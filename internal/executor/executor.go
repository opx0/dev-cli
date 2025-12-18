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

type Result struct {
	Command   string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Timestamp time.Time
	Shell     string
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

func Execute(command string) Result {
	return ExecuteWithTimeout(command, 60*time.Second)
}

func ExecuteWithTimeout(command string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecuteWithContext(ctx, command)
}

func ExecuteWithContext(ctx context.Context, command string) Result {
	start := time.Now()
	shell := getShell()

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

	cwd, _ := os.Getwd()
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

	return Result{
		Command:   command,
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     shell,
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
