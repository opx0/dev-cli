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
		details     TEXT
	);

	`

	_, err := db.Exec(schema)
	return err
}
