package cmd

import (
	"fmt"
	"strings"

	"dev-cli/internal/config"
	"dev-cli/internal/executor"
	"dev-cli/internal/llm"
	"dev-cli/internal/tools"

	"github.com/spf13/cobra"
)

var (
	fixVerbose     bool
	fixMaxIter     int
	fixAutoApprove bool
	fixDryRun      bool
	fixScope       string
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
				query_docker, check_ports, git_info, read_diff, package_info`,
	Example: `  dev-cli fix "my nginx container keeps crashing"
  dev-cli fix "disk is full on /var"
  dev-cli fix "kubectl can't connect to cluster"
  dev-cli fix --verbose "tests are failing in the auth module"
  dev-cli fix --max-iterations 20 "complex refactoring task"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVarP(&fixVerbose, "verbose", "v", false, "Show detailed progress")
	fixCmd.Flags().IntVar(&fixMaxIter, "max-iterations", 10, "Maximum tool-calling iterations")
	fixCmd.Flags().BoolVar(&fixAutoApprove, "auto-approve", false, "Auto-approve all tool calls (dangerous)")
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "Preview actions without executing (safe for exploration)")
	fixCmd.Flags().StringVar(&fixScope, "scope", "", "Limit changes to files under this directory")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	issue := strings.Join(args, " ")
	if fixMaxIter < 1 {
		return fmt.Errorf("--max-iterations must be at least 1")
	}
	cfg := config.Load()
	providerName, model := llm.SelectAgentModel(cfg, false)

	// Dry-run mode announcement
	if fixDryRun {
		fmt.Printf("%s DRY-RUN MODE: No changes will be made\n\n", iconWarn())
	}

	// Create tool registry
	registry := tools.NewDefaultRegistry()
	var scopeRoot string
	if fixScope != "" {
		var err error
		scopeRoot, err = executor.ResolvePath(fixScope)
		if err != nil {
			return fmt.Errorf("invalid --scope: %w", err)
		}
	}

	// Create agent with dependencies
	agent := llm.NewAgentWithDeps(
		llm.NewHybridClient(),
		registry,
	)

	// Configure agent
	agent.SetConfig(llm.AgentConfig{
		MaxIterations: fixMaxIter,
		Model:         model,
		Verbose:       fixVerbose,
		DryRun:        fixDryRun,
		ApprovalFunc: func(toolName string, params map[string]any) bool {
			if scopeRoot != "" && toolName != "write_file" {
				fmt.Printf("\n%s Blocked: %s cannot be contained by --scope\n", iconWarn(), toolName)
				return false
			}
			if scopeRoot != "" && toolName == "write_file" {
				path, ok := params["path"].(string)
				if !ok {
					fmt.Printf("\n%s Blocked: write_file did not provide a valid path\n", iconWarn())
					return false
				}
				resolved, err := executor.ResolvePath(path)
				if err != nil || !executor.IsPathWithin(scopeRoot, resolved) {
					fmt.Printf("\n%s Blocked: write to %s is outside --scope %s\n", iconWarn(), path, fixScope)
					return false
				}
				params["path"] = resolved
			}
			if fixDryRun {
				fmt.Printf("\n%s [DRY-RUN] Would execute: %s\n", iconInfo(), toolName)
				if command, ok := params["command"].(string); ok {
					fmt.Printf("  Command: %s\n", command)
				}
				if path, ok := params["path"].(string); ok {
					fmt.Printf("  File: %s\n", path)
				}
				return false
			}
			if fixAutoApprove {
				return true
			}

			fmt.Printf("\n%s Tool: %s\n", iconInfo(), toolName)
			if command, ok := params["command"].(string); ok {
				fmt.Printf("  Command: %s\n", command)
			}
			if path, ok := params["path"].(string); ok {
				fmt.Printf("  File: %s\n", path)
			}
			fmt.Print("  Allow? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				return false
			}
			return response == "y" || response == "Y"
		},
	})

	// Run the agent
	providerBadge := fmt.Sprintf("%s⚡ local%s", colorGreen, colorReset)
	if providerName == "openai" {
		providerBadge = fmt.Sprintf("%s☁ cloud%s", colorCyan, colorReset)
	}
	fmt.Printf("%s [%s] Analyzing (%s/%s): %s\n", iconInfo(), providerBadge, providerName, model, issue)

	if fixScope != "" {
		printInfo(fmt.Sprintf("Scope limited to: %s", fixScope))
	}

	result, err := agent.Resolve(cmd.Context(), issue)

	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}

	// Print summary
	fmt.Println()

	if fixDryRun {
		fmt.Printf("%s Dry-run complete. Actions that would be taken:\n", iconInfo())
		if result.Summary != "" {
			fmt.Printf("\n%s\n", result.Summary)
		}
	} else if result.Success {
		printSuccess("Issue resolved!")
		if result.Summary != "" {
			fmt.Printf("\n%s Summary:\n%s\n", iconInfo(), result.Summary)
		}
	} else {
		return fmt.Errorf("could not resolve: %s", result.Error)
	}

	// Print tool call summary in verbose mode
	if fixVerbose && len(result.ToolCalls) > 0 {
		fmt.Printf("\n%s Tool calls made: %d\n", iconInfo(), len(result.ToolCalls))
		for i, tc := range result.ToolCalls {
			status := iconOK()
			if !tc.Success {
				status = iconFail()
			}
			if fixDryRun {
				status = "[DRY-RUN]"
			}
			fmt.Printf("  %d. %s %s (%v)\n", i+1, status, tc.ToolName, tc.Duration)
		}
	}
	return nil
}
