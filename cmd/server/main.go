package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ecosyste-ms/archives/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	// Find project root (where templates and openapi dirs live)
	root := projectRoot()

	templateDir := filepath.Join(root, "templates")
	if err := handler.InitTemplates(templateDir); err != nil {
		slog.Error("failed to load templates", "error", err)
		os.Exit(1)
	}

	docs := handler.NewDocsHandler(filepath.Join(root, "openapi"))

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/v1/archives/list", handler.HandleList)
	mux.HandleFunc("GET /api/v1/archives/contents", handler.HandleContents)
	mux.HandleFunc("GET /api/v1/archives/readme", handler.HandleReadme)
	mux.HandleFunc("GET /api/v1/archives/changelog", handler.HandleChangelog)
	mux.HandleFunc("GET /api/v1/archives/repopack", handler.HandleRepopack)
	mux.HandleFunc("GET /api/v1/archives/repomix", handler.HandleRepopack)

	// Docs
	mux.HandleFunc("GET /docs", handler.RedirectDocs)
	mux.HandleFunc("GET /docs/", docs.HandleDocs)
	mux.HandleFunc("GET /docs/api/v1/openapi.yaml", docs.HandleOpenAPISpec)

	// Error pages
	mux.HandleFunc("GET /404", handler.HandleNotFound)
	mux.HandleFunc("GET /422", handler.HandleUnprocessable)
	mux.HandleFunc("GET /500", handler.HandleInternalError)

	// Static files
	staticDir := filepath.Join(root, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Home page (must be last to act as catch-all)
	mux.HandleFunc("GET /", handler.HandleHome)

	// CORS wrapping for API routes
	corsHandler := handler.CORSMiddleware()
	wrappedMux := corsHandler.Handler(mux)

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, wrappedMux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func projectRoot() string {
	// Check if we're running from cmd/server
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		// Check a few common locations
		for _, candidate := range []string{
			dir,
			filepath.Join(dir, "..", ".."),
			".",
		} {
			if _, err := os.Stat(filepath.Join(candidate, "templates")); err == nil {
				abs, _ := filepath.Abs(candidate)
				return abs
			}
		}
	}

	// Try working directory
	wd, _ := os.Getwd()

	// Walk up looking for templates dir
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "templates")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback: check PROJECT_ROOT env
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return root
	}

	// Check if templates exist relative to binary
	if strings.Contains(wd, "cmd") {
		return filepath.Join(wd, "..", "..")
	}

	return wd
}
