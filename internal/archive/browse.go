package archive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/git-pkgs/archives"
)

type indexedFile struct {
	archives.FileInfo
	ordinal int
}

type archiveBrowser struct {
	ctx      context.Context
	cancel   context.CancelFunc
	dir      string
	filename string
	member   *indexedFile
	files    map[string]indexedFile
}

var browseLimits = archives.StreamOptions{
	MaxInputBytes:    maxFileSize,
	MaxEntryBytes:    maxDecompressedFileSize,
	MaxExpandedBytes: int64(maxFileCount) * maxDecompressedFileSize,
	MaxEntries:       maxFileCount,
}

func (a *RemoteArchive) browse() (*archiveBrowser, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	if err := a.Download(dir); err != nil {
		cleanupDirectory(dir)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context, extractionTimeout)
	b := &archiveBrowser{ctx: ctx, cancel: cancel, dir: dir}
	err = b.index()
	if err != nil {
		b.close()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, errors.ErrUnsupported) || errors.Is(err, archives.ErrEntryLimit) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidArchive, err)
	}
	return b, nil
}

func (b *archiveBrowser) close() {
	b.cancel()
	cleanupDirectory(b.dir)
}

func (b *archiveBrowser) open() (*archives.Stream, func(), error) {
	f, err := os.Open(filepath.Join(b.dir, downloadFilename))
	if err != nil {
		return nil, nil, err
	}
	s, err := archives.OpenStreamContext(b.ctx, b.filename, f, browseLimits)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	closeOuter := func() { _ = s.Close(); _ = f.Close() }
	if b.member == nil {
		return s, closeOuter, nil
	}
	if err := seekEntry(s, b.member.ordinal); err != nil {
		closeOuter()
		return nil, nil, err
	}
	limits := browseLimits
	limits.MaxInputBytes = maxDecompressedFileSize
	inner, err := archives.OpenStreamContext(b.ctx, "member.tar.gz", s, limits)
	if err != nil {
		closeOuter()
		return nil, nil, err
	}
	return inner, func() { _ = inner.Close(); closeOuter() }, nil
}

func (b *archiveBrowser) metadata() ([]archives.FileInfo, error) {
	s, closeStream, err := b.open()
	if err != nil {
		return nil, err
	}
	defer closeStream()
	var files []archives.FileInfo
	for {
		entry, err := s.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		files = append(files, entry.FileInfo)
	}
}

func (b *archiveBrowser) index() error {
	mime := detectMimeTypeContext(b.ctx, filepath.Join(b.dir, downloadFilename))
	if err := b.ctx.Err(); err != nil {
		return err
	}
	stripTop := false
	switch mime {
	case "application/zip", "application/java-archive", "application/vnd.android.package-archive":
		b.filename = "archive.zip"
	case "application/gzip":
		b.filename = "archive.tar.gz"
	case "application/x-xz":
		b.filename = "archive.tar.xz"
	case "application/x-tar":
		b.filename = "archive.tar"
	default:
		return errors.ErrUnsupported
	}
	files, err := b.metadata()
	if err != nil {
		return err
	}
	if b.filename == "archive.tar" {
		b.member = nestedMember(files)
		if b.member != nil {
			files, err = b.metadata()
			if err != nil {
				return err
			}
		}
	}
	// Strip a wrapper directory only when every entry sits under it: npm's
	// package/ has one, pub tarballs do not.
	if b.filename != "archive.tar" {
		normalized := make([]archives.FileInfo, 0, len(files))
		for _, file := range files {
			file.Path = strings.Join(splitPath(file.Path), "/")
			if file.Path != "" && file.Path != "." {
				normalized = append(normalized, file)
			}
		}
		stripTop = archives.CommonRoot(normalized) != ""
	}
	b.files = make(map[string]indexedFile)
	for i, file := range files {
		mode := os.FileMode(file.Mode)
		if !file.IsDir && !mode.IsRegular() {
			continue
		}
		name, ok := strippedTarPath(file.Path, stripTop)
		if !ok || name == "" || name == "." {
			continue
		}
		name = path.Clean(name)
		if name == "." {
			continue
		}
		if !filepath.IsLocal(name) {
			return fmt.Errorf("unsafe archive path: %s", file.Path)
		}
		if err := b.add(name, indexedFile{FileInfo: file, ordinal: i}); err != nil {
			return err
		}
	}
	return nil
}

// Preserve the service's TAR wrapper convention, including unsigned packages.
func nestedMember(files []archives.FileInfo) *indexedFile {
	for _, name := range []string{"data.tar.gz", "contents.tar.gz"} {
		for i := len(files) - 1; i >= 0; i-- {
			if path.Clean(files[i].Path) == name && os.FileMode(files[i].Mode).IsRegular() {
				return &indexedFile{FileInfo: files[i], ordinal: i}
			}
		}
	}
	return nil
}

func (b *archiveBrowser) add(name string, file indexedFile) error {
	for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
		if existing, ok := b.files[parent]; ok && !existing.IsDir {
			return fmt.Errorf("archive path beneath a file: %s", name)
		}
		b.files[parent] = indexedFile{FileInfo: archives.FileInfo{IsDir: true}}
	}
	if existing, ok := b.files[name]; ok && existing.IsDir != file.IsDir {
		return fmt.Errorf("archive file/directory conflict: %s", name)
	}
	// Disk extraction overwrites earlier files with the same normalized path.
	b.files[name] = file
	return nil
}

func (b *archiveBrowser) list(dir string) []string {
	files := make([]string, 0)
	prefix := ""
	if dir != "." && dir != "" {
		prefix = dir + "/"
	}
	for name := range b.files {
		if strings.HasPrefix(name, prefix) && name != dir {
			files = append(files, strings.TrimPrefix(name, prefix))
		}
	}
	// Compare path components to match filepath.Walk's depth-first ordering.
	slices.SortFunc(files, func(a, b string) int {
		return slices.Compare(strings.Split(a, "/"), strings.Split(b, "/"))
	})
	return files
}

func (b *archiveBrowser) read(name string, limit int64) ([]byte, error) {
	file := b.files[name]
	s, closeStream, err := b.open()
	if err != nil {
		return nil, err
	}
	defer closeStream()
	if err := seekEntry(s, file.ordinal); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(s, limit))
}

func seekEntry(s *archives.Stream, ordinal int) error {
	for i := 0; i <= ordinal; i++ {
		if _, err := s.Next(); err != nil {
			return err
		}
	}
	return nil
}
