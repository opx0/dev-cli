package diffsandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTracker_SnapshotAndRollback(t *testing.T) {
	dir := t.TempDir()

	// Create a file that will be modified
	existingFile := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("original content"), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker()

	// Snapshot existing file
	if err := tracker.Snapshot(existingFile); err != nil {
		t.Fatal(err)
	}

	// Snapshot a file that doesn't exist yet
	newFile := filepath.Join(dir, "new.txt")
	if err := tracker.Snapshot(newFile); err != nil {
		t.Fatal(err)
	}

	// Simulate writes
	if err := os.WriteFile(existingFile, []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}
	tracker.RecordWrite(existingFile, "modified content")

	if err := os.WriteFile(newFile, []byte("brand new file"), 0644); err != nil {
		t.Fatal(err)
	}
	tracker.RecordWrite(newFile, "brand new file")

	// Verify changes tracked
	if !tracker.HasChanges() {
		t.Error("expected HasChanges() = true")
	}
	if tracker.Count() != 2 {
		t.Errorf("expected Count() = 2, got %d", tracker.Count())
	}

	// Verify diff output
	diff := tracker.FormatDiff()
	if diff == "" {
		t.Error("expected non-empty diff")
	}

	// Rollback
	count, errs := tracker.Rollback()
	if len(errs) > 0 {
		t.Errorf("rollback errors: %v", errs)
	}
	if count != 2 {
		t.Errorf("expected 2 files rolled back, got %d", count)
	}

	// Verify existing file was restored
	data, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original content" {
		t.Errorf("expected original content, got %q", string(data))
	}

	// Verify new file was deleted
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Error("expected new file to be deleted after rollback")
	}
}

func TestTracker_DuplicateSnapshot(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	os.WriteFile(file, []byte("v1"), 0644)

	tracker := NewTracker()
	tracker.Snapshot(file)

	// Modify file
	os.WriteFile(file, []byte("v2"), 0644)
	tracker.RecordWrite(file, "v2")

	// Second snapshot should NOT overwrite the original
	tracker.Snapshot(file)

	changes := tracker.Changes()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Original != "v1" {
		t.Errorf("expected original to be 'v1', got %q", changes[0].Original)
	}
}

func TestTracker_NoChanges(t *testing.T) {
	tracker := NewTracker()
	if tracker.HasChanges() {
		t.Error("expected HasChanges() = false for empty tracker")
	}
	diff := tracker.FormatDiff()
	if diff != "No changes recorded." {
		t.Errorf("unexpected diff for empty tracker: %q", diff)
	}
}
