package tools

import (
	"context"
	"strconv"
	"strings"
	"time"

	"dev-cli/internal/executor"
)

// GitInfoTool retrieves Git repository information.
type GitInfoTool struct{}

func (t *GitInfoTool) Name() string { return "git_info" }
func (t *GitInfoTool) Description() string {
	return "Get Git repository info: commits, blame, diff, status"
}

func (t *GitInfoTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "action", Type: "string", Description: "Action: log, blame, diff, status, branch", Required: true},
		{Name: "path", Type: "string", Description: "File path (for blame)", Required: false},
		{Name: "count", Type: "int", Description: "Number of commits (for log)", Required: false, Default: 10},
		{Name: "ref", Type: "string", Description: "Git ref (branch, commit, tag)", Required: false, Default: "HEAD"},
	}
}

// GitLogResult contains git log output.
type GitLogResult struct {
	Commits []GitCommit `json:"commits"`
	Count   int         `json:"count"`
}

// GitCommit represents a single commit.
type GitCommit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

// GitBlameResult contains git blame output.
type GitBlameResult struct {
	Path  string      `json:"path"`
	Lines []BlameLine `json:"lines"`
}

// BlameLine represents a line with blame info.
type BlameLine struct {
	LineNum int    `json:"line"`
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

// GitDiffResult contains git diff output.
type GitDiffResult struct {
	Ref     string `json:"ref"`
	Diff    string `json:"diff"`
	Stats   string `json:"stats"`
	Changed int    `json:"changed"`
}

// GitStatusResult contains git status output.
type GitStatusResult struct {
	Branch   string       `json:"branch"`
	Clean    bool         `json:"clean"`
	Staged   []FileChange `json:"staged"`
	Unstaged []FileChange `json:"unstaged"`
}

// FileChange represents a changed file.
type FileChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// GitBranchResult contains git branch info.
type GitBranchResult struct {
	Current  string   `json:"current"`
	Branches []string `json:"branches"`
}

func (t *GitInfoTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	action := GetString(params, "action", "")
	if action == "" {
		return NewErrorResult("action is required (log, blame, diff, status, branch)", time.Since(start))
	}

	switch action {
	case "log":
		return t.getLog(params, start)
	case "blame":
		return t.getBlame(params, start)
	case "diff":
		return t.getDiff(params, start)
	case "status":
		return t.getStatus(start)
	case "branch":
		return t.getBranch(start)
	default:
		return NewErrorResult("unknown action: "+action, time.Since(start))
	}
}

func (t *GitInfoTool) getLog(params map[string]any, start time.Time) ToolResult {
	count := GetInt(params, "count", 10)
	ref := GetString(params, "ref", "HEAD")

	cmd := "git log --pretty=format:'%h|%an|%ad|%s' --date=short -n " + strconv.Itoa(count) + " " + ref
	result := executor.ExecuteSimple(cmd)

	if result.ExitCode != 0 {
		return NewErrorResult("git log failed: "+result.Output, time.Since(start))
	}

	commits := make([]GitCommit, 0)
	for _, line := range strings.Split(result.Output, "\n") {
		line = strings.Trim(line, "'")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, GitCommit{
				Hash:    parts[0],
				Author:  parts[1],
				Date:    parts[2],
				Subject: parts[3],
			})
		}
	}

	return NewResult(GitLogResult{Commits: commits, Count: len(commits)}, time.Since(start))
}

func (t *GitInfoTool) getBlame(params map[string]any, start time.Time) ToolResult {
	path := GetString(params, "path", "")
	if path == "" {
		return NewErrorResult("path is required for blame action", time.Since(start))
	}

	cmd := "git blame --line-porcelain " + path
	result := executor.ExecuteSimple(cmd)

	if result.ExitCode != 0 {
		return NewErrorResult("git blame failed: "+result.Output, time.Since(start))
	}

	lines := parseBlameOutput(result.Output)

	return NewResult(GitBlameResult{Path: path, Lines: lines}, time.Since(start))
}

func parseBlameOutput(output string) []BlameLine {
	var lines []BlameLine
	var current BlameLine
	lineNum := 0

	for _, line := range strings.Split(output, "\n") {
		if len(line) >= 40 && !strings.HasPrefix(line, "\t") {

			parts := strings.Fields(line)
			if len(parts) >= 1 {
				current.Hash = parts[0][:8]
			}
		} else if strings.HasPrefix(line, "author ") {
			current.Author = strings.TrimPrefix(line, "author ")
		} else if strings.HasPrefix(line, "\t") {
			lineNum++
			current.LineNum = lineNum
			current.Content = strings.TrimPrefix(line, "\t")
			lines = append(lines, current)
			current = BlameLine{}
		}
	}

	return lines
}

func (t *GitInfoTool) getDiff(params map[string]any, start time.Time) ToolResult {
	ref := GetString(params, "ref", "HEAD")

	diffResult := executor.ExecuteSimple("git diff " + ref)

	statsResult := executor.ExecuteSimple("git diff --stat " + ref)

	changedResult := executor.ExecuteSimple("git diff --name-only " + ref + " | wc -l")
	changed, _ := strconv.Atoi(strings.TrimSpace(changedResult.Output))

	return NewResult(GitDiffResult{
		Ref:     ref,
		Diff:    diffResult.Output,
		Stats:   statsResult.Output,
		Changed: changed,
	}, time.Since(start))
}

func (t *GitInfoTool) getStatus(start time.Time) ToolResult {

	branchResult := executor.ExecuteSimple("git branch --show-current")
	branch := strings.TrimSpace(branchResult.Output)

	statusResult := executor.ExecuteSimple("git status --porcelain")

	staged := make([]FileChange, 0)
	unstaged := make([]FileChange, 0)

	for _, line := range strings.Split(statusResult.Output, "\n") {
		if len(line) < 3 {
			continue
		}
		indexStatus := line[0]
		workStatus := line[1]
		path := strings.TrimSpace(line[3:])

		if indexStatus != ' ' && indexStatus != '?' {
			staged = append(staged, FileChange{
				Status: string(indexStatus),
				Path:   path,
			})
		}
		if workStatus != ' ' {
			unstaged = append(unstaged, FileChange{
				Status: string(workStatus),
				Path:   path,
			})
		}
	}

	return NewResult(GitStatusResult{
		Branch:   branch,
		Clean:    len(staged) == 0 && len(unstaged) == 0,
		Staged:   staged,
		Unstaged: unstaged,
	}, time.Since(start))
}

func (t *GitInfoTool) getBranch(start time.Time) ToolResult {

	currentResult := executor.ExecuteSimple("git branch --show-current")
	current := strings.TrimSpace(currentResult.Output)

	branchesResult := executor.ExecuteSimple("git branch --format='%(refname:short)'")

	branches := make([]string, 0)
	for _, line := range strings.Split(branchesResult.Output, "\n") {
		line = strings.Trim(strings.TrimSpace(line), "'")
		if line != "" {
			branches = append(branches, line)
		}
	}

	return NewResult(GitBranchResult{
		Current:  current,
		Branches: branches,
	}, time.Since(start))
}
