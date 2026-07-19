package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dev-cli/internal/executor"
)

// ReadDiffTool exposes `git diff` to the agent so it can pull more context
// while reviewing changes (staged, unstaged, or a ref range).
type ReadDiffTool struct{}

var _ Tool = (*ReadDiffTool)(nil)

func (ReadDiffTool) Name() string { return "read_diff" }

func (ReadDiffTool) Description() string {
	return "Read a git diff: staged changes, working-tree changes, or a commit range (ref1..ref2)."
}

func (ReadDiffTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "scope", Type: "string", Description: "One of: staged, unstaged, range", Required: true},
		{Name: "range", Type: "string", Description: "Commit range when scope=range, e.g. main..HEAD", Required: false},
		{Name: "path", Type: "string", Description: "Optional path filter", Required: false},
		{Name: "max_bytes", Type: "int", Description: "Truncate the diff to this many bytes (default 24000)", Default: 24000},
	}
}

func (ReadDiffTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()
	scope := GetString(params, "scope", "staged")
	rangeArg := GetString(params, "range", "")
	path := GetString(params, "path", "")
	maxBytes := GetInt(params, "max_bytes", 24000)
	if maxBytes < 1 {
		maxBytes = 1
	} else if maxBytes > 1024*1024 {
		maxBytes = 1024 * 1024
	}

	args := []string{"diff", "--no-color"}
	switch scope {
	case "staged":
		args = append(args, "--staged")
	case "unstaged":
		// default diff without --staged is unstaged
	case "range":
		if rangeArg == "" {
			return NewErrorResult("scope=range requires range parameter (e.g. main..HEAD)", time.Since(start))
		}
		if strings.HasPrefix(rangeArg, "-") {
			return NewErrorResult("git range cannot start with '-'", time.Since(start))
		}
		args = append(args, rangeArg)
	default:
		return NewErrorResult(fmt.Sprintf("unknown scope %q (use staged|unstaged|range)", scope), time.Since(start))
	}
	if path != "" {
		resolved, err := executor.ResolvePath(path)
		if err != nil {
			return NewErrorResult("invalid diff path: "+err.Error(), time.Since(start))
		}
		if check := executor.CheckFileSafety(resolved); !check.IsSafe {
			return NewErrorResult("blocked diff path: "+check.Reason, time.Since(start))
		}
		args = append(args, "--", path)
	}

	commandResult := executor.ExecuteProgram(ctx, "", "git", args...)
	if commandResult.ExitCode != 0 {
		return NewErrorResult(fmt.Sprintf("git %s: %s", strings.Join(args, " "), commandResult.Output), time.Since(start))
	}

	diff := commandResult.Output
	truncated := false
	if len(diff) > maxBytes {
		diff = diff[:maxBytes]
		truncated = true
	}

	return NewResult(map[string]any{
		"scope":     scope,
		"range":     rangeArg,
		"path":      path,
		"diff":      diff,
		"truncated": truncated,
		"bytes":     len(commandResult.Output),
	}, time.Since(start))
}
