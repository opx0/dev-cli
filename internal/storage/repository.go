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
func GetLastUnresolvedFailure(db *sql.DB) (*HistoryItem, error) {
	query := `SELECT id, timestamp, command, exit_code, duration_ms, directory, session_id, details, COALESCE(resolution, '')
			  FROM history 
			  WHERE exit_code != 0 AND exit_code != 130 AND (resolution IS NULL OR resolution = '')
			  ORDER BY id DESC LIMIT 1`

	row := db.QueryRow(query)
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
