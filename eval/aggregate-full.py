#!/usr/bin/env python3
"""Un-blind eval/<slug>/full/sheet-part-NN.scored.csv with full/key.json and print
per-title + merged 0/1/2 tallies (haiku vs sonnet) over EVERY cue, plus AC #4 verdict.
Also writes eval/full-scores.csv (per title) and eval/zeros-full.csv (all 0-scored cues)."""
import csv, json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
MAX_ZERO, MIN_NAT = 0.05, 0.60
slugs = sorted(p.parent.parent.name for p in ROOT.glob("*/full/key.json"))
merged = {"haiku": [0, 0, 0], "sonnet": [0, 0, 0]}
per = []; zeros = []; missing = []
def rate(c): n = sum(c); return (c[0] / n, c[2] / n) if n else (0, 0)
def verdict(c, worst=None):
    z, nat = rate(c); ok = z <= MAX_ZERO and nat >= MIN_NAT and (worst is None or worst <= 0.10)
    return "✅ 可看" if ok else "❌ 需校稿"
print(f"{'title/model':<14}{'n':>6}{'0':>6}{'1':>6}{'2':>6}{'0-rate':>8}{'2-rate':>8}")
for s in slugs:
    d = ROOT / s / "full"; rows = json.loads((d / "key.json").read_text(encoding="utf-8"))["rows"]
    t = {"haiku": [0, 0, 0], "sonnet": [0, 0, 0]}; unscored = 0; expected = len(rows); seen = 0
    for part in sorted(d.glob("sheet-part-*.csv")):
        if ".scored" in part.name: continue
        scored = part.with_name(part.name.replace(".csv", ".scored.csv"))
        if not scored.exists(): missing.append(str(scored.relative_to(ROOT))); continue
        with scored.open(encoding="utf-8") as fh:
            for r in csv.DictReader(fh):
                seen += 1
                ls, rs = r["left_score"].strip(), r["right_score"].strip()
                if ls not in ("0", "1", "2") or rs not in ("0", "1", "2"): unscored += 1; continue
                left = "haiku" if rows.get(r["idx"]) == "left=a" else "sonnet"; right = "sonnet" if left == "haiku" else "haiku"
                t[left][int(ls)] += 1; t[right][int(rs)] += 1
                if ls == "0" or rs == "0":
                    h, sn = (r["left"], r["right"]) if left == "haiku" else (r["right"], r["left"])
                    hs, ss = (ls, rs) if left == "haiku" else (rs, ls)
                    zeros.append([s, r["idx"], r["start"], r["source"], h, sn, hs, ss, r.get("note", "")])
    print(f"[{s}] cues={expected} scored={seen - unscored} unscored={unscored}")
    for m in ("haiku", "sonnet"):
        c = t[m]; z, nat = rate(c)
        print(f"  {m:<12}{sum(c):>6}{c[0]:>6}{c[1]:>6}{c[2]:>6}{z:>8.1%}{nat:>8.1%}  {verdict(c)}")
        per.append([s, m, sum(c), *c, f"{z:.4f}", f"{nat:.4f}", verdict(c)])
        for i in range(3): merged[m][i] += c[i]
print("\n[merged — all cues]")
worst = {m: max((float(p[6]) for p in per if p[1] == m), default=0) for m in merged}
for m in ("haiku", "sonnet"):
    c = merged[m]; z, nat = rate(c)
    print(f"  {m:<12}{sum(c):>6}{c[0]:>6}{c[1]:>6}{c[2]:>6}{z:>8.1%}{nat:>8.1%}  {verdict(c, worst[m])}  (worst title 0-rate {worst[m]:.1%})")
zh, zs = rate(merged["haiku"])[0], rate(merged["sonnet"])[0]
okh = verdict(merged["haiku"], worst["haiku"]).startswith("✅"); oks = verdict(merged["sonnet"], worst["sonnet"]).startswith("✅")
print("\n→ " + ("Haiku 過關，預設不動。" if okh else "預設改 claude-sonnet-5。" if oks and zs <= zh / 2 else "Sonnet 過但差距不到一半 → 複驗。" if oks else "兩個都不過 → 立品質 story。"))
with (ROOT / "full-scores.csv").open("w", newline="", encoding="utf-8") as fh:
    w = csv.writer(fh); w.writerow(["slug", "model", "n", "score0", "score1", "score2", "zero_rate", "natural_rate", "verdict"]); w.writerows(per)
with (ROOT / "zeros-full.csv").open("w", newline="", encoding="utf-8") as fh:
    w = csv.writer(fh); w.writerow(["slug", "idx", "start", "source", "haiku", "sonnet", "haiku_score", "sonnet_score", "note"]); w.writerows(zeros)
print(f"\nwrote eval/full-scores.csv, eval/zeros-full.csv ({len(zeros)} zero rows)")
if missing: print(f"\nmissing {len(missing)} scored parts:\n  " + "\n  ".join(missing))
