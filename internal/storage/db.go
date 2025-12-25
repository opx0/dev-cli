package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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

	-- Workflow automation tables
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

	-- RCA (Root Cause Analysis) tables
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

	-- Solutions table: Maps error patterns to known fixes
	CREATE TABLE IF NOT EXISTS solutions (
		id TEXT PRIMARY KEY,
		error_signature TEXT NOT NULL,
		solution_command TEXT NOT NULL,
		solution_description TEXT,
		success_count INTEGER DEFAULT 0,
		failure_count INTEGER DEFAULT 0,
		created_at INTEGER,
		last_used_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_solutions_signature ON solutions(error_signature);
	CREATE INDEX IF NOT EXISTS idx_solutions_success ON solutions(success_count DESC);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	_, _ = db.Exec("ALTER TABLE history ADD COLUMN resolution TEXT")

	return nil
}
