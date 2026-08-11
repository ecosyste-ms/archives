package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosyste-ms/archives/internal/handler"
	"github.com/ecosyste-ms/archives/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	shutdownTelemetry, err := telemetry.Start(context.Background(), telemetry.ConfigFromEnv())
	if err != nil {
		slog.Warn("AppSignal disabled", "error", err)
	}
	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			slog.Warn("failed to shut down AppSignal", "error", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	// Find project root (where templates and openapi dirs live)
	root := projectRoot()

	staticDir := filepath.Join(root, "static")
	if err := handler.InitAssets(staticDir); err != nil {
		return fmt.Errorf("load static assets: %w", err)
	}

	templateDir := filepath.Join(root, "templates")
	if err := handler.InitTemplates(templateDir); err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	docs := handler.NewDocsHandler(filepath.Join(root, "openapi"))

	mux := http.NewServeMux()

	// API routes
	handle := func(pattern string, h http.Handler) {
		mux.Handle(pattern, telemetry.HTTPHandler(pattern, h))
	}
	handleFunc := func(pattern string, h http.HandlerFunc) {
		handle(pattern, h)
	}

	handleFunc("GET /api/v1/archives/list", handler.HandleList)
	handleFunc("GET /api/v1/archives/contents", handler.HandleContents)
	handleFunc("GET /api/v1/archives/readme", handler.HandleReadme)
	handleFunc("GET /api/v1/archives/changelog", handler.HandleChangelog)
	handleFunc("GET /api/v1/archives/repopack", handler.HandleRepopack)
	handleFunc("GET /api/v1/archives/repomix", handler.HandleRepopack)

	// Docs
	handleFunc("GET /docs", handler.RedirectDocs)
	handleFunc("GET /docs/", docs.HandleDocs)
	handleFunc("GET /docs/api/v1/openapi.yaml", docs.HandleOpenAPISpec)

	// Error pages
	handleFunc("GET /404", handler.HandleNotFound)
	handleFunc("GET /422", handler.HandleUnprocessable)
	handleFunc("GET /500", handler.HandleInternalError)

	// Static files
	handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Home page (must be last to act as catch-all)
	handleFunc("GET /", handler.HandleHome)

	// Middleware: CORS for API routes, security headers for all
	corsHandler := handler.CORSMiddleware()
	wrapped := handler.SecurityHeaders(corsHandler.Handler(mux))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      wrapped,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // long enough for archive download + extraction
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("starting server", "port", port)
	return server.ListenAndServe()
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
