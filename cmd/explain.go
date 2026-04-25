package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/config"
	"dev-cli/internal/errordb"
	"dev-cli/internal/executor"
	"dev-cli/internal/llm"
	"dev-cli/internal/storage"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	explainCommand     string
	explainExitCode    int
	explainOutput      string
	explainInteractive bool
	explainLast        int
	explainFilter      string
	explainSince       string
	explainJSON        bool
	explainContext     int
)

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain why the last command failed",
	Long: `Analyze command failures using AI to understand the root cause and get fix suggestions.
Reads from your command history (requires shell integration via 'dev-cli init zsh').`,
	Example: `  # Analyze the last failed command
  dev-cli explain

  # Analyze last 3 failures
  dev-cli explain --last 3

  # Filter by keyword and time
  dev-cli explain --filter npm --since 1h

  # Interactive: run the suggested fix directly
  dev-cli explain -i

  # Include surrounding commands for context
  dev-cli explain --context 3

  # Machine-readable JSON output
  dev-cli explain --json`,
	Aliases: []string{"why", "rca"},
	Run: func(cmd *cobra.Command, args []string) {
		if explainInteractive && !term.IsTerminal(int(os.Stdin.Fd())) {
			return
		}

		if explainLast > 0 || explainFilter != "" || explainSince != "" || explainCommand == "" {
			analyzeFromLog(explainLast, explainFilter, explainSince, explainInteractive)
			return
		}

		if explainExitCode == 130 {
			return
		}
		analyzeEntry(storage.LogEntry{
			Command:  explainCommand,
			ExitCode: explainExitCode,
			Output:   explainOutput,
		}, explainInteractive)
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)

	explainCmd.Flags().StringVar(&explainCommand, "command", "", "The failed command")
	explainCmd.Flags().IntVar(&explainExitCode, "exit-code", 0, "Exit code of the command")
	explainCmd.Flags().StringVar(&explainOutput, "output", "", "Command output")
	explainCmd.Flags().BoolVarP(&explainInteractive, "interactive", "i", false, "Interactive mode with fix prompts")

	explainCmd.Flags().IntVarP(&explainLast, "last", "l", 0, "Analyze last N failures from log")
	explainCmd.Flags().StringVarP(&explainFilter, "filter", "f", "", "Filter by command keyword (npm, prisma, etc)")
	explainCmd.Flags().StringVarP(&explainSince, "since", "s", "", "Filter by time (1h, 30m, etc)")
	explainCmd.Flags().BoolVar(&explainJSON, "json", false, "Output results as JSON for programmatic use")
	explainCmd.Flags().IntVarP(&explainContext, "context", "c", 0, "Include N surrounding commands for context")
}

// ExplainResult is the JSON output format for --json mode.
type ExplainResult struct {
	Command      string               `json:"command"`
	ExitCode     int                  `json:"exit_code"`
	Explanation  string               `json:"explanation"`
	Fix          string               `json:"fix,omitempty"`
	Source       string               `json:"source"`       // "pattern_db" or "llm"
	PatternMatch *errordb.Pattern     `json:"pattern,omitempty"` // When source is pattern_db
	Context      []ContextCommand     `json:"context,omitempty"` // Surrounding commands
}

// ContextCommand is a surrounding command for context.
type ContextCommand struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Ago      string `json:"ago"`
}

func analyzeFromLog(limit int, filterStr, sinceStr string, interactive bool) {
	db := storage.DB()

	var sinceDur time.Duration
	if sinceStr != "" {
		var err error
		sinceDur, err = time.ParseDuration(sinceStr)
		if err != nil {
			printWarning(fmt.Sprintf("Invalid duration: %v", err))
			return
		}
	}

	if limit == 0 {
		limit = 1
	}

	items, err := storage.GetFailures(db, storage.QueryOpts{
		Limit:  limit,
		Filter: filterStr,
		Since:  sinceDur,
	})
	if err != nil {
		printWarning(fmt.Sprintf("Failed to read history: %v", err))
		return
	}

	if len(items) == 0 {
		fmt.Println("No failures found matching criteria")
		return
	}

	for _, item := range items {
		var details map[string]interface{}
		output := ""
		if item.Details != "" {
			if err := json.Unmarshal([]byte(item.Details), &details); err == nil {
				if out, ok := details["output"].(string); ok {
					output = out
				}
			}
		}

		analyzeEntry(storage.LogEntry{
			Command:  item.Command,
			ExitCode: item.ExitCode,
			Output:   output,
		}, interactive)
	}
}

// gatherContext fetches surrounding commands from history for richer analysis.
func gatherContext(n int) []ContextCommand {
	if n <= 0 {
		return nil
	}

	db := storage.DB()
	if db == nil {
		return nil
	}

	items, err := storage.GetRecentHistory(db, n+1) // +1 because current might be included
	if err != nil || len(items) == 0 {
		return nil
	}

	var ctx []ContextCommand
	for _, item := range items {
		ago := time.Since(item.Timestamp).Truncate(time.Second).String()
		ctx = append(ctx, ContextCommand{
			Command:  item.Command,
			ExitCode: item.ExitCode,
			Ago:      ago + " ago",
		})
	}
	return ctx
}

func analyzeEntry(entry storage.LogEntry, interactive bool) {
	// Gather surrounding context if requested
	contextCmds := gatherContext(explainContext)

	// ── Step 1: Check the local error pattern database for instant results ──
	if pattern, ok := errordb.Lookup(entry.Command, entry.Output); ok {
		if explainJSON {
			result := ExplainResult{
				Command:      entry.Command,
				ExitCode:     entry.ExitCode,
				Explanation:  pattern.Explanation,
				Fix:          pattern.Fix,
				Source:       "pattern_db",
				PatternMatch: &pattern,
				Context:      contextCmds,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(result)
			return
		}

		fmt.Printf("\n%s %s %s(exit %d)%s\n", iconFail(), entry.Command, colorGray, entry.ExitCode, colorReset)
		fmt.Printf("  %s %s %s[%s]%s\n", iconInfo(), pattern.Explanation, colorGray, pattern.Category, colorReset)
		if pattern.Fix != "" {
			fmt.Printf("  %s$%s %s\n", colorGreen, colorReset, pattern.Fix)
		}
		printInfo(fmt.Sprintf("(instant match from %d-pattern database)", errordb.Count()))

		// Show context if requested
		if len(contextCmds) > 0 {
			fmt.Printf("\n  %sContext (last %d commands):%s\n", colorGray, len(contextCmds), colorReset)
			for _, c := range contextCmds {
				exitBadge := fmt.Sprintf("%s✓%s", colorGreen, colorReset)
				if c.ExitCode != 0 {
					exitBadge = fmt.Sprintf("%s✗ %d%s", colorRed, c.ExitCode, colorReset)
				}
				fmt.Printf("    %s %s %s(%s)%s\n", exitBadge, c.Command, colorGray, c.Ago, colorReset)
			}
		}

		if interactive && pattern.Fix != "" {
			promptAndRunFix(pattern.Fix)
		}
		return
	}

	// ── Step 2: Fall back to LLM analysis ──────────────────────────────────
	if !explainJSON {
		fmt.Printf("\n%s %s %s(exit %d)%s\n", iconFail(), entry.Command, colorGray, entry.ExitCode, colorReset)
	}

	if err := llm.EnsureOllamaRunning(); err != nil {
		printWarning(fmt.Sprintf("Ollama not available: %v", err))
		return
	}

	s := newSpinner("Analyzing failure...")

	cfg := config.Load()
	client := llm.NewClient(cfg)

	// Offline mode indicator
	ollamaStatus := llm.CheckOllamaStatus()
	modeBadge := fmt.Sprintf("%s⚡ local%s", colorGreen, colorReset)
	if !ollamaStatus.Running {
		modeBadge = fmt.Sprintf("%s⚠ offline%s", colorYellow, colorReset)
	}
	if !explainJSON {
		fmt.Printf("\r  [%s] ", modeBadge)
	}

	// Build prompt with context if available
	prompt := entry.Command
	if len(contextCmds) > 0 {
		var ctxLines []string
		for _, c := range contextCmds {
			ctxLines = append(ctxLines, fmt.Sprintf("$ %s (exit %d, %s)", c.Command, c.ExitCode, c.Ago))
		}
		prompt = fmt.Sprintf("Context (recent commands):\n%s\n\nFailed command: %s",
			strings.Join(ctxLines, "\n"), entry.Command)
	}

	result, err := client.Explain(prompt, entry.ExitCode, entry.Output)
	s.Stop()

	if err != nil {
		printWarning(fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	if explainJSON {
		jsonResult := ExplainResult{
			Command:     entry.Command,
			ExitCode:    entry.ExitCode,
			Explanation: result.Explanation,
			Fix:         result.Fix,
			Source:      "llm",
			Context:     contextCmds,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(jsonResult)
		return
	}

	fmt.Printf("  %s %s\n", iconInfo(), result.Explanation)

	// Show context if requested
	if len(contextCmds) > 0 {
		fmt.Printf("\n  %sContext (last %d commands):%s\n", colorGray, len(contextCmds), colorReset)
		for _, c := range contextCmds {
			exitBadge := fmt.Sprintf("%s✓%s", colorGreen, colorReset)
			if c.ExitCode != 0 {
				exitBadge = fmt.Sprintf("%s✗ %d%s", colorRed, c.ExitCode, colorReset)
			}
			fmt.Printf("    %s %s %s(%s)%s\n", exitBadge, c.Command, colorGray, c.Ago, colorReset)
		}
	}

	if result.Fix != "" {
		fmt.Printf("  %s$%s %s\n", colorGreen, colorReset, result.Fix)

		if interactive {
			promptAndRunFix(result.Fix)
		}
	}
}

// promptAndRunFix handles the interactive fix execution flow.
func promptAndRunFix(fix string) {
	if pattern := executor.IsDangerousCommand(fix); pattern != "" {
		fmt.Fprintf(os.Stderr, "   %sWARNING: Potentially dangerous command detected (%s)%s\n", colorRed, pattern, colorReset)
		fmt.Print("   This command could cause data loss. Are you SURE? (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(response)) != "yes" {
			fmt.Println("   Aborted.")
			return
		}
	}

	fmt.Print("   [Run Fix?] (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		fmt.Printf("   Running: %s\n", fix)
		cmd := exec.Command("sh", "-c", fix)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			printWarning(fmt.Sprintf("Fix failed: %v", err))
		} else {
			printSuccess("Fix applied")
		}
	}
}
