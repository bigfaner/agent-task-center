package service

import (
	"context"
	"encoding/json"
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
