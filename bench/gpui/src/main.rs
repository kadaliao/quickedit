//! QuickEdit benchmark app for GPUI + gpui-component (via the gpui-kit
//! umbrella crate). See ../README.md for the contract and NOTES.md for how
//! "ready" and "highlighted" are decided.
//!
//! `bench-gpui <path> [ignored...]`
//!
//! - `.md`: gpui-component's Markdown `TextView`, scrollable, selectable.
//! - anything else: gpui-component's code `Editor` (an `EditorState` in
//!   code-editor mode, language "go", line numbers on by default, editable).
//!
//! Test hooks (not used by the runner): `BENCH_TRACE=1` prints startup
//! milestones to stderr. `BENCH_FREEZE=ready[:ms]` or
//! `BENCH_FREEZE=highlighted[:ms]` blocks the main thread `ms` (default 40)
//! milliseconds after printing that line, so nothing drawn later can show up
//! in a screenshot. (Blocking at once is no good: AppKit then never puts the
//! window on screen at all, as ordering it in also needs the main run loop.)

use std::io::Write as _;
use std::path::Path;

use gpui_kit::component::input::{Editor, EditorState, RopeExt as _};
use gpui_kit::component::text::{TextView, TextViewState};
use gpui_kit::*;

actions!(quickedit, [Quit]);

/// `BENCH_TRACE=1` prints startup milestones to stderr, in ms since `main`.
static TRACE: std::sync::OnceLock<Option<std::time::Instant>> = std::sync::OnceLock::new();
fn trace(what: &str) {
    if let Some(Some(t0)) = TRACE.get() {
        eprintln!(
            "TRACE {:8.1} ms {what}",
            t0.elapsed().as_secs_f64() * 1000.0
        );
    }
}

fn emit(line: &str, cx: &mut App) {
    trace(line);
    let mut out = std::io::stdout().lock();
    let _ = writeln!(out, "{line}");
    let _ = out.flush();
    drop(out);
    if let Ok(freeze) = std::env::var("BENCH_FREEZE") {
        let (what, ms) = freeze.split_once(':').unwrap_or((&freeze, "40"));
        if what == line {
            let delay = std::time::Duration::from_millis(ms.parse().unwrap_or(40));
            cx.spawn(async move |cx| {
                cx.background_executor().timer(delay).await;
                loop {
                    std::thread::park();
                }
            })
            .detach();
        }
    }
}

/// Whether the editor's tree-sitter tree has landed. The highlighter hands its
/// fold ranges to the editor in the same step it installs a parsed tree (both
/// for the synchronous parse inside the editor's first render and for the
/// background parse), and every Go file with a multi-line function has some.
fn has_syntax_tree(editor: &Entity<EditorState>, cx: &App) -> bool {
    let state = editor.read(cx);
    let lines = state.text().lines_len();
    let display_map = state.display_map();
    (0..lines.min(2000)).any(|line| display_map.is_fold_candidate(line))
}

enum Content {
    Code(Entity<EditorState>),
    Markdown(Entity<TextViewState>),
}

struct QuickEdit {
    content: Content,
    /// `ready` has been scheduled for the frame being drawn.
    ready_scheduled: bool,
    /// Code: `ready` printed without a tree; waiting for the background parse.
    awaiting_tree: bool,
    /// Code: the tree landed; the next drawn frame shows the highlights.
    highlight_due: bool,
    _observer: Subscription,
}

impl QuickEdit {
    fn new(path: &Path, text: String, window: &mut Window, cx: &mut Context<Self>) -> Self {
        let is_markdown = path
            .extension()
            .is_some_and(|e| e.eq_ignore_ascii_case("md") || e.eq_ignore_ascii_case("markdown"));
        if is_markdown {
            let state = cx.new(|cx| TextViewState::markdown(&text, cx));
            // Markdown over 4 KB is parsed on a background thread, so the
            // first frames can be empty. React when the parse is committed.
            let observer = cx.observe(&state, |_, _, cx| cx.notify());
            Self {
                content: Content::Markdown(state),
                ready_scheduled: false,
                awaiting_tree: false,
                highlight_due: false,
                _observer: observer,
            }
        } else {
            // Soft wrap is on by default in gpui-component; code editors
            // (Sublime, Zed, VS Code) leave it off for source code. With it
            // on, wrapping big.go's 108K lines adds ~0.7 s to the first frame.
            let editor = cx.new(|cx| {
                EditorState::new(window, cx)
                    .language("go")
                    .soft_wrap(false)
                    .default_value(text)
            });
            // An editor opens ready to type into.
            editor.read(cx).focus_handle(cx).focus(window, cx);
            let observer = cx.observe(&editor, |this: &mut Self, editor, cx| {
                if this.awaiting_tree && has_syntax_tree(&editor, cx) {
                    this.awaiting_tree = false;
                    this.highlight_due = true;
                    cx.notify();
                }
            });
            Self {
                content: Content::Code(editor),
                ready_scheduled: false,
                awaiting_tree: false,
                highlight_due: false,
                _observer: observer,
            }
        }
    }
}

impl Render for QuickEdit {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        trace("QuickEdit::render");
        let root = div().size_full().flex().flex_col();
        match &self.content {
            Content::Code(editor) => {
                if !self.ready_scheduled {
                    // The editor sets its text and runs its first (time-boxed)
                    // parse in its own render, later in this same frame.
                    self.ready_scheduled = true;
                    let editor = editor.clone();
                    let this = cx.entity().downgrade();
                    window.on_next_frame(move |_, cx| {
                        let tree = has_syntax_tree(&editor, cx);
                        emit("ready", cx);
                        if tree {
                            // The synchronous parse finished inside the first
                            // frame, so that frame was already highlighted.
                            emit("highlighted", cx);
                        } else {
                            let _ = this.update(cx, |this, _| this.awaiting_tree = true);
                        }
                    });
                }
                if self.highlight_due {
                    self.highlight_due = false;
                    window.on_next_frame(|_, cx| emit("highlighted", cx));
                }
                root.child(Editor::new(editor).size_full().bordered(false))
            }
            Content::Markdown(state) => {
                if !self.ready_scheduled && !state.read(cx).rendered_text().source().is_empty() {
                    // Code blocks are highlighted synchronously while the
                    // blocks are laid out, so this frame is highlighted too.
                    self.ready_scheduled = true;
                    window.on_next_frame(|_, cx| {
                        emit("ready", cx);
                        emit("highlighted", cx);
                    });
                }
                root.child(
                    div().flex_1().min_h_0().px_6().py_4().child(
                        TextView::new(state)
                            .scrollable(true)
                            .selectable(true)
                            .size_full(),
                    ),
                )
            }
        }
    }
}

fn main() {
    TRACE.get_or_init(|| std::env::var_os("BENCH_TRACE").map(|_| std::time::Instant::now()));
    trace("main");
    let Some(path) = std::env::args_os().nth(1) else {
        eprintln!("usage: bench-gpui <file>");
        std::process::exit(2);
    };
    let path = std::path::PathBuf::from(path);
    let text = match std::fs::read(&path) {
        Ok(bytes) => String::from_utf8(bytes)
            .unwrap_or_else(|e| String::from_utf8_lossy(e.as_bytes()).into_owned()),
        Err(err) => {
            eprintln!("bench-gpui: {}: {err}", path.display());
            std::process::exit(1);
        }
    };
    let title: SharedString = path
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default()
        .into();

    gpui_kit::application()
        .with_assets(gpui_kit::assets::Assets)
        .run(move |cx: &mut App| {
            trace("launched");
            gpui_kit::init(cx);
            trace("init done");
            cx.on_action(|_: &Quit, cx| cx.quit());
            cx.bind_keys([KeyBinding::new("cmd-q", Quit, None)]);
            cx.set_menus([Menu::new("QuickEdit").items([MenuItem::action("Quit", Quit)])]);
            cx.on_window_closed(|cx, _| cx.quit()).detach();

            let options = WindowOptions {
                focus: true,
                titlebar: Some(TitlebarOptions {
                    title: Some(title.clone()),
                    ..Default::default()
                }),
                window_bounds: Some(WindowBounds::Windowed(Bounds::centered(
                    None,
                    size(px(1000.0), px(700.0)),
                    cx,
                ))),
                ..Default::default()
            };
            gpui_kit::open_window(options, cx, move |window, cx| {
                trace("window created");
                cx.new(|cx| QuickEdit::new(&path, text, window, cx))
            })
            .expect("failed to open window");
            trace("open_window returned");
            cx.activate(true);
        });
}
