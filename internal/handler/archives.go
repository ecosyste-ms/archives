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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	files, err := a.ListFiles()
	if err != nil {
		slog.Error("error in list", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contents, err := a.Contents(filePath)
	if err != nil {
		slog.Error("error in contents", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	readme, err := a.Readme()
	if err != nil {
		slog.Error("error in readme", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cl, err := a.Changelog()
	if err != nil {
		slog.Error("error in changelog", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := a.Repopack()
	if err != nil {
		slog.Error("error in repopack", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result == nil {
		writeError(w, http.StatusNotFound, "path not found")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
