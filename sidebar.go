package main

import (
	"cmp"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// A window opened on a folder (File > Open Folder…, or qe with a folder)
// lists the folder's files in a sidebar at its left, and a click opens one
// as a tab. The sidebar belongs to the window, not to a document, as the
// tabs of a window may come from several folders. It lists one level, as
// the folder is read when it opens and when the sidebar is refreshed, off
// the main thread, and never watched: opening files takes no more than it
// did, and a window without a folder draws as it did.

// sidebarWidth is the width of the sidebar, in DIPs.
const sidebarWidth = 220

// folderEntry is a file or a folder that a sidebar lists.
type folderEntry struct {
	name string
	dir  bool
}

// listFolder returns what the sidebar lists of dir: its folders, then its
// files, each by name, without the hidden ones, as .git, nor those that
// can't open, as sockets and broken links.
func listFolder(dir string) ([]folderEntry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]folderEntry, 0, len(des))
	for _, d := range des {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		t := d.Type()
		if t&fs.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			t = info.Mode().Type()
		}
		if t.IsDir() || t.IsRegular() {
			entries = append(entries, folderEntry{name: name, dir: t.IsDir()})
		}
	}
	slices.SortFunc(entries, compareEntries)
	return entries, nil
}

// compareEntries orders folders before files, and each by name, whatever
// its case, as Finder does.
func compareEntries(a, b folderEntry) int {
	if a.dir != b.dir {
		if a.dir {
			return -1
		}
		return 1
	}
	if c := cmp.Compare(strings.ToLower(a.name), strings.ToLower(b.name)); c != 0 {
		return c
	}
	return cmp.Compare(a.name, b.name)
}

// openFolder lists dir in the sidebar of the window in front, or of a new
// window when no window can take it. Main thread.
func openFolder(dir string) {
	w := front
	if w == nil || !w.usable() {
		w = spare
		spare = nil
		if w == nil {
			w = newWindow(false)
		}
		w.showing = true
		hosts[w.win] = w
		front = w
		time.AfterFunc(400*time.Millisecond, func() { mygo.RunOnMain(prepareSpare) })
	}
	w.setFolder(filepath.Clean(dir))
	go mygo.RunOnMain(w.win.Show)
}

// openFolderDialog asks for a folder to list in the sidebar. Main thread.
func openFolderDialog() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.FocusedWindow(), Directory: true})
	if err != nil || len(paths) == 0 {
		return
	}
	openFolder(paths[0])
}

// setFolder makes dir the folder w lists, and shows the sidebar. Main
// thread.
func (w *docWindow) setFolder(dir string) {
	w.folder = dir
	w.files, w.folderErr = nil, nil
	w.showSidebar = true
	if w.current() == nil {
		w.win.SetTitle(filepath.Base(dir))
	}
	w.refresh()
	syncSidebarItem()
	w.win.Update(func() {})
}

// refresh lists w's folder again, off the main thread. Main thread.
func (w *docWindow) refresh() {
	dir := w.folder
	go func() {
		files, err := listFolder(dir)
		mygo.RunOnMain(func() {
			// Another folder since, or the window closed.
			if w.folder != dir || w.win.IsDestroyed() {
				return
			}
			w.files, w.folderErr = files, err
			w.win.Update(func() {})
		})
	}()
}

// toggleSidebar shows or hides the sidebar of a window with a folder.
// Main thread.
func (w *docWindow) toggleSidebar() {
	if w.folder != "" {
		w.showSidebar = !w.showSidebar
		w.win.Update(func() {})
	}
}

// sidebarItemOn tells whether the View menu's sidebar item is enabled.
var sidebarItemOn bool

// syncSidebarItem enables the View menu's sidebar item while the window in
// front has a folder. It changes the menu only when that changes, which a
// window without a folder never does. Main thread.
func syncSidebarItem() {
	on := front != nil && front.folder != ""
	if on == sidebarItemOn {
		return
	}
	m := mygo.App.Menu()
	if m == nil {
		return
	}
	if item := m.ItemByID("sidebar"); item != nil {
		sidebarItemOn = on
		item.SetEnabled(on)
	}
}

// sidebar shows the name of w's folder, with buttons to go up to the folder
// it is in and to list it again, and its entries: a click on a file opens
// it, and on a folder lists that one instead. The current document's file
// is highlighted.
func (w *docWindow) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c.Key("sidebar")).Width(sidebarWidth).Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Height(32).Padding(0, 6, 0, 12).Gap(2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, filepath.Base(w.folder)).FontSize(12).Bold().SingleLine().Grow(1).Shrink(1).Tooltip(w.folder)
			parent := filepath.Dir(w.folder)
			if parent != w.folder && sidebarButton(c, "up", "↑", tr("上一级文件夹", "Enclosing Folder")) {
				go mygo.RunOnMain(func() { w.setFolder(parent) })
			}
			if sidebarButton(c, "refresh", "↻", tr("刷新", "Refresh")) {
				go mygo.RunOnMain(w.refresh)
			}
		})
		ui.Divider(c)
		switch {
		case w.folderErr != nil:
			ui.Text(c, tr("无法读取这个文件夹。", "Can't read this folder.")).FontSize(12).TextColor(t.TextMuted).Padding(8, 12)
		case w.files != nil && len(w.files) == 0:
			ui.Text(c, tr("文件夹是空的。", "The folder is empty.")).FontSize(12).TextColor(t.TextMuted).Padding(8, 12)
		}
		cur := ""
		if v := w.current(); v != nil && filepath.Dir(v.path) == w.folder {
			cur = filepath.Base(v.path)
		}
		// Keyed by the folder: another one shows from its top.
		ui.List(c.Key(w.folder), nil, len(w.files), func(i int) {
			f := w.files[i]
			name := f.name
			if f.dir {
				name += "/"
			}
			row := ui.Row(c).FillWidth().Height(24).Padding(0, 8).Radius(4).AlignItems(ui.Center)
			if !f.dir && f.name == cur {
				row.Background(t.SurfacePressed)
			} else if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.Children(func() {
				ui.Text(c, name).FontSize(12).SingleLine().Grow(1).Shrink(1)
			})
			if row.Clicked() {
				p := filepath.Join(w.folder, f.name)
				if f.dir {
					go mygo.RunOnMain(func() { w.setFolder(p) })
				} else {
					go openAnyFile(p)
				}
			}
		}).Grow(1).FillWidth().Padding(4, 6)
	})
	ui.Column(c).Width(1).Background(t.Border)
}

// sidebarButton shows a small button of the sidebar's header, and reports
// whether it was clicked.
func sidebarButton(c *ui.Context, key, glyph, tip string) bool {
	t := c.Theme()
	b := ui.Row(c.Key(key)).Width(22).Height(22).Radius(4).AlignItems(ui.Center).Justify(ui.Center).Tooltip(tip)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, glyph).FontSize(13).TextColor(t.TextMuted)
	})
	return b.Clicked()
}
