// Package mountfs composes named fs.FS values into one fs.FS whose top-level directories are the
// mounts.
//
// One composed tree, rather than a separate fs.FS per source, lets a file in one mount refer to a
// file in another (a design naming a shared symbol library, a project naming a common config) by an
// ordinary path such as "lib/symbols/r.sym", resolved against the composed root.
//
// The mount table can change while the FS is being read. Mount and Unmount are safe to call
// concurrently with reads, and a change applies to the next lookup, not to a file already open.
package mountfs

import (
	"errors"
	"io"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"time"
)

// Mount is one named tree. Name is a single path element: non-empty, not "." or "..", and without a
// slash.
type Mount struct {
	Name string
	FS   fs.FS
}

// FS is a mount table served as one fs.FS. The zero value is an empty table ready to use.
type FS struct {
	mu     sync.RWMutex
	byName map[string]fs.FS
}

// New composes ms. A mount with an invalid name or a nil FS is skipped, and of two mounts with the
// same name the first wins, so a caller assembling the list from several sources can put the
// overriding ones first.
func New(ms ...Mount) *FS {
	r := &FS{byName: map[string]fs.FS{}}
	for _, m := range ms {
		if _, dup := r.byName[m.Name]; dup || !validName(m.Name) || m.FS == nil {
			continue
		}
		r.byName[m.Name] = m.FS
	}
	return r
}

// Root composes ms as New does, returned as a plain fs.FS.
func Root(ms []Mount) fs.FS { return New(ms...) }

// Mount adds fsys under name, replacing any mount already there. It fails if name is not a single
// path element or fsys is nil.
func (r *FS) Mount(name string, fsys fs.FS) error {
	if !validName(name) || fsys == nil {
		return &fs.PathError{Op: "mount", Path: name, Err: fs.ErrInvalid}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byName == nil {
		r.byName = map[string]fs.FS{}
	}
	r.byName[name] = fsys
	return nil
}

// Unmount removes the mount name. Removing a name that is not mounted does nothing.
func (r *FS) Unmount(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byName, name)
}

// Mounts returns the mounts in name order.
func (r *FS) Mounts() []Mount {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Mount, 0, len(r.byName))
	for n, f := range r.byName {
		out = append(out, Mount{Name: n, FS: f})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Open implements fs.FS. "." lists the mounts as directories; "name/rest" opens rest in mount name.
func (r *FS) Open(name string) (fs.File, error) {
	if name == "." {
		return &rootDir{entries: r.entries()}, nil
	}
	fsys, rest, err := r.split("open", name)
	if err != nil {
		return nil, err
	}
	f, err := fsys.Open(rest)
	if err != nil || rest != "." {
		return f, err
	}
	// A mount's own root is "." inside its FS and the mount's name here, and fs.WalkDir and fs.Sub
	// expect a file's Stat to agree with the directory entry that led to it.
	return mountRoot{File: f, name: name}, nil
}

// ReadFile implements fs.ReadFileFS, using the mount's own ReadFile when it has one.
func (r *FS) ReadFile(name string) ([]byte, error) {
	fsys, rest, err := r.split("read", name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(fsys, rest)
}

// ReadDir implements fs.ReadDirFS.
func (r *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "." {
		return r.entries(), nil
	}
	fsys, rest, err := r.split("readdir", name)
	if err != nil {
		return nil, err
	}
	return fs.ReadDir(fsys, rest)
}

// Stat implements fs.StatFS. A mount's root reports the mount's name, as its directory entry does.
func (r *FS) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		return dirInfo("."), nil
	}
	fsys, rest, err := r.split("stat", name)
	if err != nil {
		return nil, err
	}
	if rest == "." {
		return dirInfo(name), nil
	}
	return fs.Stat(fsys, rest)
}

// split takes "mount/rest" apart, with rest "." for the mount's own root.
func (r *FS) split(op, name string) (fs.FS, string, error) {
	if !fs.ValidPath(name) {
		return nil, "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	m, rest, _ := strings.Cut(name, "/")
	r.mu.RLock()
	fsys, ok := r.byName[m]
	r.mu.RUnlock()
	if !ok {
		return nil, "", &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	}
	if rest == "" {
		rest = "."
	}
	return fsys, rest, nil
}

func (r *FS) entries() []fs.DirEntry {
	r.mu.RLock()
	names := make([]string, 0, len(r.byName))
	for n := range r.byName {
		names = append(names, n)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	out := make([]fs.DirEntry, 0, len(names))
	for _, n := range names {
		out = append(out, fs.FileInfoToDirEntry(dirInfo(n)))
	}
	return out
}

func validName(n string) bool {
	return n != "" && n != "." && n != ".." && !strings.Contains(n, "/") && fs.ValidPath(n)
}

// mountRoot is a mount's root directory opened through the composed FS.
type mountRoot struct {
	fs.File
	name string
}

func (m mountRoot) Stat() (fs.FileInfo, error) { return dirInfo(m.name), nil }

func (m mountRoot) ReadDir(n int) ([]fs.DirEntry, error) {
	d, ok := m.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: m.name, Err: errors.New("not a directory")}
	}
	return d.ReadDir(n)
}

// rootDir is the composed root opened as a directory.
type rootDir struct {
	entries []fs.DirEntry
	off     int
}

func (d *rootDir) Stat() (fs.FileInfo, error) { return dirInfo("."), nil }
func (d *rootDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: ".", Err: errors.New("is a directory")}
}
func (d *rootDir) Close() error { return nil }

func (d *rootDir) ReadDir(n int) ([]fs.DirEntry, error) {
	rest := d.entries[d.off:]
	if n <= 0 {
		d.off = len(d.entries)
		return rest, nil
	}
	if len(rest) == 0 {
		return nil, io.EOF
	}
	if n > len(rest) {
		n = len(rest)
	}
	d.off += n
	return rest[:n], nil
}

// dirInfo describes a synthetic directory, the root or a mount's own root.
type dirInfo string

func (d dirInfo) Name() string       { return string(d) }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }
