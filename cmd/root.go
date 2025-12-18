package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dev-cli",
	Short: "DevOps command logging and analysis tool",
	Long: `dev-cli is an AI-powered terminal companion for DevOps engineers.
It logs commands, analyzes failures using LLMs, and provides instant help.

Quick Start:
  dev-cli ask kubectl              Get useful kubectl commands
  dev-cli ask "how to resize LVM"  Research a DevOps question
  dev-cli explain                  Analyze why your last command failed
  dev-cli fix "docker won't start" Let the AI agent fix it for you
  dev-cli watch --docker myapp     Monitor logs with AI error detection
  dev-cli ui                       Open the interactive dashboard`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
}
