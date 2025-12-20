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

func ExecutePTY(command string) Result {
	return ExecutePTYWithTimeout(command, 60*time.Second)
}

func ExecutePTYWithTimeout(command string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecutePTYWithContext(ctx, command)
}

func ExecutePTYWithContext(ctx context.Context, command string) Result {
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

	return Result{
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
