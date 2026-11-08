# QuickEdit 测试文档

这是一份**中英混排**的 Markdown，用来测试渲染速度。它包含标题、列表、表格、代码块、引用和任务列表。**结论。**后文紧跟中文，检查加粗是否正确。

## 功能清单

- [x] 语法高亮
- [x] Markdown 渲染
- [ ] 分屏预览
- [ ] 插件系统

| 框架 | 语言 | 渲染方式 | 备注 |
|---|---|---|---|
| MyGo | Go | GPU 自绘 | 纯 Go，无 cgo |
| GPUI | Rust | GPU 自绘 | Zed 编辑器使用 |
| iced | Rust | wgpu | Elm 架构 |
| egui | Rust | 即时模式 | 每帧重建 |

> 引用：快速打开、快速查看、快速编辑。

```rust
fn main() {
    let path = std::env::args().nth(1).expect("文件路径");
    let text = std::fs::read_to_string(&path).unwrap();
    println!("{} 行", text.lines().count());
}
```

```go
func main() {
	b, _ := os.ReadFile(os.Args[1])
	fmt.Println(len(b), "字节")
}
```


<img width="800" height="360" alt="MyGo!!!!!" src="https://github.com/user-attachments/assets/c3eee9bc-5f22-4032-95b6-776e8381910a" />
</p>

**Desktop apps in Go, with native UI or a web frontend.**

Sponsored by:


  <a href="https://openuptime.app/?utm_source=github&utm_medium=sponsor&utm_campaign=mygo">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://openuptime.app/sponsor/banner-dark.png">
      <img width="600" alt="OpenUptime: open-source uptime monitoring and status pages" src="https://openuptime.app/sponsor/banner-light.png">
    </picture>
  </a>
</p>

---

Each MyGo window shows one of two kinds of interface, and one app, or one
window, can mix them:

- **A web page**, in the webview the OS already has: WKWebView on macOS,
  WebKitGTK on Linux, WebView2 on Windows. Build the frontend with any web
  tools; it calls your Go code through a TypeScript client generated from it.
- **Native UI**, written in Go alone with package `ui` and
  drawn by MyGo itself on the GPU: no HTML, no JavaScript and no webview, so
  the window opens at once and takes little memory.

Either way, an app is a single Go binary of a few megabytes, focused on low
memory and CPU use.

- **Pure Go, no cgo**: build for every platform from any machine.
- **Typed IPC** for pages: bind Go services, stream values through channels
  and send typed events; the TypeScript client is generated from your Go
  code.
- **A Go UI toolkit** for native UI: flexbox and grid layout, widgets, text
  editing with input methods, virtualized lists, SVG icons, animations,
  screen reader support, and views you test without a window; Liquid Glass
  on every platform with the glass plugin; web pages inside native UI, under
  the menus and dialogs it shows over them.
- **Desktop APIs** for both: windows, menus, tray, dialogs, notifications,
  global shortcuts, deep links, file associations, recent documents, Now
  Playing and the media keys, the system accent color and accessibility
  settings, watching files, and more.
- **Ready to ship**: app bundles and disk images, Windows installers, Debian
  packages, PKGBUILDs for Arch Linux and a Linux install script, code signing, notarization, and signed
  auto-updates with delta updates and an update window in the manner of
  Sparkle.

A window with a web page, whose frontend calls Go:

```go
type Greeter struct{}

// Greet returns a greeting.
func (Greeter) Greet(name string) string { return "Hello, " + name }

func main() {
	mygo.Bind(Greeter{})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Hello", URL: "/"})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

```ts
import { Greeter } from "./mygo"; // generated

document.body.textContent = await Greeter.greet("Ada");
```

A window with native UI, whose view is a Go function of the app's state:

```go
type counter struct{ n int }

func (s *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, fmt.Sprint(s.n)).FontSize(40).Bold()
		if ui.PrimaryButton(c, "Increment").Clicked() {
			s.n++
		}
	})
}

func main() {
	s := &counter{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Counter",
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Getting started

With [Go](https://go.dev/dl/) 1.27.3+ and [Bun](https://bun.sh), create an app
with a web frontend:

```sh
bunx mygo-cli init my-app      # or: npx mygo-cli init my-app
cd my-app
bun run dev
```

An app of native UI needs Go alone:

```sh
go run github.com/egoist/mygo/cmd/mygo@latest init -template native my-app
cd my-app
go tool mygo dev
```

Read the [documentation](docs/README.md).

## License

MIT

# Code editor


Package `ui/editor` is a code editor and a [diff viewer](diff-viewer.md)
for native UI, designed after Zed's: syntax highlighting, several carets
and selections, soft wraps, folding, finding and replacing, completions,
hovers, diagnostics and inlay hints from the app or a language server,
and the commands and keys of code editors. It lays out and paints only the
lines in view, so a file of a million lines scrolls and edits as one of
ten.

```go
import "github.com/egoist/mygo/ui/editor"

// Kept by the app, as its other state.
buf := editor.NewBuffer(source)
buf.SetLanguage(editor.LanguageFor("main.go"))
app.code = &editor.State{Buffer: buf, WrapGuides: []int{100}}

// In the view:
editor.View(c, app.code).Grow(1).FillWidth()
```

A `State` is the editor's state, kept across frames as a text input's
string is: its buffer, its options, and what the editor keeps of its own,
as its selections, folds and scroll position. Its options are fields set
before the editor is built; their zero values are the defaults, those of
Zed. `View` builds the editor of a state, which is a focusable element as
any other: give it a size, and `AutoFocus` to type in it at once.

The [code editor example](../../examples/code-editor) edits files of
several languages, with completions of the words of the file, hovers,
diagnostics of TODO comments, the changes in the gutter and their diff.

## Buffers

A `Buffer` is the text an editor edits, with its undo history and its
language. Several editors, and both sides of a diff, can show one buffer:
an edit in one shows in the others, their carets moving with the text
around them, and undo takes back the buffer's last edit, whichever editor
made it.

```go
buf.Edit(editor.Edit{Start: 0, End: 0, Text: "// Generated.\n"})
buf.Transact(func() { // one step of undo
	buf.Edit(editor.Edit{Start: a, End: b, Text: name})
	buf.Edit(editor.Edit{Start: c, End: d, Text: name})
})
if buf.Modified() {
	_, err := buf.WriteTo(file)
	if err == nil {
		buf.MarkSaved()
	}
}
```

Offsets count bytes of UTF-8, and a `Pos` is a line and the bytes into it,
both from 0: `Pos` and `Offset` convert between them, and `Line`, `Slice`
and `String` read the text. `Set` replaces the text as an edit, which undo
takes back, and `Reset` as a new document, without history.

Lines end in `\n`, as in Zed: a text with `\r\n` line endings is kept with
`\n` ones, and `LineEnding` says which it had, which `WriteTo` writes, so
that a file saves as it was; `SetLineEnding` changes it.

The text is a persistent tree of chunks, as Zed's rope: an edit copies a
few kilobytes and a path of the tree, whatever the size of the document,
and each version shares the rest with the one before. Undo keeps versions
rather than copies, and a `Snapshot` is a version that stays as it is
while the buffer changes, which other goroutines read without locks.
Buffers' methods are safe from any goroutine; change them with
`Window.Update` so the window shows the change.

## Editing

The editor edits as Zed does, with the commands bound to the same keys
(Control replaces Command on Linux and Windows, as in Visual Studio Code):

- **Several carets.** Cmd+D selects the next occurrence of the selection,
  or the word at the caret; Cmd+Shift+L all of them; Cmd+Alt+Up and Down
  add a caret on the line above or below; Alt+click adds a caret, and
  Alt+Shift+drag selects a column. Typing, deleting and every command act
  at each caret, and Escape goes back to one. Cmd+U undoes a change of the
  selections.
- **Selections by structure.** Ctrl+Shift+Right selects the word, then
  the string or brackets around it, and on out; Ctrl+Shift+Left goes back.
- **Lines.** Alt+Up and Down move lines, Alt+Shift+Up and Down duplicate
  them, Cmd+Shift+K deletes them, Ctrl+J joins them, Cmd+/ comments them
  out, Cmd+] and Cmd+[ indent them; Cmd+Enter opens a line below.
- **Typing.** Brackets and quotes close as they are typed, and wrap the
  selection; Enter keeps the indentation, indents after an opening bracket
  and between a pair, and continues line comments; a closing bracket, or
  a word such as `end`, outdents its line as it is typed. Undo takes back
  typing a burst at a time.
- **Folding.** The gutter's markers fold the lines indented under a line,
  up to its closing bracket, or a run of imports, with the blank lines
  between them; Cmd+Alt+[ and ] fold and unfold, Cmd+K
  Cmd+0 folds everything and Cmd+K Cmd+J unfolds it, Cmd+K Cmd+1 to 9 by
  level.
- **Finding.** Cmd+F opens the find bar, with case, whole words, regular
  expressions and finding in the selection; Cmd+Alt+F replaces. Cmd+G goes
  to the next match, and Ctrl+G to a line.
- **The pointer.** A double click selects a word, a triple click a line,
  and dragging extends by them; dragging near the edges scrolls; Cmd+click
  goes to a definition.

`Run` runs a command, as a menu item or the app's own key does, and
`Bindings` binds keys: Mods and Key, then ThenMods and ThenKey for chords
such as Cmd+K Cmd+0. An app's bindings come before the defaults
(`DefaultBindings`), and a binding to `NoCommand` takes a key away from
the editor, for the app's own shortcut. `CommandNamed` finds a command by
the name keymaps give it, as "selectNext".

```go
app.code.Bindings = []editor.Binding{
	{Mods: ui.Cmd, Key: ui.KeyP, Command: editor.NoCommand}, // the app's
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyD, Command: editor.DuplicateSelection},
}
if ui.Button(c, "Format").Clicked() {
	app.code.Run(editor.SortLines)
}
```

`Selections`, `SetSelections`, `Select`, `Cursor` and `CursorPos` read
and set the carets and selections, `Insert` types at each of them, `GoTo`
moves the caret to a position and `ScrollTo` shows a line. `Find` opens
the find bar on a query and `FindMatches` counts its matches; `Fold`,
`Unfold` and `Folded` fold lines, and `FoldImports` the first run of
imports, as an editor can on opening a file; `Focused` reports whether the editor has
the keyboard focus, and `VisibleLines` the first and last lines the last
frame showed, as a half page of Vim's moves by. `ScrollY` and
`SetScrollY` read and set how far the lines are scrolled, in DIPs,
`ViewportHeight` how high the part of them in view is, and `LineTop`
where a line starts, below the first, `LineTop` of the count of lines
being the height of them all: editors side by side scroll together with
them, an app draws what joins their lines, as the bands between the
panes of a merge, and a minimap beside the editor shows where its view
is. The scroll bars' thumbs show while the pointer is over the editor, or
drags one, as a scroll container's do, or all the time with
`KeepScrollbars`; the marks of changes, matches and diagnostics on the
vertical one show all the time. `NoScrollbars` takes the bars and their
marks away, and gives their room to the text.

`OnInput` sees each input event before the editor does: keys, the text
typed, edit commands and the pointer. Returning true takes the event, and
the editor leaves it; the others go on as without it. An app's own mode of
editing, as Vim's normal mode, takes keys this way while the editor still
shows and moves the carets. Letters typed at the carets come as
`InputText`: on some systems their keys come first as `InputKeyDown` too,
and taking a key doesn't take its text, so a mode reads letters from
`InputText`. What an input method composes, commits or replaces, as the
accents of macOS's press and hold replace the letter before the caret,
goes to the editor alone. The key after the first of a chord of
`Bindings`, as Cmd+K Cmd+0, is the chord's, as VS Code resolves chords
before an extension's keys.

`NoInputMethod` turns the system's input methods off while the editor has
the keyboard, as Neovide does in Vim's normal mode: keys type the
characters of the keyboard's layout, which come to `OnInput` as
`InputText`, and nothing is composed, so j is j with a Chinese input
method on. Set it in the modes that read letters as commands and clear it
in insert mode. Browsers cannot turn input methods off: there they compose
in the text still.

```go
app.code.CursorShape = editor.CursorBlock // normal mode
app.code.NoInputMethod = true
app.code.OnInput = func(ev ui.InputEvent) bool {
	if ev.Kind == ui.InputText {
		return app.vim.normalKeys(ev.Text) // j, k, dd, …
	}
	return false
}
```

## Options

- `SoftWrap` wraps lines at the editor's width, or at `WrapColumn` columns
  when narrower; wrapped rows keep the line's indentation.
- `LineNumbers` shows the gutter's numbers, absolute, relative to the
  caret's line, or not at all; `GutterMarks` put dots of a color by lines,
  as breakpoints, and `OnGutterClick` reports clicks on numbers.
- `LineBackgrounds` fill lines, from `First` to `Last`, with a color under
  the text and the selections, as the statement a key would run; the
  gutter keeps its own, and lines a fold hides take no room to fill. They
  keep their line numbers as the text is edited, unlike diagnostics and
  inlays, which move with it: compute them again when the buffer's
  `Version` changes.
- `TextBackgrounds` fill ranges of the text, bytes `Start` to `End`, with
  a color under the text and the selections and over line backgrounds, as
  the words a merge changed within its lines. They follow the editor's
  layout: the rows a line wraps into, tabs, wide characters and scrolling
  sideways; lines a fold hides show none. As line backgrounds, they stay
  where they are as the text is edited, cut to its end until computed
  again when the buffer's `Version` changes.
- `TabSize` and `Indent` set indentation: tabs or spaces, as the
  language's files usually have them when zero (Go's tabs).
- `Whitespace` shows spaces and tabs as dots and arrows: within selections
  by default, everywhere, at the ends of lines, or nowhere.
- `WrapGuides` draws guides at columns, as 80 and 100; `NoIndentGuides`
  hides those of indentation.
- `ReadOnly` lets the text be selected and copied, not edited.
- `FitContent` makes the editor as high as its lines, rather than
  scrolling them, to show code in a scroll view or a list.
- `KeepScrollbars` keeps the scroll bars' thumbs showing with the pointer
  away, as a scroll container's `KeepScrollbars`; `NoScrollbars` takes
  the bars away, with their room, the wheel and the keys still
  scrolling.
- `Font` and `LineHeight` set the text: the system's monospaced font of
  13 DIPs, 1.5 times as high, by default.
- `NoAutoClose`, `NoFolding`, `NoActiveLine` and `NoScrollPastEnd` turn
  those behaviors off. `Placeholder` shows while the buffer is empty.
- `CursorShape` paints carets as bars (`CursorBar`), as blocks over the
  character after them (`CursorBlock`), as Vim's normal mode does,
  translucent so that the character shows and outlined while the editor
  hasn't the keyboard, or as underlines (`CursorUnderline`), as its
  pending operators do.
- `DiffBase` is the text the buffer's changes show against in the gutter,
  as the file in Git: bars mark the lines added and changed, and a mark
  where lines were deleted. Cmd+F8 goes to the next change, and Cmd+Alt+Z
  reverts the one at the caret.
- `ContextMenu` adds items to the editor's context menu, after its own.

## Languages and highlighting

`LanguageFor` finds a file's language by its name, and `LanguageNamed` by
its own; `Languages` lists them. The editor knows Go, JavaScript,
TypeScript, JSON, Python, Rust, C, C++, Java, Kotlin, Scala, Swift, Dart,
C#, shell scripts, YAML, TOML, SQL, R, Ruby, Lua, PHP, CSS, HTML, XML,
Markdown, diffs, Dockerfiles, Makefiles, Protocol Buffers, GraphQL and Zig.
A `Language` says how to highlight, comment, match brackets and indent its
text, and with `Imports`, a pattern of a line that imports (`^import\b` for
Go), which lines fold as a run: build one for another language, or copy
one and change it.

A language highlights its text with a `Tokenizer` by default: a function
from a line, and the state the line before it ended in, to the line's
tokens and the state it ends in, as TextMate grammars and Monaco's
tokenizers work. The editor tokenizes the lines it shows, and after an
edit only the lines below it until one starts in the state it started in
before, a frame's time budget at a time.

```go
// Lines starting with # are comments.
lang := &editor.Language{
	Name:        "Notes",
	Extensions:  []string{".notes"},
	LineComment: "#",
	Tokenizer: editor.TokenizerFunc(func(line string, state uint64, tokens []editor.Token) ([]editor.Token, uint64) {
		kind := editor.TokenText
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			kind = editor.TokenComment
		}
		return append(tokens, editor.Token{End: len(line), Kind: kind}), 0
	}),
}
```

A token is a run of a line of one `TokenKind`, which the theme colors:
keywords, types, functions, strings, comments and the others.

### Highlighters

A language can instead bring a highlighter of its own, as one of a parser
such as tree-sitter: `Language.Highlighter` makes the `Highlighter` of each
buffer of the language, and the editor uses it in place of the tokenizer.
The tokenizer itself is one (`NewTokenHighlighter`), so a highlighter
plugs in without the editor changing. The text of a patch's side
(`DiffState.SetPatch`) is its hunks' lines one after another: the
tokenizer starts each hunk in state 0, while a highlighter of its own
sees one text, and highlights it as it sees fit.

```go
type Highlighter interface {
	// Edit follows an edit of the text, old and new its text before and
	// after, and changes what it replaced, in order.
	Edit(old, new editor.Snapshot, changes []editor.Change)
	// Highlight works toward the tokens of the lines up to last, for at
	// most budget, and reports whether they are ready.
	Highlight(last int, budget time.Duration) bool
	// Tokens appends the tokens of line i to dst, and reports whether
	// they are final rather than a guess made before Highlight is done.
	Tokens(i int, text string, dst []editor.Token) ([]editor.Token, bool)
}
```

The editor calls these methods on the main thread: `Edit` as the buffer
changes, `Highlight` before it paints, with the lines it shows, and again
in later frames while it reports false, and `Tokens` for each line it
paints. A `Change` gives the ranges an edit replaced in the old text and
the new, what an incremental parser takes to edit its tree. A parser can
work in the background instead: snapshots are safe to read from any
goroutine, `Chunks` reads them a piece at a time, and the `invalidate`
function the highlighter was made with draws the editors showing the
buffer again once a parse is done.

A highlighter that knows the structure of the text can tell the editor
more, by implementing any of these interfaces too:

- `SyntaxSelector`: `Enclosing` returns the smallest node holding more
  than a range, which Select Larger selects, rather than the word, string
  or brackets around it.
- `Folder`: `FoldEnd` returns the last line folding a line hides, rather
  than what indentation and brackets say.
- `BracketMatcher`: `EnclosingBrackets` returns the innermost pair around
  a range, which the editor highlights and moves between, as `do` and
  `end`.
- `Indenter`: `IndentAfter` returns how many levels more a line Enter
  opens is indented.

## Completions, hovers and diagnostics

The editor shows what the app or a language server knows of the code; the
app answers its requests, at once or later from another goroutine through
`Window.Update`:

```go
app.code.OnCompletion = func(r editor.CompletionRequest) {
	go func() {
		items := app.lsp.Complete(r.Offset) // a language server's
		app.win.Update(func() { app.code.ShowCompletions(r.Offset-len(r.Prefix), items) })
	}()
}
app.code.OnHover = func(r editor.HoverRequest) {
	if doc, start, end, ok := app.lsp.Hover(r.Offset); ok {
		app.code.ShowHover(start, end, doc, nil)
	}
}
app.code.SetDiagnostics(buf.Version(), app.lsp.Diagnostics())
```

- `OnCompletion` asks for completions as a word is typed, or Ctrl+Space
  pressed. `ShowCompletions` lists them below the caret, filtered and
  ranked as typing goes on by the word typed, with their kinds, details
  and documentation; Enter or Tab inserts the one chosen.
- `OnHover` asks what to show as the pointer rests on the text
  (`HoverDelay`, 600 ms by default), or for Cmd+K Cmd+I at the caret.
  `NoPointerHover` turns hovers from the resting pointer off, diagnostics'
  too, as VS Code's `editor.hover.enabled`; the keys still show one.
  `ShowHover` shows text in a popover over the range, or what a function
  builds there. The pointer can go into the
  hover, which stays while it is there, to select and copy its text; the
  hover goes a moment after the pointer leaves the text and the hover,
  and Escape closes it.
- `SetDiagnostics` underlines problems with wavy lines of their severity,
  shows their messages in the hover and marks them on the scroll bar. The
  version is the buffer's the diagnostics are of: the editor moves them
  through the edits made since.
- `SetInlays` shows inlay hints, text in the lines that is not theirs, as
  the types of variables; `SetSuggestion` shows text after the caret as a
  suggestion, which Tab accepts, as an AI's.
- `OnDefinition` is called for Cmd+click and F12 on a word.

## Code snippets

A completion's `Insert` is always plain text, even with
`Kind: editor.CompletionSnippet` (which chooses an icon). To insert a
code template with editable placeholders, parse one and set `Snippet`:

```go
snippet, err := editor.ParseSnippet("for ${1:index}, ${2:value} := range ${3:items} {\n\t$0\n}")
if err != nil {
    // Offer a plain-text completion instead when a server uses
    // snippet syntax the editor does not support.
    return err
}
code.ShowCompletions(start, []editor.Completion{{
    Label: "forrange", Kind: editor.CompletionSnippet, Snippet: snippet,
}})
// Or insert at every current selection, outside the completion list:
code.InsertSnippet(snippet)
```

Parse templates once; a `Snippet` is immutable and can be shared by
editors. `Text()` returns its initial text. The parser supports `$1`,
`${1}`, `${1:default}`, escaped dollars, braces and backslashes, and `$0`
as the final stop (the end when omitted). Tab visits positive numbers in
numeric order, with 0 last; Shift+Tab goes back, stopping at the first
field. Repeated numbers share their default, including forward references,
and edit together through the editor's normal multiple selections.
Coincident empty stops of the same number coalesce as carets do.
Scrolling and input methods follow the first occurrence at the original
primary insertion site.

A completion list takes Tab before the snippet's navigation, so a field
can itself be completed. Escape ends the session after closing any open
completion list or other editor overlay. Moving outside the active
fields, editing the buffer externally, undo/redo or changing buffers
ends the session too. Each field's typing is a separate undo group;
undoing the insertion restores the old text and selections. Redo restores
the text and selections without restarting snippet navigation. Input
methods compose and commit at mirrored fields as at ordinary carets.
A new snippet replaces the previous session.

This is the tab-stop subset of TextMate/LSP syntax: nested placeholders,
variables, choices, transformations, conflicting defaults for one number,
and invalid UTF-8 return errors. Text keeps the template's indentation;
CRLF is normalized to LF. `InsertSnippet` leaves a read-only editor
unchanged. State methods, including snippet insertion, run on the UI
thread as the rest of editor interaction does.

Run `go run ./examples/code-editor`, type `funcdef` or `forrange` in a Go
file, then press Tab to try it.

## Styling

### Themes

`Theme` colors the editor: its background, text, gutter, selections,
carets, guides, diagnostics, changes and each kind of token. Colors left
zero take those of `DefaultTheme`, which follows the window's theme and
appearance, light or dark.

```go
th := editor.DefaultTheme(c.Theme())
th.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: ui.RGB(0xc6, 0x78, 0xdd), Bold: true}
app.code.Theme = th
```

### Bars and popups

The find bar, the go to line bar, the completions and the hovers are
parts the app can build itself, as it styles ui's
[bases](custom-widgets.md): `RenderFindBar`, `RenderGoToLine`,
`RenderCompletions` and `RenderHover` are each given the element the
editor places (a toolbar over its top right corner, a bar at its top, a
list below the caret, a card above the text hovered), which keeps the
presses from the text under it and the keyboard where the part needs it
(in the editor while the completions show, in the hover as its text is
selected), and a handle to what the part shows and does. Style the element and build its
content; its position can change too. The editor's own looks are
`DefaultFindBar`, `DefaultGoToLine`, `DefaultCompletions` and
`DefaultHover`, which a render function can call and then restyle.

```go
app.code.RenderFindBar = func(c *ui.Context, bar ui.Element, f *editor.FindBar) {
	t := c.Theme()
	bar.Row().Gap(8).Padding(6).Background(t.Surface).Radius(12).Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, 0.2))
	bar.Children(func() {
		f.QueryInput(ui.TextInputBase(c, f.Query()).Placeholder("Search").Width(180))
		ui.Toggle(c, f.Regexp(), ".*").KeepFocus()
		ui.Text(c, f.Status()).TextColor(t.TextMuted)
		if ui.Button(c, "Next").KeepFocus().Clicked() {
			f.Next()
		}
	})
}
app.code.RenderHover = func(c *ui.Context, panel ui.Element, h *editor.HoverCard) {
	editor.DefaultHover(c, panel, h) // the editor's look, on another background
	panel.Background(app.hoverColor)
}
```

- `FindBar` binds fields and toggles: `Query` and `Replacement`, and
  `CaseSensitive`, `WholeWord`, `Regexp` and `InSelection`, are pointers,
  and the matches follow them as they change. `QueryInput` and
  `ReplacementInput` give inputs the bar's keys: the query takes the
  focus as the bar opens, Enter and Shift+Enter go through the matches,
  Alt+Enter selects them all, Enter and Cmd+Enter in the replacement
  replace one or all, and Escape closes. `Matches` and `Status` count the
  matches, and `Next`, `Previous`, `SelectAll`, `Replace`, `ReplaceAll`
  and `Close` act; `Replacing` says whether the bar replaces too, as
  Cmd+Alt+F opens it.
- `GoToLineBar` has `Text` for its field, which `Input` gives the bar's
  keys, `Cursor`, `LineCount` and `Target` to tell where it goes, and
  `Go` and `Close`.
- `CompletionMenu` has the `Items` shown, best first, and the one
  `Selected`. `Item(i)` makes the row of an item, selected as the arrows
  move, scrolled into view and inserting the item when clicked; style it
  and give it content. `Font`, `LineHeight` and `Theme` show code as the
  editor does, and a `CompletionKind` has an `Icon` and the `Token` whose
  color tints it.
- `HoverCard` has the `Diagnostics` of the text hovered, the `Text` the
  app gave `ShowHover`, and `Content`, which builds what its function
  builds.

## Accessibility

Assistive technology sees a multi-line text field, named by `Label`, with
the text of the buffer and the newest selection, which it moves; the
completions are a list, and the find bar a toolbar that tells the count
of matches. The editor's menu, its find and go to line bars and their
names are in the window's language
([MyGo's own strings](../i18n.md#mygos-own-strings)). Input methods compose in the editor as in other text fields,
with their candidate window at the caret.

# Markdown


Package `ui/markdown` shows Markdown in native UI: CommonMark, GitHub's
tables, task lists, strikethrough and autolinks, footnotes and math. It
is built for the replies of language models, which stream in token by
token: text appended is parsed alone, half-written syntax shows as what
it will become, and text comes in smoothly, fading in.

```go
import "github.com/egoist/mygo/ui/markdown"

// Kept in the app's state, shared by every view.
app.md = markdown.Options{}

// In the view:
markdown.View(c, app.readme, &app.md)
```

`View` shows a finished document. Give it the source each frame, as a
`Text` its string: it parses again only what changed since the last
frame, and keeps what it made of the rest, the text layouts of its
paragraphs included.

## Streaming replies

`Stream` shows Markdown still being written. Append the tokens of a
reply to its string as they come, and show it with `View` once it is
complete, in the same place, so that the view carries on:

```go
func (app *chat) message(c *ui.Context, m *Message) {
	if m.Done {
		markdown.View(c, m.Text, &app.md)
	} else {
		markdown.Stream(c, m.Text, &app.md)
	}
}
```

Appending text parses only the outermost block still open before it,
or nothing but the new lines when no block is: a token costs the same at
the end of a long reply as at its start, and blocks already shown are
neither parsed nor laid out again.

The end of a reply shows what its syntax will most likely become, as
the chat apps show it, without changing its text: `**bol` is bold, an
unclosed code span is code, `$$x_i` (and `$x_i` with `ExtDollarMath`)
is math, whose `_` and `*` are TeX rather than emphasis,
`[the docs](https://ex` is a link whose destination is still coming (not
clickable yet), and an open code fence shows as code. Lines that could still turn into something else wait for
what follows them: a line of a few markers, as `-` or `==`, which would
make the paragraph above a heading, and a table's header, which waits
for its delimiter row so that it does not flash as a paragraph of pipes.
Blocks already finished are never changed, and a reply shown with
`View` once done is exactly as CommonMark reads it.

### Pace and fade

Tokens arrive in bursts. `Stream` shows text at an even pace instead,
a word at a time, catching up with what came within a quarter of a
second, and each word fades in. `Options.Reveal` changes how:

```go
app.md.Reveal = &markdown.Reveal{
	Unit:    markdown.ByLine,         // a line at a time, once complete
	CatchUp: 400 * time.Millisecond,  // text waits at most this long
	Fade:    500 * time.Millisecond,  // and fades in over this
}
```

`ByWord` (the default), `ByCharacter`, `BySentence`, `ByLine` and
`ByParagraph` show a unit at a time, the last three only once it is
complete: a sentence ends at a period, a question or exclamation mark
followed by a space, an ideographic full stop or the end of a line.
`AtOnce` shows text as it comes. A negative `Fade` turns fading off, as does the
desktop's Reduce Motion. Text fades by its color and the rest of blocks
(list markers, code blocks, a table's rows) by their opacity, which
paint only: nothing is laid out again for it. What is still to reveal
when the message switches to `View` keeps coming in.

### Once a reply has shown

The last bytes of a reply arrive before its last words show: the pace
is still catching up, and they still fade in. `Revealed`, given the
element `View` or `Stream` returned, reports whether it shows all of its
text, done fading in, so that a chat shows a reply's time and actions
then, not ahead of words still coming in:

```go
var md ui.Element
if m.Done {
	md = markdown.View(c, m.Text, &app.md)
} else {
	md = markdown.Stream(c, m.Text, &app.md)
}
if m.Done && markdown.Revealed(md) {
	ui.Text(c, m.Time).TextColor(c.Theme().TextMuted)
}
```

It tells what the frame shows: read it after the view, and build what it
decides below it. The view asks for frames while it reveals and fades,
so the time shows on its own once the fade ends. Text that shows at once
(`AtOnce`, a `View` of text no `Stream` left to reveal) is revealed in
its first frame, and without a fade (a negative `Fade`, Reduce Motion)
once its last word shows. More text makes it false again. Of a `Stream`
it tells only that what came so far shows, so check that the reply is
complete too, as `m.Done` above.

## A chat

Show each message as a row of a [list](list.md) that follows its end:
the list keeps the end in view as the reply grows, gliding up to it with
`FollowAnimation`, until the user scrolls up, and follows again once
they scroll back to the end. `AtEnd` and `ScrollToEnd` make a button
back to the latest.

```go
app.list.Key = func(i int) any { return app.messages[i].ID }
app.list.FollowEnd = true
app.list.FollowAnimation = &ui.Smooth
ui.List(c, &app.list, len(app.messages), func(i int) {
	app.message(c, app.messages[i])
}).Grow(1).Gap(16).Justify(ui.End)
```

The [chat example](../../examples/markdown-chat) streams canned replies
in this way.

## Selecting and copying

A view's text is one selection across its blocks, as in a browser. Copy
puts the selected text on the clipboard, with blank lines between
paragraphs, a list's items on lines of their own and a table's cells
separated by tabs; as HTML, so that it pastes with its formatting into a
[rich text editor](rich-text-editor.md), mail or a word processor; and as
Markdown (`transfer.Markdown`). The context menu's Copy as Markdown, and Shift+Cmd+C
(Shift+Ctrl+C on Linux and Windows), copy the source of the selection,
completing the emphasis, code and links it cuts through, or the whole
document when nothing is selected. Text keeps its selection as a reply
streams in.

`ShareSelection` leaves the selection to a `Selectable` container around
the view, as one across a chat's messages.

## Links and images

Links to `http`, `https` and `mailto` addresses (`ui.SafeToOpen`, as the
rich text editor opens them) open as a [link](link.md)'s do, and show their address in a tooltip; a path
without a scheme goes there in the router around the view; `#a-heading`
scrolls to the heading of that anchor (`Slug`); and links of other
schemes, as `javascript:`, do nothing. `OnLink` handles every click
instead:

```go
app.md.OnLink = func(url string) { app.open(url) }
```

`LinkMenu` gives each link a [context menu](context-menu.md) of its own,
given its destination, as a chat whose replies link files offers Copy
Path and Reveal in Finder for the file right-clicked. The link's menu
replaces the one around the view, as its `Selectable` container's, and
`EditItems` puts the text's Copy and Select All back in it; links to
`#a-heading`, footnote references and a link whose destination is still
coming keep the menu around them:

```go
app.md.LinkMenu = func(m *ui.Menu, url string) {
	if m.Item("Copy Path").Chosen() {
		app.copyPath(url)
	}
	if m.Item("Reveal in Finder").Chosen() {
		app.reveal(url)
	}
	m.Separator()
	m.EditItems()
}
```

`Anchor` opens a document at a heading, given its slug with or without
the `#`: a docs reader that opens `guide.md#install` from a search hit,
or that comes back where its user was. The view scrolls to it as the
document shows, as another document replaces it and as `Anchor`
changes; `AnchorVersion` asks for the same heading again. An `OnLink`
that handles `#a-heading` links sets it:

```go
app.md.OnLink = func(url string) {
	if strings.HasPrefix(url, "#") {
		app.md.Anchor = url
		app.md.AnchorVersion++
		return
	}
	app.open(url)
}
```

`LinkIcon` puts an icon at the start of a link's text, from its
destination and the size of the text around it: a box as `InlineMath`
makes, which clicks as the link and stays on the line of its first
word. It copies as nothing, as text, HTML and Markdown. When it returns
false, the link has none; a link still streaming in has none yet. Links
take their color from `Theme.Link`:

```go
app.md.LinkIcon = func(dest string, size float32) (markdown.InlineObject, bool) {
	if !strings.HasPrefix(dest, "http") {
		return markdown.InlineObject{}, false
	}
	return markdown.InlineObject{
		Width: size * 1.2, Height: size * 0.86, Depth: size * 0.14,
		Draw: func(p *ui.Painter, r ui.Rect) {
			p.Icon(globe, ui.Rect{X: r.X, Y: r.Y, W: size, H: size}, app.md.Theme.Link)
		},
	}, true
}
```

Images show their description until `LoadImage` loads them, on a
goroutine of their own; without it, they stay descriptions, as a view
loads nothing from the network by itself. `FileImages(dir)` loads the
files of a directory. A paragraph of images alone shows them as large
as they are, at most as wide as the view.

```go
app.md.LoadImage = markdown.FileImages(filepath.Dir(app.path))
```

## Code

Fenced code is highlighted in its language with the tokenizers and the
colors of the [code editor](code-editor.md), a line at a time from the
state of the line before, so that a block streaming in highlights only
its new lines; its lines are laid out in chunks, so that it lays out
only its last chunk again as it grows. A block has its language and a
Copy button above its code, which scrolls sideways (`WrapCode` wraps it
instead). The button waits for a block being written to close.

`CodeBlock` builds code blocks in place of `DefaultCodeBlock`, and
`Code` the blocks of the languages it lists, as diagrams. Each gets the
block's language, its code and whether its fence is still open, and can
use the parts of the default:

```go
app.md.Code = map[string]func(*ui.Context, *markdown.CodeBlockView){
	"mermaid": func(c *ui.Context, b *markdown.CodeBlockView) {
		if b.Open { // still streaming: show the code until it closes
			markdown.DefaultCodeBlock(c, b)
			return
		}
		ui.Image(c, app.diagram(b.Code()))
	},
}
app.md.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, b.Language).Bold()
		b.Body(c)       // the highlighted, selectable code
		b.CopyButton(c) // the copy button
	})
}
```

## Math and HTML

Math between `$$` and `$$`, `\[` and `\]`, or `\(` and `\)` shows as its
TeX in the code font, as language models write it; `$…$` too with the
`ExtDollarMath` extension, except where the dollars read as money.
`Math` builds blocks of display math instead, as with a renderer of
your own; a block still streaming is `Open` to `Block`, which can show
its TeX until it closes.

`InlineMath` makes math within text an object of the paragraph, from
its TeX and the size of the text around it: a box `Width` wide,
`Height` above the baseline and `Depth` below it, which wraps with the
words and which `Draw` paints. Copied, as text or Markdown, it is its
source, as `$x_i$`. When it returns false, the math shows as TeX. It is
asked whenever its paragraph changes, as each frame of one streaming in,
so keep what it makes:

```go
app.md.InlineMath = func(tex string, size float32) (markdown.InlineObject, bool) {
	f, err := app.formula(tex, size) // laid out once, kept by TeX and size
	if err != nil {
		return markdown.InlineObject{}, false
	}
	return markdown.InlineObject{Width: f.W, Height: f.Ascent, Depth: f.Descent, Draw: f.Draw}, true
}
```

Raw HTML shows as text, so that a reply cannot hide anything. Within
text, the tags of formatting that models write (`<b>`, `<i>`, `<u>`,
`<s>`, `<code>`, `<kbd>`, `<mark>`, `<sub>`, `<sup>` and `<br>`) apply;
`HTML` builds HTML blocks, as for `<details>` or `<think>`.

## Styling

A view looks as the window does, in its fonts and colors. `Theme` sets
what changes, the rest taking `DefaultTheme`'s values for the window's
theme, so that a theme of one field changes one thing:

```go
app.md.Theme = &markdown.Theme{
	FontSize:     15,
	Link:         ui.Hex("#7c3aed"),
	CodeFont:     "JetBrains Mono",
	BlockSpacing: 1,
	Headings: [6]markdown.HeadingStyle{
		{Size: 1.8, Weight: 800, Rule: true},
		{Size: 1.4, Rule: true},
	},
}
```

A theme sets the font, the code font and their line heights; the colors
of text, secondary text, links, strong text and code; the size of code;
the spacing between blocks and list items; each heading level's size,
weight, color, spacing and rule; list indentation; block quotes' bar and
text; code blocks' background, border, corners, padding and token colors
(an `editor.Theme`); tables' lines, header, stripes, padding and widest
column; marked text; and check boxes. Lengths are in ems, times the font
size, so that a theme scales with it. `LinkWeight` is the least weight of
links' text, for links that read as links where color alone doesn't, in a
gray print theme or to a colorblind reader.

`Block` replaces how any block shows: it is asked first for each block,
and builds it when it returns true, from the document's blocks and
inlines (see [the parser](#the-parser)):

```go
app.md.Block = func(c *ui.Context, d *parser.Document, i int) bool {
	b := d.Block(i)
	if b.Kind != parser.ThematicBreak {
		return false
	}
	ui.Text(c, "⁂").TextAlign(ui.Center).AlignSelf(ui.Stretch)
	return true
}
```

## Accessibility

Headings are headings of their level, lists and their items lists and
list items, links links, tables tables of rows, column headers and
cells, task items check boxes, checked or not, images images named by
their descriptions, and code blocks groups named by their language,
with a Copy code button.

## The parser

The parser is a package of its own, `ui/markdown/parser`, which
imports nothing of ui, so that what only reads Markdown (a rich text
editor importing it, a server rendering HTML) leaves the view out.
`Parse` reads a string, `Append` adds text and
`Edit` changes a range, each parsing again only what the change touches.
An edit parses from the top-level block before it until a block starts,
with nothing open, where one started before, from which it keeps the
blocks it had: typing in a long document's preview parses a few blocks.
A link defined later than where it is used makes the blocks using it
parse their inlines again, and only those.

```go
d := parser.Parse(src)
d.Append(token)
for i := range d.Children(0) { // the top-level blocks
	b := d.Block(i)
	if b.Kind == parser.CodeBlock {
		fmt.Println(b.Language(), d.Lines(i))
	}
}
html := d.HTML()
```

Blocks and inlines are flat slices in document order, each with its
byte range in the source (`Start` and `End`) and the index after its
children (`Next`). `Inlines` parses a block's inlines the first time it
is asked. `New(ext)` reads other extensions than `Default`, and
`SetStreaming` completes the end as `Stream` does.

It passes all 652 examples of the CommonMark 0.31.2 spec and those of
GitHub's extensions, takes linear time on the inputs that are slow in
other parsers, as cmark does, and depends on nothing outside the
standard library.

One rule is relaxed for Chinese, Japanese and Korean, which put no space
around emphasis: next to a CJK character, full-width punctuation or a
curly quote, `*` and `~~` open and close even where punctuation would
stop them in CommonMark, so `**结论。**后文` is bold. `_` and a single `~`
keep CommonMark's rule. No example of the spec has CJK text there, so
all of them still pass.

## Performance

Measured on an Apple M5 with `CGO_ENABLED=0 GOMAXPROCS=1`, on a 50 KB
reply of headings, nested lists, code, a table and links:

| | Time | Allocated |
|---|---:|---:|
| Parsing it whole, inlines included | 0.3 ms | 0.8 MB, 1,300 allocations |
| Appending a token of 4 bytes, the tail's inlines parsed | 2 µs | 600 B, 2 allocations |
| A frame showing it whole, nothing changing | 1.3 ms | 16 KB, 350 allocations |
| A frame with a token of 16 bytes appended | 0.8 ms | 36 KB, 350 allocations |

Frames build every block's elements, as views do, while the parse, the
spans of paragraphs and their text layouts are kept: a document much
longer than a chat's reply is best shown a section at a time.


## Run the benchmarks and tests

```sh
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui/markdown -run '^$' -bench . -benchmem
go test ./ui/markdown -run 'TestCommonMark|TestGFM|TestAppend|TestEdits|TestPathological' -v
```


# Text input


`ui.TextInput` creates a single-line text input editing a `*string`, and
`ui.TextArea` a multi-line one. They change the string as the user types:
read it, or ask `Changed` for work that follows, and `Submitted` for Enter
in a single-line input.

```go
if ui.TextInput(c, &app.query).Placeholder("Search").Label("Search").Changed() {
	app.results = search(app.query)
}

if ui.TextInput(c, &app.name).AutoFocus().Submitted() {
	app.rename()
}

ui.TextArea(c, &app.notes).Height(160)
```

- `Placeholder` shows a text while the input is empty, on one line in a
  `TextInput`, wrapped in a text area. On a field around an input, as a
  `SearchField`, `TokenField` or `Combobox`, it shows in that input.
  `PlaceholderColor` paints it in another color than the theme's
  `TextMuted`, as a warning hint in a form.
- `CaretWidth` paints the caret wider than one DIP, as a terminal's block
  or a large-print theme's, or hides it with zero; it changes nothing else.
- `Password` hides what it holds, and keeps it from the clipboard and input
  methods, in the frames that call it: an eye button that shows the text
  stops calling it. Only a `TextInput` takes it: as on every platform, a text
  area has no password mode.
- `AutoFocus` gives it the keyboard focus as it appears, as the first field
  of a dialog.
- `ReadOnly(true)` shows the text without letting the user change it: it
  still takes the focus, without a caret, and its text can be selected and
  copied.
- `MaxLength(n)` keeps the user from making the text longer than n, counted
  as HTML's `maxlength` counts it, in UTF-16 code units: what is typed,
  pasted or committed by an input method is cut to what fits. A value the app
  sets is not cut, and n <= 0 means no limit.
- `Disabled(true)` grays it out.
- `OnPaste(fn)` sees a paste (Cmd+V or Ctrl+V, the Edit menu's or the
  context menu's Paste) before it is inserted, with the clipboard's text,
  empty when it holds none, as with an image. Returning true takes the
  paste, which then changes nothing, as `preventDefault` does on the web:

  ```go
  ui.TextArea(c, &app.draft).Lines(1, 8).OnPaste(func(text string) bool {
  	if len(text) > 10_000 {
  		app.attach(text) // a large block becomes an attachment
  		return true
  	}
  	if text == "" {
  		if png := mygo.Clipboard.ReadImage(); png != nil {
  			app.attachImage(png)
  			return true
  		}
  	}
  	return false
  })
  ```

- `OnTextKey(fn)` sees a key that types no text before the input takes
  it: Enter, Tab, Escape, Backspace and Delete, the arrows and the other
  keys that move the caret, and chords with Cmd or Ctrl, as AppKit's
  `doCommandBySelector` and a web page's `keydown` do. fn runs as the key
  comes, after the keys and text typed before it, and returning true takes
  the key: a text area inserts no newline, a `TextInput` does not submit,
  Tab does not move the focus. A key left goes on as without fn. Keys that
  a `Shortcut` around the focus handles go to it first, as with
  `HandleInput`, unless the input edits with them, as with Enter and the
  arrows. fn may run between frames, so it reads the text and replaces it
  through its `ui.TextKey` rather than an `Element` kept from the view
  (they panic once fn returns), as a chat composer sends with Enter, breaks
  the line with Shift+Enter, and puts the last message back with Up:

  ```go
  ui.TextArea(c, &app.draft).Lines(1, 8).OnTextKey(func(k ui.TextKey) bool {
  	switch {
  	case k.Key == ui.KeyEnter && k.Mods == 0:
  		app.send(k.Text())
  		k.SetText("")
  		return true
  	case k.Key == ui.KeyUp && k.Mods == 0 && k.Text() == "":
  		k.SetText(app.lastSent())
  		return true
  	}
  	return false // Shift+Enter breaks the line
  })
  ```

A `TextInput` too narrow for its text keeps the caret in view while it has
the focus, and shows the start of its text without it.

`TextRanges` colors runs of the text, and sets their weight, in the frames
that call it, as a message field shows the mentions in what is typed:

```go
in := ui.TextAreaBase(c, &app.draft).Lines(1, 8)
for _, m := range mentions(app.draft) {
	in.TextRanges(ui.TextRange{Start: m.start, End: m.end, Color: m.color, Weight: 600})
}
```

The ranges are runes of the text, found in it each frame. They may overlap
and nest, as the attributes of an attributed string (`NSAttributedString`)
do: a range sets its `Color` when its alpha is not zero and its `Weight`
when it is not zero, and where ranges overlap, each of the two is the last
given range's that sets it, across the calls of the frame. A bold range
with a colored mention inside it shows the mention bold and colored:

```go
in.TextRanges(
	ui.TextRange{Start: 0, End: 40, Weight: 700},
	ui.TextRange{Start: 10, End: 16, Color: t.Accent},
)
```

A password shows none. While an input method composes, its text sits
unstyled at the caret and the ranges keep their style around it.

A text area is at least a few lines high and grows with its text; given a
height, it scrolls within it, with the wheel and a scroll bar, and keeps
the caret in view as it moves. `Lines(min, max)` makes it as high as its
text wraps at its width, from `min` lines up to `max`, past which it
scrolls, as a message field grows with what is typed:

```go
ui.TextArea(c, &app.draft).Lines(1, 8)
```

A text area lays out only the paragraphs in view and keeps their layouts
until they change, so that it holds texts of hundreds of thousands of
lines, as a log or a source file, and stays as quick to type in.

## Editing

Text inputs edit as the platform's text fields do: selection with the
pointer (a double click selects a word, a triple click a line), with Shift
and the arrows, and by words and lines with the platform's keys; undo and
redo; cut, copy and paste, also from the Edit menu's roles; and a context
menu of the editing commands. They take text composed with input methods,
which see the text around the caret, so that press and hold, Japanese
conversion and predictions work as in other apps. The keys typed while an
input method composes are its own, as Enter choosing a candidate or Escape
giving the composition up: they submit nothing, press no shortcut and
close no dialog. `Composing` reports whether it composes, its text not yet
in the string.

The input keeps the text being edited, its selection and its undo history
from frame to frame; setting the string from elsewhere replaces the text.
It preserves the last undo step, so an app can reformat what the user
typed without losing that step. Setting the string to `""` does not discard
the history: Undo can restore text from before the clear.

To start a new editing session, clear the string and change the input's
[key](views.md#keys). The new identity gets a new editor, without the old
selection, composition or undo history:

```go
// Keep secret and secretSession in app state.
ui.TextInput(c.Key(app.secretSession), &app.secret).Password().Label("Secret")

if ui.Button(c, "Cancel").Clicked() {
	app.secret = ""
	app.secretSession++ // the next build creates a fresh editor
}
```

For indexed storage, `ui.TextInputBuffer` and `ui.TextAreaBuffer` edit a
`*ui.TextBuffer`. Replacing its contents externally with `buffer.Set("")`
discards the control's undo and redo history and composition when it next
synchronizes with the buffer; its selection is clamped to the new text.
Neither changing a key nor replacing a buffer guarantees that old text is
erased from memory; existing `TextSnapshot` values retain their contents.

A single-line input replaces line breaks in pasted text with spaces. Use
`OnPaste` to reject a multiline value before it is changed or inserted:

```go
ui.TextInput(c, &app.secret).Password().OnPaste(func(text string) bool {
	return strings.ContainsAny(text, "\r\n") // import "strings"
})
```

## The caret

`TextSelection` returns the selection as offsets in runes into the text,
the caret where they are equal, and `SetTextSelection` moves it, as an app
completing the word being typed puts the caret after it:

```go
input := ui.TextArea(c, &app.draft)
if start, _ := input.TextSelection(); app.completed != "" {
	app.draft, start = complete(app.draft, start, app.completed)
	input.SetTextSelection(start, start)
}
```

`TextSelection` returns the selection in order. `TextAnchorCaret` returns
it from where it was started to its caret, so an app can tell a selection
made backwards from one made forwards: an editor's own keys that extend a
selection, such as Shift-Home moving to the first character of the line,
extend it from its anchor:

```go
anchor, caret := input.TextAnchorCaret()
input.SetTextSelection(anchor, lineStart(app.draft, caret))
```

## Beside the lines

`TrackLines` keeps in a `ui.TextLines` where a text area lays out the lines
of its text, for an app that paints beside them: numbers by the first row
of each line, or marks on runs of a line. A line is the text between
newlines, on several rows where the area wraps it. `Top(i)` is the top of
line `i` in the area's content, below its padding and before it scrolls,
`At(y)` the line at a height, `Rows(i)` the runes where each row of line
`i` starts, and `Height` the height of the text. With the area's
[`ScrollState`](scroll.md#keeping-the-offset), a gutter numbers the lines
in view:

```go
const pad = 8 // the text area's top padding
ui.Row(c).Grow(1).Children(func() {
	ui.Box(c).Width(48).FillHeight().Draw(func(p *ui.Painter, r ui.Rect) {
		top := r.Y + pad - app.scroll.Y // the top of the area's content
		p.Clip(r, 0, func() {
			for i := app.lines.At(app.scroll.Y); i < app.lines.Count() && top+app.lines.Top(i) < r.Y+r.H; i++ {
				p.Text(r.X+8, top+app.lines.Top(i), strconv.Itoa(i+1), 12, p.Theme().TextMuted)
			}
		})
	})
	ui.TextArea(c, &app.source).Padding(pad, 12).Grow(1).FillHeight().
		TrackLines(&app.lines).TrackScroll(&app.scroll)
})
```

The heights of the lines the area has not laid out are estimates, as its
scroll bar's are, until it shows them. In `Draw` the lines are those of the
frame; while the view is built, those of the last one.

## Errors

In a [field](form.md), `Error` marks the value invalid: the input draws its
border in the theme's `Danger` color, and assistive technology reads the
message with it.

```go
ui.Field(c, "Email", func() {
	ui.TextInput(c, &app.email)
}).Error(app.emailError)
```

## Without a look

`ui.TextInputBase` and `ui.TextAreaBase` are text inputs without padding,
background, border or corners, for inputs of your own design: see
[custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a text field, or a text area, named by its
`Label` or its field, whose value is its text, with the caret and the
selection; it edits it as typing does, unless it is read-only. A password
field hides its value.

