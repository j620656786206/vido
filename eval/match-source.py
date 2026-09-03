#!/usr/bin/env python3
"""For each eval slug, pick the source-N.srt whose (index,start,end) set best matches the pipeline output."""
import re, sys
from pathlib import Path
TS = re.compile(r"^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})")
def cues(p):
    raw = p.read_text(encoding="utf-8-sig", errors="replace").replace("\r\n", "\n")
    out = set()
    for chunk in re.split(r"\n{2,}", raw.strip()):
        lines = chunk.split("\n")
        if len(lines) < 2: continue
        try: idx = int(lines[0].strip())
        except ValueError: continue
        m = TS.match(lines[1].strip())
        if m: out.add((idx, m.group(1), m.group(2)))
    return out
root = Path(sys.argv[1])
for d in sorted(p for p in root.iterdir() if p.is_dir()):
    outs = [f for f in ("haiku.srt", "sonnet.srt") if (d / f).exists()]
    if not outs: continue
    ref = cues(d / outs[0])
    best = None
    for s in sorted(d.glob("source-*.srt")):
        c = cues(s)
        if not c: continue
        overlap = len(ref & c) / max(len(ref), 1)
        print(f"{d.name}: {s.name} cues={len(c)} overlap_with_{outs[0]}={overlap:.3f}")
        if best is None or overlap > best[1]: best = (s, overlap)
    if best and best[1] > 0.95:
        (d / "source.srt").write_bytes(best[0].read_bytes())
        print(f"  -> {d.name}/source.srt = {best[0].name}")
    else:
        print(f"  !! {d.name}: no source matched well enough")
