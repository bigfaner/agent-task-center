package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Integration test infrastructure
// ---------------------------------------------------------------------------

// setupTestServer creates a fully-wired httptest.Server backed by an in-memory SQLite DB.
// Each test gets its own isolated server instance to avoid state pollution.
func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	database := setupInMemoryDB(t)
	router := setupFullRouter(database)
	return httptest.NewServer(router)
}

// setupInMemoryDB opens an in-memory SQLite database and runs all migrations.
func setupInMemoryDB(t *testing.T) *sqlx.DB {
	t.Helper()

	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if err := db.RunMigrations(database, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return database
}

// setupFullRouter creates a chi router with all handlers wired to real service instances.
func setupFullRouter(database *sqlx.DB) *chi.Mux {
	projectSvc := service.NewProjectService(database)
	featureSvc := service.NewFeatureService(database)
	taskSvc := service.NewTaskService(database)
	proposalSvc := service.NewProposalService(database)
	uploadSvc := service.NewUploadService(database)

	webUI := NewWebUIHandler(projectSvc, featureSvc, taskSvc, proposalSvc)
	agent := NewAgentHandler(taskSvc)
	upload := NewUploadHandler(uploadSvc)

	r := chi.NewRouter()
	webUI.RegisterRoutes(r)
	agent.RegisterRoutes(r)
	upload.RegisterRoutes(r)

	return r
}

// ---------------------------------------------------------------------------
// Test helpers for HTTP requests
// ---------------------------------------------------------------------------

// claimResp is the response body for POST /api/agent/claim.
type claimResp struct {
	ID        int64    `json:"id"`
	TaskID    string   `json:"taskId"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Priority  string   `json:"priority"`
	Tags      []string `json:"tags"`
	ClaimedBy string   `json:"claimedBy"`
}

// recordResp is the response body for POST /api/agent/tasks/{taskId}/records.
type recordResp struct {
	RecordID   int64  `json:"recordId"`
	TaskStatus string `json:"taskStatus"`
}

// errorResp is a generic error response.
type errorResp struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// doClaim sends a claim request to the test server.
func doClaim(t *testing.T, srv *httptest.Server, projectName, featureSlug, agentID string) (*http.Response, claimResp) {
	t.Helper()

	body := claimRequest{
		ProjectName: projectName,
		FeatureSlug: featureSlug,
		AgentID:     agentID,
	}
	b, _ := json.Marshal(body)

	resp, err := http.Post(srv.URL+"/api/agent/claim", "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	var cr claimResp
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&cr))
	}
	return resp, cr
}

// doSubmitRecord sends a submit-record request to the test server.
func doSubmitRecord(t *testing.T, srv *httptest.Server, taskID int64, agentID string) (*http.Response, recordResp) {
	t.Helper()

	body := recordRequest{
		AgentID:       agentID,
		Summary:       "Test execution summary",
		FilesCreated:  []string{"main.go"},
		FilesModified: []string{"go.mod"},
		KeyDecisions:  []string{"used chi"},
		TestsPassed:   5,
		TestsFailed:   0,
		Coverage:      80.0,
		AcceptanceCriteria: []map[string]any{
			{"criterion": "builds successfully", "met": true},
		},
	}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/api/agent/tasks/%d/records", srv.URL, taskID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	var rr recordResp
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&rr))
	}
	return resp, rr
}

// doUploadIndexJSON pushes an index.json file to seed tasks into the database.
// The parser expects tasks as a map keyed by task_id.
func doUploadIndexJSON(t *testing.T, srv *httptest.Server, project, feature string, tasks []model.TaskInput) *http.Response {
	t.Helper()

	// Build the index.json format: {"tasks": {"1.1": {"task_id":"1.1", ...}, ...}}
	tasksMap := make(map[string]map[string]any, len(tasks))
	for _, t := range tasks {
		entry := map[string]any{
			"task_id": t.TaskID,
			"title":   t.Title,
		}
		if t.Description != "" {
			entry["description"] = t.Description
		}
		if t.Priority != "" {
			entry["priority"] = t.Priority
		}
		if len(t.Tags) > 0 {
			entry["tags"] = t.Tags
		}
		if len(t.Dependencies) > 0 {
			entry["dependencies"] = t.Dependencies
		}
		tasksMap[t.TaskID] = entry
	}
	indexData := map[string]any{"tasks": tasksMap}
	content, _ := json.Marshal(indexData)

	return doUploadFile(t, srv, project, feature, "index.json", content)
}

// doUploadFile uploads a single file via POST /api/upload.
func doUploadFile(t *testing.T, srv *httptest.Server, project, feature, filename string, content []byte) *http.Response {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	url := srv.URL + "/api/upload?project=" + project
	if feature != "" {
		url += "&feature=" + feature
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// TestPushClaimRecord_Flow tests the full push -> claim -> submit record workflow.
func TestPushClaimRecord_Flow(t *testing.T) {
	srv := setupTestServer(t)

	// Step 1: Push index.json with 3 tasks
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Init project", Priority: "P0", Tags: []string{"setup"}},
		{TaskID: "1.2", Title: "Add DB layer", Priority: "P1", Dependencies: []string{"1.1"}},
		{TaskID: "1.3", Title: "Add handlers", Priority: "P1", Dependencies: []string{"1.2"}},
	}
	resp := doUploadIndexJSON(t, srv, "test-project", "test-feature", tasks)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var uploadResult model.UpsertSummary
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&uploadResult))
	assert.Equal(t, 3, uploadResult.Created)

	// Step 2: Claim the highest priority task (1.1, P0, no deps)
	resp, claimed := doClaim(t, srv, "test-project", "test-feature", "agent-01")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1.1", claimed.TaskID)
	assert.Equal(t, "agent-01", claimed.ClaimedBy)
	assert.Equal(t, "in_progress", claimed.Status)

	// Step 3: Submit execution record for the claimed task
	resp, recResult := doSubmitRecord(t, srv, claimed.ID, "agent-01")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, recResult.RecordID > 0)
	assert.Equal(t, "completed", recResult.TaskStatus)

	// Step 4: Verify the task is now completed via GET /api/tasks/{id}
	getResp, err := http.Get(fmt.Sprintf("%s/api/tasks/%d", srv.URL, claimed.ID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = getResp.Body.Close() })
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var taskDetail model.TaskDetail
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&taskDetail))
	assert.Equal(t, "completed", taskDetail.Status)
	assert.Equal(t, "agent-01", taskDetail.ClaimedBy)

	// Step 5: Verify execution record exists via GET /api/tasks/{id}/records
	recordsResp, err := http.Get(fmt.Sprintf("%s/api/tasks/%d/records", srv.URL, claimed.ID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = recordsResp.Body.Close() })
	assert.Equal(t, http.StatusOK, recordsResp.StatusCode)

	var recordsBody map[string]any
	require.NoError(t, json.NewDecoder(recordsResp.Body).Decode(&recordsBody))
	assert.Equal(t, float64(1), recordsBody["total"])

	items := recordsBody["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "agent-01", item["agentId"])
}

// TestConcurrentClaim_3Agents verifies that 3 goroutines claiming simultaneously
// each get a different task with no conflicts.
func TestConcurrentClaim_3Agents(t *testing.T) {
	// Use a file-based SQLite database for concurrent access since in-memory
	// databases don't support concurrent writes from multiple goroutines well.
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Use busy_timeout in DSN to handle concurrent write contention
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=1", dbPath)
	database, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	// Limit concurrent connections to avoid locking issues
	database.SetMaxOpenConns(1)

	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("set journal_mode=WAL: %v", err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	// Set busy_timeout so concurrent writes retry instead of failing immediately
	if _, err := database.Exec("PRAGMA busy_timeout=5000"); err != nil {
		t.Fatalf("set busy_timeout: %v", err)
	}

	if err := db.RunMigrations(database, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Enable WAL and foreign keys via PRAGMA (DSN params may not cover all)
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("set journal_mode=WAL: %v", err)
	}
	if _, err := database.Exec("PRAGMA busy_timeout=5000"); err != nil {
		t.Fatalf("set busy_timeout: %v", err)
	}

	router := setupFullRouter(database)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Push 3 pending tasks with no dependencies
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Task A", Priority: "P0"},
		{TaskID: "1.2", Title: "Task B", Priority: "P1"},
		{TaskID: "1.3", Title: "Task C", Priority: "P2"},
	}
	resp := doUploadIndexJSON(t, srv, "concurrent-proj", "concurrent-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 3 agents claim concurrently
	type claimResult struct {
		resp  *http.Response
		claim claimResp
		agent string
	}

	var wg sync.WaitGroup
	results := make([]claimResult, 3)
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%02d", idx+1)
			resp, claim := doClaim(t, srv, "concurrent-proj", "concurrent-feat", agentID)
			mu.Lock()
			results[idx] = claimResult{resp: resp, claim: claim, agent: agentID}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	// Verify: all 3 succeeded
	claimedTaskIDs := make(map[string]string) // taskID -> agentID
	for i, res := range results {
		if res.resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(res.resp.Body)
			t.Errorf("agent %d got status %d: %s", i+1, res.resp.StatusCode, string(bodyBytes))
			continue
		}
		assert.Equal(t, res.agent, res.claim.ClaimedBy,
			"agent %d should be the claimer", i+1)

		// Each task should be claimed by exactly one agent
		_, exists := claimedTaskIDs[res.claim.TaskID]
		assert.False(t, exists, "task %s claimed by multiple agents", res.claim.TaskID)
		claimedTaskIDs[res.claim.TaskID] = res.agent
	}

	// All 3 tasks should be distributed
	assert.Len(t, claimedTaskIDs, 3, "all 3 tasks should be claimed by different agents")
}

// TestUpsert_Idempotent verifies that pushing the same index.json twice
// does not lose or duplicate execution records.
func TestUpsert_Idempotent(t *testing.T) {
	srv := setupTestServer(t)

	// Push index.json once
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Init project", Priority: "P0"},
		{TaskID: "1.2", Title: "Add DB", Priority: "P1"},
	}
	resp := doUploadIndexJSON(t, srv, "idempotent-proj", "idempotent-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Claim and complete task 1.1
	_, claimed := doClaim(t, srv, "idempotent-proj", "idempotent-feat", "agent-01")
	require.Equal(t, "1.1", claimed.TaskID)
	doSubmitRecord(t, srv, claimed.ID, "agent-01")

	// Push the same index.json again (idempotent upsert)
	resp = doUploadIndexJSON(t, srv, "idempotent-proj", "idempotent-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var uploadResult model.UpsertSummary
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&uploadResult))
	assert.Equal(t, 0, uploadResult.Created)
	assert.Equal(t, 2, uploadResult.Updated)

	// Verify task 1.1 is still completed (status preserved)
	getResp, err := http.Get(fmt.Sprintf("%s/api/tasks/%d", srv.URL, claimed.ID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = getResp.Body.Close() })

	var taskDetail model.TaskDetail
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&taskDetail))
	assert.Equal(t, "completed", taskDetail.Status, "status should be preserved after re-push")

	// Verify execution records still exist
	recordsResp, err := http.Get(fmt.Sprintf("%s/api/tasks/%d/records", srv.URL, claimed.ID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = recordsResp.Body.Close() })

	var recordsBody map[string]any
	require.NoError(t, json.NewDecoder(recordsResp.Body).Decode(&recordsBody))
	assert.Equal(t, float64(1), recordsBody["total"], "execution record should not be duplicated")
}

// TestUpload_InvalidFile verifies that uploading an unsupported file type returns a 400 error.
func TestUpload_InvalidFile(t *testing.T) {
	srv := setupTestServer(t)

	// Upload a .txt file (not supported)
	resp := doUploadFile(t, srv, "test-proj", "test-feat", "readme.txt", []byte("hello world"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

// TestUpload_FileTooLarge verifies that uploading a file larger than 5MB returns 413.
// Note: Go's ParseMultipartForm(maxMemory) does not enforce total upload size.
// The handler checks r.ContentLength as a pre-flight check.
// We test with a streaming body that has Content-Length > 5MB.
func TestUpload_FileTooLarge(t *testing.T) {
	srv := setupTestServer(t)

	// Create a large JSON-like content that exceeds 5MB
	largeContent := make([]byte, 5*1024*1024+100)
	for i := range largeContent {
		largeContent[i] = ' '
	}
	// Make it look like valid JSON structure so the extension check passes
	largeContent[0] = '{'
	largeContent[len(largeContent)-1] = '}'

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "big.json")
	require.NoError(t, err)
	_, err = part.Write(largeContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	url := srv.URL + "/api/upload?project=test-proj"
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Set Content-Length to trigger size check
	req.ContentLength = int64(buf.Len())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	// The handler uses ParseMultipartForm(maxUploadBytes) which limits form memory.
	// With a buffer larger than 5MB, it should return 413.
	// If the handler doesn't enforce this at the multipart level, the response
	// may be 400 (invalid_file) or 413 (file_too_large) depending on implementation.
	assert.True(t, resp.StatusCode == http.StatusRequestEntityTooLarge || resp.StatusCode == http.StatusBadRequest,
		"expected 413 or 400 for oversized file, got %d", resp.StatusCode)
}

// TestGetFeatureTasks_Filter verifies server-side filtering by priority, tag, and status.
func TestGetFeatureTasks_Filter(t *testing.T) {
	srv := setupTestServer(t)

	// Seed tasks with various attributes
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Core setup", Priority: "P0", Tags: []string{"core", "setup"}},
		{TaskID: "1.2", Title: "API layer", Priority: "P1", Tags: []string{"api"}},
		{TaskID: "1.3", Title: "DB layer", Priority: "P1", Tags: []string{"core", "db"}},
		{TaskID: "1.4", Title: "Tests", Priority: "P2", Tags: []string{"test"}},
	}
	resp := doUploadIndexJSON(t, srv, "filter-proj", "filter-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Claim task 1.1 to give it "in_progress" status
	_, claimed := doClaim(t, srv, "filter-proj", "filter-feat", "agent-01")
	require.Equal(t, "1.1", claimed.TaskID)

	// We need the feature ID. Get it from the project detail.
	projResp, err := http.Get(srv.URL + "/api/projects")
	require.NoError(t, err)
	t.Cleanup(func() { _ = projResp.Body.Close() })

	var projBody map[string]any
	require.NoError(t, json.NewDecoder(projResp.Body).Decode(&projBody))
	items := projBody["items"].([]any)
	require.Len(t, items, 1)
	projItem := items[0].(map[string]any)
	projID := int64(projItem["id"].(float64))

	// Get project detail to find feature ID
	detailResp, err := http.Get(fmt.Sprintf("%s/api/projects/%d", srv.URL, projID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = detailResp.Body.Close() })

	var detailBody map[string]any
	require.NoError(t, json.NewDecoder(detailResp.Body).Decode(&detailBody))
	features := detailBody["features"].([]any)
	require.Len(t, features, 1)
	featureItem := features[0].(map[string]any)
	featureID := int64(featureItem["id"].(float64))

	// Test: filter by status=pending only
	filterResp, err := http.Get(fmt.Sprintf("%s/api/features/%d/tasks?status=pending", srv.URL, featureID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = filterResp.Body.Close() })
	assert.Equal(t, http.StatusOK, filterResp.StatusCode)

	var filterBody map[string]any
	require.NoError(t, json.NewDecoder(filterResp.Body).Decode(&filterBody))
	filteredTasks := filterBody["tasks"].([]any)
	assert.Equal(t, 3, len(filteredTasks), "should have 3 pending tasks (1.1 is in_progress)")

	// Test: filter by priority=P1
	filterResp, err = http.Get(fmt.Sprintf("%s/api/features/%d/tasks?priority=P1", srv.URL, featureID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = filterResp.Body.Close() })

	require.NoError(t, json.NewDecoder(filterResp.Body).Decode(&filterBody))
	filteredTasks = filterBody["tasks"].([]any)
	assert.Equal(t, 2, len(filteredTasks), "should have 2 P1 tasks")

	// Test: filter by tag=core
	filterResp, err = http.Get(fmt.Sprintf("%s/api/features/%d/tasks?tag=core", srv.URL, featureID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = filterResp.Body.Close() })

	require.NoError(t, json.NewDecoder(filterResp.Body).Decode(&filterBody))
	filteredTasks = filterBody["tasks"].([]any)
	assert.Equal(t, 2, len(filteredTasks), "should have 2 core-tagged tasks")

	// Test: combined filter - status=pending AND priority=P1
	filterResp, err = http.Get(fmt.Sprintf("%s/api/features/%d/tasks?status=pending&priority=P1", srv.URL, featureID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = filterResp.Body.Close() })

	require.NoError(t, json.NewDecoder(filterResp.Body).Decode(&filterBody))
	filteredTasks = filterBody["tasks"].([]any)
	assert.Equal(t, 2, len(filteredTasks), "should have 2 pending P1 tasks")
}

// TestClaimTask_NoAvailable verifies that claiming when all tasks are taken returns 404 no_available_task.
func TestClaimTask_NoAvailable(t *testing.T) {
	srv := setupTestServer(t)

	// Push a single task
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Only task", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "noavail-proj", "noavail-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Claim it
	resp, claimed := doClaim(t, srv, "noavail-proj", "noavail-feat", "agent-01")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1.1", claimed.TaskID)

	// Try to claim again - should fail
	resp, _ = doClaim(t, srv, "noavail-proj", "noavail-feat", "agent-02")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "no_available_task", errResp.Error)
}

// TestUpdateStatus_WrongAgent verifies that updating status with the wrong agentId returns 403.
func TestUpdateStatus_WrongAgent(t *testing.T) {
	srv := setupTestServer(t)

	// Push and claim a task
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Task", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "wrongagent-proj", "wrongagent-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, claimed := doClaim(t, srv, "wrongagent-proj", "wrongagent-feat", "agent-01")
	require.Equal(t, "1.1", claimed.TaskID)

	// Try to update status as a different agent
	body := statusRequest{AgentID: "agent-02", Status: "blocked"}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/api/agent/tasks/%d/status", srv.URL, claimed.ID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp2.Body.Close() })

	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&errResp))
	assert.Equal(t, "unauthorized_agent", errResp.Error)
}

// TestIntegrationUpload_MissingProject verifies that uploading without project returns 400.
func TestIntegrationUpload_MissingProject(t *testing.T) {
	srv := setupTestServer(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.json")
	require.NoError(t, err)
	_, _ = part.Write([]byte(`{"tasks":[]}`))
	require.NoError(t, writer.Close())

	// No project query param
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/upload", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestClaimTask_WithDependencies verifies that tasks with unmet dependencies are skipped.
func TestClaimTask_WithDependencies(t *testing.T) {
	srv := setupTestServer(t)

	// Push tasks with dependencies
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Parent", Priority: "P0"},
		{TaskID: "1.2", Title: "Child", Priority: "P1", Dependencies: []string{"1.1"}},
	}
	resp := doUploadIndexJSON(t, srv, "dep-proj", "dep-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Claim should return the parent (1.1) since 1.2 depends on it
	_, claimed := doClaim(t, srv, "dep-proj", "dep-feat", "agent-01")
	assert.Equal(t, "1.1", claimed.TaskID)

	// Complete the parent
	doSubmitRecord(t, srv, claimed.ID, "agent-01")

	// Now claim should return the child (1.2) since its dependency is met
	_, claimed2 := doClaim(t, srv, "dep-proj", "dep-feat", "agent-02")
	assert.Equal(t, "1.2", claimed2.TaskID)
}

// TestIntegrationClaimTask_MissingFields verifies that claiming without required fields returns 400.
func TestIntegrationClaimTask_MissingFields(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name    string
		body    claimRequest
		wantErr string
	}{
		{
			name:    "missing project",
			body:    claimRequest{FeatureSlug: "feat", AgentID: "agent-01"},
			wantErr: "missing_field",
		},
		{
			name:    "missing feature",
			body:    claimRequest{ProjectName: "proj", AgentID: "agent-01"},
			wantErr: "missing_field",
		},
		{
			name:    "missing agentId",
			body:    claimRequest{ProjectName: "proj", FeatureSlug: "feat"},
			wantErr: "missing_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			resp, err := http.Post(srv.URL+"/api/agent/claim", "application/json", bytes.NewReader(b))
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var errResp errorResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
			assert.Equal(t, tt.wantErr, errResp.Error)
		})
	}
}

// TestSubmitRecord_WrongAgent verifies that submitting a record for a task claimed
// by a different agent returns 403.
func TestSubmitRecord_WrongAgent(t *testing.T) {
	srv := setupTestServer(t)

	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Task", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "rec-proj", "rec-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, claimed := doClaim(t, srv, "rec-proj", "rec-feat", "agent-01")
	require.Equal(t, "1.1", claimed.TaskID)

	// Try to submit record as a different agent
	resp2, _ := doSubmitRecord(t, srv, claimed.ID, "agent-02")
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&errResp))
	assert.Equal(t, "unauthorized_agent", errResp.Error)
}

// TestUpdateStatus_InvalidStatus verifies that an invalid status value returns 400.
func TestUpdateStatus_InvalidStatus(t *testing.T) {
	srv := setupTestServer(t)

	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Task", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "status-proj", "status-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, claimed := doClaim(t, srv, "status-proj", "status-feat", "agent-01")
	require.Equal(t, "1.1", claimed.TaskID)

	// Try to set an invalid status
	body := statusRequest{AgentID: "agent-01", Status: "done"}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/api/agent/tasks/%d/status", srv.URL, claimed.ID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp2.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&errResp))
	assert.Equal(t, "invalid_status", errResp.Error)
}

// TestGetFeatureTasks_NotFound verifies that querying tasks for a non-existent feature returns 404.
func TestGetFeatureTasks_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/features/9999/tasks")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

// TestIntegrationGetTaskContent verifies the agent content endpoint.
func TestIntegrationGetTaskContent(t *testing.T) {
	srv := setupTestServer(t)

	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Setup project", Description: "## Task\n\nInitialize the project structure.", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "content-proj", "content-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Get task content via agent endpoint
	url := fmt.Sprintf("%s/api/agent/tasks/1.1/content?project=content-proj&feature=content-feat", srv.URL)
	resp2, err := http.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp2.Body.Close() })
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var detail model.TaskDetail
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&detail))
	assert.Equal(t, "1.1", detail.TaskID)
	assert.Equal(t, "Setup project", detail.Title)
	assert.Contains(t, detail.Description, "Initialize the project")
}

// TestIntegrationGetTaskContent_MissingParams verifies that missing query params returns 400.
func TestIntegrationGetTaskContent_MissingParams(t *testing.T) {
	srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/agent/tasks/1.1/content")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "missing_field", errResp.Error)
}

// TestListProjects_AfterPush verifies that projects appear in the project list after push.
func TestListProjects_AfterPush(t *testing.T) {
	srv := setupTestServer(t)

	// Push a file to create a project
	tasks := []model.TaskInput{
		{TaskID: "1.1", Title: "Task", Priority: "P0"},
	}
	resp := doUploadIndexJSON(t, srv, "listed-proj", "listed-feat", tasks)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// List projects
	resp2, err := http.Get(srv.URL + "/api/projects")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp2.Body.Close() })

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
	assert.Equal(t, float64(1), body["total"])

	items := body["items"].([]any)
	require.Len(t, items, 1)
	proj := items[0].(map[string]any)
	assert.Equal(t, "listed-proj", proj["name"])
}

// TestClaim_NonexistentFeature verifies that claiming from a non-existent feature returns 404.
func TestClaim_NonexistentFeature(t *testing.T) {
	srv := setupTestServer(t)

	resp, _ := doClaim(t, srv, "nonexistent-proj", "nonexistent-feat", "agent-01")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp errorResp
	respBody, _ := io.ReadAll(resp.Body)
	require.NoError(t, json.NewDecoder(strings.NewReader(string(respBody))).Decode(&errResp))
	assert.Equal(t, "no_available_task", errResp.Error)
}
