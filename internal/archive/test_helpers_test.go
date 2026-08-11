package archive

import (
	"os"
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
