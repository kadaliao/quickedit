# bench-iced

iced 版的"打开一个文件"应用，约定见 `../README.md`。

## 构建

```sh
export PATH="$HOME/.cargo/bin:$PATH"
cd bench/iced && cargo build --release   # → target/release/bench-iced
```

- Rust 1.99.1，release：`lto = "fat"`、`codegen-units = 1`、`strip = true`
- 二进制 9,049,440 字节（8.6 MiB）
- 干净构建（依赖已下载）：86 s 墙钟 / 240 s user（M2 Mac mini，同时有别的 agent 在编译，有干扰）

## 版本（Cargo.lock）

iced 0.14.0（crates.io 最新稳定版，features `highlighter` + `markdown`，其余默认）、
iced_widget 0.14.2、iced_winit 0.14.1、iced_wgpu 0.14.0、iced_graphics 0.14.0、
iced_highlighter 0.14.0、cosmic-text 0.15.0、harfrust 0.3.2、fontdb 0.23.0、
syntect 5.3.0（经 two-face 0.4.5，fancy-regex）、pulldown-cmark 0.12.2、wgpu 27.0.1、winit 0.30.13。

## 实现

- 非 `.md`：`text_editor(&Content)`，`.highlight(扩展名, highlighter::Theme::InspiredGitHub)`，
  字体 `Font::with_name("Menlo")`，13 px，`Length::Fill`，可编辑（`on_action` → `Content::perform`）。
  `Content::with_text` 在 `boot` 里同步构造。
- `.md`：`markdown::Content::parse` + `markdown::view`，外面套 `scrollable`；代码块和行内代码字体也设成 Menlo。
- 应用主题 `Theme::Light`，窗口 1000×700，标题 = 文件名，多余参数忽略。

### 为什么用具名字体 Menlo 而不是 `Font::MONOSPACE`

一开始用的是 iced 示例里常见的 `Font::MONOSPACE`（泛型 monospace family），结果：

- big.go 要 ~13–14 s 才 ready（同一台机器，Menlo 只要 ~3.3 s）。采样显示时间花在 cosmic-text 的
  monospace 回退路径（`FontFallbackIter`、`MonospaceFallbackInfo`）上。
- 含中文的行全坏：编辑器里中文是方框，字符串里含中文的那一行后半截直接消失、留下大段空白；
  Markdown 代码块里含中文的行互相重叠、中文丢失。

换成具名字体后这两个问题都没了，所以交付的是 Menlo 版本。真实的 macOS 编辑器本来也会指定具体字体。

## `ready` / `highlighted` 怎么判断

- 订阅 `window::frames()`。看 iced_winit 0.14.1 的 `RedrawRequested` 处理：先 `interface.draw(...)`，
  再 `runtime.broadcast(RedrawRequested)`，再 `compositor.present(...)`。`frames()` 就是听这个广播，
  它产生的消息要经过订阅流 → event loop proxy 再回到 `update`，所以 `update` 收到时，
  这次重绘（含 present）已经做完。
- 文件内容在 `boot` 里就放进了状态，第一帧的 view 里就有内容，所以第一个 `Frame` 消息到达时打印 `ready`。
- 高亮在第一帧里：`text_editor` 在 `draw` 里同步调用 `Editor::highlight`，把从高亮器当前行到最后一个可见行
  都高亮好再画（iced_graphics `text/editor.rs`）；Markdown 代码块在 `parse` 时就高亮了。
  所以 `ready` 之后紧跟着打印 `highlighted`。截图确认了第一屏是带颜色的。
- 打印后取消 `frames()` 订阅，之后空闲（CPU 0%）。

## 粗测结果（本机，非正式 runner）

从启动进程到读到 `ready` 的时间，3 次，系统负载 ~3–4；内存是 ready 后 2 s 的 `footprint`。

| 文件 | ready（= highlighted） | footprint |
|---|---|---|
| small.go | 242 / 276 / 291 ms | 70 MB |
| doc.md | 381 / 460 / 448 ms | 152 MB |
| big.go | 3235 / 3321 / 3282 ms | 1100 MB |

- 新编译出来的二进制第一次启动会多 0.7–6 s（macOS 对新二进制的首次安全扫描，trustd 很忙），正式测量前要先跑一次热身。
- 第一次测 big.go 时是 40 s：那会儿同时在编译 egui 和 GPUI，CPU 被占满，而且还是 `Font::MONOSPACE` 版本。

## 注意事项 / 能力缺口

- **大文件**：`Content::with_text` 用 `Buffer::new_empty`（没有高度）调 `set_text`，cosmic-text 会把
  全部 108k 行都 shape 一遍（boot 里约 3.7 s，非 LTO 构建测的）；第一次 layout 字体从默认换成 Menlo，
  又把每行重置重排一遍。所以打开 3 MB 文件要 ~3.3 s、占 1.1 GB。加载完以后滚动是流畅的：
  每帧滚 3 行时，帧间隔中位数约 7 ms（非 LTO 构建，`window::frames` 计时）。
  `Wrapping::None` 和默认的 word wrap 差别不大（`Font::MONOSPACE` 下 15.3 s 对 16.0 s）。
- **斜体中文是方框**：InspiredGitHub 主题把注释设成斜体，iced_highlighter 遇到斜体会换成默认 family 的 Italic，
  cosmic-text 不会为斜体中文回退到 PingFang，于是**中文注释显示成方框**（中文字符串正常）。
  换成不带斜体的主题（试过 `Base16Ocean`）就正常了，但内置主题里只有 InspiredGitHub 是浅色的。
  测试文件全是 ASCII，不影响这次的测量。
- **没有行号**：`text_editor` 没有 gutter API。编辑器自己管滚动，外面再拼一列行号也没法跟它同步滚动。
- **IME**：iced 0.14 的 text_editor 有输入法支持（winit 层会 `request_input_method`、`draw_preedit`），这里没有实测。
- **Markdown 质量**：标题、表格（中文对齐正常）、任务列表、引用、带高亮的代码块都能渲染，中文显示正常。
  原始 HTML（`<img …>`）不显示；`**结论。**后文` 没有加粗，这是 CommonMark 的 flanking 规则，egui 那边也一样。
  图片需要开 `image` feature，这里没开。
- 默认 features 带 tiny-skia 软件渲染兜底和 x11/wayland（macOS 上用不到），没有去掉，按普通应用的用法来。
