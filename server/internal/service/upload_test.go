package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/model"

	"github.com/jmoiron/sqlx"
)

// ===========================================================================
// UploadService tests
// ===========================================================================

// seedUploadTest creates a project for upload tests and returns the DB and project name.
func seedUploadTest(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	d := openServiceTestDB(t)
	ctx := context.Background()

	_, err := db.GetOrCreateProject(ctx, d, "test-project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	return d, "test-project"
}

// ---------------------------------------------------------------------------
// ParseAndUpsert — index.json
// ---------------------------------------------------------------------------

func TestUploadService_ParseAndUpsert_IndexJSON_CreatesTasks(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	// Create a feature first (required for task upsert)
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	_, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "my-feature", Name: "My Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	indexJSON := `{
		"tasks": {
			"1.1": {"id": "1.1", "title": "Task One", "priority": "P0"},
			"1.2": {"id": "1.2", "title": "Task Two", "priority": "P1", "tags": ["core"]}
		}
	}`

	summary, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}
	if summary.Filename != "index.json" {
		t.Errorf("expected filename 'index.json', got %q", summary.Filename)
	}
	if summary.Created != 2 {
		t.Errorf("expected 2 created, got %d", summary.Created)
	}
	if summary.Updated != 0 {
		t.Errorf("expected 0 updated, got %d", summary.Updated)
	}
}

func TestUploadService_ParseAndUpsert_IndexJSON_Idempotent(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	// Create project + feature
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	_, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "my-feature", Name: "My Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	indexJSON := `{
		"tasks": {
			"1.1": {"id": "1.1", "title": "Task One", "priority": "P0"},
			"1.2": {"id": "1.2", "title": "Task Two", "priority": "P1"}
		}
	}`

	// First upload: all created
	s1, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if s1.Created != 2 {
		t.Errorf("first: expected 2 created, got %d", s1.Created)
	}

	// Second upload: all updated
	s2, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if s2.Created != 0 {
		t.Errorf("second: expected 0 created, got %d", s2.Created)
	}
	if s2.Updated != 2 {
		t.Errorf("second: expected 2 updated, got %d", s2.Updated)
	}
}

func TestUploadService_ParseAndUpsert_IndexJSON_PreservesStatus(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	feat, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "my-feature", Name: "My Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	indexJSON := `{"tasks":{"1.1":{"id":"1.1","title":"Task One","priority":"P0"}}}`

	// First upload
	_, err = svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatal(err)
	}

	// Claim the task to change its status
	claimed, err := db.ClaimTask(ctx, d, feat.ID, "agent-01")
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Status != "in_progress" {
		t.Fatalf("expected 'in_progress' after claim, got %q", claimed.Status)
	}

	// Second upload with same data — status should NOT be overwritten
	_, err = svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatal(err)
	}

	task, err := db.GetTaskByTaskID(ctx, d, feat.ID, "1.1")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "in_progress" {
		t.Errorf("expected status preserved as 'in_progress', got %q", task.Status)
	}
	if task.ClaimedBy != "agent-01" {
		t.Errorf("expected claimed_by preserved as 'agent-01', got %q", task.ClaimedBy)
	}
}

func TestUploadService_ParseAndUpsert_IndexJSON_CreatesFeature(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	// Do NOT create a feature manually — it should be created automatically
	indexJSON := `{"tasks":{"1.1":{"id":"1.1","title":"Auto Feature Task","priority":"P0"}}}`

	summary, err := svc.ParseAndUpsert(ctx, projectName, "auto-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}
	if summary.Created != 1 {
		t.Errorf("expected 1 created, got %d", summary.Created)
	}

	// Verify feature was created
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	feat, err := db.GetFeatureBySlug(ctx, d, proj.ID, "auto-feature")
	if err != nil {
		t.Fatalf("expected feature to be created, got error: %v", err)
	}
	if feat.Slug != "auto-feature" {
		t.Errorf("expected feature slug 'auto-feature', got %q", feat.Slug)
	}
}

func TestUploadService_ParseAndUpsert_IndexJSON_EmptyTasks(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	indexJSON := `{"tasks":{}}`

	summary, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}
	if summary.Created != 0 {
		t.Errorf("expected 0 created, got %d", summary.Created)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped (empty tasks), got %d", summary.Skipped)
	}
}

// ---------------------------------------------------------------------------
// ParseAndUpsert — manifest.md
// ---------------------------------------------------------------------------

func TestUploadService_ParseAndUpsert_ManifestMD_CreatesFeature(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	manifest := `---
feature: agent-task-center
status: in-progress
---
# Agent Task Center
Some content here.
`
	summary, err := svc.ParseAndUpsert(ctx, projectName, "agent-task-center", "manifest.md", []byte(manifest))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}
	if summary.Filename != "manifest.md" {
		t.Errorf("expected filename 'manifest.md', got %q", summary.Filename)
	}
	if summary.Created != 1 {
		t.Errorf("expected 1 created, got %d", summary.Created)
	}

	// Verify feature in DB
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	feat, err := db.GetFeatureBySlug(ctx, d, proj.ID, "agent-task-center")
	if err != nil {
		t.Fatalf("expected feature to exist, got error: %v", err)
	}
	if feat.Name != "agent-task-center" {
		t.Errorf("expected name 'agent-task-center', got %q", feat.Name)
	}
	if feat.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", feat.Status)
	}
}

func TestUploadService_ParseAndUpsert_ManifestMD_Idempotent(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	manifest := `---
feature: my-feature
status: prd
---
# My Feature
Content here.
`
	// First upload
	s1, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "manifest.md", []byte(manifest))
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if s1.Created != 1 {
		t.Errorf("first: expected 1 created, got %d", s1.Created)
	}

	// Second upload
	s2, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "manifest.md", []byte(manifest))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if s2.Created != 0 {
		t.Errorf("second: expected 0 created, got %d", s2.Created)
	}
	if s2.Updated != 1 {
		t.Errorf("second: expected 1 updated, got %d", s2.Updated)
	}
}

// ---------------------------------------------------------------------------
// ParseAndUpsert — proposal.md
// ---------------------------------------------------------------------------

func TestUploadService_ParseAndUpsert_ProposalMD_CreatesProposal(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	proposal := `# My Proposal Title

This is the proposal content.
It has multiple lines.
`
	summary, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "proposal.md", []byte(proposal))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}
	if summary.Filename != "proposal.md" {
		t.Errorf("expected filename 'proposal.md', got %q", summary.Filename)
	}
	if summary.Created != 1 {
		t.Errorf("expected 1 created, got %d", summary.Created)
	}

	// Verify proposal in DB
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	proposals, err := db.ListProposalsByProject(ctx, d, proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	if proposals[0].Slug != "my-feature" {
		t.Errorf("expected slug 'my-feature', got %q", proposals[0].Slug)
	}
	if proposals[0].Title != "My Proposal Title" {
		t.Errorf("expected title 'My Proposal Title', got %q", proposals[0].Title)
	}
}

func TestUploadService_ParseAndUpsert_ProposalMD_Idempotent(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	proposal := `# Proposal Title
Content.`

	// First upload
	s1, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "proposal.md", []byte(proposal))
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if s1.Created != 1 {
		t.Errorf("first: expected 1 created, got %d", s1.Created)
	}

	// Second upload
	s2, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "proposal.md", []byte(proposal))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if s2.Created != 0 {
		t.Errorf("second: expected 0 created, got %d", s2.Created)
	}
	if s2.Updated != 1 {
		t.Errorf("second: expected 1 updated, got %d", s2.Updated)
	}
}

// ---------------------------------------------------------------------------
// ParseAndUpsert — invalid files
// ---------------------------------------------------------------------------

func TestUploadService_ParseAndUpsert_InvalidFilename(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	_, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "unknown.txt", []byte("data"))
	if !errors.Is(err, ErrInvalidFile) {
		t.Errorf("expected ErrInvalidFile, got %v", err)
	}
}

func TestUploadService_ParseAndUpsert_IndexJSON_InvalidJSON(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	_, err := svc.ParseAndUpsert(ctx, projectName, "my-feature", "index.json", []byte("not json"))
	if !errors.Is(err, ErrInvalidFile) {
		t.Errorf("expected ErrInvalidFile, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// PushDocs
// ---------------------------------------------------------------------------

func TestUploadService_PushDocs(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	manifest := `---
feature: push-feature
status: prd
---
# Push Feature
`
	indexJSON := `{"tasks":{"1.1":{"id":"1.1","title":"Push Task","priority":"P0"}}}`
	proposal := `# Push Proposal`

	files := []UploadFile{
		{Path: "docs/features/push-feature/manifest.md", Filename: "manifest.md", Content: []byte(manifest)},
		{Path: "docs/features/push-feature/tasks/index.json", Filename: "index.json", Content: []byte(indexJSON)},
		{Path: "docs/features/push-feature/proposal.md", Filename: "proposal.md", Content: []byte(proposal)},
	}

	summaries, err := svc.PushDocs(ctx, "push-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}

	// Verify manifest created feature
	proj, _ := db.GetOrCreateProject(ctx, d, "push-project")
	feat, err := db.GetFeatureBySlug(ctx, d, proj.ID, "push-feature")
	if err != nil {
		t.Fatalf("expected feature to exist: %v", err)
	}
	if feat.Status != "prd" {
		t.Errorf("expected status 'prd', got %q", feat.Status)
	}

	// Verify tasks created
	tasks, err := db.ListTasksByFeature(ctx, d, feat.ID, model.TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestUploadService_PushDocs_ExtractsFeatureSlugFromPath(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	indexJSON := `{"tasks":{"2.1":{"id":"2.1","title":"Slug Task","priority":"P1"}}}`

	files := []UploadFile{
		{Path: "docs/features/slug-extraction/tasks/index.json", Filename: "index.json", Content: []byte(indexJSON)},
	}

	summaries, err := svc.PushDocs(ctx, "slug-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	// Verify feature was auto-created with slug from path
	proj, _ := db.GetOrCreateProject(ctx, d, "slug-project")
	feat, err := db.GetFeatureBySlug(ctx, d, proj.ID, "slug-extraction")
	if err != nil {
		t.Fatalf("expected feature 'slug-extraction' to be auto-created: %v", err)
	}

	tasks, _ := db.ListTasksByFeature(ctx, d, feat.ID, model.TaskFilter{})
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestUploadService_PushDocs_ManifestUsesParsedSlug(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	manifest := `---
feature: parsed-slug
status: done
---
# Parsed Slug
`
	files := []UploadFile{
		{Path: "docs/features/parsed-slug/manifest.md", Filename: "manifest.md", Content: []byte(manifest)},
	}

	summaries, err := svc.PushDocs(ctx, "manifest-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if summaries[0].Created != 1 {
		t.Errorf("expected 1 created, got %d", summaries[0].Created)
	}

	// Verify feature uses the slug from manifest frontmatter
	proj, _ := db.GetOrCreateProject(ctx, d, "manifest-project")
	feat, err := db.GetFeatureBySlug(ctx, d, proj.ID, "parsed-slug")
	if err != nil {
		t.Fatalf("expected feature with slug from manifest: %v", err)
	}
	if feat.Status != "done" {
		t.Errorf("expected status 'done', got %q", feat.Status)
	}
}

func TestUploadService_PushDocs_InvalidFileSkipped(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	files := []UploadFile{
		{Path: "docs/features/test/readme.txt", Filename: "readme.txt", Content: []byte("hello")},
	}

	summaries, err := svc.PushDocs(ctx, "skip-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Filename != "readme.txt" {
		t.Errorf("expected filename 'readme.txt', got %q", summaries[0].Filename)
	}
	if summaries[0].Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summaries[0].Skipped)
	}
}

func TestUploadService_PushDocs_MixedFiles(t *testing.T) {
	d := openServiceTestDB(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	manifest := `---
feature: mixed-test
status: prd
---
# Mixed
`
	indexJSON := `{"tasks":{"3.1":{"id":"3.1","title":"Mixed Task","priority":"P0"}}}`

	files := []UploadFile{
		{Path: "docs/features/mixed-test/manifest.md", Filename: "manifest.md", Content: []byte(manifest)},
		{Path: "docs/features/mixed-test/tasks/index.json", Filename: "index.json", Content: []byte(indexJSON)},
		{Path: "docs/features/mixed-test/notes.txt", Filename: "notes.txt", Content: []byte("notes")},
	}

	summaries, err := svc.PushDocs(ctx, "mixed-project", files)
	if err != nil {
		t.Fatalf("PushDocs() error: %v", err)
	}
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}

	// First: manifest created
	if summaries[0].Filename != "manifest.md" || summaries[0].Created != 1 {
		t.Errorf("manifest: expected created=1, got %+v", summaries[0])
	}
	// Second: index.json created task
	if summaries[1].Filename != "index.json" || summaries[1].Created != 1 {
		t.Errorf("index.json: expected created=1, got %+v", summaries[1])
	}
	// Third: unknown file skipped
	if summaries[2].Filename != "notes.txt" || summaries[2].Skipped != 1 {
		t.Errorf("notes.txt: expected skipped=1, got %+v", summaries[2])
	}
}

// ---------------------------------------------------------------------------
// Summary message
// ---------------------------------------------------------------------------

func TestUploadService_ParseAndUpsert_SummaryMessage(t *testing.T) {
	d, projectName := seedUploadTest(t)
	svc := NewUploadService(d)
	ctx := context.Background()

	// Create feature
	proj, _ := db.GetOrCreateProject(ctx, d, projectName)
	_, err := db.UpsertFeature(ctx, d, proj.ID, model.FeatureInput{
		Slug: "msg-feature", Name: "Msg Feature", Status: "in-progress", Content: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	indexJSON := `{"tasks":{"1.1":{"id":"1.1","title":"T1","priority":"P0"},"1.2":{"id":"1.2","title":"T2","priority":"P1"}}}`

	summary, err := svc.ParseAndUpsert(ctx, projectName, "msg-feature", "index.json", []byte(indexJSON))
	if err != nil {
		t.Fatalf("ParseAndUpsert() error: %v", err)
	}

	expected := fmt.Sprintf("新增 %d 个任务，更新 %d 个任务", summary.Created, summary.Updated)
	if summary.Message != expected {
		t.Errorf("expected message %q, got %q", expected, summary.Message)
	}
}
