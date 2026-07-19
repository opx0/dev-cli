package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFileTool(t *testing.T) {
	tool := &ReadFileTool{}

	t.Run("Name and Description", func(t *testing.T) {
		if tool.Name() != "read_file" {
			t.Errorf("expected name 'read_file', got %s", tool.Name())
		}
		if tool.Description() == "" {
			t.Error("expected non-empty description")
		}
	})

	t.Run("Read existing file", func(t *testing.T) {

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "line1\nline2\nline3"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := tool.Execute(context.Background(), map[string]any{
			"path": testFile,
		})

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data, ok := result.Data.(ReadFileResult)
		if !ok {
			t.Fatal("expected ReadFileResult data")
		}
		if data.Content != content {
			t.Errorf("expected content %q, got %q", content, data.Content)
		}
		if data.Lines != 3 {
			t.Errorf("expected 3 lines, got %d", data.Lines)
		}
	})

	t.Run("Read with line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := tool.Execute(context.Background(), map[string]any{
			"path":       testFile,
			"start_line": 2,
			"end_line":   4,
		})

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data := result.Data.(ReadFileResult)
		if data.Lines != 3 {
			t.Errorf("expected 3 lines, got %d", data.Lines)
		}
	})

	t.Run("File not found", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]any{
			"path": "/nonexistent/file.txt",
		})

		if result.Success {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("Missing path parameter", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]any{})

		if result.Success {
			t.Error("expected error for missing path")
		}
	})
}

func TestWriteFileTool(t *testing.T) {
	tool := &WriteFileTool{}

	t.Run("Write new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "new.txt")
		content := "hello world"

		result := tool.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"content": content,
		})

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data := result.Data.(WriteFileResult)
		if !data.Created {
			t.Error("expected Created to be true")
		}

		actual, _ := os.ReadFile(testFile)
		if string(actual) != content {
			t.Errorf("expected %q, got %q", content, string(actual))
		}
	})

	t.Run("Write with backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "existing.txt")

		os.WriteFile(testFile, []byte("original"), 0644)

		result := tool.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"content": "updated",
			"backup":  true,
		})

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data := result.Data.(WriteFileResult)
		if data.BackupPath == "" {
			t.Error("expected backup path")
		}

		backup, _ := os.ReadFile(data.BackupPath)
		if string(backup) != "original" {
			t.Errorf("expected backup content 'original', got %q", string(backup))
		}
	})
}

func TestRunCommandTool(t *testing.T) {
	tool := &RunCommandTool{}

	t.Run("Execute simple command", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]any{
			"command": "echo hello",
		})

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data := result.Data.(CommandResult)
		if data.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", data.ExitCode)
		}
	})

	t.Run("Missing command", func(t *testing.T) {
		result := tool.Execute(context.Background(), map[string]any{})

		if result.Success {
			t.Error("expected error for missing command")
		}
	})
}

func TestRegistry(t *testing.T) {
	t.Run("Register and get tool", func(t *testing.T) {
		reg := NewRegistry()

		tool := &ReadFileTool{}
		if err := reg.Register(tool); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		got, ok := reg.Get("read_file")
		if !ok {
			t.Error("expected to find registered tool")
		}
		if got.Name() != "read_file" {
			t.Errorf("expected 'read_file', got %s", got.Name())
		}
	})

	t.Run("Duplicate registration", func(t *testing.T) {
		reg := NewRegistry()

		tool := &ReadFileTool{}
		reg.Register(tool)

		err := reg.Register(tool)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})

	t.Run("List tools", func(t *testing.T) {
		reg := NewRegistry()
		reg.Register(&ReadFileTool{})
		reg.Register(&WriteFileTool{})

		list := reg.List()
		if len(list) != 2 {
			t.Errorf("expected 2 tools, got %d", len(list))
		}
	})

	t.Run("RegisterDefaults", func(t *testing.T) {
		reg := NewRegistry()
		reg.RegisterDefaults()

		if reg.Count() != 10 {
			t.Errorf("expected 10 default tools, got %d", reg.Count())
		}
	})
}

func TestParameterHelpers(t *testing.T) {
	t.Run("GetString", func(t *testing.T) {
		params := map[string]any{"key": "value"}
		if GetString(params, "key", "") != "value" {
			t.Error("expected 'value'")
		}
		if GetString(params, "missing", "default") != "default" {
			t.Error("expected 'default'")
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		params := map[string]any{"key": 42, "float": 3.14}
		if GetInt(params, "key", 0) != 42 {
			t.Error("expected 42")
		}
		if GetInt(params, "float", 0) != 3 {
			t.Error("expected 3 from float")
		}
		if GetInt(params, "missing", 99) != 99 {
			t.Error("expected 99")
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		params := map[string]any{"key": true}
		if !GetBool(params, "key", false) {
			t.Error("expected true")
		}
		if GetBool(params, "missing", true) != true {
			t.Error("expected true default")
		}
	})

	t.Run("GetDuration", func(t *testing.T) {
		params := map[string]any{"key": "30s", "seconds": 60}
		if GetDuration(params, "key", 0) != 30*time.Second {
			t.Error("expected 30s")
		}
		if GetDuration(params, "seconds", 0) != 60*time.Second {
			t.Error("expected 60s from int")
		}
	})
}
