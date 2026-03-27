package archive

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxFileSize  = 100 * 1024 * 1024 // 100MB
	maxFileCount = 10_000
	userAgent    = "archives.ecosyste.ms"
)

type RemoteArchive struct {
	URL string
}

func New(rawURL string) (*RemoteArchive, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only HTTP/HTTPS URLs are allowed")
	}
	return &RemoteArchive{URL: rawURL}, nil
}

func (a *RemoteArchive) Basename() string {
	return filepath.Base(a.URL)
}

func (a *RemoteArchive) Extension() string {
	return filepath.Ext(a.Basename())
}

func (a *RemoteArchive) Domain() string {
	u, err := url.Parse(a.URL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func (a *RemoteArchive) WorkingDirectory(dir string) string {
	return filepath.Join(dir, a.Basename())
}

func (a *RemoteArchive) Download(dir string) error {
	path := a.WorkingDirectory(dir)

	req, err := http.NewRequest("GET", a.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Read with size limit
	limited := io.LimitReader(resp.Body, maxFileSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if len(data) > maxFileSize {
		return fmt.Errorf("file is larger than 100MB")
	}

	return os.WriteFile(path, data, 0644)
}

func (a *RemoteArchive) ListFiles() ([]string, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if err := a.Download(dir); err != nil {
		slog.Info("download failed", "error", err)
		return []string{}, nil
	}

	extractDir, err := a.Extract(dir)
	if err != nil {
		slog.Info("extraction failed", "error", err)
		return []string{}, nil
	}
	if extractDir == "" {
		return []string{}, nil
	}

	return listAllFiles(extractDir)
}

type FileContent struct {
	Name      string   `json:"name"`
	Directory bool     `json:"directory"`
	Contents  any      `json:"contents,omitempty"`
	Binary    bool     `json:"binary,omitempty"`
	MimeType  string   `json:"mime_type,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func (a *RemoteArchive) Contents(filePath string) (*FileContent, error) {
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

	fullPath := filepath.Join(extractDir, filePath)

	// Prevent path traversal
	absExtract, _ := filepath.Abs(extractDir)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absExtract) {
		return nil, fmt.Errorf("path traversal blocked")
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		files, err := listAllFiles(fullPath)
		if err != nil {
			return nil, err
		}
		return &FileContent{
			Name:      filePath,
			Directory: true,
			Contents:  files,
		}, nil
	}

	mime := detectMimeType(fullPath)

	if !isTextMime(mime) {
		return &FileContent{
			Name:      filePath,
			Directory: false,
			Binary:    true,
			MimeType:  mime,
			Error:     "Binary file detected. Cannot display contents as text.",
		}, nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	contents := scrubUTF8(data)

	return &FileContent{
		Name:      filePath,
		Directory: false,
		Contents:  contents,
	}, nil
}

func listAllFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func detectMimeType(path string) string {
	out, err := exec.Command("file", "--brief", "--mime-type", path).Output()
	if err != nil {
		return "application/octet-stream"
	}
	return strings.TrimSpace(string(out))
}

func isTextMime(mime string) bool {
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	for _, sub := range []string{"json", "xml", "javascript"} {
		if strings.Contains(mime, sub) {
			return true
		}
	}
	return mime == "application/octet-stream"
}

func scrubUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var b strings.Builder
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size <= 1 {
			b.WriteString("\uFFFD")
			data = data[1:]
		} else {
			b.WriteRune(r)
			data = data[size:]
		}
	}
	return b.String()
}
