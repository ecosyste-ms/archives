package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestHandleReadmeCancellationStopsExtractionBeforeCleanup(t *testing.T) {
	server := setupFixtureServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	binDir := t.TempDir()
	started := filepath.Join(binDir, "started")
	release := filepath.Join(binDir, "release")
	t.Setenv("EXTRACTION_STARTED", started)
	t.Setenv("EXTRACTION_RELEASE", release)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Hold MIME detection so cancellation happens after the archive is downloaded.
	script := "#!/bin/sh\nprintf '%s' \"$$\" > \"$EXTRACTION_STARTED\"\nwhile [ ! -e \"$EXTRACTION_RELEASE\" ]; do sleep 0.01; done\nprintf 'application/gzip\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "file"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(release, nil, 0o644); err != nil {
			t.Error(err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/archives/readme?url="+server.URL+"/base62/-/base62-2.0.1.tgz", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		HandleReadme(w, req)
		close(done)
	}()

	var pid int
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for pid == 0 {
		select {
		case <-done:
			t.Fatalf("handler returned before extraction started: %s", w.Body.String())
		case <-deadline:
			t.Fatal("extraction did not start")
		case <-ticker.C:
			data, err := os.ReadFile(started)
			if err == nil {
				pid, _ = strconv.Atoi(string(data))
			}
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not finish after cancellation")
	}

	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("extraction process %d still exists after handler returned: %v", pid, err)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("temporary directories remain after cancellation: %v", entries)
	}
}

func TestHandleReadmeCleansExtractedFiles(t *testing.T) {
	for _, format := range []string{"tar", "zip"} {
		t.Run(format, func(t *testing.T) {
			entries := []archiveFixtureEntry{{name: "pkg/README.md", contents: "# Example"}, {name: "pkg/data/file.txt", contents: "data"}}
			var data []byte
			if format == "tar" {
				data = tarGzFixture(t, entries)
			} else {
				data = zipFixture(t, entries)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(data)
			}))
			defer server.Close()
			tempDir := t.TempDir()
			t.Setenv("TMPDIR", tempDir)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/archives/readme?url="+server.URL+"/pkg", nil)
			HandleReadme(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", w.Code, w.Body.String())
			}
			var result struct{ Raw string }
			decodeResponse(t, w, &result)
			if result.Raw != "# Example" {
				t.Errorf("readme = %q", result.Raw)
			}
			remaining, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(remaining) != 0 {
				t.Errorf("temporary directories remain after extraction: %v", remaining)
			}
		})
	}
}
