package cmd

import (
	"fmt"
	"os"
)

// Execute - CLI entry point
func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "hook":
		handleHook()
	case "log-event":
		handleLogEvent()
	case "rca":
		handleRCA()
	case "assist":
		handleAssist()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`dev-cli - DevOps command logging and analysis tool

Usage:
  dev-cli <command> [args]

Commands:
  hook zsh         Print the Zsh shell integration script
  log-event        Log a command execution event
  rca              Analyze command failure with LLM
  assist <query>   Get help (tool commands or solutions)
  help             Show this help message

Examples:
  dev-cli assist zoxide                    # Tool commands
  dev-cli assist how to install postgres   # Top 3 solutions
  dev-cli assist setup tailwindcss         # Multi-step guide

Flags for assist:
  -n        Number of commands (tool mode only)
  -local    Force local Ollama (skip Perplexity)

Environment:
  PERPLEXITY_API_KEY   Enable web-sourced solutions
  DEV_CLI_OLLAMA_URL   Custom Ollama endpoint

Shell Integration:
  Add this to your .zshrc:
    eval "$(dev-cli hook zsh)"`)
}
