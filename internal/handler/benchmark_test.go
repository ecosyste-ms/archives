package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupBenchmarkFixtureServer(b *testing.B) *httptest.Server {
	b.Helper()
	testdata := filepath.Join("..", "archive", "testdata")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixtures := map[string]string{
			"/base62/-/base62-2.0.1.tgz":                            "base62-2.0.1.tgz",
			"/adobe/parcel-plugin-htl/archive/refs/heads/master.zip": "parcel-plugin-htl-master.zip",
			"/splitrb/split/archive/refs/heads/main.zip":             "main.zip",
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

func BenchmarkHandleListTarGz(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
		w := httptest.NewRecorder()
		HandleList(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkHandleListZip(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/adobe/parcel-plugin-htl/archive/refs/heads/master.zip"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
		w := httptest.NewRecorder()
		HandleList(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkHandleContents(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/api/v1/archives/contents?url="+url+"&path=package.json", nil)
		w := httptest.NewRecorder()
		HandleContents(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkHandleReadme(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/api/v1/archives/readme?url="+url, nil)
		w := httptest.NewRecorder()
		HandleReadme(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkHandleChangelog(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/splitrb/split/archive/refs/heads/main.zip"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/api/v1/archives/changelog?url="+url, nil)
		w := httptest.NewRecorder()
		HandleChangelog(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkHandleHomeHTML(b *testing.B) {
	templateDir := filepath.Join("..", "..", "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		b.Skip("templates directory not found")
	}
	InitTemplates(templateDir)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		HandleHome(w, req)
	}
}

// BenchmarkConcurrentListRequests tests how well the server handles
// concurrent requests - this is the main advantage of the Go rewrite.
func BenchmarkConcurrentListRequests(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/v1/archives/list?url="+url, nil)
			w := httptest.NewRecorder()
			HandleList(w, req)
			if w.Code != 200 {
				b.Fatalf("expected 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkConcurrentReadmeRequests tests concurrent readme extraction.
func BenchmarkConcurrentReadmeRequests(b *testing.B) {
	server := setupBenchmarkFixtureServer(b)
	defer server.Close()

	url := server.URL + "/base62/-/base62-2.0.1.tgz"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/v1/archives/readme?url="+url, nil)
			w := httptest.NewRecorder()
			HandleReadme(w, req)
			if w.Code != 200 {
				b.Fatalf("expected 200, got %d", w.Code)
			}
		}
	})
}
