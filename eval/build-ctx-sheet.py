#!/usr/bin/env python3
"""Add ±N cues of context (source + the same side's translation) to a blind sheet.
Writes <slug>/claude-judge-ctx/sheet-ctx.csv. Left/right stay anonymous: the key is used
only to pull the matching variant's neighbours, never written out."""
import csv, json, re, sys
from pathlib import Path
N = 3
TS = re.compile(r"^(\d{2}:\d{2}:\d{2},\d{3})\s*-->")
def cues(p):
    raw = p.read_text(encoding="utf-8-sig", errors="replace").replace("\r\n", "\n"); out = {}
    for ch in re.split(r"\n{2,}", raw.strip()):
        l = [x for x in ch.split("\n") if x.strip()]
        if len(l) < 2: continue
        try: i = int(l[0].strip())
        except ValueError: continue
        if not TS.match(l[1].strip()): continue
        out[i] = " ".join(x.strip() for x in l[2:])
    return out
def ctx(d, idx, keys):
    pos = keys.index(idx)
    before = [f"[{k}] {d[k]}" for k in keys[max(0, pos - N):pos]]
    after = [f"[{k}] {d[k]}" for k in keys[pos + 1:pos + 1 + N]]
    return " / ".join(before), " / ".join(after)
for slug in sys.argv[1:]:
    d = Path("eval") / slug
    key = json.loads((d / "key.json").read_text(encoding="utf-8"))
    src, a, b = cues(d / "source.srt"), cues(d / "haiku.srt"), cues(d / "sonnet.srt")
    keys = sorted(set(src) & set(a) & set(b))
    out = d / "claude-judge-ctx"; out.mkdir(exist_ok=True)
    with (d / "sheet.csv").open(encoding="utf-8") as fh, (out / "sheet-ctx.csv").open("w", newline="", encoding="utf-8") as fo:
        rd = csv.DictReader(fh)
        w = csv.writer(fo)
        w.writerow(["idx", "start", "source_before", "source", "source_after", "left_before", "left", "left_after", "right_before", "right", "right_after", "left_score", "right_score", "note"])
        n = 0
        for r in rd:
            i = int(r["idx"]); left_is_a = key["rows"][r["idx"]] == "left=a"
            L, R = (a, b) if left_is_a else (b, a)
            sb, sa = ctx(src, i, keys); lb, la = ctx(L, i, keys); rb, ra = ctx(R, i, keys)
            w.writerow([r["idx"], r["start"], sb, r["source"], sa, lb, r["left"], la, rb, r["right"], ra, "", "", ""]); n += 1
    print(f"{slug}: {n} rows -> {out/'sheet-ctx.csv'}")
