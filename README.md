# QuickEdit

快速打开、查看、编辑任何文本文件的 macOS 编辑器，用来取代 TextEdit 和 Sublime Text。
用 [MyGo](https://github.com/egoist/mygo)（Go，原生 GPU 绘制，没有 WebView）写成。

- **秒开**：关掉最后一个窗口后进程留在后台，再打开文件只需新建一个窗口。
  10 万行的文件和小文件一样快，因为编辑器只排版看得见的行。
- **按文件自适应**：代码按语言高亮（30 多种语言，没有扩展名的脚本按 `#!` 行识别），
  Markdown 默认渲染预览，可以切到源码或并排显示；二进制文件以十六进制只读显示。
- **编码**：自动识别 UTF-8、UTF-8 BOM、UTF-16、GBK/GB18030，保存时按原编码、原换行符写回。
- **像 Mac 应用一样保存**：自动保存、崩溃后恢复、外部修改提示，文件菜单有复制、重命名、移动、还原。
- 界面跟随系统的深浅色和语言（简体中文 / 英文）。

## 快捷键

| 键 | 作用 |
|---|---|
| ⌘N / ⌘O / ⌘S | 新建 / 打开（任何文件）/ 保存 |
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

`scripts/qe` 的用法和 `subl`、`code` 一样：

```sh
qe notes.md main.go     # 打开文件，不存在的先建空文件
git diff | qe -         # 打开命令的输出
```

## 构建

需要 Go 1.27.3+ 和 Xcode 命令行工具。

```sh
make build     # build/darwin-arm64/QuickEdit.app
make test
make install   # 复制到 /Applications，qe 装到 ~/.local/bin
```

QuickEdit 用的是打了补丁的 MyGo：`scripts/patch-mygo.sh` 把 MyGo v0.8.3 复制到 `.mygo/`，
应用 `patches/mygo/` 下的补丁，`go.mod` 用 `replace` 指向它（`make` 会自动做这一步）。
目前只有一个补丁：中文标点单独测量时会被排成半宽，导致中文行自动换行时超出窗口
（`cjk-wrap.patch`）。上游修复后可以去掉。

## 性能对比

和 TextEdit、Sublime Text、CotEditor、Zed、VS Code、Typora 打开同样的文件（Mac mini M2，5 次中位数，毫秒，
到内容含高亮或渲染稳定显示为止）：

| | QuickEdit | TextEdit | Sublime Text | Zed | VS Code | CotEditor |
|---|---:|---:|---:|---:|---:|---:|
| 冷启动打开 10.8 万行 Go | **406** | 458 | 1,018 | 548 | 1,679 | 979 |
| 冷启动打开 44 MB 日志 | **531** | 776 | 1,699 | 620 | 1,905 | 1,391 |
| 已运行时打开 10.8 万行 Go | 135 | 182 | 492 | **56** | 394 | 1,892 |
| 内存（冷启动打开 10.8 万行后） | 123 MB | 45 MB | 203 MB | 269 MB | 1,349 MB | 225 MB |

冷启动它最快；已经开着时，Zed、Sublime、VS Code 在原窗口开标签页，Zed 最快，QuickEdit 每个文件开新窗口。
测法和全部数据见 [bench/compare/](bench/compare/)。

框架选型时的对比测试在 [bench/](bench/)：同一个"打开文件"的小程序分别用 MyGo、GPUI、iced、egui 和原生 AppKit 实现，
结果见 [bench/RESULTS.md](bench/RESULTS.md)。
