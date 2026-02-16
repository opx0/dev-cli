package cmd

import (
	"dev-cli/internal/config"
	"dev-cli/internal/llm"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	assistCount int
	assistLocal bool
)

var askCmd = &cobra.Command{
	Use:   "ask [query]",
	Short: "Get help with tool commands or solutions",
	Long: `Get AI-powered assistance for DevOps tasks.

Two modes:
  1. Tool Mode   - Pass a tool name to get a cheat sheet of useful commands.
  2. Research    - Ask a natural language question for step-by-step solutions.`,
	Example: `  # Tool Mode: Get common commands for a tool
  dev-cli ask tar
  dev-cli ask kubectl
  dev-cli ask git "undo commits"
  dev-cli ask ffmpeg --count 5        # Get 5 commands

  # Research Mode: Ask a question
  dev-cli ask "how to mount an NTFS drive on Linux"
  dev-cli ask "fix permission denied on docker.sock"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		if assistLocal {
			os.Setenv("DEV_CLI_FORCE_LOCAL", "1")
		}

		if err := llm.EnsureOllamaRunning(); err != nil {
			printWarning(fmt.Sprintf("Ollama not available: %v", err))
		}

		if looksLikeToolName(args) {
			toolName := args[0]
			topic := "important and commonly used"
			if len(args) > 1 {
				topic = strings.Join(args[1:], " ")
			}
			fetchCommands(toolName, topic, assistCount)
		} else {
			fetchSolutions(query)
		}
	},
}

func init() {
	rootCmd.AddCommand(askCmd)
	askCmd.Flags().IntVarP(&assistCount, "count", "n", 10, "Number of commands to show (tool mode)")
	askCmd.Flags().BoolVar(&assistLocal, "local", false, "Force local Ollama (skip Perplexity)")
}

func looksLikeToolName(args []string) bool {
	if len(args) > 3 {
		return false
	}

	questionWords := []string{"how", "what", "why", "when", "where", "which", "can", "should", "is", "are", "do", "does"}
	first := strings.ToLower(args[0])
	for _, qw := range questionWords {
		if first == qw {
			return false
		}
	}

	actionWords := []string{"install", "setup", "configure", "deploy", "create", "build", "run", "start", "stop", "fix", "debug", "undo", "remove", "delete"}
	for _, aw := range actionWords {
		if first == aw {
			return false
		}
	}

	return len(args) == 1 || (len(args) <= 3 && !strings.Contains(strings.Join(args, " "), " to "))
}

func fetchSolutions(query string) {
	client := llm.NewHybridClient()

	backend := "Ollama"
	if client.HasPerplexity() {
		backend = "Perplexity"
	}
	printInfo(fmt.Sprintf("Researching via %s: %s...", backend, query))

	s := newSpinner("Researching...")
	result, err := client.Research(query)
	s.Stop()

	if err != nil {
		printError(fmt.Sprintf("Failed to get solutions: %v", err))
		os.Exit(1)
	}

	if len(result.Solutions) == 0 {
		printWarning("No solutions found")
		return
	}

	fmt.Printf("\n%s Found %d Solutions:%s\n\n", boldGreen, len(result.Solutions), colorReset)

	for _, sol := range result.Solutions {
		fmt.Printf("%s[%d] %s%s\n", boldCyan, sol.ID, sol.Title, colorReset)
		fmt.Printf("    %s%s%s\n\n", colorWhite, sol.Description, colorReset)

		for _, step := range sol.Steps {
			if step.Type == "command" {
				fmt.Printf("    %s$%s %s%s%s\n", colorGray, colorReset, boldYellow, step.Content, colorReset)
				if step.Note != "" {
					fmt.Printf("      %s# %s%s\n", colorGray, step.Note, colorReset)
				}
			} else if step.Type == "file" {
				lines := strings.Split(step.Content, "\n")
				lineCount := len(lines)

				fmt.Printf("    %s# %s (%d lines)%s\n", colorGray, step.File, lineCount, colorReset)
				fmt.Printf("    %s```%s\n", colorGray, colorReset)
				for _, line := range lines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Printf("    %s```%s\n", colorGray, colorReset)

				if step.Note != "" {
					fmt.Printf("    %s# %s%s\n", colorGray, step.Note, colorReset)
				}
			}
		}

		if sol.Source != "" {
			fmt.Printf("\n    %sSource: %s%s\n", colorGray, sol.Source, colorReset)
		}
		fmt.Println()
	}
}

func fetchCommands(toolName, topic string, count int) {
	cfg := config.Load()
	client := llm.NewClient(cfg)

	query := toolName
	if topic != "important and commonly used" {
		query = toolName + " " + topic
	}

	s := newSpinner(fmt.Sprintf("Fetching %s commands...", query))
	result, err := client.CheatSheet(toolName, topic, count)
	s.Stop()

	if err != nil {
		printError(fmt.Sprintf("Failed to fetch commands: %v", err))
		os.Exit(1)
	}

	fmt.Printf("\n%s%s%s\n", boldCyan, query, colorReset)

	if len(result.Prerequisites) > 0 {
		fmt.Printf("\n%s> Prerequisites:%s\n", boldYellow, colorReset)
		for _, pkg := range result.Prerequisites {
			fmt.Printf("   %s$%s %s\n", colorGray, colorReset, pkg)
		}
	}

	fmt.Printf("\n%s> Commands:%s\n", boldGreen, colorReset)
	for i, cmd := range result.Commands {
		fmt.Printf("  %s%2d.%s %s%s%s\n", colorGreen, i+1, colorReset, colorBold, cmd.Command, colorReset)
		fmt.Printf("      %s%s%s\n\n", colorGray, cmd.Description, colorReset)
	}
}
