// Package handler implements HTTP handlers for the Agent Task Center.
// Web UI handlers are mounted under /api/ prefix.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/service"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Service interfaces consumed by WebUIHandler
// ---------------------------------------------------------------------------

// ProjectLister is the subset of service.ProjectService used by web UI handlers.
type ProjectLister interface {
	List(ctx context.Context, search string, page, pageSize int) ([]model.ProjectSummary, int, error)
	Get(ctx context.Context, id int64) (*model.ProjectDetail, error)
}

// FeatureLister is the subset of service.FeatureService used by web UI handlers,
// extended with a GetByID method for fetching feature content.
type FeatureLister interface {
	GetTasks(ctx context.Context, featureID int64, filter model.TaskFilter) ([]model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Feature, error)
}

// TaskGetter is the subset of service.TaskService used by web UI handlers.
type TaskGetter interface {
	Get(ctx context.Context, id int64) (*model.TaskDetail, error)
	ListRecords(ctx context.Context, taskID int64, page, pageSize int) ([]model.ExecutionRecord, int, error)
}

// ProposalGetter fetches a proposal by ID (for content endpoint).
type ProposalGetter interface {
	GetByID(ctx context.Context, id int64) (*model.Proposal, error)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// WebUIHandler implements all Web UI HTTP endpoints mounted under /api/.
type WebUIHandler struct {
	projects  ProjectLister
	features  FeatureLister
	tasks     TaskGetter
	proposals ProposalGetter
}

// NewWebUIHandler creates a WebUIHandler with the given service dependencies.
func NewWebUIHandler(projects ProjectLister, features FeatureLister, tasks TaskGetter, proposals ProposalGetter) *WebUIHandler {
	return &WebUIHandler{
		projects:  projects,
		features:  features,
		tasks:     tasks,
		proposals: proposals,
	}
}

// RegisterRoutes registers all Web UI routes on the given chi router.
func (h *WebUIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/projects", h.ListProjects)
		r.Get("/projects/{id}", h.GetProject)
		r.Get("/features/{id}/tasks", h.GetFeatureTasks)
		r.Get("/features/{id}/content", h.GetFeatureContent)
		r.Get("/tasks/{id}", h.GetTask)
		r.Get("/tasks/{id}/records", h.ListTaskRecords)
		r.Get("/proposals/{id}/content", h.GetProposalContent)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errorBody builds a structured error response matching the API handbook format.
func errorBody(code, message, hint string) map[string]string {
	return map[string]string{
		"error":   code,
		"message": message,
		"hint":    hint,
	}
}

// writeError maps a service-layer error to the appropriate HTTP response.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		respondJSON(w, http.StatusNotFound, errorBody("not_found", "Resource not found", "Check the ID is correct"))
	case errors.Is(err, service.ErrNoAvailableTask):
		respondJSON(w, http.StatusNotFound, errorBody("no_available_task", "No tasks available to claim", "All pending tasks are either claimed or have unmet dependencies"))
	case errors.Is(err, service.ErrVersionConflict):
		respondJSON(w, http.StatusConflict, errorBody("version_conflict", "Task was claimed by another agent", "Retry claim to get the next available task"))
	case errors.Is(err, service.ErrUnauthorizedAgent):
		respondJSON(w, http.StatusForbidden, errorBody("unauthorized_agent", "Task is claimed by a different agent", "Only the agent that claimed this task can update it"))
	case errors.Is(err, service.ErrInvalidFile):
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_file", "Invalid file format", "Only .json and .md files are accepted"))
	case errors.Is(err, service.ErrInvalidStatus):
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_status", "Invalid status value", "Valid values: pending, in_progress, blocked"))
	default:
		respondJSON(w, http.StatusInternalServerError, errorBody("internal_error", "Internal server error", "Check server logs for details"))
	}
}

// parseID extracts and validates an integer ID from the URL path parameter "id".
func parseID(r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// parseCommaParam splits a comma-separated query parameter into a slice.
// Returns nil if the parameter is empty.
func parseCommaParam(r *http.Request, key string) []string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// parseIntParam extracts an integer query parameter with a default value.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val < 1 {
		return defaultVal
	}
	return val
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// ListProjects handles GET /api/projects
func (h *WebUIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "pageSize", 20)

	items, total, err := h.projects.List(r.Context(), search, page, pageSize)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetProject handles GET /api/projects/{id}
func (h *WebUIHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid project ID", "ID must be a positive integer"))
		return
	}

	detail, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, detail)
}

// taskSummary is the JSON structure for a task in the kanban view.
type taskSummary struct {
	ID           int64    `json:"id"`
	TaskID       string   `json:"taskId"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	ClaimedBy    string   `json:"claimedBy"`
	Dependencies []string `json:"dependencies"`
}

// GetFeatureTasks handles GET /api/features/{id}/tasks
func (h *WebUIHandler) GetFeatureTasks(w http.ResponseWriter, r *http.Request) {
	featureID, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid feature ID", "ID must be a positive integer"))
		return
	}

	// Get feature info for the response header
	feature, err := h.features.GetByID(r.Context(), featureID)
	if err != nil {
		writeError(w, err)
		return
	}

	// Build filter from query parameters
	filter := model.TaskFilter{
		Priorities: parseCommaParam(r, "priority"),
		Tags:       parseCommaParam(r, "tag"),
		Statuses:   parseCommaParam(r, "status"),
	}

	tasks, err := h.features.GetTasks(r.Context(), featureID, filter)
	if err != nil {
		writeError(w, err)
		return
	}

	// Convert tasks to summary format with deserialized JSON fields
	summaries := make([]taskSummary, 0, len(tasks))
	for _, t := range tasks {
		summaries = append(summaries, taskToSummary(t))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"featureId":   feature.ID,
		"featureName": feature.Name,
		"tasks":       summaries,
	})
}

// GetTask handles GET /api/tasks/{id}
func (h *WebUIHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid task ID", "ID must be a positive integer"))
		return
	}

	detail, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, detail)
}

// ListTaskRecords handles GET /api/tasks/{id}/records
func (h *WebUIHandler) ListTaskRecords(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid task ID", "ID must be a positive integer"))
		return
	}

	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "pageSize", 10)

	records, total, err := h.tasks.ListRecords(r.Context(), taskID, page, pageSize)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":    records,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetProposalContent handles GET /api/proposals/{id}/content
func (h *WebUIHandler) GetProposalContent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid proposal ID", "ID must be a positive integer"))
		return
	}

	proposal, err := h.proposals.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"title":   proposal.Title,
		"content": proposal.Content,
	})
}

// GetFeatureContent handles GET /api/features/{id}/content
func (h *WebUIHandler) GetFeatureContent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_id", "Invalid feature ID", "ID must be a positive integer"))
		return
	}

	feature, err := h.features.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"title":   feature.Name,
		"content": feature.Content,
	})
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// taskToSummary converts a model.Task to a taskSummary, deserializing JSON fields.
func taskToSummary(t model.Task) taskSummary {
	var tags []string
	if err := json.Unmarshal([]byte(t.Tags), &tags); err != nil {
		tags = []string{}
	}

	var deps []string
	if err := json.Unmarshal([]byte(t.Dependencies), &deps); err != nil {
		deps = []string{}
	}

	return taskSummary{
		ID:           t.ID,
		TaskID:       t.TaskID,
		Title:        t.Title,
		Status:       t.Status,
		Priority:     t.Priority,
		Tags:         tags,
		ClaimedBy:    t.ClaimedBy,
		Dependencies: deps,
	}
}
