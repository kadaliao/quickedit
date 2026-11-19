package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
)

func codeJob(name, text string) printJob {
	lang := editor.LanguageFor(name)
	return printJob{title: name, text: editor.NewBuffer(text).Snapshot(), lang: lang, numbers: lang != nil, tabSize: 4}
}

// Code prints a line a paragraph, numbered, its tokens colored, its text
// escaped.
func TestPrintCode(t *testing.T) {
	page := codeJob("main.go", "package main\n\n// a < b && c > d\nfunc main() {}").html()
	for _, want := range []string{
		`<header>main.go</header>`,
		`<div class="lines numbered" style="--gutter:3ch">`,
		"<p><i>1\u00a0\u00a0</i><span class=\"k1\">package</span> <span class=\"k4\">main</span></p>",
		"<p><i>2\u00a0\u00a0</i></p>\n",
		`// a &lt; b &amp;&amp; c &gt; d`,
		`default-src 'none'`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page[strings.Index(page, "<body"):])
		}
	}
	if n := strings.Count(page, "<p>"); n != 4 {
		t.Errorf("%d lines, want 4", n)
	}
	// A comment's lines all are comment, as the state goes on over an
	// empty line.
	page = codeJob("a.go", "/* one\n\ntwo */ x").html()
	if !strings.Contains(page, "<p><i>3\u00a0\u00a0</i><span class=\"k11\">two */</span> <span class=\"k4\">x</span></p>") {
		t.Errorf("comment not carried over the empty line:\n%s", page[strings.Index(page, "<body"):])
	}
}

// Plain text has no numbers, nor spans.
func TestPrintPlainText(t *testing.T) {
	page := codeJob("notes.txt", "one\n<two>").html()
	if !strings.Contains(page, `<div class="lines" style="--gutter:0ch"><p>one</p>`+"\n"+`<p>&lt;two&gt;</p>`) {
		t.Errorf("plain text:\n%s", page[strings.Index(page, "<body"):])
	}
	if strings.Contains(page[strings.Index(page, "<body"):], "<span") {
		t.Error("plain text has spans")
	}
}

// Markdown prints rendered: its code colored, its images in the page, an
// image that isn't there as its description, and none from the network.
func TestPrintMarkdown(t *testing.T) {
	dir := t.TempDir()
	png := []byte("\x89PNG\r\n\x1a\nfake")
	if err := os.WriteFile(filepath.Join(dir, "my pic.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	src := "# 标题\n\n![pic](my%20pic.png) ![gone](nope.png) ![web](https://example.com/a.png)\n\n```go\nfunc f() {}\n```\n\n<img src='my pic.png'>\n"
	j := printJob{title: "doc.md", text: editor.NewBuffer(src).Snapshot(), rendered: true, dir: dir}
	page := j.html()
	body := page[strings.Index(page, "<body"):]
	for _, want := range []string{
		`<body class="md"><h1>标题</h1>`,
		`<img src="data:image/png;base64,iVBORw0KGgpmYWtl" alt="pic" />`,
		`<span class="missing-image">gone</span>`,
		`<span class="missing-image">web</span>`,
		`<pre><code class="language-go"><span class="k1">func</span> <span class="k3">f</span><span class="k13">()</span>`,
		`<img src="data:image/png;base64,iVBORw0KGgpmYWtl">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q:\n%s", want, body)
		}
	}
	// Untitled, it finds no image relative to a folder.
	j.dir = ""
	if body := j.html(); !strings.Contains(body, `<span class="missing-image">pic</span>`) {
		t.Error("an untitled document's image was found")
	}
}

// Markdown shown as its source prints its source.
func TestPrintMarkdownSource(t *testing.T) {
	page := codeJob("doc.md", "# Title").html()
	if !strings.Contains(page, `<body class="code"`) || strings.Contains(page, "<h1>") {
		t.Errorf("source printed rendered:\n%s", page[strings.Index(page, "<body"):])
	}
}

func TestPaperSize(t *testing.T) {
	for locale, want := range map[string]mygo.PageSize{
		"en-US": mygo.PageLetter, "en-US-u-fw-mon": mygo.PageLetter, "es-MX": mygo.PageLetter, "en_CA": mygo.PageLetter,
		"zh-CN": mygo.PageA4, "zh-Hans-CN": mygo.PageA4, "en-GB": mygo.PageA4, "de": mygo.PageA4, "": mygo.PageA4,
	} {
		if got := paperSize(locale); got != want {
			t.Errorf("paperSize(%q) = %v, want %v", locale, got, want)
		}
	}
}

func TestPDFName(t *testing.T) {
	for name, want := range map[string]string{
		"README.md": "README.pdf", "main.go": "main.pdf", "Makefile": "Makefile.pdf", ".bashrc": ".bashrc.pdf", "未命名": "未命名.pdf",
	} {
		if got := pdfName(name); got != want {
			t.Errorf("pdfName(%q) = %q, want %q", name, got, want)
		}
	}
}

// Tabs print as the spaces to the next stop, as the editor shows them,
// and numbers line up on the right.
func TestPrintTabsAndNumbers(t *testing.T) {
	text := "a\tb\n\tc" + strings.Repeat("\n", 9)
	j := printJob{title: "t.txt", text: editor.NewBuffer(text).Snapshot(), numbers: true, tabSize: 4}
	page := j.html()
	for _, want := range []string{
		"<p><i>\u00a01\u00a0\u00a0</i>a   b</p>",
		"<p><i>\u00a02\u00a0\u00a0</i>    c</p>",
		"<p><i>11\u00a0\u00a0</i></p>",
		`style="--gutter:4ch"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page[strings.Index(page, "<body"):])
		}
	}
}
