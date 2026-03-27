package archive

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-enry/go-enry/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var supportedReadmeFormats = regexp.MustCompile(
	`(?i)\.(md|mdown|mkdn|mdn|mdtext|markdown|textile|org|creole|mediawiki|wiki|adoc|asciidoc|asc|re?st(\.txt)?|pod|rdoc)$`,
)

var readmePattern = regexp.MustCompile(`(?i)^readme`)
var changelogPattern = regexp.MustCompile(`(?i)^(CHANGE|HISTORY|NEWS)`)

type ReadmeResult struct {
	Name             string   `json:"name"`
	Raw              string   `json:"raw"`
	HTML             string   `json:"html"`
	Plain            string   `json:"plain"`
	Extension        string   `json:"extension"`
	Language         string   `json:"language"`
	OtherReadmeFiles []string `json:"other_readme_files"`
}

type ChangelogResult struct {
	Name             string            `json:"name"`
	Raw              string            `json:"raw"`
	HTML             string            `json:"html"`
	Plain            string            `json:"plain"`
	Parsed           map[string]string `json:"parsed"`
	Extension        string            `json:"extension"`
	Language         string            `json:"language"`
	OtherReadmeFiles []string          `json:"other_readme_files"`
}

func (a *RemoteArchive) Readme() (*ReadmeResult, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if err := a.Download(dir); err != nil {
		return nil, err
	}

	extractDir, err := a.Extract(dir)
	if err != nil {
		return nil, err
	}
	if extractDir == "" {
		return nil, nil
	}

	allFiles, err := listAllFiles(extractDir)
	if err != nil {
		return nil, err
	}

	// Find README files (match against full path, so only top-level READMEs match ^readme)
	var readmeFiles []string
	for _, f := range allFiles {
		if readmePattern.MatchString(f) {
			readmeFiles = append(readmeFiles, f)
		}
	}

	// Sort: supported formats first, then by length (shorter paths preferred)
	sort.SliceStable(readmeFiles, func(i, j int) bool {
		iSupported := supportedReadmeFormats.MatchString(readmeFiles[i])
		jSupported := supportedReadmeFormats.MatchString(readmeFiles[j])
		if iSupported != jSupported {
			return iSupported
		}
		return len(readmeFiles[i]) < len(readmeFiles[j])
	})

	if len(readmeFiles) == 0 {
		return nil, nil
	}

	readmeFile := readmeFiles[0]
	fullPath := filepath.Join(extractDir, readmeFile)

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Info("skipping readme", "file", readmeFile, "error", err)
		return nil, nil
	}

	// Check if it's actually a directory
	info, _ := os.Stat(fullPath)
	if info != nil && info.IsDir() {
		slog.Info("skipping readme directory", "file", readmeFile)
		return nil, nil
	}

	rawStr := scrubUTF8(raw)
	htmlStr := renderMarkdown(rawStr)
	plainStr := stripHTML(htmlStr)
	language := detectLanguage(readmeFile, raw)

	others := make([]string, 0)
	for _, f := range readmeFiles {
		if f != readmeFile {
			others = append(others, f)
		}
	}

	return &ReadmeResult{
		Name:             readmeFile,
		Raw:              rawStr,
		HTML:             htmlStr,
		Plain:            plainStr,
		Extension:        filepath.Ext(readmeFile),
		Language:         language,
		OtherReadmeFiles: others,
	}, nil
}

func (a *RemoteArchive) Changelog() (*ChangelogResult, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if err := a.Download(dir); err != nil {
		return nil, err
	}

	extractDir, err := a.Extract(dir)
	if err != nil {
		return nil, err
	}
	if extractDir == "" {
		return nil, nil
	}

	allFiles, err := listAllFiles(extractDir)
	if err != nil {
		return nil, err
	}

	// Find changelog files
	var changelogFiles []string
	for _, f := range allFiles {
		base := filepath.Base(f)
		fullPath := filepath.Join(extractDir, f)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			continue
		}
		if changelogPattern.MatchString(base) {
			changelogFiles = append(changelogFiles, f)
		}
	}

	// Sort: supported formats first, then by length
	sort.SliceStable(changelogFiles, func(i, j int) bool {
		iSupported := supportedReadmeFormats.MatchString(changelogFiles[i])
		jSupported := supportedReadmeFormats.MatchString(changelogFiles[j])
		if iSupported != jSupported {
			return iSupported
		}
		return len(changelogFiles[i]) < len(changelogFiles[j])
	})

	if len(changelogFiles) == 0 {
		return nil, nil
	}

	changelogFile := changelogFiles[0]
	fullPath := filepath.Join(extractDir, changelogFile)

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Info("skipping changelog", "file", changelogFile, "error", err)
		return nil, nil
	}

	info, _ := os.Stat(fullPath)
	if info != nil && info.IsDir() {
		slog.Info("skipping changelog directory", "file", changelogFile)
		return nil, nil
	}

	rawStr := scrubUTF8(raw)
	htmlStr := renderMarkdown(rawStr)
	plainStr := stripHTML(htmlStr)
	language := detectLanguage(changelogFile, raw)
	parsed := parseChangelog(rawStr)

	others := make([]string, 0)
	for _, f := range changelogFiles {
		if f != changelogFile {
			others = append(others, f)
		}
	}

	return &ChangelogResult{
		Name:             changelogFile,
		Raw:              rawStr,
		HTML:             htmlStr,
		Plain:            plainStr,
		Parsed:           parsed,
		Extension:        filepath.Ext(changelogFile),
		Language:         language,
		OtherReadmeFiles: others,
	}, nil
}

func renderMarkdown(content string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return ""
	}
	return buf.String()
}

func stripHTML(htmlStr string) string {
	var b strings.Builder
	inTag := false
	for _, r := range htmlStr {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func detectLanguage(filename string, content []byte) string {
	lang := enry.GetLanguage(filename, content)
	if lang == "" {
		return ""
	}
	return lang
}

func parseChangelog(content string) map[string]string {
	parser := parseChangelogContent(content)
	return parser
}
