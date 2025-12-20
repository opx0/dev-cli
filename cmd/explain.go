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
	"dev-cli/internal/llm"
	"dev-cli/internal/storage"

	"github.com/briandowns/spinner"

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
	db, err := storage.InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to open db: %v\n", err)
		return
	}
	defer db.Close()

	var sinceDur time.Duration
	if sinceStr != "" {
		sinceDur, err = time.ParseDuration(sinceStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Invalid duration: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "⚠️  Failed to read history: %v\n", err)
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
	fmt.Printf("\n\033[31m×\033[0m %s \033[90m(exit %d)\033[0m\n", entry.Command, entry.ExitCode)

	if err := llm.EnsureOllamaRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[33m⚠\033[0m Ollama not available: %v\n", err)
		return
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " 🧠 Analyzing failure..."
	s.Start()

	client := llm.NewClient(config.Load())
	result, err := client.Explain(entry.Command, entry.ExitCode, entry.Output)
	s.Stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[33m⚠\033[0m Analysis failed: %v\n", err)
		return
	}

	fmt.Printf("  \033[90m→\033[0m %s\n", result.Explanation)

	if result.Fix != "" {
		fmt.Printf("  \033[32m$\033[0m %s\n", result.Fix)

		if interactive {
			dangerousPatterns := []string{"rm -rf", "rm -r /", "dd if=", "mkfs", "> /dev/", "chmod 777", ":(){ :|:& };:"}
			for _, pattern := range dangerousPatterns {
				if strings.Contains(result.Fix, pattern) {
					fmt.Fprintf(os.Stderr, "   \033[31m⚠ WARNING: Potentially dangerous command detected (%s)\033[0m\n", pattern)
					fmt.Print("   This command could cause data loss. Are you SURE? (yes/no): ")
					reader := bufio.NewReader(os.Stdin)
					response, _ := reader.ReadString('\n')
					if strings.TrimSpace(strings.ToLower(response)) != "yes" {
						fmt.Println("   Aborted.")
						return
					}
					break
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
					fmt.Fprintf(os.Stderr, "   \033[33m⚠\033[0m Fix failed: %v\n", err)
				} else {
					fmt.Println("   \033[32m✓\033[0m Fix applied")
				}
			}
		}
	}
}
