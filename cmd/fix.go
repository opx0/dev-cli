package cmd

import (
	"context"
	"fmt"

	"database/sql"
	"encoding/json"
	"os"
	"strings"

	"dev-cli/internal/config"
	"dev-cli/internal/diffsandbox"
	"dev-cli/internal/llm"
	"dev-cli/internal/pipeline"
	"dev-cli/internal/storage"
	"dev-cli/internal/tools"
	"dev-cli/internal/workflow"

	"github.com/spf13/cobra"
)

var (
	fixVerbose     bool
	fixSafeMode    bool
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
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "Preview actions without executing (safe for exploration)")
	fixCmd.Flags().StringVar(&fixScope, "scope", "", "Limit changes to files under this directory")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) {
	issue := args[0]
	cfg := config.Load()
	providerName, model := llm.SelectAgentModel(cfg, false)

	// Dry-run mode announcement
	if fixDryRun {
		fmt.Printf("%s DRY-RUN MODE: No changes will be made\n\n", iconWarn())
	}

	// Create tool registry
	registry := tools.GetRegistry()
	registry.RegisterDefaults()

	// Create workflow engine for safe mode (store=nil means no checkpointing)
	bus := pipeline.NewEventBus()
	engine := workflow.NewEngine(nil, bus)
	engine.SetVerbose(fixVerbose)

	// Configure safe mode
	if fixDryRun {
		// Dry-run mode - preview only, no execution
		safeCtx := workflow.NewSafeModeContext() // Preview mode by default
		engine.SetSafeMode(safeCtx)
	} else if fixSafeMode && !fixAutoApprove {
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

	// Wire persistent store for runbook learning (best-effort; runs without if unavailable).
	if db, err := storage.InitDB(); err == nil {
		agent.SetDB(db)
	}

	// Create diff tracker for change review
	tracker := diffsandbox.NewTracker()

	// Configure agent
	agent.SetConfig(llm.AgentConfig{
		MaxIterations: fixMaxIter,
		Model:         model,
		Verbose:       fixVerbose,
		DryRun:        fixDryRun,
		ApprovalFunc: func(toolName string, params map[string]any) bool {
			if fixAutoApprove {
				return true
			}
			if fixDryRun {
				// In dry-run, show what would happen but don't actually prompt
				fmt.Printf("\n%s [DRY-RUN] Would execute: %s\n", iconInfo(), toolName)
				if toolName == "run_command" {
					if cmdStr, ok := params["command"].(string); ok {
						fmt.Printf("  Command: %s\n", cmdStr)
					}
				} else if toolName == "write_file" {
					if path, ok := params["path"].(string); ok {
						fmt.Printf("  File: %s\n", path)
					}
				}
				return false // Don't actually execute in dry-run
			}

			// Scope enforcement
			if fixScope != "" && toolName == "write_file" {
				if path, ok := params["path"].(string); ok {
					if !strings.HasPrefix(path, fixScope) {
						fmt.Printf("\n%s Blocked: write to %s is outside --scope %s\n", iconWarn(), path, fixScope)
						return false
					}
				}
			}

			// Snapshot file before write for diff tracking
			if toolName == "write_file" {
				if path, ok := params["path"].(string); ok {
					_ = tracker.Snapshot(path)
				}
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
				approved := resp == "y" || resp == "Y"

				// Record write content after approval for diff tracking
				if approved && toolName == "write_file" {
					if path, ok := params["path"].(string); ok {
						if content, ok := params["content"].(string); ok {
							tracker.RecordWrite(path, content)
						}
					}
				}

				return approved
			}
			return true
		},
	})

	ctx := context.Background()

	// Runbook replay: if we've resolved the same issue for this project before
	// and it succeeded, offer to re-run the saved steps without calling the LLM.
	if !fixDryRun {
		if db, err := storage.InitDB(); err == nil {
			if rb := findReplayableRunbook(db, issue); rb != nil {
				if promptReplayRunbook(rb, fixAutoApprove) {
					printInfo(fmt.Sprintf("Replaying runbook %q (%d steps)", rb.Name, len(rb.Steps)))
					if replayRunbook(ctx, engine, registry, rb) {
						printSuccess("Replay succeeded")
						_ = storage.UpdateRunbookStats(db, rb.ID, true)
						return
					}
					_ = storage.UpdateRunbookStats(db, rb.ID, false)
					printWarning("Replay failed, falling back to agent")
				}
			}
		}
	}

	// Run the agent
	providerBadge := fmt.Sprintf("%s⚡ local%s", colorGreen, colorReset)
	if providerName == "openai" {
		providerBadge = fmt.Sprintf("%s☁ cloud%s", colorCyan, colorReset)
	}
	fmt.Printf("%s [%s] Analyzing (%s/%s): %s\n", iconInfo(), providerBadge, providerName, model, issue)

	if fixScope != "" {
		printInfo(fmt.Sprintf("Scope limited to: %s", fixScope))
	}

	result, err := agent.Resolve(ctx, issue)

	if err != nil {
		printError(fmt.Sprintf("Agent failed: %v", err))

		// Offer rollback if changes were made before failure
		if tracker.HasChanges() {
			fmt.Printf("\n%s Changes were made before the failure:\n", iconWarn())
			fmt.Print(tracker.FormatDiff())
			fmt.Print("  Rollback all changes? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			if resp == "y" || resp == "Y" {
				rolledBack, rollErrs := tracker.Rollback()
				if len(rollErrs) > 0 {
					for _, e := range rollErrs {
						printError(fmt.Sprintf("Rollback error: %v", e))
					}
				}
				printSuccess(fmt.Sprintf("Rolled back %d file(s)", rolledBack))
			}
		}
		return
	}

	// Print summary
	fmt.Println()

	// Show cumulative diff if any files were changed
	if tracker.HasChanges() {
		fmt.Printf("%s Changes made (%d files):\n", iconInfo(), tracker.Count())
		fmt.Print(tracker.FormatDiff())
	}

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

		// Offer rollback even on success
		if tracker.HasChanges() && !fixAutoApprove {
			fmt.Print("  Undo all changes? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			if resp == "y" || resp == "Y" {
				rolledBack, rollErrs := tracker.Rollback()
				if len(rollErrs) > 0 {
					for _, e := range rollErrs {
						printError(fmt.Sprintf("Rollback error: %v", e))
					}
				}
				printSuccess(fmt.Sprintf("Rolled back %d file(s)", rolledBack))
			}
		}
	} else {
		printError(fmt.Sprintf("Could not resolve: %s", result.Error))

		// Offer rollback on failure
		if tracker.HasChanges() {
			fmt.Print("  Rollback all changes? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			if resp == "y" || resp == "Y" {
				rolledBack, rollErrs := tracker.Rollback()
				if len(rollErrs) > 0 {
					for _, e := range rollErrs {
						printError(fmt.Sprintf("Rollback error: %v", e))
					}
				}
				printSuccess(fmt.Sprintf("Rolled back %d file(s)", rolledBack))
			}
		}
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
}

// findReplayableRunbook returns a high-confidence runbook for the current
// project whose recorded issue matches the requested one. Matching is a
// case-insensitive exact or substring match against runbook name/description.
func findReplayableRunbook(db *sql.DB, issue string) *storage.Runbook {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	projectID, _, _ := storage.DetectProjectFingerprintID(cwd)
	runbooks, err := storage.GetRunbooksForProject(db, projectID)
	if err != nil {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(issue))
	for i := range runbooks {
		rb := &runbooks[i]
		if rb.SuccessRate < 0.7 {
			continue
		}
		hay := strings.ToLower(rb.Description + " " + rb.Name)
		if strings.Contains(hay, needle) || strings.Contains(needle, strings.ToLower(rb.Description)) {
			return rb
		}
	}
	return nil
}

// promptReplayRunbook asks the user whether to replay a matched runbook. Auto-approve
// accepts without prompting.
func promptReplayRunbook(rb *storage.Runbook, autoApprove bool) bool {
	fmt.Printf("\n%s Found cached runbook: %s\n", iconInfo(), rb.Name)
	fmt.Printf("  Steps: %d, success rate: %.0f%%, used: %dx\n", len(rb.Steps), rb.SuccessRate*100, rb.UsageCount)
	for i, s := range rb.Steps {
		fmt.Printf("  %d. %s  %s\n", i+1, s.Name, truncate(s.Command, 80))
	}
	if autoApprove {
		return true
	}
	fmt.Print("Replay this known fix? [Y/n]: ")
	var resp string
	fmt.Scanln(&resp)
	r := strings.TrimSpace(strings.ToLower(resp))
	return r == "" || r == "y" || r == "yes"
}

// replayRunbook executes each stored step against the tool registry via the
// workflow engine. Returns true when all steps succeed.
func replayRunbook(ctx context.Context, engine *workflow.Engine, registry *tools.Registry, rb *storage.Runbook) bool {
	for _, s := range rb.Steps {
		tool, ok := registry.Get(s.Name)
		if !ok {
			printError(fmt.Sprintf("replay: unknown tool %q (runbook is stale)", s.Name))
			return false
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(s.Command), &params); err != nil {
			printError(fmt.Sprintf("replay: malformed params for step %s: %v", s.Name, err))
			return false
		}
		resp := engine.ExecuteToolStep(ctx, workflow.StepRequest{
			ToolName:   s.Name,
			Parameters: params,
		}, func(ctx context.Context, p map[string]any) (bool, string, error) {
			r := tool.Execute(ctx, p)
			if r.Success {
				return true, r.Error, nil
			}
			return false, r.Error, fmt.Errorf("%s", r.Error)
		})
		if !resp.Success {
			printError(fmt.Sprintf("replay: step %s failed: %s", s.Name, resp.Error))
			return false
		}
	}
	return true
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
