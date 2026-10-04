package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

const extractionTimeout = 30 * time.Second

func (a *RemoteArchive) Extract(dir string) (string, error) {
	path := a.WorkingDirectory(dir)

	info, err := os.Stat(path)
	if err != nil {
		slog.Info("file does not exist", "path", path)
		return "", nil
	}
	if info.Size() > maxFileSize {
		slog.Info("file is larger than 100MB, skipping extraction")
		return "", nil
	}

	ctx, cancel := context.WithTimeout(a.context, extractionTimeout)
	defer cancel()

	if ctx.Err() != nil {
		return "", nil
	}

	dest, err := a.doExtract(ctx, path, dir)
	if ctx.Err() != nil {
		slog.Info("extraction aborted", "error", ctx.Err())
		return "", nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "too many files") {
			slog.Info("archive has too many files (>10,000), skipping extraction")
			return "", nil
		}
		return "", fmt.Errorf("%w: %w", ErrInvalidArchive, err)
	}
	return dest, nil
}

func (a *RemoteArchive) doExtract(ctx context.Context, path, dir string) (string, error) {
	mime := detectMimeTypeContext(ctx, path)
	if err := ctx.Err(); err != nil {
		return "", err
	}

	switch mime {
	case "application/zip", "application/java-archive", "application/vnd.android.package-archive":
		return extractZip(ctx, path, dir)
	case "application/gzip":
		return extractTarGz(ctx, path, dir)
	case "application/x-xz":
		return extractTarXz(ctx, path, dir)
	case "application/x-tar":
		return extractTar(ctx, path, dir)
	default:
		slog.Info("unsupported mime type", "mime", mime)
		return "", nil
	}
}

func extractZip(ctx context.Context, path, dir string) (string, error) {
	destination := filepath.Join(dir, "zip")
	if err := os.MkdirAll(destination, directoryMode); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Close() }()

	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	// Check if we should strip a single top-level directory
	shouldStrip := shouldStripTopLevel(zipEntryNames(r.File))

	fileCount := 0
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		// Skip symlinks
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			continue
		}

		components := splitPath(f.Name)
		if len(components) == 0 {
			continue
		}

		var stripped string
		if shouldStrip {
			stripped = filepath.Join(components[1:]...)
		} else {
			stripped = f.Name
		}
		if stripped == "" {
			continue
		}

		fileCount++
		if fileCount > maxFileCount {
			return "", fmt.Errorf("too many files in archive")
		}

		if f.FileInfo().IsDir() {
			if err := root.MkdirAll(stripped, directoryMode); err != nil {
				return "", err
			}
			continue
		}

		if err := root.MkdirAll(filepath.Dir(stripped), directoryMode); err != nil {
			return "", err
		}

		if err := extractZipFile(ctx, f, root, stripped); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			slog.Warn("failed to extract file", "name", f.Name, "error", err)
			continue
		}
	}

	return destination, nil
}

// maxDecompressedFileSize is the maximum size of a single decompressed file (200MB).
// This prevents decompression bombs where a small archive expands to fill disk.
const maxDecompressedFileSize = 200 * 1024 * 1024

func extractZipFile(ctx context.Context, f *zip.File, root *os.Root, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	out, err := root.Create(dest)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(contextReader{ctx, rc}, maxDecompressedFileSize))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractTarGz(ctx context.Context, path, dir string) (string, error) {
	destination := filepath.Join(dir, "tar")
	if err := os.MkdirAll(destination, directoryMode); err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(contextReader{ctx, f})
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	return destination, extractTarReader(ctx, gz, destination, true)
}

func extractTarXz(ctx context.Context, path, dir string) (string, error) {
	destination := filepath.Join(dir, "tar")
	if err := os.MkdirAll(destination, directoryMode); err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	xzr, err := xz.NewReader(contextReader{ctx, f})
	if err != nil {
		return "", fmt.Errorf("opening xz: %w", err)
	}

	return destination, extractTarReader(ctx, xzr, destination, true)
}

func extractTar(ctx context.Context, path, dir string) (string, error) {
	destination := filepath.Join(dir, "tar")
	if err := os.MkdirAll(destination, directoryMode); err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	// Extract without stripping top level first, since formats like .gem
	// have flat entries (data.tar.gz, metadata.gz) with no top-level dir.
	if err := extractTarReader(ctx, f, destination, false); err != nil {
		return "", err
	}

	// Handle nested tar.gz inside outer tar (gems use data.tar.gz, hex uses contents.tar.gz)
	for _, inner := range []string{"data.tar.gz", "contents.tar.gz"} {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		innerPath := filepath.Join(destination, inner)
		if _, err := os.Stat(innerPath); err != nil {
			continue
		}

		innerDestination := filepath.Join(dir, "inner")
		if err := os.MkdirAll(innerDestination, directoryMode); err != nil {
			return "", err
		}

		df, err := os.Open(innerPath)
		if err != nil {
			return "", err
		}
		defer func() { _ = df.Close() }()

		gz, err := gzip.NewReader(contextReader{ctx, df})
		if err != nil {
			return "", err
		}
		defer func() { _ = gz.Close() }()

		if err := extractTarReader(ctx, gz, innerDestination, false); err != nil {
			return "", err
		}
		return innerDestination, nil
	}

	return destination, nil
}

func extractTarReader(ctx context.Context, reader io.Reader, destination string, stripTop bool) error {
	tr := tar.NewReader(contextReader{ctx, reader})
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	fileCount := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip symlinks
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			continue
		}

		stripped, ok := strippedTarPath(header.Name, stripTop)
		if !ok || stripped == "" {
			continue
		}

		fileCount++
		if fileCount > maxFileCount {
			return fmt.Errorf("too many files in archive")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(stripped, directoryMode); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := extractTarFile(tr, root, stripped); err != nil {
				return err
			}
		}
	}

	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func strippedTarPath(name string, stripTop bool) (string, bool) {
	components := splitPath(name)
	if len(components) == 0 {
		return "", false
	}

	switch {
	case !stripTop:
		return name, true
	case len(components) > 1:
		return filepath.Join(components[1:]...), true
	default:
		return "", false
	}
}

func extractTarFile(tr *tar.Reader, root *os.Root, destPath string) error {
	if err := root.MkdirAll(filepath.Dir(destPath), directoryMode); err != nil {
		return err
	}

	out, err := root.Create(destPath)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, io.LimitReader(tr, maxDecompressedFileSize))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func zipEntryNames(files []*zip.File) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	return names
}

func shouldStripTopLevel(names []string) bool {
	if len(names) == 0 {
		return false
	}

	var topDir string
	hasNonRoot := false

	for _, name := range names {
		parts := splitPath(name)
		if len(parts) == 0 {
			continue
		}
		if topDir == "" {
			topDir = parts[0]
		} else if parts[0] != topDir {
			return false
		}
		if len(parts) > 1 {
			hasNonRoot = true
		}
	}

	return hasNonRoot
}

func splitPath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}
