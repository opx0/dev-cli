package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"dev-cli/internal/pipeline"
	"dev-cli/internal/storage"
	"dev-cli/internal/workflow"

	"github.com/spf13/cobra"
)

var (
	workflowVerbose bool
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage and execute multi-step workflows",
	Long: `Execute, resume, and manage multi-step workflow automations.

Workflows are defined in YAML files and support:
  - Sequential step execution
  - Conditional branching
  - Automatic retry on failure
  - Rollback capabilities
  - Checkpoint/resume for long operations`,
}

var workflowRunCmd = &cobra.Command{
	Use:   "run <file.yaml>",
	Short: "Execute a workflow from a YAML file",
	Example: `  dev-cli workflow run deploy.yaml
  dev-cli workflow run ~/.devlogs/workflows/cleanup.yaml --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		wf, err := workflow.ParseFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse workflow: %w", err)
		}

		fmt.Printf("🚀 Starting workflow: %s\n", wf.Name)
		if wf.Description != "" {
			fmt.Printf("   %s\n", wf.Description)
		}
		fmt.Printf("   Steps: %d\n\n", len(wf.Steps))

		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		store := workflow.NewCheckpointStore(db)
		if err := store.InitSchema(); err != nil {
			return fmt.Errorf("failed to initialize workflow schema: %w", err)
		}

		bus := pipeline.NewEventBus()
		engine := workflow.NewEngine(store, bus)
		engine.SetVerbose(workflowVerbose)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n⏸ Received interrupt, saving checkpoint...")
			cancel()
		}()

		result, err := engine.Run(ctx, wf)
		if err != nil && result == nil {
			return fmt.Errorf("workflow execution failed: %w", err)
		}

		fmt.Println()
		printRunResult(result)

		return nil
	},
}

var workflowResumeCmd = &cobra.Command{
	Use:     "resume <run-id>",
	Short:   "Resume a paused or failed workflow",
	Example: `  dev-cli workflow resume run_1703548800000000000`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		store := workflow.NewCheckpointStore(db)

		state, err := store.LoadRun(runID)
		if err != nil {
			return fmt.Errorf("failed to load run: %w", err)
		}

		workflowFile, err := findWorkflowFile(state.WorkflowID, state.WorkflowName)
		if err != nil {
			return fmt.Errorf("workflow file not found: %w\n\nPlease provide the workflow file path with: dev-cli workflow resume-file %s <file.yaml>", err, runID)
		}

		wf, err := workflow.ParseFile(workflowFile)
		if err != nil {
			return fmt.Errorf("failed to parse workflow: %w", err)
		}

		fmt.Printf("▶ Resuming workflow: %s (run: %s)\n", wf.Name, runID)
		fmt.Printf("  Current step: %d/%d\n\n", state.CurrentStepIdx+1, len(wf.Steps))

		bus := pipeline.NewEventBus()
		engine := workflow.NewEngine(store, bus)
		engine.SetVerbose(workflowVerbose)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n⏸ Received interrupt, saving checkpoint...")
			cancel()
		}()

		result, err := engine.Resume(ctx, wf, runID)
		if err != nil && result == nil {
			return fmt.Errorf("resume failed: %w", err)
		}

		fmt.Println()
		printRunResult(result)

		return nil
	},
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent workflow runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		store := workflow.NewCheckpointStore(db)
		if err := store.InitSchema(); err != nil {
			return err
		}

		runs, err := store.ListRuns(20)
		if err != nil {
			return fmt.Errorf("failed to list runs: %w", err)
		}

		if len(runs) == 0 {
			fmt.Println("No workflow runs found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RUN ID\tWORKFLOW\tSTATUS\tSTARTED\tDURATION")
		fmt.Fprintln(w, "------\t--------\t------\t-------\t--------")

		for _, run := range runs {
			duration := ""
			if !run.CompletedAt.IsZero() {
				duration = run.CompletedAt.Sub(run.StartedAt).Truncate(time.Second).String()
			} else if run.Status == workflow.StatusRunning {
				duration = time.Since(run.StartedAt).Truncate(time.Second).String() + " (running)"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				run.RunID,
				run.WorkflowName,
				formatStatus(run.Status),
				run.StartedAt.Format("2006-01-02 15:04"),
				duration,
			)
		}

		return w.Flush()
	},
}

var workflowStatusCmd = &cobra.Command{
	Use:   "status <run-id>",
	Short: "Show detailed status of a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		store := workflow.NewCheckpointStore(db)
		state, err := store.LoadRun(runID)
		if err != nil {
			return fmt.Errorf("failed to load run: %w", err)
		}

		fmt.Printf("Workflow: %s\n", state.WorkflowName)
		fmt.Printf("Run ID:   %s\n", state.RunID)
		fmt.Printf("Status:   %s\n", formatStatus(state.Status))
		fmt.Printf("Started:  %s\n", state.StartedAt.Format(time.RFC3339))

		if !state.CompletedAt.IsZero() {
			fmt.Printf("Finished: %s\n", state.CompletedAt.Format(time.RFC3339))
			fmt.Printf("Duration: %s\n", state.CompletedAt.Sub(state.StartedAt).Truncate(time.Second))
		}

		if state.Error != "" {
			fmt.Printf("Error:    %s\n", state.Error)
		}

		fmt.Printf("\nSteps:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  STEP\tSTATUS\tEXIT\tDURATION")
		fmt.Fprintln(w, "  ----\t------\t----\t--------")

		for stepID, result := range state.StepResults {
			fmt.Fprintf(w, "  %s\t%s\t%d\t%s\n",
				stepID,
				formatStepStatus(result.Status),
				result.ExitCode,
				result.Duration.Truncate(time.Millisecond),
			)
		}

		return w.Flush()
	},
}

var workflowRollbackCmd = &cobra.Command{
	Use:   "rollback <run-id>",
	Short: "Manually trigger rollback for a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		db, err := storage.InitDB()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer db.Close()

		store := workflow.NewCheckpointStore(db)
		state, err := store.LoadRun(runID)
		if err != nil {
			return fmt.Errorf("failed to load run: %w", err)
		}

		workflowFile, err := findWorkflowFile(state.WorkflowID, state.WorkflowName)
		if err != nil {
			return fmt.Errorf("workflow file not found: %w", err)
		}

		wf, err := workflow.ParseFile(workflowFile)
		if err != nil {
			return fmt.Errorf("failed to parse workflow: %w", err)
		}

		fmt.Printf("↺ Rolling back workflow: %s\n", wf.Name)

		bus := pipeline.NewEventBus()
		engine := workflow.NewEngine(store, bus)
		engine.SetVerbose(true)

		ctx := context.Background()
		if err := engine.Rollback(ctx, wf, runID); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		fmt.Println("\n✓ Rollback completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)

	workflowCmd.PersistentFlags().BoolVarP(&workflowVerbose, "verbose", "v", false, "Enable verbose output")

	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowResumeCmd)
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowStatusCmd)
	workflowCmd.AddCommand(workflowRollbackCmd)
}

func printRunResult(result *workflow.RunResult) {
	if result == nil {
		return
	}

	switch result.Status {
	case workflow.StatusCompleted:
		fmt.Printf("✓ Workflow completed successfully in %s\n", result.Duration.Truncate(time.Second))
	case workflow.StatusPaused:
		fmt.Printf("⏸ Workflow paused. Resume with:\n  dev-cli workflow resume %s\n", result.RunID)
	case workflow.StatusFailed:
		fmt.Printf("✗ Workflow failed: %s\n", result.Error)
		fmt.Printf("  Resume with: dev-cli workflow resume %s\n", result.RunID)
	case workflow.StatusRolledBack:
		fmt.Printf("↺ Workflow rolled back after failure: %s\n", result.Error)
	default:
		fmt.Printf("? Workflow ended with status: %s\n", result.Status)
	}
}

func formatStatus(status workflow.RunStatus) string {
	switch status {
	case workflow.StatusCompleted:
		return "✓ completed"
	case workflow.StatusRunning:
		return "▶ running"
	case workflow.StatusPaused:
		return "⏸ paused"
	case workflow.StatusFailed:
		return "✗ failed"
	case workflow.StatusRolledBack:
		return "↺ rolledback"
	default:
		return string(status)
	}
}

func formatStepStatus(status workflow.StepStatus) string {
	switch status {
	case workflow.StepSuccess:
		return "✓"
	case workflow.StepFailed:
		return "✗"
	case workflow.StepSkipped:
		return "⏭"
	case workflow.StepRolledBack:
		return "↺"
	case workflow.StepRunning:
		return "▶"
	default:
		return string(status)
	}
}

func findWorkflowFile(workflowID, workflowName string) (string, error) {

	home, _ := os.UserHomeDir()
	searchPaths := []string{
		filepath.Join(home, ".devlogs", "workflows"),
		".",
		filepath.Join(home, ".config", "dev-cli", "workflows"),
	}

	for _, dir := range searchPaths {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			name := f.Name()
			if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
				continue
			}

			fullPath := filepath.Join(dir, name)
			wf, err := workflow.ParseFile(fullPath)
			if err != nil {
				continue
			}

			if wf.ID == workflowID || wf.Name == workflowName {
				return fullPath, nil
			}
		}
	}

	return "", fmt.Errorf("could not find workflow %q", workflowName)
}

// GetDB returns a database connection (for use by external callers)
func GetDB() (*sql.DB, error) {
	return storage.InitDB()
}
