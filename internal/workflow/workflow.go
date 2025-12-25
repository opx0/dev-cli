// Package workflow provides multi-step workflow automation with conditional
// branching, rollback capabilities, and checkpoint/resume functionality.
package workflow

import (
	"time"
)

// RunStatus represents the current state of a workflow run.
type RunStatus string

const (
	StatusPending    RunStatus = "pending"
	StatusRunning    RunStatus = "running"
	StatusPaused     RunStatus = "paused"
	StatusCompleted  RunStatus = "completed"
	StatusFailed     RunStatus = "failed"
	StatusRolledBack RunStatus = "rolledback"
)

// StepStatus represents the current state of a step execution.
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepRunning    StepStatus = "running"
	StepSuccess    StepStatus = "success"
	StepFailed     StepStatus = "failed"
	StepSkipped    StepStatus = "skipped"
	StepRolledBack StepStatus = "rolledback"
)

// ConditionType defines how a condition should be evaluated.
type ConditionType string

const (
	CondExitCode       ConditionType = "exit_code"
	CondOutputContains ConditionType = "output_contains"
	CondOutputMatches  ConditionType = "output_matches"
	CondFileExists     ConditionType = "file_exists"
	CondEnvSet         ConditionType = "env_set"
)

// FailureAction defines what to do when a workflow fails.
type FailureAction string

const (
	FailureAbort    FailureAction = "abort"
	FailureRollback FailureAction = "rollback"
	FailureContinue FailureAction = "continue"
)

// Condition specifies when a step should execute.
type Condition struct {
	Type  ConditionType `yaml:"type"`
	Value string        `yaml:"value"`
	// StepRef references a previous step's result (optional, defaults to previous step)
	StepRef string `yaml:"step_ref,omitempty"`
}

// RollbackAction defines how to undo a step.
type RollbackAction struct {
	Command string        `yaml:"command"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// Step represents a single executable action in a workflow.
type Step struct {
	ID        string            `yaml:"id"`
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command"`
	Condition *Condition        `yaml:"condition,omitempty"`
	OnSuccess string            `yaml:"on_success,omitempty"` // Next step ID (optional)
	OnFailure string            `yaml:"on_failure,omitempty"` // Step ID, "rollback", or "abort"
	Rollback  *RollbackAction   `yaml:"rollback,omitempty"`
	Timeout   time.Duration     `yaml:"timeout,omitempty"`
	Retries   int               `yaml:"retries,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	WorkDir   string            `yaml:"workdir,omitempty"`
}

// FailurePolicy defines workflow-level failure handling.
type FailurePolicy struct {
	Action FailureAction `yaml:"action"`
}

// Workflow represents a complete multi-step automation definition.
type Workflow struct {
	ID          string            `yaml:"id,omitempty"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Steps       []Step            `yaml:"steps"`
	OnFailure   *FailurePolicy    `yaml:"on_failure,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

// StepResult holds the outcome of executing a single step.
type StepResult struct {
	StepID      string
	Status      StepStatus
	ExitCode    int
	Output      string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Retries     int
}

// RunState holds the complete state of a workflow execution.
type RunState struct {
	RunID          string
	WorkflowID     string
	WorkflowName   string
	Status         RunStatus
	CurrentStepIdx int
	StepResults    map[string]*StepResult
	StartedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    time.Time
	Error          string
}

// NewRunState creates a new RunState for a workflow execution.
func NewRunState(runID string, wf *Workflow) *RunState {
	return &RunState{
		RunID:        runID,
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Status:       StatusPending,
		StepResults:  make(map[string]*StepResult),
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// GetStepResult returns the result for a given step ID.
func (r *RunState) GetStepResult(stepID string) *StepResult {
	return r.StepResults[stepID]
}

// SetStepResult stores the result for a step.
func (r *RunState) SetStepResult(result *StepResult) {
	r.StepResults[result.StepID] = result
	r.UpdatedAt = time.Now()
}

// LastStepResult returns the most recently completed step result.
func (r *RunState) LastStepResult() *StepResult {
	var last *StepResult
	for _, result := range r.StepResults {
		if last == nil || result.CompletedAt.After(last.CompletedAt) {
			last = result
		}
	}
	return last
}
