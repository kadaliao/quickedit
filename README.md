<p align="center">
  <img src="resources/icon.png" width="128" height="128" alt="QuickEdit 图标">
</p>

<h1 align="center">QuickEdit</h1>

<p align="center">
  <b>极致快速的轻量编辑器，为灵活编辑而生。</b><br>
  看代码、改配置、读 Markdown、翻日志：打开就用，用完就走。<br>
  同一标准实测，三种情况下打开文件的总用时都比 Zed、Sublime Text、VS Code、CotEditor 短。
</p>

<p align="center">
  <a href="https://github.com/kadaliao/quickedit/releases/latest">下载</a> ·
  <code>brew install --cask kadaliao/tap/quickedit</code> ·
  macOS 13+，Apple 芯片
</p>

<p align="center">
  <img src="docs/images/screenshot-markdown.png" width="860" alt="QuickEdit 并排显示一篇 Markdown 的源码和渲染结果">
</p>

## 为什么快

- **原生，轻**：原生 GPU 绘制（[MyGo](https://github.com/egoist/mygo)），没有 Electron，没有 WebView，应用 12 MB。
  冷启动打开 10.8 万行的 Go 文件 0.5 秒，比 Zed、Sublime Text、CotEditor、VS Code 都快。
- **只排版看得见的行**：文件再大，第一屏也一样快；大文件多核并行读入，60 万行、44 MB 的日志冷启动 0.6 秒打开。
- **开着窗口时开成标签页**：不用新建窗口，10.8 万行的 Go 文件 68 毫秒。
- **提前备好窗口**：关掉所有窗口后再打开文件，直接用预先建好的窗口，同一个文件 135 毫秒。
- **命令行直达**：`qe` 经本地 socket 把文件交给运行中的 QuickEdit，不经过系统转发。

## 速度对比

同一台 Mac（Apple M2，macOS 26），一次测完：QuickEdit、Sublime Text、Zed、CotEditor、VS Code 依次在**同样的三种状态**下，
打开**同样的四个文件**，各测 3 次取中位数。计时从请求打开（与在 Finder 里双击走同一条路），到内容（含语法高亮）显示稳定、
窗口到了最前面为止，越短越好。Markdown 大家都显示带高亮的源码。

**打开 10.8 万行的 Go 文件（3.1 MB）**

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/bench-bigfile-dark.svg">
  <img src="docs/images/bench-bigfile-light.svg" width="100%" alt="打开 10.8 万行的 Go 文件：冷启动 QuickEdit 517 毫秒，第二名 Zed 730；运行中无窗口 QuickEdit 135 毫秒，第二名 CotEditor 438；运行中有窗口 QuickEdit 68 毫秒，第二名 VS Code 448">
</picture>

三种状态都是 QuickEdit 第一：冷启动 517 毫秒（第二名 Zed 730），运行中无窗口 135 毫秒（第二名 CotEditor 438），
运行中有窗口 68 毫秒（第二名 VS Code 448）。

**依次打开四个文件的总时间**

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/bench-files-dark.svg">
  <img src="docs/images/bench-files-light.svg" width="100%" alt="依次打开四个文件的总时间：冷启动 QuickEdit 2,214 毫秒，Zed 2,819；运行中无窗口 QuickEdit 625 毫秒，Zed 1,779；运行中有窗口 QuickEdit 321 毫秒，Zed 1,340">
</picture>

三种状态都是 QuickEdit 最短：冷启动 2.2 秒（Zed 2.8 秒），运行中无窗口 0.6 秒（Zed 1.8 秒），运行中有窗口 0.3 秒（Zed 1.3 秒）。

逐个文件看（下表），12 项里 QuickEdit 9 项第一。另外 3 项都是 1,300 行左右的小文件：运行中无窗口时打开小 Go 文件和 Markdown，
Sublime Text 分别快 58 和 19 毫秒；运行中有窗口时打开小 Go 文件，Zed 快 13 毫秒。

**体积和内存**

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/bench-footprint-dark.svg">
  <img src="docs/images/bench-footprint-light.svg" width="100%" alt="应用体积：QuickEdit 12 MB，其他 129 到 963 MB；冷启动打开 10.8 万行 Go 文件后的内存：QuickEdit 124 MB 最少，其他 197 到 1,370 MB">
</picture>

<details>
<summary>全部数据（毫秒，3 次中位数，粗体为每行最快）</summary>

文件：`small.go` 为 1,300 行的 Go 源码；`big.go` 为 10.8 万行、3.1 MB 的 Go 源码；`doc.md` 为 1,332 行中英混排的 Markdown；
`huge.log` 为 60 万行、44 MB 的日志。

**冷启动**：编辑器没在运行。

| | QuickEdit | Sublime Text | Zed | CotEditor | VS Code |
|---|---:|---:|---:|---:|---:|
| small.go | **531** | 616 | 623 | 1,115 | 1,739 |
| big.go | **517** | 1,016 | 730 | 1,289 | 2,066 |
| doc.md | **565** | 685 | 698 | 1,183 | 2,232 |
| huge.log | **601** | 1,712 | 768 | 1,558 | 2,508 |

**运行中，无窗口**：打开过文件，窗口已关。

| | QuickEdit | Sublime Text | Zed | CotEditor | VS Code |
|---|---:|---:|---:|---:|---:|
| small.go | 180 | **122** | 203 | 344 | 1,400 |
| big.go | **135** | 464 | 1,157 | 438 | 1,539 |
| doc.md | 134 | **115** | 188 | 313 | 1,606 |
| huge.log | **176** | 1,166 | 231 | 865 | 1,894 |

**运行中，有窗口**：编辑器开着一个文件。

| | QuickEdit | Sublime Text | Zed | CotEditor | VS Code |
|---|---:|---:|---:|---:|---:|
| small.go | 70 | 148 | **57** | 429 | 1,264 |
| big.go | **68** | 478 | 1,074 | 457 | 448 |
| doc.md | **67** | 182 | 72 | 494 | 425 |
| huge.log | **116** | 1,209 | 137 | 901 | 1,058 |

</details>

测法：

- **三种状态**：冷启动是编辑器没在运行；运行中无窗口是编辑器打开过一个文件、窗口已关（macOS 上的编辑器关掉最后一个窗口通常不退出），
  每个编辑器都一样；运行中有窗口是编辑器开着一个文件，新文件开成标签页（CotEditor 开新窗口）。
- **显示同样的内容**：Markdown 都显示源码（QuickEdit 测试时关掉了"打开 Markdown 时显示预览"）；谁都不开语言服务器。
  Zed 打开了"信任所有项目"：否则它打开每个文件都弹 Restricted Mode 对话框、压暗编辑区，语法高亮就计不进去。
- **为了公平**：测试期间关掉系统的窗口打开动画；关掉每个编辑器恢复上次窗口的功能；VS Code 用临时数据目录、不装扩展；
  每个编辑器先启动一次不计时。TextEdit 不参加：它不做语法高亮。
- 版本：Sublime Text 4（build 4215）、CotEditor 7.1.3、Zed 1.27.0、VS Code 1.145.2。2026-11-08 测试，脚本 `bench/compare/fair.sh`，
  原始数据 `bench/compare/results/fair.jsonl`，见 [bench/compare/](bench/compare/)。

## 按文件自适应

<p align="center">
  <img src="docs/images/screenshot-code.png" width="860" alt="QuickEdit 显示带语法高亮的 Go 源码">
</p>

- **代码**按语言高亮，30 多种语言；没有扩展名的脚本按 `#!` 行识别。
- **Markdown** 默认显示渲染结果，可以切到源码或并排显示。
- **标签页**：窗口开着时，新打开的文件成为它的标签页；两个以上才显示标签栏。
- **二进制文件**以十六进制只读显示。
- **编码**：自动识别 UTF-8、UTF-8 BOM、UTF-16、GBK/GB18030，保存时按原编码、原换行符写回。
- **像 Mac 应用一样保存**：自动保存、崩溃后恢复、外部修改提示，文件菜单有复制、重命名、移动、还原。
- 界面跟随系统的深浅色和语言（简体中文 / 英文）。

## 安装

**Homebrew**（同时装好 `qe` 命令）：

```sh
brew install --cask kadaliao/tap/quickedit
```

**手动**：从 [Releases](https://github.com/kadaliao/quickedit/releases/latest) 下载 DMG，把 QuickEdit 拖进「应用程序」。
应用只做了 ad-hoc 签名，没有经过 Apple 公证：第一次打开如果被拦，到「系统设置 › 隐私与安全性」点「仍要打开」，
或者运行 `xattr -dr com.apple.quarantine /Applications/QuickEdit.app`。要用 `qe` 命令：

```sh
sudo mkdir -p /usr/local/bin
sudo ln -sf /Applications/QuickEdit.app/Contents/Resources/qe /usr/local/bin/qe
```

想让每天第一次打开文件也不用冷启动，可以在 QuickEdit 菜单里勾选"登录时在后台启动"（默认关闭）。

## 快捷键

| 键 | 作用 |
|---|---|
| ⌘N / ⌘O / ⌘S | 新建 / 打开（任何文件）/ 保存 |
| ⌘W / ⇧⌘W | 关闭标签页 / 关闭窗口 |
| ⇧⌘[ / ⇧⌘] | 上一个 / 下一个标签页 |
| ⌘E | Markdown：切换预览 / 源码 |
| ⌥⌘E | Markdown：并排预览 |
| ⌥Z | 自动换行 |
| ⌘= / ⌘- / ⌘0 | 放大 / 缩小 / 实际大小 |
| ⌘F / ⌥⌘F / ⌃G | 查找 / 替换 / 跳到行 |
| ⌘D / ⇧⌘L | 选中下一个相同内容 / 全部（多光标） |
| ⌘/ | 注释 |
| ⌥↑ / ⌥↓ | 上下移动行 |

编辑键位与 Zed 相同，完整列表见 MyGo 的[代码编辑器文档](https://github.com/egoist/mygo/blob/main/docs/ui/code-editor.md#editing)。

## 命令行

`qe` 的用法和 `subl`、`code` 一样：

```sh
qe notes.md main.go     # 打开文件，不存在的先建空文件
git diff | qe -         # 打开命令的输出
```

QuickEdit 在运行时，`qe` 经本地 socket 把文件交给它，比双击再快约 20 毫秒；没在运行时就用 `open` 启动它。

## 构建

需要 Go 1.27.3+ 和 Xcode 命令行工具。

```sh
make build     # build/darwin-arm64/QuickEdit.app
make test
make dmg       # 安装包
make install   # 复制到 /Applications，qe 装到 ~/.local/bin
```

QuickEdit 用的是打了补丁的 MyGo：`scripts/patch-mygo.sh` 把 MyGo v0.8.3 复制到 `.mygo/`，
应用 `patches/mygo/` 下的补丁，`go.mod` 用 `replace` 指向它（`make` 会自动做这一步）。
两个补丁：

- `cjk-wrap.patch`：中文标点单独测量时会被排成半宽，导致中文行自动换行时超出窗口。上游修复后可以去掉。
- `document-tabs.patch`：让多个文档共用一个窗口（标签页）：文档可以挂到窗口上、从窗口上取下，关闭时可以不关窗口。

框架选型时的对比测试在 [bench/](bench/)：同一个"打开文件"的小程序分别用 MyGo、GPUI、iced、egui 和原生 AppKit 实现，
结果见 [bench/RESULTS.md](bench/RESULTS.md)。

## 许可证

[MIT](LICENSE)
