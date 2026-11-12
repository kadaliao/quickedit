package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func makeTree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if f[len(f)-1] == '/' {
			os.MkdirAll(p, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, nil, 0o644)
	}
	return root
}

func TestCompareNames(t *testing.T) {
	names := []string{"file10", "File2", "b", "a", "file1", "_x", "A"}
	slices.SortFunc(names, compareNames)
	want := []string{"_x", "a", "A", "b", "file1", "File2", "file10"}
	if !slices.Equal(names, want) && !slices.Equal(names, []string{"_x", "A", "a", "b", "file1", "File2", "file10"}) {
		t.Fatalf("sorted %q", names)
	}
}

// Folders come first, the stores of git and Finder stay out, and a folder
// is read as it opens.
func TestFolderTreeEntries(t *testing.T) {
	root := makeTree(t, "b.go", "a.md", "src/x.go", ".git/HEAD", ".DS_Store", "docs/", ".github/ci.yml")
	tr := newFolderTree(root)
	var names []string
	for _, p := range tr.children(root) {
		names = append(names, filepath.Base(p))
	}
	if want := []string{".github", "docs", "src", "a.md", "b.go"}; !slices.Equal(names, want) {
		t.Fatalf("entries %q, want %q", names, want)
	}
	src := filepath.Join(root, "src")
	if kids := tr.children(src); kids == nil || len(kids) != 0 {
		t.Fatalf("a closed folder not read: %q", kids)
	}
	tr.state.Open.Add(src)
	if kids := tr.children(src); len(kids) != 1 || filepath.Base(kids[0]) != "x.go" {
		t.Fatalf("src: %q", kids)
	}
	if tr.children(filepath.Join(root, "a.md")) != nil {
		t.Fatal("a file has children")
	}
	// A file added shows once the watch says its folder changed.
	os.WriteFile(filepath.Join(src, "y.go"), nil, 0o644)
	tr.forget([]string{src})
	if kids := tr.children(src); len(kids) != 2 {
		t.Fatalf("after forget: %q", kids)
	}
}

// A click on a folder opens it, and the tab's file shows with its folders
// open.
func TestFolderTreeView(t *testing.T) {
	root := makeTree(t, "README.md", "main.go", "Makefile", "go.mod", "LICENSE", "package.json",
		"src/app.ts", "src/view.tsx", "src/style.css", "src/deep/data.json", "src/deep/logo.png",
		"docs/guide.md", ".github/workflows/ci.yml", "scripts/build.sh", "notes.txt", "config.yaml")
	w := &docWindow{tree: newFolderTree(root), sidebar: defaultSidebarWidth}
	view := func(c *ui.Context) {
		// Not emptyFolderView: its text asks the app, which tests have not,
		// for its language.
		ui.Split(c, &w.sidebar, func() { w.tree.view(c, w) }, func() { ui.Box(c).Fill() }).Fill()
	}
	tt := ui.NewTester(view, 720, 520)
	tt.SetScale(2)
	tt.Frame()
	if !tt.HasText("src") || tt.HasText("app.ts") {
		t.Fatalf("closed src: %q", tt.Texts())
	}
	if err := tt.Click("src"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("app.ts") || !tt.HasText("deep") {
		t.Fatalf("src did not open: %q", tt.Texts())
	}
	// The tab of a file deep down: its folders open.
	w.tabs = []*docView{{path: filepath.Join(root, "src", "deep", "data.json")}}
	tt.Frame()
	if !tt.HasText("data.json") {
		t.Fatalf("not revealed: %q", tt.Texts())
	}
	if err := tt.Click("docs"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if out := os.Getenv("QE_TREE_SHOT"); out != "" {
		light := tt.Image()
		tt.SetDark(true)
		tt.Frame()
		f, err := os.Create(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, stack(light, tt.Image()))
	}
}
