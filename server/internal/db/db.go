// Package db provides database connection management.
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"agent-task-center/server/internal/config"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // SQLite driver
)

// Connect opens a database connection based on the config driver.
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	switch cfg.DBDriver {
	case "sqlite":
		return connectSQLite(cfg.DBPath)
	case "postgres":
		return connectPostgres(cfg.DBConnStr)
	default:
		return nil, fmt.Errorf("unsupported db driver: %q", cfg.DBDriver)
	}
}

func connectSQLite(path string) (*sqlx.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	dsn := path
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode and foreign keys via PRAGMA statements
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set journal_mode=WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set foreign_keys=ON: %w", err)
	}

	return db, nil
}

func connectPostgres(connStr string) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
