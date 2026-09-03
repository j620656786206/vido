#!/usr/bin/env python3
"""Full-file blind sheets: every cue shared by source/haiku/sonnet, left/right shuffled per row
(seeded), split into consecutive parts of CHUNK rows. Writes eval/<slug>/full/sheet-part-NN.csv
and eval/<slug>/full/key.json ({idx: "left=a"|"left=b"}, a=haiku, b=sonnet)."""
import csv, json, random, re, sys
from pathlib import Path
CHUNK = 300
TS = re.compile(r"^(\d{2}:\d{2}:\d{2},\d{3})\s*-->")
def cues(p):
    raw = p.read_text(encoding="utf-8-sig", errors="replace").replace("\r\n", "\n"); out = {}
    for ch in re.split(r"\n{2,}", raw.strip()):
        l = [x for x in ch.split("\n") if x.strip()]
        if len(l) < 2: continue
        try: i = int(l[0].strip())
        except ValueError: continue
        m = TS.match(l[1].strip())
        if not m: continue
        out[i] = (m.group(1), " ".join(x.strip() for x in l[2:]))
    return out
total = 0
for slug in sys.argv[1:]:
    d = Path("eval") / slug
    src, a, b = cues(d / "source.srt"), cues(d / "haiku.srt"), cues(d / "sonnet.srt")
    keys = sorted(set(src) & set(a) & set(b))
    rng = random.Random(f"{slug}-full-42")
    out = d / "full"; out.mkdir(exist_ok=True)
    key = {}
    for n, start in enumerate(range(0, len(keys), CHUNK), 1):
        part = keys[start:start + CHUNK]
        with (out / f"sheet-part-{n:02d}.csv").open("w", newline="", encoding="utf-8") as fo:
            w = csv.writer(fo); w.writerow(["idx", "start", "source", "left", "right", "left_score", "right_score", "note"])
            for i in part:
                la = rng.random() < 0.5
                key[str(i)] = "left=a" if la else "left=b"
                L, R = (a[i][1], b[i][1]) if la else (b[i][1], a[i][1])
                w.writerow([i, src[i][0], src[i][1], L, R, "", "", ""])
    (out / "key.json").write_text(json.dumps({"a": "haiku", "b": "sonnet", "rows": key}, ensure_ascii=False, indent=1), encoding="utf-8")
    parts = n
    print(f"{slug}: {len(keys)} cues -> {parts} parts"); total += len(keys)
print("total cues:", total)
