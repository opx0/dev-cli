package workflow

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"dev-cli/internal/executor"
)

// RollbackHook defines undo logic for a remediation step.
type RollbackHook struct {
	StepID    string      // ID of the step this rolls back
	Name      string      // Human-readable name
	Command   string      // Rollback command to execute
	Validator func() bool // Optional: check if rollback succeeded
	Priority  int         // Higher priority = execute first
}

// RollbackRegistry tracks rollback hooks for active remediations.
type RollbackRegistry struct {
	hooks []RollbackHook
	mu    sync.Mutex
}

// NewRollbackRegistry creates a new rollback registry.
func NewRollbackRegistry() *RollbackRegistry {
	return &RollbackRegistry{
		hooks: make([]RollbackHook, 0),
	}
}

// Register adds a rollback hook.
func (r *RollbackRegistry) Register(hook RollbackHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// Unregister removes a rollback hook by step ID.
func (r *RollbackRegistry) Unregister(stepID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]RollbackHook, 0, len(r.hooks))
	for _, hook := range r.hooks {
		if hook.StepID != stepID {
			filtered = append(filtered, hook)
		}
	}
	r.hooks = filtered
}

// Count returns the number of registered hooks.
func (r *RollbackRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.hooks)
}

// Clear removes all registered hooks.
func (r *RollbackRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = make([]RollbackHook, 0)
}

// RollbackResult represents the outcome of a single rollback.
type RollbackResult struct {
	StepID  string
	Name    string
	Success bool
	Output  string
	Error   error
}

// ExecuteAll runs all rollbacks in priority order (highest first).
// Returns results for each rollback attempt.
func (r *RollbackRegistry) ExecuteAll(ctx context.Context) []RollbackResult {
	r.mu.Lock()

	hooks := make([]RollbackHook, len(r.hooks))
	copy(hooks, r.hooks)
	r.mu.Unlock()

	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority > hooks[j].Priority
	})

	results := make([]RollbackResult, 0, len(hooks))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			results = append(results, RollbackResult{
				StepID:  hook.StepID,
				Name:    hook.Name,
				Success: false,
				Error:   ctx.Err(),
			})
			continue
		default:
		}

		result := r.executeRollback(ctx, hook)
		results = append(results, result)
	}

	return results
}

// executeRollback runs a single rollback hook.
func (r *RollbackRegistry) executeRollback(ctx context.Context, hook RollbackHook) RollbackResult {
	result := RollbackResult{
		StepID: hook.StepID,
		Name:   hook.Name,
	}

	execResult := executor.ExecuteWithContext(ctx, hook.Command)
	result.Output = execResult.Output
	result.Success = execResult.ExitCode == 0

	if !result.Success {
		result.Error = fmt.Errorf("rollback failed with exit code %d", execResult.ExitCode)
	}

	if result.Success && hook.Validator != nil {
		if !hook.Validator() {
			result.Success = false
			result.Error = fmt.Errorf("rollback validation failed")
		}
	}

	return result
}

// ExecuteForStep runs rollback only for a specific step.
func (r *RollbackRegistry) ExecuteForStep(ctx context.Context, stepID string) *RollbackResult {
	r.mu.Lock()
	var hook *RollbackHook
	for _, h := range r.hooks {
		if h.StepID == stepID {
			hCopy := h
			hook = &hCopy
			break
		}
	}
	r.mu.Unlock()

	if hook == nil {
		return nil
	}

	result := r.executeRollback(ctx, *hook)
	return &result
}

// GetHooks returns a copy of all registered hooks (for inspection).
func (r *RollbackRegistry) GetHooks() []RollbackHook {
	r.mu.Lock()
	defer r.mu.Unlock()

	hooks := make([]RollbackHook, len(r.hooks))
	copy(hooks, r.hooks)
	return hooks
}

// CreateRollbackHook is a helper to create a rollback hook from step info.
func CreateRollbackHook(stepID, name, command string, priority int) RollbackHook {
	return RollbackHook{
		StepID:   stepID,
		Name:     name,
		Command:  command,
		Priority: priority,
	}
}

// CreateRollbackHookWithValidator creates a hook with a validation function.
func CreateRollbackHookWithValidator(stepID, name, command string, priority int, validator func() bool) RollbackHook {
	return RollbackHook{
		StepID:    stepID,
		Name:      name,
		Command:   command,
		Priority:  priority,
		Validator: validator,
	}
}
