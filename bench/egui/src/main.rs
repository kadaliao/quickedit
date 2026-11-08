//! egui's (eframe's) "open a file" app for the QuickEdit comparison (see
//! ../README.md).
//!
//! `bench-egui <path> [ignored...]`: a 1000×700 window titled with the file's
//! name, showing rendered Markdown (egui_commonmark) for `.md` files and an
//! editable `TextEdit::multiline` with egui_extras' syntect highlighting, via
//! a layouter, for anything else.

use std::io::Write;
use std::path::Path;

use eframe::egui;
use egui_commonmark::{CommonMarkCache, CommonMarkViewer};
use egui_extras::syntax_highlighting::{CodeTheme, highlight};

const CODE_SIZE: f32 = 13.0;

enum Doc {
    Code { text: String, language: String },
    Markdown { text: String, cache: CommonMarkCache },
}

struct App {
    doc: Doc,
    frames: usize,
}

impl eframe::App for App {
    fn ui(&mut self, root: &mut egui::Ui, _frame: &mut eframe::Frame) {
        // As in MyGo's benchmarks/egui: eframe draws and presents a frame after
        // `ui` returns, so the start of the second frame is the first point at
        // which the first frame (which already lays out the whole file, and
        // highlights it synchronously) has been drawn.
        self.frames += 1;
        if self.frames == 1 {
            root.ctx().request_repaint();
        } else if self.frames == 2 {
            let mut out = std::io::stdout().lock();
            let _ = writeln!(out, "ready");
            let _ = writeln!(out, "highlighted");
            let _ = out.flush();
        }

        match &mut self.doc {
            Doc::Code { text, language } => {
                let theme = CodeTheme::light(CODE_SIZE);
                let language = language.as_str();
                let mut layouter = |ui: &egui::Ui, buf: &dyn egui::TextBuffer, wrap_width: f32| {
                    let mut job = highlight(ui.ctx(), ui.style(), &theme, buf.as_str(), language);
                    job.wrap.max_width = wrap_width;
                    ui.fonts_mut(|f| f.layout_job(job))
                };
                egui::CentralPanel::default()
                    .frame(egui::Frame::central_panel(root.style()).inner_margin(0.0))
                    .show(root, |ui| {
                        egui::ScrollArea::both().auto_shrink(false).show(ui, |ui| {
                            ui.add(
                                egui::TextEdit::multiline(text)
                                    .font(egui::FontId::monospace(CODE_SIZE))
                                    .code_editor()
                                    .lock_focus(true)
                                    .desired_width(f32::INFINITY)
                                    .layouter(&mut layouter),
                            );
                        });
                    });
            }
            Doc::Markdown { text, cache } => {
                egui::CentralPanel::default().show(root, |ui| {
                    egui::ScrollArea::vertical().auto_shrink(false).show(ui, |ui| {
                        CommonMarkViewer::new().show(ui, cache, text);
                    });
                });
            }
        }
    }
}

fn main() -> eframe::Result {
    let path = std::env::args().nth(1).expect("usage: bench-egui <path>");
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
        Doc::Markdown { text, cache: CommonMarkCache::default() }
    } else {
        Doc::Code { text, language: extension }
    };

    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_title(title.clone())
            .with_inner_size([1000.0, 700.0]),
        ..Default::default()
    };
    eframe::run_native(
        &title,
        options,
        Box::new(move |cc| -> Result<Box<dyn eframe::App>, _> {
            cc.egui_ctx.set_theme(egui::Theme::Light);
            Ok(Box::new(App { doc, frames: 0 }))
        }),
    )
}
