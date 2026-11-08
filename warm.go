package main

import (
	"log"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// A process draws its first text about 20 ms slower than the next, and
// its first Chinese 20 ms slower again: it loads the fonts and shapes
// their glyphs. So once the app has started, while it waits, warmText
// draws a little code and Markdown, Latin and Chinese, off screen, and
// the first file opened shows without that wait. Drawing it in the spare
// window instead made that window slower to show its document.

const warmCode = `package main // 注释：中文、标点，以及“引号”。

func main() {
	fmt.Println("Hello, 世界", 42, 3.14) // TODO: 完成
}
`

const warmMarkdown = "# 标题 Title\n\n这是一段**中文**和 *English* 混排的文字，含 `code` 与[链接](https://example.com)。\n\n" +
	"- 列表 item\n- [x] 任务\n\n| 表头 | Head |\n|---|---|\n| 内容 | cell |\n\n> 引用 quote\n\n```go\nfunc f() {}\n```\n"

// warmText draws warmCode and warmMarkdown off screen. Main thread.
func warmText() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("warm text: %v", r)
		}
	}()
	st := &editor.State{Buffer: editor.NewBuffer(warmCode), Font: ui.Font{Size: settings.fontSize}}
	st.Buffer.SetLanguage(editor.LanguageFor("main.go"))
	var md markdown.Options
	tt := ui.NewTester(func(c *ui.Context) {
		mdTheme := markdown.DefaultTheme(c.Theme())
		mdTheme.FontSize = settings.fontSize + 2
		md.Theme = mdTheme
		ui.Row(c).Fill().Children(func() {
			editor.View(c, st).Grow(1).Basis(0)
			ui.Column(c).Grow(1).Basis(0).Children(func() { markdown.View(c, warmMarkdown, &md) })
		})
	}, 640, 360)
	tt.SetScale(2)
	tt.Frame()
}
