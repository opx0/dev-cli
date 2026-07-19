package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndReadHistory(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	entry := LogEntry{
		Command:    "git status",
		ExitCode:   1,
		Output:     "failed",
		Cwd:        "/tmp",
		DurationMs: 12,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	if err := SaveCommand(db, entry); err != nil {
		t.Fatal(err)
	}

	items, err := GetRecentHistory(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Command != entry.Command || items[0].ExitCode != entry.ExitCode {
		t.Fatalf("unexpected history: %+v", items)
	}

	failures, err := GetFailures(db, QueryOpts{Limit: 1, Filter: "git"})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 1 || failures[0].Command != entry.Command {
		t.Fatalf("unexpected failures: %+v", failures)
	}
}
