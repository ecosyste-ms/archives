package handler

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ecosyste-ms/archives/internal/archive"
	"github.com/ulikunitz/xz"
)

const (
	browseFormatZIP     = "zip"
	browseFormatTAR     = "tar"
	browseReadme        = "README.md"
	browsePackageReadme = "pkg/README.md"
	browseSymlink       = "sym"
)

func serveBrowseFixture(t *testing.T, data []byte) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/download?token=example"
}

func browseRequest(t *testing.T, handler http.HandlerFunc, endpoint, source, file string, result any) *httptest.ResponseRecorder {
	t.Helper()
	query := url.Values{"url": {source}, "path": {file}}
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/archives/"+endpoint+"?"+query.Encode(), nil))
	if result != nil {
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", endpoint, w.Code, w.Body.String())
		}
		decodeResponse(t, w, result)
	}
	return w
}

func TestBrowsePreservesTreeAndDuplicateFiles(t *testing.T) {
	entries := []archiveFixtureEntry{
		{name: "pkg/z.txt", contents: "z"},
		{name: "pkg/docs/b.txt", contents: "b"},
		{name: "pkg/docs.txt", contents: "sibling"},
		{name: "pkg/docs/a.txt", contents: "old"},
		{name: browsePackageReadme, contents: "# Old"},
		{name: browsePackageReadme, contents: "# New"},
		{name: "pkg/docs/./a.txt", contents: "new"},
		{name: "pkg/docs/CHANGELOG.md", contents: "# Changelog\n\n## 1.0.0\n\nFirst release\n"},
		{name: "pkg/README", contents: "plain readme"},
		{name: "pkg/docs/README.md", contents: "nested readme"},
	}
	for _, format := range []string{browseFormatZIP, "gzip", "xz"} {
		t.Run(format, func(t *testing.T) {
			var data []byte
			switch format {
			case browseFormatZIP:
				data = zipFixture(t, entries)
			case "gzip":
				data = tarGzFixture(t, entries)
			case "xz":
				data = tarXZFixture(t, entries)
			}
			source := serveBrowseFixture(t, data)
			var files []string
			browseRequest(t, HandleList, "list", source, "", &files)
			want := []string{"README", browseReadme, "docs", "docs/CHANGELOG.md", "docs/README.md", "docs/a.txt", "docs/b.txt", "docs.txt", "z.txt"}
			if !reflect.DeepEqual(files, want) {
				t.Fatalf("files = %v, want %v", files, want)
			}
			var file archive.FileContent
			browseRequest(t, HandleContents, "contents", source, "docs/a.txt", &file)
			if file.Contents != "new" {
				t.Fatalf("duplicate body = %#v", file)
			}
			var dir struct {
				Directory bool
				Contents  []string
			}
			browseRequest(t, HandleContents, "contents", source, "docs", &dir)
			if !dir.Directory || !reflect.DeepEqual(dir.Contents, []string{"CHANGELOG.md", browseReadme, "a.txt", "b.txt"}) {
				t.Fatalf("directory = %#v", dir)
			}
			var readme archive.ReadmeResult
			browseRequest(t, HandleReadme, "readme", source, "", &readme)
			if readme.Name != browseReadme || readme.Raw != "# New" || !strings.Contains(readme.HTML, "New") || !reflect.DeepEqual(readme.OtherReadmeFiles, []string{"README"}) {
				t.Fatalf("readme = %#v", readme)
			}
			var changelog archive.ChangelogResult
			browseRequest(t, HandleChangelog, "changelog", source, "", &changelog)
			if changelog.Name != "docs/CHANGELOG.md" || !strings.Contains(changelog.Parsed["1.0.0"], "First release") {
				t.Fatalf("changelog = %#v", changelog)
			}
		})
	}
}

func plainTarFixture(t *testing.T, entries []archiveFixtureEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, entry := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.contents))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(entry.contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBrowseNestedPackages(t *testing.T) {
	entries := []archiveFixtureEntry{{name: browseReadme, contents: "# Package"}, {name: "lib/code.txt", contents: "code"}}
	inner := tarGzFixture(t, entries)
	for _, format := range []string{browseFormatTAR, "gem", "hex", "unsigned-hex"} {
		t.Run(format, func(t *testing.T) {
			var data []byte
			switch format {
			case browseFormatTAR:
				data = plainTarFixture(t, entries)
			case "gem":
				data = plainTarFixture(t, []archiveFixtureEntry{{name: "metadata.gz", contents: "metadata"}, {name: "data.tar.gz", contents: string(inner)}})
			case "hex":
				data = plainTarFixture(t, []archiveFixtureEntry{
					{name: "contents.tar.gz", contents: string(inner)},
					{name: "VERSION", contents: "3"},
					{name: "CHECKSUM", contents: strings.Repeat("A", 64)},
					{name: "metadata.config", contents: "metadata"},
				})
			case "unsigned-hex":
				data = plainTarFixture(t, []archiveFixtureEntry{{name: "contents.tar.gz", contents: string(inner)}})
			}
			source := serveBrowseFixture(t, data)
			var files []string
			browseRequest(t, HandleList, "list", source, "", &files)
			if !reflect.DeepEqual(files, []string{browseReadme, "lib", "lib/code.txt"}) {
				t.Fatalf("files = %v", files)
			}
			var readme archive.ReadmeResult
			browseRequest(t, HandleReadme, "readme", source, "", &readme)
			if readme.Raw != "# Package" {
				t.Fatalf("readme = %#v", readme)
			}
		})
	}
}

func TestBrowseSkipsLinks(t *testing.T) {
	for _, format := range []string{browseFormatTAR, browseFormatZIP} {
		t.Run(format, func(t *testing.T) {
			data := tarLinkFixture(t)
			if format == browseFormatZIP {
				data = zipLinkFixture(t)
			}
			source := serveBrowseFixture(t, data)
			var files []string
			browseRequest(t, HandleList, "list", source, "", &files)
			if !reflect.DeepEqual(files, []string{browseReadme}) {
				t.Fatalf("files = %v", files)
			}
			if w := browseRequest(t, HandleContents, "contents", source, browseSymlink, nil); w.Code != http.StatusNotFound {
				t.Fatalf("link status = %d", w.Code)
			}
		})
	}
}

func TestBrowseLimits(t *testing.T) {
	entries := make([]archiveFixtureEntry, 10001)
	for i := range entries {
		entries[i].name = fmt.Sprintf("pkg/%05d", i)
	}
	source := serveBrowseFixture(t, zipFixture(t, entries))
	if w := browseRequest(t, HandleReadme, "readme", source, "", nil); w.Code != http.StatusNotFound {
		t.Fatalf("entry limit status = %d: %s", w.Code, w.Body.String())
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: browsePackageReadme, Mode: 0o644, Size: 201 * 1024 * 1024}); err != nil {
		t.Fatal(err)
	}
	// The oversized header must be rejected before its missing body is read.
	_ = tw.Close()
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	source = serveBrowseFixture(t, buf.Bytes())
	if w := browseRequest(t, HandleContents, "contents", source, browseReadme, nil); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("size limit status = %d: %s", w.Code, w.Body.String())
	}
}

func TestContentsDoesNotWriteExpandedFiles(t *testing.T) {
	realFile, err := exec.LookPath("file")
	if err != nil {
		t.Fatal(err)
	}
	tempDir, binDir := t.TempDir(), t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	t.Setenv("REAL_FILE", realFile)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\nif [ \"$3\" = - ]; then\n  count=$(find \"$TMPDIR\" -type f | wc -l)\n  if [ \"$count\" -ne 1 ]; then printf 'application/x-expanded-files\\n'; exit 0; fi\nfi\nexec \"$REAL_FILE\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "file"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	source := serveBrowseFixture(t, tarGzFixture(t, []archiveFixtureEntry{{name: browsePackageReadme, contents: "# Browse"}, {name: "pkg/unrelated.txt", contents: "unrelated"}}))
	var file archive.FileContent
	browseRequest(t, HandleContents, "contents", source, browseReadme, &file)
	if file.Contents != "# Browse" || file.Binary {
		t.Fatalf("contents = %#v", file)
	}
	remaining, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("temporary files remain: %v", remaining)
	}
}

func tarXZFixture(t *testing.T, entries []archiveFixtureEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := xw.Write(plainTarFixture(t, entries)); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarLinkFixture(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, header := range []*tar.Header{
		{Name: browseReadme, Typeflag: tar.TypeReg, Mode: 0o644, Size: 2},
		{Name: browseSymlink, Typeflag: tar.TypeSymlink, Linkname: browseReadme},
		{Name: "hard", Typeflag: tar.TypeLink, Linkname: browseReadme},
		{Name: "pipe", Typeflag: tar.TypeFifo},
	} {
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := tw.Write([]byte("ok")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipLinkFixture(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{browseReadme, browseSymlink} {
		h := &zip.FileHeader{Name: name}
		if name == browseSymlink {
			h.SetMode(os.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestContentsDisplayPolicy(t *testing.T) {
	for _, tt := range []struct {
		name      string
		body      string
		contents  string
		errorText string
	}{
		{name: "text", body: "hello\n", contents: "hello\n"},
		{name: "invalid UTF-8", body: "hello \xff\n", contents: "hello \ufffd\n"},
		{name: "binary", body: "GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff", errorText: "Binary file detected. Cannot display contents as text."},
		{name: "large", body: strings.Repeat("x", 10*1024*1024+1), errorText: "File too large to display (>10MB)."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := serveBrowseFixture(t, zipFixture(t, []archiveFixtureEntry{{name: "pkg/body", contents: tt.body}}))
			var result archive.FileContent
			browseRequest(t, HandleContents, "contents", source, "body", &result)
			if result.Error != tt.errorText || result.Binary != (tt.errorText != "") {
				t.Fatalf("display policy = %#v", result)
			}
			if tt.errorText == "" && result.Contents != tt.contents {
				t.Fatalf("text = %#v, want %q", result.Contents, tt.contents)
			}
			if tt.errorText != "" && result.MimeType == "" {
				t.Fatal("missing MIME type")
			}
		})
	}
}

func TestBrowseZipRootPaths(t *testing.T) {
	for _, names := range [][]string{
		{browseReadme},
		{"./pkg/", "./pkg/README.md"},
		{"./", "./pkg/", "./pkg/README.md"},
	} {
		t.Run(strings.Join(names, ","), func(t *testing.T) {
			entries := make([]archiveFixtureEntry, len(names))
			for i, name := range names {
				entries[i].name = name
			}
			entries[len(entries)-1].contents = "# Root"
			source := serveBrowseFixture(t, zipFixture(t, entries))
			var files []string
			browseRequest(t, HandleList, "list", source, "", &files)
			if !reflect.DeepEqual(files, []string{browseReadme}) {
				t.Fatalf("files = %v", files)
			}
			var readme archive.ReadmeResult
			browseRequest(t, HandleReadme, "readme", source, "", &readme)
			if readme.Raw != "# Root" {
				t.Fatalf("readme = %#v", readme)
			}
		})
	}
}
