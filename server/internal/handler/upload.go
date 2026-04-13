package handler

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/service"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Service interface consumed by UploadHandler
// ---------------------------------------------------------------------------

// UploadParser is the subset of service.UploadService used by upload handlers.
type UploadParser interface {
	ParseAndUpsert(ctx context.Context, projectName, featureSlug, filename string, content []byte) (*model.UpsertSummary, error)
	PushDocs(ctx context.Context, projectName string, files []service.UploadFile) ([]model.UpsertSummary, error)
}

// ---------------------------------------------------------------------------
// Allowed extensions
// ---------------------------------------------------------------------------

// allowedExts maps lowercase file extensions to true for quick lookup.
var allowedExts = map[string]bool{
	".json": true,
	".md":   true,
}

// maxUploadBytes is the maximum allowed multipart form size (5 MB).
const maxUploadBytes = 5 << 20

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// UploadHandler implements file upload and push endpoints.
type UploadHandler struct {
	uploads UploadParser
}

// NewUploadHandler creates an UploadHandler with the given service dependency.
func NewUploadHandler(uploads UploadParser) *UploadHandler {
	return &UploadHandler{uploads: uploads}
}

// RegisterRoutes registers upload and push routes on the given chi router.
func (h *UploadHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/upload", h.UploadFile)
	r.Post("/api/push", h.PushDocs)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// UploadFile handles POST /api/upload
//
// Query params:
//   - project (required): project name
//   - feature (optional): feature slug, required when uploading index.json
//
// Multipart field: "file" (single file, .json or .md, max 5MB).
func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: project", "Provide the project query parameter"))
		return
	}

	// Parse multipart with size limit
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		respondJSON(w, http.StatusRequestEntityTooLarge, errorBody("file_too_large", "File exceeds 5MB limit", "Split the file or reduce its size"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: file", "Provide a file in the 'file' form field"))
		return
	}
	defer func() { _ = file.Close() }()

	// Validate extension
	if !isValidExtension(header.Filename) {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_file", "Invalid file format", "Only .json and .md files are accepted"))
		return
	}

	// Path traversal protection
	if containsPathTraversal(header.Filename) {
		respondJSON(w, http.StatusBadRequest, errorBody("invalid_file", "Invalid file path", "Filename must not contain '..'"))
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, errorBody("internal_error", "Failed to read file", "Check server logs for details"))
		return
	}

	feature := r.URL.Query().Get("feature")
	filename := filepath.Base(header.Filename)

	summary, err := h.uploads.ParseAndUpsert(r.Context(), project, feature, filename, content)
	if err != nil {
		writeError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// PushDocs handles POST /api/push
//
// Query params:
//   - project (required): project name
//
// Multipart field: "files" (multiple files, filenames encode relative paths).
func (h *UploadHandler) PushDocs(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: project", "Provide the project query parameter"))
		return
	}

	// Parse multipart with size limit
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		respondJSON(w, http.StatusRequestEntityTooLarge, errorBody("file_too_large", "File exceeds 5MB limit", "Split the file or reduce its size"))
		return
	}

	form := r.MultipartForm
	if form == nil || form.File == nil {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: files", "Provide files in the 'files' form field"))
		return
	}

	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		respondJSON(w, http.StatusBadRequest, errorBody("missing_field", "Required field missing: files", "Provide at least one file in the 'files' form field"))
		return
	}

	// Validate all files first
	for _, fh := range fileHeaders {
		if !isValidExtension(fh.Filename) {
			respondJSON(w, http.StatusBadRequest, errorBody("invalid_file", "Invalid file format", "Only .json and .md files are accepted"))
			return
		}
		if containsPathTraversal(fh.Filename) {
			respondJSON(w, http.StatusBadRequest, errorBody("invalid_file", "Invalid file path", "Filename must not contain '..'"))
			return
		}
	}

	// Build UploadFile list
	uploadFiles := make([]service.UploadFile, 0, len(fileHeaders))
	for _, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, errorBody("internal_error", "Failed to open uploaded file", "Check server logs for details"))
			return
		}

		content, readErr := io.ReadAll(f)
		_ = f.Close()
		if readErr != nil {
			respondJSON(w, http.StatusInternalServerError, errorBody("internal_error", "Failed to read uploaded file", "Check server logs for details"))
			return
		}

		uploadFiles = append(uploadFiles, service.UploadFile{
			Path:     fh.Filename,
			Filename: filepath.Base(fh.Filename),
			Content:  content,
		})
	}

	summaries, err := h.uploads.PushDocs(r.Context(), project, uploadFiles)
	if err != nil {
		writeError(w, err)
		return
	}

	// Calculate totals
	var totalCreated, totalUpdated int
	for _, s := range summaries {
		totalCreated += s.Created
		totalUpdated += s.Updated
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"results":      summaries,
		"totalCreated": totalCreated,
		"totalUpdated": totalUpdated,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isValidExtension checks if the file has an allowed extension (.json or .md).
func isValidExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedExts[ext]
}

// containsPathTraversal checks if the filename contains path traversal sequences.
func containsPathTraversal(filename string) bool {
	return strings.Contains(filename, "..")
}
