package cmd

import (
	"fmt"
	"os"
	"time"

	"dev-cli/internal/hook"
	"dev-cli/internal/storage"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:       "init [shell]",
	Short:     "Print shell integration script",
	Aliases:   []string{"hook"},
	Hidden:    true,
	ValidArgs: []string{"zsh", "bash"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := args[0]
		switch shell {
		case "zsh":
			_, err := os.Stdout.WriteString(hook.ZshHook)
			return err
		case "bash":
			_, err := os.Stdout.WriteString(hook.BashHook)
			return err
		default:
			return fmt.Errorf("unsupported shell %q (supported: zsh, bash)", shell)
		}
	},
}

var (
	logCommand    string
	logExitCode   int
	logCwd        string
	logDurationMs int64
	logOutput     string
)

var logEventCmd = &cobra.Command{
	Use:    "log-event",
	Short:  "Internal: Log a command execution",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logCommand == "" {
			return nil
		}

		db := storage.DB()

		entry := storage.LogEntry{
			Command:    logCommand,
			ExitCode:   logExitCode,
			Cwd:        logCwd,
			DurationMs: logDurationMs,
			Output:     logOutput,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		}

		if err := storage.SaveCommand(db, entry); err != nil {
			return fmt.Errorf("log event: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(logEventCmd)
	logEventCmd.Flags().StringVar(&logCommand, "command", "", "The command that was executed")
	logEventCmd.Flags().IntVar(&logExitCode, "exit-code", 0, "Exit code of the command")
	logEventCmd.Flags().StringVar(&logCwd, "cwd", "", "Working directory")
	logEventCmd.Flags().Int64Var(&logDurationMs, "duration-ms", 0, "Duration in milliseconds")
	logEventCmd.Flags().StringVar(&logOutput, "output", "", "Command stdout/stderr output")
}
