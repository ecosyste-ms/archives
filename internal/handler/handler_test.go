package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecosyste-ms/archives/internal/archive"
)

func TestMain(m *testing.M) {
	// Use an unrestricted HTTP client for tests since the fixture
	// server runs on localhost, which the SSRF protection blocks.
	archive.SetHTTPClient(http.DefaultClient)
	os.Exit(m.Run())
}

func setupFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	// Find testdata directory
	testdata := filepath.Join("..", "archive", "testdata")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Map URL paths to fixture files
		fixtures := map[string]string{
			"/base62/-/base62-2.0.1.tgz":                                          "base62-2.0.1.tgz",
			"/adobe/parcel-plugin-htl/archive/refs/heads/master.zip":               "parcel-plugin-htl-master.zip",
			"/org/clojars/majorcluster/clj-data-adapter/0.2.1/clj-data-adapter-0.2.1.jar": "clj-data-adapter-0.2.1.jar",
			"/splitrb/split/archive/refs/heads/main.zip":                           "main.zip",
		}

		filename, ok := fixtures[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}

		data, err := os.ReadFile(filepath.Join(testdata, filename))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
	}))
}

func TestHandleListTarGz(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var files []string
	json.NewDecoder(w.Body).Decode(&files)

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["package.json"] {
		t.Error("expected package.json in file list")
	}
	if !fileSet["Readme.md"] {
		t.Error("expected Readme.md in file list")
	}
	if !fileSet["LICENSE"] {
		t.Error("expected LICENSE in file list")
	}
}

func TestHandleListZip(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/adobe/parcel-plugin-htl/archive/refs/heads/master.zip"
	req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var files []string
	json.NewDecoder(w.Body).Decode(&files)

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["README.md"] {
		t.Error("expected README.md in file list")
	}
	if !fileSet["package.json"] {
		t.Error("expected package.json in file list")
	}
}

func TestHandleListJar(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/org/clojars/majorcluster/clj-data-adapter/0.2.1/clj-data-adapter-0.2.1.jar"
	req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var files []string
	json.NewDecoder(w.Body).Decode(&files)

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["META-INF"] {
		t.Error("expected META-INF in file list")
	}
}

func TestHandleContentsFile(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/contents?url="+url+"&path=.eslintignore", nil)
	w := httptest.NewRecorder()

	HandleContents(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	if result["name"] != ".eslintignore" {
		t.Errorf("expected name .eslintignore, got %v", result["name"])
	}
	if result["directory"] != false {
		t.Errorf("expected directory false, got %v", result["directory"])
	}
	contents, ok := result["contents"].(string)
	if !ok || contents == "" {
		t.Error("expected non-empty contents")
	}
}

func TestHandleContentsFolder(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/contents?url="+url+"&path=lib", nil)
	w := httptest.NewRecorder()

	HandleContents(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	if result["name"] != "lib" {
		t.Errorf("expected name lib, got %v", result["name"])
	}
	if result["directory"] != true {
		t.Errorf("expected directory true, got %v", result["directory"])
	}

	contents, ok := result["contents"].([]any)
	if !ok {
		t.Fatal("expected contents to be an array")
	}

	fileSet := make(map[string]bool)
	for _, f := range contents {
		fileSet[fmt.Sprint(f)] = true
	}
	if !fileSet["ascii.js"] {
		t.Error("expected ascii.js in directory contents")
	}
}

func TestHandleContentsMissing(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/contents?url="+url+"&path=nonexistent", nil)
	w := httptest.NewRecorder()

	HandleContents(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleReadme(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/readme?url="+url, nil)
	w := httptest.NewRecorder()

	HandleReadme(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	if result["name"] != "Readme.md" {
		t.Errorf("expected name Readme.md, got %v", result["name"])
	}

	raw, ok := result["raw"].(string)
	if !ok || raw == "" {
		t.Error("expected non-empty raw content")
	}

	html, ok := result["html"].(string)
	if !ok || html == "" {
		t.Error("expected non-empty html content")
	}

	if result["extension"] != ".md" {
		t.Errorf("expected extension .md, got %v", result["extension"])
	}

	if result["language"] != "Markdown" {
		t.Errorf("expected language Markdown, got %v", result["language"])
	}

	others, ok := result["other_readme_files"].([]any)
	if !ok {
		t.Error("expected other_readme_files to be an array")
	}
	if len(others) != 0 {
		t.Errorf("expected 0 other readme files, got %d", len(others))
	}
}

func TestHandleChangelog(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/splitrb/split/archive/refs/heads/main.zip"
	req := httptest.NewRequest("GET", "/api/v1/archives/changelog?url="+url, nil)
	w := httptest.NewRecorder()

	HandleChangelog(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	if result["name"] != "CHANGELOG.md" {
		t.Errorf("expected name CHANGELOG.md, got %v", result["name"])
	}

	if result["extension"] != ".md" {
		t.Errorf("expected extension .md, got %v", result["extension"])
	}

	if result["language"] != "Markdown" {
		t.Errorf("expected language Markdown, got %v", result["language"])
	}

	html, ok := result["html"].(string)
	if !ok || html == "" {
		t.Error("expected non-empty html content")
	}
}

func TestHandleListMissingURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/archives/list", nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListInvalidURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/archives/list?url=ftp://example.com/file.zip", nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleContentsMissingURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/archives/contents", nil)
	w := httptest.NewRecorder()

	HandleContents(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleContentsMissingPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/archives/contents?url=http://example.com/file.tgz", nil)
	w := httptest.NewRecorder()

	HandleContents(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCacheHeaders(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"
	req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
	w := httptest.NewRecorder()

	HandleList(w, req)

	cc := w.Header().Get("Cache-Control")
	if cc == "" {
		t.Error("expected Cache-Control header")
	}
	if !contains(cc, "public") {
		t.Errorf("expected public in Cache-Control, got %q", cc)
	}
	// 60 days = 5184000 seconds, matching Rails expires_in(60.days)
	if !contains(cc, "max-age=5184000") {
		t.Errorf("expected max-age=5184000 in Cache-Control, got %q", cc)
	}
	if !contains(cc, "s-maxage=5184000") {
		t.Errorf("expected s-maxage=5184000 in Cache-Control, got %q", cc)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestHandleHomeReturnsHTML(t *testing.T) {
	// Need to init templates first
	templateDir := filepath.Join("..", "..", "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skip("templates directory not found")
	}
	if err := InitTemplates(templateDir); err != nil {
		t.Fatalf("InitTemplates: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	HandleHome(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %q", ct)
	}
	body := w.Body.String()
	if !containsStr(body, "ecosyste.ms") {
		t.Error("expected ecosyste.ms in home page")
	}
	if !containsStr(body, "Archives") {
		t.Error("expected Archives in home page")
	}
}

func TestHandleNotFoundJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleNotFound(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["error"] != "not found" {
		t.Errorf("expected error 'not found', got %q", result["error"])
	}
}

func TestHandleNotFoundHTML(t *testing.T) {
	templateDir := filepath.Join("..", "..", "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skip("templates directory not found")
	}
	InitTemplates(templateDir)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()

	HandleNotFound(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %q", ct)
	}
}
