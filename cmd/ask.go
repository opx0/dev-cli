package cmd

import (
	"bytes"
	"dev-cli/internal/ai"
	"dev-cli/internal/core"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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
  dev-cli ask ffmpeg -n 5          # Get 5 commands

  # Research Mode: Ask a question
  dev-cli ask "how to mount an NTFS drive on Linux"
  dev-cli ask "fix permission denied on docker.sock"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		if assistLocal {
			os.Setenv("DEV_CLI_FORCE_LOCAL", "1")
		}

		if err := ai.EnsureOllamaRunning(); err != nil {
			fmt.Fprintf(os.Stderr, "\033[33m⚠\033[0m Ollama not available: %v\n", err)

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
	askCmd.Flags().IntVarP(&assistCount, "n", "n", 10, "Number of commands to show (tool mode)")
	askCmd.Flags().BoolVar(&assistLocal, "local", false, "Force local Ollama (skip Perplexity)")
}

type AssistResult struct {
	Prerequisites []string      `json:"prerequisites"`
	Commands      []CommandInfo `json:"commands"`
}

type CommandInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
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
	client := ai.NewHybridClient()

	backend := "Ollama"
	if client.HasPerplexity() {
		backend = "Perplexity"
	}
	fmt.Printf("\033[90mResearching via %s: %s...\033[0m\n", backend, query)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = "Researching..."
	s.Start()
	result, err := client.Research(query)
	s.Stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m Failed to get solutions: %v\n", err)
		os.Exit(1)
	}

	if len(result.Solutions) == 0 {
		fmt.Println("\033[33m!\033[0m No solutions found")
		return
	}

	fmt.Printf("\n\033[1;32m✓ Found %d Solutions:\033[0m\n\n", len(result.Solutions))

	for _, sol := range result.Solutions {
		fmt.Printf("\033[1;36m[%d] %s\033[0m\n", sol.ID, sol.Title)
		fmt.Printf("    \033[37m%s\033[0m\n\n", sol.Description)

		for _, step := range sol.Steps {
			if step.Type == "command" {
				fmt.Printf("    \033[90m$\033[0m \033[1;33m%s\033[0m\n", step.Content)
				if step.Note != "" {
					fmt.Printf("      \033[90m# %s\033[0m\n", step.Note)
				}
			} else if step.Type == "file" {
				lines := strings.Split(step.Content, "\n")
				lineCount := len(lines)

				fmt.Printf("    \033[90m# %s\033[0m \033[90m(%d lines)\033[0m\n", step.File, lineCount)
				fmt.Println("    \033[90m```\033[0m")
				for _, line := range lines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Println("    \033[90m```\033[0m")

				if step.Note != "" {
					fmt.Printf("    \033[90m# %s\033[0m\n", step.Note)
				}
			}
		}

		if sol.Source != "" {
			fmt.Printf("\n    \033[90mSource: %s\033[0m\n", sol.Source)
		}
		fmt.Println()
	}
}

func fetchCommands(toolName, topic string, count int) {
	cfg := core.LoadConfig()
	baseURL := cfg.OllamaURL
	model := cfg.OllamaModel

	query := toolName
	if topic != "important and commonly used" {
		query = toolName + " " + topic
	}

	prompt := fmt.Sprintf(`Give me %d useful shell commands for: "%s"

Include "prerequisites" array with package install commands if special packages are needed.

JSON format:
{"prerequisites":["sudo pacman -S ntfs-3g"],"commands":[{"command":"sudo mount -t ntfs-3g /dev/sda1 /mnt","description":"Mount NTFS partition"}]}

Commands for "%s":`, count, query, query)

	reqBody, err := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[33m!\033[0m Failed to create request: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[33m!\033[0m Failed to connect to Ollama: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "\033[33m!\033[0m Ollama error %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var genResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		fmt.Fprintf(os.Stderr, "\033[33m!\033[0m Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	var result AssistResult
	responseText := strings.TrimSpace(genResp.Response)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		fmt.Println(responseText)
		return
	}

	fmt.Printf("\n\033[1;36m%s\033[0m\n", query)

	if len(result.Prerequisites) > 0 {
		fmt.Println("\n\033[1;33m> Prerequisites:\033[0m")
		for _, pkg := range result.Prerequisites {
			fmt.Printf("   \033[90m$\033[0m %s\n", pkg)
		}
	}

	fmt.Println("\n\033[1;32m> Commands:\033[0m")
	for i, cmd := range result.Commands {
		fmt.Printf("  \033[32m%2d.\033[0m \033[1m%s\033[0m\n", i+1, cmd.Command)
		fmt.Printf("      \033[90m%s\033[0m\n\n", cmd.Description)
	}
}
