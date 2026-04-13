package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

// MigrationsFS holds the embedded SQL migration files.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS

// RunMigrations runs all pending database migrations using the embedded SQL files.
// It is idempotent: calling it multiple times is safe and will not error if already up to date.
func RunMigrations(db *sqlx.DB, driverName string) error {
	sourceDriver, err := iofs.New(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	dbInstance, err := asMigrateDB(db, driverName)
	if err != nil {
		return fmt.Errorf("create migration database: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, driverName, dbInstance)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// asMigrateDB wraps a *sqlx.DB into a migrate database driver.
func asMigrateDB(db *sqlx.DB, driverName string) (database.Driver, error) {
	switch driverName {
	case "sqlite":
		return sqlite.WithInstance(db.DB, &sqlite.Config{})
	default:
		return nil, fmt.Errorf("unsupported migration driver: %q", driverName)
	}
}

// TableName is a helper for tests to verify a table exists.
func TableName(db *sqlx.DB, name string) (bool, error) {
	var count int
	err := db.Get(&count,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name)
	if err != nil {
		return false, fmt.Errorf("check table %q: %w", name, err)
	}
	return count > 0, nil
}

// AllTableNames returns the five expected table names for verification.
func AllTableNames() []string {
	return []string{
		"projects",
		"proposals",
		"features",
		"tasks",
		"execution_records",
	}
}

//nolint:unused,revive // keep for future postgres support
func asMigratePostgresDB(_ *sql.DB) (database.Driver, error) {
	// Will be implemented when postgres support is needed for migrations.
	// import _ "github.com/golang-migrate/migrate/v4/database/postgres"
	return nil, fmt.Errorf("postgres migration driver not yet implemented")
}
