package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkExtractTarGz(b *testing.B) {
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}
	data, _ := os.ReadFile(fixture)

	a, _ := New("http://example.com/base62-2.0.1.tgz")

	b.ResetTimer()
	for b.Loop() {
		dir := b.TempDir()
		os.WriteFile(a.WorkingDirectory(dir), data, 0644)
		a.Extract(dir)
	}
}

func BenchmarkExtractZip(b *testing.B) {
	fixture := filepath.Join("testdata", "parcel-plugin-htl-master.zip")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}
	data, _ := os.ReadFile(fixture)

	a, _ := New("http://example.com/parcel-plugin-htl-master.zip")

	b.ResetTimer()
	for b.Loop() {
		dir := b.TempDir()
		os.WriteFile(a.WorkingDirectory(dir), data, 0644)
		a.Extract(dir)
	}
}

func BenchmarkExtractJar(b *testing.B) {
	fixture := filepath.Join("testdata", "clj-data-adapter-0.2.1.jar")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}
	data, _ := os.ReadFile(fixture)

	a, _ := New("http://example.com/clj-data-adapter-0.2.1.jar")

	b.ResetTimer()
	for b.Loop() {
		dir := b.TempDir()
		os.WriteFile(a.WorkingDirectory(dir), data, 0644)
		a.Extract(dir)
	}
}

func BenchmarkListFiles(b *testing.B) {
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}
	data, _ := os.ReadFile(fixture)

	a, _ := New("http://example.com/base62-2.0.1.tgz")

	// Pre-extract once to get the directory
	dir := b.TempDir()
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)
	extractDir, _ := a.Extract(dir)

	b.ResetTimer()
	for b.Loop() {
		listAllFiles(extractDir)
	}
}

func BenchmarkRenderMarkdown(b *testing.B) {
	// Read a real README from the fixture
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}
	data, _ := os.ReadFile(fixture)
	a, _ := New("http://example.com/base62-2.0.1.tgz")

	dir := b.TempDir()
	os.WriteFile(a.WorkingDirectory(dir), data, 0644)
	extractDir, _ := a.Extract(dir)

	readmeData, err := os.ReadFile(filepath.Join(extractDir, "Readme.md"))
	if err != nil {
		b.Skip("Readme.md not found in fixture")
	}
	b.ResetTimer()
	for b.Loop() {
		renderFile("Readme.md", readmeData)
	}
}

func BenchmarkScrubUTF8(b *testing.B) {
	// Create a mix of valid and invalid UTF-8
	data := make([]byte, 10000)
	for i := range data {
		if i%100 == 0 {
			data[i] = 0x80 // invalid byte
		} else {
			data[i] = byte('a' + (i % 26))
		}
	}

	b.ResetTimer()
	for b.Loop() {
		scrubUTF8(data)
	}
}

func BenchmarkShouldStripTopLevel(b *testing.B) {
	names := []string{
		"pkg/a.txt", "pkg/b.txt", "pkg/sub/c.txt",
		"pkg/sub/d.txt", "pkg/sub/e.txt", "pkg/",
	}

	b.ResetTimer()
	for b.Loop() {
		shouldStripTopLevel(names)
	}
}

func BenchmarkDetectMimeType(b *testing.B) {
	fixture := filepath.Join("testdata", "base62-2.0.1.tgz")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found")
	}

	b.ResetTimer()
	for b.Loop() {
		detectMimeType(fixture)
	}
}
