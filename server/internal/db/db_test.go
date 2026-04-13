package db

import (
	"os"
	"path/filepath"
	"testing"

	"agent-task-center/server/internal/config"
)

func TestConnectSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	cfg := &config.Config{
		DBDriver: "sqlite",
		DBPath:   dbPath,
	}

	testDB, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer func() { _ = testDB.Close() }()

	if testDB == nil {
		t.Fatal("Connect() returned nil db")
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = testDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", journalMode)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = testDB.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}

	// Verify the DB file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestConnectSQLiteCreatesDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "nested", "test.db")

	cfg := &config.Config{
		DBDriver: "sqlite",
		DBPath:   dbPath,
	}

	testDB, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer func() { _ = testDB.Close() }()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created in nested directory")
	}
}

func TestConnectUnsupportedDriver(t *testing.T) {
	cfg := &config.Config{
		DBDriver: "mysql",
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}
