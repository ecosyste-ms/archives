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

func TestHandleRepopackConfig(t *testing.T) {
	binDir := t.TempDir()
	script := `#!/bin/sh
set -eu
while [ "$#" -gt 0 ]; do
  case "$1" in
    --config) config="$2"; shift ;;
    --output) output="$2"; shift ;;
  esac
  shift
done
test -f "$config"
test "$(cat "$config")" = '{}'
case "$config" in
  "$PWD"/*) exit 1 ;;
  /*.json) ;;
  *) exit 1 ;;
esac
printf '%s' "$config" > "$CONFIG_PATH_RECORD"
cat README.md > "$output"
`
	if err := os.WriteFile(filepath.Join(binDir, "repomix"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(t.TempDir(), "config-path")
	t.Setenv("CONFIG_PATH_RECORD", recordPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	checkRepopackResponse(t)

	configPath, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(configPath)); !os.IsNotExist(err) {
		t.Fatalf("expected temporary config to be removed, got %v", err)
	}
}

func TestHandleRepopackWithRepomix(t *testing.T) {
	if _, err := exec.LookPath("repomix"); err != nil {
		t.Skip("repomix is not installed")
	}
	checkRepopackResponse(t)
}

func checkRepopackResponse(t *testing.T) {
	t.Helper()
	const contents = "# Example package\n\nThis README must appear in the packed output.\n"
	data := zipFixture(t, []archiveFixtureEntry{
		{name: "package/README.md", contents: contents},
		{name: "package/repomix.config.json", contents: `{"ignore":{"customPatterns":["**/*"]}}`},
		{name: "package/repomix.config.cjs", contents: `throw new Error("Archive config must not execute");`},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/archives/repopack?url="+url.QueryEscape(server.URL+"/package.zip"), nil)
	w := httptest.NewRecorder()
	HandleRepopack(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result archive.RepopackResult
	decodeResponse(t, w, &result)
	if !strings.Contains(result.Output, contents) {
		t.Fatalf("expected README contents in packed output, got %q", result.Output)
	}
}
