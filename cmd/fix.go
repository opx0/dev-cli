package cmd

import (
	"dev-cli/internal/agent"
	"fmt"

	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix [issue]",
	Short: "Autonomously repair a failure state",
	Long:  "The agent will analyze the issue, propose commands, and execute them upon your approval.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Init Agent
		ag := agent.New()

		// 2. Start Loop
		err := ag.Resolve(args[0], func(proposal string) bool {
			// 3. Safety Check
			fmt.Printf("🤖 I want to run: \033[1m%s\033[0m\n", proposal)
			fmt.Print("   Allow? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			return resp == "y"
		})

		if err != nil {
			fmt.Println("❌ Could not fix the issue automatically.")
		} else {
			fmt.Println("✅ System repaired.")
		}
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)
}
