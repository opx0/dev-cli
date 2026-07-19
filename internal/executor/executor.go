package executor

import (
	"bytes"
	"context"
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
	Cwd       string
}

const maxCapturedOutput = 1024 * 1024

type cappedBuffer struct {
	buf       bytes.Buffer
	remaining int
	truncated bool
}

func newCappedBuffer() *cappedBuffer {
	return &cappedBuffer{remaining: maxCapturedOutput}
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	written := len(p)
	if len(p) > b.remaining {
		p = p[:b.remaining]
		b.truncated = true
	}
	if len(p) > 0 {
		_, _ = b.buf.Write(p)
		b.remaining -= len(p)
	}
	return written, nil
}

func (b *cappedBuffer) String() string {
	if b.truncated {
		return b.buf.String() + "\n... output truncated"
	}
	return b.buf.String()
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

func ExecuteWithContextInDir(ctx context.Context, command, cwd string) Result {
	start := time.Now()
	shell := getShell()
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// #nosec G204 -- this is the single shell boundary; callers apply command safety and approval.
	cmd := exec.CommandContext(ctx, shell, "-c", command)

	stdout, stderr := newCappedBuffer(), newCappedBuffer()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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

func ExecuteProgram(ctx context.Context, cwd, name string, args ...string) Result {
	start := time.Now()
	// #nosec G204 -- executable and arguments stay separate; internal callers choose the program.
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, stderr := newCappedBuffer(), newCappedBuffer()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	err := cmd.Run()

	duration := time.Since(start)

	output := stdout.String()
	if stderr.buf.Len() > 0 {
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
		Command:   strings.Join(append([]string{name}, args...), " "),
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     name,
		Cwd:       cwd,
	}
}
