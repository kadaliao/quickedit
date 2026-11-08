package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// mode is what the window of a Markdown file shows.
type mode int

const (
	modeSource  mode = iota // the text, in the editor
	modePreview             // the rendered Markdown
	modeSplit               // both, side by side
)

// docView is the window of a document.
type docView struct {
	doc *mygo.Document[content]
	win *mygo.Window
	st  *editor.State

	enc       textEncoding
	binary    bool
	truncated bool

	path     string // the file's, "" while untitled
	markdown bool
	mode     mode

	// setVersion is the buffer's version the document last got.
	setVersion uint64

	// The Markdown preview's source, kept by the buffer's version.
	md        markdown.Options
	mdText    string
	mdVersion uint64
	mdDir     string
}

var views = map[*mygo.Window]*docView{}

func viewOf(w *mygo.Window) *docView {
	if w == nil {
		return nil
	}
	return views[w]
}

func newDocWindow(doc *mygo.Document[content]) *mygo.Window {
	c := doc.Content()
	v := &docView{doc: doc, enc: c.enc, binary: c.binary, truncated: c.truncated}
	if c.buf == nil { // an untitled document
		c.buf = editor.NewBuffer("")
	}
	v.st = &editor.State{Buffer: c.buf, Bindings: appBindings}
	v.setVersion = c.buf.Version()
	v.applyPath(doc.State().Path)
	if v.markdown {
		v.mode = modePreview
		if doc.State().Path == "" || c.snap.Len() == 0 {
			v.mode = modeSource // nothing to preview yet: write
		}
	}
	v.applySettings()

	v.win = mygo.NewWindow(mygo.WindowOptions{
		Width: 960, Height: 720, MinWidth: 360, MinHeight: 240,
		StateKey: "document",
		Content:  ui.View(v.view),
	})
	views[v.win] = v
	v.win.OnClosed(func() { delete(views, v.win) })

	// Reverted, or changed by another app: show the file's text. Setting
	// it as an edit keeps the undo history, as Sublime and VS Code do.
	doc.OnReload(func() {
		v.win.Update(func() {
			nc := doc.Content()
			v.enc, v.binary, v.truncated = nc.enc, nc.binary, nc.truncated
			if nc.buf != v.st.Buffer {
				v.st.Buffer.Set(nc.snap.String())
			}
			v.syncDocument()
		})
	})
	// Saved as, renamed or moved: the name may tell another language.
	doc.OnStateChange(func(s mygo.DocumentState) {
		v.win.Update(func() {
			if s.Path != v.path {
				v.applyPath(s.Path)
				v.applySettings()
			}
		})
	})
	return v.win
}

// applyPath sets what the file's name tells: its language, and whether it
// is Markdown.
func (v *docView) applyPath(path string) {
	v.path = path
	name := filepath.Base(path)
	v.markdown = isMarkdown(name)
	if v.binary {
		v.st.Buffer.SetLanguage(nil)
		return
	}
	lang := editor.LanguageFor(name)
	if lang == nil && path != "" {
		lang = languageOfShebang(v.st.Buffer.Snapshot())
	}
	v.st.Buffer.SetLanguage(lang)
	v.mdDir = filepath.Dir(path)
	if path != "" {
		v.md.LoadImage = markdown.FileImages(v.mdDir)
	}
}

// applySettings sets the editor's options for the kind of file and the
// app's settings.
func (v *docView) applySettings() {
	v.st.Font = ui.Font{Size: settings.fontSize}
	v.st.ReadOnly = v.binary
	v.st.SoftWrap = settings.wrap(v.prose())
	if v.binary {
		v.st.LineNumbers = editor.LineNumbersOff
		v.st.SoftWrap = false
	} else {
		v.st.LineNumbers = editor.LineNumbersOn
	}
}

// prose reports whether the text is prose rather than code, which wraps
// by default, as in TextEdit.
func (v *docView) prose() bool {
	return v.markdown || v.st.Buffer.Language() == nil
}

func isMarkdown(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".mkdn", ".mdx":
		return true
	}
	return false
}

// languageOfShebang finds the language of a script without an extension
// by its first line, as #!/usr/bin/env python3.
func languageOfShebang(s editor.Snapshot) *editor.Language {
	if s.LineCount() == 0 {
		return nil
	}
	line := s.Line(0)
	if !strings.HasPrefix(line, "#!") {
		return nil
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return nil
	}
	prog := filepath.Base(fields[0])
	if prog == "env" && len(fields) > 1 {
		prog = fields[len(fields)-1]
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				prog = f
				break
			}
		}
	}
	prog = strings.TrimRight(prog, "0123456789.")
	names := map[string]string{
		"sh": "Shell", "bash": "Shell", "zsh": "Shell", "dash": "Shell", "ksh": "Shell", "fish": "Shell",
		"python": "Python", "ruby": "Ruby", "node": "JavaScript", "deno": "TypeScript", "bun": "TypeScript",
		"php": "PHP", "lua": "Lua", "Rscript": "R",
	}
	if n, ok := names[prog]; ok {
		return editor.LanguageNamed(n)
	}
	return nil
}

// syncDocument gives the document the buffer's text when it changed.
func (v *docView) syncDocument() {
	buf := v.st.Buffer
	if ver := buf.Version(); ver != v.setVersion && !v.binary {
		v.setVersion = ver
		v.doc.Set(content{buf: buf, snap: buf.Snapshot(), enc: v.enc})
	}
}

func (v *docView) view(c *ui.Context) {
	v.syncDocument()
	t := c.Theme()
	state := v.doc.State()
	ui.Column(c).Fill().Children(func() {
		switch {
		case state.Conflict:
			v.banner(c, t.Warning, tr("另一个应用修改了这个文件。保存会覆盖它，还原则读入它的版本。",
				"Another app changed this file. Save to keep yours, or revert to theirs."), func() {
				if ui.Button(c.Key("revert"), tr("还原", "Revert")).Clicked() {
					go v.doc.Revert()
				}
				if ui.PrimaryButton(c.Key("save"), tr("保存", "Save Anyway")).Clicked() {
					go v.doc.SaveAnyway()
				}
			})
		case v.binary:
			msg := tr("二进制文件，以十六进制只读显示。", "Binary file, shown read-only as hex.")
			if v.truncated {
				msg = tr("二进制文件，只显示前 1 MB 的十六进制，只读。", "Binary file: the first 1 MB as hex, read-only.")
			}
			v.banner(c, t.TextMuted, msg, nil)
		case state.Deleted:
			v.banner(c, t.Warning, tr("文件已被删除，保存会重新创建它。", "The file was deleted. Saving writes it again."), nil)
		}

		ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			switch {
			case !v.markdown || v.mode == modeSource:
				v.source(c)
			case v.mode == modePreview:
				v.preview(c)
			default:
				v.source(c)
				ui.Column(c).Width(1).Background(t.Border)
				v.preview(c)
			}
		})
		v.statusBar(c, state)
	})
}

func (v *docView) banner(c *ui.Context, color ui.Color, msg string, buttons func()) {
	ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Background(color.Alpha(0.12)).Children(func() {
		ui.Text(c, msg).FontSize(12).Grow(1)
		if buttons != nil {
			buttons()
		}
	})
}

func (v *docView) source(c *ui.Context) {
	el := editor.View(c.Key("source"), v.st).Grow(1).Basis(0)
	if v.mode != modePreview {
		el.AutoFocus()
	}
}

func (v *docView) preview(c *ui.Context) {
	buf := v.st.Buffer
	if ver := buf.Version(); ver != v.mdVersion || v.mdText == "" {
		v.mdVersion = ver
		v.mdText = buf.String()
	}
	th := markdown.DefaultTheme(c.Theme())
	th.FontSize = settings.fontSize + 2
	v.md.Theme = th
	ui.Scroll(c.Key("preview")).Grow(1).Basis(0).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(28, 32).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(860).Children(func() {
				markdown.View(c, v.mdText, &v.md)
			})
		})
	})
}

func (v *docView) statusBar(c *ui.Context, state mygo.DocumentState) {
	t := c.Theme()
	muted := func(s string) { ui.Text(c, s).FontSize(11).TextColor(t.TextMuted) }
	ui.Divider(c)
	ui.Row(c).MinHeight(26).Padding(3, 12).Gap(14).AlignItems(ui.Center).Children(func() {
		if v.mode != modePreview || !v.markdown {
			p := v.st.CursorPos()
			col := utf8.RuneCountInString(v.st.Buffer.Snapshot().Line(p.Line)[:min(p.Col, len(v.st.Buffer.Snapshot().Line(p.Line)))]) + 1
			muted(fmt.Sprintf(tr("行 %d，列 %d", "Ln %d, Col %d"), p.Line+1, col))
			if sel := v.st.SelectedText(); sel != "" {
				muted(fmt.Sprintf(tr("已选 %d 字", "%d selected"), utf8.RuneCountInString(sel)))
			}
		} else {
			muted(fmt.Sprintf(tr("%d 行", "%d lines"), v.st.Buffer.Snapshot().LineCount()))
		}
		ui.Spacer(c)
		if state.ReadOnly && !v.binary {
			muted(tr("只读", "Read-only"))
		}
		if !v.binary {
			lang := tr("纯文本", "Plain Text")
			if l := v.st.Buffer.Language(); l != nil {
				lang = l.Name
			}
			muted(lang)
			muted(v.enc.String())
			if v.st.Buffer.LineEnding() == "\r\n" {
				muted("CRLF")
			} else {
				muted("LF")
			}
		}
		if v.markdown {
			sel := int(v.mode)
			if ui.Segmented(c, &sel, tr("源码", "Source"), tr("预览", "Preview"), tr("并排", "Split")).
				Label(tr("显示方式", "Show")).Changed() {
				v.setMode(mode(sel))
			}
		}
	})
}

func (v *docView) setMode(m mode) {
	if !v.markdown {
		return
	}
	v.mode = m
	v.win.Update(func() {})
}

// toggleMode switches a Markdown file between its preview and its source.
func (v *docView) toggleMode() {
	if v.mode == modePreview {
		v.setMode(modeSource)
	} else {
		v.setMode(modePreview)
	}
}
