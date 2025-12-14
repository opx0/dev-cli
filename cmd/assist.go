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
	count := fs.Int("n", 10, "Number of commands to show")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: dev-cli assist <tool-name> [topic]")
		fmt.Fprintln(os.Stderr, "Example: dev-cli assist zoxide important commands")
		os.Exit(1)
	}

	toolName := args[0]
	topic := "important and commonly used"
	if len(args) > 1 {
		topic = strings.Join(args[1:], " ")
	}

	fetchCommands(toolName, topic, *count)
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
