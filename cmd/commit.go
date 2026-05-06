package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"dev-cli/internal/config"
	"dev-cli/internal/llm"
	"dev-cli/internal/memory"

	"github.com/spf13/cobra"
)

var (
	commitWithTest bool
	commitAuto     bool
	commitAmend    bool
	commitAll      bool
	commitScope    string
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Stage-aware AI commit message generator",
	Long: `Read the currently staged diff, generate a Conventional Commits
message with the configured LLM, and commit. The proposed message is shown
for review (unless --auto) with options to accept, edit, or abort.

Never pushes. --all stages modified (non-untracked) files first.`,
	Example: `  git add -p
  dev-cli commit
  dev-cli commit --scope auth --test
  dev-cli commit --all --auto`,
	RunE: runCommit,
}

func init() {
	commitCmd.Flags().BoolVar(&commitWithTest, "test", false, "Run dev-cli test first; abort commit on failure")
	commitCmd.Flags().BoolVar(&commitAuto, "auto", false, "Accept the generated message without prompting")
	commitCmd.Flags().BoolVar(&commitAmend, "amend", false, "Amend the previous commit instead of creating a new one")
	commitCmd.Flags().BoolVarP(&commitAll, "all", "a", false, "Stage all tracked modifications (git add -u) before committing")
	commitCmd.Flags().StringVar(&commitScope, "scope", "", "Force scope in the Conventional Commits header (e.g. --scope auth)")
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found on PATH")
	}
	if out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").CombinedOutput(); err != nil {
		return fmt.Errorf("not a git repository: %s", strings.TrimSpace(string(out)))
	}

	if commitWithTest {
		printInfo("Running tests before committing…")
		bin, runnerArgs, label := detectTestRunner()
		if bin == "" {
			return fmt.Errorf("--test given but no runner detected")
		}
		exit, _, err := streamCommand(bin, runnerArgs, true)
		if err != nil || exit != 0 {
			return fmt.Errorf("%s tests failed (exit %d); aborting commit", label, exit)
		}
		printSuccess("Tests passed")
	}

	if commitAll {
		if out, err := exec.Command("git", "add", "-u").CombinedOutput(); err != nil {
			return fmt.Errorf("git add -u: %s", strings.TrimSpace(string(out)))
		}
	}

	diff, err := captureStagedDiff()
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no staged changes (use `git add` or --all)")
	}

	message, err := generateCommitMessage(diff, commitScope)
	if err != nil {
		return fmt.Errorf("generate message: %w", err)
	}

	if !commitAuto {
		fmt.Printf("\n%s Proposed commit message:%s\n", colorBold, colorReset)
		fmt.Println(indentEach(message, "    "))
		fmt.Print("\n[A]ccept  [E]dit  [N]o: ")
		var resp string
		fmt.Scanln(&resp)
		switch strings.ToLower(strings.TrimSpace(resp)) {
		case "", "a", "accept", "y", "yes":
			// accept
		case "e", "edit":
			edited, err := editMessageInteractively(message)
			if err != nil {
				return err
			}
			message = edited
		default:
			return fmt.Errorf("aborted")
		}
	}

	gitArgs := []string{"commit", "-m", message}
	if commitAmend {
		gitArgs = append(gitArgs, "--amend")
	}
	c := exec.Command("git", gitArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	printSuccess("Committed")
	return nil
}

// captureStagedDiff returns `git diff --staged` truncated to a token-safe size.
func captureStagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--staged", "--no-color").Output()
	if err != nil {
		return "", fmt.Errorf("git diff --staged: %w", err)
	}
	const maxDiffBytes = 24 * 1024 // ~6k tokens; safe across all providers
	if len(out) > maxDiffBytes {
		out = append(out[:maxDiffBytes], []byte("\n... (diff truncated)\n")...)
	}
	return string(out), nil
}

func generateCommitMessage(diff, scope string) (string, error) {
	cfg := config.Load()
	client := llm.NewHybridClient()

	scopeHint := ""
	if scope != "" {
		scopeHint = fmt.Sprintf("Use scope %q.", scope)
	}
	memBlock := ""
	if mc, ok := memory.BuildPromptContext(cfg, "commit message style for: "+firstLines(diff, 10)); ok {
		memBlock = mc + "\n\n"
	}
	prompt := memBlock + fmt.Sprintf(`You are writing a Conventional Commits message for a staged diff.

Rules:
- First line: "<type>(<scope>): <subject>" (scope optional). Max 72 chars. Imperative mood. No trailing period.
- Allowed types: feat, fix, refactor, perf, test, docs, chore, build, ci, style, revert.
- If the diff warrants it, add a blank line and a body with ONE short paragraph explaining *why*, not *what*.
- No marketing adjectives, no emoji, no signatures.
%s

OUTPUT: the commit message itself, and nothing else.

DIFF:
%s`, scopeHint, diff)

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    cfg.OllamaModel, // hybrid picks the right provider
		Messages: []llm.Message{llm.UserMsg(prompt)},
	})
	if err != nil {
		return "", err
	}
	msg := strings.TrimSpace(resp.Content)
	msg = strings.TrimPrefix(msg, "```")
	msg = strings.TrimSuffix(msg, "```")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", fmt.Errorf("provider returned empty message")
	}
	return msg, nil
}

func editMessageInteractively(initial string) (string, error) {
	tmp, err := os.CreateTemp("", "devcli-commit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(initial); err != nil {
		return "", err
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, tmp.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func indentEach(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
