// Package markup renders README and documentation files to HTML.
//
// Markdown is rendered natively via goldmark. Other formats shell out
// to external tools: asciidoctor for AsciiDoc, rst2html for
// reStructuredText, pod2html for Perl POD, and pandoc as a fallback
// for Textile, Org, Creole, MediaWiki, and RDoc.
//
// When a required tool is not installed, Render returns the content
// unchanged with a false ok value so callers can decide what to do.
package markup

import (
	"path/filepath"
	"strings"
)

// Format represents a markup format.
type Format int

const (
	FormatUnknown   Format = iota
	FormatMarkdown         // .md, .markdown, .mdown, .mkdn, .mdn, .mdtext
	FormatAsciiDoc         // .adoc, .asciidoc, .asc
	FormatRST              // .rst, .rest, .rst.txt
	FormatTextile          // .textile
	FormatOrg              // .org
	FormatCreole           // .creole
	FormatMediaWiki        // .mediawiki, .wiki
	FormatPod              // .pod
	FormatRDoc             // .rdoc
)

// Result holds the rendered output.
type Result struct {
	HTML     string
	Language string
	Format   Format
}

// Render converts markup content to HTML based on the filename extension.
// Returns the rendered result and true if rendering succeeded, or an
// empty result and false if the format is unsupported or the required
// tool is not available.
func Render(filename string, content []byte) (Result, bool) {
	format := Detect(filename)
	if format == FormatUnknown {
		return Result{}, false
	}

	renderer, ok := renderers[format]
	if !ok {
		return Result{}, false
	}

	html, err := renderer(content)
	if err != nil {
		return Result{}, false
	}

	return Result{
		HTML:     html,
		Language: languageName(format),
		Format:   format,
	}, true
}

// Detect identifies the markup format from a filename.
func Detect(filename string) Format {
	lower := strings.ToLower(filename)

	// Handle .rst.txt as a special case
	if strings.HasSuffix(lower, ".rst.txt") {
		return FormatRST
	}

	ext := filepath.Ext(lower)
	if f, ok := extensionMap[ext]; ok {
		return f
	}
	return FormatUnknown
}

// Supported returns true if the format can be rendered, meaning the
// required external tool (if any) is installed.
func Supported(format Format) bool {
	switch format {
	case FormatMarkdown:
		return true
	case FormatAsciiDoc:
		return toolAvailable("asciidoctor")
	case FormatRST:
		return toolAvailable("rst2html") || toolAvailable("rst2html.py")
	case FormatPod:
		return toolAvailable("pod2html")
	case FormatTextile, FormatOrg, FormatCreole, FormatMediaWiki, FormatRDoc:
		return toolAvailable("pandoc")
	default:
		return false
	}
}

var extensionMap = map[string]Format{
	".md":       FormatMarkdown,
	".markdown": FormatMarkdown,
	".mdown":    FormatMarkdown,
	".mkdn":     FormatMarkdown,
	".mdn":      FormatMarkdown,
	".mdtext":   FormatMarkdown,
	".adoc":     FormatAsciiDoc,
	".asciidoc": FormatAsciiDoc,
	".asc":      FormatAsciiDoc,
	".rst":      FormatRST,
	".rest":     FormatRST,
	".textile":  FormatTextile,
	".org":      FormatOrg,
	".creole":   FormatCreole,
	".mediawiki": FormatMediaWiki,
	".wiki":     FormatMediaWiki,
	".pod":      FormatPod,
	".rdoc":     FormatRDoc,
}

func languageName(f Format) string {
	switch f {
	case FormatMarkdown:
		return "Markdown"
	case FormatAsciiDoc:
		return "AsciiDoc"
	case FormatRST:
		return "reStructuredText"
	case FormatTextile:
		return "Textile"
	case FormatOrg:
		return "Org"
	case FormatCreole:
		return "Creole"
	case FormatMediaWiki:
		return "MediaWiki"
	case FormatPod:
		return "Pod"
	case FormatRDoc:
		return "RDoc"
	default:
		return ""
	}
}

type renderFunc func(content []byte) (string, error)

var renderers = map[Format]renderFunc{
	FormatMarkdown:  renderMarkdownToHTML,
	FormatAsciiDoc:  renderAsciiDoc,
	FormatRST:       renderRST,
	FormatTextile:   renderTextile,
	FormatOrg:       renderOrg,
	FormatCreole:    renderCreole,
	FormatMediaWiki: renderMediaWiki,
	FormatPod:       renderPod,
	FormatRDoc:      renderRDoc,
}
