# 对比结果（2026-11-08，Mac mini M2，macOS 26）

毫秒数是从请求打开，到窗口里内容稳定下来（含语法高亮、Markdown 渲染）、窗口也到了最前面的时间，5 次的中位数。方法见 [../README.md](../README.md)。

"QuickEdit 常驻"是登录时已在后台启动、没有窗口的 QuickEdit，像双击一样打开文件（`resident.sh`）；"QuickEdit 常驻 · qe"改用 `qe` 命令打开（`VIA=qe ./resident.sh`）。这两行是之后单独重测的：重测时机器较忙（平均负载约 4），QuickEdit 的窗口也比之前大（1199×874），所以 big.go、doc.md、huge.log 比第一次测的慢 5 到 58 毫秒（第一次：133、123、111、219），内存也多一些。两行在同样条件下先后测完，可以直接比较。

### 冷启动打开（应用未运行）：到画面稳定的毫秒数，5 轮中位数

| 编辑器 | small.go | big.go | doc.md | huge.log | 内存 (big.go) |
|---|---:|---:|---:|---:|---:|
| CotEditor | 944 | 979 | 898 | 1391 | 225 MB |
| QuickEdit | 403 | 406 | 476 | 531 | 123 MB |
| QuickEdit 常驻 | 124 | 128 | 169 | 253 | 121 MB |
| QuickEdit 常驻 · qe | 102 | 114 | 150 | 226 | 120 MB |
| Sublime Text | 558 | 1018 | 598 | 1699 | 203 MB |
| TextEdit | 449 | 458 | 454 | 776 | 45 MB |
| Typora | — | — | 1107 | — | 46 MB |
| VS Code | 1573 | 1679 | 1837 | 1905 | 1349 MB |
| Zed | 635 | 548 | 560 | 620 | 269 MB |

### 应用已运行时打开：到画面稳定的毫秒数，5 轮中位数

| 编辑器 | small.go | big.go | doc.md | huge.log |
|---|---:|---:|---:|---:|
| CotEditor | 352 | 1892 | 361 | 838 |
| QuickEdit | 130 | 135 | 168 | 251 |
| QuickEdit 常驻 | — | — | — | — |
| QuickEdit 常驻 · qe | — | — | — | — |
| Sublime Text | 120 | 492 | 161 | 1202 |
| TextEdit | 166 | 182 | 167 | 496 |
| Typora | — | — | 582 | — |
| VS Code | 246 | 394 | 417 | 925 |
| Zed | 53 | 56 | 52 | 57 |
