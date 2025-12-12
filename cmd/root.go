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
  hook zsh       Print the Zsh shell integration script
  log-event      Log a command execution event
  help           Show this help message

Shell Integration:
  Add this to your .zshrc:
    eval "$(dev-cli hook zsh)"`)
}
