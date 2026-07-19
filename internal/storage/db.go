package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"dev-cli/internal/config"

	_ "modernc.org/sqlite"
)

// --- Singleton DB access ---

var (
	dbOnce sync.Once
	dbInst *sql.DB
	dbErr  error
)

// DB returns a lazily-initialized singleton database connection.
// It uses the configured data directory for the database path.
// Panics if the database cannot be opened — this ensures a fast fail
// at startup rather than silent errors.
func DB() *sql.DB {
	dbOnce.Do(func() {
		dbInst, dbErr = InitDB()
	})
	if dbErr != nil {
		panic("storage: failed to init db: " + dbErr.Error())
	}
	return dbInst
}

func InitDB() (*sql.DB, error) {
	dir := config.Load().LogDir
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	// #nosec G302 -- 0700 is the intended owner-only permission for a directory.
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("secure data dir: %w", err)
	}
	return OpenDB(filepath.Join(dir, "history.db"))
}

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("secure db: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
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

	CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp);
	CREATE INDEX IF NOT EXISTS idx_history_exit_code ON history(exit_code);
	CREATE INDEX IF NOT EXISTS idx_history_session ON history(session_id);

	`

	_, err := db.ExecContext(context.Background(), schema)
	if err != nil {
		return err
	}

	return nil
}
