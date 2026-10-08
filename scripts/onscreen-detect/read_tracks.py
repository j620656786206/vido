"""Round 8 of the on-screen text spike: read each text track once.

Takes the tracks from track.py, crops each track's best frame around its box,
puts a small whole-frame thumbnail (box drawn in red) next to it so the model
can tell a shot's subject from background text, packs the cells into sheets
and asks Claude to read them. One look per track — the YORO idea.

  python read_tracks.py FRAMES_DIR TRACKS.json OUTDIR [--budget 1.0]
  (CLAUDE_API_KEY or ANTHROPIC_API_KEY in the environment)
"""
import argparse, base64, concurrent.futures, io, json, os, pathlib, sys, threading
import anthropic
from PIL import Image, ImageDraw

PRICE_IN, PRICE_OUT = 4.00, 20.00  # claude-opus-5-5, $ per million tokens (same table as onscreen-text-spike.py)
STORY_KINDS = {"caption", "book", "sign", "note", "screen", "other"}
CROP_W, CROP_H, THUMB_W, LABEL = 240, 120, 128, 16
COLS, ROWS = 3, 7

SCHEMA = {
    "type": "object",
    "properties": {"items": {"type": "array", "items": {
        "type": "object",
        "properties": {
            "cell": {"type": "integer"},
            "text": {"type": "string"},
            "kind": {"type": "string", "enum": ["caption", "book", "sign", "note", "screen", "credits", "logo", "other"]},
            "focus": {"type": "boolean"},
            "zh": {"type": "string"},
        },
        "required": ["cell", "text", "kind", "focus", "zh"],
        "additionalProperties": False,
    }}},
    "required": ["items"],
    "additionalProperties": False,
}

PROMPT = """This sheet has {n} numbered cells from one TV episode. In each cell the \
LEFT picture is a zoomed crop of a spot where a text detector thinks there is \
text; the RIGHT picture is the whole frame, with that spot boxed in red. Many \
cells are false alarms (trees, faces, textures) — skip those.

List only the cells whose crop shows text you can actually read. For each: the \
cell number, the exact English text, its kind, focus, and a natural Taiwan \
Traditional Chinese rendering (a published book gets its established Taiwan \
title; proper names — people, places, shops — are transliterated the way \
Taiwan subtitles do). Do NOT list opening/closing credits, studio logos or \
watermarks — they are never translated.

focus: use the whole-frame picture. True when the SHOT is showing this text to \
the viewer — a close-up or insert of it, held up or near the centre and in \
focus, or a card laid over the picture ("3 YEARS LATER"). Background text — \
street and parking signs, licence plates, posters on a wall, book spines on a \
shelf, words on clothing, a passing van — is false unless the camera is \
pointing at it. Keep each entry short."""


def dhash(im):
    g = im.convert("L").resize((9, 8))
    p = list(g.getdata())
    return sum(1 << i for i in range(64) if p[(i // 8) * 9 + i % 8] > p[(i // 8) * 9 + i % 8 + 1])


def cell(frame, box):
    W, H = frame.size
    x1, y1, x2, y2 = box
    pw, ph = (x2 - x1) * 0.3 + 8, (y2 - y1) * 0.3 + 8
    crop = frame.crop((max(0, x1 - pw), max(0, y1 - ph), min(W, x2 + pw), min(H, y2 + ph)))
    s = min(CROP_W / crop.width, CROP_H / crop.height, 3.0)
    crop = crop.resize((max(1, int(crop.width * s)), max(1, int(crop.height * s))))
    thumb = frame.copy()
    ImageDraw.Draw(thumb).rectangle(box, outline=(255, 0, 0), width=6)
    thumb = thumb.resize((THUMB_W, int(THUMB_W * H / W)))
    return crop, thumb


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("frames"); ap.add_argument("tracks"); ap.add_argument("out")
    ap.add_argument("--budget", type=float, default=1.0)
    ap.add_argument("--workers", type=int, default=4)
    args = ap.parse_args()
    out = pathlib.Path(args.out); (out / "sheets").mkdir(parents=True, exist_ok=True)
    tracks = sorted(json.load(open(args.tracks))["track_list"], key=lambda t: t["best"])

    # Build cells; drop a crop that repeats one from the last two minutes.
    cells, recent = [], []
    for t in tracks:
        frame = Image.open(f"{args.frames}/f{t['best'] // 2 + 1:05d}.jpg").convert("RGB")
        crop, thumb = cell(frame, t["box"])
        h = dhash(crop)
        recent = [(s, x) for s, x in recent if t["best"] - s <= 120]
        if any(bin(h ^ x).count("1") <= 6 for _, x in recent):
            continue
        recent.append((t["best"], h))
        cells.append((t, crop, thumb))

    cw, chh = CROP_W + 6 + THUMB_W, LABEL + max(CROP_H, THUMB_W * 9 // 16 + 4)
    per = COLS * ROWS
    sheets = []
    for k in range(0, len(cells), per):
        group = cells[k:k + per]
        sheet = Image.new("RGB", (COLS * cw, ((len(group) + COLS - 1) // COLS) * chh), "white")
        d = ImageDraw.Draw(sheet)
        for i, (t, crop, thumb) in enumerate(group):
            x, y = (i % COLS) * cw, (i // COLS) * chh
            d.text((x + 2, y + 2), str(i + 1), fill="black")
            sheet.paste(crop, (x, y + LABEL)); sheet.paste(thumb, (x + CROP_W + 6, y + LABEL))
        p = out / "sheets" / f"s{len(sheets) + 1:03d}.jpg"
        sheet.save(p, quality=88)
        sheets.append((p, group))
    print(f"{len(tracks)} tracks → {len(cells)} cells → {len(sheets)} sheets", file=sys.stderr)

    client = anthropic.Anthropic(api_key=os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("CLAUDE_API_KEY"))
    lock, spent, found = threading.Lock(), {"usd": 0.0, "in": 0, "out": 0}, []

    def ask(p, group):
        with lock:
            if spent["usd"] >= args.budget:
                return
        r = client.messages.create(
            model="claude-opus-5-5", max_tokens=4000,
            output_config={"effort": "low", "format": {"type": "json_schema", "schema": SCHEMA}},
            messages=[{"role": "user", "content": [
                {"type": "image", "source": {"type": "base64", "media_type": "image/jpeg",
                                             "data": base64.standard_b64encode(p.read_bytes()).decode()}},
                {"type": "text", "text": PROMPT.format(n=len(group))}]}])
        u = r.usage
        text = next((b.text for b in r.content if b.type == "text"), "")
        items = json.loads(text)["items"] if text and r.stop_reason != "refusal" else []
        with lock:
            spent["in"] += u.input_tokens; spent["out"] += u.output_tokens
            spent["usd"] += u.input_tokens / 1e6 * PRICE_IN + u.output_tokens / 1e6 * PRICE_OUT
            for it in items:
                if 1 <= it["cell"] <= len(group):
                    t = group[it["cell"] - 1][0]
                    found.append({**it, "sec": t["best"], "start": t["start"], "end": t["end"], "sheet": p.name,
                                  "translate": it["kind"] in STORY_KINDS and it["focus"]})

    with concurrent.futures.ThreadPoolExecutor(args.workers) as pool:
        for f in [pool.submit(ask, p, g) for p, g in sheets]:
            f.result()
    found.sort(key=lambda x: x["sec"])
    res = {"tracks": len(tracks), "cells": len(cells), "sheets": len(sheets),
           "input_tokens": spent["in"], "output_tokens": spent["out"], "usd": round(spent["usd"], 4),
           "findings": found}
    (out / "findings.json").write_text(json.dumps(res, ensure_ascii=False, indent=1))
    print(json.dumps({k: v for k, v in res.items() if k != "findings"}))


if __name__ == "__main__":
    sys.exit(main())
