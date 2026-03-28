package markup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const renderTimeout = 30 * time.Second

// toolCache caches which external tools are available.
var (
	toolCache   = make(map[string]bool)
	toolCacheMu sync.RWMutex
)

func toolAvailable(name string) bool {
	toolCacheMu.RLock()
	avail, cached := toolCache[name]
	toolCacheMu.RUnlock()
	if cached {
		return avail
	}

	_, err := exec.LookPath(name)
	avail = err == nil

	toolCacheMu.Lock()
	toolCache[name] = avail
	toolCacheMu.Unlock()
	return avail
}

// runTool executes an external command with content on stdin and returns stdout.
func runTool(content []byte, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(content)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	return stdout.String(), nil
}

// runToolWithTempFile writes content to a temp file, runs the command with the
// file path as an argument, and returns stdout.
func runToolWithTempFile(content []byte, ext string, name string, args ...string) (string, error) {
	f, err := os.CreateTemp("", "markup-*"+ext)
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(content); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	// Replace placeholder in args with the temp file path
	resolvedArgs := make([]string, len(args))
	for i, arg := range args {
		resolvedArgs[i] = strings.ReplaceAll(arg, "{file}", f.Name())
	}

	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, resolvedArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	return stdout.String(), nil
}

// renderAsciiDoc uses asciidoctor to convert AsciiDoc to HTML.
func renderAsciiDoc(content []byte) (string, error) {
	if !toolAvailable("asciidoctor") {
		return "", fmt.Errorf("asciidoctor not installed")
	}
	// -s = standalone (no header/footer), -o - = output to stdout
	return runTool(content, "asciidoctor", "-s", "-o", "-", "-")
}

// renderRST uses rst2html to convert reStructuredText to HTML.
func renderRST(content []byte) (string, error) {
	// Try rst2html first, then rst2html.py (naming varies by distro)
	for _, cmd := range []string{"rst2html", "rst2html.py"} {
		if toolAvailable(cmd) {
			html, err := runTool(content, cmd, "--no-raw", "--no-file-insertion")
			if err != nil {
				return "", err
			}
			return extractRSTBody(html), nil
		}
	}
	return "", fmt.Errorf("rst2html not installed")
}

// extractRSTBody pulls just the body content from rst2html's full HTML output.
var rstBodyRe = regexp.MustCompile(`(?s)<body>\s*(.*?)\s*</body>`)

func extractRSTBody(html string) string {
	matches := rstBodyRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return html
}

// renderPod uses pod2html to convert Perl POD to HTML.
func renderPod(content []byte) (string, error) {
	if !toolAvailable("pod2html") {
		return "", fmt.Errorf("pod2html not installed")
	}
	html, err := runToolWithTempFile(content, ".pod", "pod2html", "--infile={file}", "--quiet")
	if err != nil {
		return "", err
	}
	// pod2html creates a tmp cache file, clean it up
	os.Remove("pod2htmd.tmp")
	os.Remove("pod2htmi.tmp")
	return extractPodBody(html), nil
}

var podBodyRe = regexp.MustCompile(`(?s)<body[^>]*>\s*(.*?)\s*</body>`)

func extractPodBody(html string) string {
	matches := podBodyRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return html
}

// pandocRender uses pandoc to convert various formats.
func pandocRender(content []byte, fromFormat string) (string, error) {
	if !toolAvailable("pandoc") {
		return "", fmt.Errorf("pandoc not installed")
	}
	return runTool(content, "pandoc", "-f", fromFormat, "-t", "html")
}

func renderTextile(content []byte) (string, error) {
	return pandocRender(content, "textile")
}

func renderOrg(content []byte) (string, error) {
	return pandocRender(content, "org")
}

func renderCreole(content []byte) (string, error) {
	return pandocRender(content, "creole")
}

func renderMediaWiki(content []byte) (string, error) {
	return pandocRender(content, "mediawiki")
}

func renderRDoc(content []byte) (string, error) {
	// Pandoc doesn't support rdoc, so try rdoc command directly
	if toolAvailable("rdoc") {
		return runToolWithTempFile(content, ".rdoc", "rdoc", "--fmt=html", "--quiet", "{file}")
	}
	// Fall back to treating it as markdown (better than nothing)
	return renderMarkdownToHTML(content)
}
