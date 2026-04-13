package db

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-task-center/server/internal/model"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// openQueryTestDB opens an in-memory SQLite DB with migrations applied.
func openQueryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}
	return db
}

// seedProj inserts a default project and returns its ID.
func seedProj(t *testing.T, db *sqlx.DB) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow("INSERT INTO projects (name) VALUES ('proj') RETURNING id").Scan(&id); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return id
}

// seedFeat inserts a default feature under a project and returns its ID.
func seedFeat(t *testing.T, db *sqlx.DB, projectID int64) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(
		"INSERT INTO features (project_id, slug, name) VALUES (?, 'feat', 'Feature') RETURNING id",
		projectID,
	).Scan(&id); err != nil {
		t.Fatalf("seed feature: %v", err)
	}
	return id
}

// seedRawTask inserts a raw task row with default values and returns its ID.
func seedRawTask(t *testing.T, db *sqlx.DB, featureID int64) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(
		`INSERT INTO tasks (feature_id, task_id, title, priority, status, dependencies)
		 VALUES (?, '1.1', 'Task', 'P0', 'pending', '[]') RETURNING id`,
		featureID,
	).Scan(&id); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return id
}

// ===========================================================================
// Projects
// ===========================================================================

func TestGetOrCreateProject_New(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	p, err := GetOrCreateProject(ctx, db, "my-project")
	if err != nil {
		t.Fatalf("GetOrCreateProject() error: %v", err)
	}
	if p.ID <= 0 {
		t.Error("expected positive ID")
	}
	if p.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", p.Name)
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestGetOrCreateProject_Idempotent(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	p1, err := GetOrCreateProject(ctx, db, "my-project")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	p2, err := GetOrCreateProject(ctx, db, "my-project")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if p1.ID != p2.ID {
		t.Errorf("expected same ID, got %d and %d", p1.ID, p2.ID)
	}
}

func TestGetOrCreateProject_DifferentNames(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	p1, err := GetOrCreateProject(ctx, db, "alpha")
	if err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	p2, err := GetOrCreateProject(ctx, db, "beta")
	if err != nil {
		t.Fatalf("create beta: %v", err)
	}
	if p1.ID == p2.ID {
		t.Error("different names should produce different IDs")
	}
}

func TestListProjects_Empty(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	projects, total, err := ListProjects(ctx, db, "", 1, 10)
	if err != nil {
		t.Fatalf("ListProjects() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(projects) != 0 {
		t.Errorf("expected empty list, got %d items", len(projects))
	}
}

func TestListProjects_WithSearch(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	if _, err := GetOrCreateProject(ctx, db, "alpha-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetOrCreateProject(ctx, db, "beta-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetOrCreateProject(ctx, db, "gamma"); err != nil {
		t.Fatal(err)
	}

	projects, total, err := ListProjects(ctx, db, "project", 1, 10)
	if err != nil {
		t.Fatalf("ListProjects() error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestListProjects_Pagination(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	for i := range 5 {
		if _, err := GetOrCreateProject(ctx, db, fmt.Sprintf("proj-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	// Page 1, size 2
	projects, total, err := ListProjects(ctx, db, "", 1, 2)
	if err != nil {
		t.Fatalf("ListProjects() page 1 error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects on page 1, got %d", len(projects))
	}

	// Page 3, size 2 — should have 1 item
	projects, _, err = ListProjects(ctx, db, "", 3, 2)
	if err != nil {
		t.Fatalf("ListProjects() page 3 error: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project on page 3, got %d", len(projects))
	}
}

func TestGetProject_Found(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	created, err := GetOrCreateProject(ctx, db, "test-project")
	if err != nil {
		t.Fatal(err)
	}
	found, err := GetProject(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("GetProject() error: %v", err)
	}
	if found.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", found.Name)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	_, err := GetProject(ctx, db, 99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ===========================================================================
// Proposals
// ===========================================================================

func TestUpsertProposal_Insert(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	input := model.ProposalInput{
		Slug:    "my-proposal",
		Title:   "My Proposal",
		Content: "# Hello World",
	}
	p, err := UpsertProposal(ctx, db, pid, input)
	if err != nil {
		t.Fatalf("UpsertProposal() error: %v", err)
	}
	if p.ID <= 0 {
		t.Error("expected positive ID")
	}
	if p.Slug != "my-proposal" {
		t.Errorf("expected slug 'my-proposal', got %q", p.Slug)
	}
	if p.Title != "My Proposal" {
		t.Errorf("expected title 'My Proposal', got %q", p.Title)
	}
}

func TestUpsertProposal_Update(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	input1 := model.ProposalInput{Slug: "slug", Title: "V1", Content: "v1"}
	p1, err := UpsertProposal(ctx, db, pid, input1)
	if err != nil {
		t.Fatal(err)
	}

	input2 := model.ProposalInput{Slug: "slug", Title: "V2", Content: "v2"}
	p2, err := UpsertProposal(ctx, db, pid, input2)
	if err != nil {
		t.Fatalf("UpsertProposal() update error: %v", err)
	}
	if p2.ID != p1.ID {
		t.Errorf("expected same ID on update, got %d and %d", p1.ID, p2.ID)
	}
	if p2.Title != "V2" {
		t.Errorf("expected updated title 'V2', got %q", p2.Title)
	}
}

func TestListProposalsByProject(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	if _, err := UpsertProposal(ctx, db, pid, model.ProposalInput{Slug: "a", Title: "A", Content: ""}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertProposal(ctx, db, pid, model.ProposalInput{Slug: "b", Title: "B", Content: ""}); err != nil {
		t.Fatal(err)
	}

	proposals, err := ListProposalsByProject(ctx, db, pid)
	if err != nil {
		t.Fatalf("ListProposalsByProject() error: %v", err)
	}
	if len(proposals) != 2 {
		t.Errorf("expected 2 proposals, got %d", len(proposals))
	}
}

func TestGetProposal_Found(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	created, err := UpsertProposal(ctx, db, pid, model.ProposalInput{Slug: "s", Title: "T", Content: "C"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := GetProposal(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("GetProposal() error: %v", err)
	}
	if found.Title != "T" {
		t.Errorf("expected title 'T', got %q", found.Title)
	}
}

func TestGetProposal_NotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	_, err := GetProposal(ctx, db, 99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ===========================================================================
// Features
// ===========================================================================

func TestUpsertFeature_Insert(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	input := model.FeatureInput{Slug: "feat", Name: "Feature", Status: "prd", Content: "content"}
	f, err := UpsertFeature(ctx, db, pid, input)
	if err != nil {
		t.Fatalf("UpsertFeature() error: %v", err)
	}
	if f.ID <= 0 {
		t.Error("expected positive ID")
	}
	if f.Slug != "feat" {
		t.Errorf("expected slug 'feat', got %q", f.Slug)
	}
}

func TestUpsertFeature_Update(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	input1 := model.FeatureInput{Slug: "feat", Name: "V1", Status: "prd", Content: "v1"}
	f1, err := UpsertFeature(ctx, db, pid, input1)
	if err != nil {
		t.Fatal(err)
	}

	input2 := model.FeatureInput{Slug: "feat", Name: "V2", Status: "done", Content: "v2"}
	f2, err := UpsertFeature(ctx, db, pid, input2)
	if err != nil {
		t.Fatalf("UpsertFeature() update error: %v", err)
	}
	if f2.ID != f1.ID {
		t.Errorf("expected same ID on update, got %d and %d", f1.ID, f2.ID)
	}
	if f2.Name != "V2" {
		t.Errorf("expected updated name 'V2', got %q", f2.Name)
	}
}

func TestListFeaturesByProject(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	if _, err := UpsertFeature(ctx, db, pid, model.FeatureInput{Slug: "f1", Name: "F1", Status: "prd", Content: ""}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertFeature(ctx, db, pid, model.FeatureInput{Slug: "f2", Name: "F2", Status: "done", Content: ""}); err != nil {
		t.Fatal(err)
	}

	features, err := ListFeaturesByProject(ctx, db, pid)
	if err != nil {
		t.Fatalf("ListFeaturesByProject() error: %v", err)
	}
	if len(features) != 2 {
		t.Errorf("expected 2 features, got %d", len(features))
	}
}

func TestGetFeatureBySlug_Found(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	if _, err := UpsertFeature(ctx, db, pid, model.FeatureInput{Slug: "my-feat", Name: "My Feature", Status: "prd", Content: ""}); err != nil {
		t.Fatal(err)
	}
	f, err := GetFeatureBySlug(ctx, db, pid, "my-feat")
	if err != nil {
		t.Fatalf("GetFeatureBySlug() error: %v", err)
	}
	if f.Name != "My Feature" {
		t.Errorf("expected name 'My Feature', got %q", f.Name)
	}
}

func TestGetFeatureBySlug_NotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	pid := seedProj(t, db)

	_, err := GetFeatureBySlug(ctx, db, pid, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ===========================================================================
// Tasks
// ===========================================================================

func TestUpsertTask_Insert(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	input := model.TaskInput{
		TaskID:       "1.1",
		Title:        "Task One",
		Description:  "Do something",
		Priority:     "P0",
		Tags:         []string{"core", "api"},
		Dependencies: []string{},
	}
	task, err := UpsertTask(ctx, db, fid, input)
	if err != nil {
		t.Fatalf("UpsertTask() error: %v", err)
	}
	if task.ID <= 0 {
		t.Error("expected positive ID")
	}
	if task.Status != "pending" {
		t.Errorf("expected default status 'pending', got %q", task.Status)
	}
	if task.Version != 0 {
		t.Errorf("expected default version 0, got %d", task.Version)
	}
	if task.ClaimedBy != "" {
		t.Errorf("expected empty claimed_by, got %q", task.ClaimedBy)
	}
	if !strings.Contains(task.Tags, "core") {
		t.Errorf("expected tags to contain 'core', got %q", task.Tags)
	}
}

func TestUpsertTask_UpdatePreservesStatus(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	// Create task
	input := model.TaskInput{TaskID: "1.1", Title: "V1", Description: "", Priority: "P1"}
	if _, err := UpsertTask(ctx, db, fid, input); err != nil {
		t.Fatal(err)
	}

	// Simulate someone claiming it
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = 'in_progress', claimed_by = 'agent-x', version = 1 WHERE feature_id = ? AND task_id = ?", fid, "1.1"); err != nil {
		t.Fatal(err)
	}

	// Upsert should not overwrite status/claimed_by/version
	input2 := model.TaskInput{TaskID: "1.1", Title: "V2", Description: "updated", Priority: "P0"}
	updated, err := UpsertTask(ctx, db, fid, input2)
	if err != nil {
		t.Fatalf("UpsertTask() update error: %v", err)
	}
	if updated.Title != "V2" {
		t.Errorf("expected updated title 'V2', got %q", updated.Title)
	}
	if updated.Status != "in_progress" {
		t.Errorf("expected status preserved as 'in_progress', got %q", updated.Status)
	}
	if updated.ClaimedBy != "agent-x" {
		t.Errorf("expected claimed_by preserved as 'agent-x', got %q", updated.ClaimedBy)
	}
	if updated.Version != 1 {
		t.Errorf("expected version preserved as 1, got %d", updated.Version)
	}
	if updated.Priority != "P0" {
		t.Errorf("expected priority updated to 'P0', got %q", updated.Priority)
	}
}

func TestUpsertTask_DefaultPriority(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	input := model.TaskInput{TaskID: "1.1", Title: "No Priority", Priority: ""}
	task, err := UpsertTask(ctx, db, fid, input)
	if err != nil {
		t.Fatalf("UpsertTask() error: %v", err)
	}
	if task.Priority != "P1" {
		t.Errorf("expected default priority 'P1', got %q", task.Priority)
	}
}

func TestListTasksByFeature_Basic(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "T2", Priority: "P1"}); err != nil {
		t.Fatal(err)
	}

	tasks, err := ListTasksByFeature(ctx, db, fid, model.TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasksByFeature() error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestListTasksByFeature_FilterByStatus(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	// Claim one task to change its status
	if _, err := ClaimTask(ctx, db, fid, "agent-1"); err != nil {
		t.Fatal(err)
	}

	tasks, err := ListTasksByFeature(ctx, db, fid, model.TaskFilter{Statuses: []string{"pending"}})
	if err != nil {
		t.Fatalf("ListTasksByFeature() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 pending tasks (only in_progress), got %d", len(tasks))
	}

	tasks, err = ListTasksByFeature(ctx, db, fid, model.TaskFilter{Statuses: []string{"in_progress"}})
	if err != nil {
		t.Fatalf("ListTasksByFeature() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 in_progress task, got %d", len(tasks))
	}
}

func TestListTasksByFeature_FilterByPriority(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "T2", Priority: "P2"}); err != nil {
		t.Fatal(err)
	}

	tasks, err := ListTasksByFeature(ctx, db, fid, model.TaskFilter{Priorities: []string{"P0"}})
	if err != nil {
		t.Fatalf("ListTasksByFeature() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 P0 task, got %d", len(tasks))
	}
	if tasks[0].Priority != "P0" {
		t.Errorf("expected P0, got %q", tasks[0].Priority)
	}
}

func TestListTasksByFeature_FilterByTags(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1", Tags: []string{"core", "api"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "T2", Tags: []string{"ui"}}); err != nil {
		t.Fatal(err)
	}

	tasks, err := ListTasksByFeature(ctx, db, fid, model.TaskFilter{Tags: []string{"core"}})
	if err != nil {
		t.Fatalf("ListTasksByFeature() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task with 'core' tag, got %d", len(tasks))
	}
	if tasks[0].TaskID != "1.1" {
		t.Errorf("expected task 1.1, got %q", tasks[0].TaskID)
	}
}

func TestGetTask_Found(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	created, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := GetTask(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("GetTask() error: %v", err)
	}
	if found.Title != "T1" {
		t.Errorf("expected title 'T1', got %q", found.Title)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	_, err := GetTask(ctx, db, 99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetTaskByTaskID(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "2.1", Title: "FindMe"}); err != nil {
		t.Fatal(err)
	}

	found, err := GetTaskByTaskID(ctx, db, fid, "2.1")
	if err != nil {
		t.Fatalf("GetTaskByTaskID() error: %v", err)
	}
	if found.Title != "FindMe" {
		t.Errorf("expected title 'FindMe', got %q", found.Title)
	}
}

func TestGetTaskByTaskID_NotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	_, err := GetTaskByTaskID(ctx, db, fid, "9.9")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ===========================================================================
// ClaimTask — optimistic locking
// ===========================================================================

func TestClaimTask_Basic(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}

	task, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != nil {
		t.Fatalf("ClaimTask() error: %v", err)
	}
	if task.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", task.Status)
	}
	if task.ClaimedBy != "agent-1" {
		t.Errorf("expected claimed_by 'agent-1', got %q", task.ClaimedBy)
	}
	if task.Version != 1 {
		t.Errorf("expected version 1 after claim, got %d", task.Version)
	}
}

func TestClaimTask_PriorityOrder(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "P2 Task", Priority: "P2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "P0 Task", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.3", Title: "P1 Task", Priority: "P1"}); err != nil {
		t.Fatal(err)
	}

	task, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != nil {
		t.Fatalf("ClaimTask() error: %v", err)
	}
	if task.Title != "P0 Task" {
		t.Errorf("expected P0 task claimed first, got %q", task.Title)
	}
}

func TestClaimTask_DependenciesMet(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	// Task 1.1 is already completed
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "Done Task", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE feature_id = ? AND task_id = ?", fid, "1.1"); err != nil {
		t.Fatal(err)
	}

	// Task 1.2 depends on 1.1 — should be claimable
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "Dependent Task", Priority: "P1", Dependencies: []string{"1.1"}}); err != nil {
		t.Fatal(err)
	}

	task, err := ClaimTask(ctx, db, fid, "agent-2")
	if err != nil {
		t.Fatalf("ClaimTask() error: %v", err)
	}
	if task.TaskID != "1.2" {
		t.Errorf("expected task 1.2 (deps met), got %q", task.TaskID)
	}
}

func TestClaimTask_DependenciesNotMet(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	// Task 1.1 is in_progress (already claimed by another agent)
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "In Progress", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = 'in_progress', claimed_by = 'other', version = 1 WHERE feature_id = ? AND task_id = ?", fid, "1.1"); err != nil {
		t.Fatal(err)
	}

	// Task 1.2 depends on 1.1 — should NOT be claimable because 1.1 is not completed
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "Blocked", Priority: "P1", Dependencies: []string{"1.1"}}); err != nil {
		t.Fatal(err)
	}

	_, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != ErrNoAvailableTask {
		t.Errorf("expected ErrNoAvailableTask for blocked deps, got %v", err)
	}
}

func TestClaimTask_NoAvailableTasks(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	// No tasks at all
	_, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != ErrNoAvailableTask {
		t.Errorf("expected ErrNoAvailableTask for empty feature, got %v", err)
	}
}

func TestClaimTask_SkipsCompletedTasks(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "Done", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE feature_id = ? AND task_id = ?", fid, "1.1"); err != nil {
		t.Fatal(err)
	}

	_, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != ErrNoAvailableTask {
		t.Errorf("expected ErrNoAvailableTask when only completed tasks exist, got %v", err)
	}
}

func TestClaimTask_ConcurrentContention(t *testing.T) {
	// Use file-based SQLite for concurrent goroutine access
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db, err := sqlx.Open("sqlite", dbPath+"?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var pid, fid int64
	if err := db.QueryRow("INSERT INTO projects (name) VALUES ('proj') RETURNING id").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("INSERT INTO features (project_id, slug, name) VALUES (?, 'feat', 'F') RETURNING id", pid).Scan(&fid); err != nil {
		t.Fatal(err)
	}

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "Contended", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
		errs    []error
	)

	for i := range 5 {
		wg.Add(1)
		go func(agentNum int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", agentNum)
			_, claimErr := ClaimTask(ctx, db, fid, agentID)
			mu.Lock()
			defer mu.Unlock()
			if claimErr != nil {
				errs = append(errs, claimErr)
			} else {
				success++
			}
		}(i)
	}
	wg.Wait()

	if success != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d", success)
	}
	for _, e := range errs {
		if e != ErrNoAvailableTask {
			t.Errorf("expected ErrNoAvailableTask for failed claims, got %v", e)
		}
	}
}

func TestClaimTask_MultipleCandidates_VersionConflictRetries(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	// Create 2 tasks; first one will be claimed by someone else before our ClaimTask
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "First", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.2", Title: "Second", Priority: "P1"}); err != nil {
		t.Fatal(err)
	}

	// Manually claim first task to simulate version conflict for first candidate
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = 'in_progress', claimed_by = 'other', version = 1 WHERE feature_id = ? AND task_id = ?", fid, "1.1"); err != nil {
		t.Fatal(err)
	}

	// ClaimTask should fall back to the second candidate
	task, err := ClaimTask(ctx, db, fid, "agent-1")
	if err != nil {
		t.Fatalf("ClaimTask() error: %v", err)
	}
	if task.TaskID != "1.2" {
		t.Errorf("expected fallback to task 1.2, got %q", task.TaskID)
	}
}

// ===========================================================================
// UpdateTaskStatus
// ===========================================================================

func TestUpdateTaskStatus_Success(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ClaimTask(ctx, db, fid, "agent-1"); err != nil {
		t.Fatal(err)
	}

	task, err := GetTaskByTaskID(ctx, db, fid, "1.1")
	if err != nil {
		t.Fatal(err)
	}
	if err := UpdateTaskStatus(ctx, db, task.ID, "agent-1", "completed"); err != nil {
		t.Fatalf("UpdateTaskStatus() error: %v", err)
	}

	updated, err := GetTask(ctx, db, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", updated.Status)
	}
}

func TestUpdateTaskStatus_WrongAgent(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ClaimTask(ctx, db, fid, "agent-1"); err != nil {
		t.Fatal(err)
	}

	task, err := GetTaskByTaskID(ctx, db, fid, "1.1")
	if err != nil {
		t.Fatal(err)
	}
	err = UpdateTaskStatus(ctx, db, task.ID, "agent-evil", "completed")
	if err != ErrUnauthorizedAgent {
		t.Errorf("expected ErrUnauthorizedAgent, got %v", err)
	}
}

func TestUpdateTaskStatus_TaskNotFound(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	err := UpdateTaskStatus(ctx, db, 99999, "agent-1", "completed")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateTaskStatus_CASCoversVersionCheck(t *testing.T) {
	// UpdateTaskStatus uses a CAS pattern with version check in the SQL WHERE clause.
	// Under true concurrency, if another agent modifies the row between the read and write,
	// rowsAffected will be 0, returning ErrVersionConflict.
	// This test verifies the success path through the CAS mechanism.
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))

	if _, err := UpsertTask(ctx, db, fid, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ClaimTask(ctx, db, fid, "agent-1"); err != nil {
		t.Fatal(err)
	}

	task, err := GetTaskByTaskID(ctx, db, fid, "1.1")
	if err != nil {
		t.Fatal(err)
	}

	// Verify the task was claimed with version 1
	if task.Version != 1 {
		t.Fatalf("expected version 1 after claim, got %d", task.Version)
	}

	// Normal status update should succeed (CAS match)
	if err := UpdateTaskStatus(ctx, db, task.ID, "agent-1", "completed"); err != nil {
		t.Fatalf("UpdateTaskStatus() success case error: %v", err)
	}
}

// ===========================================================================
// Execution Records
// ===========================================================================

func TestInsertRecord(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))
	tid := seedRawTask(t, db, fid)

	record := model.ExecutionRecord{
		AgentID:            "agent-1",
		Summary:            "Completed task",
		FilesCreated:       `["main.go"]`,
		FilesModified:      `["util.go"]`,
		KeyDecisions:       `["used sqlx"]`,
		TestsPassed:        5,
		TestsFailed:        0,
		Coverage:           85.5,
		AcceptanceCriteria: `[{"criterion":"works","met":true}]`,
	}
	result, err := InsertRecord(ctx, db, tid, record)
	if err != nil {
		t.Fatalf("InsertRecord() error: %v", err)
	}
	if result.ID <= 0 {
		t.Error("expected positive ID")
	}
	if result.TaskID != tid {
		t.Errorf("expected task_id %d, got %d", tid, result.TaskID)
	}
	if result.AgentID != "agent-1" {
		t.Errorf("expected agent_id 'agent-1', got %q", result.AgentID)
	}
	if result.Summary != "Completed task" {
		t.Errorf("expected summary 'Completed task', got %q", result.Summary)
	}
	if result.Coverage != 85.5 {
		t.Errorf("expected coverage 85.5, got %f", result.Coverage)
	}
	if result.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestInsertRecord_SetsCreatedAt(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))
	tid := seedRawTask(t, db, fid)

	before := time.Now()
	result, err := InsertRecord(ctx, db, tid, model.ExecutionRecord{
		AgentID: "a",
		Summary: "s",
	})
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()

	if result.CreatedAt.Before(before) || result.CreatedAt.After(after) {
		t.Errorf("expected created_at between %v and %v, got %v", before, after, result.CreatedAt)
	}
}

func TestListRecordsByTask(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))
	tid := seedRawTask(t, db, fid)

	for i := range 5 {
		if _, err := InsertRecord(ctx, db, tid, model.ExecutionRecord{
			AgentID: fmt.Sprintf("agent-%d", i),
			Summary: fmt.Sprintf("Run %d", i),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Page 1, size 3
	records, total, err := ListRecordsByTask(ctx, db, tid, 1, 3)
	if err != nil {
		t.Fatalf("ListRecordsByTask() error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records on page 1, got %d", len(records))
	}

	// Page 2
	records, _, err = ListRecordsByTask(ctx, db, tid, 2, 3)
	if err != nil {
		t.Fatalf("ListRecordsByTask() page 2 error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records on page 2, got %d", len(records))
	}
}

func TestListRecordsByTask_Empty(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))
	tid := seedRawTask(t, db, fid)

	records, total, err := ListRecordsByTask(ctx, db, tid, 1, 10)
	if err != nil {
		t.Fatalf("ListRecordsByTask() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected empty list, got %d records", len(records))
	}
}

func TestListRecordsByTask_OrderedDesc(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()
	fid := seedFeat(t, db, seedProj(t, db))
	tid := seedRawTask(t, db, fid)

	r1, err := InsertRecord(ctx, db, tid, model.ExecutionRecord{AgentID: "a", Summary: "first"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := InsertRecord(ctx, db, tid, model.ExecutionRecord{AgentID: "a", Summary: "second"})
	if err != nil {
		t.Fatal(err)
	}

	// Verify IDs are auto-incremented, so higher ID = more recent
	if r2.ID <= r1.ID {
		t.Fatalf("expected r2.ID > r1.ID, got %d and %d", r1.ID, r2.ID)
	}

	records, _, err := ListRecordsByTask(ctx, db, tid, 1, 10)
	if err != nil {
		t.Fatalf("ListRecordsByTask() error: %v", err)
	}
	// Most recent (higher ID) first (ORDER BY id DESC)
	if records[0].ID < records[1].ID {
		t.Errorf("expected most recent record first, got summaries in wrong order: %q then %q", records[0].Summary, records[1].Summary)
	}
}

// ===========================================================================
// Parameterized query verification (no SQL injection)
// ===========================================================================

func TestGetOrCreateProject_SQLInjectionSafe(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	malicious := "test'; DROP TABLE projects; --"
	p, err := GetOrCreateProject(ctx, db, malicious)
	if err != nil {
		t.Fatalf("GetOrCreateProject() error: %v", err)
	}
	if p.Name != malicious {
		t.Errorf("expected name to be stored literally, got %q", p.Name)
	}

	// Verify table still exists
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM projects"); err != nil {
		t.Fatalf("projects table should still exist: %v", err)
	}
}

func TestListProjects_SQLInjectionSafe(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	if _, err := GetOrCreateProject(ctx, db, "normal-project"); err != nil {
		t.Fatal(err)
	}

	search := "' OR 1=1 --"
	_, total, err := ListProjects(ctx, db, search, 1, 10)
	if err != nil {
		t.Fatalf("ListProjects() error: %v", err)
	}
	// The literal string "' OR 1=1 --" shouldn't match anything
	if total != 0 {
		t.Errorf("expected 0 results for injection attempt, got %d", total)
	}
}

// ===========================================================================
// Integration: full workflow test
// ===========================================================================

func TestFullWorkflow(t *testing.T) {
	db := openQueryTestDB(t)
	ctx := context.Background()

	// Create project
	project, err := GetOrCreateProject(ctx, db, "test-project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create feature
	feature, err := UpsertFeature(ctx, db, project.ID, model.FeatureInput{
		Slug: "agent-task-center", Name: "Agent Task Center", Status: "prd", Content: "",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Create tasks with dependencies
	if _, err := UpsertTask(ctx, db, feature.ID, model.TaskInput{
		TaskID: "1.1", Title: "Setup", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, feature.ID, model.TaskInput{
		TaskID: "1.2", Title: "Implement", Priority: "P1",
		Dependencies: []string{"1.1"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertTask(ctx, db, feature.ID, model.TaskInput{
		TaskID: "1.3", Title: "Test", Priority: "P2",
		Dependencies: []string{"1.2"},
	}); err != nil {
		t.Fatal(err)
	}

	// 1.2 and 1.3 should be blocked (deps not met)
	task, err := ClaimTask(ctx, db, feature.ID, "agent-1")
	if err != nil {
		t.Fatalf("claim first task: %v", err)
	}
	if task.TaskID != "1.1" {
		t.Errorf("expected 1.1 claimed, got %q", task.TaskID)
	}

	// No more claimable tasks (1.2 depends on 1.1 which is in_progress)
	_, err = ClaimTask(ctx, db, feature.ID, "agent-2")
	if err != ErrNoAvailableTask {
		t.Errorf("expected ErrNoAvailableTask, got %v", err)
	}

	// Complete 1.1
	if err := UpdateTaskStatus(ctx, db, task.ID, "agent-1", "completed"); err != nil {
		t.Fatal(err)
	}

	// Now 1.2 should be claimable
	task2, err := ClaimTask(ctx, db, feature.ID, "agent-2")
	if err != nil {
		t.Fatalf("claim second task: %v", err)
	}
	if task2.TaskID != "1.2" {
		t.Errorf("expected 1.2 claimed after 1.1 completed, got %q", task2.TaskID)
	}

	// Insert execution record
	if _, err := InsertRecord(ctx, db, task.ID, model.ExecutionRecord{
		AgentID: "agent-1",
		Summary: "Setup completed successfully",
	}); err != nil {
		t.Fatal(err)
	}

	// List records
	_, recTotal, err := ListRecordsByTask(ctx, db, task.ID, 1, 10)
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if recTotal != 1 {
		t.Errorf("expected 1 record, got %d", recTotal)
	}

	// Verify list tasks shows all statuses
	allTasks, err := ListTasksByFeature(ctx, db, feature.ID, model.TaskFilter{})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(allTasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(allTasks))
	}
}
