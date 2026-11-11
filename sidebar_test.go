package main

import (
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

// The sidebar lists folders first, then files, by name whatever its case,
// without hidden entries and those that can't open.
func TestListFolder(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"b.txt", "A.md", "Makefile", ".env"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"src", "Docs", ".git"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{"linked": "src", "readme": "A.md", "broken": "missing"}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	// A FIFO would block whoever reads it.
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := listFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []folderEntry{
		{"Docs", true}, {"linked", true}, {"src", true},
		{"A.md", false}, {"b.txt", false}, {"Makefile", false}, {"readme", false},
	}
	if !slices.Equal(got, want) {
		t.Errorf("listFolder:\n got %v\nwant %v", got, want)
	}

	if _, err := listFolder(filepath.Join(dir, "missing")); err == nil {
		t.Error("listFolder of a missing folder: no error")
	}
	if got, err := listFolder(filepath.Join(dir, "Docs")); err != nil || got == nil || len(got) != 0 {
		t.Errorf("listFolder of an empty folder: %v, %v; want an empty list", got, err)
	}
}
