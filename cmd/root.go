package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetVersionInfo is called from main() to inject ldflags-provided build metadata.
func SetVersionInfo(v, c, d string) {
	if v != "" {
		buildVersion = v
	}
	if c != "" {
		buildCommit = c
	}
	if d != "" {
		buildDate = d
	}
	rootCmd.Version = buildVersion
	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"dev-cli %s\ncommit: %s\nbuilt:  %s\n",
		buildVersion, buildCommit, buildDate,
	))
}

var rootCmd = &cobra.Command{
	Use:           "dev-cli",
	Short:         "AI-powered terminal companion for developers",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `dev-cli is an AI-powered terminal companion for developers and DevOps engineers.
It analyzes failures using LLMs and provides instant help with autonomous fixing.

Quick Start:
  dev-cli ask kubectl              Get useful kubectl commands
  dev-cli ask "how to resize LVM"  Research a DevOps question
  dev-cli explain                  Analyze why your last command failed
  dev-cli fix "docker won't start" Let the AI agent fix it for you
  dev-cli doctor                   Check system health and dependencies
  dev-cli ui                       Open the interactive dashboard`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
}
