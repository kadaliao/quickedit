# summarize.py: medians of results/raw.jsonl by editor, file and mode,
# written to results/summary.json and printed as a Markdown table.
import json
import statistics
import sys
from collections import defaultdict
from pathlib import Path

here = Path(__file__).parent
raw = here / "results" / "raw.jsonl"
rows = [json.loads(l) for l in raw.read_text().splitlines() if l.strip()]

groups = defaultdict(list)
errors = defaultdict(list)
for r in rows:
    f = r["file"].removeprefix("w-")
    key = (r.get("name", r["app"]), f, r["mode"])
    if "error" in r:
        errors[key].append(r["error"])
    else:
        groups[key].append(r)


def med(xs):
    return statistics.median(xs) if xs else None


summary = []
for key in sorted(set(groups) | set(errors)):
    name, f, mode = key
    rs = groups.get(key, [])
    summary.append({
        "editor": name, "file": f, "mode": mode, "n": len(rs), "errors": errors.get(key, []),
        "windowMs": med([r["windowMs"] for r in rs]),
        "visualMs": med([r["visualMs"] for r in rs]),
        "visualMin": min((r["visualMs"] for r in rs), default=None),
        "visualMax": max((r["visualMs"] for r in rs), default=None),
        "memoryMB": med([r["memoryMB"] for r in rs if r["mode"] == "cold"]) if mode == "cold" else None,
    })
(here / "results" / "summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=1) + "\n")

files = ["small.go", "big.go", "doc.md", "huge.log"]
editors = list(dict.fromkeys(s["editor"] for s in summary))
by = {(s["editor"], s["file"], s["mode"]): s for s in summary}


def cell(s, k="visualMs"):
    if s is None:
        return "—"
    if s[k] is None:
        return "失败" if s["errors"] else "—"
    v = f"{s[k]:.0f}"
    return v + (f" ({len(s['errors'])} 次失败)" if s["errors"] else "")


for mode, title in [("cold", "冷启动打开（应用未运行）"), ("warm", "应用已运行时打开")]:
    print(f"\n### {title}：到画面稳定的毫秒数，{max((s['n'] for s in summary), default=0)} 轮中位数\n")
    print("| 编辑器 | " + " | ".join(files) + (" | 内存 (big.go) |" if mode == "cold" else " |"))
    print("|---|" + "---:|" * len(files) + ("---:|" if mode == "cold" else ""))
    for e in editors:
        cells = [cell(by.get((e, f, mode))) for f in files]
        mem = ""
        if mode == "cold":
            s = by.get((e, "big.go", "cold")) or by.get((e, "doc.md", "cold"))
            mem = f" {s['memoryMB']:.0f} MB |" if s and s["memoryMB"] else " — |"
        print(f"| {e} | " + " | ".join(cells) + " |" + mem)
if any(errors.values()):
    print("\n失败：", file=sys.stderr)
    for k, v in errors.items():
        print(" ", k, v[:2], file=sys.stderr)
