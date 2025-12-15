package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"dev-cli/internal/llm"
)

type AssistResult struct {
	Prerequisites []string      `json:"prerequisites"`
	Commands      []CommandInfo `json:"commands"`
}

type CommandInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

func handleAssist() {
	fs := flag.NewFlagSet("assist", flag.ExitOnError)
	count := fs.Int("n", 10, "Number of commands to show (tool mode)")
	local := fs.Bool("local", false, "Force local Ollama (skip Perplexity)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: dev-cli assist <query>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  dev-cli assist zoxide              # Tool commands")
		fmt.Fprintln(os.Stderr, "  dev-cli assist how to install postgres")
		fmt.Fprintln(os.Stderr, "  dev-cli assist setup tailwindcss in react")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -n       Number of commands (tool mode)")
		fmt.Fprintln(os.Stderr, "  -local   Force local Ollama")
		os.Exit(1)
	}

	query := strings.Join(args, " ")

	if *local {
		os.Setenv("DEV_CLI_FORCE_LOCAL", "1")
	}

	if looksLikeToolName(args) {
		toolName := args[0]
		topic := "important and commonly used"
		if len(args) > 1 {
			topic = strings.Join(args[1:], " ")
		}
		fetchCommands(toolName, topic, *count)
	} else {
		fetchSolutions(query)
	}
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
	fmt.Printf("\033[90mResearching via %s: %s...\033[0m\n", backend, query)

	result, err := client.Research(query)
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
				fmt.Printf("    \033[90m📄 %s:\033[0m\n", step.File)
				fmt.Println("    \033[40m┌───────────────────────────────────────────────────┐\033[0m")
				lines := strings.Split(step.Content, "\n")
				for _, line := range lines {
					if len(line) > 50 {
						line = line[:47] + "..."
					}
					fmt.Printf("    \033[40m│\033[0m \033[33m%-49s\033[0m \033[40m│\033[0m\n", line)
				}
				fmt.Println("    \033[40m└───────────────────────────────────────────────────┘\033[0m")
				if step.Note != "" {
					fmt.Printf("      \033[90m# %s\033[0m\n", step.Note)
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
	baseURL := "http://localhost:11434"
	if envURL := os.Getenv("DEV_CLI_OLLAMA_URL"); envURL != "" {
		baseURL = envURL
	}

	model := "qwen2.5-coder:3b-instruct"
	if envModel := os.Getenv("DEV_CLI_OLLAMA_MODEL"); envModel != "" {
		model = envModel
	}

	query := toolName
	if topic != "important and commonly used" {
		query = toolName + " " + topic
	}

	prompt := fmt.Sprintf(`Give me %d useful shell commands for: "%s"

Include "prerequisites" array with package install commands if special packages are needed.

JSON format:
{"prerequisites":["sudo pacman -S ntfs-3g"],"commands":[{"command":"sudo mount -t ntfs-3g /dev/sda1 /mnt","description":"Mount NTFS partition"}]}

Commands for "%s":`, count, query, query)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	})

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
