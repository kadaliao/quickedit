# charts.py: the README's comparison charts, as SVG in light and dark
# variants (docs/images/*-light.svg, *-dark.svg), from results/:
#   bigfile    the 108k-line Go file in each of the three states
#   files      the four files one after the other, in each state
#   footprint  app size and memory
# QuickEdit's bars are the accent color, the others gray; every bar has
# its value at its tip, and the README has the numbers as a table.
import json
import statistics
from html import escape
from pathlib import Path

here = Path(__file__).parent
res = here / "results"
out = here.parent.parent / "docs" / "images"
out.mkdir(parents=True, exist_ok=True)

# TextEdit is left out: it neither highlights code nor renders Markdown.
EDITORS = ["QuickEdit", "Sublime Text", "Zed", "CotEditor", "VS Code"]
THEMES = {
    # Transparent: the chart sits on GitHub's page, white or #0d1117.
    "light": {"ink": "#0b0b0b", "ink2": "#52514e", "muted": "#6f6d66", "me": "#2a78d6", "other": "#8e8c84", "rule": "#d6d4cc"},
    "dark": {"ink": "#f0efec", "ink2": "#c3c2b7", "muted": "#9a988f", "me": "#3987e5", "other": "#6e6c66", "rule": "#383835"},
}
FONT = '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Noto Sans CJK SC", "Microsoft YaHei", sans-serif'

# --- data -----------------------------------------------------------------

# One session, every editor in the same states (fair.sh): the median of
# its runs, open meaning settled and in front.
runs = {}
memory_runs = {}
for line in (res / "fair.jsonl").read_text().splitlines():
    r = json.loads(line)
    if "error" in r:
        continue
    runs.setdefault((r["state"], r["name"], r["file"]), []).append(max(r["visualMs"], r.get("frontMs", 0)))
    if r["state"] == "cold" and r["file"] == "big.go" and r.get("memoryMB"):
        memory_runs.setdefault(r["name"], []).append(r["memoryMB"])
fair = {k: statistics.median(v) for k, v in runs.items()}
memory = {k: statistics.median(v) for k, v in memory_runs.items()}
sizes = json.loads((res / "sizes.json").read_text())
STATES = [("cold", "冷启动", "编辑器没在运行"), ("background", "运行中，无窗口", "打开过文件，窗口已关"), ("window", "运行中，有窗口", "编辑器开着一个文件")]


# --- drawing ---------------------------------------------------------------

def text(x, y, s, size: float = 13, fill="ink", weight=400, anchor="start", tabular=False):
    style = "font-variant-numeric: tabular-nums;" if tabular else ""
    return (f'<text x="{x:.1f}" y="{y:.1f}" font-size="{size}" font-weight="{weight}" '
            f'fill="{{{fill}}}" text-anchor="{anchor}" style="{style}">{escape(s)}</text>')


def bar(x, y, w, h):
    """A bar from the baseline at x, square there, its data end rounded 4px."""
    w = max(w, 2)
    r = min(4, w / 2, h / 2)
    return (f'<path d="M{x:.1f},{y:.1f} H{x + w - r:.1f} Q{x + w:.1f},{y:.1f} {x + w:.1f},{y + r:.1f} '
            f'V{y + h - r:.1f} Q{x + w:.1f},{y + h:.1f} {x + w - r:.1f},{y + h:.1f} H{x:.1f} Z"')


def panel(x, y, w, title, sub, rows, vmax, fmt):
    """Ranked horizontal bars, fastest (smallest) first."""
    parts = [text(x, y + 15, title, 15, "ink", 600), text(x, y + 35, sub, 12.5, "ink2")]
    label_w, value_w, pitch, bh = 104, 70, 30, 16
    x0 = x + label_w
    span = w - label_w - value_w
    top = y + 52
    rows = sorted(rows, key=lambda r: r[1])
    for i, (name, v) in enumerate(rows):
        me = name == "QuickEdit"
        cy = top + i * pitch + pitch / 2
        bw = span * v / vmax
        parts.append(text(x0 - 10, cy + 4.5, name, 13, "ink" if me else "ink2", 600 if me else 400, "end"))
        parts.append(bar(x0, cy - bh / 2, bw, bh) + f' fill="{{{"me" if me else "other"}}}"/>')
        parts.append(text(x0 + max(bw, 2) + 6, cy + 4.5, fmt(v), 12.5, "ink" if me else "ink2", 600 if me else 400, tabular=True))
    parts.append(f'<line x1="{x0}" y1="{top - 2}" x2="{x0}" y2="{top + len(rows) * pitch + 2}" stroke="{{rule}}" stroke-width="1"/>')
    return parts, top + len(rows) * pitch - y


def svg(name, width, height, body, label):
    for theme, c in THEMES.items():
        s = "\n".join(body)
        for k, v in c.items():
            s = s.replace("{" + k + "}", v)
        doc = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width + 16}" height="{height + 16}" '
               f'viewBox="-8 -8 {width + 16} {height + 16}" role="img" aria-label="{escape(label)}" '
               f'font-family=\'{FONT}\'>\n<title>{escape(label)}</title>\n{s}\n</svg>\n')
        (out / f"{name}-{theme}.svg").write_text(doc)


ms = lambda v: f"{v:,.0f} ms"

# 1. The 108k-line Go file, in each state.
W, gap = 960, 36
pw = (W - gap * 2) / 3
vmax = max(fair[(st, e, "big.go")] for st, _, _ in STATES for e in EDITORS)
body, h = [], 0
for i, (st, title, sub) in enumerate(STATES):
    p, ph = panel(i * (pw + gap), 0, pw, title, sub, [(e, fair[(st, e, "big.go")]) for e in EDITORS], vmax, ms)
    body += p
    h = max(h, ph)
svg("bench-bigfile", W, int(h + 8), body, "打开 10.8 万行的 Go 文件，三种状态下各编辑器的毫秒数")

# 2. The four files one after the other, in each state: the sum of
# their medians.
FILES = ["small.go", "big.go", "doc.md", "huge.log"]
total = {(st, e): sum(fair[(st, e, f)] for f in FILES) for st, _, _ in STATES for e in EDITORS}
pw = (W - gap * 2) / 3
vmax = max(total.values())
body, h = [], 0
for i, (st, title, sub) in enumerate(STATES):
    p, ph = panel(i * (pw + gap), 0, pw, title, sub, [(e, total[(st, e)]) for e in EDITORS], vmax, ms)
    body += p
    h = max(h, ph)
svg("bench-files", W, int(h + 8), body, "依次打开四种文件的总毫秒数，三种状态")

# 3. App size and memory.
pw = (W - gap) / 2
body = []
size_rows = [(e, sizes[e]) for e in EDITORS if e in sizes]
p1, h1 = panel(0, 0, pw, "应用体积", "安装后的 .app 大小", size_rows, max(v for _, v in size_rows), lambda v: f"{v:,.0f} MB")
mem_rows = [(e, memory[e]) for e in EDITORS]
p2, h2 = panel(pw + gap, 0, pw, "内存", "打开 10.8 万行 Go 文件后，全部进程", mem_rows, max(v for _, v in mem_rows), lambda v: f"{v:,.0f} MB")
svg("bench-footprint", W, int(max(h1, h2) + 8), p1 + p2, "各编辑器的应用体积和内存")

print("wrote", sorted(p.name for p in out.glob("bench-*.svg")))

# The numbers as tables, for the README: results/FAIR.md.
lines = []
for st, title, sub in STATES:
    lines += [f"**{title}**：{sub}。", "", "| | " + " | ".join(EDITORS) + " |", "|---|" + "---:|" * len(EDITORS)]
    for f in FILES:
        vals = [fair.get((st, e, f)) for e in EDITORS]
        best = min(v for v in vals if v is not None)
        lines.append(f"| {f} | " + " | ".join("—" if v is None else (f"**{v:,.0f}**" if v == best else f"{v:,.0f}") for v in vals) + " |")
    lines.append("")
(res / "FAIR.md").write_text("\n".join(lines))
print("wrote results/FAIR.md")
