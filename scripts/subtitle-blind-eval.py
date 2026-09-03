#!/usr/bin/env python3
"""Blind A/B scoring sheet for two subtitle-translation variants (eval-1).

Two sub-commands, stdlib only:

  build  — sample N cues shared by the source SRT and both variant SRTs, shuffle
           which variant lands on the left/right of every row, and write
           <out>/sheet.csv (what you score) + <out>/key.json (which side was
           which — do NOT open it until you have scored).
  score  — read the scored sheet back, un-blind it with key.json, and print the
           per-variant 0/1/2 distribution plus the pre-registered verdict.

Scoring rubric (fill the `left_score` / `right_score` columns with one digit):
  0 = 看不懂或翻錯意思（不改不能看）
  1 = 看得懂但生硬、語氣不對、用語不像台灣（勉強能看）
  2 = 自然，像人翻的（不用改）

Alignment is by SRT cue index. The pipeline keeps Index/Start/End byte-equal
to the source (TranslateResult contract), so an index present in all three
files is the same cue. Cues the SDH filter dropped are simply absent from the
variants and are never sampled.
"""

from __future__ import annotations

import argparse
import csv
import json
import random
import re
import sys
from pathlib import Path

TIMESTAMP = re.compile(r"^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})")

# Pre-registered thresholds (eval-1 story AC #4). Change them in the story
# first, then here — never the other way round.
WATCHABLE_MAX_ZERO_RATE = 0.05  # ≤5% of sampled cues scored 0
WATCHABLE_MIN_NATURAL_RATE = 0.60  # ≥60% scored 2


def parse_srt(path: Path) -> dict[int, dict[str, str]]:
    """Return {index: {start, end, text}} — tolerant of CRLF and BOM."""
    raw = path.read_text(encoding="utf-8-sig").replace("\r\n", "\n")
    blocks: dict[int, dict[str, str]] = {}
    for chunk in re.split(r"\n{2,}", raw.strip()):
        lines = chunk.split("\n")
        if len(lines) < 2:
            continue
        try:
            idx = int(lines[0].strip())
        except ValueError:
            continue
        m = TIMESTAMP.match(lines[1].strip())
        if not m:
            continue
        blocks[idx] = {
            "start": m.group(1),
            "end": m.group(2),
            "text": " ".join(l.strip() for l in lines[2:] if l.strip()),
        }
    return blocks


def cmd_build(args: argparse.Namespace) -> int:
    src = parse_srt(Path(args.source))
    a = parse_srt(Path(args.a))
    b = parse_srt(Path(args.b))
    shared = sorted(set(src) & set(a) & set(b))
    if not shared:
        print("error: no cue index is present in all three files", file=sys.stderr)
        return 2

    rng = random.Random(args.seed)
    n = min(args.sample, len(shared))
    picked = sorted(rng.sample(shared, n))

    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    key: dict[str, str] = {}  # row idx -> "left=a" | "left=b"

    with (out / "sheet.csv").open("w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh)
        w.writerow(["idx", "start", "source", "left", "right", "left_score", "right_score", "note"])
        for idx in picked:
            left_is_a = rng.random() < 0.5
            left, right = (a[idx], b[idx]) if left_is_a else (b[idx], a[idx])
            key[str(idx)] = "left=a" if left_is_a else "left=b"
            w.writerow([idx, src[idx]["start"], src[idx]["text"], left["text"], right["text"], "", "", ""])

    (out / "key.json").write_text(
        json.dumps(
            {
                "a": str(args.a),
                "b": str(args.b),
                "source": str(args.source),
                "seed": args.seed,
                "sampled": n,
                "shared_cues": len(shared),
                "rows": key,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    print(f"wrote {out / 'sheet.csv'} ({n} of {len(shared)} shared cues)")
    print(f"wrote {out / 'key.json'}  <- do not open until scoring is done")
    return 0


def cmd_score(args: argparse.Namespace) -> int:
    out = Path(args.dir)
    key = json.loads((out / "key.json").read_text(encoding="utf-8"))
    rows = key["rows"]
    tally = {"a": [0, 0, 0], "b": [0, 0, 0]}
    unscored = 0

    with (out / "sheet.csv").open(encoding="utf-8") as fh:
        for row in csv.DictReader(fh):
            side = rows.get(row["idx"])
            ls, rs = row["left_score"].strip(), row["right_score"].strip()
            if ls not in "012" or rs not in "012" or not ls or not rs:
                unscored += 1
                continue
            left_variant = "a" if side == "left=a" else "b"
            right_variant = "b" if left_variant == "a" else "a"
            tally[left_variant][int(ls)] += 1
            tally[right_variant][int(rs)] += 1

    names = {"a": Path(key["a"]).stem, "b": Path(key["b"]).stem}
    print(f"sample: {key['sampled']} cues (seed {key['seed']}), unscored rows: {unscored}\n")
    print(f"{'variant':<14}{'n':>4}{'0 翻錯':>8}{'1 生硬':>8}{'2 自然':>8}{'0-rate':>9}{'2-rate':>9}  verdict")
    verdicts = {}
    for v in ("a", "b"):
        c = tally[v]
        n = sum(c)
        if n == 0:
            print(f"{names[v]:<14}{0:>4}  (no scored rows)")
            continue
        zero_rate, nat_rate = c[0] / n, c[2] / n
        ok = zero_rate <= WATCHABLE_MAX_ZERO_RATE and nat_rate >= WATCHABLE_MIN_NATURAL_RATE
        verdicts[v] = (ok, zero_rate)
        print(
            f"{names[v]:<14}{n:>4}{c[0]:>8}{c[1]:>8}{c[2]:>8}"
            f"{zero_rate:>9.1%}{nat_rate:>9.1%}  {'✅ 可看' if ok else '❌ 需校稿'}"
        )

    if len(verdicts) == 2:
        (ok_a, z_a), (ok_b, z_b) = verdicts["a"], verdicts["b"]
        print()
        if ok_a:
            print(f"→ keep default: {names['a']} passes on its own.")
        elif ok_b and z_b <= z_a / 2:
            print(f"→ switch default to {names['b']}: {names['a']} fails, {names['b']} passes with ≤ half the 0-rate.")
        elif ok_b:
            print(f"→ {names['b']} passes but not decisively; re-check with a second sample before switching.")
        else:
            print("→ both fail: do not recruit testers; file quality stories first.")
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)

    b = sub.add_parser("build", help="write a blind scoring sheet")
    b.add_argument("--source", required=True, help="English source .srt (same cue numbering as the variants)")
    b.add_argument("--a", required=True, help="variant A .srt (e.g. haiku.srt)")
    b.add_argument("--b", required=True, help="variant B .srt (e.g. sonnet.srt)")
    b.add_argument("--out", required=True, help="output directory")
    b.add_argument("--sample", type=int, default=50)
    b.add_argument("--seed", type=int, default=42)
    b.set_defaults(fn=cmd_build)

    s = sub.add_parser("score", help="tally a scored sheet")
    s.add_argument("dir", help="directory holding sheet.csv + key.json")
    s.set_defaults(fn=cmd_score)

    args = p.parse_args()
    return args.fn(args)


if __name__ == "__main__":
    sys.exit(main())
