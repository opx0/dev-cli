// Package diffsandbox tracks file changes made by the fix agent so they can be
// reviewed as a cumulative diff and optionally rolled back.
package diffsandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileChange records a single file modification.
type FileChange struct {
	Path       string    // Absolute path to the file
	OriginalOK bool      // True if the file existed before the change
	Original   string    // Original contents (empty if created)
	Modified   string    // New contents
	Timestamp  time.Time // When the change happened
}

// Tracker records all file modifications during an agent session.
type Tracker struct {
	mu      sync.Mutex
	changes map[string]*FileChange // path → change
	order   []string               // insertion order
}

// NewTracker creates a new change tracker.
func NewTracker() *Tracker {
	return &Tracker{
		changes: make(map[string]*FileChange),
		order:   make([]string, 0),
	}
}

// Snapshot reads and stores the current contents of a file before it gets
// modified. Call this before every write_file operation.
func (t *Tracker) Snapshot(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Only snapshot once per path (keep the original)
	if _, exists := t.changes[abs]; exists {
		return nil
	}

	change := &FileChange{
		Path:      abs,
		Timestamp: time.Now(),
	}

	data, err := os.ReadFile(abs)
	if err == nil {
		change.OriginalOK = true
		change.Original = string(data)
	}
	// If file doesn't exist yet, OriginalOK stays false (it's a new file)

	t.changes[abs] = change
	t.order = append(t.order, abs)
	return nil
}

// RecordWrite records what was written to a file. Call this after every
// successful write_file operation.
func (t *Tracker) RecordWrite(path, content string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if change, exists := t.changes[abs]; exists {
		change.Modified = content
	} else {
		// Snapshot wasn't called — record as new file
		t.changes[abs] = &FileChange{
			Path:      abs,
			Modified:  content,
			Timestamp: time.Now(),
		}
		t.order = append(t.order, abs)
	}
}

// Changes returns all tracked changes in insertion order.
func (t *Tracker) Changes() []FileChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]FileChange, 0, len(t.order))
	for _, p := range t.order {
		if c, ok := t.changes[p]; ok {
			result = append(result, *c)
		}
	}
	return result
}

// HasChanges returns true if any files were modified.
func (t *Tracker) HasChanges() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.changes) > 0
}

// Count returns the number of changed files.
func (t *Tracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.changes)
}

// Rollback restores all files to their original state. Files that were newly
// created are deleted. Returns the number of files rolled back and any errors.
func (t *Tracker) Rollback() (int, []error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var errors []error
	count := 0

	// Roll back in reverse order
	for i := len(t.order) - 1; i >= 0; i-- {
		path := t.order[i]
		change, ok := t.changes[path]
		if !ok {
			continue
		}

		if !change.OriginalOK {
			// File was created by the agent — delete it
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errors = append(errors, fmt.Errorf("remove %s: %w", path, err))
			} else {
				count++
			}
		} else {
			// File existed before — restore original
			if err := os.WriteFile(path, []byte(change.Original), 0644); err != nil {
				errors = append(errors, fmt.Errorf("restore %s: %w", path, err))
			} else {
				count++
			}
		}
	}

	return count, errors
}

// FormatDiff generates a unified-diff-like summary of all changes. This is
// the "diff sandbox" view shown to the user before applying.
func (t *Tracker) FormatDiff() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.order) == 0 {
		return "No changes recorded."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("─── %d file(s) changed ───\n\n", len(t.order)))

	for _, path := range t.order {
		change, ok := t.changes[path]
		if !ok {
			continue
		}

		relPath := path
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, path); err == nil {
				relPath = rel
			}
		}

		if !change.OriginalOK {
			// New file
			sb.WriteString(fmt.Sprintf("╭─ \033[32m+ %s\033[0m (new file)\n", relPath))
			lines := strings.Split(change.Modified, "\n")
			maxPreview := 15
			for i, line := range lines {
				if i >= maxPreview {
					sb.WriteString(fmt.Sprintf("│  \033[90m... +%d more lines\033[0m\n", len(lines)-maxPreview))
					break
				}
				sb.WriteString(fmt.Sprintf("│  \033[32m+%s\033[0m\n", line))
			}
			sb.WriteString("╰───\n\n")
		} else if change.Modified == change.Original {
			// No actual change
			continue
		} else {
			// Modified file — show a simplified diff
			sb.WriteString(fmt.Sprintf("╭─ \033[33m~ %s\033[0m (modified)\n", relPath))

			origLines := strings.Split(change.Original, "\n")
			modLines := strings.Split(change.Modified, "\n")

			// Simple line-by-line comparison
			maxI := len(origLines)
			if len(modLines) > maxI {
				maxI = len(modLines)
			}
			diffCount := 0
			maxDiffLines := 30

			for i := 0; i < maxI && diffCount < maxDiffLines; i++ {
				origLine := ""
				modLine := ""
				if i < len(origLines) {
					origLine = origLines[i]
				}
				if i < len(modLines) {
					modLine = modLines[i]
				}

				if origLine != modLine {
					if origLine != "" {
						sb.WriteString(fmt.Sprintf("│  \033[31m-%s\033[0m\n", origLine))
					}
					if modLine != "" {
						sb.WriteString(fmt.Sprintf("│  \033[32m+%s\033[0m\n", modLine))
					}
					diffCount++
				}
			}

			if diffCount >= maxDiffLines {
				sb.WriteString(fmt.Sprintf("│  \033[90m... (diff truncated)\033[0m\n"))
			}

			sb.WriteString(fmt.Sprintf("│  \033[90m%d → %d lines\033[0m\n", len(origLines), len(modLines)))
			sb.WriteString("╰───\n\n")
		}
	}

	return sb.String()
}
