package archive

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestExtractTarGzFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/base62-2.0.1.tgz")
	dir := t.TempDir()

	// Copy fixture to working directory
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if dest == "" {
		t.Fatal("Extract() returned empty destination")
	}

	files, _ := listAllFiles(dest)
	if len(files) == 0 {
		t.Fatal("expected some files to be extracted")
	}

	// Verify some known files exist (top-level dir should be stripped)
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}
	if !fileSet["package.json"] {
		t.Error("expected package.json in extracted files")
	}
	if !fileSet["Readme.md"] {
		t.Error("expected Readme.md in extracted files")
	}
}

func TestExtractZipFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "parcel-plugin-htl-master.zip")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/parcel-plugin-htl-master.zip")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if dest == "" {
		t.Fatal("Extract() returned empty destination")
	}

	files, _ := listAllFiles(dest)
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["README.md"] {
		t.Error("expected README.md in extracted files")
	}
	if !fileSet["package.json"] {
		t.Error("expected package.json in extracted files")
	}
}

func TestExtractJarFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "clj-data-adapter-0.2.1.jar")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/clj-data-adapter-0.2.1.jar")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if dest == "" {
		t.Fatal("Extract() returned empty destination")
	}

	files, _ := listAllFiles(dest)
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	// JAR files should not have top-level dir stripped
	if !fileSet[filepath.Join("META-INF", "MANIFEST.MF")] {
		t.Errorf("expected META-INF/MANIFEST.MF, got files: %v", files)
	}
}

func TestExtractApkFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "sample.apk")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/sample.apk")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if dest == "" {
		t.Fatal("Extract() returned empty destination")
	}

	files, _ := listAllFiles(dest)
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["AndroidManifest.xml"] {
		t.Errorf("expected AndroidManifest.xml, got: %v", files)
	}
	if !fileSet["classes.dex"] {
		t.Errorf("expected classes.dex, got: %v", files)
	}
}

func TestExtractGemFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "rake-13.2.1.gem")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/rake-13.2.1.gem")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if dest == "" {
		t.Fatal("Extract() returned empty destination")
	}

	files, _ := listAllFiles(dest)
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	// Gem data.tar.gz should be extracted (flat, no top-level stripping needed)
	if !fileSet["rake.gemspec"] {
		t.Errorf("expected rake.gemspec, got: %v", files)
	}
	if !fileSet[filepath.Join("lib", "rake.rb")] {
		t.Errorf("expected lib/rake.rb, got: %v", files)
	}
	if !fileSet[filepath.Join("exe", "rake")] {
		t.Errorf("expected exe/rake, got: %v", files)
	}
}

func TestExtractRejectsLargeFile(t *testing.T) {
	a, _ := New("http://example.com/big.tgz")
	dir := t.TempDir()

	// Create a file larger than 100MB
	path := a.WorkingDirectory(dir)
	f, _ := os.Create(path)
	f.Truncate(101 * 1024 * 1024)
	f.Close()

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest != "" {
		t.Error("expected empty destination for oversized file")
	}
}

func TestExtractRejectsUnsupportedMimeType(t *testing.T) {
	a, _ := New("http://example.com/file.unknown")
	dir := t.TempDir()

	path := a.WorkingDirectory(dir)
	os.WriteFile(path, []byte("not an archive"), 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest != "" {
		t.Error("expected empty destination for unsupported mime type")
	}
}

func TestExtractBlocksPathTraversal(t *testing.T) {
	a, _ := New("http://example.com/evil.tar.gz")
	dir := t.TempDir()

	path := a.WorkingDirectory(dir)

	// Create a tar.gz with path traversal
	f, _ := os.Create(path)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write a file with path traversal - include a top-level dir since tar extraction strips it
	header := &tar.Header{
		Name: "pkg/../../evil.txt",
		Mode: 0644,
		Size: 5,
	}
	tw.WriteHeader(header)
	tw.Write([]byte("oops!"))
	tw.Close()
	gw.Close()
	f.Close()

	dest, err := a.Extract(dir)
	if err == nil && dest != "" {
		t.Error("expected error or empty dest for path traversal archive")
	}
}

func TestShouldStripTopLevel(t *testing.T) {
	tests := []struct {
		names []string
		want  bool
	}{
		{[]string{"pkg/a.txt", "pkg/b.txt", "pkg/"}, true},
		{[]string{"a.txt", "b.txt"}, false},
		{[]string{"pkg/a.txt", "other/b.txt"}, false},
		{[]string{}, false},
		{[]string{"pkg/"}, false},           // only a root dir, no non-root entries
		{[]string{"pkg/", "pkg/a.txt"}, true}, // root dir plus files inside
	}
	for _, tt := range tests {
		got := shouldStripTopLevel(tt.names)
		if got != tt.want {
			t.Errorf("shouldStripTopLevel(%v) = %v, want %v", tt.names, got, tt.want)
		}
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a/b/c", []string{"a", "b", "c"}},
		{"/a/b", []string{"a", "b"}},
		{"./a/b", []string{"a", "b"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitPath(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitPath(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractTarGzFileList(t *testing.T) {
	// Test that the file list from the tar.gz fixture matches expected files
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/base62-2.0.1.tgz")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatal(err)
	}

	files, _ := listAllFiles(dest)
	sort.Strings(files)

	// These are the expected files from the Rails test (top-level dir stripped)
	expected := []string{
		".codeclimate.yml",
		".eslintignore",
		".eslintrc",
		".travis.yml",
		"CODE_OF_CONDUCT.md",
		"CONTRIBUTING.md",
		"LICENSE",
		"Readme.md",
		"benchmark",
		filepath.Join("benchmark", "benchmarks.js"),
		filepath.Join("benchmark", "benchmarks_legacy.js"),
		"fork",
		filepath.Join("fork", ".editorconfig"),
		filepath.Join("fork", ".eslintrc"),
		filepath.Join("fork", "README.md"),
		filepath.Join("fork", "package.json"),
		filepath.Join("fork", "src"),
		filepath.Join("fork", "src", "ascii.js"),
		filepath.Join("fork", "src", "custom.js"),
		filepath.Join("fork", "test"),
		filepath.Join("fork", "test", "test_base62_ascii.js"),
		filepath.Join("fork", "test", "test_base62_custom.js"),
		"index.d.ts",
		"lib",
		filepath.Join("lib", "ascii.js"),
		filepath.Join("lib", "custom.js"),
		filepath.Join("lib", "legacy.js"),
		"package.json",
		"test",
		filepath.Join("test", "test_ascii.js"),
		filepath.Join("test", "test_custom.js"),
		filepath.Join("test", "test_legacy.js"),
	}
	sort.Strings(expected)

	if len(files) != len(expected) {
		t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(files), len(expected), files, expected)
		return
	}

	for i, f := range files {
		if f != expected[i] {
			t.Errorf("files[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

func TestExtractZipFileList(t *testing.T) {
	fixture := filepath.Join("testdata", "parcel-plugin-htl-master.zip")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/parcel-plugin-htl-master.zip")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatal(err)
	}

	files, _ := listAllFiles(dest)
	sort.Strings(files)

	// Expected files from the Rails test (top-level dir stripped)
	expected := []string{
		".circleci",
		filepath.Join(".circleci", "config.yml"),
		".eslintignore",
		".eslintrc.js",
		".github",
		filepath.Join(".github", "move.yml"),
		".gitignore",
		".npmignore",
		".releaserc.js",
		".snyk",
		"CHANGELOG.md",
		"CODE_OF_CONDUCT.md",
		"CONTRIBUTING.md",
		"LICENSE.txt",
		"README.md",
		"package-lock.json",
		"package.json",
		"src",
		filepath.Join("src", "HTLAsset.js"),
		filepath.Join("src", "HelixJSAsset.js"),
		filepath.Join("src", "engine"),
		filepath.Join("src", "engine", "RuntimeTemplate.js"),
		filepath.Join("src", "index.js"),
		"test",
		filepath.Join("test", "example"),
		filepath.Join("test", "example", "bla.css"),
		filepath.Join("test", "example", "html.htl"),
		filepath.Join("test", "testGeneratedCode.js"),
	}
	sort.Strings(expected)

	if len(files) != len(expected) {
		t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(files), len(expected), files, expected)
		return
	}

	for i, f := range files {
		if f != expected[i] {
			t.Errorf("files[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

func TestExtractJarFileList(t *testing.T) {
	fixture := filepath.Join("testdata", "clj-data-adapter-0.2.1.jar")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	a, _ := New("http://example.com/clj-data-adapter-0.2.1.jar")
	dir := t.TempDir()

	data, _ := os.ReadFile(fixture)
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)

	dest, err := a.Extract(dir)
	if err != nil {
		t.Fatal(err)
	}

	files, _ := listAllFiles(dest)
	sort.Strings(files)

	// JAR files should NOT have top-level dir stripped (multiple top-level dirs)
	expected := []string{
		"META-INF",
		filepath.Join("META-INF", "MANIFEST.MF"),
		filepath.Join("META-INF", "leiningen"),
		filepath.Join("META-INF", "leiningen", "org.clojars.majorcluster"),
		filepath.Join("META-INF", "leiningen", "org.clojars.majorcluster", "clj-data-adapter"),
		filepath.Join("META-INF", "leiningen", "org.clojars.majorcluster", "clj-data-adapter", "README.md"),
		filepath.Join("META-INF", "leiningen", "org.clojars.majorcluster", "clj-data-adapter", "project.clj"),
		filepath.Join("META-INF", "maven"),
		filepath.Join("META-INF", "maven", "org.clojars.majorcluster"),
		filepath.Join("META-INF", "maven", "org.clojars.majorcluster", "clj-data-adapter"),
		filepath.Join("META-INF", "maven", "org.clojars.majorcluster", "clj-data-adapter", "pom.properties"),
		filepath.Join("META-INF", "maven", "org.clojars.majorcluster", "clj-data-adapter", "pom.xml"),
		"clj_data_adapter",
		filepath.Join("clj_data_adapter", "core.clj"),
	}
	sort.Strings(expected)

	if len(files) != len(expected) {
		t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(files), len(expected), files, expected)
		return
	}

	for i, f := range files {
		if f != expected[i] {
			t.Errorf("files[%d] = %q, want %q", i, f, expected[i])
		}
	}
}
