package workflow

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseFile reads a workflow definition from a YAML file.
func ParseFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return Parse(data)
}

// Parse parses a workflow definition from YAML bytes.
func Parse(data []byte) (*Workflow, error) {
	var raw rawWorkflow
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return raw.toWorkflow()
}

// rawWorkflow is the YAML structure with string durations.
type rawWorkflow struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Steps       []rawStep         `yaml:"steps"`
	OnFailure   *FailurePolicy    `yaml:"on_failure"`
	Env         map[string]string `yaml:"env"`
}

type rawStep struct {
	ID        string            `yaml:"id"`
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command"`
	Condition *Condition        `yaml:"condition"`
	OnSuccess string            `yaml:"on_success"`
	OnFailure string            `yaml:"on_failure"`
	Rollback  *rawRollback      `yaml:"rollback"`
	Timeout   string            `yaml:"timeout"`
	Retries   int               `yaml:"retries"`
	Env       map[string]string `yaml:"env"`
	WorkDir   string            `yaml:"workdir"`
}

type rawRollback struct {
	Command string `yaml:"command"`
	Timeout string `yaml:"timeout"`
}

// UnmarshalYAML allows rollback to be specified as just a string.
func (r *rawRollback) UnmarshalYAML(node *yaml.Node) error {

	if node.Kind == yaml.ScalarNode {
		r.Command = node.Value
		return nil
	}

	// Otherwise, parse as struct
	type plain rawRollback
	return node.Decode((*plain)(r))
}

func (rw *rawWorkflow) toWorkflow() (*Workflow, error) {
	wf := &Workflow{
		ID:          rw.ID,
		Name:        rw.Name,
		Description: rw.Description,
		OnFailure:   rw.OnFailure,
		Env:         rw.Env,
		Steps:       make([]Step, 0, len(rw.Steps)),
	}

	if wf.ID == "" {
		wf.ID = generateID()
	}

	for i, rs := range rw.Steps {
		step, err := rs.toStep(i)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i, rs.ID, err)
		}
		wf.Steps = append(wf.Steps, step)
	}

	if err := validateWorkflow(wf); err != nil {
		return nil, err
	}

	return wf, nil
}

func (rs *rawStep) toStep(index int) (Step, error) {
	step := Step{
		ID:        rs.ID,
		Name:      rs.Name,
		Command:   rs.Command,
		Condition: rs.Condition,
		OnSuccess: rs.OnSuccess,
		OnFailure: rs.OnFailure,
		Retries:   rs.Retries,
		Env:       rs.Env,
		WorkDir:   rs.WorkDir,
	}

	if step.ID == "" {
		step.ID = fmt.Sprintf("step_%d", index)
	}

	if rs.Timeout != "" {
		d, err := time.ParseDuration(rs.Timeout)
		if err != nil {
			return step, fmt.Errorf("invalid timeout %q: %w", rs.Timeout, err)
		}
		step.Timeout = d
	} else {
		step.Timeout = 5 * time.Minute
	}

	if rs.Rollback != nil {
		step.Rollback = &RollbackAction{
			Command: rs.Rollback.Command,
		}
		if rs.Rollback.Timeout != "" {
			d, err := time.ParseDuration(rs.Rollback.Timeout)
			if err != nil {
				return step, fmt.Errorf("invalid rollback timeout: %w", err)
			}
			step.Rollback.Timeout = d
		} else {
			step.Rollback.Timeout = 2 * time.Minute
		}
	}

	return step, nil
}

func validateWorkflow(wf *Workflow) error {
	if wf.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(wf.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	stepIDs := make(map[string]bool)
	for _, step := range wf.Steps {
		if step.Command == "" {
			return fmt.Errorf("step %q: command is required", step.ID)
		}

		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	for _, step := range wf.Steps {
		if step.OnSuccess != "" && !stepIDs[step.OnSuccess] {
			return fmt.Errorf("step %q: on_success references unknown step %q", step.ID, step.OnSuccess)
		}
		if step.OnFailure != "" && step.OnFailure != "abort" && step.OnFailure != "rollback" && step.OnFailure != "continue" {
			if !stepIDs[step.OnFailure] {
				return fmt.Errorf("step %q: on_failure references unknown step %q", step.ID, step.OnFailure)
			}
		}
	}

	return nil
}

// generateID creates a simple unique ID based on timestamp.
func generateID() string {
	return fmt.Sprintf("wf_%d", time.Now().UnixNano())
}

// GenerateRunID creates a unique run ID.
func GenerateRunID() string {
	return fmt.Sprintf("run_%d", time.Now().UnixNano())
}
