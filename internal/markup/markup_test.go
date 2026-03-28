package markup

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"README.md", FormatMarkdown},
		{"readme.markdown", FormatMarkdown},
		{"CHANGES.mdown", FormatMarkdown},
		{"doc.mkdn", FormatMarkdown},
		{"doc.mdn", FormatMarkdown},
		{"doc.mdtext", FormatMarkdown},
		{"README.adoc", FormatAsciiDoc},
		{"README.asciidoc", FormatAsciiDoc},
		{"README.asc", FormatAsciiDoc},
		{"README.rst", FormatRST},
		{"README.rest", FormatRST},
		{"README.rst.txt", FormatRST},
		{"README.textile", FormatTextile},
		{"README.org", FormatOrg},
		{"README.creole", FormatCreole},
		{"README.mediawiki", FormatMediaWiki},
		{"README.wiki", FormatMediaWiki},
		{"README.pod", FormatPod},
		{"README.rdoc", FormatRDoc},
		{"README", FormatUnknown},
		{"README.txt", FormatUnknown},
		{"README.exe", FormatUnknown},
		{"file.go", FormatUnknown},
	}

	for _, tt := range tests {
		got := Detect(tt.filename)
		if got != tt.want {
			t.Errorf("Detect(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestDetectCaseInsensitive(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"README.MD", FormatMarkdown},
		{"README.Markdown", FormatMarkdown},
		{"README.ADOC", FormatAsciiDoc},
		{"README.RST", FormatRST},
		{"README.RST.TXT", FormatRST},
	}

	for _, tt := range tests {
		got := Detect(tt.filename)
		if got != tt.want {
			t.Errorf("Detect(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestLanguageName(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatMarkdown, "Markdown"},
		{FormatAsciiDoc, "AsciiDoc"},
		{FormatRST, "reStructuredText"},
		{FormatTextile, "Textile"},
		{FormatOrg, "Org"},
		{FormatCreole, "Creole"},
		{FormatMediaWiki, "MediaWiki"},
		{FormatPod, "Pod"},
		{FormatRDoc, "RDoc"},
		{FormatUnknown, ""},
	}

	for _, tt := range tests {
		got := languageName(tt.format)
		if got != tt.want {
			t.Errorf("languageName(%v) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	content := []byte("# Hello\n\nThis is **bold** and *italic*.\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok for markdown")
	}
	if result.Format != FormatMarkdown {
		t.Errorf("Format = %v, want FormatMarkdown", result.Format)
	}
	if result.Language != "Markdown" {
		t.Errorf("Language = %q, want %q", result.Language, "Markdown")
	}
	if !strings.Contains(result.HTML, "<h1>Hello</h1>") {
		t.Errorf("expected h1 tag in HTML, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<strong>bold</strong>") {
		t.Errorf("expected strong tag in HTML, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<em>italic</em>") {
		t.Errorf("expected em tag in HTML, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMTable(t *testing.T) {
	content := []byte("| a | b |\n|---|---|\n| 1 | 2 |\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Errorf("expected table tag (GFM), got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMTaskList(t *testing.T) {
	content := []byte("- [x] Done\n- [ ] Todo\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if !strings.Contains(result.HTML, "checkbox") || !strings.Contains(result.HTML, "checked") {
		t.Errorf("expected checkbox in HTML, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMStrikethrough(t *testing.T) {
	content := []byte("~~deleted~~\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if !strings.Contains(result.HTML, "<del>deleted</del>") {
		t.Errorf("expected del tag, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMAutolink(t *testing.T) {
	content := []byte("Visit https://example.com for more.\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if !strings.Contains(result.HTML, `href="https://example.com"`) {
		t.Errorf("expected autolink, got: %s", result.HTML)
	}
}

func TestRenderMarkdownUnsafeHTML(t *testing.T) {
	// goldmark with WithUnsafe should pass through raw HTML
	content := []byte("<div class=\"custom\">Hello</div>\n")
	result, ok := Render("README.md", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if !strings.Contains(result.HTML, `<div class="custom">Hello</div>`) {
		t.Errorf("expected raw HTML passthrough, got: %s", result.HTML)
	}
}

func TestRenderUnknownFormat(t *testing.T) {
	_, ok := Render("README.txt", []byte("hello"))
	if ok {
		t.Error("expected not ok for unknown format")
	}
}

func TestRenderEmptyContent(t *testing.T) {
	result, ok := Render("README.md", []byte(""))
	if !ok {
		t.Fatal("Render returned not ok for empty markdown")
	}
	if result.HTML != "" {
		t.Errorf("expected empty HTML for empty content, got: %q", result.HTML)
	}
}

func TestRenderAsciiDoc(t *testing.T) {
	if !Supported(FormatAsciiDoc) {
		t.Skip("asciidoctor not installed")
	}
	content := []byte("= Hello\n\nThis is *bold* text.\n")
	result, ok := Render("README.adoc", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "AsciiDoc" {
		t.Errorf("Language = %q, want AsciiDoc", result.Language)
	}
	if !strings.Contains(result.HTML, "bold") {
		t.Errorf("expected bold in HTML, got: %s", result.HTML)
	}
}

func TestRenderRST(t *testing.T) {
	if !Supported(FormatRST) {
		t.Skip("rst2html not installed")
	}
	content := []byte("Hello\n=====\n\nThis is **bold** text.\n")
	result, ok := Render("README.rst", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "reStructuredText" {
		t.Errorf("Language = %q, want reStructuredText", result.Language)
	}
	if !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected Hello in HTML, got: %s", result.HTML)
	}
}

func TestRenderPod(t *testing.T) {
	if !Supported(FormatPod) {
		t.Skip("pod2html not installed")
	}
	content := []byte("=head1 NAME\n\nHello - a test\n\n=cut\n")
	result, ok := Render("README.pod", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "Pod" {
		t.Errorf("Language = %q, want Pod", result.Language)
	}
	if !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected Hello in HTML, got: %s", result.HTML)
	}
}

func TestRenderTextile(t *testing.T) {
	if !Supported(FormatTextile) {
		t.Skip("pandoc not installed")
	}
	content := []byte("h1. Hello\n\nThis is *bold* text.\n")
	result, ok := Render("README.textile", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "Textile" {
		t.Errorf("Language = %q, want Textile", result.Language)
	}
	if !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected Hello in HTML, got: %s", result.HTML)
	}
}

func TestRenderOrg(t *testing.T) {
	if !Supported(FormatOrg) {
		t.Skip("pandoc not installed")
	}
	content := []byte("* Hello\n\nThis is *bold* text.\n")
	result, ok := Render("README.org", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "Org" {
		t.Errorf("Language = %q, want Org", result.Language)
	}
	if !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected Hello in HTML, got: %s", result.HTML)
	}
}

func TestRenderMediaWiki(t *testing.T) {
	if !Supported(FormatMediaWiki) {
		t.Skip("pandoc not installed")
	}
	content := []byte("== Hello ==\n\nThis is '''bold''' text.\n")
	result, ok := Render("README.mediawiki", content)
	if !ok {
		t.Fatal("Render returned not ok")
	}
	if result.Language != "MediaWiki" {
		t.Errorf("Language = %q, want MediaWiki", result.Language)
	}
	if !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected Hello in HTML, got: %s", result.HTML)
	}
}

func TestSupported(t *testing.T) {
	// Markdown should always be supported (native)
	if !Supported(FormatMarkdown) {
		t.Error("expected Markdown to be supported")
	}

	// Unknown should never be supported
	if Supported(FormatUnknown) {
		t.Error("expected Unknown to not be supported")
	}
}

func TestToolAvailableCaching(t *testing.T) {
	// Call twice to exercise the cache path
	first := toolAvailable("sh")
	second := toolAvailable("sh")
	if first != second {
		t.Error("expected consistent results from toolAvailable cache")
	}
}

func TestExtractRSTBody(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
<div class="document">
<h1>Hello</h1>
</div>
</body></html>`
	got := extractRSTBody(input)
	if !strings.Contains(got, "<h1>Hello</h1>") {
		t.Errorf("expected body content, got: %s", got)
	}
	if strings.Contains(got, "<html>") {
		t.Error("expected html wrapper to be stripped")
	}
}

func TestExtractPodBody(t *testing.T) {
	input := `<html><body id="x"><h1>Hello</h1></body></html>`
	got := extractPodBody(input)
	if !strings.Contains(got, "<h1>Hello</h1>") {
		t.Errorf("expected body content, got: %s", got)
	}
}

func BenchmarkRenderMarkdown(b *testing.B) {
	content := []byte("# Hello\n\nThis is a **benchmark** with [links](https://example.com).\n\n- item 1\n- item 2\n- item 3\n\n```go\nfunc main() {}\n```\n")
	for b.Loop() {
		Render("README.md", content)
	}
}

func BenchmarkDetect(b *testing.B) {
	for b.Loop() {
		Detect("README.md")
		Detect("README.adoc")
		Detect("README.rst.txt")
		Detect("README.textile")
		Detect("README.unknown")
	}
}
