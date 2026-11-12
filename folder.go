package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// A folder opens in a window of its own, which shows its files as a tree
// beside the documents, as Zed's and VS Code's do: a click on a file opens
// it there, as a tab, and one on a folder opens or closes it. The tree
// reads a folder the first time it opens, and again when the system says
// that something in it changed.

// folderTree is the tree of a window's folder. Main thread.
type folderTree struct {
	root, name string
	// dirs holds the entries of the folders read: their paths, folders
	// first, in the order of their names. isDir tells the folders among
	// the paths seen.
	dirs  map[string][]string
	isDir map[string]bool
	state ui.OutlineState[string]
	// row is the row of the outline being built, which its Prefix gives.
	row ui.OutlineRow
	// shown is the file of the tab last shown, which the tree reveals, and
	// reveal the one to scroll to once the tree is built.
	shown, reveal string
	stop          context.CancelFunc
}

// noEntries are those of a folder not read yet: none, but a folder.
var noEntries = []string{}

func newFolderTree(root string) *folderTree {
	return &folderTree{root: root, name: filepath.Base(root), dirs: map[string][]string{}, isDir: map[string]bool{root: true}}
}

// hiddenEntry reports whether the tree leaves out a file or folder named
// name: the stores of version control and of Finder.
func hiddenEntry(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".DS_Store":
		return true
	}
	return false
}

// children returns the entries of the folder at p, nil for a file. It
// reads a folder that shows its entries, the root or an open one, the
// first time; a closed folder not read yet has noEntries.
func (t *folderTree) children(p string) []string {
	if !t.isDir[p] {
		return nil
	}
	if kids, ok := t.dirs[p]; ok {
		return kids
	}
	if p != t.root && !t.state.Open.Has(p) {
		return noEntries
	}
	kids := t.read(p)
	t.dirs[p] = kids
	return kids
}

// read lists the folder at dir, folders first, each in the order of their
// names.
func (t *folderTree) read(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return noEntries
	}
	type entry struct {
		path, name string
		dir        bool
	}
	list := make([]entry, 0, len(entries))
	for _, e := range entries {
		if hiddenEntry(e.Name()) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			if info, err := os.Stat(p); err == nil {
				isDir = info.IsDir()
			}
		}
		list = append(list, entry{p, e.Name(), isDir})
	}
	slices.SortFunc(list, func(a, b entry) int {
		if a.dir != b.dir {
			if a.dir {
				return -1
			}
			return 1
		}
		if c := compareNames(a.name, b.name); c != 0 {
			return c
		}
		return strings.Compare(a.name, b.name)
	})
	paths := make([]string, len(list))
	for i, e := range list {
		paths[i] = e.path
		t.isDir[e.path] = e.dir
	}
	return paths
}

// forget drops what the tree read of the folders at dirs and in them, to
// read them again as they show.
func (t *folderTree) forget(dirs []string) {
	for _, d := range dirs {
		if d == t.root {
			clear(t.dirs)
			return
		}
		for k := range t.dirs {
			if k == d || strings.HasPrefix(k, d+string(filepath.Separator)) {
				delete(t.dirs, k)
			}
		}
	}
}

// open opens the folders down to p, a path in the tree, to show it.
func (t *folderTree) open(p string) {
	for d := filepath.Dir(p); d != t.root && t.contains(d); d = filepath.Dir(d) {
		t.isDir[d] = true
		t.state.Open.Add(d)
	}
}

// contains reports whether p is in the folder.
func (t *folderTree) contains(p string) bool {
	return strings.HasPrefix(p, t.root+string(filepath.Separator))
}

// watch reads the folders again as the system says that something in them
// changed, until the window closes.
func (t *folderTree) watch(win *mygo.Window) {
	ctx, cancel := context.WithCancel(context.Background())
	t.stop = cancel
	root := t.root
	go func() {
		fw, err := mygo.WatchFiles(ctx, root, mygo.FileWatchOptions{
			Recursive: true,
			// Which folder changed is all the tree needs.
			HintsOnly: true,
			Debounce:  100 * time.Millisecond,
			Ignore:    func(p string, isDir bool) bool { return isDir && hiddenEntry(filepath.Base(p)) },
		})
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("watch %s: %v", root, err)
			}
			return
		}
		defer fw.Close()
		for {
			events, err := fw.Next(ctx)
			if err != nil {
				return
			}
			dirs := make([]string, 0, len(events))
			for _, e := range events {
				dirs = append(dirs, filepath.Join(root, filepath.FromSlash(e.Path)))
			}
			win.Update(func() { t.forget(dirs) })
		}
	}()
}

func (t *folderTree) close() {
	if t.stop != nil {
		t.stop()
	}
}

// chevron is the arrow of a folder, pointing at its entries as it opens.
var chevron = sync.OnceValue(func() *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16">` +
		`<path d="M6 3.5 10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
})

// view shows the folder's name over its tree. w is the window, whose
// current tab's file shows chosen.
func (t *folderTree) view(c *ui.Context, w *docWindow) {
	th := c.Theme()
	cur := ""
	if v := w.current(); v != nil {
		cur = v.path
	}
	if cur != t.shown {
		t.shown = cur
		if t.contains(cur) {
			t.open(cur)
			t.reveal = cur
		}
	}
	ui.Column(c).Fill().Background(th.Surface).Children(func() {
		ui.Row(c).Height(34).Padding(0, 14).AlignItems(ui.Center).Children(func() {
			ui.Text(c, t.name).FontSize(12).FontWeight(600).TextColor(th.TextMuted).SingleLine().Tooltip(t.root)
		}).ContextMenu(func(m *ui.Menu) { t.menu(m, t.root) })
		t.state.Prefix = func(r ui.OutlineRow) { t.row = r }
		ui.Outline(c, &t.state, t.children(t.root), t.children, func(p string) { t.item(c, w, p, cur) }).
			Grow(1).Padding(0, 6, 8, 6).Label(t.name)
		if t.reveal != "" {
			for i := range t.state.Rows() {
				if t.state.Item(i) == t.reveal {
					t.state.List.ScrollIntoView(i)
					break
				}
			}
			t.reveal = ""
		}
	})
}

// item builds the row of the file or folder at p.
func (t *folderTree) item(c *ui.Context, w *docWindow, p, cur string) {
	th := c.Theme()
	r := t.row
	name := filepath.Base(p)
	dir := t.isDir[p]
	open := dir && t.state.Open.Has(p)
	row := ui.Row(c).Grow(1).Height(24).PaddingStart(4 + 14*float32(r.Depth)).PaddingEnd(8).Gap(4).AlignItems(ui.Center).Radius(5)
	switch {
	case p == cur:
		row.Background(th.Accent.Alpha(0.18))
	case row.Hovered():
		row.Background(th.SurfaceHover)
	}
	row.Children(func() {
		arrow := ui.Icon(c, chevron()).Size(12, 12).TextColor(th.TextMuted)
		if open {
			arrow.Rotate(90)
		}
		if !dir {
			arrow.Opacity(0)
		}
		ui.Image(c, fileIcon(name, dir, open)).Size(16, 16).Margin(0, 2, 0, 0)
		ui.Text(c, name).FontSize(13).SingleLine().Shrink(1)
	}).ContextMenu(func(m *ui.Menu) { t.menu(m, p) })
	if row.Clicked() {
		switch {
		case open:
			t.state.Open.Remove(p)
		case dir:
			t.state.Open.Add(p)
		default:
			go mygo.RunOnMain(func() { w.openFile(p) })
		}
	}
}

// menu is the context menu of the file or folder at p.
func (t *folderTree) menu(m *ui.Menu, p string) {
	if m.Item(tr("在访达中显示", "Reveal in Finder")).Chosen() {
		mygo.Shell.ShowItemInFolder(p)
	}
	if m.Item(tr("拷贝路径", "Copy Path")).Chosen() {
		mygo.Clipboard.WriteText(p)
	}
}

// openFile shows the file at p in a tab of w, opening it there when no
// tab shows it. Main thread.
func (w *docWindow) openFile(p string) {
	for i, v := range w.tabs {
		if v.path == p {
			w.choose(i)
			return
		}
	}
	front = w
	go openAnyFile(p)
}

// openFolder shows the folder at root in a window of its own, or brings
// forward the one that shows it. Main thread.
func openFolder(root string) {
	for _, h := range hosts {
		if h.tree != nil && h.tree.root == root {
			h.win.Show()
			h.win.Focus()
			return
		}
	}
	w := newWindow(false, newFolderTree(root))
	w.showing = true
	hosts[w.win] = w
	front = w
	w.win.SetTitle(w.tree.name)
	w.win.SetRepresentedFile(root)
	w.tree.watch(w.win)
	syncFolderItems()
}

// folderItemsOn tells whether the View menu's items of the folder's tree
// are enabled.
var folderItemsOn bool

// syncFolderItems enables the View menu's items of the tree while the
// window in front shows a folder. It changes the menu only when that
// changes, which windows without a folder never do. Main thread.
func syncFolderItems() {
	on := front != nil && front.tree != nil
	if on == folderItemsOn {
		return
	}
	m := mygo.App.Menu()
	if m == nil {
		return
	}
	folderItemsOn = on
	for _, id := range []string{"sidebar", "tree-focus"} {
		if item := m.ItemByID(id); item != nil {
			item.SetEnabled(on)
		}
	}
}

// openFolderDialog asks for a folder to open.
func openFolderDialog() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.FocusedWindow(), Directory: true, CreateDirectories: true})
	if err != nil || len(paths) == 0 {
		return
	}
	go openAnyFile(paths[0])
}

// emptyFolderView fills a folder's window that shows no file.
func emptyFolderView(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Gap(6).Children(func() {
		ui.Text(c, tr("在左侧选择要打开的文件", "Choose a file on the left to open it")).FontSize(13).TextColor(th.TextMuted)
	})
}

// compareNames orders names as Finder does: without regard to case, and
// numbers by their value, file2 before file10.
func compareNames(a, b string) int {
	for a != "" && b != "" {
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if isDigit(ra) && isDigit(rb) {
			da, db := digits(a), digits(b)
			ta, tb := strings.TrimLeft(da, "0"), strings.TrimLeft(db, "0")
			if len(ta) != len(tb) {
				return len(ta) - len(tb)
			}
			if c := strings.Compare(ta, tb); c != 0 {
				return c
			}
			a, b = a[len(da):], b[len(db):]
			continue
		}
		la, lb := unicode.ToLower(ra), unicode.ToLower(rb)
		if la != lb {
			if la < lb {
				return -1
			}
			return 1
		}
		a, b = a[na:], b[nb:]
	}
	return len(a) - len(b)
}

func isDigit(r rune) bool { return '0' <= r && r <= '9' }

// digits returns the digits s starts with.
func digits(s string) string {
	i := 0
	for i < len(s) && isDigit(rune(s[i])) {
		i++
	}
	return s[:i]
}
