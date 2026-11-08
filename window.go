package main

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Making a window and drawing its first frame takes macOS most of the time
// a file takes to open in a running app. So a hidden window waits for the
// next document, which only gives it its view and shows it, and another is
// made once that one has shown.

// docWindow is a window of a document, whose view it shows once it has one.
type docWindow struct {
	win  *mygo.Window
	view *docView
}

var spare *docWindow

func newWindow(hidden bool) *docWindow {
	w := &docWindow{}
	w.win = mygo.NewWindow(mygo.WindowOptions{
		Width: 960, Height: 720, MinWidth: 360, MinHeight: 240,
		StateKey: "document",
		Hidden:   hidden,
		Content: ui.View(func(c *ui.Context) {
			if w.view != nil {
				w.view.view(c)
			}
		}),
	})
	return w
}

// takeWindow returns the window of v: the spare one, shown once the
// documents have given it its title, or else a new one. Main thread.
func takeWindow(v *docView) *mygo.Window {
	w := spare
	spare = nil
	if w == nil {
		w = newWindow(false)
		w.view = v
	} else {
		w.view = v
		w.win.Update(func() {})
		go mygo.RunOnMain(w.win.Show)
	}
	// The next one, made while nothing else is going on.
	time.AfterFunc(400*time.Millisecond, func() {
		mygo.RunOnMain(func() {
			if spare == nil {
				spare = newWindow(true)
			}
		})
	})
	return w.win
}
