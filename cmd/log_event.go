package cmd

import (
	"flag"
	"fmt"
	"os"

	"dev-cli/internal/logger"
)

func handleLogEvent() {
	fs := flag.NewFlagSet("log-event", flag.ExitOnError)
	command := fs.String("command", "", "The command that was executed")
	exitCode := fs.Int("exit-code", 0, "Exit code of the command")
	cwd := fs.String("cwd", "", "Working directory")
	durationMs := fs.Int64("duration-ms", 0, "Duration in milliseconds")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *command == "" {
		return // silently skip empty commands
	}

	log, err := logger.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger init failed: %v\n", err)
		os.Exit(1)
	}

	if err := log.LogEvent(*command, *exitCode, *cwd, *durationMs); err != nil {
		fmt.Fprintf(os.Stderr, "Log failed: %v\n", err)
		os.Exit(1)
	}
}
