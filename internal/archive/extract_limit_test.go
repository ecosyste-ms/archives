package archive

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func shrinkExtractionLimit(t *testing.T) map[string]string {
	t.Helper()
	previous := maxExtractedBytes
	t.Cleanup(func() { maxExtractedBytes = previous })
	maxExtractedBytes = 1024
	return map[string]string{
		"package/a.txt": strings.Repeat("a", 600),
		"package/b.txt": strings.Repeat("b", 600),
	}
}

func TestRepopackSkipsArchiveOverExtractionLimit(t *testing.T) {
	files := shrinkExtractionLimit(t)
	fakeRepomix(t, "echo ran > \"$REPOMIX_RAN\"\nexit 1\n")
	ran := filepath.Join(t.TempDir(), "ran")
	t.Setenv("REPOMIX_RAN", ran)
	a, tempDir := repopackFixture(t, files)

	result, err := a.Repopack()
	if err != nil || result != nil {
		t.Fatalf("Repopack() = %v, %v, want nil, nil", result, err)
	}
	if _, err := os.Stat(ran); !os.IsNotExist(err) {
		t.Errorf("repomix ran on an archive over the extraction limit")
	}
	assertEmptyDir(t, tempDir)
}

func TestExtractZipStopsAtExtractionLimit(t *testing.T) {
	files := shrinkExtractionLimit(t)
	a, err := New("https://example.com/pkg.zip")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeTestFile(t, a.WorkingDirectory(dir), zipBytes(t, files))

	dest, err := a.Extract(dir)
	if err != nil || dest != "" {
		t.Fatalf("Extract() = %q, %v, want skipped", dest, err)
	}
}
