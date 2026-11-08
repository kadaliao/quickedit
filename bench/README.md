# 编辑器框架启动速度对比

目的：为 QuickEdit（一个取代 TextEdit / Sublime 的快速查看、编辑器）选 UI 框架。
比的是"打开一个文件，到窗口里看到带高亮的内容"要多久，以及内存和包体积。

候选：MyGo（Go）、GPUI + gpui-component（Rust，Zed 的框架）、iced（Rust）、egui（Rust），
外加一个原生 AppKit（Swift + NSTextView，不做高亮）作为速度下限参照。

## 每个应用都要遵守的约定

- 命令行：`<app> <文件路径> [其余参数忽略]`。runner 会在路径后面追加 `-_NS_4445425547 NO`
  （关掉 AppKit 的调试菜单），应用必须忽略多余参数。
- 打开一个 1000×700（逻辑点）的窗口，标题是文件名。
- 内容：
  - `.md` 文件：渲染后的 Markdown（标题、列表、表格、代码块），可以只读，可滚动，从顶部开始显示。
  - 其他文件：框架里"真的会拿来做编辑器"的代码编辑控件，按文件语言做语法高亮（测试文件是 Go），
    可编辑，能显示行号就显示，从顶部开始显示。不允许用只读 Label 冒充。
- 文件在启动时读取（同步、异步都行）。**只有在画出了包含文件内容的那一帧之后**，才往 stdout 打印一行
  `ready` 并立即 flush。用框架自己的"下一帧/帧已提交"回调来判断。
- 如果第一帧里高亮还没出来（后台高亮），等高亮后的那一帧画出来再额外打印一行 `highlighted`；
  第一帧就带高亮则 `ready` 之后紧跟着打印 `highlighted` 即可。
- 打印 `ready` 之后保持运行、空闲，直到被 kill。
- 发布构建：Rust 用 `[profile.release] lto = "fat"`、`codegen-units = 1`、`strip = true`；
  Go 用 `-trimpath -ldflags "-s -w"`。
- 用 crates.io 上当前最新的稳定版本，`Cargo.lock` 提交在目录里，版本号写进该目录的 NOTES.md。

## 测试文件（files/）

| 文件 | 内容 | 规模 |
|---|---|---|
| `small.go` | Go 标准库 strings.go | 1,300 行，34 KB |
| `big.go` | net/http 全部源码拼两遍 | 107,958 行，3.1 MB |
| `doc.md` | 中英混排，含表格、代码块、任务列表 | 1,332 行，55 KB |

## 怎么测

`run/` 下的 runner 轮流启动每个应用（每个文件、每个框架交替，跑 N 轮取中位数），测：

- **启动时间**：从启动进程到"应用打印 ready"和"窗口服务器报告窗口已上屏"两者中较晚的那个。
- **高亮时间**：到打印 `highlighted` 为止。
- **内存**：ready 后 2 秒的 physical footprint（和活动监视器一致）。
- **包体积**：发布构建的二进制大小。

方法沿用 MyGo 仓库 `benchmarks/run` 的做法（窗口上屏判断、footprint 读取），
但场景换成了"打开真实文件"。

`tools/winshot <pid> <out.png>` 可以截下某个进程的窗口，用来确认内容确实画出来了。

## 复现

```sh
cd bench
go build -o run/run ./run
./run/run -runs 10 -out results/startup.json   # 冷启动，测 bin/ 下所有 bench-*
./run/run -resident -apps native,mygo -runs 5   # 已运行时打开文件，先跑 tools/mkbundle.sh 打包 build/QE-*.app
```

结果见 [RESULTS.md](RESULTS.md)。
