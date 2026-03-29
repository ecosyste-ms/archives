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
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type result struct {
		dest string
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		dest, err := a.doExtract(path, dir)
		ch <- result{dest, err}
	}()

	select {
	case <-ctx.Done():
		slog.Info("extraction timed out after 30 seconds")
		return "", nil
	case r := <-ch:
		if r.err != nil {
			if strings.Contains(r.err.Error(), "too many files") {
				slog.Info("archive has too many files (>10,000), skipping extraction")
				return "", nil
			}
			return "", r.err
		}
		return r.dest, nil
	}
}

func (a *RemoteArchive) doExtract(path, dir string) (string, error) {
	mime := detectMimeType(path)

	switch mime {
	case "application/zip", "application/java-archive":
		return extractZip(path, dir)
	case "application/gzip":
		return extractTarGz(path, dir)
	case "application/x-tar":
		return extractTar(path, dir)
	default:
		slog.Info("unsupported mime type", "mime", mime)
		return "", nil
	}
}

func extractZip(path, dir string) (string, error) {
	destination := filepath.Join(dir, "zip")
	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", err
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	// Check if we should strip a single top-level directory
	shouldStrip := shouldStripTopLevel(zipEntryNames(r.File))

	fileCount := 0
	for _, f := range r.File {
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

		entryPath := filepath.Join(destination, stripped)
		absEntry, _ := filepath.Abs(entryPath)
		absDest, _ := filepath.Abs(destination)
		if !strings.HasPrefix(absEntry, absDest) {
			return "", fmt.Errorf("blocked extraction outside target dir")
		}

		fileCount++
		if fileCount > maxFileCount {
			return "", fmt.Errorf("too many files in archive")
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(entryPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(entryPath), 0755); err != nil {
			return "", err
		}

		if err := extractZipFile(f, entryPath); err != nil {
			slog.Warn("failed to extract file", "name", f.Name, "error", err)
			continue
		}
	}

	return destination, nil
}

// maxDecompressedFileSize is the maximum size of a single decompressed file (200MB).
// This prevents decompression bombs where a small archive expands to fill disk.
const maxDecompressedFileSize = 200 * 1024 * 1024

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, io.LimitReader(rc, maxDecompressedFileSize))
	return err
}

func extractTarGz(path, dir string) (string, error) {
	destination := filepath.Join(dir, "tar")
	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()

	return destination, extractTarReader(tar.NewReader(gz), destination, true)
}

func extractTar(path, dir string) (string, error) {
	destination := filepath.Join(dir, "tar")
	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Extract without stripping top level first, since formats like .gem
	// have flat entries (data.tar.gz, metadata.gz) with no top-level dir.
	if err := extractTarReader(tar.NewReader(f), destination, false); err != nil {
		return "", err
	}

	// Handle nested tar.gz inside outer tar (gems use data.tar.gz, hex uses contents.tar.gz)
	for _, inner := range []string{"data.tar.gz", "contents.tar.gz"} {
		innerPath := filepath.Join(destination, inner)
		if _, err := os.Stat(innerPath); err != nil {
			continue
		}

		innerDestination := filepath.Join(dir, "inner")
		if err := os.MkdirAll(innerDestination, 0755); err != nil {
			return "", err
		}

		df, err := os.Open(innerPath)
		if err != nil {
			return "", err
		}
		defer df.Close()

		gz, err := gzip.NewReader(df)
		if err != nil {
			return "", err
		}
		defer gz.Close()

		if err := extractTarReader(tar.NewReader(gz), innerDestination, false); err != nil {
			return "", err
		}
		return innerDestination, nil
	}

	return destination, nil
}

func extractTarReader(tr *tar.Reader, destination string, stripTop bool) error {
	fileCount := 0

	for {
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

		components := splitPath(header.Name)
		if len(components) == 0 {
			continue
		}

		var stripped string
		if stripTop && len(components) > 1 {
			stripped = filepath.Join(components[1:]...)
		} else if stripTop {
			continue
		} else {
			stripped = header.Name
		}
		if stripped == "" {
			continue
		}

		destPath := filepath.Join(destination, stripped)
		absDest, _ := filepath.Abs(destPath)
		absBase, _ := filepath.Abs(destination)
		if !strings.HasPrefix(absDest, absBase) {
			return fmt.Errorf("blocked extraction outside target dir")
		}

		fileCount++
		if fileCount > maxFileCount {
			return fmt.Errorf("too many files in archive")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(destPath, 0755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, io.LimitReader(tr, maxDecompressedFileSize)); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
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
