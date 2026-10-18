package archive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const (
	testContributingFilename = "CONTRIBUTING.md"
	testPackageDirectory     = "pkg/"
	testPackageFilename      = "pkg/a.txt"
	testPackageJSONFilename  = "package.json"
	testReadmeFilename       = "README.md"
)

func makeTestDirectory(tb testing.TB, path string) {
	tb.Helper()
	if err := os.MkdirAll(path, directoryMode); err != nil {
		tb.Fatalf("create test directory: %v", err)
	}
}

func writeTestFile(tb testing.TB, path string, data []byte) {
	tb.Helper()
	if err := os.WriteFile(path, data, downloadFileMode); err != nil {
		tb.Fatalf("write test file: %v", err)
	}
}

func listAllFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func detectMimeType(path string) string {
	return detectMimeTypeContext(context.Background(), path)
}
