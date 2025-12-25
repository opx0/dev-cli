package agent

import (
	"errors"
	"testing"
)

// mockSolver is a test double for the Solver interface
type mockSolver struct {
	responses []string
	errors    []error
	callCount int
	prompts   []string
}

func (m *mockSolver) Solve(goal string) (string, error) {
	m.prompts = append(m.prompts, goal)
	idx := m.callCount
	m.callCount++

	if idx < len(m.errors) && m.errors[idx] != nil {
		return "", m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", nil
}

// mockExecutor is a test double for the Executor interface
type mockExecutor struct {
	results   []bool
	errors    []string
	callCount int
	commands  []string
}

func (m *mockExecutor) Execute(command string) (bool, string) {
	m.commands = append(m.commands, command)
	idx := m.callCount
	m.callCount++

	success := true
	errOutput := ""
	if idx < len(m.results) {
		success = m.results[idx]
	}
	if idx < len(m.errors) {
		errOutput = m.errors[idx]
	}
	return success, errOutput
}

func TestAgent_ResolveSuccess_FirstAttempt(t *testing.T) {
	solver := &mockSolver{
		responses: []string{"echo hello"},
	}
	executor := &mockExecutor{
		results: []bool{true},
	}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("print hello", func(proposal string) bool {
		return true
	})

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if solver.callCount != 1 {
		t.Errorf("expected 1 solver call, got %d", solver.callCount)
	}
	if executor.callCount != 1 {
		t.Errorf("expected 1 executor call, got %d", executor.callCount)
	}
	if len(executor.commands) > 0 && executor.commands[0] != "echo hello" {
		t.Errorf("expected command 'echo hello', got '%s'", executor.commands[0])
	}
}

func TestAgent_ResolveWithRetries(t *testing.T) {
	solver := &mockSolver{

		responses: []string{"bad-cmd", "good-cmd"},
	}
	executor := &mockExecutor{
		results: []bool{false, true},
		errors:  []string{"command not found", ""},
	}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("do something", func(proposal string) bool {
		return true
	})

	if err != nil {
		t.Errorf("expected success after retry, got error: %v", err)
	}
	if solver.callCount != 2 {
		t.Errorf("expected 2 solver calls, got %d", solver.callCount)
	}
	if executor.callCount != 2 {
		t.Errorf("expected 2 executor calls, got %d", executor.callCount)
	}

	if len(solver.prompts) >= 2 {
		if !contains(solver.prompts[1], "command not found") {
			t.Errorf("retry prompt should include previous error")
		}
	}
}

func TestAgent_ResolveMaxRetriesExceeded(t *testing.T) {
	solver := &mockSolver{
		responses: []string{"cmd1", "cmd2", "cmd3"},
	}
	executor := &mockExecutor{
		results: []bool{false, false, false},
		errors:  []string{"err1", "err2", "err3"},
	}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("impossible task", func(proposal string) bool {
		return true
	})

	if err == nil {
		t.Error("expected error for max retries exceeded")
	}
	if err.Error() != "max retries exceeded" {
		t.Errorf("expected 'max retries exceeded', got '%s'", err.Error())
	}
	if solver.callCount != 3 {
		t.Errorf("expected 3 solver calls (maxRetries), got %d", solver.callCount)
	}
}

func TestAgent_ResolveLLMError(t *testing.T) {
	llmErr := errors.New("connection refused")
	solver := &mockSolver{
		errors: []error{llmErr},
	}
	executor := &mockExecutor{}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("task", func(proposal string) bool {
		return true
	})

	if err == nil {
		t.Error("expected error from LLM")
	}
	if err != llmErr {
		t.Errorf("expected LLM error, got: %v", err)
	}
	if executor.callCount != 0 {
		t.Error("executor should not be called when LLM fails")
	}
}

func TestAgent_ResolveUserDenied(t *testing.T) {
	solver := &mockSolver{
		responses: []string{"rm -rf /"},
	}
	executor := &mockExecutor{}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("delete everything", func(proposal string) bool {
		return false
	})

	if err == nil {
		t.Error("expected error for denied proposal")
	}
	if err.Error() != "denied by user" {
		t.Errorf("expected 'denied by user', got '%s'", err.Error())
	}
	if executor.callCount != 0 {
		t.Error("executor should not be called when user denies")
	}
}

func TestAgent_ResolveEmptyProposal(t *testing.T) {
	solver := &mockSolver{
		responses: []string{""},
	}
	executor := &mockExecutor{}

	agent := NewWithDeps(solver, executor)

	err := agent.Resolve("unknown task", func(proposal string) bool {
		return true
	})

	if err == nil {
		t.Error("expected error for empty proposal")
	}
	if err.Error() != "no solution" {
		t.Errorf("expected 'no solution', got '%s'", err.Error())
	}
	if executor.callCount != 0 {
		t.Error("executor should not be called for empty proposal")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"hello world", 5, "hello..."},
		{"  trimmed  ", 20, "trimmed"},
		{"exactly10!", 10, "exactly10!"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
