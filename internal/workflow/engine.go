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
				e.store.SaveRun(state)
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
				e.store.SaveStepResult(state.RunID, result)
				e.store.SaveRun(state)
			}

			e.log("⏭ Skipping step: %s (condition not met)", step.Name)
			continue
		}

		result := e.executeStep(ctx, &step, wf.Env, state)
		state.SetStepResult(result)

		if e.store != nil {
			e.store.SaveStepResult(state.RunID, result)
			e.store.SaveRun(state)
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
					e.store.SaveRun(state)
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
					e.store.SaveRun(state)
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
		e.store.SaveRun(state)
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
					e.store.SaveStepResult(state.RunID, stepResult)
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

// ── Agent Integration ────────────────────────────────────────────────────────

// StepRequest represents a single tool execution request from the agent.
type StepRequest struct {
	ToolName   string         // Name of the tool to execute
	Parameters map[string]any // Tool parameters
	RunID      string         // Workflow run ID (for checkpoint association)
}

// StepResponse represents the outcome of a single tool execution.
type StepResponse struct {
	Success  bool   // Whether the tool succeeded
	Output   string // Tool output (JSON serialized data or error)
	Error    string // Error message if failed
	StepID   string // Unique step ID for this execution
	Blocked  bool   // True if blocked by safe mode
	NeedsAck bool   // True if safe mode requires acknowledgment
}

// ExecuteToolStep runs a single tool invocation with checkpointing.
// This is the primary entry point for the agent loop.
//
// Unlike ExecuteStep (for workflow Steps), this accepts a tool name and params
// and handles safe mode checks, execution via the tools registry, and checkpointing.
//
// The executor func is provided by the caller (typically tools.Registry.Get(name).Execute).
func (e *Engine) ExecuteToolStep(ctx context.Context, req StepRequest, executor func(ctx context.Context, params map[string]any) (success bool, output string, err error)) *StepResponse {
	stepID := GenerateRunID() // Unique ID for this step

	// Check safe mode for destructive operations
	if e.safeCtx != nil && e.safeCtx.Mode == SafeModeExecute {
		// Check if this is a destructive tool operation
		if e.isDestructiveTool(req.ToolName, req.Parameters) {
			// Use the approval function if set
			if e.safeCtx.ApprovalFunc != nil {
				action := fmt.Sprintf("Execute destructive tool: %s", req.ToolName)
				if !e.safeCtx.RequireApproval(action) {
					e.log("⚠ Safe mode: destructive tool %s denied by user", req.ToolName)
					return &StepResponse{
						Success:  false,
						StepID:   stepID,
						Blocked:  true,
						NeedsAck: true,
						Error:    fmt.Sprintf("safe mode: tool %s is destructive and was denied", req.ToolName),
					}
				}
			}
		}
	} else if e.safeCtx != nil && e.safeCtx.IsPreview() {
		// In preview mode, record but don't execute destructive tools
		if e.isDestructiveTool(req.ToolName, req.Parameters) {
			e.safeCtx.PreviewAction(stepID, fmt.Sprintf("Tool: %s", req.ToolName), fmt.Sprintf("params: %v", req.Parameters))
			return &StepResponse{
				Success:  false,
				StepID:   stepID,
				Blocked:  true,
				NeedsAck: true,
				Error:    fmt.Sprintf("preview mode: destructive tool %s would require approval", req.ToolName),
			}
		}
	}

	// Create step result for checkpointing
	result := &StepResult{
		StepID:    stepID,
		Status:    StepRunning,
		StartedAt: time.Now(),
	}

	// Save initial state if checkpointing is enabled
	if e.store != nil && req.RunID != "" {
		e.store.SaveStepResult(req.RunID, result)
	}

	e.log("▶ Executing tool: %s", req.ToolName)

	// Execute the tool
	success, output, err := executor(ctx, req.Parameters)

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if err != nil {
		result.Status = StepFailed
		result.Error = err.Error()
		result.Output = output
	} else if success {
		result.Status = StepSuccess
		result.Output = output
		result.ExitCode = 0
	} else {
		result.Status = StepFailed
		result.Output = output
		result.ExitCode = 1
	}

	// Save final state
	if e.store != nil && req.RunID != "" {
		e.store.SaveStepResult(req.RunID, result)
	}

	// Publish event
	e.publishEvent(pipeline.Event{
		Type:      pipeline.EventType("workflow.tool_step"),
		Timestamp: time.Now(),
		Source:    "workflow",
		BlockID:   stepID,
		Data: map[string]interface{}{
			"run_id":    req.RunID,
			"step_id":   stepID,
			"tool_name": req.ToolName,
			"status":    string(result.Status),
			"success":   success,
		},
	})

	if success {
		e.log("✓ Tool completed: %s", req.ToolName)
	} else {
		e.log("✗ Tool failed: %s", req.ToolName)
	}

	return &StepResponse{
		Success: success,
		Output:  output,
		Error:   result.Error,
		StepID:  stepID,
		Blocked: false,
	}
}

// isDestructiveTool checks if a tool with given params is destructive.
func (e *Engine) isDestructiveTool(toolName string, params map[string]any) bool {
	// Known destructive tools
	destructiveTools := map[string]bool{
		"write_file":  true,
		"run_command": true, // Commands can be destructive
	}

	if destructiveTools[toolName] {
		// For run_command, check the actual command for destructive patterns
		if toolName == "run_command" {
			if cmd, ok := params["command"].(string); ok {
				return e.safeCtx != nil && e.safeCtx.isDestructive(cmd)
			}
		}
		return true
	}

	return false
}

// AcknowledgeStep marks a blocked step as acknowledged, allowing it to proceed.
// Returns the updated step response after execution.
func (e *Engine) AcknowledgeStep(ctx context.Context, req StepRequest, executor func(ctx context.Context, params map[string]any) (success bool, output string, err error)) *StepResponse {
	// Temporarily set to execute mode with auto-approve
	originalMode := SafeModePreview
	var originalApprovalFunc func(string) bool

	if e.safeCtx != nil {
		originalMode = e.safeCtx.Mode
		originalApprovalFunc = e.safeCtx.ApprovalFunc
		e.safeCtx.Mode = SafeModeExecute
		e.safeCtx.ApprovalFunc = func(string) bool { return true } // Auto-approve
	}

	result := e.ExecuteToolStep(ctx, req, executor)

	// Restore original settings
	if e.safeCtx != nil {
		e.safeCtx.Mode = originalMode
		e.safeCtx.ApprovalFunc = originalApprovalFunc
	}

	return result
}
