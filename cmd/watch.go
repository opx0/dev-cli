package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/llm"

	"github.com/spf13/cobra"
)

var (
	watchDocker string
	watchFile   string
	watchAI     string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch logs for errors and analyze them",
	Long: `Stream logs in real-time and get instant AI analysis when errors are detected.
Monitors for keywords like 'error', 'exception', 'panic', 'fatal', 'failed'.`,
	Example: `  # Watch a log file
  dev-cli watch --file /var/log/syslog

  # Watch Docker container logs
  dev-cli watch --docker my-container

  # Use cloud AI (Perplexity) for smarter analysis
  dev-cli watch --docker db --ai cloud`,
	Run: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().StringVar(&watchDocker, "docker", "", "Docker container ID/name to monitor")
	watchCmd.Flags().StringVar(&watchFile, "file", "", "Log file path to monitor")
	watchCmd.Flags().StringVar(&watchAI, "ai", "local", "AI backend to use: 'local' (Ollama) or 'cloud' (Perplexity)")
}

func runWatch(cmd *cobra.Command, args []string) {
	if watchDocker == "" && watchFile == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify --docker or --file")
		cmd.Usage()
		os.Exit(1)
	}

	var logStream io.ReadCloser
	var err error

	if watchDocker != "" {
		fmt.Printf("\033[36mWatching Docker container: %s\033[0m\n", watchDocker)
		c := exec.Command("docker", "logs", "-f", "--tail", "20", watchDocker)
		logStream, err = c.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting stdout pipe: %v\n", err)
			os.Exit(1)
		}
		// Merge stderr into stdout (Docker often logs errors to stderr)
		c.Stderr = c.Stdout
		if err := c.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting docker logs: %v\n", err)
			os.Exit(1)
		}
	} else {
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

	lastAnalysis := time.Now().Add(-time.Hour) // Allow immediate
	analysisCooldown := 10 * time.Second

	client := llm.NewHybridClient()

	fmt.Println("\033[90mWaiting for logs... (Ctrl+C to exit)\033[0m")

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
			fmt.Println("\n\033[33m[!] Error detected! Analyzing...\033[0m")

			logContent := strings.Join(buffer, "\n")
			result, err := client.AnalyzeLog(logContent, watchAI)
			if err != nil {
				fmt.Printf("\033[31mError analyzing log: %v\033[0m\n", err)
			} else {
				source := "Local"
				if watchAI == "cloud" {
					source = "Cloud"
				}
				fmt.Printf("\033[90m> [%s AI]\033[0m \033[1m%s\033[0m\n", source, result.Explanation)
				if result.Fix != "" {
					fmt.Printf("\033[32mSuggested Fix: %s\033[0m\n", result.Fix)
				}
			}
			fmt.Println("\033[90m----------------------------------------\033[0m")
			lastAnalysis = time.Now()
		}
	}
}
