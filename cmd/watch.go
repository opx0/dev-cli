package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"dev-cli/internal/llm"

	"github.com/spf13/cobra"
)

var (
	watchDocker   string
	watchFile     string
	watchAI       string
	watchOpenCode bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch logs for errors and analyze them",
	Long: `Stream logs in real-time and get instant AI analysis when errors are detected.
Monitors for keywords like 'error', 'exception', 'panic', 'fatal', 'failed'.

Use --opencode to save error context for OpenCode handoff instead of local analysis.`,
	Example: `  # Watch a log file
  dev-cli watch --file /var/log/syslog

  # Watch Docker container logs
  dev-cli watch --docker my-container

  # Use cloud AI (Perplexity) for smarter analysis
  dev-cli watch --docker db --ai cloud

  # Save errors for OpenCode handoff (no local AI)
  dev-cli watch --docker db --opencode`,
	Run: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().StringVar(&watchDocker, "docker", "", "Docker container ID/name to monitor")
	watchCmd.Flags().StringVar(&watchFile, "file", "", "Log file path to monitor")
	watchCmd.Flags().StringVar(&watchAI, "ai", "local", "AI backend to use: 'local' (Ollama) or 'cloud' (Perplexity)")
	watchCmd.Flags().BoolVar(&watchOpenCode, "opencode", false, "Save error context for OpenCode handoff instead of local analysis")
}

func runWatch(cmd *cobra.Command, args []string) {
	if watchDocker == "" && watchFile == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify --docker or --file")
		cmd.Usage()
		os.Exit(1)
	}

	var logStream io.ReadCloser
	var err error
	var source string

	if watchDocker != "" {
		source = fmt.Sprintf("Docker container: %s", watchDocker)
		fmt.Printf("\033[36mWatching Docker container: %s\033[0m\n", watchDocker)
		c := exec.Command("docker", "logs", "-f", "--tail", "20", watchDocker)
		logStream, err = c.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting stdout pipe: %v\n", err)
			os.Exit(1)
		}
		c.Stderr = c.Stdout
		if err := c.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting docker logs: %v\n", err)
			os.Exit(1)
		}
	} else {
		source = fmt.Sprintf("Log file: %s", watchFile)
		fmt.Printf("\033[36mWatching file: %s\033[0m\n", watchFile)
		c := exec.Command("tail", "-f", "-n", "20", watchFile)
		logStream, err = c.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting stdout pipe: %v\n", err)
			os.Exit(1)
		}
		if err := c.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting tail: %v\n", err)
			os.Exit(1)
		}
	}

	scanner := bufio.NewScanner(logStream)
	buffer := []string{}
	maxBuffer := 20
	errorKeywords := []string{"error", "exception", "panic", "fatal", "failed"}

	lastAnalysis := time.Now().Add(-time.Hour)
	analysisCooldown := 10 * time.Second

	var client *llm.HybridClient
	if !watchOpenCode {
		client = llm.NewHybridClient()
	}

	if watchOpenCode {
		fmt.Println("\033[90mOpenCode mode: errors will be saved for handoff (Ctrl+C to exit)\033[0m")
	} else {
		fmt.Println("\033[90mWaiting for logs... (Ctrl+C to exit)\033[0m")
	}

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)

		buffer = append(buffer, line)
		if len(buffer) > maxBuffer {
			buffer = buffer[1:]
		}

		lowerLine := strings.ToLower(line)
		isError := false
		for _, kw := range errorKeywords {
			if strings.Contains(lowerLine, kw) {
				isError = true
				break
			}
		}

		if isError && time.Since(lastAnalysis) > analysisCooldown {
			logContent := strings.Join(buffer, "\n")

			if watchOpenCode {

				fmt.Println("\n\033[33m[!] Error detected! Saving for OpenCode...\033[0m")
				savePath, err := saveErrorForOpenCode(source, logContent)
				if err != nil {
					fmt.Printf("\033[31mError saving context: %v\033[0m\n", err)
				} else {
					fmt.Printf("\033[32mSaved to: %s\033[0m\n", savePath)
					fmt.Printf("\033[36mRun 'opencode' and use: @%s\033[0m\n", savePath)
				}
			} else {

				fmt.Println("\n\033[33m[!] Error detected! Analyzing...\033[0m")
				result, err := client.AnalyzeLog(logContent, watchAI)
				if err != nil {
					fmt.Printf("\033[31mError analyzing log: %v\033[0m\n", err)
				} else {
					aiSource := "Local"
					if watchAI == "cloud" {
						aiSource = "Cloud"
					}
					fmt.Printf("\033[90m> [%s AI]\033[0m \033[1m%s\033[0m\n", aiSource, result.Explanation)
					if result.Fix != "" {
						fmt.Printf("\033[32mSuggested Fix: %s\033[0m\n", result.Fix)
					}
				}
			}
			fmt.Println("\033[90m----------------------------------------\033[0m")
			lastAnalysis = time.Now()
		}
	}
}

func saveErrorForOpenCode(source, logs string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	devCliDir := filepath.Join(homeDir, ".devlogs")
	if err := os.MkdirAll(devCliDir, 0755); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("# Error Context from dev-cli watch\n\n")
	sb.WriteString(fmt.Sprintf("**Source:** %s\n", source))
	sb.WriteString(fmt.Sprintf("**Detected:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("## Logs\n\n")
	sb.WriteString("```\n")
	sb.WriteString(logs)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Analyze these logs, identify the root cause of the error, and implement a fix.\n")

	savePath := filepath.Join(devCliDir, "last-error.md")
	if err := os.WriteFile(savePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	return savePath, nil
}
