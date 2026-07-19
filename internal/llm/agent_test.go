package llm

import (
	"context"
	"testing"
	"time"

	"dev-cli/internal/tools"
)

type recordingTool struct {
	name     string
	executed bool
}

func (t *recordingTool) Name() string                  { return t.name }
func (t *recordingTool) Description() string           { return "test tool" }
func (t *recordingTool) Parameters() []tools.ToolParam { return nil }
func (t *recordingTool) Execute(context.Context, map[string]any) tools.ToolResult {
	t.executed = true
	return tools.NewResult("ok", time.Millisecond)
}

func TestAgentDryRunAllowsReadOnlyTools(t *testing.T) {
	tool := &recordingTool{name: "read_file"}
	agent := agentWithTool(t, tool)
	agent.config.DryRun = true

	result := agent.executeToolCall(context.Background(), ToolCall{Name: tool.name, Arguments: `{}`})
	if !result.Success || !tool.executed {
		t.Fatalf("read-only tool should execute in dry-run: %+v", result)
	}
}

func TestAgentMutatingToolsFailClosed(t *testing.T) {
	tool := &recordingTool{name: "future_mutation"}
	agent := agentWithTool(t, tool)

	result := agent.executeToolCall(context.Background(), ToolCall{Name: tool.name, Arguments: `{}`})
	if result.Success || tool.executed || result.Error != "denied by user" {
		t.Fatalf("unclassified tool should require approval: %+v", result)
	}
}

func TestAgentDryRunBlocksMutatingTools(t *testing.T) {
	tool := &recordingTool{name: "write_file"}
	agent := agentWithTool(t, tool)
	agent.config.DryRun = true
	agent.config.ApprovalFunc = func(string, map[string]any) bool { return true }

	result := agent.executeToolCall(context.Background(), ToolCall{Name: tool.name, Arguments: `{}`})
	if result.Success || tool.executed || result.Error != "blocked by dry-run" {
		t.Fatalf("mutating tool executed in dry-run: %+v", result)
	}
}

func TestAgentExecutesApprovedMutation(t *testing.T) {
	tool := &recordingTool{name: "write_file"}
	agent := agentWithTool(t, tool)
	agent.config.ApprovalFunc = func(string, map[string]any) bool { return true }

	result := agent.executeToolCall(context.Background(), ToolCall{Name: tool.name, Arguments: `{}`})
	if !result.Success || !tool.executed {
		t.Fatalf("approved mutation should execute: %+v", result)
	}
}

func agentWithTool(t *testing.T, tool tools.Tool) *Agent {
	t.Helper()
	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}
	return &Agent{registry: registry, config: DefaultAgentConfig()}
}
