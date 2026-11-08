# charts.py: the README's comparison, as SVG in a light and a dark variant
# (docs/images/bench-*-light.svg, -dark.svg), from results/fair.jsonl:
#
#   hero        the 108k-line Go file, a card per state: QuickEdit's time,
#               how much faster than the runner-up, every editor's bar
#   scoreboard  the 12 measurements, a tile each, blue where QuickEdit is
#               the fastest
#   footprint   app size and memory, as the hero's cards
#
# One accent, QuickEdit's; every other editor gray. Numbers are in ink,
# never in the bars' colors; each bar has its value at its tip, and
# results/FAIR.md has them all as tables.
import json
import statistics
from html import escape
from pathlib import Path

here = Path(__file__).parent
res = here / "results"
out = here.parent.parent / "docs" / "images"
out.mkdir(parents=True, exist_ok=True)

# TextEdit is left out: it doesn't highlight code.
EDITORS = ["QuickEdit", "Sublime Text", "Zed", "CotEditor", "VS Code"]
STATES = [
    ("cold", "冷启动", "编辑器没在运行"),
    ("background", "运行中，无窗口", "打开过文件，窗口已关"),
    ("window", "运行中，有窗口", "编辑器开着一个文件"),
]
FILES = [("small.go", "Go 源码", "1,300 行"), ("big.go", "大 Go 文件", "10.8 万行"),
         ("doc.md", "Markdown", "1,332 行，中英混排"), ("huge.log", "日志", "60 万行，44 MB")]
SHORT = {"Sublime Text": "Sublime", "VS Code": "VS Code", "CotEditor": "CotEditor", "Zed": "Zed", "QuickEdit": "QuickEdit"}

# GitHub shows the images on white or on #0d1117: the cards are its
# subtle canvas, the accent and the grays pass 3:1 on them.
THEMES = {
    "light": dict(card="#f6f8fa", line="#d0d7de", ink="#1f2328", ink2="#59636e", muted="#818b98",
                  me="#2a78d6", other="#8a8880", tile="#256abf", pill="#2a78d6"),
    "dark": dict(card="#161b22", line="#30363d", ink="#f0f6fc", ink2="#9198a1", muted="#6e7681",
                 me="#3987e5", other="#6e6c66", tile="#256abf", pill="#3987e5"),
}
FONT = '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Noto Sans CJK SC", "Microsoft YaHei", sans-serif'

# --- data -------------------------------------------------------------------

runs, mem_runs = {}, {}
for line in (res / "fair.jsonl").read_text().splitlines():
    r = json.loads(line)
    if "error" in r:
        continue
    runs.setdefault((r["state"], r["name"], r["file"]), []).append(max(r["visualMs"], r.get("frontMs", 0)))
    if r["state"] == "cold" and r["file"] == "big.go" and r.get("memoryMB"):
        mem_runs.setdefault(r["name"], []).append(r["memoryMB"])
fair = {k: statistics.median(v) for k, v in runs.items()}
memory = {k: statistics.median(v) for k, v in mem_runs.items()}
sizes = json.loads((res / "sizes.json").read_text())


def ranked(values):
    """(name, value) by value, smallest first."""
    return sorted(values.items(), key=lambda kv: kv[1])


def num(v):
    return f"{v:,.0f}"


def times(a, b):
    """How many times b is a: 1.4 for 517 against 730."""
    return f"{b / a:.1f}"


# --- drawing ----------------------------------------------------------------

def t(x, y, s, size: float = 13, fill="ink", weight=400, anchor="start", opacity=None):
    o = f' fill-opacity="{opacity}"' if opacity else ""
    return (f'<text x="{x:.1f}" y="{y:.1f}" font-size="{size}" font-weight="{weight}" fill="{{{fill}}}"'
            f' text-anchor="{anchor}"{o}>{s}</text>')


def big(x, y, value, unit, size, fill="ink", unit_fill="ink2"):
    """A large number with its unit after it."""
    return (f'<text x="{x:.1f}" y="{y:.1f}" fill="{{{fill}}}"><tspan font-size="{size}" font-weight="650">{value}</tspan>'
            f'<tspan font-size="{size * 0.42:.0f}" fill="{{{unit_fill}}}" dx="5">{unit}</tspan></text>')


def card(x, y, w, h):
    return f'<rect x="{x:.1f}" y="{y:.1f}" width="{w:.1f}" height="{h:.1f}" rx="14" fill="{{card}}" stroke="{{line}}"/>'


def rule(x1, x2, y):
    return f'<line x1="{x1:.1f}" y1="{y:.1f}" x2="{x2:.1f}" y2="{y:.1f}" stroke="{{line}}"/>'


def bar(x, y, w, h, fill):
    """A bar from the baseline at x, square there, rounded at its end."""
    w = max(w, 3)
    r = min(4, w / 2, h / 2)
    return (f'<path d="M{x:.1f},{y:.1f} H{x + w - r:.1f} Q{x + w:.1f},{y:.1f} {x + w:.1f},{y + r:.1f} '
            f'V{y + h - r:.1f} Q{x + w:.1f},{y + h:.1f} {x + w - r:.1f},{y + h:.1f} H{x:.1f} Z" fill="{{{fill}}}"/>')


def text_width(s, size):
    """About how wide s is: CJK a full em, the rest about 0.6."""
    return sum(size if ord(ch) > 0x2E80 else size * 0.6 for ch in s)


def pill(x_right, y, label):
    """A rounded badge that ends at x_right, its text centered on y."""
    w = text_width(label, 13) + 22
    return (f'<rect x="{x_right - w:.1f}" y="{y - 13:.1f}" width="{w:.1f}" height="26" rx="13" fill="{{pill}}" fill-opacity="0.16"/>'
            + t(x_right - w / 2, y + 4.5, escape(label), 13, "ink", 600, "middle"))


def bars(x, y, w, rows, vmax, unit, pitch=21):
    """Ranked bars: the name, the bar, and its value at its tip."""
    parts, label_w, value_w, bh = [], 78, 66, 10
    x0, span = x + label_w, w - label_w - value_w
    for i, (name, v) in enumerate(rows):
        me = name == "QuickEdit"
        cy = y + i * pitch
        bw = span * v / vmax
        parts.append(t(x0 - 9, cy + 4, escape(SHORT[name]), 12, "ink" if me else "ink2", 600 if me else 400, "end"))
        parts.append(bar(x0, cy - bh / 2, bw, bh, "me" if me else "other"))
        parts.append(t(x0 + max(bw, 3) + 6, cy + 4, f"{num(v)}{unit}", 11.5, "ink" if me else "ink2", 600 if me else 400))
    return parts


def svg(name, width, height, body, label):
    for theme, c in THEMES.items():
        s = "\n".join(body)
        for k, v in c.items():
            s = s.replace("{" + k + "}", v)
        doc = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height:.0f}" viewBox="0 0 {width} {height:.0f}" '
               f'role="img" aria-label="{escape(label)}" font-family=\'{FONT}\'>\n<title>{escape(label)}</title>\n{s}\n</svg>\n')
        (out / f"{name}-{theme}.svg").write_text(doc)


def heading(title, sub):
    return [t(2, 24, escape(title), 20, "ink", 650), t(2, 48, escape(sub), 13, "ink2")]


W, GAP = 960, 16

# 1. Hero: the 108k-line Go file in the three states.
body = heading("打开 10.8 万行的 Go 文件", "同一台 Mac、同一标准实测，3 次中位数，越短越好")
cw, top, ch = (W - 2 * GAP) / 3, 70, 272
vmax = max(fair[(st, e, "big.go")] for st, _, _ in STATES for e in EDITORS)
for i, (st, title, sub) in enumerate(STATES):
    x = i * (cw + GAP)
    rows = ranked({e: fair[(st, e, "big.go")] for e in EDITORS})
    me = fair[(st, "QuickEdit", "big.go")]
    n2, v2 = next((n, v) for n, v in rows if n != "QuickEdit")
    body += [card(x, top, cw, ch),
             t(x + 20, top + 32, escape(title), 15, "ink", 600),
             t(x + 20, top + 51, escape(sub), 12, "ink2"),
             big(x + 20, top + 104, num(me), "ms", 40),
             pill(x + cw - 20, top + 91, f"快 {times(me, v2)}×"),
             t(x + 20, top + 128, escape(f"第二名 {SHORT[n2]} {num(v2)} ms"), 12, "ink2"),
             rule(x + 20, x + cw - 20, top + 146)]
    body += bars(x + 20, top + 170, cw - 40, rows, vmax, "")
svg("bench-hero", W, top + ch + 2, body, "打开 10.8 万行的 Go 文件：三种状态下 QuickEdit 都最快")

# 2. Scoreboard: the twelve measurements.
wins = sum(1 for st, _, _ in STATES for f, _, _ in FILES if min(EDITORS, key=lambda e: fair[(st, e, f)]) == "QuickEdit")
body = heading(f"12 项成绩，QuickEdit {wins} 项第一", "三种状态 × 四个文件。蓝色格子是 QuickEdit 最快的；格里是它的用时，和最接近的对手")
left, top, th, g = 132, 104, 86, 10
tw = (W - left - 3 * g) / 4
for j, (f, name, sub) in enumerate(FILES):
    x = left + j * (tw + g)
    body += [t(x + 2, top - 26, escape(name), 13, "ink", 600), t(x + 2, top - 10, escape(sub), 11.5, "ink2")]
for i, (st, title, sub) in enumerate(STATES):
    y = top + i * (th + g)
    body += [t(2, y + th / 2 - 3, escape(title), 14, "ink", 600), t(2, y + th / 2 + 15, escape(sub), 11.5, "ink2")]
    for j, (f, _, _) in enumerate(FILES):
        x = left + j * (tw + g)
        rows = ranked({e: fair[(st, e, f)] for e in EDITORS})
        me = fair[(st, "QuickEdit", f)]
        rank = [n for n, _ in rows].index("QuickEdit") + 1
        if rank == 1:
            n2, v2 = rows[1]
            # White on the blue tile, in both themes.
            body += [f'<rect x="{x:.1f}" y="{y:.1f}" width="{tw:.1f}" height="{th}" rx="12" fill="{{tile}}"/>',
                     t(x + 16, y + 25, "第 1", 12.5, "white", 600),
                     t(x + tw - 14, y + 25, f"快 {times(me, v2)}×", 12.5, "white", 600, "end"),
                     big(x + 16, y + 56, num(me), "ms", 24, "white", "white"),
                     t(x + 16, y + 75, escape(f"第二名 {SHORT[n2]} {num(v2)}"), 11.5, "white", 400, opacity="0.9")]
        else:
            n1, v1 = rows[0]
            body += [f'<rect x="{x:.1f}" y="{y:.1f}" width="{tw:.1f}" height="{th}" rx="12" fill="{{card}}" stroke="{{line}}"/>',
                     t(x + 16, y + 25, f"第 {rank}", 12.5, "ink2", 600),
                     big(x + 16, y + 56, num(me), "ms", 24),
                     t(x + 16, y + 75, escape(f"第一名 {SHORT[n1]} {num(v1)}"), 11.5, "ink2")]
body = [b.replace("{white}", "#ffffff") for b in body]
svg("bench-scoreboard", W, top + 3 * th + 2 * g + 2, body, f"12 项成绩里 QuickEdit {wins} 项第一")

# 3. Footprint: app size and memory.
body = heading("体积和内存", "安装后的 .app 大小；冷启动打开 10.8 万行 Go 文件后，全部进程的内存")
cw, top, ch = (W - GAP) / 2, 70, 240
for i, (title, values) in enumerate([("应用体积", {e: sizes[e] for e in EDITORS}),
                                     ("内存", {e: memory[e] for e in EDITORS})]):
    x = i * (cw + GAP)
    rows = ranked(values)
    me = values["QuickEdit"]
    n2, v2 = next((n, v) for n, v in rows if n != "QuickEdit")
    body += [card(x, top, cw, ch),
             t(x + 20, top + 32, escape(title), 15, "ink", 600),
             big(x + 20, top + 80, num(me), "MB", 36),
             pill(x + cw - 20, top + 68, f"仅为 {SHORT[n2]} 的 {me / v2 * 100:.0f}%"),
             rule(x + 20, x + cw - 20, top + 100)]
    body += bars(x + 20, top + 126, cw - 40, rows, max(values.values()), " MB")
svg("bench-footprint", W, top + ch + 2, body, "应用体积和内存：QuickEdit 都最小")

# The numbers as tables, for the README: results/FAIR.md.
lines = []
for st, title, sub in STATES:
    lines += [f"**{title}**：{sub}。", "", "| | " + " | ".join(EDITORS) + " |", "|---|" + "---:|" * len(EDITORS)]
    for f, _, _ in FILES:
        vals = [fair.get((st, e, f)) for e in EDITORS]
        best = min(v for v in vals if v is not None)
        lines.append(f"| {f} | " + " | ".join("—" if v is None else (f"**{v:,.0f}**" if v == best else f"{v:,.0f}") for v in vals) + " |")
    lines.append("")
(res / "FAIR.md").write_text("\n".join(lines))
print("wins", wins, "| wrote", sorted(p.name for p in out.glob("bench-*.svg")))
