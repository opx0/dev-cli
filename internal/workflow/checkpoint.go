package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CheckpointStore handles persistence of workflow run states.
type CheckpointStore struct {
	db *sql.DB
}

// NewCheckpointStore creates a new checkpoint store.
func NewCheckpointStore(db *sql.DB) *CheckpointStore {
	return &CheckpointStore{db: db}
}

// InitSchema creates the workflow tables if they don't exist.
func (s *CheckpointStore) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS workflow_runs (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		workflow_name TEXT,
		status TEXT NOT NULL,
		current_step INTEGER DEFAULT 0,
		started_at DATETIME,
		updated_at DATETIME,
		completed_at DATETIME,
		error TEXT
	);

	CREATE TABLE IF NOT EXISTS workflow_step_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		step_id TEXT NOT NULL,
		status TEXT NOT NULL,
		exit_code INTEGER,
		output TEXT,
		error TEXT,
		retries INTEGER DEFAULT 0,
		started_at DATETIME,
		completed_at DATETIME,
		duration_ms INTEGER,
		FOREIGN KEY (run_id) REFERENCES workflow_runs(id)
	);

	CREATE INDEX IF NOT EXISTS idx_workflow_runs_status ON workflow_runs(status);
	CREATE INDEX IF NOT EXISTS idx_step_results_run_id ON workflow_step_results(run_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveRun persists or updates a workflow run state.
func (s *CheckpointStore) SaveRun(state *RunState) error {
	query := `
	INSERT OR REPLACE INTO workflow_runs 
		(id, workflow_id, workflow_name, status, current_step, started_at, updated_at, completed_at, error)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var completedAt *time.Time
	if !state.CompletedAt.IsZero() {
		completedAt = &state.CompletedAt
	}

	_, err := s.db.Exec(query,
		state.RunID,
		state.WorkflowID,
		state.WorkflowName,
		string(state.Status),
		state.CurrentStepIdx,
		state.StartedAt,
		state.UpdatedAt,
		completedAt,
		state.Error,
	)

	return err
}

// SaveStepResult persists a step execution result.
func (s *CheckpointStore) SaveStepResult(runID string, result *StepResult) error {
	query := `
	INSERT OR REPLACE INTO workflow_step_results 
		(run_id, step_id, status, exit_code, output, error, retries, started_at, completed_at, duration_ms)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var completedAt *time.Time
	if !result.CompletedAt.IsZero() {
		completedAt = &result.CompletedAt
	}

	_, err := s.db.Exec(query,
		runID,
		result.StepID,
		string(result.Status),
		result.ExitCode,
		truncateString(result.Output, 10240),
		result.Error,
		result.Retries,
		result.StartedAt,
		completedAt,
		result.Duration.Milliseconds(),
	)

	return err
}

// LoadRun retrieves a workflow run state by ID.
func (s *CheckpointStore) LoadRun(runID string) (*RunState, error) {
	query := `
	SELECT id, workflow_id, workflow_name, status, current_step, started_at, updated_at, completed_at, error
	FROM workflow_runs WHERE id = ?
	`

	row := s.db.QueryRow(query, runID)

	state := &RunState{
		StepResults: make(map[string]*StepResult),
	}

	var completedAt sql.NullTime
	var errStr sql.NullString
	var status string

	err := row.Scan(
		&state.RunID,
		&state.WorkflowID,
		&state.WorkflowName,
		&status,
		&state.CurrentStepIdx,
		&state.StartedAt,
		&state.UpdatedAt,
		&completedAt,
		&errStr,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", runID)
	}
	if err != nil {
		return nil, err
	}

	state.Status = RunStatus(status)
	if completedAt.Valid {
		state.CompletedAt = completedAt.Time
	}
	if errStr.Valid {
		state.Error = errStr.String
	}

	stepResults, err := s.LoadStepResults(runID)
	if err != nil {
		return nil, err
	}
	state.StepResults = stepResults

	return state, nil
}

// LoadStepResults retrieves all step results for a run.
func (s *CheckpointStore) LoadStepResults(runID string) (map[string]*StepResult, error) {
	query := `
	SELECT step_id, status, exit_code, output, error, retries, started_at, completed_at, duration_ms
	FROM workflow_step_results WHERE run_id = ? ORDER BY started_at
	`

	rows, err := s.db.Query(query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]*StepResult)

	for rows.Next() {
		result := &StepResult{}
		var status string
		var completedAt sql.NullTime
		var errStr sql.NullString
		var durationMs int64

		err := rows.Scan(
			&result.StepID,
			&status,
			&result.ExitCode,
			&result.Output,
			&errStr,
			&result.Retries,
			&result.StartedAt,
			&completedAt,
			&durationMs,
		)
		if err != nil {
			return nil, err
		}

		result.Status = StepStatus(status)
		result.Duration = time.Duration(durationMs) * time.Millisecond
		if completedAt.Valid {
			result.CompletedAt = completedAt.Time
		}
		if errStr.Valid {
			result.Error = errStr.String
		}

		results[result.StepID] = result
	}

	return results, rows.Err()
}

// ListRuns returns recent workflow runs.
func (s *CheckpointStore) ListRuns(limit int) ([]*RunState, error) {
	query := `
	SELECT id, workflow_id, workflow_name, status, current_step, started_at, updated_at, completed_at, error
	FROM workflow_runs ORDER BY started_at DESC LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*RunState

	for rows.Next() {
		state := &RunState{
			StepResults: make(map[string]*StepResult),
		}

		var completedAt sql.NullTime
		var errStr sql.NullString
		var status string

		err := rows.Scan(
			&state.RunID,
			&state.WorkflowID,
			&state.WorkflowName,
			&status,
			&state.CurrentStepIdx,
			&state.StartedAt,
			&state.UpdatedAt,
			&completedAt,
			&errStr,
		)
		if err != nil {
			return nil, err
		}

		state.Status = RunStatus(status)
		if completedAt.Valid {
			state.CompletedAt = completedAt.Time
		}
		if errStr.Valid {
			state.Error = errStr.String
		}

		runs = append(runs, state)
	}

	return runs, rows.Err()
}

// DeleteRun removes a workflow run and its step results.
func (s *CheckpointStore) DeleteRun(runID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM workflow_step_results WHERE run_id = ?", runID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM workflow_runs WHERE id = ?", runID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-20] + "\n...[truncated]..."
}

// MarshalRunState serializes run state to JSON.
func MarshalRunState(state *RunState) ([]byte, error) {
	return json.Marshal(state)
}

// UnmarshalRunState deserializes run state from JSON.
func UnmarshalRunState(data []byte) (*RunState, error) {
	var state RunState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
