// Command bench-mygo opens a file the way QuickEdit would: a code editor
// with highlighting, or rendered Markdown for .md files. See ../README.md.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: bench-mygo <file>")
	}
	path := os.Args[1]
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	text := string(b)

	var content mygo.Content
	if strings.EqualFold(filepath.Ext(path), ".md") {
		opts := &markdown.Options{}
		content = ui.View(func(c *ui.Context) {
			ui.Scroll(c).Grow(1).FillWidth().Padding(24).Children(func() {
				markdown.View(c, text, opts)
			})
		})
	} else {
		buf := editor.NewBuffer(text)
		buf.SetLanguage(editor.LanguageFor(path))
		st := &editor.State{Buffer: buf}
		content = ui.View(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				editor.View(c, st).Grow(1).FillWidth().AutoFocus()
			})
		})
	}

	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title: filepath.Base(path), Width: 1000, Height: 700,
			UseContentSize: true, Content: content,
		})
		// Called once the first frame is drawn. The editor tokenizes the
		// lines it paints in that frame, so highlighting is in it too.
		w.OnReadyToShow(func() {
			fmt.Println("ready")
			fmt.Println("highlighted")
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
