package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReadFileTool reads file contents.
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read file contents with optional line range" }

func (t *ReadFileTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "path", Type: "string", Description: "Path to file", Required: true},
		{Name: "start_line", Type: "int", Description: "Start line (1-indexed)", Required: false, Default: 0},
		{Name: "end_line", Type: "int", Description: "End line (1-indexed, 0 = all)", Required: false, Default: 0},
		{Name: "max_size", Type: "int", Description: "Max bytes to read (0 = 1MB default)", Required: false, Default: 0},
	}
}

// ReadFileResult contains the file reading output.
type ReadFileResult struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Lines     int    `json:"lines"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	path := GetString(params, "path", "")
	if path == "" {
		return NewErrorResult("path is required", time.Since(start))
	}

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("invalid path: %v", err), time.Since(start))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("file not found: %s", absPath), time.Since(start))
		}
		return NewErrorResult(fmt.Sprintf("cannot access file: %v", err), time.Since(start))
	}

	if info.IsDir() {
		return NewErrorResult("path is a directory, not a file", time.Since(start))
	}

	maxSize := GetInt(params, "max_size", 0)
	if maxSize <= 0 {
		maxSize = 1024 * 1024
	}

	truncated := false
	if info.Size() > int64(maxSize) {
		truncated = true
	}

	file, err := os.Open(absPath)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("cannot open file: %v", err), time.Since(start))
	}
	defer file.Close()

	reader := io.LimitReader(file, int64(maxSize))
	data, err := io.ReadAll(reader)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("read error: %v", err), time.Since(start))
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	startLine := GetInt(params, "start_line", 0)
	endLine := GetInt(params, "end_line", 0)

	if startLine > 0 || endLine > 0 {
		if startLine < 1 {
			startLine = 1
		}
		if endLine < 1 || endLine > totalLines {
			endLine = totalLines
		}
		if startLine > totalLines {
			startLine = totalLines
		}
		if startLine > endLine {
			startLine = endLine
		}

		lines = lines[startLine-1 : endLine]
		content = strings.Join(lines, "\n")
	}

	result := ReadFileResult{
		Path:      absPath,
		Content:   content,
		Lines:     len(lines),
		Size:      info.Size(),
		Truncated: truncated,
	}
	if startLine > 0 {
		result.StartLine = startLine
		result.EndLine = endLine
	}

	return NewResult(result, time.Since(start))
}

// WriteFileTool writes content to a file.
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Write content to a file with optional backup" }

func (t *WriteFileTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "path", Type: "string", Description: "Path to file", Required: true},
		{Name: "content", Type: "string", Description: "Content to write", Required: true},
		{Name: "mode", Type: "string", Description: "File mode (e.g., '0644')", Required: false, Default: "0644"},
		{Name: "backup", Type: "bool", Description: "Create backup of existing file", Required: false, Default: false},
		{Name: "create_dirs", Type: "bool", Description: "Create parent directories", Required: false, Default: true},
	}
}

// WriteFileResult contains the file writing output.
type WriteFileResult struct {
	Path       string `json:"path"`
	Size       int    `json:"size"`
	BackupPath string `json:"backup_path,omitempty"`
	Created    bool   `json:"created"`
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	path := GetString(params, "path", "")
	if path == "" {
		return NewErrorResult("path is required", time.Since(start))
	}

	content := GetString(params, "content", "")

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("invalid path: %v", err), time.Since(start))
	}

	_, err = os.Stat(absPath)
	exists := err == nil
	created := !exists

	// Create backup if requested
	var backupPath string
	if exists && GetBool(params, "backup", false) {
		backupPath = absPath + ".bak"
		if err := copyFile(absPath, backupPath); err != nil {
			return NewErrorResult(fmt.Sprintf("backup failed: %v", err), time.Since(start))
		}
	}

	if GetBool(params, "create_dirs", true) {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return NewErrorResult(fmt.Sprintf("cannot create directory: %v", err), time.Since(start))
		}
	}

	mode := os.FileMode(0644)
	modeStr := GetString(params, "mode", "0644")
	if _, err := fmt.Sscanf(modeStr, "%o", &mode); err != nil {
		mode = 0644
	}

	if err := os.WriteFile(absPath, []byte(content), mode); err != nil {
		return NewErrorResult(fmt.Sprintf("write failed: %v", err), time.Since(start))
	}

	return NewResult(WriteFileResult{
		Path:       absPath,
		Size:       len(content),
		BackupPath: backupPath,
		Created:    created,
	}, time.Since(start))
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// ReadDirTool lists directory contents.
type ReadDirTool struct{}

func (t *ReadDirTool) Name() string { return "read_dir" }
func (t *ReadDirTool) Description() string {
	return "List directory contents with optional recursive traversal"
}

func (t *ReadDirTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "path", Type: "string", Description: "Path to directory", Required: true},
		{Name: "recursive", Type: "bool", Description: "Recursively list subdirectories", Required: false, Default: false},
		{Name: "max_depth", Type: "int", Description: "Max depth for recursive listing (0 = unlimited)", Required: false, Default: 0},
		{Name: "include_hidden", Type: "bool", Description: "Include hidden files (starting with .)", Required: false, Default: false},
		{Name: "max_entries", Type: "int", Description: "Max entries to return (0 = 1000 default)", Required: false, Default: 0},
	}
}

// DirEntry represents a single directory entry.
type DirEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"` // "file" or "dir"
	Size    int64  `json:"size,omitempty"`
	ModTime string `json:"mod_time,omitempty"`
}

// ReadDirResult contains the directory listing output.
type ReadDirResult struct {
	Path       string     `json:"path"`
	Entries    []DirEntry `json:"entries"`
	TotalCount int        `json:"total_count"`
	Truncated  bool       `json:"truncated"`
}

func (t *ReadDirTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	path := GetString(params, "path", "")
	if path == "" {
		return NewErrorResult("path is required", time.Since(start))
	}

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("invalid path: %v", err), time.Since(start))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("directory not found: %s", absPath), time.Since(start))
		}
		return NewErrorResult(fmt.Sprintf("cannot access directory: %v", err), time.Since(start))
	}

	if !info.IsDir() {
		return NewErrorResult("path is a file, not a directory", time.Since(start))
	}

	recursive := GetBool(params, "recursive", false)
	maxDepth := GetInt(params, "max_depth", 0)
	includeHidden := GetBool(params, "include_hidden", false)
	maxEntries := GetInt(params, "max_entries", 0)
	if maxEntries <= 0 {
		maxEntries = 1000
	}

	entries := make([]DirEntry, 0)
	truncated := false

	if recursive {
		entries, truncated = t.walkDir(absPath, absPath, 0, maxDepth, includeHidden, maxEntries)
	} else {
		entries, truncated = t.listDir(absPath, includeHidden, maxEntries)
	}

	return NewResult(ReadDirResult{
		Path:       absPath,
		Entries:    entries,
		TotalCount: len(entries),
		Truncated:  truncated,
	}, time.Since(start))
}

func (t *ReadDirTool) listDir(dir string, includeHidden bool, maxEntries int) ([]DirEntry, bool) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}

	entries := make([]DirEntry, 0, len(files))
	for _, f := range files {
		if !includeHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}

		if len(entries) >= maxEntries {
			return entries, true
		}

		entry := DirEntry{
			Name: f.Name(),
			Path: filepath.Join(dir, f.Name()),
		}

		if f.IsDir() {
			entry.Type = "dir"
		} else {
			entry.Type = "file"
			if info, err := f.Info(); err == nil {
				entry.Size = info.Size()
				entry.ModTime = info.ModTime().Format(time.RFC3339)
			}
		}

		entries = append(entries, entry)
	}

	return entries, false
}

func (t *ReadDirTool) walkDir(basePath, currentPath string, currentDepth, maxDepth int, includeHidden bool, maxEntries int) ([]DirEntry, bool) {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil, false
	}

	files, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, false
	}

	entries := make([]DirEntry, 0)
	for _, f := range files {
		if !includeHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}

		if len(entries) >= maxEntries {
			return entries, true
		}

		fullPath := filepath.Join(currentPath, f.Name())
		entry := DirEntry{
			Name: f.Name(),
			Path: fullPath,
		}

		if f.IsDir() {
			entry.Type = "dir"
			entries = append(entries, entry)

			subEntries, truncated := t.walkDir(basePath, fullPath, currentDepth+1, maxDepth, includeHidden, maxEntries-len(entries))
			entries = append(entries, subEntries...)
			if truncated {
				return entries, true
			}
		} else {
			entry.Type = "file"
			if info, err := f.Info(); err == nil {
				entry.Size = info.Size()
				entry.ModTime = info.ModTime().Format(time.RFC3339)
			}
			entries = append(entries, entry)
		}
	}

	return entries, false
}
