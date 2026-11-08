# QuickEdit 与其他编辑器的打开速度对比

同一批文件，分别用 QuickEdit、TextEdit、Sublime Text、CotEditor、Zed、VS Code 打开
（Markdown 另加 Typora），测从"请求打开"到"文件内容稳定显示在窗口里"要多久。

## 场景

- **冷启动**：编辑器没在运行，像在 Finder 里双击文件那样打开它。
- **QuickEdit 常驻**：QuickEdit 以登录项的方式在后台启动、没有窗口，再打开文件（`resident.sh`）。
  对用户来说这和冷启动是同一个场景：双击文件时没有打开的编辑器窗口。别的编辑器默认不在登录时启动。
- **已运行**：编辑器开着（打开过一个别的文件），再打开一个文件。Sublime、Zed、VS Code 会在原窗口里开新标签页，
  TextEdit、CotEditor、Typora、QuickEdit 会开新窗口。

文件：

| 文件 | 内容 |
|---|---|
| `small.go` | Go 标准库 strings.go，1,300 行 |
| `big.go` | net/http 源码拼两遍，10.8 万行，3.1 MB |
| `doc.md` | 中英混排 Markdown，1,332 行，含表格和代码块 |
| `huge.log` | 生成的日志，60 万行，44 MB（`genlog.py`） |

## 怎么测（`measure.swift`）

1. 用 `NSWorkspace.open(_:withApplicationAt:)` 打开文件，和 Finder 双击走同一条路。
2. **窗口出现**：窗口服务器里出现了这个应用、标题里带文件名的窗口。
3. **画面稳定**：之后不断截这个窗口（缩成 320×224 的缩略图比较），取最后一次有超过 1% 像素变化的时间，
   之后 1.5 秒不再变化。闪烁的光标、淡出的滚动条变化太小，不算；文字出现、语法高亮上色、Markdown 渲染都算。
   表里的数字都是这一项。
4. **内存**：画面稳定 2 秒后，这个应用所有进程（含 Electron 的辅助进程）的 physical footprint 之和。

为了公平：

- 测试期间关掉系统的窗口打开动画（`NSAutomaticWindowAnimationsEnabled`），测完恢复。
- 每个编辑器都关掉"恢复上次打开的窗口"，每次冷启动前删掉它保存的窗口状态。
- 每个编辑器先启动一次不计时（系统第一次运行新程序时要检查它）。
- VS Code 用临时的数据目录启动，不装扩展。
- 校验实际运行的是指定路径的应用（同一个 bundle ID 的另一个副本不算）。

## 运行

```sh
swiftc -O measure.swift -o measure
APPS_DIR=下载的编辑器所在目录 ./run.sh 5          # 全部；或 ./run.sh 5 QuickEdit Zed
./resident.sh 5                                   # 后台常驻的 /Applications/QuickEdit.app
python3 summarize.py                              # 中位数表格，results/summary.json
```

`run.sh` 结束时会删除这些编辑器在 `~/Library`、`~/.config` 里新建的目录（`KEEP_EDITOR_DATA=1` 保留）。

结果见 [results/](results/)。
