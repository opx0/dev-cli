package workflow

import (
	"testing"
)

func TestSafeMode_DefaultIsPreview(t *testing.T) {
	ctx := NewSafeModeContext()

	if ctx.Mode != SafeModePreview {
		t.Errorf("expected default mode to be preview, got %v", ctx.Mode)
	}
	if !ctx.IsPreview() {
		t.Error("IsPreview should return true by default")
	}
}

func TestSafeMode_ExecuteContext(t *testing.T) {
	approvalCalled := false
	ctx := NewExecuteContext(func(action string) bool {
		approvalCalled = true
		return true
	})

	if ctx.Mode != SafeModeExecute {
		t.Errorf("expected execute mode, got %v", ctx.Mode)
	}
	if ctx.IsPreview() {
		t.Error("IsPreview should return false for execute context")
	}

	ctx.RequireApproval("test action")
	if !approvalCalled {
		t.Error("approval function should be called")
	}
}

func TestSafeMode_PreviewAction(t *testing.T) {
	ctx := NewSafeModeContext()

	ctx.PreviewAction("step-1", "Install dependencies", "npm install")
	ctx.PreviewAction("step-2", "Delete temp files", "rm -rf /tmp/test")

	if len(ctx.DryRunOutput) != 2 {
		t.Errorf("expected 2 preview actions, got %d", len(ctx.DryRunOutput))
	}
	if ctx.DryRunOutput[0].Command != "npm install" {
		t.Errorf("expected first command 'npm install', got '%s'", ctx.DryRunOutput[0].Command)
	}
	if !ctx.DryRunOutput[1].Destructive {
		t.Error("rm -rf should be detected as destructive")
	}
}

func TestSafeMode_DestructivePatternDetection(t *testing.T) {
	ctx := NewSafeModeContext()

	tests := []struct {
		command     string
		destructive bool
	}{
		{"npm install", false},
		{"cat package.json", false},
		{"rm -rf /tmp/test", true},
		{"dd if=/dev/zero of=/dev/sda", true},
		{"DROP TABLE users", true},
		{"git reset --hard HEAD", true},
		{"docker system prune", true},
		{"echo hello", false},
	}

	for _, tt := range tests {
		ctx.ClearPreview()
		ctx.PreviewAction("test", "Test action", tt.command)

		if ctx.DryRunOutput[0].Destructive != tt.destructive {
			t.Errorf("command '%s': expected destructive=%v, got %v",
				tt.command, tt.destructive, ctx.DryRunOutput[0].Destructive)
		}
	}
}

func TestSafeMode_RequireApprovalForDestructive(t *testing.T) {
	approvalCount := 0
	ctx := NewExecuteContext(func(action string) bool {
		approvalCount++
		return true
	})

	result := ctx.RequireApprovalForDestructive("npm install")
	if !result {
		t.Error("non-destructive command should be approved")
	}
	if approvalCount != 0 {
		t.Errorf("expected 0 approval calls for non-destructive, got %d", approvalCount)
	}

	result = ctx.RequireApprovalForDestructive("rm -rf /important")
	if !result {
		t.Error("destructive command should be approved when approval func returns true")
	}
	if approvalCount != 1 {
		t.Errorf("expected 1 approval call for destructive, got %d", approvalCount)
	}
}

func TestSafeMode_GetPreviewSummary(t *testing.T) {
	ctx := NewSafeModeContext()

	summary := ctx.GetPreviewSummary()
	if summary != "No actions would be taken." {
		t.Errorf("expected 'No actions would be taken.', got '%s'", summary)
	}

	ctx.PreviewAction("s1", "Safe action", "npm install")
	ctx.PreviewAction("s2", "Dangerous action", "rm -rf /")

	summary = ctx.GetPreviewSummary()
	if len(summary) == 0 {
		t.Error("summary should not be empty")
	}

	if !contains(summary, "destructive") {
		t.Error("summary should mention destructive actions")
	}
}

func TestSafeMode_ClearPreview(t *testing.T) {
	ctx := NewSafeModeContext()

	ctx.PreviewAction("s1", "Action 1", "cmd1")
	ctx.PreviewAction("s2", "Action 2", "cmd2")
	ctx.ClearPreview()

	if len(ctx.DryRunOutput) != 0 {
		t.Errorf("expected 0 actions after clear, got %d", len(ctx.DryRunOutput))
	}
}

func TestSafeMode_String(t *testing.T) {
	if SafeModePreview.String() != "preview" {
		t.Errorf("expected 'preview', got '%s'", SafeModePreview.String())
	}
	if SafeModeExecute.String() != "execute" {
		t.Errorf("expected 'execute', got '%s'", SafeModeExecute.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
