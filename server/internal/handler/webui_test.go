package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock service implementations
// ---------------------------------------------------------------------------

// mockProjectService implements service.ProjectService for testing.
type mockProjectService struct {
	items []model.ProjectSummary
	total int
	err   error

	detail    *model.ProjectDetail
	detailErr error
}

func (m *mockProjectService) List(_ context.Context, _ string, _, _ int) ([]model.ProjectSummary, int, error) {
	return m.items, m.total, m.err
}

func (m *mockProjectService) Get(_ context.Context, _ int64) (*model.ProjectDetail, error) {
	return m.detail, m.detailErr
}

func (m *mockProjectService) Upsert(_ context.Context, _ string) (*model.Project, error) {
	return nil, nil
}

// mockFeatureService implements service.FeatureService for testing.
type mockFeatureService struct {
	features   []model.FeatureSummary
	featureErr error

	tasks    []model.Task
	tasksErr error

	featureByID    *model.Feature
	featureByIDErr error
}

func (m *mockFeatureService) ListByProject(_ context.Context, _ int64) ([]model.FeatureSummary, error) {
	return m.features, m.featureErr
}

func (m *mockFeatureService) GetTasks(_ context.Context, _ int64, _ model.TaskFilter) ([]model.Task, error) {
	return m.tasks, m.tasksErr
}

func (m *mockFeatureService) GetByID(_ context.Context, _ int64) (*model.Feature, error) {
	return m.featureByID, m.featureByIDErr
}

// mockTaskService implements service.TaskService for testing.
type mockTaskService struct {
	detail    *model.TaskDetail
	detailErr error

	records    []model.ExecutionRecord
	total      int
	recordsErr error
}

func (m *mockTaskService) Get(_ context.Context, _ int64) (*model.TaskDetail, error) {
	return m.detail, m.detailErr
}

func (m *mockTaskService) GetByTaskID(_ context.Context, _, _, _ string) (*model.TaskDetail, error) {
	return nil, nil
}

func (m *mockTaskService) ListRecords(_ context.Context, _ int64, _, _ int) ([]model.ExecutionRecord, int, error) {
	return m.records, m.total, m.recordsErr
}

func (m *mockTaskService) Claim(_ context.Context, _, _, _ string) (*model.Task, error) {
	return nil, nil
}

func (m *mockTaskService) UpdateStatus(_ context.Context, _ int64, _, _ string) error {
	return nil
}

func (m *mockTaskService) SubmitRecord(_ context.Context, _ int64, _ string, _ model.ExecutionRecord) (*model.ExecutionRecord, error) {
	return nil, nil
}

// mockProposalService implements ProposalContentService for testing.
type mockProposalService struct {
	proposal *model.Proposal
	err      error
}

func (m *mockProposalService) GetByID(_ context.Context, _ int64) (*model.Proposal, error) {
	return m.proposal, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fixedTime is a known time for deterministic test assertions.
var fixedTime = time.Date(2026, 4, 12, 14, 30, 0, 0, time.UTC)

// setupRouter creates a chi mux with the WebUIHandler routes registered.
func setupRouter(h *WebUIHandler) *chi.Mux {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

// errorResponse is used to parse error JSON responses in tests.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

// ---------------------------------------------------------------------------
// Tests: ListProjects
// ---------------------------------------------------------------------------

func TestListProjects_Success(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{
			{ID: 1, Name: "proj-a", FeatureCount: 3, TaskTotal: 24, CompletionRate: 62.5, UpdatedAt: fixedTime},
		},
		total: 1,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects?page=1&pageSize=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	assert.Equal(t, float64(1), body["total"])
	assert.Equal(t, float64(1), body["page"])
	assert.Equal(t, float64(20), body["pageSize"])

	items := body["items"].([]any)
	require.Len(t, items, 1)

	item := items[0].(map[string]any)
	assert.Equal(t, "proj-a", item["name"])
	assert.Equal(t, float64(62.5), item["completionRate"])
}

func TestListProjects_SearchQuery(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{
			{ID: 1, Name: "my-project", FeatureCount: 1, TaskTotal: 5, CompletionRate: 0, UpdatedAt: fixedTime},
		},
		total: 1,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects?search=my-project&page=1&pageSize=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListProjects_DefaultPagination(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{},
		total: 0,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, float64(1), body["page"])
	assert.Equal(t, float64(20), body["pageSize"])
}

func TestListProjects_InternalError(t *testing.T) {
	ps := &mockProjectService{
		err: errors.New("db connection lost"),
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: GetProject
// ---------------------------------------------------------------------------

func TestGetProject_Success(t *testing.T) {
	ps := &mockProjectService{
		detail: &model.ProjectDetail{
			ID:   1,
			Name: "agent-task-center",
			Proposals: []model.ProposalSummary{
				{ID: 1, Slug: "agent-task-center", Title: "Proposal", CreatedAt: fixedTime, FeatureCount: 1},
			},
			Features: []model.FeatureSummary{
				{ID: 1, Slug: "agent-task-center", Name: "ATC", Status: "in-progress", CompletionRate: 62.5, UpdatedAt: fixedTime},
			},
		},
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "agent-task-center", body["name"])

	proposals := body["proposals"].([]any)
	assert.Len(t, proposals, 1)

	features := body["features"].([]any)
	assert.Len(t, features, 1)
}

func TestGetProject_NotFound(t *testing.T) {
	ps := &mockProjectService{
		detailErr: service.ErrNotFound,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetProject_InvalidID(t *testing.T) {
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_id", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: GetFeatureTasks
// ---------------------------------------------------------------------------

func TestGetFeatureTasks_Success(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{ID: 1, Name: "Agent Task Center", Slug: "agent-task-center"},
		tasks: []model.Task{
			{ID: 101, TaskID: "1.1", Title: "Init project", Status: "completed", Priority: "P0", Tags: `["core"]`, Dependencies: `[]`, ClaimedBy: "agent-01"},
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	assert.Equal(t, float64(1), body["featureId"])
	assert.Equal(t, "Agent Task Center", body["featureName"])

	tasks := body["tasks"].([]any)
	require.Len(t, tasks, 1)

	task := tasks[0].(map[string]any)
	assert.Equal(t, "1.1", task["taskId"])
	assert.Equal(t, "completed", task["status"])
	assert.Equal(t, "P0", task["priority"])
}

func TestGetFeatureTasks_WithFilters(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{ID: 1, Name: "ATC", Slug: "atc"},
		tasks: []model.Task{
			{ID: 101, TaskID: "1.1", Title: "Init", Status: "pending", Priority: "P0", Tags: `["core"]`, Dependencies: `[]`},
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/tasks?priority=P0,P1&tag=core&status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetFeatureTasks_FeatureNotFound(t *testing.T) {
	fs := &mockFeatureService{
		featureByIDErr: service.ErrNotFound,
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/999/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetFeatureTasks_InvalidID(t *testing.T) {
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/abc/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_id", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: GetTask
// ---------------------------------------------------------------------------

func TestGetTask_Success(t *testing.T) {
	ts := &mockTaskService{
		detail: &model.TaskDetail{
			ID: 101, TaskID: "1.1", Title: "Init project", Description: "## Task desc",
			Status: "completed", Priority: "P0", Tags: []string{"core", "setup"},
			ClaimedBy: "agent-01", Dependencies: []string{},
			CreatedAt: fixedTime, UpdatedAt: fixedTime,
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "1.1", body["taskId"])
	assert.Equal(t, "Init project", body["title"])
	assert.Equal(t, "## Task desc", body["description"])

	tags := body["tags"].([]any)
	assert.Contains(t, tags, "core")
	assert.Contains(t, tags, "setup")
}

func TestGetTask_NotFound(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrNotFound,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetTask_InvalidID(t *testing.T) {
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/not-a-number", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_id", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: ListTaskRecords
// ---------------------------------------------------------------------------

func TestListTaskRecords_Success(t *testing.T) {
	ts := &mockTaskService{
		records: []model.ExecutionRecord{
			{
				ID: 1, TaskID: 101, AgentID: "agent-01", Summary: "Implemented feature",
				FilesCreated: `["main.go"]`, FilesModified: `[]`, KeyDecisions: `["used chi"]`,
				TestsPassed: 12, TestsFailed: 0, Coverage: 85.6,
				AcceptanceCriteria: `[{"criterion":"compiles","met":true}]`,
				CreatedAt:          fixedTime,
			},
		},
		total: 1,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/101/records?page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	assert.Equal(t, float64(1), body["total"])
	assert.Equal(t, float64(1), body["page"])
	assert.Equal(t, float64(10), body["pageSize"])

	items := body["items"].([]any)
	require.Len(t, items, 1)

	item := items[0].(map[string]any)
	assert.Equal(t, "agent-01", item["agentId"])
	assert.Equal(t, "Implemented feature", item["summary"])
}

func TestListTaskRecords_DefaultPagination(t *testing.T) {
	ts := &mockTaskService{
		records: []model.ExecutionRecord{},
		total:   0,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/records", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, float64(1), body["page"])
	assert.Equal(t, float64(10), body["pageSize"])
}

func TestListTaskRecords_InvalidTaskID(t *testing.T) {
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/abc/records", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_id", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: GetProposalContent
// ---------------------------------------------------------------------------

func TestGetProposalContent_Success(t *testing.T) {
	ps := &mockProposalService{
		proposal: &model.Proposal{
			ID:      1,
			Slug:    "agent-task-center",
			Title:   "Proposal: Agent Task Center",
			Content: "# Proposal: Agent Task Center\n\nSome content here",
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, ps)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/proposals/1/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "Proposal: Agent Task Center", body["title"])
	assert.Contains(t, body["content"], "# Proposal")
}

func TestGetProposalContent_NotFound(t *testing.T) {
	ps := &mockProposalService{
		err: service.ErrNotFound,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, ps)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/proposals/999/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetProposalContent_InvalidID(t *testing.T) {
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/proposals/abc/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_id", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: GetFeatureContent
// ---------------------------------------------------------------------------

func TestGetFeatureContent_Success(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{
			ID:      1,
			Slug:    "agent-task-center",
			Name:    "Agent Task Center",
			Content: "# Manifest\n\nFeature content here",
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "Agent Task Center", body["title"])
	assert.Contains(t, body["content"], "Feature content here")
}

func TestGetFeatureContent_NotFound(t *testing.T) {
	fs := &mockFeatureService{
		featureByIDErr: service.ErrNotFound,
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/999/content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: Content-Type header
// ---------------------------------------------------------------------------

func TestRespondJSON_SetsContentType(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{},
		total: 0,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// Tests: writeError covers all sentinel errors
// ---------------------------------------------------------------------------

func TestWriteError_VersionConflict(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrVersionConflict,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "version_conflict", errResp.Error)
}

func TestWriteError_UnauthorizedAgent(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrUnauthorizedAgent,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "unauthorized_agent", errResp.Error)
}

func TestWriteError_InvalidStatus(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrInvalidStatus,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_status", errResp.Error)
}

func TestWriteError_NoAvailableTask(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrNoAvailableTask,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "no_available_task", errResp.Error)
}

func TestWriteError_InvalidFile(t *testing.T) {
	ts := &mockTaskService{
		detailErr: service.ErrInvalidFile,
	}
	handler := NewWebUIHandler(&mockProjectService{}, &mockFeatureService{}, ts, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: TaskSummary format in GetFeatureTasks
// ---------------------------------------------------------------------------

func TestGetFeatureTasks_TaskSummaryFormat(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{ID: 1, Name: "ATC", Slug: "atc"},
		tasks: []model.Task{
			{
				ID: 101, TaskID: "1.1", Title: "Init", Status: "pending",
				Priority: "P0", Tags: `["core","setup"]`, Dependencies: `["1.0"]`,
				ClaimedBy: "agent-01",
			},
		},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	tasks := body["tasks"].([]any)
	require.Len(t, tasks, 1)

	task := tasks[0].(map[string]any)
	assert.Equal(t, float64(101), task["id"])
	assert.Equal(t, "1.1", task["taskId"])
	assert.Equal(t, "Init", task["title"])
	assert.Equal(t, "pending", task["status"])
	assert.Equal(t, "P0", task["priority"])

	tags := task["tags"].([]any)
	assert.Contains(t, tags, "core")
	assert.Contains(t, tags, "setup")

	deps := task["dependencies"].([]any)
	assert.Contains(t, deps, "1.0")

	assert.Equal(t, "agent-01", task["claimedBy"])
}

// ---------------------------------------------------------------------------
// Tests: Pagination parameters parsing
// ---------------------------------------------------------------------------

func TestListProjects_CustomPagination(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{},
		total: 50,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects?page=3&pageSize=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, float64(3), body["page"])
	assert.Equal(t, float64(5), body["pageSize"])
	assert.Equal(t, float64(50), body["total"])
}

// ---------------------------------------------------------------------------
// Tests: Empty results
// ---------------------------------------------------------------------------

func TestListProjects_EmptyResult(t *testing.T) {
	ps := &mockProjectService{
		items: []model.ProjectSummary{},
		total: 0,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	items := body["items"].([]any)
	assert.Len(t, items, 0)
	assert.Equal(t, float64(0), body["total"])
}

func TestGetFeatureTasks_EmptyTasks(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{ID: 1, Name: "ATC", Slug: "atc"},
		tasks:       []model.Task{},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	tasks := body["tasks"].([]any)
	assert.Len(t, tasks, 0)
}

// ---------------------------------------------------------------------------
// Tests: Edge case - filter parsing with single values
// ---------------------------------------------------------------------------

func TestGetFeatureTasks_SingleFilterValue(t *testing.T) {
	fs := &mockFeatureService{
		featureByID: &model.Feature{ID: 1, Name: "ATC", Slug: "atc"},
		tasks:       []model.Task{},
	}
	handler := NewWebUIHandler(&mockProjectService{}, fs, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/features/1/tasks?status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkListProjects(b *testing.B) {
	ps := &mockProjectService{
		items: makeProjects(20),
		total: 20,
	}
	handler := NewWebUIHandler(ps, &mockFeatureService{}, &mockTaskService{}, &mockProposalService{})
	router := setupRouter(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/projects?page=1&pageSize=20", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func makeProjects(n int) []model.ProjectSummary {
	items := make([]model.ProjectSummary, n)
	for i := range items {
		items[i] = model.ProjectSummary{
			ID:             int64(i + 1),
			Name:           fmt.Sprintf("project-%d", i+1),
			FeatureCount:   3,
			TaskTotal:      24,
			CompletionRate: 50.0,
			UpdatedAt:      fixedTime,
		}
	}
	return items
}
