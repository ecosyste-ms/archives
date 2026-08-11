package archive

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestNewValidHTTPURL(t *testing.T) {
	a, err := New("http://example.com/file.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.URL != "http://example.com/file.zip" {
		t.Errorf("expected URL to be set, got %q", a.URL)
	}
}

func TestNewValidHTTPSURL(t *testing.T) {
	a, err := New("https://example.com/file.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.URL != "https://example.com/file.tar.gz" {
		t.Errorf("expected URL to be set, got %q", a.URL)
	}
}

func TestNewRejectsInvalidScheme(t *testing.T) {
	_, err := New("ftp://example.com/file.zip")
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}
}

func TestNewRejectsInvalidURL(t *testing.T) {
	_, err := New("not a url at all ://")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDownloadUsesArchiveContext(t *testing.T) {
	type contextKey string
	const key contextKey = "request"

	previousClient := httpClient
	t.Cleanup(func() { SetHTTPClient(previousClient) })

	var value string
	SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		value, _ = req.Context().Value(key).(string)
		return nil, errors.New("stop download")
	})})

	ctx := context.WithValue(context.Background(), key, "parent-span")
	a, err := NewWithContext(ctx, "https://example.com/file.zip")
	if err != nil {
		t.Fatalf("NewWithContext() error: %v", err)
	}
	if err := a.Download(t.TempDir()); err == nil {
		t.Fatal("Download() error = nil, want transport error")
	}
	if value != "parent-span" {
		t.Errorf("request context value = %q, want parent-span", value)
	}
}

func TestBasename(t *testing.T) {
	a, _ := New("https://example.com/files/foo.tar.gz")
	if got := a.Basename(); got != "foo.tar.gz" {
		t.Errorf("Basename() = %q, want %q", got, "foo.tar.gz")
	}
}

func TestExtension(t *testing.T) {
	a, _ := New("https://example.com/files/foo.tar.gz")
	if got := a.Extension(); got != ".gz" {
		t.Errorf("Extension() = %q, want %q", got, ".gz")
	}
}

func TestDomain(t *testing.T) {
	a, _ := New("https://repo.hex.pm/tarball/package-1.0.0")
	if got := a.Domain(); got != "repo.hex.pm" {
		t.Errorf("Domain() = %q, want %q", got, "repo.hex.pm")
	}
}

func TestWorkingDirectory(t *testing.T) {
	a, _ := New("https://example.com/thing.zip")
	got := a.WorkingDirectory("/tmp")
	if got != "/tmp/thing.zip" {
		t.Errorf("WorkingDirectory() = %q, want %q", got, "/tmp/thing.zip")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestScrubUTF8Valid(t *testing.T) {
	input := "hello world"
	got := scrubUTF8([]byte(input))
	if got != input {
		t.Errorf("scrubUTF8(%q) = %q, want %q", input, got, input)
	}
}

func TestScrubUTF8Invalid(t *testing.T) {
	input := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x80, 0x57} // "Hello" + invalid + "W"
	got := scrubUTF8(input)
	if got != "Hello\uFFFDW" {
		t.Errorf("scrubUTF8 = %q, want %q", got, "Hello\uFFFDW")
	}
}

func TestIsTextMime(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		{"text/plain", true},
		{"text/html", true},
		{"application/json", true},
		{"application/xml", true},
		{"application/javascript", true},
		{"application/octet-stream", true},
		{"image/png", false},
		{"application/zip", false},
	}
	for _, tt := range tests {
		if got := isTextMime(tt.mime); got != tt.want {
			t.Errorf("isTextMime(%q) = %v, want %v", tt.mime, got, tt.want)
		}
	}
}

func TestListAllFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644)

	files, err := listAllFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(files)
	if len(files) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(files), files)
	}
	expected := []string{"a.txt", "sub", filepath.Join("sub", "b.txt")}
	sort.Strings(expected)
	for i, e := range expected {
		if files[i] != e {
			t.Errorf("files[%d] = %q, want %q", i, files[i], e)
		}
	}
}
