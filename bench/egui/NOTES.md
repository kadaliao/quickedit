# bench-egui

egui（eframe）版的"打开一个文件"应用，约定见 `../README.md`。

## 构建

```sh
export PATH="$HOME/.cargo/bin:$PATH"
cd bench/egui && cargo build --release   # → target/release/bench-egui
```

- Rust 1.99.1，release：`lto = "fat"`、`codegen-units = 1`、`strip = true`
- 二进制 11,372,256 字节（10.8 MiB）
- 干净构建（依赖已下载）：102 s 墙钟 / 292 s user（M2 Mac mini，同时有别的 agent 在编译，有干扰；
  第一次在重负载下构建用了 6 分 48 秒）

## 版本（Cargo.lock）

eframe 0.37.0（默认 features：wgpu、accesskit、system_fonts 等）、egui 0.37.0、epaint 0.37.0、
egui_extras 0.37.0（feature `syntect`）、egui_commonmark 0.26.0（默认 features + `better_syntax_highlighting`）、
egui_system_font_provider 0.37.0（fontique 0.11.1）、syntect 5.3.0（fancy-regex 0.16.2）、
pulldown-cmark 0.13.6、accesskit_macos 0.27.2、wgpu 30.0.1、winit 0.30.13。都是 crates.io 上的最新稳定版。

## 实现

- 非 `.md`：`TextEdit::multiline(&mut String).code_editor().layouter(...)`，放在 `ScrollArea::both()` 里，
  layouter 调 `egui_extras::syntax_highlighting::highlight(ctx, style, &CodeTheme::light(13.0), text, "go")`，
  这和 egui demo 里 code editor 的写法一样。可编辑。
- 为什么不用 egui_code_editor 0.5：它有行号，但用的是自带的分词器，只支持 asm/lua/python/rust/shell/sql，
  **不支持 Go**，也不是 syntect。egui_extras + syntect 才是 egui 的标准高亮路线。
- `.md`：`egui_commonmark::CommonMarkViewer::new().show(ui, &mut cache, text)`，放在 `ScrollArea::vertical()` 里（README 的用法）。
- `ctx.set_theme(Theme::Light)`，窗口 1000×700，标题 = 文件名，多余参数忽略。

### 中文字体

不用手动加载 PingFang。eframe 0.37 默认开 `system_fonts`，会装上 `egui_system_font_provider`：
启动时在后台线程用 fontique 枚举系统字体，遇到 egui 自带字体里没有的字时走 CoreText 的回退，
同步查找（如果枚举还没完成就阻塞等待），找到的字体用 mmap 映射进来。所以含中文的那一帧里中文就是对的，截图确认过。
代价：doc.md 和 small.go 的 ready 时间都是 ~250 ms，在本机测不出差别（枚举和 wgpu 初始化是并行的）。

## `ready` / `highlighted` 怎么判断

- 和 MyGo 仓库 `benchmarks/egui` 的做法一样：第 1 帧在 `ui()` 里排版完整个文件，再 `request_repaint()`；
  eframe 在 `ui()` 返回后才绘制、提交这一帧，所以第 2 帧开始时，第 1 帧已经画完，这时打印 `ready`。
- 高亮在第 1 帧里：`egui_extras::syntax_highlighting::highlight` 在 layouter 里**同步**把整个文件高亮成一个
  `LayoutJob`（结果放在 `FrameCache` 里缓存），所以 `ready` 之后紧跟着打印 `highlighted`。
  Markdown 代码块也是同步高亮的。截图确认第一屏有颜色。

## 粗测结果（本机，非正式 runner）

从启动进程到读到 `ready` 的时间，3 次，系统负载 ~3–4；内存是 ready 后 2 s 的 `footprint`。

| 文件 | ready（= highlighted） | footprint |
|---|---|---|
| small.go | 263 / 230 / 252 ms | 88 MB |
| doc.md | 246 / 246 / 252 ms | 116 MB |
| big.go | 3658 / 3642 / 3650 ms | 1006 MB |

新编译出来的二进制第一次启动会多 0.6–1.5 s（macOS 对新二进制的首次扫描），正式测量前要先跑一次热身。

## 注意事项 / 能力缺口

- **大文件，启动**：big.go 的 3.6 s 里约 3 s 是 syntect（fancy-regex）在第 1 帧里同步高亮全部 108k 行（采样确认）。
  整个文件被排成一个 galley，所以占 ~1 GB。
- **大文件，每帧**：之后只要重绘（滚动、获得焦点时光标闪烁、鼠标悬停），每帧就要 **~300 ms CPU**（约 3 fps）。
  采样显示其中约 60% 在 `egui_winit::State::handle_platform_output` → `accesskit_macos` 更新：
  这台机器上有辅助功能客户端在运行，AccessKit 处于激活状态，每帧都把 3 MB 的 TextEdit 内容推进无障碍树。
  另外约 35% 在 `TextEdit::ui`：高亮缓存每帧返回 3 MB `LayoutJob` 的克隆，galley 缓存每帧还要对它做哈希。
  打字的话，每按一个键都会把整个文件重新高亮一遍（几秒）。真要做 QuickEdit，大文件得自己写虚拟化编辑器、
  逐行 galley、后台高亮。
- **小文件**：small.go 每帧滚 3 行时，每帧 CPU 约 2.4 ms，没问题。
- **Markdown 每帧**：`CommonMarkViewer::show` 每帧都要重新解析、重新排版整个文档，doc.md 滚动时
  每帧约 22 ms CPU，掉到 60 fps 以下（其中多少来自 AccessKit 没有拆开测）。crate 提供的办法是
  `show_scrollable` + `use_viewport_cache`，这里没用，也没测。
- **Markdown 质量**：中文、表格（中文对齐正常）、任务列表、引用、带高亮的代码块（含中文）都正常。
  缺点：标题字号偏小，层次不明显；代码块右上角的"复制"图标显示成方框（这个 emoji 字形没有，system font provider 跳过 emoji）；
  原始 HTML（`<img …>`、`</p>`）原样显示成文字；`**结论。**后文` 没有加粗（CommonMark 规则，iced 也一样）。
- **中文代码**：Go 文件里的中文注释和字符串在 TextEdit 里显示正常（另外拿一个小文件测的）。
- **没有行号**：TextEdit 没有 gutter，要自己在旁边拼一列（egui_code_editor 就是这么做的）。
- **IME**：TextEdit 会处理 winit 的 IME 事件，这里没有实测。
- Tab 按 4 个空格宽显示。
