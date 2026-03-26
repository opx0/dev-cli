package cmd

import (
	"context"
	"fmt"

	"dev-cli/internal/llm"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/tools"
	"dev-cli/internal/workflow"

	"github.com/spf13/cobra"
)

var (
	fixVerbose     bool
	fixSafeMode    bool
	fixMaxIter     int
	fixAutoApprove bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [issue]",
	Short: "Autonomously repair a failure state",
	Long: `Launch an autonomous AI agent to solve a problem using structured tool calling.

The agent will:
  1. Analyze the issue using available diagnostic tools
  2. Gather information (read files, search code, inspect git history)
  3. Propose fixes using write_file or run_command tools
  4. Wait for your approval before destructive actions
  5. Repeat until the issue is resolved

Available tools: read_file, read_dir, write_file, run_command, search_codebase,
                query_docker, check_ports, git_info, git_inspector, package_info`,
	Example: `  dev-cli fix "my nginx container keeps crashing"
  dev-cli fix "disk is full on /var"
  dev-cli fix "kubectl can't connect to cluster"
  dev-cli fix --verbose "tests are failing in the auth module"
  dev-cli fix --max-iterations 20 "complex refactoring task"`,
	Args: cobra.MinimumNArgs(1),
	Run:  runFix,
}

func init() {
	fixCmd.Flags().BoolVarP(&fixVerbose, "verbose", "v", false, "Show detailed progress")
	fixCmd.Flags().BoolVar(&fixSafeMode, "safe", true, "Enable safe mode (require approval for destructive tools)")
	fixCmd.Flags().IntVar(&fixMaxIter, "max-iterations", 10, "Maximum tool-calling iterations")
	fixCmd.Flags().BoolVar(&fixAutoApprove, "auto-approve", false, "Auto-approve all tool calls (dangerous)")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) {
	issue := args[0]

	// Create tool registry
	registry := tools.GetRegistry()
	registry.RegisterDefaults()

	// Create workflow engine for safe mode (store=nil means no checkpointing)
	bus := pipeline.NewEventBus()
	engine := workflow.NewEngine(nil, bus)
	engine.SetVerbose(fixVerbose)

	// Configure safe mode
	if fixSafeMode && !fixAutoApprove {
		safeCtx := workflow.NewExecuteContext(func(action string) bool {
			fmt.Printf("\n%s %s\n", iconWarn(), action)
			fmt.Print("  Allow? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			return resp == "y" || resp == "Y"
		})
		engine.SetSafeMode(safeCtx)
	} else if fixAutoApprove {
		// Auto-approve mode - no confirmation required
		safeCtx := workflow.NewExecuteContext(func(string) bool { return true })
		engine.SetSafeMode(safeCtx)
	}

	// Create agent with dependencies
	agent := llm.NewAgentWithDeps(
		llm.NewHybridClient(),
		registry,
		engine,
	)

	// Configure agent
	agent.SetConfig(llm.AgentConfig{
		MaxIterations: fixMaxIter,
		Model:         llm.DefaultModel,
		Verbose:       fixVerbose,
		ApprovalFunc: func(toolName string, params map[string]any) bool {
			if fixAutoApprove {
				return true
			}
			// Only prompt for potentially destructive tools
			if toolName == "write_file" || toolName == "run_command" {
				fmt.Printf("\n%s Tool: %s\n", iconInfo(), toolName)
				if toolName == "run_command" {
					if cmdStr, ok := params["command"].(string); ok {
						fmt.Printf("  Command: %s\n", cmdStr)
					}
				} else if toolName == "write_file" {
					if path, ok := params["path"].(string); ok {
						fmt.Printf("  File: %s\n", path)
					}
				}
				fmt.Print("  Allow? [y/N]: ")
				var resp string
				fmt.Scanln(&resp)
				return resp == "y" || resp == "Y"
			}
			return true
		},
	})

	// Run the agent
	fmt.Printf("%s Analyzing: %s\n", iconInfo(), issue)

	ctx := context.Background()
	result, err := agent.Resolve(ctx, issue)

	if err != nil {
		printError(fmt.Sprintf("Agent failed: %v", err))
		return
	}

	// Print summary
	fmt.Println()
	if result.Success {
		printSuccess("Issue resolved!")
		if result.Summary != "" {
			fmt.Printf("\n%s Summary:\n%s\n", iconInfo(), result.Summary)
		}
	} else {
		printError(fmt.Sprintf("Could not resolve: %s", result.Error))
	}

	// Print tool call summary in verbose mode
	if fixVerbose && len(result.ToolCalls) > 0 {
		fmt.Printf("\n%s Tool calls made: %d\n", iconInfo(), len(result.ToolCalls))
		for i, tc := range result.ToolCalls {
			status := iconOK()
			if !tc.Success {
				status = iconFail()
			}
			fmt.Printf("  %d. %s %s (%v)\n", i+1, status, tc.ToolName, tc.Duration)
		}
	}
}
