package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	// Setup temp DB
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

	// Test Insert
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

	// Test GetRecentHistory
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

	// Test FTS Search
	// Add another item specifically for search
	entry2 := LogEntry{
		Command:   "docker run hello-world",
		ExitCode:  0,
		Output:    "Hello from Docker!",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := SaveCommand(db, entry2); err != nil {
		t.Errorf("SaveCommand failed: %v", err)
	}

	// FTS triggers might be asynchronous if using FTS5? No, usually sync within transaction.
	// But sqlite pure driver might handle it fine.

	results, err := SearchHistory(db, "docker")
	if err != nil {
		t.Errorf("SearchHistory failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 search result for 'docker', got %d", len(results))
	} else if results[0].Command != "docker run hello-world" {
		t.Errorf("Got wrong result: %s", results[0].Command)
	}

	// Search in output (details)
	results2, err := SearchHistory(db, "Hello")
	if err != nil {
		t.Errorf("SearchHistory failed: %v", err)
	}
	if len(results2) < 1 {
		t.Errorf("Expected 1 search result for 'Hello' (in output), got %d", len(results2))
	}
}
