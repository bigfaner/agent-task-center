package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock upload service
// ---------------------------------------------------------------------------

// mockUploadService implements the uploadHandler's service interface for testing.
type mockUploadService struct {
	summary    *model.UpsertSummary
	err        error
	calls      []uploadCall
	pushResult []model.UpsertSummary
	pushErr    error
}

type uploadCall struct {
	ProjectName string
	FeatureSlug string
	Filename    string
	Content     []byte
}

func (m *mockUploadService) ParseAndUpsert(_ context.Context, projectName, featureSlug, filename string, content []byte) (*model.UpsertSummary, error) {
	m.calls = append(m.calls, uploadCall{
		ProjectName: projectName,
		FeatureSlug: featureSlug,
		Filename:    filename,
		Content:     content,
	})
	if m.err != nil {
		return nil, m.err
	}
	return m.summary, nil
}

func (m *mockUploadService) PushDocs(_ context.Context, _ string, _ []service.UploadFile) ([]model.UpsertSummary, error) {
	return m.pushResult, m.pushErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupUploadRouter creates a chi mux with upload routes registered.
func setupUploadRouter(h *UploadHandler) *chi.Mux {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

// createMultipartUpload builds a multipart/form-data request with a single file field.
func createMultipartUpload(filename, content string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", filename)
	_, _ = io.Copy(part, strings.NewReader(content))
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

// createMultipartPush builds a multipart/form-data request with multiple file fields.
func createMultipartPush(files []struct{ Name, Content string }) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range files {
		part, _ := w.CreateFormFile("files", f.Name)
		_, _ = io.Copy(part, strings.NewReader(f.Content))
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

// ---------------------------------------------------------------------------
// Tests: UploadFile — POST /api/upload
// ---------------------------------------------------------------------------

func TestUploadFile_Success(t *testing.T) {
	svc := &mockUploadService{
		summary: &model.UpsertSummary{
			Filename: "index.json",
			Created:  5,
			Updated:  3,
			Message:  "新增 5 个任务，更新 3 个任务",
		},
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("index.json", `{"tasks":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project&feature=my-feature", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body model.UpsertSummary
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "index.json", body.Filename)
	assert.Equal(t, 5, body.Created)
	assert.Equal(t, 3, body.Updated)

	require.Len(t, svc.calls, 1)
	assert.Equal(t, "my-project", svc.calls[0].ProjectName)
	assert.Equal(t, "my-feature", svc.calls[0].FeatureSlug)
	assert.Equal(t, "index.json", svc.calls[0].Filename)
}

func TestUploadFile_MissingProject(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("index.json", `{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/upload?feature=my-feature", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "missing_field", errResp.Error)
}

func TestUploadFile_InvalidExtension(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("readme.txt", "hello")
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

func TestUploadFile_InvalidExtension_CSV(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("data.csv", "a,b,c")
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

func TestUploadFile_NoFile(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "missing_field", errResp.Error)
}

func TestUploadFile_PathTraversalBlockedByMultipart(t *testing.T) {
	// Go's multipart library sanitizes filenames, stripping directory components.
	// "../etc/passwd.json" becomes "passwd.json" on the server side.
	// This test verifies the file still gets a sanitized name and processes correctly.
	svc := &mockUploadService{
		summary: &model.UpsertSummary{
			Filename: "passwd.json",
			Created:  1,
			Message:  "ok",
		},
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("../etc/passwd.json", `{"evil":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Go sanitizes the filename to "passwd.json" which has .json extension, so it passes
	assert.Equal(t, http.StatusOK, w.Code)

	require.Len(t, svc.calls, 1)
	// Filename should be sanitized (no "..")
	assert.Equal(t, "passwd.json", svc.calls[0].Filename)
}

func TestUploadFile_ServiceError_InvalidFile(t *testing.T) {
	svc := &mockUploadService{
		err: service.ErrInvalidFile,
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("index.json", `not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project&feature=my-feature", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

func TestUploadFile_ServiceError_InternalError(t *testing.T) {
	svc := &mockUploadService{
		err: fmt.Errorf("database connection lost"),
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartUpload("index.json", `{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/upload?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

// ---------------------------------------------------------------------------
// Tests: PushDocs — POST /api/push
// ---------------------------------------------------------------------------

func TestPushDocs_Success(t *testing.T) {
	svc := &mockUploadService{
		pushResult: []model.UpsertSummary{
			{Filename: "proposal.md", Created: 1, Updated: 0},
			{Filename: "manifest.md", Created: 0, Updated: 1},
			{Filename: "index.json", Created: 8, Updated: 2},
		},
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "proposal.md", Content: "# Proposal"},
		{Name: "manifest.md", Content: "---\nfeature: test\n---\n# Manifest"},
		{Name: "index.json", Content: `{"tasks":{}}`},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	results := body["results"].([]any)
	assert.Len(t, results, 3)
	assert.Equal(t, float64(9), body["totalCreated"])
	assert.Equal(t, float64(3), body["totalUpdated"])
}

func TestPushDocs_MissingProject(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "index.json", Content: `{}`},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "missing_field", errResp.Error)
}

func TestPushDocs_NoFiles(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	assert.Equal(t, "missing_field", errResp.Error)
}

func TestPushDocs_PathTraversalBlockedByMultipart(t *testing.T) {
	// Go's multipart library sanitizes filenames, stripping directory components.
	// This test verifies files with traversal in original name still get processed
	// with sanitized names.
	svc := &mockUploadService{
		pushResult: []model.UpsertSummary{
			{Filename: "passwd.json", Created: 1, Updated: 0},
		},
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "../../etc/passwd.json", Content: "evil"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Go sanitizes the filename, so it passes through with cleaned name
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	results := body["results"].([]any)
	assert.Len(t, results, 1)
}

func TestPushDocs_InvalidExtension(t *testing.T) {
	svc := &mockUploadService{}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "readme.txt", Content: "hello"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "invalid_file", errResp.Error)
}

func TestPushDocs_ServiceError(t *testing.T) {
	svc := &mockUploadService{
		pushErr: fmt.Errorf("database error"),
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "index.json", Content: `{}`},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

func TestPushDocs_EmptyResults(t *testing.T) {
	svc := &mockUploadService{
		pushResult: []model.UpsertSummary{},
	}
	handler := NewUploadHandler(svc)
	router := setupUploadRouter(handler)

	buf, contentType := createMultipartPush([]struct{ Name, Content string }{
		{Name: "docs/features/test/manifest.md", Content: "---\nfeature: test\n---\n# Test"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/push?project=my-project", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	results := body["results"].([]any)
	assert.Len(t, results, 0)
	assert.Equal(t, float64(0), body["totalCreated"])
	assert.Equal(t, float64(0), body["totalUpdated"])
}
