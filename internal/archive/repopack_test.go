package archive

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeRepomix(t *testing.T, script string) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "repomix"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func repopackFixture(t *testing.T, files map[string]string) (*RemoteArchive, string) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	serveArchive(t, skillsTarGz(t, files))
	a, err := New("https://example.com/pkg.tgz")
	if err != nil {
		t.Fatal(err)
	}
	return a, tempDir
}

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected %s to be empty, got %v", dir, entries)
	}
}

func TestRepopackStopsRepomixAtTimeout(t *testing.T) {
	previous := repomixTimeout
	t.Cleanup(func() { repomixTimeout = previous })
	repomixTimeout = 200 * time.Millisecond

	fakeRepomix(t, "exec sleep 30\n")
	a, tempDir := repopackFixture(t, map[string]string{"pkg/notes.md": "# hi\n"})

	start := time.Now()
	result, err := a.Repopack()
	if err != nil || result != nil {
		t.Fatalf("Repopack() = %v, %v, want nil, nil", result, err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Repopack() took %s, want it stopped near the timeout", elapsed)
	}
	assertEmptyDir(t, tempDir)
}

func TestRepopackRejectsOversizedOutput(t *testing.T) {
	previous := maxRepopackOutput
	t.Cleanup(func() { maxRepopackOutput = previous })
	maxRepopackOutput = 16

	fakeRepomix(t, "printf '%s' 'this output is longer than sixteen bytes' > repomix-output.txt\n")
	a, tempDir := repopackFixture(t, map[string]string{"pkg/notes.md": "# hi\n"})

	_, err := a.Repopack()
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Repopack() error = %v, want %v", err, ErrTooLarge)
	}
	assertEmptyDir(t, tempDir)
}
