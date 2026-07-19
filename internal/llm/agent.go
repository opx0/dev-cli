// Package llm provides AI-powered agents for automated problem solving.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dev-cli/internal/tools"

	"github.com/briandowns/spinner"
)

// ── Agent Configuration ──────────────────────────────────────────────────────

const (
	// DefaultMaxIterations limits the agent loop to prevent runaway execution.
	DefaultMaxIterations = 10
)

// AgentConfig configures the agent behavior.
type AgentConfig struct {
	// MaxIterations limits the number of tool-calling rounds.
	MaxIterations int

	// Model specifies which LLM model to use.
	Model string

	// Verbose enables detailed logging.
	Verbose bool

	// DryRun enables preview mode (no actual execution).
	DryRun bool

	// ApprovalFunc is called before executing destructive tools.
	// Returns true to proceed, false to skip.
	ApprovalFunc func(toolName string, params map[string]any) bool
}

// DefaultAgentConfig returns sensible defaults.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxIterations: DefaultMaxIterations,
		Model:         DefaultModel,
		Verbose:       false,
	}
}

// ── Agent ────────────────────────────────────────────────────────────────────

// Agent is an AI-powered problem solver that uses structured tool calling.
// It maintains a conversation with an LLM provider, executing tools as requested
// and feeding results back until the task is complete.
type Agent struct {
	provider Provider
	registry *tools.Registry
	config   AgentConfig
}

// NewAgentWithDeps creates an agent with explicit dependencies (for testing).
func NewAgentWithDeps(provider Provider, registry *tools.Registry) *Agent {
	return &Agent{
		provider: provider,
		registry: registry,
		config:   DefaultAgentConfig(),
	}
}

// SetConfig updates the agent configuration.
func (a *Agent) SetConfig(cfg AgentConfig) {
	a.config = cfg
}

// ── Execution ────────────────────────────────────────────────────────────────

// AgentResult contains the outcome of an agent run.
type AgentResult struct {
	Success    bool          // Whether the task was completed successfully
	Summary    string        // Final summary from the agent
	Iterations int           // Number of tool-calling rounds
	ToolCalls  []ToolCallLog // Log of all tool calls made
	Error      string        // Error message if failed
}

// ToolCallLog records a single tool invocation.
type ToolCallLog struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
	Success    bool           `json:"success"`
	Output     string         `json:"output"`
	Error      string         `json:"error,omitempty"`
	Duration   time.Duration  `json:"duration"`
}

// Resolve attempts to solve the given issue using tool calls.
// This is the main entry point for automated problem resolution.
func (a *Agent) Resolve(ctx context.Context, issue string) (*AgentResult, error) {
	return a.Run(ctx, issue, a.buildSystemPrompt())
}

// Run executes the agent loop with a custom system prompt.
func (a *Agent) Run(ctx context.Context, task string, systemPrompt string) (*AgentResult, error) {
	result := &AgentResult{
		ToolCalls: make([]ToolCallLog, 0),
	}

	// Build tool definitions for the LLM
	toolDefs := a.buildToolDefs()

	// Initialize conversation
	messages := []Message{
		SystemMsg(systemPrompt),
		UserMsg(task),
	}

	// Show spinner for initial analysis
	spin := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spin.Suffix = " Analyzing..."
	spin.Start()

	// Agent loop
	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		result.Iterations = iteration + 1

		// Make LLM request
		resp, err := a.provider.ChatCompletion(ctx, ChatRequest{
			Model:    a.config.Model,
			Messages: messages,
			Tools:    toolDefs,
		})

		spin.Stop()

		if err != nil {
			result.Error = fmt.Sprintf("LLM error: %v", err)
			return result, err
		}

		// Check if we're done (no tool calls)
		if len(resp.ToolCalls) == 0 {
			result.Success = true
			result.Summary = resp.Content
			a.log("✓ Task completed")
			return result, nil
		}

		// Process tool calls
		a.log("→ Agent requested %d tool call(s)", len(resp.ToolCalls))

		// Add assistant message with tool calls to conversation
		messages = append(messages, AssistantMsg(resp.Content, resp.ToolCalls...))

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			toolLog := a.executeToolCall(ctx, tc)
			result.ToolCalls = append(result.ToolCalls, toolLog)

			// Add tool result to conversation
			var resultContent string
			if toolLog.Success {
				resultContent = toolLog.Output
			} else {
				resultContent = fmt.Sprintf("Error: %s", toolLog.Error)
			}
			messages = append(messages, ToolResultMsg(tc.ID, resultContent))
		}

		// Start spinner for next iteration
		spin.Suffix = fmt.Sprintf(" Iteration %d/%d...", iteration+2, a.config.MaxIterations)
		spin.Start()
	}

	spin.Stop()
	result.Error = "max iterations reached"
	a.log("✗ Max iterations reached (%d)", a.config.MaxIterations)
	return result, fmt.Errorf("max iterations reached")
}

// executeToolCall runs a single tool and returns the result.
func (a *Agent) executeToolCall(ctx context.Context, tc ToolCall) ToolCallLog {
	startTime := time.Now()

	log := ToolCallLog{
		ToolName: tc.Name,
	}

	// Parse parameters
	params, err := tc.ParseArgumentsMap()
	if err != nil {
		log.Error = fmt.Sprintf("failed to parse arguments: %v", err)
		log.Duration = time.Since(startTime)
		a.log("  ✗ %s: %s", tc.Name, log.Error)
		return log
	}
	log.Parameters = params

	// Get tool from registry
	tool, ok := a.registry.Get(tc.Name)
	if !ok {
		log.Error = fmt.Sprintf("unknown tool: %s", tc.Name)
		log.Duration = time.Since(startTime)
		a.log("  ✗ %s: %s", tc.Name, log.Error)
		return log
	}

	// Only explicitly known read-only tools may run without approval. This makes
	// newly added tools fail closed until their risk has been classified.
	if !isReadOnlyTool(tc.Name) {
		if a.config.DryRun {
			if a.config.ApprovalFunc != nil {
				a.config.ApprovalFunc(tc.Name, params)
			}
			log.Error = "blocked by dry-run"
			log.Duration = time.Since(startTime)
			a.log("  ⚠ %s: blocked by dry-run", tc.Name)
			return log
		}
		if a.config.ApprovalFunc == nil || !a.config.ApprovalFunc(tc.Name, params) {
			log.Error = "denied by user"
			log.Duration = time.Since(startTime)
			a.log("  ⚠ %s: denied by user", tc.Name)
			return log
		}
	}

	a.log("  ▶ Executing: %s", tc.Name)

	result := tool.Execute(ctx, params)
	log.Success = result.Success
	log.Output = a.formatToolOutput(result)
	log.Duration = result.Duration

	if !result.Success {
		log.Error = result.Error
		a.log("  ✗ %s failed: %s", tc.Name, log.Error)
	} else {
		a.log("  ✓ %s completed", tc.Name)
	}

	return log
}

func isReadOnlyTool(name string) bool {
	switch name {
	case "read_file", "read_dir", "search_codebase", "query_docker", "check_ports",
		"git_info", "package_info", "read_diff":
		return true
	default:
		return false
	}
}

// formatToolOutput converts tool result data to a string for the LLM.
func (a *Agent) formatToolOutput(result tools.ToolResult) string {
	if result.Data == nil {
		if result.Success {
			return "Success"
		}
		return result.Error
	}

	// Try to JSON marshal the data
	data, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", result.Data)
	}

	// Truncate very long outputs
	output := string(data)
	const maxLen = 4000
	if len(output) > maxLen {
		output = output[:maxLen] + "\n... (truncated)"
	}

	return output
}

// buildToolDefs converts registry tools to LLM tool definitions.
func (a *Agent) buildToolDefs() []ToolDef {
	providerDefs := tools.RegistryToProviderToolDefs(a.registry)
	defs := make([]ToolDef, len(providerDefs))
	for i, pd := range providerDefs {
		defs[i] = ToolDef{
			Name:        pd.Name,
			Description: pd.Description,
			Parameters:  pd.Parameters,
		}
	}
	return defs
}

// buildSystemPrompt creates the system prompt for the agent.
func (a *Agent) buildSystemPrompt() string {
	toolNames := a.registry.Names()
	return fmt.Sprintf(`You are an expert DevOps engineer and troubleshooter. Your goal is to analyze issues and resolve them using the available tools.

Available tools: %s

Guidelines:
1. Analyze the problem carefully before taking action
2. Use read_file, read_dir, and search_codebase to gather information
3. Use git_info and read_diff to understand recent changes
4. Use run_command for diagnostic commands (prefer non-destructive)
5. Use write_file only when you have a clear fix
6. Explain your reasoning as you work
7. When the issue is resolved, provide a summary without calling more tools

Be methodical and thorough. If you're unsure, gather more information before making changes.`,
		strings.Join(toolNames, ", "))
}

// log outputs a message if verbose mode is enabled.
func (a *Agent) log(format string, args ...interface{}) {
	if a.config.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}
