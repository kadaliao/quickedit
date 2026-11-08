# bench-gpui：GPUI + gpui-component

```sh
export PATH="$HOME/.cargo/bin:$PATH"
cargo build --release          # → target/release/bench-gpui
target/release/bench-gpui ../files/small.go -_NS_4445425547 NO
```

## 依赖与版本（2026-11-08，全部来自 crates.io，无 git 依赖）

| crate | 版本 | 说明 |
|---|---|---|
| `gpui-kit` | 0.9.0 | longbridge 的总入口 crate（仓库已改名 longbridge/gpui-kit），默认特性 `component` + `assets`，另开 `tree-sitter-go` |
| `gpui-component` | 0.9.0 | 代码编辑器 `Editor`/`EditorState`、Markdown `TextView` |
| `gpui-base` | 0.9.0 | 编辑器、富文本的底层实现 |
| `gpui-kit-assets` | 0.9.0 | 组件用到的 101 个 Lucide 图标 |
| `gpui-pre` (+ `gpui-pre-platform`, `gpui-pre-macos` …) | 0.3.11 | GPUI 本体，“gpui-pre snapshot of zed@f816e1f”，由 gpui-kit 用 `=0.3.11` 钉死 |
| `tree-sitter` / `tree-sitter-go` | 0.26.13 / 0.23.4 | 语法高亮 |
| `markdown` | 1.0.0 | Markdown 解析（markdown-rs） |
| `ropey` | 2.0.0-beta.1 | 编辑器文本存储 |

rustc 1.99.1。Cargo.lock 共 804 个包。

**为什么不是 crates.io 上的 `gpui = "0.2.2"`**：它比 Zed 主干落后约 11 个月，gpui-component 的
已发布版本早就不基于它了。gpui-component 0.9.0 依赖的是 `gpui-pre =0.3.11`——一组从 Zed 主干
快照发布到 crates.io 的 crate。照 gpui-kit README 的推荐只依赖 `gpui-kit`，它带上配套的
gpui-pre、平台层、gpui-base、gpui-component 和图标，不需要 git 依赖或 `[patch]`。
`gpui-fast` 特性（另一个 GPUI 分支）没有打开。

只编进了 Go 语法（tree-sitter-json 是 gpui-component 的必选依赖，也在里面）。doc.md 里
38 个 go 代码块有高亮，另外 rust/sh/ts 共 5 个代码块是纯文本。真要发布的编辑器会开
`tree-sitter-languages`，二进制会明显变大（未测）。

## 应用做了什么

- `.md`：`TextViewState::markdown` + `TextView::new(&state).scrollable(true).selectable(true)`，
  四周 24/16 px 内边距。
- 其他文件：`EditorState::new(..).language("go").soft_wrap(false).default_value(text)`，
  用 `Editor::new(&state)` 铺满窗口，启动后编辑器获得焦点（可直接打字，光标闪烁）。
  行号、缩进参考线、代码折叠、搜索面板都是 `EditorState` 的默认设置。
- 关掉了自动换行（soft wrap）。gpui-component 默认开着；Sublime、Zed、VS Code 对代码都默认关。
  开着时，big.go 的 10.8 万行要在第一帧里全部算换行，ready 从约 265 ms 涨到约 1000 ms
  （small.go 不受影响）。
- 菜单只有 Quit（⌘Q），关窗即退出。
- 文件在 `main` 里同步读取（3.1 MB 约 1 ms）。不是 UTF-8 时按 lossy 转换。

## ready / highlighted 怎么判断

`ready` 和参考实现一样用 `window.on_next_frame`：在某一帧的 render 里注册，回调在下一次
帧请求开头执行，也就是那一帧 present 之后。

**代码文件：第一帧没有高亮，`highlighted` 晚于 `ready` 单独打印。** 依据：

1. 源码：`gpui-component/src/highlighter/input_adapter.rs` 在编辑器第一次 render 时同步跑
   tree-sitter，但限时 2 ms（`SYNC_PARSE_TIMEOUT`），文本超过 256 KB 时干脆不同步解析。
   超时或超限就转到后台：先等 150 ms（`PARSE_DEBOUNCE`），再在后台线程完整解析，结果回到
   主线程后装进高亮器并刷新编辑器。34 KB 的 small.go 在 M2 上 2 ms 解析不完，所以两个 Go
   文件都走后台。
2. 判断“树到了没有”：高亮器装树的同时会把折叠范围交给编辑器（同步路径在
   `update_fold_candidates`，后台路径在 `apply_highlighter_fold_candidates`）。应用检查前
   2000 行里有没有折叠候选（`display_map().is_fold_candidate`），Go 文件只要有多行函数就有。
   `ready` 回调里如果已经有树，说明第一帧就高亮了，紧接着打印 `highlighted`；否则观察编辑器
   实体，树一到就让根视图重绘，并在那一帧上注册 `on_next_frame` 打印 `highlighted`。
3. 截图核实：`BENCH_FREEZE=ready` / `BENCH_FREEZE=highlighted` 会在打印该行后 40 ms
   阻塞主线程，之后不会再画新帧，截到的就是那一帧。结果：ready 帧是完整的纯文本（有行号、
   没颜色），highlighted 帧是彩色。（打印后立即阻塞不行：窗口根本不会上屏，AppKit 把窗口
   放上屏幕也要主 run loop 转一圈。所以“首帧 present”和“窗口上屏”确实是两回事，runner
   取两者较晚的那个是对的。）

**Markdown：内容和高亮在同一帧，`ready` 后紧跟 `highlighted`。** 超过 4 KB 的 Markdown
（`MAX_SYNC_FULL_REPLACE_BYTES`）在后台线程解析，前几帧是空的。应用观察 `TextViewState`，
等 `rendered_text().source()` 非空，在那一帧注册 `on_next_frame`。代码块高亮是在布局代码块时
同步算的（`component_code_block_highlighter`，不限时），所以内容出来的那一帧已经带高亮。
冻结截图核实了这一点。

## 粗测结果（M2 Mac mini，release，从 spawn 算起；另一个 agent 同时在编译，机器有负载）

先把第一次冷启动（新二进制要换页进内存，约 1.4 s）扔掉，剩下 5–6 次取中位数：

| 文件 | ready | highlighted | footprint（highlighted 后 2 s） |
|---|---|---|---|
| small.go | ~255 ms | ~475 ms | 79 MB |
| big.go | ~265 ms | ~790 ms | 189 MB |
| doc.md | ~395 ms | 同 ready | 81 MB |

空闲 CPU：编辑器约 7 ms/s（光标闪烁），Markdown 约 1 ms/s。

`BENCH_TRACE=1` 时间线（small.go，从 `main` 起）：NSApplication 启动完 ~60 ms →
`gpui_kit::init` 完 ~140 ms（这一步约 85 ms，没细查花在哪）→ 窗口和视图建好 ~235 ms →
第一帧画完 ~245 ms → ready ~258 ms。big.go 的第一帧约 30 ms（关自动换行时）。高亮再多花：
small.go 约 150 ms 等待 + 解析；big.go 约 150 ms 等待 + 约 400 ms 后台全量解析 3.1 MB。

二进制 14,890,528 字节（14.2 MiB，已 strip）。在这台机器上（同时有另一个编译）干净的
release 构建用了 5 分 49 秒（依赖已下载）；只改 main.rs 后重新链接（fat LTO）约 1.5–2 分钟。

## 作为编辑器的能力和问题

- **大文件**：`EditorState` 的文档说它的定位是“简单代码编辑/展示……能处理约 5 万行的大文本”。
  108K 行的 big.go 关掉自动换行后打开不慢，但开着自动换行要多约 0.7 s。超过 256 KB 的文本，每次
  编辑都要等 150 ms 再在后台重新解析（传入旧树，增量解析），这段时间高亮可能是旧的。
  内存：3.1 MB 文件 189 MB footprint。滚动、打字延迟没有测。
- **高亮**：tree-sitter，质量好（关键字、字符串、注释、函数名、类型都分了颜色）。但第一帧永远
  没颜色，小文件也要约 200 ms 后才变色，肉眼能看到一闪。Go 的 tab 按 2 列显示（默认
  tab size 2）。
- **Markdown**：标题、任务列表（复选框）、表格（有边框和表头底色）、引用、代码块（右上角显示
  语言名）、行内代码、粗体、列表、分隔线都正常，中英混排排得好。问题：
  - `**结论。**后文` 显示成字面的 `**`。这是 CommonMark 的定界符规则（闭合的 `**` 前面是标点、
    后面紧跟汉字，不算右定界），markdown-rs 严格照规范来；GitHub 也一样，但中文用户会觉得是 bug。
  - 原始 HTML 块（`<img>`、`<picture>`、`<a>`）被直接丢掉：没有图片，也没有 alt 文本。
  - 只编进了 Go 语法，所以 rust/sh/ts 代码块没有颜色。
  - 文本可以选择，但 Markdown 视图是只读的。“源码视图”要另外用一个 `EditorState`（可以开
    `tree-sitter-markdown`），这里没做。
- **输入法**：编辑器实现了 GPUI 的 `EntityInputHandler`（NSTextInputClient 那一套，带 marked
  text），设计上支持中文输入法，但这次没有手动试。
- **其他**：右键菜单、查找（⌘F 面板）、撤销/重做、多光标、折叠、LSP 钩子都是 `EditorState`
  自带的。字体用主题的等宽字体（macOS 上默认 Menlo）。`gpui_component::init` 固定用浅色主题，
  不跟随系统深色模式，要跟随得应用自己切换主题（这里没做）。

## 测试钩子（runner 不用）

- `BENCH_TRACE=1`：把启动各阶段的时间打到 stderr。
- `BENCH_FREEZE=ready[:ms]` / `BENCH_FREEZE=highlighted[:ms]`：打印该行后 ms（默认 40）毫秒
  阻塞主线程，用来截“就是那一帧”的图。
