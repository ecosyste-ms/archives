package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ecosyste-ms/archives/internal/archive"
)

const cacheDuration = 60 * 24 * time.Hour // 60 days

func setCacheHeaders(w http.ResponseWriter) {
	seconds := int(cacheDuration.Seconds())
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", seconds, seconds))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func HandleList(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}

	a, err := archive.New(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	files, err := a.ListFiles()
	if err != nil {
		slog.Error("error in list", "error", err, "url", rawURL)
		writeError(w, http.StatusInternalServerError, "failed to list archive contents")
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

	a, err := archive.New(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	contents, err := a.Contents(filePath)
	if err != nil {
		slog.Error("error in contents", "error", err, "url", rawURL, "path", filePath)
		writeError(w, http.StatusInternalServerError, "failed to read archive contents")
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

	a, err := archive.New(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	readme, err := a.Readme()
	if err != nil {
		slog.Error("error in readme", "error", err, "url", rawURL)
		writeError(w, http.StatusInternalServerError, "failed to extract readme")
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

	a, err := archive.New(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	cl, err := a.Changelog()
	if err != nil {
		slog.Error("error in changelog", "error", err, "url", rawURL)
		writeError(w, http.StatusInternalServerError, "failed to extract changelog")
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

	a, err := archive.New(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}

	result, err := a.Repopack()
	if err != nil {
		slog.Error("error in repopack", "error", err, "url", rawURL)
		writeError(w, http.StatusInternalServerError, "failed to generate repopack output")
		return
	}

	if result == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
