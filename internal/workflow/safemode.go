package workflow

import (
	"fmt"
	"strings"
)

// SafeMode controls whether remediation actions are executed or just previewed.
type SafeMode int

const (
	// SafeModePreview is the default: shows what would happen without executing
	SafeModePreview SafeMode = iota
	// SafeModeExecute actually runs remediation commands
	SafeModeExecute
)

// String returns a human-readable representation of the SafeMode.
func (m SafeMode) String() string {
	switch m {
	case SafeModePreview:
		return "preview"
	case SafeModeExecute:
		return "execute"
	default:
		return "unknown"
	}
}

// SafeModeContext wraps execution with preview/approval logic.
// It provides governance controls for automated remediation.
type SafeModeContext struct {
	// Mode controls preview vs execute behavior
	Mode SafeMode

	// ApprovalFunc is called to prompt for user confirmation
	// Returns true if approved, false if denied
	ApprovalFunc func(action string) bool

	// RollbackEnabled indicates whether rollback hooks should be registered
	RollbackEnabled bool

	// DryRunOutput collects preview actions when in SafeModePreview
	DryRunOutput []PreviewAction

	// DestructivePatterns are command patterns that require extra confirmation
	DestructivePatterns []string
}

// PreviewAction represents an action that would be taken in execute mode.
type PreviewAction struct {
	Description string
	Command     string
	Destructive bool
	StepID      string
}

// DefaultDestructivePatterns returns common dangerous command patterns.
func DefaultDestructivePatterns() []string {
	return []string{
		"rm -rf",
		"rm -r /",
		"dd if=",
		"mkfs",
		"> /dev/",
		"chmod 777",
		":(){ :|:& };:",
		"drop database",
		"drop table",
		"truncate table",
		"delete from",
		"git reset --hard",
		"git clean -fdx",
		"docker system prune",
	}
}

// NewSafeModeContext creates a preview-only context by default.
func NewSafeModeContext() *SafeModeContext {
	return &SafeModeContext{
		Mode:                SafeModePreview,
		RollbackEnabled:     true,
		DryRunOutput:        make([]PreviewAction, 0),
		DestructivePatterns: DefaultDestructivePatterns(),
	}
}

// NewExecuteContext creates a context that will actually execute commands.
func NewExecuteContext(approvalFunc func(string) bool) *SafeModeContext {
	return &SafeModeContext{
		Mode:                SafeModeExecute,
		ApprovalFunc:        approvalFunc,
		RollbackEnabled:     true,
		DryRunOutput:        make([]PreviewAction, 0),
		DestructivePatterns: DefaultDestructivePatterns(),
	}
}

// IsPreview returns true if in preview mode.
func (c *SafeModeContext) IsPreview() bool {
	return c.Mode == SafeModePreview
}

// PreviewAction records an action without executing (in preview mode).
func (c *SafeModeContext) PreviewAction(stepID, description, command string) {
	destructive := c.isDestructive(command)
	c.DryRunOutput = append(c.DryRunOutput, PreviewAction{
		Description: description,
		Command:     command,
		Destructive: destructive,
		StepID:      stepID,
	})
}

// RequireApproval prompts for confirmation before destructive operations.
// Returns true if approved or no approval function is set.
func (c *SafeModeContext) RequireApproval(action string) bool {
	if c.ApprovalFunc == nil {
		return true
	}
	return c.ApprovalFunc(action)
}

// RequireApprovalForDestructive checks if the command is destructive and requires approval.
// Returns true if approved (or not destructive), false if denied.
func (c *SafeModeContext) RequireApprovalForDestructive(command string) bool {
	if !c.isDestructive(command) {
		return true
	}
	return c.RequireApproval(fmt.Sprintf("⚠️  Potentially destructive command:\n  %s\n\nProceed?", command))
}

// isDestructive checks if a command matches any destructive pattern.
func (c *SafeModeContext) isDestructive(command string) bool {
	lower := strings.ToLower(command)
	for _, pattern := range c.DestructivePatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// GetPreviewSummary returns a formatted summary of all preview actions.
func (c *SafeModeContext) GetPreviewSummary() string {
	if len(c.DryRunOutput) == 0 {
		return "No actions would be taken."
	}

	var sb strings.Builder
	sb.WriteString("Preview of actions that would be taken:\n\n")

	for i, action := range c.DryRunOutput {
		marker := "  "
		if action.Destructive {
			marker = "⚠️"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, marker, action.Description))
		if action.Command != "" {
			sb.WriteString(fmt.Sprintf("   $ %s\n", action.Command))
		}
		sb.WriteString("\n")
	}

	destructiveCount := 0
	for _, a := range c.DryRunOutput {
		if a.Destructive {
			destructiveCount++
		}
	}

	if destructiveCount > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  %d potentially destructive action(s) detected.\n", destructiveCount))
		sb.WriteString("Use --force to execute, or review commands carefully.\n")
	}

	return sb.String()
}

// ClearPreview resets the preview output.
func (c *SafeModeContext) ClearPreview() {
	c.DryRunOutput = make([]PreviewAction, 0)
}
