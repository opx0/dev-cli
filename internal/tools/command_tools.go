package tools

import (
	"context"
	"fmt"
	"time"

	"dev-cli/internal/executor"
)

// RunCommandTool executes shell commands with timeout.
type RunCommandTool struct{}

func (t *RunCommandTool) Name() string { return "run_command" }
func (t *RunCommandTool) Description() string {
	return "Execute shell command with timeout and capture output"
}

func (t *RunCommandTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "command", Type: "string", Description: "Command to execute", Required: true},
		{Name: "timeout", Type: "duration", Description: "Timeout (e.g., '30s', '5m')", Required: false, Default: "60s"},
		{Name: "cwd", Type: "string", Description: "Working directory", Required: false},
	}
}

// CommandResult contains the command execution output.
type CommandResult struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
	Cwd      string `json:"cwd,omitempty"`
	Blocked  bool   `json:"blocked,omitempty"`
}

func (t *RunCommandTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	command := GetString(params, "command", "")
	if command == "" {
		return NewErrorResult("command is required", time.Since(start))
	}

	// Safety check: block dangerous commands
	if check := executor.CheckCommandSafety(command); !check.IsSafe {
		return NewErrorResult(fmt.Sprintf("BLOCKED: %s (matched pattern: '%s', severity: %s). This command could cause data loss or system damage.",
			check.Reason, check.MatchedRule, check.Severity), time.Since(start))
	}

	timeout := GetDuration(params, "timeout", 60*time.Second)

	result := executor.ExecuteWithTimeout(command, timeout)

	return NewResult(CommandResult{
		Command:  result.Command,
		Output:   result.Output,
		ExitCode: result.ExitCode,
		Duration: result.Duration.String(),
		Cwd:      result.Cwd,
	}, time.Since(start))
}
