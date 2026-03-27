package archive

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	input := "# Hello\n\nWorld **bold** text"
	html := renderMarkdown(input)

	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Errorf("expected h1 tag, got: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("expected strong tag, got: %s", html)
	}
}

func TestRenderMarkdownGFM(t *testing.T) {
	input := "| a | b |\n|---|---|\n| 1 | 2 |"
	html := renderMarkdown(input)

	if !strings.Contains(html, "<table>") {
		t.Errorf("expected table tag (GFM), got: %s", html)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	html := renderMarkdown("")
	if html != "" {
		t.Errorf("expected empty string, got: %q", html)
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<h1>Hello</h1>", "Hello"},
		{"<p>A <strong>bold</strong> word</p>", "A bold word"},
		{"no tags", "no tags"},
		{"", ""},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSupportedReadmeFormats(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"README.markdown", true},
		{"README.textile", true},
		{"README.org", true},
		{"README.rdoc", true},
		{"README.adoc", true},
		{"README.rst", true},
		{"README.rst.txt", true},
		{"README.exe", false},
		{"README", false},
		{"README.txt", false},
	}
	for _, tt := range tests {
		got := supportedReadmeFormats.MatchString(tt.path)
		if got != tt.want {
			t.Errorf("supportedReadmeFormats.MatchString(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestReadmePatternMatches(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"README.md", true},
		{"Readme.md", true},
		{"readme.txt", true},
		{"README", true},
		{"CONTRIBUTING.md", false},
		{"package.json", false},
	}
	for _, tt := range tests {
		got := readmePattern.MatchString(tt.name)
		if got != tt.want {
			t.Errorf("readmePattern.MatchString(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestChangelogPatternMatches(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"CHANGELOG.md", true},
		{"CHANGES.md", true},
		{"HISTORY.md", true},
		{"NEWS.md", true},
		{"changelog.md", true},
		{"README.md", false},
		{"package.json", false},
	}
	for _, tt := range tests {
		got := changelogPattern.MatchString(tt.name)
		if got != tt.want {
			t.Errorf("changelogPattern.MatchString(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseChangelogContent(t *testing.T) {
	content := `# Changelog

## 2.0.0

- Breaking change
- New feature

## 1.0.0

- Initial release
`
	parsed := parseChangelogContent(content)

	if len(parsed) == 0 {
		t.Fatal("expected parsed entries")
	}

	if _, ok := parsed["2.0.0"]; !ok {
		t.Error("expected version 2.0.0 in parsed entries")
	}
	if _, ok := parsed["1.0.0"]; !ok {
		t.Error("expected version 1.0.0 in parsed entries")
	}
}
