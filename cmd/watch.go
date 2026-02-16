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
	"dev-cli/internal/storage"

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
		fmt.Printf("%sWatching Docker container: %s%s\n", colorCyan, watchDocker, colorReset)
		c := exec.Command("docker", "logs", "-f", "--tail", "20", watchDocker)
		logStream, err = c.StdoutPipe()
		if err != nil {
			printError(fmt.Sprintf("Error getting stdout pipe: %v", err))
			os.Exit(1)
		}
		c.Stderr = c.Stdout
		if err := c.Start(); err != nil {
			printError(fmt.Sprintf("Error starting docker logs: %v", err))
			os.Exit(1)
		}
	} else {
		source = fmt.Sprintf("Log file: %s", watchFile)
		fmt.Printf("%sWatching file: %s%s\n", colorCyan, watchFile, colorReset)
		c := exec.Command("tail", "-f", "-n", "20", watchFile)
		logStream, err = c.StdoutPipe()
		if err != nil {
			printError(fmt.Sprintf("Error getting stdout pipe: %v", err))
			os.Exit(1)
		}
		if err := c.Start(); err != nil {
			printError(fmt.Sprintf("Error starting tail: %v", err))
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
		printInfo("OpenCode mode: errors will be saved for handoff (Ctrl+C to exit)")
	} else {
		printInfo("Waiting for logs... (Ctrl+C to exit)")
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
				fmt.Printf("\n%s[!] Error detected! Saving for OpenCode...%s\n", colorYellow, colorReset)
				savePath, err := storage.SaveErrorContext(source, logContent)
				if err != nil {
					printError(fmt.Sprintf("Error saving context: %v", err))
				} else {
					printSuccess(fmt.Sprintf("Saved to: %s", savePath))
					fmt.Printf("%sRun 'opencode' and use: @%s%s\n", colorCyan, savePath, colorReset)
				}
			} else {
				fmt.Printf("\n%s[!] Error detected! Analyzing...%s\n", colorYellow, colorReset)
				result, err := client.AnalyzeLog(logContent, watchAI)
				if err != nil {
					printError(fmt.Sprintf("Error analyzing log: %v", err))
				} else {
					aiSource := "Local"
					if watchAI == "cloud" {
						aiSource = "Cloud"
					}
					fmt.Printf("%s> [%s AI]%s %s%s%s\n", colorGray, aiSource, colorReset, colorBold, result.Explanation, colorReset)
					if result.Fix != "" {
						fmt.Printf("%sSuggested Fix: %s%s\n", colorGreen, result.Fix, colorReset)
					}
				}
			}
			separator()
			lastAnalysis = time.Now()
		}
	}
}
