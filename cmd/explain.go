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
  dev-cli explain -i`,
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

func analyzeEntry(entry storage.LogEntry, interactive bool) {
	fmt.Printf("\n%s %s %s(exit %d)%s\n", iconFail(), entry.Command, colorGray, entry.ExitCode, colorReset)

	if err := llm.EnsureOllamaRunning(); err != nil {
		printWarning(fmt.Sprintf("Ollama not available: %v", err))
		return
	}

	s := newSpinner("Analyzing failure...")

	cfg := config.Load()
	client := llm.NewClient(cfg)
	result, err := client.Explain(entry.Command, entry.ExitCode, entry.Output)
	s.Stop()

	if err != nil {
		printWarning(fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	fmt.Printf("  %s %s\n", iconInfo(), result.Explanation)

	if result.Fix != "" {
		fmt.Printf("  %s$%s %s\n", colorGreen, colorReset, result.Fix)

		if interactive {
			if pattern := executor.IsDangerousCommand(result.Fix); pattern != "" {
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
				fmt.Printf("   Running: %s\n", result.Fix)
				cmd := exec.Command("sh", "-c", result.Fix)
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
	}
}
