package memfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"
)

// writableFS is templar's WritableFS method set, copied so this package need not import templar.
// templar's own MemFS is meant to become a wrapper over this one.
type writableFS interface {
	fs.FS
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(name string) error
	Rename(oldname, newname string) error
}

var (
	_ writableFS    = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
)

func mustNew(t *testing.T, files map[string]string) *FS {
	t.Helper()
	b := map[string][]byte{}
	for k, v := range files {
		b[k] = []byte(v)
	}
	m, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func read(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}
	return string(b)
}

func TestFSConformance(t *testing.T) {
	m := mustNew(t, map[string]string{"top.txt": "t", "a/b.txt": "b", "a/c/d.txt": "d"})
	if err := m.MkdirAll("empty/inner", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fstest.TestFS(m, "top.txt", "a/b.txt", "a/c/d.txt", "empty/inner"); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirListsSubdirectories(t *testing.T) {
	m := mustNew(t, map[string]string{"a/b.txt": "b", "a/c/d.txt": "d"})
	es, err := m.ReadDir("a")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range es {
		got = append(got, fmt.Sprintf("%s:%v", e.Name(), e.IsDir()))
	}
	if fmt.Sprint(got) != "[b.txt:false c:true]" {
		t.Errorf("ReadDir(a) = %v", got)
	}
}

func TestZeroValueIsUsable(t *testing.T) {
	var m FS
	if _, err := m.Open("x"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Open on empty FS: %v", err)
	}
	if err := m.Put("x/y", []byte("y")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, &m, "x/y"); got != "y" {
		t.Errorf("got %q", got)
	}
}

func TestPutIsVisibleToTheNextReadButNotAnOpenFile(t *testing.T) {
	m := mustNew(t, map[string]string{"f": "old"})
	f, err := m.Open("f")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := m.Put("f", []byte("new")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, m, "f"); got != "new" {
		t.Errorf("after Put, ReadFile = %q", got)
	}
	b, _ := io.ReadAll(f)
	if string(b) != "old" {
		t.Errorf("a file opened before Put read %q", b)
	}
}

func TestWriteFileCopiesAndReadFileReturnsACopy(t *testing.T) {
	var m FS
	src := []byte("abc")
	if err := m.WriteFile("f", src, 0o644); err != nil {
		t.Fatal(err)
	}
	src[0] = 'X'
	got, _ := m.ReadFile("f")
	if string(got) != "abc" {
		t.Errorf("WriteFile kept the caller's slice: %q", got)
	}
	got[0] = 'Y'
	if again := read(t, &m, "f"); again != "abc" {
		t.Errorf("ReadFile handed out the stored slice: %q", again)
	}
	if st, _ := m.Stat("f"); st.Mode().Perm() != 0o644 {
		t.Errorf("mode %v, want 0644", st.Mode())
	}
}

func TestAPathIsAFileOrADirectoryNeverBoth(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
	}{
		{"file under a file", map[string]string{"a": "1", "a/b": "2"}},
		{"invalid path", map[string]string{"../x": "1"}},
		{"leading slash", map[string]string{"/x": "1"}},
		{"root", map[string]string{".": "1"}},
	}
	for _, c := range cases {
		b := map[string][]byte{}
		for k, v := range c.files {
			b[k] = []byte(v)
		}
		if _, err := New(b); err == nil {
			t.Errorf("%s: New accepted %v", c.name, c.files)
		}
	}

	m := mustNew(t, map[string]string{"d/f": "1", "file": "x"})
	if err := m.Put("d", nil); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Put over a directory: %v", err)
	}
	if err := m.Put("file/inner", nil); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Put under a file: %v", err)
	}
	if err := m.MkdirAll("file", 0o755); !errors.Is(err, fs.ErrExist) {
		t.Errorf("MkdirAll over a file: %v", err)
	}
	if err := m.MkdirAll("file/x", 0o755); !errors.Is(err, fs.ErrExist) {
		t.Errorf("MkdirAll under a file: %v", err)
	}
	if err := m.MkdirAll("d", 0o755); err != nil {
		t.Errorf("MkdirAll on an existing directory: %v", err)
	}
}

func TestReplaceSwapsTheWholeTreeOrNothing(t *testing.T) {
	m := mustNew(t, map[string]string{"old": "o"})
	if err := m.Replace(map[string][]byte{"a": nil, "a/b": nil}); err == nil {
		t.Fatal("Replace accepted a conflicting tree")
	}
	if got := read(t, m, "old"); got != "o" {
		t.Errorf("a failed Replace changed the FS: %q", got)
	}
	if err := m.Replace(map[string][]byte{"new": []byte("n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Stat("old"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("old file survived Replace: %v", err)
	}
	if got := read(t, m, "new"); got != "n" {
		t.Errorf("got %q", got)
	}
}

func TestRemove(t *testing.T) {
	m := mustNew(t, map[string]string{"d/f": "1"})
	if err := m.MkdirAll("empty", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("d"); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("Remove of a directory with entries: %v", err)
	}
	if err := m.Remove("missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Remove of a missing name: %v", err)
	}
	if err := m.Remove("d/f"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Stat("d"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an implied directory outlived its last file: %v", err)
	}
	if err := m.Remove("empty"); err != nil {
		t.Errorf("Remove of an empty directory: %v", err)
	}
	if _, err := m.Stat("empty"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("removed directory still there: %v", err)
	}
}

func TestRename(t *testing.T) {
	m := mustNew(t, map[string]string{"d/a": "a", "d/sub/b": "b", "f": "f", "g": "g"})
	if err := m.Rename("d", "e/moved"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, m, "e/moved/sub/b"); got != "b" {
		t.Errorf("subtree did not move: %q", got)
	}
	if _, err := m.Stat("d"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("old directory still there: %v", err)
	}
	if err := m.Rename("f", "g"); err != nil {
		t.Errorf("file over file: %v", err)
	}
	if got := read(t, m, "g"); got != "f" {
		t.Errorf("after rename over g, g = %q", got)
	}
	if err := m.Rename("e", "g"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("directory over file: %v", err)
	}
	if err := m.Rename("g", "e"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("file over directory: %v", err)
	}
	if err := m.Rename("e", "e/inside"); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("directory into itself: %v", err)
	}
	if err := m.Rename("nope", "x"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing source: %v", err)
	}
}

// Run with -race: readers walk and read the tree while writers change it.
func TestConcurrentReadsAndWrites(t *testing.T) {
	var m FS
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				name := fmt.Sprintf("w%d/f%d", w, i%10)
				_ = m.Put(name, []byte(name))
				if i%7 == 0 {
					_ = m.Remove(name)
				}
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = fs.WalkDir(&m, ".", func(p string, d fs.DirEntry, err error) error {
					if err == nil && !d.IsDir() {
						_, _ = m.ReadFile(p)
					}
					return nil
				})
			}
		}()
	}
	wg.Wait()
}
