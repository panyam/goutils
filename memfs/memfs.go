// Package memfs is an in-memory fs.FS that can be written to while it is being read.
//
// It exists for hosts with no filesystem, such as a WebAssembly build in a browser, where bytes are
// pushed in (a dropped folder, a fetched bundle) before the code that reads them through fs.FS runs.
// Every method is safe for concurrent use, so a host can add files between requests while handler
// goroutines are still reading.
//
// Directories are implied by file paths ("a/b.txt" makes "a" a directory) and can also be created
// empty with MkdirAll. A path is either a file or a directory, never both.
//
// The write methods (WriteFile, MkdirAll, Remove, Rename) follow the os package's shapes, so an FS
// satisfies read-write interfaces such as templar's WritableFS.
package memfs

import (
	"errors"
	"io/fs"
	"path"
	"strings"
	"sync"
	"testing/fstest"
)

// ErrNotEmpty is the error Remove returns, wrapped in an *fs.PathError, for a directory that still
// has entries.
var ErrNotEmpty = errors.New("directory not empty")

// FS is an in-memory file tree. The zero value is an empty FS ready to use.
//
// A file opened from an FS keeps the contents it had when it was opened. A later Put, WriteFile or
// Remove changes what the next Open sees, never a file already open.
type FS struct {
	mu    sync.RWMutex
	files fstest.MapFS
}

// New returns an FS holding files, keyed by slash-separated path. It takes ownership of each byte
// slice, so the caller must not modify one afterwards. It returns an error for an invalid path or for
// a path that is also another path's parent directory.
func New(files map[string][]byte) (*FS, error) {
	m := &FS{}
	if err := m.Replace(files); err != nil {
		return nil, err
	}
	return m, nil
}

// Open implements fs.FS.
func (m *FS) Open(name string) (fs.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files.Open(name)
}

// ReadFile implements fs.ReadFileFS. The returned slice is a copy the caller may modify.
func (m *FS) ReadFile(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files.ReadFile(name)
}

// ReadDir implements fs.ReadDirFS, listing files and subdirectories sorted by name.
func (m *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files.ReadDir(name)
}

// Stat implements fs.StatFS.
func (m *FS) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files.Stat(name)
}

// Glob implements fs.GlobFS.
func (m *FS) Glob(pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files.Glob(pattern)
}

// Put adds or replaces the file name. Like New it takes ownership of data instead of copying it,
// which matters when a host pushes in tens of megabytes it has no other use for. Parent directories
// are implied. It fails if name is invalid, is a directory, or has a file as one of its parents.
func (m *FS) Put(name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkFile("put", name); err != nil {
		return err
	}
	m.files[name] = &fstest.MapFile{Data: data, Mode: 0o444}
	return nil
}

// Replace swaps the whole tree for files in one step, so a concurrent reader sees either the old tree
// or the new one and never a mix. It takes ownership of the byte slices as New does. On error the FS
// is unchanged.
func (m *FS) Replace(files map[string][]byte) error {
	next := &FS{files: make(fstest.MapFS, len(files))}
	for name, data := range files {
		if err := next.checkFile("put", name); err != nil {
			return err
		}
		next.files[name] = &fstest.MapFile{Data: data, Mode: 0o444}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = next.files
	return nil
}

// WriteFile writes the file name with a copy of data, as os.WriteFile does, so the caller may reuse
// data. perm is recorded as the file's mode. Unlike os.WriteFile, missing parent directories are
// implied rather than an error.
func (m *FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkFile("write", name); err != nil {
		return err
	}
	m.files[name] = &fstest.MapFile{Data: append([]byte(nil), data...), Mode: perm.Perm()}
	return nil
}

// MkdirAll creates the directory p, which then exists even with nothing in it. Like os.MkdirAll it
// succeeds if p is already a directory, and fails if p or one of its parents is a file.
func (m *FS) MkdirAll(p string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p == "." {
		return nil
	}
	if err := m.checkPath("mkdir", p); err != nil {
		return err
	}
	if f, ok := m.files[p]; ok && !f.Mode.IsDir() {
		return &fs.PathError{Op: "mkdir", Path: p, Err: fs.ErrExist}
	}
	m.files[p] = &fstest.MapFile{Mode: fs.ModeDir | perm.Perm()}
	return nil
}

// Remove deletes the file or empty directory name, as os.Remove does. A directory with entries is
// an *fs.PathError wrapping ErrNotEmpty, and a missing name one wrapping fs.ErrNotExist.
func (m *FS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !fs.ValidPath(name) || name == "." {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrInvalid}
	}
	f, explicit := m.files[name]
	if explicit && !f.Mode.IsDir() {
		delete(m.files, name)
		return nil
	}
	if m.hasChildren(name) {
		return &fs.PathError{Op: "remove", Path: name, Err: ErrNotEmpty}
	}
	if !explicit {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}
	delete(m.files, name)
	return nil
}

// Rename moves the file or directory oldname to newname, taking a directory's whole subtree with it.
// As with os.Rename, a file replaces an existing file at newname. Anything else already at newname,
// or a file among newname's parents, is an error wrapping fs.ErrExist.
func (m *FS) Rename(oldname, newname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range []string{oldname, newname} {
		if !fs.ValidPath(n) || n == "." {
			return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrInvalid}
		}
	}
	old, explicit := m.files[oldname]
	if !explicit && !m.hasChildren(oldname) {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}
	if oldname == newname {
		return nil
	}
	if strings.HasPrefix(newname, oldname+"/") {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrInvalid}
	}
	if err := m.checkFile("rename", newname); err != nil {
		return &fs.PathError{Op: "rename", Path: oldname, Err: errors.Unwrap(err)}
	}
	if _, taken := m.files[newname]; taken && (!explicit || old.Mode.IsDir()) {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrExist}
	}
	moved := fstest.MapFS{}
	for k, f := range m.files {
		if k == oldname {
			moved[newname] = f
		} else if rest, ok := strings.CutPrefix(k, oldname+"/"); ok {
			moved[newname+"/"+rest] = f
		}
	}
	for k := range m.files {
		if k == oldname || strings.HasPrefix(k, oldname+"/") {
			delete(m.files, k)
		}
	}
	for k, f := range moved {
		m.files[k] = f
	}
	return nil
}

// checkFile reports whether name can be written as a file: a valid path that is not a directory,
// explicit or implied, with no file among its parents.
func (m *FS) checkFile(op, name string) error {
	if name == "." {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	if err := m.checkPath(op, name); err != nil {
		return err
	}
	if f, ok := m.files[name]; (ok && f.Mode.IsDir()) || m.hasChildren(name) {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}
	return nil
}

// checkPath reports whether name is valid with no file among its parents.
func (m *FS) checkPath(op, name string) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	if m.files == nil {
		m.files = fstest.MapFS{}
	}
	for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
		if f, ok := m.files[dir]; ok && !f.Mode.IsDir() {
			return &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
		}
	}
	return nil
}

func (m *FS) hasChildren(dir string) bool {
	for k := range m.files {
		if strings.HasPrefix(k, dir+"/") {
			return true
		}
	}
	return false
}
