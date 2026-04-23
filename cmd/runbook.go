package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"dev-cli/internal/pipeline"
	"dev-cli/internal/storage"
	"dev-cli/internal/tools"
	"dev-cli/internal/workflow"

	"github.com/spf13/cobra"
)

var runbookCmd = &cobra.Command{
	Use:   "runbook",
	Short: "List, inspect, replay, or remove saved fix runbooks",
	Long: `Runbooks are automatically recorded when dev-cli fix successfully resolves
an issue. They capture the exact tool calls and parameters used, so the same
fix can be replayed without calling the LLM.`,
}

var runbookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved runbooks for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		cwd, _ := os.Getwd()
		projectID, projectType, _ := storage.DetectProjectFingerprintID(cwd)

		rbs, err := storage.GetRunbooksForProject(db, projectID)
		if err != nil {
			return err
		}
		if len(rbs) == 0 {
			fmt.Printf("%s No runbooks for this project (%s)\n", iconInfo(), projectType)
			return nil
		}
		fmt.Printf("Project %s (%s) — %d runbook(s):\n\n", projectID, projectType, len(rbs))
		for _, rb := range rbs {
			fmt.Printf("  %s  %s\n", rb.ID, rb.Name)
			fmt.Printf("    steps: %d, success: %.0f%%, used: %dx, last: %s\n",
				len(rb.Steps), rb.SuccessRate*100, rb.UsageCount, rb.LastUsed.Format("2006-01-02"))
		}
		return nil
	},
}

var runbookShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Pretty-print a runbook's steps",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.InitDB()
		if err != nil {
			return err
		}
		rb, err := storage.GetRunbookByID(db, args[0])
		if err != nil || rb == nil {
			return fmt.Errorf("runbook not found: %s", args[0])
		}
		fmt.Printf("ID:          %s\nName:        %s\nDescription: %s\nProject:     %s\n",
			rb.ID, rb.Name, rb.Description, rb.ProjectID)
		fmt.Printf("Success:     %.0f%% over %d use(s)\nSteps:\n", rb.SuccessRate*100, rb.UsageCount)
		for i, s := range rb.Steps {
			fmt.Printf("  %d. %s\n     %s\n", i+1, s.Name, s.Command)
		}
		return nil
	},
}

var runbookRunCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "Replay a saved runbook (with safe-mode prompts)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.InitDB()
		if err != nil {
			return err
		}
		rb, err := storage.GetRunbookByID(db, args[0])
		if err != nil || rb == nil {
			return fmt.Errorf("runbook not found: %s", args[0])
		}

		registry := tools.GetRegistry()
		registry.RegisterDefaults()

		bus := pipeline.NewEventBus()
		engine := workflow.NewEngine(nil, bus)
		engine.SetSafeMode(workflow.NewExecuteContext(func(action string) bool {
			fmt.Printf("\n%s %s\n  Allow? [y/N]: ", iconWarn(), action)
			var resp string
			fmt.Scanln(&resp)
			return resp == "y" || resp == "Y"
		}))

		ctx := context.Background()
		ok := true
		for _, s := range rb.Steps {
			tool, exists := registry.Get(s.Name)
			if !exists {
				printError(fmt.Sprintf("unknown tool %q", s.Name))
				ok = false
				break
			}
			var params map[string]any
			if err := json.Unmarshal([]byte(s.Command), &params); err != nil {
				printError(fmt.Sprintf("malformed params for %s: %v", s.Name, err))
				ok = false
				break
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
				printError(fmt.Sprintf("%s failed: %s", s.Name, resp.Error))
				ok = false
				break
			}
			printSuccess(fmt.Sprintf("%s done", s.Name))
		}
		_ = storage.UpdateRunbookStats(db, rb.ID, ok)
		if !ok {
			return fmt.Errorf("runbook replay failed")
		}
		printSuccess("Runbook complete")
		return nil
	},
}

var runbookRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Delete a saved runbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.InitDB()
		if err != nil {
			return err
		}
		if err := storage.DeleteRunbook(db, args[0]); err != nil {
			return err
		}
		printSuccess(fmt.Sprintf("Deleted runbook %s", args[0]))
		return nil
	},
}

func init() {
	runbookCmd.AddCommand(runbookListCmd, runbookShowCmd, runbookRunCmd, runbookRmCmd)
	rootCmd.AddCommand(runbookCmd)
}
