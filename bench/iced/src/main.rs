//! iced's "open a file" app for the QuickEdit comparison (see ../README.md).
//!
//! `bench-iced <path> [ignored...]`: a 1000×700 window titled with the file's
//! name, showing rendered Markdown for `.md` files and an editable
//! `text_editor` with syntect highlighting for anything else.

use std::io::Write;
use std::path::Path;

use iced::widget::{container, markdown, scrollable, text_editor};
use iced::{Element, Font, Length, Subscription, Theme, highlighter, window};

/// A named macOS monospace font rather than the generic `Font::MONOSPACE`.
///
/// With the generic family, iced 0.14 / cosmic-text 0.15 goes through its
/// monospace-fallback path: big.go took ~14 s instead of ~3.7 s to open, and
/// any line containing CJK broke (tofu, the rest of the line dropped, lines
/// overlapping in Markdown code blocks). See NOTES.md.
const CODE_FONT: Font = Font::with_name("Menlo");

enum Doc {
    Code {
        content: text_editor::Content,
        language: String,
    },
    Markdown(markdown::Content),
}

struct App {
    title: String,
    doc: Doc,
    ready: bool,
}

#[derive(Debug, Clone)]
enum Message {
    Edit(text_editor::Action),
    Frame,
    Link,
}

fn boot() -> App {
    let path = std::env::args().nth(1).expect("usage: bench-iced <path>");
    let path = Path::new(&path);
    let title = path
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default();
    let extension = path
        .extension()
        .map(|e| e.to_string_lossy().to_lowercase())
        .unwrap_or_default();
    let text = std::fs::read_to_string(path).expect("read file");

    let doc = if extension == "md" || extension == "markdown" {
        Doc::Markdown(markdown::Content::parse(&text))
    } else {
        Doc::Code {
            content: text_editor::Content::with_text(&text),
            language: if extension.is_empty() { "txt".into() } else { extension },
        }
    };

    App { title, doc, ready: false }
}

fn update(app: &mut App, message: Message) {
    match message {
        Message::Edit(action) => {
            if let Doc::Code { content, .. } = &mut app.doc {
                content.perform(action);
            }
        }
        // `window::frames` is broadcast by iced_winit right after the UI has
        // been drawn for a `RedrawRequested` (and its message reaches `update`
        // only after that redraw has been presented). The file's content is in
        // the state from `boot`, so the very first frame already contained it,
        // and `text_editor` highlights its visible lines inside `draw`.
        Message::Frame => {
            if !app.ready {
                app.ready = true;
                let mut out = std::io::stdout().lock();
                let _ = writeln!(out, "ready");
                let _ = writeln!(out, "highlighted");
                let _ = out.flush();
            }
        }
        Message::Link => {}
    }
}

fn view(app: &App) -> Element<'_, Message> {
    match &app.doc {
        Doc::Code { content, language } => text_editor(content)
            .on_action(Message::Edit)
            .highlight(language, highlighter::Theme::InspiredGitHub)
            .font(CODE_FONT)
            .size(13)
            .height(Length::Fill)
            .into(),
        Doc::Markdown(content) => scrollable(
            container(
                markdown::view(content.items(), markdown_settings()).map(|_| Message::Link),
            )
            .padding(20)
            .width(Length::Fill),
        )
        .width(Length::Fill)
        .height(Length::Fill)
        .into(),
    }
}

fn markdown_settings() -> markdown::Settings {
    let mut settings = markdown::Settings::from(&Theme::Light);
    settings.style.code_block_font = CODE_FONT;
    settings.style.inline_code_font = CODE_FONT;
    settings
}

fn subscription(app: &App) -> Subscription<Message> {
    if app.ready {
        Subscription::none()
    } else {
        window::frames().map(|_| Message::Frame)
    }
}

fn main() -> iced::Result {
    iced::application(boot, update, view)
        .title(|app: &App| app.title.clone())
        .subscription(subscription)
        .theme(|_: &App| Theme::Light)
        .window_size((1000.0, 700.0))
        .run()
}
