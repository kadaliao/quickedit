# export.py: the comparison page's datasets, from the raw results:
#   results/editors.csv     editor × file × mode: medians of 5 runs
#   results/frameworks.csv  the framework benchmark of bench/results
import csv
import json
import statistics
from collections import defaultdict
from pathlib import Path

here = Path(__file__).parent
res = here / "results"

files = {"small.go": "1,300 行 Go", "big.go": "10.8 万行 Go", "doc.md": "Markdown", "huge.log": "44 MB 日志"}
groups = defaultdict(list)
for line in (res / "raw.jsonl").read_text().splitlines():
    r = json.loads(line)
    if "error" in r:
        continue
    groups[(r["name"], r["file"].removeprefix("w-"), r["mode"])].append(r)

with open(res / "editors.csv", "w", newline="") as f:
    w = csv.writer(f)
    w.writerow(["editor", "file", "file_label", "mode", "visual_ms", "window_ms", "visual_min_ms", "visual_max_ms", "memory_mb", "runs"])
    for (editor, file, mode), rs in sorted(groups.items()):
        vis = [r["visualMs"] for r in rs]
        w.writerow([editor, file, files[file], mode, round(statistics.median(vis)),
                    round(statistics.median(r["windowMs"] for r in rs)), min(vis), max(vis),
                    round(statistics.median(r["memoryMB"] for r in rs)), len(rs)])

samples = json.loads((here.parent / "results" / "startup.json").read_text())["samples"]
sizes = {"mygo": 11.2, "gpui": 14.9, "iced": 9.0, "egui": 11.4, "native": 0.06}
names = {"mygo": "MyGo", "gpui": "GPUI", "iced": "iced", "egui": "egui", "native": "原生 AppKit"}
with open(res / "frameworks.csv", "w", newline="") as f:
    w = csv.writer(f)
    w.writerow(["framework", "file", "startup_ms", "highlighted_ms", "memory_mb", "binary_mb"])
    for fw, byfile in samples.items():
        for file, ss in byfile.items():
            w.writerow([names[fw], file, round(statistics.median(s["StartupMs"] for s in ss)),
                        round(statistics.median(s["HighlightMs"] for s in ss)),
                        round(statistics.median(s["FootprintMB"] for s in ss)), sizes[fw]])
print((res / "frameworks.csv").read_text())
