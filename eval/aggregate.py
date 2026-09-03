#!/usr/bin/env python3
"""Un-blind every eval/<slug>/<judge>/sheet.csv with eval/<slug>/key.json and print
per-title + merged 0/1/2 tallies for haiku vs sonnet, plus the AC #4 verdict.

usage: aggregate.py <judge-subdir or '.'> [slug ...]
"""
import csv, json, sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
MAX_ZERO, MIN_NAT = 0.05, 0.60  # story AC #4, mirrors scripts/subtitle-blind-eval.py
judge = sys.argv[1] if len(sys.argv) > 1 else "."
slugs = sys.argv[2:] or sorted(p.name for p in ROOT.iterdir() if (p / "key.json").exists())

def tally(slug):
    key = json.loads((ROOT / slug / "key.json").read_text(encoding="utf-8"))
    rows = key["rows"]
    t = {"haiku": [0, 0, 0], "sonnet": [0, 0, 0]}
    zeros = []
    sheet = ROOT / slug / judge / "sheet.csv"
    if not sheet.exists():
        return None, []
    with sheet.open(encoding="utf-8") as fh:
        for r in csv.DictReader(fh):
            ls, rs = r["left_score"].strip(), r["right_score"].strip()
            if ls not in ("0", "1", "2") or rs not in ("0", "1", "2"):
                continue
            left = "haiku" if rows.get(r["idx"]) == "left=a" else "sonnet"
            right = "sonnet" if left == "haiku" else "haiku"
            t[left][int(ls)] += 1
            t[right][int(rs)] += 1
            if ls == "0" or rs == "0":
                zeros.append((slug, r["idx"], r["source"], r["left"] if left == "haiku" else r["right"],
                              r["right"] if left == "haiku" else r["left"], ls if left == "haiku" else rs,
                              rs if left == "haiku" else ls, r.get("note", "")))
    return t, zeros

def line(name, c):
    n = sum(c)
    if not n:
        return f"{name:<12}  (no scored rows)"
    z, nat = c[0] / n, c[2] / n
    ok = z <= MAX_ZERO and nat >= MIN_NAT
    return f"{name:<12}{n:>4}{c[0]:>6}{c[1]:>6}{c[2]:>6}{z:>8.1%}{nat:>8.1%}  {'✅ 可看' if ok else '❌ 需校稿'}"

merged = {"haiku": [0, 0, 0], "sonnet": [0, 0, 0]}
per_title_zero = {"haiku": {}, "sonnet": {}}
all_zeros = []
print(f"{'title/model':<12}{'n':>4}{'0':>6}{'1':>6}{'2':>6}{'0-rate':>8}{'2-rate':>8}")
for s in slugs:
    t, zeros = tally(s)
    if t is None:
        print(f"{s}: no {judge}/sheet.csv"); continue
    all_zeros += zeros
    print(f"[{s}]")
    for m in ("haiku", "sonnet"):
        print("  " + line(m, t[m]))
        for i in range(3): merged[m][i] += t[m][i]
        n = sum(t[m]); per_title_zero[m][s] = t[m][0] / n if n else 0
print("\n[merged]")
for m in ("haiku", "sonnet"):
    print("  " + line(m, merged[m]))
    worst = max(per_title_zero[m].items(), key=lambda kv: kv[1]) if per_title_zero[m] else ("-", 0)
    print(f"    worst title 0-rate: {worst[0]} {worst[1]:.1%} {'(>10% → fails AC #4 per-title rule)' if worst[1] > 0.10 else ''}")
zh, zs = merged["haiku"], merged["sonnet"]
if sum(zh) and sum(zs):
    ok = lambda c: c[0] / sum(c) <= MAX_ZERO and c[2] / sum(c) >= MIN_NAT and max(per_title_zero[m].values()) <= 0.10
    okh, oks = ok(zh), ok(zs)
    zr_h, zr_s = zh[0] / sum(zh), zs[0] / sum(zs)
    print()
    if okh: print("→ 裁定：Haiku 過關，預設不動。")
    elif oks and zr_s <= zr_h / 2: print("→ 裁定：Haiku 不過、Sonnet 過且 0 分率 ≤ Haiku 一半 → 預設改 claude-sonnet-5。")
    elif oks: print("→ 裁定：Sonnet 過但差距不到一半 → 換 seed 再抽 50 句複驗。")
    else: print("→ 裁定：兩個都不過 → 不招募，立品質 story（上下文窗／terminology corrector／二段式自審）。")
if all_zeros:
    out = ROOT / f"zeros-{judge.strip('./') or 'human'}.csv"
    with out.open("w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh); w.writerow(["slug", "idx", "source", "haiku", "sonnet", "haiku_score", "sonnet_score", "note"]); w.writerows(all_zeros)
    print(f"\n0 分對照 {len(all_zeros)} 句 → {out.relative_to(ROOT.parent)}")
