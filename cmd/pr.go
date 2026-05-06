package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/config"
	"dev-cli/internal/llm"
	"dev-cli/internal/memory"

	"github.com/spf13/cobra"
)

var (
	prPush     bool
	prBase     string
	prDraft    bool
	prTitle    string
	prBody     string
	prAuto     bool
	prWithTest bool
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Draft a GitHub PR (title + body) from the current branch",
	Long: `Draft title and body for a pull request from the commits on the
current branch vs --base (default: origin/main). Uses the configured LLM.

Defaults to DRY-RUN: prints the proposed title/body without pushing.
Add --push to actually push the branch and open the PR via the gh CLI.

Never force-pushes.`,
	Example: `  dev-cli pr                         # preview only
  dev-cli pr --push                  # push branch + gh pr create
  dev-cli pr --push --draft --test`,
	RunE: runPR,
}

func init() {
	prCmd.Flags().BoolVar(&prPush, "push", false, "Push the branch and call gh pr create")
	prCmd.Flags().StringVar(&prBase, "base", "origin/main", "Base branch to diff against")
	prCmd.Flags().BoolVar(&prDraft, "draft", false, "Open PR as draft (requires --push)")
	prCmd.Flags().StringVar(&prTitle, "title", "", "Override the generated title")
	prCmd.Flags().StringVar(&prBody, "body", "", "Override the generated body")
	prCmd.Flags().BoolVar(&prAuto, "auto", false, "Skip the review prompt (with --push, creates the PR without confirmation)")
	prCmd.Flags().BoolVar(&prWithTest, "test", false, "Run tests before pushing; abort on failure")
	rootCmd.AddCommand(prCmd)
}

func runPR(cmd *cobra.Command, args []string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found on PATH")
	}

	branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("git rev-parse: %w", err)
	}
	branchName := strings.TrimSpace(string(branch))
	if branchName == "HEAD" || branchName == "main" || branchName == "master" {
		return fmt.Errorf("current branch is %q — create a feature branch first", branchName)
	}

	// Resolve merge base with configured base.
	base, err := exec.Command("git", "merge-base", prBase, "HEAD").Output()
	if err != nil {
		return fmt.Errorf("git merge-base %s HEAD: %w", prBase, err)
	}
	baseRef := strings.TrimSpace(string(base))

	// Commit titles in the range (not full diff — keeps token use low).
	commits, err := exec.Command("git", "log", "--pretty=format:%s%n%b%n---", baseRef+"..HEAD").Output()
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	if strings.TrimSpace(string(commits)) == "" {
		return fmt.Errorf("no commits on %s vs %s", branchName, prBase)
	}

	// Stats for the body context.
	stats, _ := exec.Command("git", "diff", "--stat", baseRef+"..HEAD").Output()

	title, body := prTitle, prBody
	if title == "" || body == "" {
		gt, gb, err := generatePRContent(string(commits), string(stats))
		if err != nil {
			return fmt.Errorf("generate PR content: %w", err)
		}
		if title == "" {
			title = gt
		}
		if body == "" {
			body = gb
		}
	}

	fmt.Printf("\n%sBranch:%s %s → %s\n", colorBold, colorReset, branchName, prBase)
	fmt.Printf("%sTitle:%s  %s\n\n", colorBold, colorReset, title)
	fmt.Printf("%sBody:%s\n%s\n", colorBold, colorReset, indentEach(body, "  "))

	if !prPush {
		fmt.Printf("\n%s Dry-run: nothing pushed. Re-run with --push to open the PR.\n", iconInfo())
		return nil
	}

	if prWithTest {
		printInfo("Running tests before push…")
		bin, rargs, label := detectTestRunner()
		if bin == "" {
			return fmt.Errorf("--test given but no runner detected")
		}
		exit, _, err := streamCommand(bin, rargs, true)
		if err != nil || exit != 0 {
			return fmt.Errorf("%s tests failed (exit %d); aborting", label, exit)
		}
		printSuccess("Tests passed")
	}

	if !prAuto {
		fmt.Print("\nPush branch and create PR? [y/N]: ")
		var resp string
		fmt.Scanln(&resp)
		r := strings.ToLower(strings.TrimSpace(resp))
		if r != "y" && r != "yes" {
			return fmt.Errorf("aborted")
		}
	}

	if out, err := exec.Command("git", "push", "-u", "origin", branchName).CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %s", strings.TrimSpace(string(out)))
	}

	if _, err := exec.LookPath("gh"); err != nil {
		printWarning("gh CLI not found — branch pushed but PR not created. Install gh and run: gh pr create")
		return nil
	}

	ghArgs := []string{"pr", "create", "--title", title, "--body", body, "--base", strings.TrimPrefix(prBase, "origin/")}
	if prDraft {
		ghArgs = append(ghArgs, "--draft")
	}
	c := exec.Command("gh", ghArgs...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr create: %s", strings.TrimSpace(string(out)))
	}
	prURL := strings.TrimSpace(string(out))
	fmt.Printf("\n%s %s\n", iconOK(), prURL)

	storePRMemory(title, body, branchName, prURL)
	return nil
}

// storePRMemory writes the accepted PR draft to MemPalace as a `decision`
// memory so future `dev-cli pr` runs on related work can recall the
// established voice and structure. Best-effort; never blocks the CLI.
func storePRMemory(title, body, branch, prURL string) {
	cfg := config.Load()
	if !cfg.MemPalaceEnabled || !cfg.MemPalaceWriteback {
		return
	}
	text := fmt.Sprintf("PR title: %s\nBranch: %s\nURL: %s\n\n%s", title, branch, prURL, body)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = memory.Store(ctx, cfg, memory.StoreReq{
			Hall:   "decision",
			Text:   text,
			Source: "dev-cli/pr",
		})
	}()
}

func generatePRContent(commitsBlock, stats string) (title, body string, err error) {
	cfg := config.Load()
	client := llm.NewHybridClient()

	memBlock := ""
	if mc, ok := memory.BuildPromptContext(cfg, "PR draft for: "+commitsBlock); ok {
		memBlock = mc + "\n\n"
	}

	prompt := memBlock + fmt.Sprintf(`You are drafting a GitHub pull request based on the commit history below.

Return STRICT JSON only with this shape:
{"title": "<short imperative title, no prefix>", "body": "<markdown body>"}

Body structure:
## Summary
One short paragraph describing what changes and why (not how).

## Changes
- 2–5 bullet points. No restating commit messages verbatim.

## Test plan
- [ ] Bulleted checklist of things to verify.

Do NOT invent unrelated changes. Do NOT add emoji or marketing language.

COMMITS:
%s

STATS:
%s`, commitsBlock, stats)

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    cfg.OllamaModel,
		Messages: []llm.Message{llm.UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return "", "", err
	}
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var out struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return "", "", fmt.Errorf("provider returned non-JSON: %s", content)
	}
	if out.Title == "" {
		return "", "", fmt.Errorf("provider returned empty title")
	}
	return out.Title, out.Body, nil
}
