package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dev-cli",
	Short: "DevOps command logging and analysis tool",
	Long: `dev-cli is a CLI tool for DevOps engineers to log commands, 
analyze failures using LLMs, and get assistance with tool usage.`,
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
