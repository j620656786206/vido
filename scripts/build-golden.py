#!/usr/bin/env python3
"""Rebuild apps/api/internal/eval/golden/golden-v1.jsonl (story sub-7-8a).

The sample is 200 cues: 76 from Tears of Steel (CC BY 3.0, Blender
Foundation), 84 dialogue cues from Sita Sings the Blues (film CC0; subtitle
text from Wikimedia Commons, CC BY-SA 4.0) and 40 hand-written trap cues.
Everything hand-written lives in authored-v1.json next to the output; this
script only downloads the two public subtitle pairs, aligns them by timestamp
and merges. Run it with no arguments; pass --cache-dir to keep the downloaded
.srt files somewhere (they are never committed — .gitignore blocks eval .srt).

    python3 scripts/build-golden.py [--out PATH] [--cache-dir DIR]
"""
import argparse
import json
import re
import sys
import tempfile
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GOLDEN_DIR = ROOT / "apps/api/internal/eval/golden"
SAMPLE_VERSION = "golden-v1"

SOURCES = {
    "tos_en": "https://download.blender.org/demo/movies/ToS/subtitles/TOS-en.srt",
    "tos_zh": "https://download.blender.org/demo/movies/ToS/subtitles/TOS-CH-traditional.srt",
    "sita_en": "https://commons.wikimedia.org/w/index.php?title=TimedText:Sita_Sings_the_Blues.webm.en.srt&action=raw",
    "sita_zh": "https://commons.wikimedia.org/w/index.php?title=TimedText:Sita_Sings_the_Blues.webm.zh-hant.srt&action=raw",
}
ATTRIBUTION = {
    "tos": "Tears of Steel © Blender Foundation (mango.blender.org), CC BY 3.0; zh-TW subtitle: community file TOS-CH-traditional.srt from download.blender.org",
    "sita": "Sita Sings the Blues © Nina Paley, CC0 1.0 (since 2013-01-18); en/zh-Hant subtitle text from Wikimedia Commons TimedText, CC BY-SA 4.0",
    "trap": "Written for Vido golden-v1, CC BY-SA 4.0",
}


def fetch(name: str, cache: Path) -> str:
    path = cache / f"{name}.srt"
    if not path.exists():
        req = urllib.request.Request(SOURCES[name], headers={"User-Agent": "vido-build-golden/1.0"})
        with urllib.request.urlopen(req, timeout=60) as resp:
            path.write_bytes(resp.read())
    return path.read_text(encoding="utf-8-sig").replace("\r", "")


def parse_srt(text: str):
    cues = []
    for block in re.split(r"\n\s*\n", text.strip()):
        lines = block.strip().split("\n")
        if len(lines) < 3 or "-->" not in lines[1]:
            continue
        start, end = [x.strip() for x in lines[1].split("-->")]
        body = " ".join(x.strip() for x in lines[2:])
        body = re.sub(r"<[^>]+>", "", body)
        body = re.sub(r"\s+", " ", body).strip()
        cues.append({"start": start, "end": end, "text": body})
    return cues


def names_in(text: str, table: dict) -> dict:
    found = {}
    for en, zh in table.items():
        if re.search(rf"\b{re.escape(en)}\b", text):
            found[en] = zh
    return found


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(GOLDEN_DIR / f"{SAMPLE_VERSION}.jsonl"))
    ap.add_argument("--cache-dir", default=str(Path(tempfile.gettempdir()) / "vido-golden-src"))
    args = ap.parse_args()
    cache = Path(args.cache_dir)
    cache.mkdir(parents=True, exist_ok=True)

    authored = json.loads((GOLDEN_DIR / "authored-v1.json").read_text(encoding="utf-8"))
    rows = []

    # ── Tears of Steel: every cue, zh aligned by start timestamp, ordinal fallback ──
    tos_en = parse_srt(fetch("tos_en", cache))
    tos_zh = parse_srt(fetch("tos_zh", cache))
    zh_by_start = {c["start"]: c["text"] for c in tos_zh}
    if len(tos_en) != len(authored["tos"]):
        sys.exit(f"tos: expected {len(authored['tos'])} cues, source has {len(tos_en)}")
    for i, cue in enumerate(tos_en, start=1):
        zh1 = zh_by_start.get(cue["start"]) or tos_zh[i - 1]["text"]
        rows.append({
            "id": f"tos-{i:02d}", "source": "tos", "start": cue["start"], "end": cue["end"],
            "text": cue["text"], "refs": [zh1, authored["tos"][str(i)]],
            "names": names_in(cue["text"], authored["names"]["tos"]), "traps": [],
            "attribution": ATTRIBUTION["tos"],
        })

    # ── Sita: hand-picked dialogue cues (index into the English file), zh by timestamp ──
    sita_en = parse_srt(fetch("sita_en", cache))
    sita_zh = parse_srt(fetch("sita_zh", cache))
    zh_by_start = {}
    for c in sita_zh:
        zh_by_start.setdefault(c["start"], c["text"])
    picked = sorted(int(k) for k in authored["sita"])
    for idx in picked:
        cue = sita_en[idx]
        zh1 = zh_by_start.get(cue["start"])
        if not zh1:
            sys.exit(f"sita[{idx}] {cue['start']}: no zh-Hant cue at that timestamp")
        rows.append({
            "id": f"sita-{idx:03d}", "source": "sita", "start": cue["start"], "end": cue["end"],
            "text": cue["text"], "refs": [zh1, authored["sita"][str(idx)]],
            "names": names_in(cue["text"], authored["names"]["sita"]), "traps": [],
            "attribution": ATTRIBUTION["sita"],
        })

    # ── Traps: hand-written; synthetic 3-second timeline so time_shift pairs are adjacent ──
    for n, trap in enumerate(authored["traps"]):
        t0 = n * 3
        start = f"01:{30 + t0 // 60:02d}:{t0 % 60:02d},000"
        end = f"01:{30 + (t0 + 2) // 60:02d}:{(t0 + 2) % 60:02d},500"
        row = {
            "id": trap["id"], "source": "trap", "start": start, "end": end, "text": trap["text"],
            "refs": trap["refs"], "names": trap.get("names", {}), "traps": [trap["cat"]],
            "attribution": ATTRIBUTION["trap"],
        }
        for opt in ("anchors", "forbid", "note"):
            if opt in trap:
                row[opt] = trap[opt]
        rows.append(row)

    out = Path(args.out)
    with out.open("w", encoding="utf-8") as fh:
        for r in rows:
            fh.write(json.dumps(r, ensure_ascii=False) + "\n")
    counts = {}
    for r in rows:
        counts[r["source"]] = counts.get(r["source"], 0) + 1
    print(f"wrote {out} — {len(rows)} cues {counts}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
