package executor

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

// ExecutePTY runs a command in a real PTY with interactive shell
// This gives full access to aliases, functions, and shell features
func ExecutePTY(command string) Result {
	return ExecutePTYWithTimeout(command, 60*time.Second)
}

// ExecutePTYWithTimeout runs a command in PTY with timeout
func ExecutePTYWithTimeout(command string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecutePTYWithContext(ctx, command)
}

// ExecutePTYWithContext runs a command in a real PTY
// This spawns an interactive zsh that loads all config, aliases, etc.
func ExecutePTYWithContext(ctx context.Context, command string) Result {
	start := time.Now()
	shell := getShell()

	// Build command: start interactive shell, run command, exit
	// -i makes it interactive (loads .zshrc with aliases)
	// We use script wrapper to handle TTY issues
	var cmd *exec.Cmd
	if strings.HasSuffix(shell, "zsh") {
		// Use zsh -i -c for interactive command execution with aliases
		cmd = exec.CommandContext(ctx, shell, "-i", "-c", command)
	} else if strings.HasSuffix(shell, "bash") {
		cmd = exec.CommandContext(ctx, shell, "-i", "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	// Set working directory
	cwd, _ := os.Getwd()
	cmd.Dir = cwd

	// Set environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	// Start with PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return Result{
			Command:   command,
			Output:    "Failed to start PTY: " + err.Error(),
			ExitCode:  1,
			Duration:  time.Since(start),
			Timestamp: start,
			Shell:     shell,
		}
	}
	defer ptmx.Close()

	// Capture output
	var output bytes.Buffer
	done := make(chan error, 1)

	go func() {
		io.Copy(&output, ptmx)
		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case err = <-done:
		// Command completed
	case <-ctx.Done():
		// Timeout
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
			exitCode = 124 // timeout exit code
			outputStr = "Command timed out"
		} else {
			exitCode = 1
			if outputStr == "" {
				outputStr = err.Error()
			}
		}
	}

	return Result{
		Command:   command,
		Output:    outputStr,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: start,
		Shell:     shell,
	}
}

// cleanPTYOutput removes ANSI escape codes and shell prompts from PTY output
func cleanPTYOutput(output string) string {
	// Remove common ANSI escape sequences
	output = stripANSI(output)

	// Remove common shell startup noise
	lines := strings.Split(output, "\n")
	var cleaned []string

	for _, line := range lines {
		// Skip empty lines at start
		if len(cleaned) == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		// Skip shell prompts and zinit loading messages
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

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(str); i++ {
		if str[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// End of escape sequence
			if (str[i] >= 'a' && str[i] <= 'z') || (str[i] >= 'A' && str[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		// Skip other control characters except newline and tab
		if str[i] < 32 && str[i] != '\n' && str[i] != '\t' && str[i] != '\r' {
			continue
		}
		result.WriteByte(str[i])
	}

	return result.String()
}
