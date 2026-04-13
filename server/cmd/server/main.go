// Package main is the entry point for the Agent Task Center server.
//
// Startup flow:
//  1. config.Load() reads configuration from environment variables.
//  2. db.Connect(cfg) establishes a database connection.
//  3. db.RunMigrations(db, driver) runs pending schema migrations.
//  4. Services are initialized (ProjectService, FeatureService, TaskService, UploadService, ProposalService).
//  5. Handlers are initialized and routes are registered on a chi mux.
//  6. HTTP server starts listening.
//
// Static file serving:
//   - If STATIC_DIR is set (dev mode), files are served from the filesystem.
//   - Otherwise (prod mode), the embedded web/dist directory is used via embed.FS.
//   - SPA fallback: any non-/api/ GET request that doesn't match a static file returns index.html.
package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"agent-task-center/server/internal/config"
	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/handler"
	"agent-task-center/server/internal/service"
	"agent-task-center/server/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer func() { _ = database.Close() }()

	if err := db.RunMigrations(database, cfg.DBDriver); err != nil {
		// Close DB explicitly since log.Fatalf bypasses defer.
		//nolint:gocritic // acceptable: we must exit on migration failure
		log.Fatalf("migrations: %v", err)
	}

	// Initialize services
	projects := service.NewProjectService(database)
	features := service.NewFeatureService(database)
	tasks := service.NewTaskService(database)
	uploads := service.NewUploadService(database)
	proposals := service.NewProposalService(database)

	// Initialize handlers
	webUI := handler.NewWebUIHandler(projects, features, tasks, proposals)
	agent := handler.NewAgentHandler(tasks)
	upload := handler.NewUploadHandler(uploads)

	// Build router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// Register handler routes
	webUI.RegisterRoutes(r)
	agent.RegisterRoutes(r)
	upload.RegisterRoutes(r)

	// Static file serving with SPA fallback
	staticFS := newStaticFS(cfg.StaticDir)
	r.NotFound(spaHandler(staticFS))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CORS middleware
// ---------------------------------------------------------------------------

// corsMiddleware adds permissive CORS headers for development.
// In production, the allowed origins should be restricted.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Static file serving
// ---------------------------------------------------------------------------

// newStaticFS returns an fs.FS for serving web UI static files.
// If staticDir is non-empty and points to a valid directory, it is used (dev mode).
// Otherwise the embedded DistFS from the web package is used (prod mode).
func newStaticFS(staticDir string) fs.FS {
	if staticDir != "" {
		dirFS := os.DirFS(staticDir)
		// Verify that index.html exists in the provided directory
		if f, err := dirFS.Open("index.html"); err == nil {
			_ = f.Close()
			return dirFS
		}
	}

	// Fall back to embedded FS (strip the "dist" prefix)
	sub, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}
	return sub
}

// spaHandler returns an http.HandlerFunc that:
//  1. Serves static files from the given filesystem.
//  2. For any path that doesn't match a file, returns index.html (SPA fallback).
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))

	return func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		path := r.URL.Path

		// Clean the path and try to serve the file
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// Check if the file exists in the FS and is not a directory
		if f, err := fsys.Open(cleanPath); err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !stat.IsDir() {
				// Use a clean request to avoid http.FileServer redirect issues
				fileServer.ServeHTTP(w, r.Clone(r.Context()))
				return
			}
		}

		// SPA fallback: serve index.html directly from the FS
		// (not via http.FileServer, to avoid redirect issues)
		idx, err := fsys.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = idx.Close() }()

		stat, err := idx.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", stat.ModTime(), idx.(readSeeker))
	}
}

// readSeeker is a type assertion target for fs.File to implement io.ReadSeeker.
type readSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

// init ensures the data directory exists for SQLite before anything runs.
func init() {
	cfg := config.Load()
	if cfg.DBDriver == "sqlite" && cfg.DBPath != "" {
		dir := filepath.Dir(cfg.DBPath)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
	}
}
