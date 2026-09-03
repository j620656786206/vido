#!/usr/bin/env python3
"""Side-by-side: source / haiku / sonnet / existing zh-TW reference, aligned by nearest start time (±0.5s)."""
import bisect, csv, re, sys
from pathlib import Path
TS = re.compile(r"^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})")
def sec(t):
    h, m, s = t.split(":"); s, ms = s.split(",")
    return int(h) * 3600 + int(m) * 60 + int(s) + int(ms) / 1000
def cues(p):
    raw = p.read_text(encoding="utf-8-sig", errors="replace").replace("\r\n", "\n")
    out = []
    for ch in re.split(r"\n{2,}", raw.strip()):
        l = [x for x in ch.split("\n") if x.strip()]
        if len(l) < 2: continue
        idx = None
        try: idx = int(l[0].strip()); l = l[1:]
        except ValueError: pass
        m = TS.match(l[0].strip())
        if not m: continue
        out.append((idx, m.group(1), sec(m.group(1)), " ".join(x.strip() for x in l[1:])))
    return out
d = Path(sys.argv[1])
src = cues(d / "source.srt")
byidx = lambda p: {c[0]: c[3] for c in cues(p)}
haiku, sonnet = byidx(d / "haiku.srt"), byidx(d / "sonnet.srt")
ref = sorted(cues(d / "reference-existing-zh-TW.srt"), key=lambda c: c[2])
rt = [c[2] for c in ref]
with (d / "compare.csv").open("w", newline="", encoding="utf-8") as f:
    w = csv.writer(f); w.writerow(["idx", "start", "source", "haiku", "sonnet", "existing_zh_TW"])
    hit = 0
    for idx, start, t, text in src:
        i = bisect.bisect_left(rt, t - 0.5); r = ""
        if i < len(rt) and rt[i] <= t + 0.5: r = ref[i][3]; hit += 1
        w.writerow([idx, start, text, haiku.get(idx, ""), sonnet.get(idx, ""), r])
print(f"{d.name}: wrote compare.csv ({len(src)} rows, {hit} with a reference line)")
