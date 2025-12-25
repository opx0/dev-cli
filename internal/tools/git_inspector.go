package tools

import (
	"context"
	"strconv"
	"strings"
	"time"

	"dev-cli/internal/executor"
)

// GitInspectorTool gathers git context for error diagnosis.
// Runs git status and git log -n 5 to provide repository context.
type GitInspectorTool struct{}

func (t *GitInspectorTool) Name() string { return "git_inspector" }
func (t *GitInspectorTool) Description() string {
	return "Gather git repository context: status and recent commits for error diagnosis"
}

func (t *GitInspectorTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "commit_count", Type: "int", Description: "Number of recent commits to include", Required: false, Default: 5},
	}
}

// GitInspectorResult contains combined git status and log output.
type GitInspectorResult struct {
	InGitRepo     bool         `json:"in_git_repo"`
	Branch        string       `json:"branch,omitempty"`
	Clean         bool         `json:"clean"`
	Staged        []FileChange `json:"staged,omitempty"`
	Unstaged      []FileChange `json:"unstaged,omitempty"`
	RecentCommits []GitCommit  `json:"recent_commits,omitempty"`
	Error         string       `json:"error,omitempty"`
}

func (t *GitInspectorTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	commitCount := GetInt(params, "commit_count", 5)

	checkResult := executor.ExecuteSimple("git rev-parse --is-inside-work-tree")
	if checkResult.ExitCode != 0 {
		return NewResult(GitInspectorResult{
			InGitRepo: false,
			Error:     "not a git repository",
		}, time.Since(start))
	}

	result := GitInspectorResult{InGitRepo: true}

	branchResult := executor.ExecuteSimple("git branch --show-current")
	result.Branch = strings.TrimSpace(branchResult.Output)

	statusResult := executor.ExecuteSimple("git status --porcelain")
	result.Clean = strings.TrimSpace(statusResult.Output) == ""

	for _, line := range strings.Split(statusResult.Output, "\n") {
		if len(line) < 3 {
			continue
		}
		indexStatus := line[0]
		workStatus := line[1]
		path := strings.TrimSpace(line[3:])

		if indexStatus != ' ' && indexStatus != '?' {
			result.Staged = append(result.Staged, FileChange{
				Status: string(indexStatus),
				Path:   path,
			})
		}
		if workStatus != ' ' {
			result.Unstaged = append(result.Unstaged, FileChange{
				Status: string(workStatus),
				Path:   path,
			})
		}
	}

	logCmd := "git log --pretty=format:'%h|%an|%ad|%s' --date=short -n " + strconv.Itoa(commitCount)
	logResult := executor.ExecuteSimple(logCmd)

	if logResult.ExitCode == 0 {
		for _, line := range strings.Split(logResult.Output, "\n") {
			line = strings.Trim(line, "'")
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 4)
			if len(parts) == 4 {
				result.RecentCommits = append(result.RecentCommits, GitCommit{
					Hash:    parts[0],
					Author:  parts[1],
					Date:    parts[2],
					Subject: parts[3],
				})
			}
		}
	}

	return NewResult(result, time.Since(start))
}

// InspectOnError is a helper that returns git context when a command fails.
// This can be called after command execution to enrich error context.
func InspectOnError(exitCode int) *GitInspectorResult {
	if exitCode == 0 {
		return nil
	}

	tool := &GitInspectorTool{}
	result := tool.Execute(context.Background(), map[string]any{"commit_count": 5})
	if result.Success {
		if data, ok := result.Data.(GitInspectorResult); ok {
			return &data
		}
	}
	return nil
}
