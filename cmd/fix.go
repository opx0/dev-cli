package cmd

import (
	"dev-cli/internal/ai"
	"dev-cli/internal/opencode"
	"dev-cli/internal/storage"
	"fmt"

	"github.com/spf13/cobra"
)

var fixLocal bool

var fixCmd = &cobra.Command{
	Use:   "fix [issue]",
	Short: "Autonomously repair a failure state",
	Long: `Launch an AI agent to solve a problem.

By default, delegates to OpenCode if available for enhanced debugging.
Uses dev-cli's command history and solution database to provide context.

The agent will:
  1. Analyze the issue with debugging context from your command history.
  2. Query known solutions for similar errors.
  3. Propose a fix command.
  4. Execute with your approval.

Use --local to force the built-in agent instead of OpenCode.`,
	Example: `  # Delegate to OpenCode (recommended)
  dev-cli fix "my nginx container keeps crashing"
  
  # Use local agent
  dev-cli fix --local "npm install failing with EACCES"
  
  # Various issues
  dev-cli fix "disk is full on /var"
  dev-cli fix "kubectl can't connect to cluster"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issue := args[0]

		// Try OpenCode delegation first (unless --local)
		if !fixLocal {
			adapter := opencode.NewAdapter()
			if adapter.IsAvailable() {
				runWithOpenCode(adapter, issue)
				return
			}
			fmt.Println("⚠ OpenCode not found, using local agent")
		}

		// Fallback to local agent
		runWithLocalAgent(issue)
	},
}

func runWithOpenCode(adapter *opencode.Adapter, issue string) {
	// Build context from command history
	db, err := storage.InitDB()
	if err != nil {
		fmt.Printf("⚠ Could not load history: %v\n", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// Build debugging context
	ctx := &opencode.DebugContext{
		Issue: issue,
	}

	// Load recent failures and known solutions
	if db != nil {
		if history, err := storage.GetRecentHistory(db, 10); err == nil {
			ctx.RecentHistory = history
		}

		// Get last failure for signature matching
		if lastFailure, err := storage.GetLastUnresolvedFailure(db); err == nil && lastFailure != nil {
			signature := storage.GenerateErrorSignature(lastFailure.Command, lastFailure.ExitCode, lastFailure.Details)
			ctx.ErrorSignature = signature

			if solutions, err := storage.GetSolutionsForError(db, signature); err == nil {
				ctx.KnownSolutions = solutions
			}

			if similar, err := storage.GetSimilarFailures(db, signature, 5); err == nil {
				ctx.SimilarFailures = similar
			}
		}
	}

	// Build prompt with context
	prompt := ctx.ToPrompt()

	fmt.Println("🚀 Delegating to OpenCode with debugging context...")
	if ctx.ErrorSignature != "" && len(ctx.ErrorSignature) >= 16 {
		fmt.Printf("   Error signature: %s\n", ctx.ErrorSignature[:16])
	}
	if len(ctx.KnownSolutions) > 0 {
		fmt.Printf("   Found %d known solutions\n", len(ctx.KnownSolutions))
	}
	fmt.Println()

	// Run OpenCode with the prompt
	err = adapter.RunPrompt(prompt, opencode.RunOptions{
		Agent: "build",
	})
	if err != nil {
		fmt.Printf("✗ OpenCode error: %v\n", err)
	}
}

func runWithLocalAgent(issue string) {
	ag := ai.NewAgent()

	err := ag.Resolve(issue, func(proposal string) bool {
		fmt.Printf("> Proposal: %s\n", proposal)
		fmt.Print("  Allow? [y/N]: ")
		var resp string
		fmt.Scanln(&resp)
		return resp == "y"
	})

	if err != nil {
		fmt.Println("✗ Could not fix the issue.")
	} else {
		fmt.Println("✓ Issue resolved.")
	}
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolVar(&fixLocal, "local", false, "Force use of local agent instead of OpenCode")
}
