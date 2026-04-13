package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/model"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// openServiceTestDB opens an in-memory SQLite DB with migrations applied.
func openServiceTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	d, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if _, err := d.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := db.RunMigrations(d, "sqlite"); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}
	return d
}

// ===========================================================================
// ProjectService tests
// ===========================================================================

func TestProjectService_Upsert_New(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p, err := svc.Upsert(ctx, "my-project")
	if err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}
	if p.ID <= 0 {
		t.Error("expected positive ID")
	}
	if p.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", p.Name)
	}
}

func TestProjectService_Upsert_Idempotent(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p1, err := svc.Upsert(ctx, "my-project")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	p2, err := svc.Upsert(ctx, "my-project")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if p1.ID != p2.ID {
		t.Errorf("expected same ID on idempotent upsert, got %d and %d", p1.ID, p2.ID)
	}
}

func TestProjectService_List_Empty(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	items, total, err := svc.List(ctx, "", 1, 20)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestProjectService_List_WithSearch(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	if _, err := svc.Upsert(ctx, "alpha-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, "beta-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, "gamma"); err != nil {
		t.Fatal(err)
	}

	items, total, err := svc.List(ctx, "project", 1, 20)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestProjectService_List_Pagination(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	for i := range 5 {
		name := "proj-" + string(rune('0'+i))
		if _, err := svc.Upsert(ctx, name); err != nil {
			t.Fatal(err)
		}
	}

	items, total, err := svc.List(ctx, "", 1, 2)
	if err != nil {
		t.Fatalf("List() page 1 error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(items))
	}

	items2, _, err := svc.List(ctx, "", 3, 2)
	if err != nil {
		t.Fatalf("List() page 3 error: %v", err)
	}
	if len(items2) != 1 {
		t.Errorf("expected 1 item on page 3, got %d", len(items2))
	}
}

func TestProjectService_List_CompletionRate(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p, err := svc.Upsert(ctx, "proj")
	if err != nil {
		t.Fatal(err)
	}

	// Create a feature under this project directly via db
	f, err := db.UpsertFeature(ctx, d, p.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create 4 tasks: 2 completed, 1 in_progress, 1 pending
	taskInputs := []model.TaskInput{
		{TaskID: "1.1", Title: "T1", Priority: "P0"},
		{TaskID: "1.2", Title: "T2", Priority: "P0"},
		{TaskID: "1.3", Title: "T3", Priority: "P1"},
		{TaskID: "1.4", Title: "T4", Priority: "P1"},
	}
	for _, ti := range taskInputs {
		if _, err := db.UpsertTask(ctx, d, f.ID, ti); err != nil {
			t.Fatal(err)
		}
	}

	// Mark 2 tasks as completed
	if _, err := d.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE task_id IN ('1.1', '1.2') AND feature_id = ?", f.ID); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(ctx, "", 1, 20)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 project, got %d", len(items))
	}

	proj := items[0]
	if proj.FeatureCount != 1 {
		t.Errorf("expected FeatureCount 1, got %d", proj.FeatureCount)
	}
	if proj.TaskTotal != 4 {
		t.Errorf("expected TaskTotal 4, got %d", proj.TaskTotal)
	}
	// 2 completed out of 4 = 50.0
	if proj.CompletionRate != 50.0 {
		t.Errorf("expected CompletionRate 50.0, got %f", proj.CompletionRate)
	}
}

func TestProjectService_List_CompletionRate_ZeroTasks(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	if _, err := svc.Upsert(ctx, "empty-proj"); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(ctx, "", 1, 20)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 project, got %d", len(items))
	}
	if items[0].CompletionRate != 0 {
		t.Errorf("expected CompletionRate 0 for project with no tasks, got %f", items[0].CompletionRate)
	}
	if items[0].FeatureCount != 0 {
		t.Errorf("expected FeatureCount 0, got %d", items[0].FeatureCount)
	}
	if items[0].TaskTotal != 0 {
		t.Errorf("expected TaskTotal 0, got %d", items[0].TaskTotal)
	}
}

func TestProjectService_Get_Found(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p, err := svc.Upsert(ctx, "test-project")
	if err != nil {
		t.Fatal(err)
	}

	// Add proposals and features
	if _, err := db.UpsertProposal(ctx, d, p.ID, model.ProposalInput{
		Slug: "my-proposal", Title: "My Proposal", Content: "# Hello",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertFeature(ctx, d, p.ID, model.FeatureInput{
		Slug: "my-feature", Name: "My Feature", Status: "prd", Content: "",
	}); err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if detail.ID != p.ID {
		t.Errorf("expected ID %d, got %d", p.ID, detail.ID)
	}
	if detail.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", detail.Name)
	}
	if len(detail.Proposals) != 1 {
		t.Errorf("expected 1 proposal, got %d", len(detail.Proposals))
	}
	// FeatureCount counts all features in the project (no direct proposal-feature link in schema)
	if detail.Proposals[0].FeatureCount != 1 {
		t.Errorf("expected proposal FeatureCount 1, got %d", detail.Proposals[0].FeatureCount)
	}
	if len(detail.Features) != 1 {
		t.Errorf("expected 1 feature, got %d", len(detail.Features))
	}
}

func TestProjectService_Get_NotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	_, err := svc.Get(ctx, 99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectService_Get_ProposalFeatureCount(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p, err := svc.Upsert(ctx, "test-proj")
	if err != nil {
		t.Fatal(err)
	}

	// Create proposal
	proposal, err := db.UpsertProposal(ctx, d, p.ID, model.ProposalInput{
		Slug: "proposal-1", Title: "P1", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create features linked to the proposal (by having the same project_id)
	// In the current schema, features are linked to project, not directly to proposal.
	// So FeatureCount for a proposal counts all features in the project for now.
	// Let's verify the current behavior.
	_, err = db.UpsertFeature(ctx, d, p.ID, model.FeatureInput{
		Slug: "feat-1", Name: "F1", Status: "prd", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.UpsertFeature(ctx, d, p.ID, model.FeatureInput{
		Slug: "feat-2", Name: "F2", Status: "done", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	_ = proposal // proposal exists
	if len(detail.Proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(detail.Proposals))
	}
	// ProposalSummary.FeatureCount = count of features in the project
	if detail.Proposals[0].FeatureCount != 2 {
		t.Errorf("expected proposal FeatureCount 2, got %d", detail.Proposals[0].FeatureCount)
	}
}

func TestProjectService_Get_FeatureCompletionRate(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProjectService(d)
	ctx := context.Background()

	p, err := svc.Upsert(ctx, "proj")
	if err != nil {
		t.Fatal(err)
	}

	f, err := db.UpsertFeature(ctx, d, p.ID, model.FeatureInput{
		Slug: "feat", Name: "F", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create 2 tasks, 1 completed
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.2", Title: "T2", Priority: "P1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE task_id = '1.1' AND feature_id = ?", f.ID); err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if len(detail.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(detail.Features))
	}
	// 1 completed out of 2 = 50.0
	if detail.Features[0].CompletionRate != 50.0 {
		t.Errorf("expected feature CompletionRate 50.0, got %f", detail.Features[0].CompletionRate)
	}
}

// ===========================================================================
// FeatureService tests
// ===========================================================================

func TestFeatureService_ListByProject(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	// Get the actual ID
	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	pid := proj.ID

	// Create features
	if _, err := db.UpsertFeature(ctx, d, pid, model.FeatureInput{
		Slug: "feat-1", Name: "Feature 1", Status: "prd", Content: "",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertFeature(ctx, d, pid, model.FeatureInput{
		Slug: "feat-2", Name: "Feature 2", Status: "done", Content: "",
	}); err != nil {
		t.Fatal(err)
	}

	features, err := svc.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(features))
	}
	if features[0].Slug != "feat-1" {
		t.Errorf("expected first feature slug 'feat-1', got %q", features[0].Slug)
	}
}

func TestFeatureService_ListByProject_CompletionRate(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}

	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3 tasks, 1 completed
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.2", Title: "T2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.3", Title: "T3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE task_id = '1.1' AND feature_id = ?", f.ID); err != nil {
		t.Fatal(err)
	}

	features, err := svc.ListByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	// 1 completed out of 3 = 33.333...
	expected := float64(1) / float64(3) * 100
	if features[0].CompletionRate < expected-0.01 || features[0].CompletionRate > expected+0.01 {
		t.Errorf("expected CompletionRate ~%.2f, got %.2f", expected, features[0].CompletionRate)
	}
}

func TestFeatureService_ListByProject_CompletionRate_ZeroTasks(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "empty-feat", Name: "Empty", Status: "prd", Content: "",
	}); err != nil {
		t.Fatal(err)
	}

	features, err := svc.ListByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}
	if features[0].CompletionRate != 0 {
		t.Errorf("expected CompletionRate 0 for feature with no tasks, got %f", features[0].CompletionRate)
	}
}

func TestFeatureService_GetTasks(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{
		TaskID: "1.1", Title: "Task 1", Priority: "P0", Tags: []string{"core"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{
		TaskID: "1.2", Title: "Task 2", Priority: "P1", Tags: []string{"ui"},
	}); err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].TaskID != "1.1" {
		t.Errorf("expected first task 1.1, got %q", tasks[0].TaskID)
	}
}

func TestFeatureService_GetTasks_FilterByStatus(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1"}); err != nil {
		t.Fatal(err)
	}
	// Claim the task to change status
	if _, err := db.ClaimTask(ctx, d, f.ID, "agent-1"); err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{Statuses: []string{"pending"}})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 pending tasks, got %d", len(tasks))
	}

	tasks, err = svc.GetTasks(ctx, f.ID, model.TaskFilter{Statuses: []string{"in_progress"}})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 in_progress task, got %d", len(tasks))
	}
}

func TestFeatureService_GetTasks_FilterByPriority(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.2", Title: "T2", Priority: "P2"}); err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{Priorities: []string{"P0"}})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 P0 task, got %d", len(tasks))
	}
	if tasks[0].Priority != "P0" {
		t.Errorf("expected P0, got %q", tasks[0].Priority)
	}
}

func TestFeatureService_GetTasks_FilterByTags(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1", Tags: []string{"core", "api"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.2", Title: "T2", Tags: []string{"ui"}}); err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{Tags: []string{"core"}})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task with 'core' tag, got %d", len(tasks))
	}
	if tasks[0].TaskID != "1.1" {
		t.Errorf("expected task 1.1, got %q", tasks[0].TaskID)
	}
}

func TestFeatureService_GetTasks_Empty(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "empty-feat", Name: "Empty", Status: "prd", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestFeatureService_GetTasks_CombinedFilters(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.1", Title: "T1", Priority: "P0", Tags: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.2", Title: "T2", Priority: "P1", Tags: []string{"core"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{TaskID: "1.3", Title: "T3", Priority: "P0", Tags: []string{"ui"}}); err != nil {
		t.Fatal(err)
	}

	// Filter: P0 + core tag -> should return only 1.1
	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{
		Priorities: []string{"P0"},
		Tags:       []string{"core"},
	})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task matching P0+core, got %d", len(tasks))
	}
	if tasks[0].TaskID != "1.1" {
		t.Errorf("expected task 1.1, got %q", tasks[0].TaskID)
	}
}

// Verify that Task JSON tags/dependencies are properly deserialized in GetTasks
func TestFeatureService_GetTasks_TaskFieldsDeserialized(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}
	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat", Name: "Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, f.ID, model.TaskInput{
		TaskID: "1.1", Title: "Task With Tags", Priority: "P0",
		Tags:         []string{"core", "api"},
		Dependencies: []string{"0.1"},
	}); err != nil {
		t.Fatal(err)
	}

	tasks, err := svc.GetTasks(ctx, f.ID, model.TaskFilter{})
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	// Verify tags JSON is properly set
	var tags []string
	if err := json.Unmarshal([]byte(task.Tags), &tags); err != nil {
		t.Fatalf("failed to unmarshal tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "core" || tags[1] != "api" {
		t.Errorf("expected tags [core, api], got %v", tags)
	}

	// Verify dependencies JSON
	var deps []string
	if err := json.Unmarshal([]byte(task.Dependencies), &deps); err != nil {
		t.Fatalf("failed to unmarshal dependencies: %v", err)
	}
	if len(deps) != 1 || deps[0] != "0.1" {
		t.Errorf("expected dependencies [0.1], got %v", deps)
	}
}

// ===========================================================================
// TaskService tests
// ===========================================================================

// seedTaskServiceTest creates a project + feature for TaskService tests.
// Returns (projectID, featureID).
func seedTaskServiceTest(t *testing.T, d *sqlx.DB) (int64, int64) {
	t.Helper()
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "test-project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "test-feature", Name: "Test Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	return proj.ID, f.ID
}

// ---------------------------------------------------------------------------
// TaskService.Get
// ---------------------------------------------------------------------------

func TestTaskService_Get_Found(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	task, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Test Task", Description: "desc", Priority: "P0",
		Tags: []string{"core"}, Dependencies: []string{"0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	detail, err := svc.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if detail.ID != task.ID {
		t.Errorf("expected ID %d, got %d", task.ID, detail.ID)
	}
	if detail.TaskID != "1.1" {
		t.Errorf("expected taskId '1.1', got %q", detail.TaskID)
	}
	if detail.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", detail.Title)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "core" {
		t.Errorf("expected tags [core], got %v", detail.Tags)
	}
	if len(detail.Dependencies) != 1 || detail.Dependencies[0] != "0.1" {
		t.Errorf("expected dependencies [0.1], got %v", detail.Dependencies)
	}
}

func TestTaskService_Get_NotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, err := svc.Get(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService.GetByTaskID
// ---------------------------------------------------------------------------

func TestTaskService_GetByTaskID_Found(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	_, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "3.2", Title: "TaskService", Priority: "P0", Tags: []string{"service"},
	})
	if err != nil {
		t.Fatal(err)
	}

	detail, err := svc.GetByTaskID(ctx, "test-project", "test-feature", "3.2")
	if err != nil {
		t.Fatalf("GetByTaskID() error: %v", err)
	}
	if detail.TaskID != "3.2" {
		t.Errorf("expected taskId '3.2', got %q", detail.TaskID)
	}
	if detail.Title != "TaskService" {
		t.Errorf("expected title 'TaskService', got %q", detail.Title)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "service" {
		t.Errorf("expected tags [service], got %v", detail.Tags)
	}
}

func TestTaskService_GetByTaskID_FeatureNotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, err := db.GetOrCreateProject(ctx, d, "test-project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.GetByTaskID(ctx, "test-project", "nonexistent", "1.1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_GetByTaskID_TaskNotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	seedTaskServiceTest(t, d)

	_, err := svc.GetByTaskID(ctx, "test-project", "test-feature", "9.9")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService.Claim
// ---------------------------------------------------------------------------

func TestTaskService_Claim_Success(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	_, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task 1", Priority: "P0",
	})
	if err != nil {
		t.Fatal(err)
	}

	task, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatalf("Claim() error: %v", err)
	}
	if task.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", task.Status)
	}
	if task.ClaimedBy != "agent-01" {
		t.Errorf("expected claimed_by 'agent-01', got %q", task.ClaimedBy)
	}
	if task.Version != 1 {
		t.Errorf("expected version 1, got %d", task.Version)
	}
}

func TestTaskService_Claim_NoAvailableTask(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	seedTaskServiceTest(t, d)

	// No tasks created => no available tasks
	_, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if !errors.Is(err, ErrNoAvailableTask) {
		t.Errorf("expected ErrNoAvailableTask, got %v", err)
	}
}

func TestTaskService_Claim_AllTasksClaimed(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	_, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task 1", Priority: "P0",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Claim the only task
	_, err = svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	// Try claiming again => no available tasks
	_, err = svc.Claim(ctx, "test-project", "test-feature", "agent-02")
	if !errors.Is(err, ErrNoAvailableTask) {
		t.Errorf("expected ErrNoAvailableTask, got %v", err)
	}
}

func TestTaskService_Claim_PriorityOrder(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{TaskID: "1.1", Title: "P1 Task", Priority: "P1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{TaskID: "1.2", Title: "P0 Task", Priority: "P0"}); err != nil {
		t.Fatal(err)
	}

	task, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatalf("Claim() error: %v", err)
	}
	if task.TaskID != "1.2" {
		t.Errorf("expected P0 task '1.2', got %q", task.TaskID)
	}
}

func TestTaskService_Claim_DependencyUnmet(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Dep", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.2", Title: "Blocked", Priority: "P0",
		Dependencies: []string{"1.1"},
	}); err != nil {
		t.Fatal(err)
	}

	task, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatalf("Claim() error: %v", err)
	}
	if task.TaskID != "1.1" {
		t.Errorf("expected task '1.1', got %q", task.TaskID)
	}
}

func TestTaskService_Claim_DependencyMet(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	t1, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Dep", Priority: "P0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, "UPDATE tasks SET status = 'completed' WHERE id = ?", t1.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.2", Title: "Ready", Priority: "P1",
		Dependencies: []string{"1.1"},
	}); err != nil {
		t.Fatal(err)
	}

	task, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatalf("Claim() error: %v", err)
	}
	if task.TaskID != "1.2" {
		t.Errorf("expected task '1.2', got %q", task.TaskID)
	}
}

func TestTaskService_Claim_FeatureNotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, err := db.GetOrCreateProject(ctx, d, "test-project")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Claim(ctx, "test-project", "nonexistent-feature", "agent-01")
	if !errors.Is(err, ErrNoAvailableTask) {
		t.Errorf("expected ErrNoAvailableTask, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService.UpdateStatus
// ---------------------------------------------------------------------------

func TestTaskService_UpdateStatus_Success(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}

	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateStatus(ctx, claimed.ID, "agent-01", "blocked")
	if err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	updated, err := db.GetTask(ctx, d, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "blocked" {
		t.Errorf("expected status 'blocked', got %q", updated.Status)
	}
}

func TestTaskService_UpdateStatus_UnauthorizedAgent(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateStatus(ctx, claimed.ID, "agent-02", "blocked")
	if !errors.Is(err, ErrUnauthorizedAgent) {
		t.Errorf("expected ErrUnauthorizedAgent, got %v", err)
	}
}

func TestTaskService_UpdateStatus_InvalidStatus(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateStatus(ctx, claimed.ID, "agent-01", "invalid_status")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestTaskService_UpdateStatus_TaskNotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	err := svc.UpdateStatus(ctx, 99999, "agent-01", "in_progress")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_UpdateStatus_CompletedNotValid(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	// "completed" is not a valid status for UpdateStatus (should go through SubmitRecord)
	err = svc.UpdateStatus(ctx, claimed.ID, "agent-01", "completed")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus for 'completed', got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService.SubmitRecord
// ---------------------------------------------------------------------------

func TestTaskService_SubmitRecord_Success(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	record := model.ExecutionRecord{
		Summary:       "Completed the task",
		FilesCreated:  `["file1.go"]`,
		FilesModified: `["file2.go"]`,
		TestsPassed:   10,
		TestsFailed:   0,
		Coverage:      85.5,
	}

	saved, err := svc.SubmitRecord(ctx, claimed.ID, "agent-01", record)
	if err != nil {
		t.Fatalf("SubmitRecord() error: %v", err)
	}
	if saved.ID <= 0 {
		t.Error("expected positive record ID")
	}
	if saved.AgentID != "agent-01" {
		t.Errorf("expected agent_id 'agent-01', got %q", saved.AgentID)
	}
	if saved.Summary != "Completed the task" {
		t.Errorf("unexpected summary: %q", saved.Summary)
	}

	updated, err := db.GetTask(ctx, d, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", updated.Status)
	}
}

func TestTaskService_SubmitRecord_UnauthorizedAgent(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	record := model.ExecutionRecord{Summary: "hacked"}
	_, err = svc.SubmitRecord(ctx, claimed.ID, "agent-02", record)
	if !errors.Is(err, ErrUnauthorizedAgent) {
		t.Errorf("expected ErrUnauthorizedAgent, got %v", err)
	}
}

func TestTaskService_SubmitRecord_TaskNotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	record := model.ExecutionRecord{Summary: "test"}
	_, err := svc.SubmitRecord(ctx, 99999, "agent-01", record)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskService_SubmitRecord_SetsTaskCompleted(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	if claimed.Status != "in_progress" {
		t.Fatalf("expected initial status 'in_progress', got %q", claimed.Status)
	}

	_, err = svc.SubmitRecord(ctx, claimed.ID, "agent-01", model.ExecutionRecord{
		Summary: "done",
	})
	if err != nil {
		t.Fatalf("SubmitRecord() error: %v", err)
	}

	task, err := db.GetTask(ctx, d, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "completed" {
		t.Errorf("expected task status 'completed' after SubmitRecord, got %q", task.Status)
	}
}

// ---------------------------------------------------------------------------
// TaskService.ListRecords
// ---------------------------------------------------------------------------

func TestTaskService_ListRecords_Empty(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	records, total, err := svc.ListRecords(ctx, 99999, 1, 10)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestTaskService_ListRecords_WithData(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		_, err := db.InsertRecord(ctx, d, claimed.ID, model.ExecutionRecord{
			AgentID: "agent-01",
			Summary: fmt.Sprintf("Record %d", i+1),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := svc.ListRecords(ctx, claimed.ID, 1, 2)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records on page 1, got %d", len(records))
	}
	if records[0].Summary != "Record 3" {
		t.Errorf("expected first record 'Record 3', got %q", records[0].Summary)
	}
	if records[1].Summary != "Record 2" {
		t.Errorf("expected second record 'Record 2', got %q", records[1].Summary)
	}

	page2, _, err := svc.ListRecords(ctx, claimed.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListRecords() page 2 error: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 record on page 2, got %d", len(page2))
	}
}

// ---------------------------------------------------------------------------
// TaskService.Claim — Optimistic Lock
// ---------------------------------------------------------------------------

func TestTaskService_Claim_OptimisticLock(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	// Create a single pending task
	if _, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Contested Task", Priority: "P0",
	}); err != nil {
		t.Fatal(err)
	}

	// First claim succeeds — agent-01 gets the task
	task1, err := svc.Claim(ctx, "test-project", "test-feature", "agent-01")
	if err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}
	if task1.ClaimedBy != "agent-01" {
		t.Errorf("expected claimed_by 'agent-01', got %q", task1.ClaimedBy)
	}
	if task1.Version != 1 {
		t.Errorf("expected version 1, got %d", task1.Version)
	}

	// Second claim by a different agent fails — no more pending tasks
	_, err = svc.Claim(ctx, "test-project", "test-feature", "agent-02")
	if !errors.Is(err, ErrNoAvailableTask) {
		t.Errorf("expected ErrNoAvailableTask when all tasks claimed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService — UpsertTask Idempotent
// ---------------------------------------------------------------------------

func TestTaskService_UpsertTask_Idempotent(t *testing.T) {
	d := openServiceTestDB(t)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	// First upsert — no dependencies so it can be claimed
	task1, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Original Title", Description: "Original Desc", Priority: "P0",
		Tags: []string{"core"},
	})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if task1.Status != "pending" {
		t.Errorf("expected initial status 'pending', got %q", task1.Status)
	}

	// Claim the task
	claimed, err := db.ClaimTask(ctx, d, featID, "agent-01")
	if err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if claimed.Status != "in_progress" {
		t.Fatalf("expected status 'in_progress' after claim, got %q", claimed.Status)
	}

	// Second upsert with same task_id — should NOT overwrite status or claimed_by
	task2, err := db.UpsertTask(ctx, d, featID, model.TaskInput{
		TaskID: "1.1", Title: "Updated Title", Description: "Updated Desc", Priority: "P1",
		Tags: []string{"new-tag"},
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Status and claimed_by should be preserved
	if task2.Status != "in_progress" {
		t.Errorf("expected status preserved as 'in_progress', got %q", task2.Status)
	}
	if task2.ClaimedBy != "agent-01" {
		t.Errorf("expected claimed_by preserved as 'agent-01', got %q", task2.ClaimedBy)
	}
	// Title and description should be updated
	if task2.Title != "Updated Title" {
		t.Errorf("expected title updated to 'Updated Title', got %q", task2.Title)
	}
	if task2.Description != "Updated Desc" {
		t.Errorf("expected description updated, got %q", task2.Description)
	}
	if task2.Priority != "P1" {
		t.Errorf("expected priority updated to 'P1', got %q", task2.Priority)
	}
}

// ===========================================================================
// FeatureService.GetByID tests
// ===========================================================================

func TestFeatureService_GetByID_Found(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}

	f, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "feat-1", Name: "Feature 1", Status: "prd", Content: "content",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.GetByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if result.ID != f.ID {
		t.Errorf("expected ID %d, got %d", f.ID, result.ID)
	}
	if result.Slug != "feat-1" {
		t.Errorf("expected slug 'feat-1', got %q", result.Slug)
	}
	if result.Name != "Feature 1" {
		t.Errorf("expected name 'Feature 1', got %q", result.Name)
	}
	if result.Status != "prd" {
		t.Errorf("expected status 'prd', got %q", result.Status)
	}
}

func TestFeatureService_GetByID_NotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewFeatureService(d)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ===========================================================================
// ProposalService tests
// ===========================================================================

func TestProposalService_GetByID_Found(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProposalService(d)
	ctx := context.Background()

	proj, err := db.GetOrCreateProject(ctx, d, "proj")
	if err != nil {
		t.Fatal(err)
	}

	prop, err := db.UpsertProposal(ctx, d, proj.ID, model.ProposalInput{
		Slug: "my-proposal", Title: "My Proposal", Content: "# Hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.GetByID(ctx, prop.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if result.ID != prop.ID {
		t.Errorf("expected ID %d, got %d", prop.ID, result.ID)
	}
	if result.Slug != "my-proposal" {
		t.Errorf("expected slug 'my-proposal', got %q", result.Slug)
	}
	if result.Title != "My Proposal" {
		t.Errorf("expected title 'My Proposal', got %q", result.Title)
	}
}

func TestProposalService_GetByID_NotFound(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewProposalService(d)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TaskService — taskToDetail edge cases
// ---------------------------------------------------------------------------

func TestTaskService_Get_InvalidJSONFields(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewTaskService(d)
	ctx := context.Background()

	_, featID := seedTaskServiceTest(t, d)

	// Insert a task with raw SQL to set invalid JSON for tags/dependencies
	result, err := d.ExecContext(ctx,
		`INSERT INTO tasks (feature_id, task_id, title, description, status, priority, tags, dependencies, created_at, updated_at)
		 VALUES (?, '1.1', 'Bad JSON Task', '', 'pending', 'P0', 'not-valid-json', 'also-bad', datetime('now'), datetime('now'))`,
		featID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()

	// Get should handle invalid JSON gracefully by returning empty slices
	detail, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if len(detail.Tags) != 0 {
		t.Errorf("expected empty tags for invalid JSON, got %v", detail.Tags)
	}
	if len(detail.Dependencies) != 0 {
		t.Errorf("expected empty dependencies for invalid JSON, got %v", detail.Dependencies)
	}
}

// ---------------------------------------------------------------------------
// extractFeatureSlug edge cases
// ---------------------------------------------------------------------------

func TestUploadService_PushDocs_NoFeatureSlug(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	// File with no docs/features/ prefix — feature slug will be empty string
	indexJSON := `{"tasks":{"1.1":{"id":"1.1","title":"No Slug Task","priority":"P0"}}}`

	files := []UploadFile{
		{Path: "other/path/index.json", Filename: "index.json", Content: []byte(indexJSON)},
	}

	summaries, err := svc.PushDocs(ctx, "noslug-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	// Feature should be auto-created with empty string slug
	proj, _ := db.GetOrCreateProject(ctx, d, "noslug-project")
	features, _ := db.ListFeaturesByProject(ctx, d, proj.ID)
	if len(features) != 1 {
		t.Fatalf("expected 1 feature auto-created, got %d", len(features))
	}
	if features[0].Slug != "" {
		t.Errorf("expected empty slug, got %q", features[0].Slug)
	}
}
