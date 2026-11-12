package main

import (
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// A file opened while a document window is in front opens in it, as a
// tab: showing it then takes a frame, where a new window takes macOS
// about 80 ms, as Zed and Sublime Text do. A window shows one document at
// a time; the others are detached (mygo.Document.Detach) until chosen.
//
// Making a window and drawing its first frame takes macOS most of the time
// a file takes to open in a running app. So a hidden window waits for the
// next document opened without a window in front, which only gives it its
// view and shows it, and another is made once that one has shown. It is
// empty: drawing code in it beforehand, to warm fonts and tokenizers up,
// made the first document open slower, as it then had to replace that
// view, which kept drawing frames meanwhile.

// docWindow is a window of documents, one per tab.
type docWindow struct {
	win  *mygo.Window
	tabs []*docView
	cur  int
	// showing tells that the window shows, or is about to.
	showing bool
	// tree is the folder the window shows beside its documents, or nil.
	// Such a window stays open without documents.
	tree        *folderTree
	sidebar     float32
	hideSidebar bool
}

var (
	spare *docWindow
	hosts = map[*mygo.Window]*docWindow{}
	// front is the document window last in front, where files open.
	front *docWindow
	// quitting tells that the app quits: windows then close with all
	// their documents, which the quit took care of.
	quitting bool
)

// newWindow makes a window of documents, and of the folder of tree when
// it is not nil.
func newWindow(hidden bool, tree *folderTree) *docWindow {
	w := &docWindow{tree: tree, sidebar: settings.sidebarWidth}
	opts := mygo.WindowOptions{
		Width: 960, Height: 720, MinWidth: 360, MinHeight: 240,
		StateKey: "document",
		Hidden:   hidden,
		Content:  ui.View(w.view),
	}
	if tree != nil {
		opts.Width, opts.Height, opts.MinWidth, opts.StateKey = 1180, 780, 480, "folder"
	}
	w.win = mygo.NewWindow(opts)
	w.win.OnFocus(func() {
		front = w
		syncFolderItems()
	})
	// Before any document's: a window of several tabs closes them one by
	// one, each asking to save when it must.
	w.win.OnClose(func(e *mygo.CloseEvent) {
		if len(w.tabs) > 1 && !quitting {
			e.PreventDefault()
			w.closeAll()
		}
	})
	w.win.OnClosed(func() {
		if w.tree != nil {
			w.tree.close()
			saveSettings() // the sidebar's width
		}
		delete(hosts, w.win)
		if front == w {
			front = nil
		}
	})
	return w
}

// takeWindow returns the window of v, a new document: the window in front,
// as a new tab; else the spare one, shown once the documents have given
// it its title; else a new one. Main thread.
func takeWindow(v *docView) *mygo.Window {
	if h := front; h != nil && h.usable() {
		h.add(v)
		h.win.SetTitle(v.doc.State().Name)
		go mygo.RunOnMain(h.win.Show)
		return h.win
	}
	w := spare
	spare = nil
	if w == nil {
		w = newWindow(false, nil)
		w.tabs = []*docView{v}
	} else {
		w.tabs = []*docView{v}
		w.win.Update(func() {})
		go mygo.RunOnMain(w.win.Show)
	}
	w.showing = true
	hosts[w.win] = w
	front = w
	// The next one, made while nothing else is going on.
	time.AfterFunc(400*time.Millisecond, func() { mygo.RunOnMain(prepareSpare) })
	return w.win
}

// prepareSpare makes the window the next document opens in, if there is
// none. Main thread.
func prepareSpare() {
	if spare == nil {
		spare = newWindow(true, nil)
	}
}

// usable reports whether new documents may open in w: it shows, and is
// not minimized.
func (w *docWindow) usable() bool {
	return !w.win.IsDestroyed() && (len(w.tabs) > 0 || w.tree != nil) && (w.showing || w.win.IsVisible()) && !w.win.IsMinimized()
}

func (w *docWindow) current() *docView {
	if w.cur >= 0 && w.cur < len(w.tabs) {
		return w.tabs[w.cur]
	}
	return nil
}

// add opens v in a new tab after the current one, and chooses it. The
// documents attach v's document to w once Window returns. Main thread.
func (w *docWindow) add(v *docView) {
	if cur := w.current(); cur != nil {
		cur.doc.Detach()
	}
	w.cur = min(w.cur+1, len(w.tabs))
	w.tabs = slices.Insert(w.tabs, w.cur, v)
	w.win.Update(func() {})
}

// choose shows the tab i. Main thread.
func (w *docWindow) choose(i int) {
	if i < 0 || i >= len(w.tabs) {
		return
	}
	w.cur = i
	w.tabs[i].doc.Attach(w.win)
	w.win.Update(func() {})
}

// next chooses the tab d tabs away, going round.
func (w *docWindow) next(d int) {
	if n := len(w.tabs); n > 1 {
		w.choose(((w.cur+d)%n + n) % n)
	}
}

// closeTab closes v's tab, asking to save first when it must; the last tab
// closes the window, unless it shows a folder. done, if not nil, learns
// whether it closed. Main thread.
func (w *docWindow) closeTab(v *docView, done func(bool)) {
	if len(w.tabs) == 1 && w.tree == nil {
		w.win.Close()
		return
	}
	if i := slices.Index(w.tabs, v); i >= 0 && i != w.cur {
		w.choose(i) // its question shows on this window, over its text
	}
	v.doc.CloseKeepingWindow(func(ok bool) {
		if ok {
			i := slices.Index(w.tabs, v)
			w.tabs = slices.Delete(w.tabs, i, i+1)
			if len(w.tabs) == 0 {
				// A folder's window, which now shows only the folder.
				w.cur = 0
				w.win.SetTitle(w.tree.name)
				w.win.SetRepresentedFile(w.tree.root)
				w.win.SetDocumentEdited(false)
				w.win.Update(func() {})
			} else {
				w.choose(min(i, len(w.tabs)-1))
			}
		}
		if done != nil {
			done(ok)
		}
	})
}

// closeAll closes the tabs one after the other, and then the window,
// unless one of them stays open. Main thread.
func (w *docWindow) closeAll() {
	if len(w.tabs) <= 1 {
		w.win.Close()
		return
	}
	w.closeTab(w.current(), func(ok bool) {
		if ok {
			w.closeAll()
		}
	})
}

// focusDocument brings the tab of doc to the front, for the documents
// (DocumentsOptions.Focus), as before asking to save it. Main thread.
func focusDocument(doc *mygo.Document[content]) {
	for _, w := range hosts {
		for i, v := range w.tabs {
			if v.doc == doc {
				w.choose(i)
				w.win.Show()
				return
			}
		}
	}
}

// viewOf returns the view w shows, or nil.
func viewOf(w *mygo.Window) *docView {
	if h := hosts[w]; h != nil {
		return h.current()
	}
	return nil
}

func (w *docWindow) view(c *ui.Context) {
	if w.tree == nil || w.hideSidebar {
		w.documents(c)
		return
	}
	if ui.Split(c, &w.sidebar, func() { w.tree.view(c, w) }, func() { w.documents(c) }).
		SplitReset(defaultSidebarWidth).Fill().Changed() {
		settings.sidebarWidth = w.sidebar
	}
}

// documents shows the tab chosen, under the tabs when there are several.
func (w *docWindow) documents(c *ui.Context) {
	v := w.current()
	if v == nil {
		if w.tree != nil {
			emptyFolderView(c)
		}
		return
	}
	ui.Column(c).Fill().Children(func() {
		if len(w.tabs) > 1 {
			w.tabBar(c)
		}
		// Each document's own subtree, which keeps its state.
		ui.Column(c.Key(v)).Grow(1).FillWidth().Children(func() { v.view(c) })
	})
}

// tabBar shows a tab per document: its name, and a button that closes
// it, which shows a dot while the document has unsaved changes.
func (w *docWindow) tabBar(c *ui.Context) {
	t := c.Theme()
	bar := t.Surface
	ui.Row(c).FillWidth().Height(32).Background(bar).AlignItems(ui.Stretch).Clip().Children(func() {
		for i, v := range w.tabs {
			cur := i == w.cur
			s := v.doc.State()
			tab := ui.Row(c.Key(v)).Padding(0, 6, 0, 12).Gap(4).AlignItems(ui.Center).MinWidth(64).MaxWidth(220).Shrink(1)
			if cur {
				tab.Background(t.Background)
			}
			hover := tab.Hovered()
			tab.Tooltip(s.Path).Children(func() {
				name := ui.Text(c, s.Name).FontSize(12).SingleLine().Grow(1).Shrink(1)
				if !cur {
					name.TextColor(t.TextMuted)
				}
				mark := "×"
				if s.Edited && !hover {
					mark = "●"
				}
				x := ui.Row(c.Key("close")).Width(18).Height(18).Radius(4).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					ui.Text(c, mark).FontSize(12).TextColor(t.TextMuted)
				})
				if x.Hovered() {
					x.Background(t.SurfaceHover)
				}
				if !cur && !hover && !s.Edited {
					x.Opacity(0)
				}
				if x.Clicked() {
					go mygo.RunOnMain(func() { w.closeTab(v, nil) })
				} else if tab.Clicked() && !cur {
					go mygo.RunOnMain(func() { w.choose(slices.Index(w.tabs, v)) })
				}
			})
			ui.Column(c).Width(1).Background(t.Border)
		}
	})
	ui.Divider(c)
}
