package workflow

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	yaml := `
name: test-workflow
description: A test workflow
steps:
  - id: step1
    name: First Step
    command: echo "hello"
    timeout: 30s
    rollback: echo "rollback step1"
  - id: step2
    name: Second Step
    command: echo "world"
    condition:
      type: exit_code
      value: "0"
    on_failure: abort
on_failure:
  action: rollback
`

	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if wf.Name != "test-workflow" {
		t.Errorf("Name = %q, want %q", wf.Name, "test-workflow")
	}

	if wf.Description != "A test workflow" {
		t.Errorf("Description = %q, want %q", wf.Description, "A test workflow")
	}

	if len(wf.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(wf.Steps))
	}

	step1 := wf.Steps[0]
	if step1.ID != "step1" {
		t.Errorf("Step1.ID = %q, want %q", step1.ID, "step1")
	}
	if step1.Command != `echo "hello"` {
		t.Errorf("Step1.Command = %q, want %q", step1.Command, `echo "hello"`)
	}
	if step1.Timeout != 30*time.Second {
		t.Errorf("Step1.Timeout = %v, want %v", step1.Timeout, 30*time.Second)
	}
	if step1.Rollback == nil {
		t.Error("Step1.Rollback is nil, expected non-nil")
	} else if step1.Rollback.Command != `echo "rollback step1"` {
		t.Errorf("Step1.Rollback.Command = %q", step1.Rollback.Command)
	}

	step2 := wf.Steps[1]
	if step2.Condition == nil {
		t.Error("Step2.Condition is nil")
	} else {
		if step2.Condition.Type != CondExitCode {
			t.Errorf("Step2.Condition.Type = %q, want %q", step2.Condition.Type, CondExitCode)
		}
		if step2.Condition.Value != "0" {
			t.Errorf("Step2.Condition.Value = %q, want %q", step2.Condition.Value, "0")
		}
	}
	if step2.OnFailure != "abort" {
		t.Errorf("Step2.OnFailure = %q, want %q", step2.OnFailure, "abort")
	}

	if wf.OnFailure == nil {
		t.Error("OnFailure is nil")
	} else if wf.OnFailure.Action != FailureRollback {
		t.Errorf("OnFailure.Action = %q, want %q", wf.OnFailure.Action, FailureRollback)
	}
}

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "missing name",
			yaml:    "steps:\n  - command: echo test",
			wantErr: "name is required",
		},
		{
			name:    "missing steps",
			yaml:    "name: test",
			wantErr: "at least one step",
		},
		{
			name: "missing command",
			yaml: `
name: test
steps:
  - id: step1
    name: No Command`,
			wantErr: "command is required",
		},
		{
			name: "duplicate step ID",
			yaml: `
name: test
steps:
  - id: step1
    command: echo 1
  - id: step1
    command: echo 2`,
			wantErr: "duplicate step ID",
		},
		{
			name: "invalid on_success reference",
			yaml: `
name: test
steps:
  - id: step1
    command: echo 1
    on_success: nonexistent`,
			wantErr: "unknown step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil {
				t.Error("Parse() expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestRollbackYAMLShorthand(t *testing.T) {

	yaml := `
name: test
steps:
  - id: step1
    command: touch /tmp/test
    rollback: rm /tmp/test
`

	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if wf.Steps[0].Rollback == nil {
		t.Fatal("Rollback is nil")
	}

	if wf.Steps[0].Rollback.Command != "rm /tmp/test" {
		t.Errorf("Rollback.Command = %q, want %q", wf.Steps[0].Rollback.Command, "rm /tmp/test")
	}
}

func TestGenerateRunID(t *testing.T) {
	id1 := GenerateRunID()
	id2 := GenerateRunID()

	if id1 == id2 {
		t.Error("GenerateRunID() should return unique IDs")
	}

	if !strings.HasPrefix(id1, "run_") {
		t.Errorf("GenerateRunID() = %q, want prefix 'run_'", id1)
	}
}
