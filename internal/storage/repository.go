package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type LogEntry struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output,omitempty"`
	Cwd        string `json:"cwd"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"` // RFC3339 string
	SessionID  string `json:"session_id,omitempty"`
	Details    string `json:"details,omitempty"` // JSON string if pre-marshaled, or we construct it
}

type HistoryItem struct {
	ID         int64
	Timestamp  time.Time
	Command    string
	ExitCode   int
	DurationMs int64
	Directory  string
	SessionID  string
	Details    string // Raw JSON
	Resolution string // "solution", "unrelated", "skipped", or "" (empty)
}

func SaveCommand(db *sql.DB, entry LogEntry) error {
	ts, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		ts = time.Now()
	}

	detailsMap := map[string]interface{}{
		"output": entry.Output,
	}

	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	query := `INSERT INTO history (timestamp, command, exit_code, duration_ms, directory, session_id, details)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query, ts.Unix(), entry.Command, entry.ExitCode, entry.DurationMs, entry.Cwd, entry.SessionID, string(detailsJSON))
	return err
}

func GetRecentHistory(db *sql.DB, limit int) ([]HistoryItem, error) {
	query := `SELECT id, timestamp, command, exit_code, duration_ms, directory, session_id, details, COALESCE(resolution, '') 
			  FROM history ORDER BY id DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		var ts int64
		if err := rows.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution); err != nil {
			return nil, err
		}
		item.Timestamp = time.Unix(ts, 0)
		items = append(items, item)
	}
	return items, nil
}

func SearchHistory(db *sql.DB, query string) ([]HistoryItem, error) {
	sqlQuery := `SELECT id, timestamp, command, exit_code, duration_ms, directory, session_id, details, COALESCE(resolution, '') 
				 FROM history 
				 WHERE command LIKE ? OR details LIKE ?
				 ORDER BY id DESC
				 LIMIT 50`

	wildcard := "%" + query + "%"
	rows, err := db.Query(sqlQuery, wildcard, wildcard)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		var ts int64
		if err := rows.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution); err != nil {
			return nil, err
		}
		item.Timestamp = time.Unix(ts, 0)
		items = append(items, item)
	}
	return items, nil
}

type QueryOpts struct {
	Limit  int
	Filter string
	Since  time.Duration
}

func GetFailures(db *sql.DB, opts QueryOpts) ([]HistoryItem, error) {
	queryBuilder := `SELECT h.id, h.timestamp, h.command, h.exit_code, h.duration_ms, h.directory, h.session_id, h.details, COALESCE(h.resolution, '') 
					 FROM history h`
	var args []interface{}
	var whereClauses []string

	whereClauses = append(whereClauses, "h.exit_code != 0")

	if opts.Filter != "" {
		whereClauses = append(whereClauses, "h.command LIKE ?")
		args = append(args, "%"+opts.Filter+"%")
	}

	if opts.Since > 0 {
		cutoff := time.Now().Add(-opts.Since).Unix()
		whereClauses = append(whereClauses, "h.timestamp >= ?")
		args = append(args, cutoff)
	}

	if len(whereClauses) > 0 {
		queryBuilder += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	queryBuilder += " ORDER BY h.id DESC"

	if opts.Limit > 0 {
		queryBuilder += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := db.Query(queryBuilder, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		var ts int64
		if err := rows.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution); err != nil {
			return nil, err
		}
		item.Timestamp = time.Unix(ts, 0)
		items = append(items, item)
	}
	return items, nil
}

// GetLastUnresolvedFailure returns the most recent failed command that hasn't been resolved.
// Only returns failures from the last 5 minutes to avoid prompting for stale failures.
func GetLastUnresolvedFailure(db *sql.DB) (*HistoryItem, error) {
	// Only consider failures from the last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute).Unix()

	query := `SELECT id, timestamp, command, exit_code, duration_ms, directory, session_id, details, COALESCE(resolution, '')
			  FROM history 
			  WHERE exit_code != 0 AND exit_code != 130 AND (resolution IS NULL OR resolution = '')
			  AND timestamp >= ?
			  ORDER BY id DESC LIMIT 1`

	row := db.QueryRow(query, cutoff)
	var item HistoryItem
	var ts int64
	err := row.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Timestamp = time.Unix(ts, 0)
	return &item, nil
}

// GetHistoryByID retrieves a specific history item by ID.
func GetHistoryByID(db *sql.DB, id int64) (*HistoryItem, error) {
	query := `SELECT id, timestamp, command, exit_code, duration_ms, directory, session_id, details, COALESCE(resolution, '')
			  FROM history WHERE id = ?`

	row := db.QueryRow(query, id)
	var item HistoryItem
	var ts int64
	err := row.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Timestamp = time.Unix(ts, 0)
	return &item, nil
}

// MarkResolution updates the resolution status of a history entry.
// Valid values: "solution", "unrelated", "skipped"
func MarkResolution(db *sql.DB, id int64, resolution string) error {
	query := `UPDATE history SET resolution = ? WHERE id = ?`
	result, err := db.Exec(query, resolution, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("history entry not found: %d", id)
	}
	return nil
}

// Solution represents a known fix for an error pattern.
type Solution struct {
	ID                  string    `json:"id"`
	ErrorSignature      string    `json:"error_signature"`
	SolutionCommand     string    `json:"solution_command"`
	SolutionDescription string    `json:"solution_description,omitempty"`
	SuccessCount        int       `json:"success_count"`
	FailureCount        int       `json:"failure_count"`
	CreatedAt           time.Time `json:"created_at"`
	LastUsedAt          time.Time `json:"last_used_at"`
}

// StoreSolution saves a new solution for an error signature.
func StoreSolution(db *sql.DB, errorSig, command, description string) error {
	id := GenerateErrorSignature(command, 0, errorSig) // Generate unique ID
	now := time.Now().Unix()

	query := `INSERT INTO solutions (id, error_signature, solution_command, solution_description, success_count, failure_count, created_at, last_used_at)
			  VALUES (?, ?, ?, ?, 1, 0, ?, ?)
			  ON CONFLICT(id) DO UPDATE SET 
			  success_count = success_count + 1,
			  last_used_at = ?`

	_, err := db.Exec(query, id, errorSig, command, description, now, now, now)
	return err
}

// GetSolutionsForError retrieves solutions matching an error signature.
func GetSolutionsForError(db *sql.DB, errorSignature string) ([]Solution, error) {
	query := `SELECT id, error_signature, solution_command, COALESCE(solution_description, ''), 
			  success_count, failure_count, created_at, last_used_at
			  FROM solutions 
			  WHERE error_signature = ?
			  ORDER BY success_count DESC
			  LIMIT 10`

	rows, err := db.Query(query, errorSignature)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var solutions []Solution
	for rows.Next() {
		var s Solution
		var createdAt, lastUsedAt int64
		if err := rows.Scan(&s.ID, &s.ErrorSignature, &s.SolutionCommand, &s.SolutionDescription,
			&s.SuccessCount, &s.FailureCount, &createdAt, &lastUsedAt); err != nil {
			return nil, err
		}
		s.CreatedAt = time.Unix(createdAt, 0)
		s.LastUsedAt = time.Unix(lastUsedAt, 0)
		solutions = append(solutions, s)
	}
	return solutions, nil
}

// GetSimilarFailures finds past failed commands with similar error patterns.
func GetSimilarFailures(db *sql.DB, errorSignature string, limit int) ([]HistoryItem, error) {
	// First, try to find exact signature matches in root_causes
	query := `SELECT h.id, h.timestamp, h.command, h.exit_code, h.duration_ms, h.directory, h.session_id, h.details, COALESCE(h.resolution, '')
			  FROM history h
			  JOIN root_causes rc ON h.id = rc.history_item_id
			  WHERE rc.error_signature = ?
			  ORDER BY h.id DESC
			  LIMIT ?`

	rows, err := db.Query(query, errorSignature, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		var ts int64
		if err := rows.Scan(&item.ID, &ts, &item.Command, &item.ExitCode, &item.DurationMs, &item.Directory, &item.SessionID, &item.Details, &item.Resolution); err != nil {
			return nil, err
		}
		item.Timestamp = time.Unix(ts, 0)
		items = append(items, item)
	}
	return items, nil
}

// IncrementSolutionSuccess updates the success count for a solution.
func IncrementSolutionSuccess(db *sql.DB, solutionID string) error {
	query := `UPDATE solutions SET success_count = success_count + 1, last_used_at = ? WHERE id = ?`
	_, err := db.Exec(query, time.Now().Unix(), solutionID)
	return err
}

// IncrementSolutionFailure updates the failure count for a solution.
func IncrementSolutionFailure(db *sql.DB, solutionID string) error {
	query := `UPDATE solutions SET failure_count = failure_count + 1, last_used_at = ? WHERE id = ?`
	_, err := db.Exec(query, time.Now().Unix(), solutionID)
	return err
}
