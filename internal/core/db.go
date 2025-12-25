package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func InitDB() (*sql.DB, error) {
	var dbPath string
	if envDir := os.Getenv("DEV_CLI_LOG_DIR"); envDir != "" {
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
		dbPath = filepath.Join(envDir, "history.db")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home dir: %w", err)
		}
		dir := filepath.Join(home, ".devlogs")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
		dbPath = filepath.Join(dir, "history.db")
	}
	return OpenDB(dbPath)
}

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp   INTEGER NOT NULL,
		command     TEXT NOT NULL,
		exit_code   INTEGER,
		duration_ms INTEGER,
		directory   TEXT,
		session_id  TEXT,
		details     TEXT,
		resolution  TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp);
	CREATE INDEX IF NOT EXISTS idx_history_exit_code ON history(exit_code);
	CREATE INDEX IF NOT EXISTS idx_history_session ON history(session_id);

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

	CREATE TABLE IF NOT EXISTS root_causes (
		id TEXT PRIMARY KEY,
		error_signature TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		root_cause_nodes TEXT,
		remediation_steps TEXT,
		confidence REAL DEFAULT 0.0,
		history_item_id INTEGER,
		FOREIGN KEY (history_item_id) REFERENCES history(id)
	);

	CREATE TABLE IF NOT EXISTS runbooks (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		name TEXT NOT NULL,
		description TEXT,
		steps TEXT NOT NULL,
		success_rate REAL DEFAULT 0.0,
		last_used INTEGER,
		usage_count INTEGER DEFAULT 0,
		tags TEXT
	);

	CREATE TABLE IF NOT EXISTS project_fingerprints (
		id TEXT PRIMARY KEY,
		project_type TEXT NOT NULL,
		package_manager TEXT,
		common_issues TEXT,
		associated_runbooks TEXT,
		detected_at TEXT NOT NULL,
		detected_time INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_root_cause_signature ON root_causes(error_signature);
	CREATE INDEX IF NOT EXISTS idx_root_cause_history ON root_causes(history_item_id);
	CREATE INDEX IF NOT EXISTS idx_runbook_project ON runbooks(project_id);
	CREATE INDEX IF NOT EXISTS idx_fingerprint_type ON project_fingerprints(project_type);
	CREATE INDEX IF NOT EXISTS idx_fingerprint_path ON project_fingerprints(detected_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	_, _ = db.Exec("ALTER TABLE history ADD COLUMN resolution TEXT")

	return nil
}

type LogEntry struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output,omitempty"`
	Cwd        string `json:"cwd"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
	SessionID  string `json:"session_id,omitempty"`
	Details    string `json:"details,omitempty"`
}

type HistoryItem struct {
	ID         int64
	Timestamp  time.Time
	Command    string
	ExitCode   int
	DurationMs int64
	Directory  string
	SessionID  string
	Details    string
	Resolution string
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
