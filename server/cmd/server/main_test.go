package main

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Tests: CORS middleware
// ---------------------------------------------------------------------------

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Get("/api/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Get("/api/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------------------------------------------------------------------
// Tests: SPA handler
// ---------------------------------------------------------------------------

//go:embed testdata/spa
var testSPA embed.FS

func testSPASub(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(testSPA, "testdata/spa")
	require.NoError(t, err)
	return sub
}

func TestSPAHandler_APIRoutesNotIntercepted(t *testing.T) {
	sub := testSPASub(t)

	r := chi.NewRouter()
	r.Handle("/api/*", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	r.NotFound(spaHandler(sub))

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok":true`)
}

func TestSPAHandler_RootReturnsIndexHTML(t *testing.T) {
	sub := testSPASub(t)

	r := chi.NewRouter()
	r.NotFound(spaHandler(sub))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<!doctype html>")
}

func TestSPAHandler_SPARoutesReturnIndexHTML(t *testing.T) {
	sub := testSPASub(t)

	r := chi.NewRouter()
	r.NotFound(spaHandler(sub))

	for _, path := range []string{"/projects/1", "/features/2", "/some/deep/nested/path"} {
		t.Run("path="+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), "<!doctype html>")
		})
	}
}

func TestSPAHandler_StaticAssetsServedDirectly(t *testing.T) {
	sub := testSPASub(t)

	r := chi.NewRouter()
	r.NotFound(spaHandler(sub))

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "console.log")
}

func TestSPAHandler_NonGetReturnsNotFound(t *testing.T) {
	sub := testSPASub(t)

	r := chi.NewRouter()
	r.NotFound(spaHandler(sub))

	req := httptest.NewRequest(http.MethodPost, "/projects/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: newStaticFS helper
// ---------------------------------------------------------------------------

func TestNewStaticFS_EmptyStringUsesEmbed(t *testing.T) {
	// When staticDir is empty, it should fall back to the embedded web.DistFS
	f := newStaticFS("")
	assert.NotNil(t, f)
}

func TestNewStaticFS_NonexistentDirUsesEmbed(t *testing.T) {
	f := newStaticFS("/nonexistent/path/that/does/not/exist")
	assert.NotNil(t, f)
}
