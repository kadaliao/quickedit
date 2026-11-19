package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"log"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// Print (⌘P) and Export as PDF lay the document out as HTML in a web view
// of its window, which no view shows, and WebKit pages it for the printer:
// code colored as the editor colors it, in the light theme, with the
// numbers of its lines; Markdown as its preview shows it. WebKit starts
// with the first print, not with the app.

// printJob is what a document prints: its text when the print was asked.
type printJob struct {
	title string
	text  editor.Snapshot
	lang  *editor.Language
	// rendered prints the Markdown as its preview shows it, rather than
	// its source.
	rendered bool
	// dir is the folder of the file, where the Markdown's images are; ""
	// while untitled.
	dir     string
	tabSize int
	// numbers prints the number of each line, as for code.
	numbers bool
}

// printJob returns what v prints: what it shows. Main thread.
func (v *docView) printJob() printJob {
	j := printJob{title: v.doc.State().Name, text: v.st.Buffer.Snapshot(), tabSize: v.st.TabSize}
	if !v.binary {
		j.lang = v.st.Buffer.Language()
	}
	j.rendered = v.markdown && v.mode != modeSource
	j.numbers = j.lang != nil
	if v.path != "" {
		j.dir = filepath.Dir(v.path)
	}
	if j.tabSize <= 0 && j.lang != nil {
		j.tabSize = j.lang.TabSize
	}
	if j.tabSize <= 0 {
		j.tabSize = 4
	}
	return j
}

// html returns the page the job prints. It runs no script and loads
// nothing, whatever HTML the Markdown holds: the images it shows are in
// the page.
func (j printJob) html() string {
	var b strings.Builder
	b.Grow(j.text.Len()*2 + 8<<10)
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8">`)
	b.WriteString(`<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:">`)
	b.WriteString("<title>" + html.EscapeString(j.title) + "</title><style>" + printStyle() + "</style></head>")
	if j.rendered {
		b.WriteString(`<body class="md">`)
		b.WriteString(markdownHTML(j.text.String(), j.dir))
	} else {
		j.lines(&b)
	}
	b.WriteString("</body></html>")
	return b.String()
}

// lines writes the job's text as code: a paragraph a line, which wraps
// under its own start, after the line's number. The numbers are text:
// numbered by CSS counters and placed beside their lines, 20,000 lines
// took WebKit 21 s to lay out, rather than 1.6 s.
func (j printJob) lines(b *strings.Builder) {
	n := j.text.LineCount()
	digits, gutter, class := len(strconv.Itoa(n)), 0, "lines"
	if j.numbers {
		gutter, class = digits+2, "lines numbered"
	}
	fmt.Fprintf(b, `<body class="code"><header>%s</header><div class="%s" style="--gutter:%dch">`,
		html.EscapeString(j.title), class, gutter)
	var tok editor.Tokenizer
	if j.lang != nil {
		tok = j.lang.Tokenizer
	}
	w := codeWriter{b: b, tabSize: j.tabSize}
	var (
		toks  []editor.Token
		state uint64
		num   []byte
	)
	for i := range n {
		line := j.text.Line(i)
		toks = toks[:0]
		if tok != nil {
			toks, state = tok.Tokenize(line, state, toks)
		}
		b.WriteString("<p>")
		if j.numbers {
			// Spaces that don't break: a line too long wraps after its
			// number, not between it and its text.
			num = strconv.AppendInt(num[:0], int64(i+1), 10)
			b.WriteString("<i>" + nbsp[:len(nbspChar)*(digits-len(num))])
			b.Write(num)
			b.WriteString(nbsp[:2*len(nbspChar)] + "</i>")
		} else if line == "" {
			b.WriteString("<br>") // a paragraph of a line's height
		}
		w.line(line, toks)
		b.WriteString("</p>\n")
	}
	b.WriteString("</div>")
}

const nbspChar = " "

var nbsp = strings.Repeat(nbspChar, 24)

// spaces are those of a tab, at most.
const spaces = "                "

var textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// codeWriter writes lines of code as HTML: their tokens in spans of their
// kinds' classes, and their tabs as the spaces to the next tab stop, as
// the editor shows them, which CSS's tab stops would put elsewhere after
// a number.
type codeWriter struct {
	b       *strings.Builder
	tabSize int
	col     int // of the line written so far
}

// line writes a line, whose tokens are toks.
func (w *codeWriter) line(s string, toks []editor.Token) {
	w.col = 0
	start := 0
	for _, t := range toks {
		end := min(t.End, len(s))
		if end <= start {
			continue
		}
		if t.Kind == editor.TokenText || t.Kind > editor.TokenInvalid {
			w.text(s[start:end])
		} else {
			w.b.WriteString(`<span class="k` + strconv.Itoa(int(t.Kind)) + `">`)
			w.text(s[start:end])
			w.b.WriteString("</span>")
		}
		start = end
	}
	w.text(s[start:])
}

func (w *codeWriter) text(s string) {
	size := max(1, min(w.tabSize, len(spaces)))
	for {
		i := strings.IndexByte(s, '\t')
		if i < 0 {
			break
		}
		w.plain(s[:i])
		n := size - w.col%size
		w.b.WriteString(spaces[:n])
		w.col += n
		s = s[i+1:]
	}
	w.plain(s)
}

func (w *codeWriter) plain(s string) {
	textEscaper.WriteString(w.b, s)
	w.col += utf8.RuneCountInString(s)
}

var (
	codeBlockRE = regexp.MustCompile(`<pre><code class="language-([^"]*)">([^<]*)</code></pre>`)
	imageRE     = regexp.MustCompile(`<img\b[^>]*>`)
	srcRE       = regexp.MustCompile(`\ssrc\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	altRE       = regexp.MustCompile(`\salt\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
)

// markdownHTML returns the HTML of a Markdown text: its code blocks
// colored as the preview colors them, and its images, as the preview loads
// them from the files of dir, in the page.
func markdownHTML(src, dir string) string {
	out := parser.Parse(src).HTML()
	out = codeBlockRE.ReplaceAllStringFunc(out, func(m string) string {
		sub := codeBlockRE.FindStringSubmatch(m)
		lang := editor.LanguageNamed(html.UnescapeString(sub[1]))
		if lang == nil || lang.Tokenizer == nil {
			return m
		}
		var b strings.Builder
		b.WriteString(`<pre><code class="language-` + sub[1] + `">`)
		w := codeWriter{b: &b, tabSize: 4}
		var (
			toks  []editor.Token
			state uint64
		)
		for i, line := range strings.Split(strings.TrimSuffix(html.UnescapeString(sub[2]), "\n"), "\n") {
			if i > 0 {
				b.WriteByte('\n')
			}
			toks, state = lang.Tokenizer.Tokenize(line, state, toks[:0])
			w.line(line, toks)
		}
		b.WriteString("\n</code></pre>")
		return b.String()
	})
	return imageRE.ReplaceAllStringFunc(out, func(tag string) string { return inlineImage(tag, dir) })
}

// inlineImage returns an image's tag with the image in it, as a data URL,
// or, when there is no such file, the image's description, as the preview
// shows it.
func inlineImage(tag, dir string) string {
	m := srcRE.FindStringSubmatchIndex(tag)
	if m == nil {
		return tag
	}
	ref := attrValue(tag[m[2]:m[3]])
	if strings.HasPrefix(ref, "data:") {
		return tag
	}
	if data, ok := readImage(ref, dir); ok {
		typ := mime.TypeByExtension(strings.ToLower(filepath.Ext(ref)))
		if !strings.HasPrefix(typ, "image/") {
			typ = "image/png" // WebKit reads an image's kind from its bytes
		}
		return tag[:m[2]] + `"data:` + typ + `;base64,` + base64.StdEncoding.EncodeToString(data) + `"` + tag[m[3]:]
	}
	label := ""
	if a := altRE.FindStringSubmatch(tag); a != nil {
		label = attrValue(a[1])
	}
	if label == "" {
		label = filepath.Base(ref)
	}
	return `<span class="missing-image">` + html.EscapeString(label) + `</span>`
}

// attrValue returns the value an attribute's text holds, unquoted and
// unescaped, as a URL's path with its escapes decoded.
func attrValue(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		s = s[1 : len(s)-1]
	}
	s = html.UnescapeString(s)
	if u, err := url.PathUnescape(s); err == nil {
		s = u
	}
	return s
}

// maxImage is the largest image a print holds.
const maxImage = 64 << 20

// readImage reads the image of a Markdown's ref, as the preview finds it
// (markdown.FileImages): a file, by its path or relative to dir; never
// one of the network.
func readImage(ref, dir string) ([]byte, bool) {
	if ref == "" || strings.Contains(ref, "://") {
		return nil, false
	}
	path := strings.TrimPrefix(ref, "file:")
	if !filepath.IsAbs(path) {
		if dir == "" {
			return nil, false
		}
		path = filepath.Join(dir, filepath.FromSlash(path))
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || info.Size() > maxImage {
		return nil, false
	}
	data, err := os.ReadFile(path)
	return data, err == nil
}

// cjkRanges are the characters the CJK font of a print draws: the
// ideographs, kana and their punctuation.
const cjkRanges = "U+2E80-2FFF,U+3000-30FF,U+3100-9FFF,U+F900-FAFF,U+FE30-FE4F,U+FF00-FFEF,U+20000-2FFFF"

// printStyle is the style sheet of a print, in the colors of the light
// theme, as paper is white.
var printStyle = sync.OnceValue(func() string {
	t := ui.LightTheme()
	ct := editor.DefaultTheme(t)
	mt := markdown.DefaultTheme(t)
	var b strings.Builder
	css := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	// Chinese in PingFang SC, named: the system's font, which -apple-system
	// falls back to, has a private name, which keeps MyGo from putting
	// back the characters that WebKit's PDFs name radicals for, as ⼈ for
	// 人 (mygo.Page.PrintToPDF).
	css(`@font-face{font-family:"QE CJK";font-weight:100 500;src:local("PingFangSC-Regular");unicode-range:%s}`, cjkRanges)
	css(`@font-face{font-family:"QE CJK";font-weight:600 900;src:local("PingFangSC-Semibold");unicode-range:%s}`, cjkRanges)
	css(`:root{-webkit-print-color-adjust:exact;print-color-adjust:exact}`)
	css(`html{background:#fff;color:%s}`, cssColor(t.Text))
	css(`body{margin:0;-webkit-text-size-adjust:none}`)
	css(`code,pre,.code{font-family:"QE CJK",ui-monospace,Menlo,monospace}`)

	// Code: a paragraph a line, its number in front of it, its wrapped
	// rows under its text (printJob.lines).
	css(`.code{font-size:8.5pt;line-height:1.5}`)
	css(`.code header{font:600 9pt -apple-system,"QE CJK",sans-serif;color:%s;padding-bottom:6pt;margin-bottom:8pt;border-bottom:0.5pt solid %s}`,
		cssColor(t.TextMuted), cssColor(t.Border))
	css(`.lines p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere;break-inside:avoid}`)
	css(`.numbered p{padding-left:var(--gutter);text-indent:calc(-1*var(--gutter))}`)
	css(`.numbered i{font-style:normal;color:%s}`, cssColor(ct.LineNumber))
	for k := editor.TokenKeyword; k <= editor.TokenInvalid; k++ {
		st := ct.Tokens[k]
		css(`.k%d{color:%s%s%s}`, k, cssColor(ct.TokenColor(k)), cond(st.Bold, ";font-weight:600"), cond(st.Italic, ";font-style:italic"))
	}

	// Markdown, as the preview shows it (markdown.DefaultTheme).
	css(`.md{font:10.5pt/%g -apple-system,"QE CJK",sans-serif;color:%s}`, mt.LineHeight, cssColor(mt.Text))
	css(`.md>:first-child{margin-top:0}`)
	css(`.md p,.md ul,.md ol,.md pre,.md table,.md blockquote,.md .markdown-alert,.md .math.display{margin:0 0 %gem}`, mt.BlockSpacing)
	css(`.md ul,.md ol{padding-left:%gem}`, mt.ListIndent)
	css(`.md li+li,.md li>ul,.md li>ol{margin-top:%gem}`, mt.ItemSpacing)
	css(`.md li>ul,.md li>ol{margin-bottom:0}`)
	css(`.md li:has(>input[type=checkbox]){list-style:none}`)
	css(`.md input[type=checkbox]{margin:0 0.45em 0 -1.35em;vertical-align:-0.1em;accent-color:%s}`, cssColor(mt.Checkbox))
	for i, h := range mt.Headings {
		css(`.md h%d{font-size:%gem;font-weight:%d;color:%s;margin:%gem 0 %gem;line-height:1.25;break-after:avoid}`,
			i+1, h.Size, h.Weight, cssColor(h.Color), h.Top/h.Size, h.Bottom/h.Size)
	}
	css(`.md strong{font-weight:%d;color:%s}`, mt.StrongWeight, cssColor(mt.Strong))
	css(`.md a{color:%s;text-decoration:none}`, cssColor(mt.Link))
	css(`.md code{font-size:%gem;color:%s;background:%s;padding:0.1em 0.3em;border-radius:4px}`,
		mt.CodeSize, cssColor(mt.Code), cssColor(mt.CodeBackground))
	css(`.md pre{font-size:%gem;line-height:%g;background:%s;border:0.5pt solid %s;border-radius:%gpx;padding:%gem;white-space:pre-wrap;overflow-wrap:anywhere;tab-size:4}`,
		mt.CodeSize, mt.CodeLineHeight, cssColor(mt.CodeBlockBackground), cssColor(mt.CodeBlockBorder), mt.CodeBlockRadius, mt.CodeBlockPadding)
	css(`.md pre code{font-size:1em;background:none;padding:0;border-radius:0}`)
	css(`.md blockquote{margin-left:0;margin-right:0;padding-left:1em;border-left:%gpx solid %s;color:%s}`,
		mt.QuoteBarWidth, cssColor(mt.QuoteBar), cssColor(mt.QuoteText))
	for i, name := range []string{"note", "tip", "important", "warning", "caution"} {
		css(`.md .markdown-alert-%s{border-left:%gpx solid %s;padding-left:1em}`, name, mt.QuoteBarWidth, cssColor(mt.Alerts[i]))
		css(`.md .markdown-alert-%s .markdown-alert-title{color:%s;font-weight:600}`, name, cssColor(mt.Alerts[i]))
	}
	css(`.md .markdown-alert>:last-child,.md blockquote>:last-child{margin-bottom:0}`)
	css(`.md hr{border:0;border-top:1px solid %s;margin:1.5em 0}`, cssColor(mt.Rule))
	css(`.md table{border-collapse:collapse}`)
	css(`.md th,.md td{border:1px solid %s;padding:%gem %gem}`, cssColor(mt.TableBorder), mt.TableCellPadding[1], mt.TableCellPadding[0])
	css(`.md th{background:%s;font-weight:600}`, cssColor(mt.TableHeader))
	css(`.md tr{break-inside:avoid}`)
	css(`.md img{max-width:100%%;border-radius:0.3em;break-inside:avoid}`)
	css(`.md .missing-image{display:inline-block;padding:0.2em 0.6em;border:1px dashed %s;border-radius:0.3em;color:%s;font-size:0.9em}`,
		cssColor(t.Border), cssColor(mt.Muted))
	css(`.md mark{background:%s;color:inherit}`, cssColor(mt.Mark))
	css(`.md .math{font-family:"QE CJK",ui-monospace,Menlo,monospace;font-size:%gem}`, mt.CodeSize)
	css(`.md .math.display{display:block;text-align:center;white-space:pre-wrap}`)
	css(`.md .footnotes{margin-top:2em;padding-top:0.75em;border-top:1px solid %s;font-size:0.9em;color:%s}`, cssColor(mt.Rule), cssColor(mt.Muted))
	css(`.md .footnote-backref{display:none}`)
	return b.String()
})

func cond(ok bool, s string) string {
	if ok {
		return s
	}
	return ""
}

func cssColor(c ui.Color) string {
	c = c.SRGB()
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("rgba(%d,%d,%d,%.3g)", c.R, c.G, c.B, float64(c.A)/255)
}

// printMargins are the margins of an exported PDF's pages, in inches.
var printMargins = mygo.Margins{Top: 0.6, Right: 0.6, Bottom: 0.6, Left: 0.6}

// letterRegions use Letter paper rather than A4.
var letterRegions = map[string]bool{"US": true, "CA": true, "MX": true, "PH": true, "CL": true, "CO": true, "VE": true, "PR": true}

// paperSize returns the paper of the user's region, as its formats name it.
func paperSize(locale string) mygo.PageSize {
	if letterRegions[regionOf(locale)] {
		return mygo.PageLetter
	}
	return mygo.PageA4
}

// regionOf returns the region of a BCP 47 tag, as "US" of "en-US", or "".
func regionOf(tag string) string {
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	for _, p := range parts[min(1, len(parts)):] {
		if len(p) == 1 {
			break // the extensions
		}
		if len(p) == 2 {
			return strings.ToUpper(p)
		}
	}
	return ""
}

// pdfName returns the name of the PDF of a document: its own, with .pdf
// for its extension.
func pdfName(name string) string {
	if base := strings.TrimSuffix(name, filepath.Ext(name)); base != "" {
		name = base
	}
	return name + ".pdf"
}

// printing tells that a print or an export lays its document out: another
// waits until it is done.
var printing atomic.Bool

// layOutTimeout is how long WebKit may take to load a print.
const layOutTimeout = 2 * time.Minute

// layOut loads doc into a web view of win, which no view shows, and
// returns it once loaded. Not on the main thread.
func layOut(win *mygo.Window, doc string) (*mygo.WebView, error) {
	// Unthrottled: WebKit takes a page that shows nowhere for one in the
	// background, whose processes then run last, and a busy Mac took
	// ten times as long to print it.
	wv, err := win.NewWebView(mygo.WebViewOptions{Page: mygo.PageOptions{Untrusted: true, BackgroundThrottling: mygo.ThrottlingOff}})
	if err != nil {
		return nil, err
	}
	pg := wv.Page()
	done := make(chan error, 1)
	offLoad := pg.OnDidFinishLoad(func() {
		select {
		case done <- nil:
		default:
		}
	})
	offFail := pg.OnDidFailLoad(func(e *mygo.LoadError) {
		select {
		case done <- e:
		default:
		}
	})
	defer offLoad()
	defer offFail()
	pg.LoadHTML(doc, "")
	select {
	case err = <-done:
	case <-time.After(layOutTimeout):
		err = errors.New(tr("排版超时", "laying the pages out took too long"))
	}
	if err != nil {
		wv.Destroy()
		return nil, err
	}
	return wv, nil
}

// exportPDF asks where to save v's document as a PDF, and saves it there.
// Main thread.
func exportPDF(v *docView) {
	if !printing.CompareAndSwap(false, true) {
		return
	}
	job, win := v.printJob(), v.win
	dir := job.dir
	if dir == "" {
		dir, _ = mygo.App.Path(mygo.PathDocuments)
	}
	paper := paperSize(mygo.App.Region().Locale)
	go func() {
		defer printing.Store(false)
		if !confirmLong(win, job, tr("导出", "Export")) {
			return
		}
		// The page, made while the user chooses where it goes.
		doc := make(chan string, 1)
		go func() { doc <- job.html() }()
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: win, DefaultPath: filepath.Join(dir, pdfName(job.title)), CreateDirectories: true,
			Filters: []mygo.FileFilter{{Name: "PDF", Extensions: []string{"pdf"}}},
		})
		if err != nil || path == "" {
			return
		}
		v.setBusy(tr("正在导出 PDF…", "Exporting PDF…"))
		defer v.setBusy("")
		if err := writePDF(win, <-doc, paper, path); err != nil {
			log.Printf("export %s: %v", path, err)
			printFailed(win, tr("无法导出 PDF", "Couldn't Export as PDF"), err)
		}
	}()
}

// longPrint is the most lines of code a print lays out without asking:
// some 350 pages, which take WebKit several seconds.
const longPrint = 20_000

// linesPerPage is about how many lines of code a page holds.
const linesPerPage = 56

// confirmLong reports whether to go on with a print: what the user
// answers when its lines are so many that it takes long. Not on the main
// thread.
func confirmLong(win *mygo.Window, job printJob, action string) bool {
	n := job.text.LineCount()
	if job.rendered || n <= longPrint {
		return true
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning,
		Message: fmt.Sprintf(tr("%s 有 %s 行，大约 %s 页", "%s has %s lines, about %s pages"), job.title, grouped(n), grouped(n/linesPerPage)),
		Detail:  tr("排版要花不少时间，中途不能取消。", "Laying them out takes a while, and can't be canceled."),
		Buttons: []string{action, tr("取消", "Cancel")}, CancelButton: 1})
	return err == nil && res.Button == 0
}

// grouped writes n with its digits in groups of three, as 108,000.
func grouped(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// setBusy shows what a print does in v's status bar, or nothing for "".
// Any goroutine.
func (v *docView) setBusy(msg string) { v.win.Update(func() { v.busy = msg }) }

// writePDF lays doc out on pages of paper, and writes them to path as a
// PDF. Not on the main thread.
func writePDF(win *mygo.Window, doc string, paper mygo.PageSize, path string) error {
	wv, err := layOut(win, doc)
	if err != nil {
		return err
	}
	defer wv.Destroy()
	pdf, err := wv.Page().PrintToPDF(mygo.PDFOptions{PageSize: paper, Margins: &printMargins})
	if err != nil {
		return err
	}
	return os.WriteFile(path, pdf, 0o644)
}

// printDocument shows the system's print panel over v's window, for its
// document. Main thread.
func printDocument(v *docView) {
	h := hosts[v.win]
	if h == nil || !printing.CompareAndSwap(false, true) {
		return
	}
	job, win := v.printJob(), v.win
	go func() {
		defer printing.Store(false)
		if !confirmLong(win, job, tr("打印", "Print")) {
			return
		}
		v.setBusy(tr("正在准备打印…", "Preparing to Print…"))
		wv, err := layOut(win, job.html())
		v.setBusy("")
		if err != nil {
			log.Printf("print: %v", err)
			printFailed(win, tr("无法打印", "Couldn't Print"), err)
			return
		}
		// The panel shows the web view's pages until it closes, which
		// nothing tells: the web view stays until the next print, or until
		// the window closes.
		mygo.RunOnMain(func() {
			if h.printed != nil {
				h.printed.Destroy()
			}
			h.printed = wv
		})
		wv.Page().Print()
	}()
}

func printFailed(win *mygo.Window, msg string, err error) {
	mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Message: msg, Detail: err.Error()})
}
