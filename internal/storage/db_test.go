package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "dev-cli-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "history.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	entry := LogEntry{
		Command:    "git status",
		ExitCode:   0,
		Output:     "On branch main",
		Cwd:        "/tmp",
		DurationMs: 12,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	if err := SaveCommand(db, entry); err != nil {
		t.Errorf("SaveCommand failed: %v", err)
	}

	items, err := GetRecentHistory(db, 10)
	if err != nil {
		t.Errorf("GetRecentHistory failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	} else {
		if items[0].Command != "git status" {
			t.Errorf("Expected 'git status', got '%s'", items[0].Command)
		}
	}

	entry2 := LogEntry{
		Command:   "docker run hello-world",
		ExitCode:  0,
		Output:    "Hello from Docker!",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := SaveCommand(db, entry2); err != nil {
		t.Errorf("SaveCommand failed: %v", err)
	}

	results, err := SearchHistory(db, "docker")
	if err != nil {
		t.Errorf("SearchHistory failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 search result for 'docker', got %d", len(results))
	} else if results[0].Command != "docker run hello-world" {
		t.Errorf("Got wrong result: %s", results[0].Command)
	}

	results2, err := SearchHistory(db, "Hello")
	if err != nil {
		t.Errorf("SearchHistory failed: %v", err)
	}
	if len(results2) < 1 {
		t.Errorf("Expected 1 search result for 'Hello' (in output), got %d", len(results2))
	}
}

func TestResolutionTracking(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "dev-cli-test-resolution")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "history.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	failEntry := LogEntry{
		Command:    "npm run build",
		ExitCode:   1,
		Output:     "Error: Module not found",
		Cwd:        "/tmp/project",
		DurationMs: 500,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	if err := SaveCommand(db, failEntry); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	failure, err := GetLastUnresolvedFailure(db)
	if err != nil {
		t.Fatalf("GetLastUnresolvedFailure failed: %v", err)
	}
	if failure == nil {
		t.Fatal("Expected to find an unresolved failure, got nil")
	}
	if failure.Command != "npm run build" {
		t.Errorf("Expected 'npm run build', got '%s'", failure.Command)
	}
	if failure.Resolution != "" {
		t.Errorf("Expected empty resolution, got '%s'", failure.Resolution)
	}

	if err := MarkResolution(db, failure.ID, "solution"); err != nil {
		t.Fatalf("MarkResolution failed: %v", err)
	}

	item, err := GetHistoryByID(db, failure.ID)
	if err != nil {
		t.Fatalf("GetHistoryByID failed: %v", err)
	}
	if item.Resolution != "solution" {
		t.Errorf("Expected resolution 'solution', got '%s'", item.Resolution)
	}

	failure2, err := GetLastUnresolvedFailure(db)
	if err != nil {
		t.Fatalf("GetLastUnresolvedFailure failed: %v", err)
	}
	if failure2 != nil {
		t.Errorf("Expected no unresolved failures, but got: %+v", failure2)
	}

	err = MarkResolution(db, 99999, "solution")
	if err == nil {
		t.Error("Expected error for invalid ID, got nil")
	}
}
