package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
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
		args = append(args, rangeArg)
	default:
		return NewErrorResult(fmt.Sprintf("unknown scope %q (use staged|unstaged|range)", scope), time.Since(start))
	}
	if path != "" {
		args = append(args, "--", path)
	}

	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return NewErrorResult(fmt.Sprintf("git %s: %v", strings.Join(args, " "), err), time.Since(start))
	}

	diff := string(out)
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
		"bytes":     len(out),
	}, time.Since(start))
}
