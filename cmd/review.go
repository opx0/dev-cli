package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"dev-cli/internal/config"
	"dev-cli/internal/llm"

	"github.com/spf13/cobra"
)

var (
	reviewSince  string
	reviewPR     string
	reviewFormat string
	reviewPath   string
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "LLM review of the current diff with structured findings",
	Long: `Run an AI code review over a diff and emit structured findings
(severity, file:line, rationale, suggestion). Defaults to staged changes.

Input selection (first match wins):
  --pr <ref>      review the diff introduced by that PR (branch-to-main)
  --since <ref>   review commits since <ref> (e.g. origin/main)
  (default)       review staged changes

Formats: text (default, coloured), json (for piping into tooling).`,
	Example: `  dev-cli review
  dev-cli review --since origin/main
  dev-cli review --pr feat/auth --format json`,
	RunE: runReview,
}

func init() {
	reviewCmd.Flags().StringVar(&reviewSince, "since", "", "Review commits since this ref (e.g. origin/main)")
	reviewCmd.Flags().StringVar(&reviewPR, "pr", "", "Review diff for this PR branch (vs merge-base with main)")
	reviewCmd.Flags().StringVar(&reviewFormat, "format", "text", "Output format: text, json")
	reviewCmd.Flags().StringVar(&reviewPath, "path", "", "Limit review to this path")
	rootCmd.AddCommand(reviewCmd)
}

type reviewFinding struct {
	Severity   string `json:"severity"` // "bug" | "risk" | "nit"
	File       string `json:"file"`
	Line       int    `json:"line"`
	Rationale  string `json:"rationale"`
	Suggestion string `json:"suggestion,omitempty"`
}

type reviewResult struct {
	Scope    string          `json:"scope"`
	Findings []reviewFinding `json:"findings"`
	Summary  string          `json:"summary,omitempty"`
}

func runReview(cmd *cobra.Command, args []string) error {
	diff, scopeLabel, err := collectReviewDiff()
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no diff to review (scope=%s)", scopeLabel)
	}

	cfg := config.Load()
	client := llm.NewHybridClient()

	prompt := fmt.Sprintf(`You are an experienced code reviewer. Review the following diff and return STRICT JSON only.

Rules:
- Findings should be concrete and actionable. Skip style nitpicks that a linter would flag.
- Severity is one of: "bug" (will fail at runtime or misbehaves), "risk" (edge cases, silent data loss, perf cliff), "nit" (minor readability).
- Keep rationale ONE sentence. Suggestion is optional and should be a specific change, not general advice.
- If the diff looks clean, return {"findings":[],"summary":"LGTM"}.

Schema:
{
  "summary": "one-sentence overall take",
  "findings": [
    {"severity": "bug|risk|nit", "file": "path/to/file", "line": 123, "rationale": "...", "suggestion": "..."}
  ]
}

DIFF (scope=%s):
%s`, scopeLabel, diff)

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    cfg.OllamaModel,
		Messages: []llm.Message{llm.UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return fmt.Errorf("review LLM call: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result reviewResult
	result.Scope = scopeLabel
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return fmt.Errorf("reviewer returned non-JSON; raw: %s", content)
	}

	if reviewFormat == "json" {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	renderReviewText(result)
	return nil
}

func collectReviewDiff() (diff, scope string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", fmt.Errorf("git not found on PATH")
	}

	args := []string{"diff", "--no-color"}
	scopeLabel := "staged"
	switch {
	case reviewPR != "":
		base, err := exec.Command("git", "merge-base", "origin/main", reviewPR).Output()
		if err != nil {
			// Fallback: try main without origin.
			base, err = exec.Command("git", "merge-base", "main", reviewPR).Output()
			if err != nil {
				return "", "", fmt.Errorf("could not find merge base for --pr %s", reviewPR)
			}
		}
		baseRef := strings.TrimSpace(string(base))
		args = append(args, fmt.Sprintf("%s..%s", baseRef, reviewPR))
		scopeLabel = fmt.Sprintf("pr:%s", reviewPR)
	case reviewSince != "":
		args = append(args, reviewSince+"..HEAD")
		scopeLabel = "since:" + reviewSince
	default:
		args = append(args, "--staged")
	}
	if reviewPath != "" {
		args = append(args, "--", reviewPath)
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	const maxDiffBytes = 48 * 1024
	if len(out) > maxDiffBytes {
		out = append(out[:maxDiffBytes], []byte("\n... (diff truncated)\n")...)
	}
	return string(out), scopeLabel, nil
}

func renderReviewText(r reviewResult) {
	if r.Summary != "" {
		fmt.Printf("%s %s%s%s\n\n", iconInfo(), colorBold, r.Summary, colorReset)
	}
	if len(r.Findings) == 0 {
		printSuccess("No findings")
		return
	}
	bugs, risks, nits := 0, 0, 0
	for _, f := range r.Findings {
		switch f.Severity {
		case "bug":
			bugs++
		case "risk":
			risks++
		default:
			nits++
		}
	}
	fmt.Printf("Findings: %s%d bug%s · %s%d risk%s · %s%d nit%s\n\n",
		colorRed, bugs, colorReset, colorYellow, risks, colorReset, colorGray, nits, colorReset)

	for _, f := range r.Findings {
		color := colorGray
		switch f.Severity {
		case "bug":
			color = colorRed
		case "risk":
			color = colorYellow
		}
		fmt.Printf("%s[%s]%s %s:%d\n", color, strings.ToUpper(f.Severity), colorReset, f.File, f.Line)
		fmt.Printf("    %s\n", f.Rationale)
		if f.Suggestion != "" {
			fmt.Printf("    %s→ %s%s\n", colorCyan, f.Suggestion, colorReset)
		}
		fmt.Println()
	}
}
