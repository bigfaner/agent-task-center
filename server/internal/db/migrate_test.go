package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Enable foreign keys for consistency
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	return db
}

func TestRunMigrationsCreatesAllTables(t *testing.T) {
	db := openTestDB(t)

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	for _, name := range AllTableNames() {
		exists, err := TableName(db, name)
		if err != nil {
			t.Fatalf("check table %q: %v", name, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist after migration", name)
		}
	}
}

func TestRunMigrationsIdempotent(t *testing.T) {
	db := openTestDB(t)

	// Run migrations twice - should not error
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("first RunMigrations() error: %v", err)
	}
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("second RunMigrations() error: %v", err)
	}

	// Verify tables still exist
	for _, name := range AllTableNames() {
		exists, err := TableName(db, name)
		if err != nil {
			t.Fatalf("check table %q: %v", name, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist after double migration", name)
		}
	}
}

func TestDownMigrationRollback(t *testing.T) {
	db := openTestDB(t)

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	// Verify tables exist before rollback
	for _, name := range AllTableNames() {
		exists, err := TableName(db, name)
		if err != nil {
			t.Fatalf("check table %q before rollback: %v", name, err)
		}
		if !exists {
			t.Fatalf("table %q should exist before rollback", name)
		}
	}

	// Execute down migration directly
	for _, name := range AllTableNames() {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + name); err != nil {
			t.Fatalf("drop table %q: %v", name, err)
		}
	}

	// Verify tables no longer exist
	for _, name := range AllTableNames() {
		exists, err := TableName(db, name)
		if err != nil {
			t.Fatalf("check table %q after rollback: %v", name, err)
		}
		if exists {
			t.Errorf("expected table %q to be dropped after rollback", name)
		}
	}
}

func TestSchemaColumns(t *testing.T) {
	db := openTestDB(t)

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	tests := []struct {
		table   string
		columns []string
	}{
		{
			table: "projects",
			columns: []string{
				"id", "name", "created_at", "updated_at",
			},
		},
		{
			table: "proposals",
			columns: []string{
				"id", "project_id", "slug", "title", "content", "created_at", "updated_at",
			},
		},
		{
			table: "features",
			columns: []string{
				"id", "project_id", "slug", "name", "status", "content", "created_at", "updated_at",
			},
		},
		{
			table: "tasks",
			columns: []string{
				"id", "feature_id", "task_id", "title", "description", "status",
				"priority", "tags", "dependencies", "claimed_by", "version",
				"created_at", "updated_at",
			},
		},
		{
			table: "execution_records",
			columns: []string{
				"id", "task_id", "agent_id", "summary",
				"files_created", "files_modified", "key_decisions",
				"tests_passed", "tests_failed", "coverage",
				"acceptance_criteria", "created_at",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			for _, col := range tt.columns {
				var count int
				err := db.Get(&count,
					"SELECT count(*) FROM pragma_table_info(?) WHERE name=?",
					tt.table, col)
				if err != nil {
					t.Fatalf("check column %q in %q: %v", col, tt.table, err)
				}
				if count != 1 {
					t.Errorf("expected column %q in table %q, got count=%d", col, tt.table, count)
				}
			}
		})
	}
}

func TestTasksDefaults(t *testing.T) {
	db := openTestDB(t)

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	// Create prerequisite rows
	_, err := db.Exec("INSERT INTO projects (name) VALUES (?)", "test-project")
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	_, err = db.Exec("INSERT INTO features (project_id, slug, name) VALUES (1, 'test-feature', 'Test Feature')")
	if err != nil {
		t.Fatalf("insert feature: %v", err)
	}

	// Insert task with minimal fields
	_, err = db.Exec(
		"INSERT INTO tasks (feature_id, task_id, title) VALUES (1, '1.1', 'Test Task')",
	)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	// Verify default values
	var status, priority, tags, deps, claimedBy string
	var version int64
	err = db.QueryRow(
		"SELECT status, priority, tags, dependencies, claimed_by, version FROM tasks WHERE id=1",
	).Scan(&status, &priority, &tags, &deps, &claimedBy, &version)
	if err != nil {
		t.Fatalf("query task defaults: %v", err)
	}

	if status != "pending" {
		t.Errorf("expected default status='pending', got %q", status)
	}
	if priority != "P1" {
		t.Errorf("expected default priority='P1', got %q", priority)
	}
	if tags != "[]" {
		t.Errorf("expected default tags='[]', got %q", tags)
	}
	if deps != "[]" {
		t.Errorf("expected default dependencies='[]', got %q", deps)
	}
	if claimedBy != "" {
		t.Errorf("expected default claimed_by='', got %q", claimedBy)
	}
	if version != 0 {
		t.Errorf("expected default version=0, got %d", version)
	}
}

func TestUniqueConstraints(t *testing.T) {
	db := openTestDB(t)

	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	// Insert prerequisite
	_, _ = db.Exec("INSERT INTO projects (name) VALUES (?)", "proj1")

	// Test unique constraint on projects.name
	_, err := db.Exec("INSERT INTO projects (name) VALUES (?)", "proj1")
	if err == nil {
		t.Error("expected error for duplicate project name")
	}

	// Test unique constraint on proposals(project_id, slug)
	_, _ = db.Exec("INSERT INTO proposals (project_id, slug, title) VALUES (1, 'slug1', 'Title')")
	_, err = db.Exec("INSERT INTO proposals (project_id, slug, title) VALUES (1, 'slug1', 'Title2')")
	if err == nil {
		t.Error("expected error for duplicate proposal slug within project")
	}

	// Test unique constraint on features(project_id, slug)
	_, _ = db.Exec("INSERT INTO features (project_id, slug, name) VALUES (1, 'feat1', 'Feature')")
	_, err = db.Exec("INSERT INTO features (project_id, slug, name) VALUES (1, 'feat1', 'Feature2')")
	if err == nil {
		t.Error("expected error for duplicate feature slug within project")
	}

	// Test unique constraint on tasks(feature_id, task_id)
	_, _ = db.Exec("INSERT INTO tasks (feature_id, task_id, title) VALUES (1, '1.1', 'Task')")
	_, err = db.Exec("INSERT INTO tasks (feature_id, task_id, title) VALUES (1, '1.1', 'Task2')")
	if err == nil {
		t.Error("expected error for duplicate task_id within feature")
	}
}

func TestRunMigrationsUnsupportedDriver(t *testing.T) {
	db := openTestDB(t)

	err := RunMigrations(db, "postgres")
	if err == nil {
		t.Fatal("expected error for unsupported migration driver, got nil")
	}
}
