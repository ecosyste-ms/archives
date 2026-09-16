package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ecosyste-ms/archives/internal/archive"
	"github.com/ecosyste-ms/archives/internal/telemetry"
)

const (
	cacheDuration      = 60 * 24 * time.Hour // 60 days
	errorCacheDuration = time.Hour
)

func setCacheHeaders(w http.ResponseWriter) {
	seconds := int(cacheDuration.Seconds())
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", seconds, seconds))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	if status >= http.StatusInternalServerError {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		seconds := int(errorCacheDuration.Seconds())
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", seconds, seconds))
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeArchiveError(w http.ResponseWriter, r *http.Request, err error, op, msg string, extra ...any) {
	status := archiveErrorStatus(err)
	level := slog.LevelInfo
	if status == http.StatusInternalServerError {
		telemetry.RecordError(r.Context(), err)
		level = slog.LevelError
	}
	attrs := append([]any{"error", err}, extra...)
	slog.Log(r.Context(), level, "error in "+op, attrs...)
	writeError(w, status, msg)
}

func archiveErrorStatus(err error) int {
	switch {
	case errors.Is(err, archive.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, archive.ErrTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, archive.ErrInvalidArchive):
		return http.StatusUnprocessableEntity
	case errors.Is(err, archive.ErrUpstream):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func HandleList(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	a, err := archive.NewWithContext(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	files, err := a.ListFiles()
	if err != nil {
		writeArchiveError(w, r, err, "list", "failed to list archive contents", "url", rawURL)
		return
	}

	writeJSON(w, http.StatusOK, files)
}

func HandleContents(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	a, err := archive.NewWithContext(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	contents, err := a.Contents(filePath)
	if err != nil {
		writeArchiveError(w, r, err, "contents", "failed to read archive contents", "url", rawURL, "path", filePath)
		return
	}

	if contents == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, contents)
}

func HandleReadme(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	a, err := archive.NewWithContext(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	readme, err := a.Readme()
	if err != nil {
		writeArchiveError(w, r, err, "readme", "failed to extract readme", "url", rawURL)
		return
	}

	if readme == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, readme)
}

func HandleChangelog(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	a, err := archive.NewWithContext(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	cl, err := a.Changelog()
	if err != nil {
		writeArchiveError(w, r, err, "changelog", "failed to extract changelog", "url", rawURL)
		return
	}

	if cl == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, cl)
}

func HandleRepopack(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	a, err := archive.NewWithContext(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	result, err := a.Repopack()
	if err != nil {
		writeArchiveError(w, r, err, "repopack", "failed to generate repopack output", "url", rawURL)
		return
	}

	if result == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
