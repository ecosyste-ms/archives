package archive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotFound       = errors.New("archive not found")
	ErrUpstream       = errors.New("upstream error")
	ErrTooLarge       = errors.New("archive too large")
	ErrInvalidArchive = errors.New("invalid archive")
)

const (
	directoryMode              os.FileMode = 0o755
	downloadFileMode           os.FileMode = 0o644
	downloadFilename                       = "archive"
	maxFileSize                            = 100 * 1024 * 1024 // 100MB
	maxFileCount                           = 10_000
	mimeApplicationOctetStream             = "application/octet-stream"
	userAgent                              = "archives.ecosyste.ms"
)

type RemoteArchive struct {
	URL     string
	context context.Context
}

func New(rawURL string) (*RemoteArchive, error) {
	return NewWithContext(context.Background(), rawURL)
}

func NewWithContext(ctx context.Context, rawURL string) (*RemoteArchive, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only HTTP/HTTPS URLs are allowed")
	}
	return &RemoteArchive{URL: rawURL, context: ctx}, nil
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

func (*RemoteArchive) WorkingDirectory(dir string) string {
	return filepath.Join(dir, downloadFilename)
}

func (a *RemoteArchive) Download(dir string) error {
	path := a.WorkingDirectory(dir)

	req, err := http.NewRequestWithContext(a.context, "GET", a.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: downloading: %w", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound, http.StatusGone, http.StatusForbidden:
			return fmt.Errorf("%w: HTTP %d", ErrNotFound, resp.StatusCode)
		default:
			return fmt.Errorf("%w: HTTP %d", ErrUpstream, resp.StatusCode)
		}
	}

	// Read with size limit
	limited := io.LimitReader(resp.Body, maxFileSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("%w: reading response: %w", ErrUpstream, err)
	}
	if len(data) > maxFileSize {
		return fmt.Errorf("%w: file is larger than 100MB", ErrTooLarge)
	}

	return os.WriteFile(path, data, downloadFileMode)
}

func (a *RemoteArchive) ListFiles() ([]string, error) {
	b, err := a.browse()
	if errors.Is(err, ErrInvalidArchive) {
		slog.Info("archive listing failed", "error", err)
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if b == nil {
		return []string{}, nil
	}
	defer b.close()
	return b.list(""), nil
}

type FileContent struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
	Contents  any    `json:"contents,omitempty"`
	Binary    bool   `json:"binary,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (a *RemoteArchive) Contents(filePath string) (*FileContent, error) {
	b, err := a.browse()
	if err != nil || b == nil {
		return nil, err
	}
	defer b.close()

	name := path.Clean(filePath)
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("path traversal blocked")
	}
	file, exists := b.files[name]
	if name == "." || (exists && file.IsDir) {
		return &FileContent{Name: filePath, Directory: true, Contents: b.list(name)}, nil
	}
	if !exists {
		return nil, nil
	}

	const maxContentSize = 10 * 1024 * 1024
	data, err := b.read(name, maxContentSize)
	if b.ctx.Err() != nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidArchive, err)
	}
	cmd := exec.CommandContext(b.ctx, "file", "--brief", "--mime-type", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if b.ctx.Err() != nil {
		return nil, nil
	}
	mime := mimeApplicationOctetStream
	if err == nil {
		mime = strings.TrimSpace(string(out))
	}
	result := &FileContent{Name: filePath}
	switch {
	case !isTextMime(mime):
		result.Error = "Binary file detected. Cannot display contents as text."
	case file.Size > maxContentSize:
		result.Error = "File too large to display (>10MB)."
	default:
		result.Contents = scrubUTF8(data)
		return result, nil
	}
	result.Binary, result.MimeType = true, mime
	return result, nil
}

func detectMimeTypeContext(ctx context.Context, path string) string {
	out, err := exec.CommandContext(ctx, "file", "--brief", "--mime-type", path).Output()
	if err != nil {
		return mimeApplicationOctetStream
	}
	return strings.TrimSpace(string(out))
}

func cleanupDirectory(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		slog.Error("failed to remove archive directory", "path", dir, "error", err)
	}
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
	return mime == mimeApplicationOctetStream
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
