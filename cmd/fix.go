package cmd

import (
	"dev-cli/internal/agent"
	"fmt"

	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix [issue]",
	Short: "Autonomously repair a failure state",
	Long: `Launch an autonomous AI agent to solve a problem.
The agent will:
  1. Analyze the issue you describe.
  2. Propose a command to run.
  3. Wait for your approval (y/n).
  4. Execute and analyze the result.
  5. Repeat until the issue is resolved.`,
	Example: `  dev-cli fix "my nginx container keeps crashing"
  dev-cli fix "disk is full on /var"
  dev-cli fix "kubectl can't connect to cluster"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ag := agent.New()

		err := ag.Resolve(args[0], func(proposal string) bool {
			fmt.Printf("> Proposal: %s\n", proposal)
			fmt.Print("  Allow? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			return resp == "y"
		})

		if err != nil {
			fmt.Println("x Could not fix the issue.")
		} else {
			fmt.Println("+ Issue resolved.")
		}
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)
}
