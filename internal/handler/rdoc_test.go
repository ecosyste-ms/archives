package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ecosyste-ms/archives/internal/archive"
)

func TestHandleReadmeRDoc(t *testing.T) {
	if _, err := exec.LookPath("rdoc"); err != nil {
		t.Skip("rdoc not installed")
	}
	data, err := os.ReadFile(filepath.Join("..", "archive", "testdata", "rake-13.2.1.gem"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	defer server.Close()
	t.Chdir(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/archives/readme?url="+url.QueryEscape(server.URL+"/rake-13.2.1.gem"), nil)
	w := httptest.NewRecorder()
	HandleReadme(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var result archive.ReadmeResult
	decodeResponse(t, w, &result)
	if result.Name != "README.rdoc" || result.Language != "RDoc" {
		t.Errorf("name = %q, language = %q", result.Name, result.Language)
	}
	if !strings.Contains(result.HTML, "<h1") || !strings.Contains(result.HTML, "Ruby Make") {
		t.Errorf("expected rendered Rake heading, got: %s", result.HTML)
	}
	if !strings.Contains(result.Plain, "Ruby Make") {
		t.Errorf("expected plain README text, got: %s", result.Plain)
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("renderer created files: %v", entries)
	}
}
