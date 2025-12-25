// Package opencode provides integration with the OpenCode AI coding agent.
// It enables dev-cli to delegate debugging tasks to OpenCode with full context.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"dev-cli/internal/storage"
)

// Adapter wraps OpenCode CLI for programmatic invocation.
type Adapter struct {
	opencodeBin string
}

// NewAdapter creates a new OpenCode adapter.
func NewAdapter() *Adapter {
	bin := "opencode"
	if envBin := os.Getenv("OPENCODE_BIN"); envBin != "" {
		bin = envBin
	}
	return &Adapter{opencodeBin: bin}
}

// IsAvailable checks if OpenCode is installed and accessible.
func (a *Adapter) IsAvailable() bool {
	_, err := exec.LookPath(a.opencodeBin)
	return err == nil
}

// RunOptions configures how OpenCode is invoked.
type RunOptions struct {
	Model    string // e.g., "anthropic/claude-sonnet"
	Agent    string // e.g., "build", "plan"
	NonBlock bool   // Run in background
}

// RunPrompt executes OpenCode in non-interactive mode with a prompt.
func (a *Adapter) RunPrompt(prompt string, opts RunOptions) error {
	// Validate the binary exists before executing
	binPath, err := exec.LookPath(a.opencodeBin)
	if err != nil {
		return fmt.Errorf("opencode binary not found: %w", err)
	}

	args := []string{"-p", prompt}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Agent != "" {
		args = append(args, "--agent", opts.Agent)
	}

	// #nosec G204 - binPath is validated via LookPath
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// HandoffToTUI replaces the current process with OpenCode TUI.
// This is useful for "agent" mode where user wants full OpenCode experience.
func (a *Adapter) HandoffToTUI() error {
	path, err := exec.LookPath(a.opencodeBin)
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}

	// Replace current process with OpenCode
	return syscall.Exec(path, []string{a.opencodeBin}, os.Environ())
}

// DebugContext represents aggregated debugging information.
type DebugContext struct {
	Issue           string                `json:"issue"`
	RecentHistory   []storage.HistoryItem `json:"recent_history,omitempty"`
	SimilarFailures []storage.HistoryItem `json:"similar_failures,omitempty"`
	KnownSolutions  []storage.Solution    `json:"known_solutions,omitempty"`
	ErrorSignature  string                `json:"error_signature,omitempty"`
}

// ToPrompt converts context to a well-formatted prompt for OpenCode.
func (ctx *DebugContext) ToPrompt() string {
	var sb strings.Builder

	sb.WriteString("# Debugging Context\n\n")
	sb.WriteString("## Issue\n")
	sb.WriteString(ctx.Issue)
	sb.WriteString("\n\n")

	if len(ctx.KnownSolutions) > 0 {
		sb.WriteString("## Previously Successful Solutions\n")
		for _, sol := range ctx.KnownSolutions {
			sb.WriteString(fmt.Sprintf("- `%s` (success rate: %d/%d)\n",
				sol.SolutionCommand, sol.SuccessCount, sol.SuccessCount+sol.FailureCount))
		}
		sb.WriteString("\n")
	}

	if len(ctx.SimilarFailures) > 0 {
		sb.WriteString("## Similar Past Failures\n")
		for _, item := range ctx.SimilarFailures[:min(3, len(ctx.SimilarFailures))] {
			sb.WriteString(fmt.Sprintf("- `%s` (exit %d)\n", item.Command, item.ExitCode))
		}
		sb.WriteString("\n")
	}

	if len(ctx.RecentHistory) > 0 {
		sb.WriteString("## Recent Commands\n")
		for _, item := range ctx.RecentHistory[:min(5, len(ctx.RecentHistory))] {
			status := "✓"
			if item.ExitCode != 0 {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("- %s `%s`\n", status, item.Command))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please analyze this issue and suggest a fix. ")
	sb.WriteString("Use the dev-cli MCP tools to query command history if needed.\n")

	return sb.String()
}

// ToJSON serializes context to JSON for debugging or MCP tools.
func (ctx *DebugContext) ToJSON() string {
	data, _ := json.MarshalIndent(ctx, "", "  ")
	return string(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
