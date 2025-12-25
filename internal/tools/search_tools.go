package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SearchCodebaseTool searches for patterns in code using ripgrep.
type SearchCodebaseTool struct{}

func (t *SearchCodebaseTool) Name() string        { return "search_codebase" }
func (t *SearchCodebaseTool) Description() string { return "Search for patterns in code using ripgrep" }

func (t *SearchCodebaseTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "pattern", Type: "string", Description: "Search pattern (regex)", Required: true},
		{Name: "path", Type: "string", Description: "Path to search in", Required: false, Default: "."},
		{Name: "file_types", Type: "[]string", Description: "File types to include (e.g., 'go', 'py')", Required: false},
		{Name: "ignore_case", Type: "bool", Description: "Case-insensitive search", Required: false, Default: false},
		{Name: "max_results", Type: "int", Description: "Maximum results", Required: false, Default: 50},
		{Name: "context_lines", Type: "int", Description: "Context lines around match", Required: false, Default: 0},
	}
}

// SearchMatch represents a single search match.
type SearchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Content string `json:"content"`
}

// SearchResult contains the search output.
type SearchResult struct {
	Pattern    string        `json:"pattern"`
	Path       string        `json:"path"`
	Matches    []SearchMatch `json:"matches"`
	TotalCount int           `json:"total_count"`
	Truncated  bool          `json:"truncated"`
}

func (t *SearchCodebaseTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	pattern := GetString(params, "pattern", "")
	if pattern == "" {
		return NewErrorResult("pattern is required", time.Since(start))
	}

	searchPath := GetString(params, "path", ".")
	ignoreCase := GetBool(params, "ignore_case", false)
	maxResults := GetInt(params, "max_results", 50)
	contextLines := GetInt(params, "context_lines", 0)
	fileTypes := GetStringSlice(params, "file_types")

	if _, err := exec.LookPath("rg"); err != nil {

		return t.executeWithGrep(ctx, pattern, searchPath, ignoreCase, maxResults)
	}

	args := []string{
		"--json",
		"--max-count", strconv.Itoa(maxResults * 2),
	}

	if ignoreCase {
		args = append(args, "-i")
	}

	if contextLines > 0 {
		args = append(args, "-C", strconv.Itoa(contextLines))
	}

	for _, ft := range fileTypes {
		args = append(args, "-t", ft)
	}

	args = append(args, pattern, searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, _ := cmd.Output()

	matches := parseRipgrepJSON(string(output))

	truncated := false
	if len(matches) > maxResults {
		matches = matches[:maxResults]
		truncated = true
	}

	return NewResult(SearchResult{
		Pattern:    pattern,
		Path:       searchPath,
		Matches:    matches,
		TotalCount: len(matches),
		Truncated:  truncated,
	}, time.Since(start))
}

func (t *SearchCodebaseTool) executeWithGrep(ctx context.Context, pattern, path string, ignoreCase bool, maxResults int) ToolResult {
	start := time.Now()

	args := []string{"-rn"}
	if ignoreCase {
		args = append(args, "-i")
	}
	args = append(args, pattern, path)

	cmd := exec.CommandContext(ctx, "grep", args...)
	output, _ := cmd.Output()

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	matches := make([]SearchMatch, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			lineNum, _ := strconv.Atoi(parts[1])
			matches = append(matches, SearchMatch{
				File:    parts[0],
				Line:    lineNum,
				Content: parts[2],
			})
		}
		if len(matches) >= maxResults {
			break
		}
	}

	return NewResult(SearchResult{
		Pattern:    pattern,
		Path:       path,
		Matches:    matches,
		TotalCount: len(matches),
		Truncated:  len(lines) > maxResults,
	}, time.Since(start))
}

func parseRipgrepJSON(output string) []SearchMatch {
	var matches []SearchMatch

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		var msg struct {
			Type string `json:"type"`
			Data struct {
				Path struct {
					Text string `json:"text"`
				} `json:"path"`
				LineNumber int `json:"line_number"`
				Lines      struct {
					Text string `json:"text"`
				} `json:"lines"`
				Submatches []struct {
					Start int `json:"start"`
				} `json:"submatches"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Type == "match" {
			match := SearchMatch{
				File:    msg.Data.Path.Text,
				Line:    msg.Data.LineNumber,
				Content: strings.TrimRight(msg.Data.Lines.Text, "\n"),
			}
			if len(msg.Data.Submatches) > 0 {
				match.Column = msg.Data.Submatches[0].Start + 1
			}
			matches = append(matches, match)
		}
	}

	return matches
}
