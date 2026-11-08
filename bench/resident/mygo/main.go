// Command resident-mygo stays running after its windows close and opens each
// file the system hands it in a new window, as a document app does.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

var logf *os.File

func openFile(path string) {
	t0 := time.Now()
	b, err := os.ReadFile(path)
	if err != nil {
		log.Print(err)
		return
	}
	text := string(b)
	var content mygo.Content
	if strings.EqualFold(filepath.Ext(path), ".md") {
		opts := &markdown.Options{}
		content = ui.View(func(c *ui.Context) {
			ui.Scroll(c).Grow(1).FillWidth().Padding(24).Children(func() { markdown.View(c, text, opts) })
		})
	} else {
		buf := editor.NewBuffer(text)
		buf.SetLanguage(editor.LanguageFor(path))
		st := &editor.State{Buffer: buf}
		content = ui.View(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() { editor.View(c, st).Grow(1).FillWidth().AutoFocus() })
		})
	}
	w := mygo.NewWindow(mygo.WindowOptions{Title: filepath.Base(path), Width: 1000, Height: 700, UseContentSize: true, Content: content})
	w.OnReadyToShow(func() {
		if logf != nil {
			fmt.Fprintf(logf, "%s %.1f\n", filepath.Base(path), float64(time.Since(t0).Microseconds())/1000)
		}
	})
}

func main() {
	if p := os.Getenv("QE_LOG"); p != "" {
		logf, _ = os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	}
	mygo.App.OnWindowAllClosed(func() {}) // keep running, as Mac document apps do
	// macOS hands over the files before the app has finished launching.
	mygo.App.OnOpenFile(func(path string) { mygo.App.WhenReady(func() { openFile(path) }) })
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
