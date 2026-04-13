package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"agent-task-center/server/internal/model"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Service interface consumed by AgentHandler
// ---------------------------------------------------------------------------

// TaskClaimer is the subset of service.TaskService used by agent handlers.
type TaskClaimer interface {
	Get(ctx context.Context, id int64) (*model.TaskDetail, error)
	GetByTaskID(ctx context.Context, projectName, featureSlug, taskID string) (*model.TaskDetail, error)
	ListRecords(ctx context.Context, taskID int64, page, pageSize int) ([]model.ExecutionRecord, int, error)
	Claim(ctx context.Context, projectName, featureSlug, agentID string) (*model.Task, error)
	UpdateStatus(ctx context.Context, taskID int64, agentID, status string) error
	SubmitRecord(ctx context.Context, taskID int64, agentID string, record model.ExecutionRecord) (*model.ExecutionRecord, error)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// AgentHandler implements all Agent HTTP endpoints mounted under /api/agent/.
type AgentHandler struct {
	tasks TaskClaimer
}

// NewAgentHandler creates an AgentHandler with the given service dependency.
func NewAgentHandler(tasks TaskClaimer) *AgentHandler {
	return &AgentHandler{tasks: tasks}
}

// RegisterRoutes registers all agent routes on the given chi router.
func (h *AgentHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/agent", func(r chi.Router) {
		r.Post("/claim", h.ClaimTask)
		r.Patch("/tasks/{taskId}/status", h.UpdateTaskStatus)
		r.Post("/tasks/{taskId}/records", h.SubmitRecord)
		r.Get("/tasks/{key}/content", h.GetTaskContent)
	})
}

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

// claimRequest is the JSON body for POST /api/agent/claim.
type claimRequest struct {
	ProjectName string `json:"projectName"`
	FeatureSlug string `json:"featureSlug"`
	AgentID     string `json:"agentId"`
}

// statusRequest is the JSON body for PATCH /api/agent/tasks/{taskId}/status.
type statusRequest struct {
	AgentID string `json:"agentId"`
	Status  string `json:"status"`
}

// recordRequest is the JSON body for POST /api/agent/tasks/{taskId}/records.
type recordRequest struct {
	AgentID            string                   `json:"agentId"`
	Summary            string                   `json:"summary"`
	FilesCreated       []string                 `json:"filesCreated"`
	FilesModified      []string                 `json:"filesModified"`
	KeyDecisions       []string                 `json:"keyDecisions"`
	TestsPassed        int                      `json:"testsPassed"`
	TestsFailed        int                      `json:"testsFailed"`
	Coverage           float64                  `json:"coverage"`
	AcceptanceCriteria []map[string]interface{} `json:"acceptanceCriteria"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// ClaimTask handles POST /api/agent/claim
func (h *AgentHandler) ClaimTask(w http.ResponseWriter, r *http.Request) {
	var req claimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: invalid JSON body", "Provide valid JSON with projectName, featureSlug, agentId"))
		return
	}

	if req.ProjectName == "" || req.FeatureSlug == "" || req.AgentID == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing", "Provide projectName, featureSlug, and agentId"))
		return
	}

	task, err := h.tasks.Claim(r.Context(), req.ProjectName, req.FeatureSlug, req.AgentID)
	if err != nil {
		writeError(w, err)
		return
	}

	// Build response with deserialized tags and dependencies
	resp := taskToSummary(*task)
	respondJSON(w, http.StatusOK, resp)
}

// UpdateTaskStatus handles PATCH /api/agent/tasks/{taskId}/status
func (h *AgentHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid task ID", "ID must be a positive integer"))
		return
	}

	var req statusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: invalid JSON body", "Provide valid JSON with agentId and status"))
		return
	}

	if req.AgentID == "" || req.Status == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing", "Provide agentId and status"))
		return
	}

	err := h.tasks.UpdateStatus(r.Context(), taskID, req.AgentID, req.Status)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// SubmitRecord handles POST /api/agent/tasks/{taskId}/records
func (h *AgentHandler) SubmitRecord(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid task ID", "ID must be a positive integer"))
		return
	}

	var req recordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: invalid JSON body", "Provide valid JSON body"))
		return
	}

	if req.AgentID == "" || req.Summary == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing", "Provide agentId and summary"))
		return
	}

	// Convert request to model.ExecutionRecord
	record := model.ExecutionRecord{
		AgentID:     req.AgentID,
		Summary:     req.Summary,
		TestsPassed: req.TestsPassed,
		TestsFailed: req.TestsFailed,
		Coverage:    req.Coverage,
	}

	// Serialize slice fields to JSON strings
	if req.FilesCreated != nil {
		b, _ := json.Marshal(req.FilesCreated)
		record.FilesCreated = string(b)
	}
	if req.FilesModified != nil {
		b, _ := json.Marshal(req.FilesModified)
		record.FilesModified = string(b)
	}
	if req.KeyDecisions != nil {
		b, _ := json.Marshal(req.KeyDecisions)
		record.KeyDecisions = string(b)
	}
	if req.AcceptanceCriteria != nil {
		b, _ := json.Marshal(req.AcceptanceCriteria)
		record.AcceptanceCriteria = string(b)
	}

	saved, err := h.tasks.SubmitRecord(r.Context(), taskID, req.AgentID, record)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recordId":    saved.ID,
		"taskStatus":  "completed",
	})
}

// GetTaskContent handles GET /api/agent/tasks/{key}/content
func (h *AgentHandler) GetTaskContent(w http.ResponseWriter, r *http.Request) {
	taskKey := chi.URLParam(r, "key")

	project := r.URL.Query().Get("project")
	feature := r.URL.Query().Get("feature")

	if project == "" || feature == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing", "Provide project and feature query parameters"))
		return
	}

	detail, err := h.tasks.GetByTaskID(r.Context(), project, feature, taskKey)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, detail)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTaskID extracts and validates an integer task ID from the URL path parameter "taskId".
func parseTaskID(r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "taskId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
