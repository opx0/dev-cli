package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"dev-cli/internal/llm"
	"dev-cli/internal/logger"

	"golang.org/x/term"
)

func handleRCA() {
	fs := flag.NewFlagSet("rca", flag.ExitOnError)

	// Legacy flags (for direct hook calls)
	command := fs.String("command", "", "The failed command")
	exitCode := fs.Int("exit-code", 0, "Exit code of the command")
	output := fs.String("output", "", "Command output")
	interactive := fs.Bool("interactive", false, "Interactive mode with fix prompts")

	// New query flags (reads from log)
	last := fs.Int("last", 0, "Analyze last N failures from log")
	filter := fs.String("filter", "", "Filter by command keyword (npm, prisma, etc)")
	since := fs.String("since", "", "Filter by time (1h, 30m, etc)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// In interactive mode, check if stdin is a TTY
	if *interactive && !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	// If query flags used OR no command provided, read from log
	if *last > 0 || *filter != "" || *since != "" || *command == "" {
		analyzeFromLog(*last, *filter, *since, *interactive)
		return
	}

	// Legacy mode: direct args (for hook)
	if *exitCode == 130 {
		return // Skip Ctrl-C
	}
	analyzeEntry(logger.LogEntry{
		Command:  *command,
		ExitCode: *exitCode,
		Output:   *output,
	}, *interactive)
}

func analyzeFromLog(limit int, filterStr, sinceStr string, interactive bool) {
	log, err := logger.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to open log: %v\n", err)
		return
	}

	// Parse since duration
	var sinceDur time.Duration
	if sinceStr != "" {
		sinceDur, err = time.ParseDuration(sinceStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Invalid duration: %v\n", err)
			return
		}
	}

	// Default limit to 1 if not specified
	if limit == 0 {
		limit = 1
	}

	entries, err := log.GetFailures(logger.QueryOpts{
		Limit:  limit,
		Filter: filterStr,
		Since:  sinceDur,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to read log: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No failures found matching criteria")
		return
	}

	for _, entry := range entries {
		analyzeEntry(entry, interactive)
	}
}

func analyzeEntry(entry logger.LogEntry, interactive bool) {
	fmt.Printf("\n❌ %s (exit %d)\n", entry.Command, entry.ExitCode)

	client := llm.NewClient()
	result, err := client.Explain(entry.Command, entry.ExitCode, entry.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Analysis failed: %v\n", err)
		return
	}

	fmt.Printf("💡 %s\n", result.Explanation)

	if result.Fix != "" {
		fmt.Printf("📝 Fix: %s\n", result.Fix)

		if interactive {
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
					fmt.Fprintf(os.Stderr, "   ⚠️  Fix failed: %v\n", err)
				} else {
					fmt.Println("   ✅ Fix applied")
				}
			}
		}
	}
}
