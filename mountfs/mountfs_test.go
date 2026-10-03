package mountfs

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

var (
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

func mapFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for k, v := range files {
		m[k] = &fstest.MapFile{Data: []byte(v)}
	}
	return m
}

func testMounts() []Mount {
	return []Mount{
		{Name: "b", FS: mapFS(map[string]string{"x/y.txt": "y", "top.txt": "t"})},
		{Name: "a", FS: mapFS(map[string]string{"one.txt": "1"})},
		{Name: "a", FS: mapFS(map[string]string{"shadow.txt": "s"})},
		{Name: "bad/name", FS: mapFS(map[string]string{"z": "z"})},
		{Name: "", FS: mapFS(map[string]string{"z": "z"})},
		{Name: "..", FS: mapFS(map[string]string{"z": "z"})},
	}
}

func TestRootIsAnFS(t *testing.T) {
	if err := fstest.TestFS(Root(testMounts()), "a/one.txt", "b/top.txt", "b/x/y.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestNewSkipsInvalidNamesAndTheFirstDuplicateWins(t *testing.T) {
	r := New(testMounts()...)
	if _, err := r.Open("a/shadow.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a repeated mount name was served: %v", err)
	}
	var names []string
	for _, m := range r.Mounts() {
		names = append(names, m.Name)
	}
	if fmt.Sprint(names) != "[a b]" {
		t.Errorf("Mounts() = %v, want [a b]", names)
	}
}

func TestAMountReachesAnotherByPath(t *testing.T) {
	r := New(
		Mount{Name: "design", FS: mapFS(map[string]string{"board.txt": "lib/symbols/r.sym"})},
		Mount{Name: "lib", FS: mapFS(map[string]string{"symbols/r.sym": "resistor"})},
	)
	ref, err := fs.ReadFile(r, "design/board.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs.ReadFile(r, string(ref))
	if err != nil || string(got) != "resistor" {
		t.Errorf("ReadFile(%q) = %q, %v", ref, got, err)
	}
	sub, err := fs.Sub(r, "lib")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fs.ReadFile(sub, "symbols/r.sym"); err != nil || string(got) != "resistor" {
		t.Errorf("fs.Sub(lib) ReadFile = %q, %v", got, err)
	}
}

func TestMountRootStatsAsTheMountName(t *testing.T) {
	r := New(testMounts()...)
	st, err := r.Stat("b")
	if err != nil || st.Name() != "b" || !st.IsDir() {
		t.Errorf("Stat(b) = %v, %v", st, err)
	}
	f, err := r.Open("b")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if st, _ := f.Stat(); st.Name() != "b" {
		t.Errorf("Open(b).Stat().Name() = %q", st.Name())
	}
}

func TestInvalidAndMissingPaths(t *testing.T) {
	r := New(testMounts()...)
	for _, p := range []string{"/a/one.txt", "a/../b", "a/"} {
		if _, err := r.Open(p); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("Open(%q): %v, want ErrInvalid", p, err)
		}
	}
	for _, p := range []string{"nope", "nope/x", "a/missing"} {
		if _, err := r.Open(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Open(%q): %v, want ErrNotExist", p, err)
		}
	}
}

func TestMountAndUnmountApplyToTheNextLookup(t *testing.T) {
	var r FS
	if err := r.Mount("m", mapFS(map[string]string{"f": "1"})); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.ReadFile("m/f"); string(got) != "1" {
		t.Errorf("got %q", got)
	}
	if err := r.Mount("m", mapFS(map[string]string{"f": "2"})); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.ReadFile("m/f"); string(got) != "2" {
		t.Errorf("Mount did not replace: %q", got)
	}
	r.Unmount("m")
	r.Unmount("never-mounted")
	if _, err := r.ReadFile("m/f"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("after Unmount: %v", err)
	}
	for _, bad := range []string{"", ".", "..", "a/b"} {
		if err := r.Mount(bad, mapFS(nil)); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("Mount(%q): %v", bad, err)
		}
	}
	if err := r.Mount("nil", nil); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("Mount with a nil FS: %v", err)
	}
}

// Run with -race: lookups and walks run while the mount table changes.
func TestConcurrentMountsAndReads(t *testing.T) {
	r := New(Mount{Name: "fixed", FS: mapFS(map[string]string{"f": "x"})})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			name := fmt.Sprintf("m%d", i%5)
			_ = r.Mount(name, mapFS(map[string]string{"f": name}))
			if i%3 == 0 {
				r.Unmount(name)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = fs.WalkDir(r, ".", func(p string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() && !strings.HasPrefix(p, "fixed/") {
					_, _ = r.ReadFile(p)
				}
				return nil
			})
			if got, err := r.ReadFile("fixed/f"); err != nil || string(got) != "x" {
				t.Errorf("fixed/f = %q, %v", got, err)
				return
			}
		}
	}()
	wg.Wait()
}
