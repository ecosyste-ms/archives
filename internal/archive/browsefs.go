package archive

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"slices"
	"time"
)

// browserFS exposes an indexed archive as a read-only fs.FS with the same
// paths HandleList returns. File contents are read on Open; Stat and ReadDir
// use the index only.
type browserFS struct {
	b        *archiveBrowser
	children map[string][]string
}

func newBrowserFS(b *archiveBrowser) *browserFS {
	children := make(map[string][]string)
	for name := range b.files {
		parent := path.Dir(name)
		children[parent] = append(children[parent], path.Base(name))
	}
	for _, names := range children {
		slices.Sort(names)
	}
	return &browserFS{b: b, children: children}
}

func (f *browserFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	if name == "." {
		return browserFileInfo{name: ".", dir: true}, nil
	}
	file, ok := f.b.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return browserFileInfo{name: path.Base(name), dir: file.IsDir, size: file.Size, mode: fs.FileMode(file.Mode).Perm(), modTime: file.ModTime}, nil
}

func (f *browserFS) ReadDir(name string) ([]fs.DirEntry, error) {
	info, err := f.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	names := f.children[name]
	entries := make([]fs.DirEntry, 0, len(names))
	for _, child := range names {
		childInfo, err := f.Stat(path.Join(name, child))
		if err != nil {
			return nil, err
		}
		entries = append(entries, fs.FileInfoToDirEntry(childInfo))
	}
	return entries, nil
}

func (f *browserFS) Open(name string) (fs.File, error) {
	info, err := f.Stat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		entries, err := f.ReadDir(name)
		if err != nil {
			return nil, err
		}
		return &browserDir{info: info, entries: entries}, nil
	}
	data, err := f.b.read(name, maxDecompressedFileSize)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &browserFile{info: info, Reader: bytes.NewReader(data)}, nil
}

type browserFileInfo struct {
	name    string
	dir     bool
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (i browserFileInfo) Name() string       { return i.name }
func (i browserFileInfo) Size() int64        { return i.size }
func (i browserFileInfo) ModTime() time.Time { return i.modTime }
func (i browserFileInfo) IsDir() bool        { return i.dir }
func (i browserFileInfo) Sys() any           { return nil }

func (i browserFileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	if i.mode == 0 {
		return 0o644
	}
	return i.mode
}

type browserFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *browserFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *browserFile) Close() error               { return nil }

type browserDir struct {
	info    fs.FileInfo
	entries []fs.DirEntry
	offset  int
}

func (d *browserDir) Stat() (fs.FileInfo, error) { return d.info, nil }
func (d *browserDir) Close() error               { return nil }
func (d *browserDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.Name(), Err: fs.ErrInvalid}
}

func (d *browserDir) ReadDir(n int) ([]fs.DirEntry, error) {
	remaining := d.entries[d.offset:]
	if n <= 0 {
		d.offset = len(d.entries)
		return remaining, nil
	}
	if len(remaining) == 0 {
		return nil, io.EOF
	}
	if n > len(remaining) {
		n = len(remaining)
	}
	d.offset += n
	return remaining[:n], nil
}
