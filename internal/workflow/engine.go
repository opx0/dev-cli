package workflow

import (
	"context"
	"fmt"
	"time"

	"dev-cli/internal/executor"
	"dev-cli/internal/pipeline"
)

// Engine executes workflows with support for conditionals, rollback, and checkpointing.
type Engine struct {
	store    *CheckpointStore
	bus      *pipeline.EventBus
	verbose  bool
	safeCtx  *SafeModeContext
	rollback *RollbackRegistry
}

// NewEngine creates a new workflow execution engine.
func NewEngine(store *CheckpointStore, bus *pipeline.EventBus) *Engine {
	return &Engine{
		store:    store,
		bus:      bus,
		safeCtx:  NewSafeModeContext(),
		rollback: NewRollbackRegistry(),
	}
}

// SetVerbose enables verbose logging.
func (e *Engine) SetVerbose(v bool) {
	e.verbose = v
}

// SetSafeMode configures the safe mode context.
func (e *Engine) SetSafeMode(ctx *SafeModeContext) {
	e.safeCtx = ctx
}

// GetSafeMode returns the current safe mode context.
func (e *Engine) GetSafeMode() *SafeModeContext {
	return e.safeCtx
}

// GetRollbackRegistry returns the rollback registry.
func (e *Engine) GetRollbackRegistry() *RollbackRegistry {
	return e.rollback
}

// RunResult contains the outcome of a workflow execution.
type RunResult struct {
	RunID       string
	Status      RunStatus
	StepResults map[string]*StepResult
	Error       string
	Duration    time.Duration
}

// Run executes a workflow from the beginning.
func (e *Engine) Run(ctx context.Context, wf *Workflow) (*RunResult, error) {
	runID := GenerateRunID()
	state := NewRunState(runID, wf)
	state.Status = StatusRunning

	if e.store != nil {
		if err := e.store.SaveRun(state); err != nil {
			return nil, fmt.Errorf("failed to save initial state: %w", err)
		}
	}

	e.publishEvent(pipeline.Event{
		Type:      pipeline.EventType("workflow.start"),
		Timestamp: time.Now(),
		Source:    "workflow",
		Data: map[string]interface{}{
			"run_id":        runID,
			"workflow_name": wf.Name,
			"total_steps":   len(wf.Steps),
		},
	})

	return e.executeSteps(ctx, wf, state)
}

// Resume continues execution of a paused or failed workflow.
func (e *Engine) Resume(ctx context.Context, wf *Workflow, runID string) (*RunResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("checkpoint store required for resume")
	}

	state, err := e.store.LoadRun(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load run state: %w", err)
	}

	if state.Status != StatusPaused && state.Status != StatusFailed {
		return nil, fmt.Errorf("cannot resume run with status: %s", state.Status)
	}

	state.Status = StatusRunning
	state.UpdatedAt = time.Now()

	if err := e.store.SaveRun(state); err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	return e.executeSteps(ctx, wf, state)
}

// Rollback executes rollback actions for a failed workflow.
func (e *Engine) Rollback(ctx context.Context, wf *Workflow, runID string) error {
	if e.store == nil {
		return fmt.Errorf("checkpoint store required for rollback")
	}

	state, err := e.store.LoadRun(runID)
	if err != nil {
		return fmt.Errorf("failed to load run state: %w", err)
	}

	return e.executeRollback(ctx, wf, state)
}

// executeSteps runs workflow steps starting from the current position.
func (e *Engine) executeSteps(ctx context.Context, wf *Workflow, state *RunState) (*RunResult, error) {
	startTime := time.Now()

	for i := state.CurrentStepIdx; i < len(wf.Steps); i++ {
		select {
		case <-ctx.Done():
			state.Status = StatusPaused
			state.UpdatedAt = time.Now()
			if e.store != nil {
				_ = e.store.SaveRun(state)
			}
			return &RunResult{
				RunID:       state.RunID,
				Status:      StatusPaused,
				StepResults: state.StepResults,
				Error:       "cancelled",
				Duration:    time.Since(startTime),
			}, ctx.Err()

		default:
		}

		step := wf.Steps[i]
		state.CurrentStepIdx = i

		if ShouldSkip(&step, state.StepResults) {
			result := &StepResult{
				StepID:      step.ID,
				Status:      StepSkipped,
				StartedAt:   time.Now(),
				CompletedAt: time.Now(),
			}
			state.SetStepResult(result)

			if e.store != nil {
				_ = e.store.SaveStepResult(state.RunID, result)
				_ = e.store.SaveRun(state)
			}

			e.log("⏭ Skipping step: %s (condition not met)", step.Name)
			continue
		}

		result := e.executeStep(ctx, &step, wf.Env, state)
		state.SetStepResult(result)

		if e.store != nil {
			_ = e.store.SaveStepResult(state.RunID, result)
			_ = e.store.SaveRun(state)
		}

		e.publishEvent(pipeline.Event{
			Type:      pipeline.EventType("workflow.step"),
			Timestamp: time.Now(),
			Source:    "workflow",
			BlockID:   step.ID,
			Data: map[string]interface{}{
				"run_id":    state.RunID,
				"step_id":   step.ID,
				"step_name": step.Name,
				"status":    string(result.Status),
				"exit_code": result.ExitCode,
			},
		})

		if result.Status == StepFailed {
			action := e.determineFailureAction(wf, &step)

			switch action {
			case FailureRollback:
				e.log("⚠ Step failed, initiating rollback...")
				if err := e.executeRollback(ctx, wf, state); err != nil {
					e.log("✗ Rollback failed: %v", err)
				}
				state.Status = StatusRolledBack
				state.Error = result.Error
				state.CompletedAt = time.Now()
				if e.store != nil {
					_ = e.store.SaveRun(state)
				}
				return &RunResult{
					RunID:       state.RunID,
					Status:      StatusRolledBack,
					StepResults: state.StepResults,
					Error:       result.Error,
					Duration:    time.Since(startTime),
				}, nil

			case FailureAbort:
				state.Status = StatusFailed
				state.Error = result.Error
				state.CompletedAt = time.Now()
				if e.store != nil {
					_ = e.store.SaveRun(state)
				}
				return &RunResult{
					RunID:       state.RunID,
					Status:      StatusFailed,
					StepResults: state.StepResults,
					Error:       result.Error,
					Duration:    time.Since(startTime),
				}, nil

			case FailureContinue:
				e.log("⚠ Step failed but continuing...")
				continue
			}
		}

		if step.OnSuccess != "" {
			nextIdx := e.findStepIndex(wf, step.OnSuccess)
			if nextIdx >= 0 {
				state.CurrentStepIdx = nextIdx - 1
			}
		}
	}

	state.Status = StatusCompleted
	state.CompletedAt = time.Now()
	if e.store != nil {
		_ = e.store.SaveRun(state)
	}

	e.publishEvent(pipeline.Event{
		Type:      pipeline.EventType("workflow.complete"),
		Timestamp: time.Now(),
		Source:    "workflow",
		Data: map[string]interface{}{
			"run_id":   state.RunID,
			"status":   string(StatusCompleted),
			"duration": time.Since(startTime).String(),
		},
	})

	return &RunResult{
		RunID:       state.RunID,
		Status:      StatusCompleted,
		StepResults: state.StepResults,
		Duration:    time.Since(startTime),
	}, nil
}

// executeStep runs a single step with retries.
func (e *Engine) executeStep(ctx context.Context, step *Step, env map[string]string, state *RunState) *StepResult {
	result := &StepResult{
		StepID:    step.ID,
		Status:    StepRunning,
		StartedAt: time.Now(),
	}

	maxRetries := step.Retries
	if maxRetries == 0 {
		maxRetries = 1
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		result.Retries = attempt

		e.log("▶ Running step: %s (attempt %d/%d)", step.Name, attempt+1, maxRetries)

		stepCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		execResult := executor.ExecuteWithContext(stepCtx, step.Command)

		result.ExitCode = execResult.ExitCode
		result.Output = execResult.Output
		result.Duration = execResult.Duration
		result.CompletedAt = time.Now()

		if execResult.ExitCode == 0 {
			result.Status = StepSuccess
			e.log("✓ Step completed: %s", step.Name)
			return result
		}

		e.log("✗ Step failed (exit %d): %s", execResult.ExitCode, step.Name)

		if attempt < maxRetries-1 {
			e.log("  Retrying in 2 seconds...")
			time.Sleep(2 * time.Second)
		}
	}

	result.Status = StepFailed
	result.Error = fmt.Sprintf("step failed with exit code %d after %d attempts", result.ExitCode, maxRetries)
	return result
}

// executeRollback runs rollback commands in reverse order.
func (e *Engine) executeRollback(ctx context.Context, wf *Workflow, state *RunState) error {
	e.publishEvent(pipeline.Event{
		Type:      pipeline.EventType("workflow.rollback"),
		Timestamp: time.Now(),
		Source:    "workflow",
		Data: map[string]interface{}{
			"run_id": state.RunID,
		},
	})

	for i := state.CurrentStepIdx; i >= 0; i-- {
		step := wf.Steps[i]

		stepResult := state.GetStepResult(step.ID)
		if stepResult == nil || stepResult.Status == StepSkipped {
			continue
		}

		if step.Rollback == nil {
			e.log("⏭ No rollback defined for: %s", step.Name)
			continue
		}

		e.log("↺ Rolling back: %s", step.Name)

		rollbackCtx := ctx
		if step.Rollback.Timeout > 0 {
			var cancel context.CancelFunc
			rollbackCtx, cancel = context.WithTimeout(ctx, step.Rollback.Timeout)
			defer cancel()
		}

		result := executor.ExecuteWithContext(rollbackCtx, step.Rollback.Command)

		if result.ExitCode != 0 {
			e.log("⚠ Rollback failed for %s: %s", step.Name, result.Output)
		} else {
			e.log("✓ Rolled back: %s", step.Name)

			if stepResult != nil {
				stepResult.Status = StepRolledBack
				if e.store != nil {
					_ = e.store.SaveStepResult(state.RunID, stepResult)
				}
			}
		}
	}

	return nil
}

// determineFailureAction returns the action to take on step failure.
func (e *Engine) determineFailureAction(wf *Workflow, step *Step) FailureAction {

	if step.OnFailure != "" {
		switch step.OnFailure {
		case "abort":
			return FailureAbort
		case "rollback":
			return FailureRollback
		case "continue":
			return FailureContinue
		}
	}

	if wf.OnFailure != nil {
		return wf.OnFailure.Action
	}

	return FailureAbort
}

// findStepIndex returns the index of a step by ID.
func (e *Engine) findStepIndex(wf *Workflow, stepID string) int {
	for i, step := range wf.Steps {
		if step.ID == stepID {
			return i
		}
	}
	return -1
}

// publishEvent sends an event to the event bus if available.
func (e *Engine) publishEvent(event pipeline.Event) {
	if e.bus != nil {
		e.bus.Publish(event)
	}
}

// log outputs a message if verbose mode is enabled.
func (e *Engine) log(format string, args ...interface{}) {
	if e.verbose {
		fmt.Printf(format+"\n", args...)
	}
}
