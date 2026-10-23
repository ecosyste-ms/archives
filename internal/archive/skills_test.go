package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"testing"
	"testing/fstest"
)

func serveArchive(t *testing.T, data []byte) {
	t.Helper()
	previousClient := httpClient
	t.Cleanup(func() { SetHTTPClient(previousClient) })
	SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(data)), Request: req}, nil
	})})
}

func skillsTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBrowserFS(t *testing.T) {
	serveArchive(t, skillsTarGz(t, map[string]string{
		"package/package.json":               "{}",
		"package/skills/lint/SKILL.md":       "---\nname: lint\n---\n",
		"package/skills/lint/scripts/run.sh": "#!/bin/sh\n",
		"package/lib/deep/nested/index.js":   "x",
	}))
	a, err := New("https://example.com/pkg.tgz")
	if err != nil {
		t.Fatal(err)
	}
	b, err := a.browse()
	if err != nil || b == nil {
		t.Fatalf("browse() = %v, %v", b, err)
	}
	defer b.close()

	if err := fstest.TestFS(newBrowserFS(b), "package.json", "skills/lint/SKILL.md", "skills/lint/scripts/run.sh", "lib/deep/nested/index.js"); err != nil {
		t.Fatal(err)
	}
}

func TestSkills(t *testing.T) {
	serveArchive(t, skillsTarGz(t, map[string]string{
		"package/package.json": `{"name":"@reduxjs/toolkit"}`,
		"package/skills/build-modern-redux-apps/modern-redux/SKILL.md": "---\nname: build-modern-redux-apps/modern-redux\n" +
			"description: Set up Redux\nlibrary: '@reduxjs/toolkit'\nmetadata:\n  owner: redux\n---\nUse configureStore.\n",
		"package/skills/build-modern-redux-apps/modern-redux/scripts/check.sh": "#!/bin/sh\n",
		"package/skills/build-modern-redux-apps/modern-redux/schema.json":      `{"type":"object"}`,
		"package/skills/broken/SKILL.md":                                       "---\nname: [unclosed\n---\n",
		"package/node_modules/dep/skills/dep/SKILL.md":                         "---\nname: dep\n---\n",
	}))
	a, err := New("https://example.com/toolkit.tgz")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.Skills()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("skills = %+v", result.Skills)
	}
	s := result.Skills[0]
	if s.Path != "skills/build-modern-redux-apps/modern-redux" || s.Name != "build-modern-redux-apps/modern-redux" {
		t.Errorf("skill = %+v", s)
	}
	if s.Extra["library"] != "@reduxjs/toolkit" || s.Metadata["owner"] != "redux" || s.Body != "Use configureStore." {
		t.Errorf("skill = %+v", s)
	}
	if !s.HasScripts || s.SchemaJSON != `{"type":"object"}` || s.SourceHash == "" {
		t.Errorf("skill = %+v", s)
	}
	if len(s.Files) != 2 || len(s.Warnings) == 0 {
		t.Errorf("files = %v warnings = %v", s.Files, s.Warnings)
	}
	if len(result.Errors) != 1 {
		t.Errorf("errors = %v", result.Errors)
	}
}

func TestSkillsUnsupportedArchive(t *testing.T) {
	serveArchive(t, []byte("not an archive"))
	a, err := New("https://example.com/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.Skills()
	if err != nil {
		t.Fatal(err)
	}
	if result.Skills == nil || len(result.Skills) != 0 || result.Errors == nil {
		t.Errorf("result = %+v", result)
	}
}

func TestSkillsDeepAndUndescribed(t *testing.T) {
	serveArchive(t, skillsTarGz(t, map[string]string{
		"package/package.json":                         "{}",
		"package/a/b/c/d/e/f/g/h/skills/deep/SKILL.md": "---\nname: deep\ndescription: Nested ten levels down\n---\n",
		"package/skills/nodesc/SKILL.md":               "---\nname: nodesc\n---\nbody\n",
	}))
	a, err := New("https://example.com/deep.tgz")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.Skills()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skills) != 2 || len(result.Errors) != 0 {
		t.Fatalf("skills = %+v errors = %v", result.Skills, result.Errors)
	}
	if result.Skills[0].Path != "a/b/c/d/e/f/g/h/skills/deep" {
		t.Errorf("deep skill path = %q", result.Skills[0].Path)
	}
	nodesc := result.Skills[1]
	if nodesc.Path != "skills/nodesc" || !slices.Contains(nodesc.Warnings, "description is missing") {
		t.Errorf("undescribed skill = %+v", nodesc)
	}
}
