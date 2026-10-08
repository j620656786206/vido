#!/usr/bin/env python3
"""On-screen text spike (spike-onscreen-text-vision.md).

Question: can a vision model, shown sampled frames of an episode, find the
on-screen text a viewer needs translated (time cards, book covers, signs,
notes) — without a local OCR — and what does one episode cost?

Pipeline (everything a NAS container can also run: ffmpeg + one API):
  1. ffmpeg: one frame every STEP seconds, 512 px wide, tiled 3x3 per image.
  2. Claude (official SDK, structured JSON output): per grid image, list the
     on-screen text with tile number, English text, kind, whether it needs
     translating, and a Taiwan Traditional Chinese rendering — plus the tiles
     that show text too small to read at 512 px (a book held up).
  2b. Those frames are taken again at full resolution and read one by one.
  3. Merge the same text seen in adjacent tiles into one finding; write
     findings.json with times, and usage/cost.

The per-line results stay out of git (the repo is public); the spike doc keeps
only the numbers.

Usage:
  python3 -m venv .venv && .venv/bin/pip install anthropic
  export CLAUDE_API_KEY=...         # or ANTHROPIC_API_KEY
  .venv/bin/python scripts/onscreen-text-spike.py VIDEO.mkv OUTDIR \\
      [--model claude-opus-5-5] [--effort low] [--step 2] [--budget 3.0]
"""

import argparse
import base64
import concurrent.futures
import json
import os
import pathlib
import subprocess
import sys
import threading

import anthropic

# $ per million tokens (input, output) — claude-api skill price table, cached 2026-09-25.
PRICES = {
    "claude-opus-5-5": (4.00, 20.00),
    "claude-sonnet-5-5": (2.00, 10.00),
    "claude-haiku-4-5": (1.00, 5.00),
}
COLS = ROWS = 3
TILES = COLS * ROWS

SCHEMA = {
    "type": "object",
    "properties": {
        "items": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "tile": {"type": "integer"},
                    "text": {"type": "string"},
                    "kind": {
                        "type": "string",
                        "enum": ["caption", "book", "sign", "note", "screen", "credits", "logo", "other"],
                    },
                    "needs_translation": {"type": "boolean"},
                    "zh": {"type": "string"},
                },
                "required": ["tile", "text", "kind", "needs_translation", "zh"],
                "additionalProperties": False,
            },
        },
        "small_text": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {"tile": {"type": "integer"}, "what": {"type": "string"}},
                "required": ["tile", "what"],
                "additionalProperties": False,
            },
        },
    },
    "required": ["items", "small_text"],
    "additionalProperties": False,
}

ZOOM_PROMPT = """This is one full-resolution frame from a TV episode at {time}. \
A smaller view suggested it shows {what}. List the on-screen text a viewer \
would need translated, exactly as above: tile is always 1; give the exact \
English text, kind, needs_translation, and a natural Taiwan Traditional \
Chinese rendering (a published book gets its established Taiwan title). \
Leave small_text empty. If no text is actually readable, return empty lists."""

PROMPT = """This image is a {cols}x{rows} grid of frames sampled from a TV episode, \
read left to right, top to bottom. Tile times: {times}.

List the text that is visible ON SCREEN in these frames and that a viewer \
watching with Chinese subtitles would need translated to follow the story: \
time or place cards ("3 YEARS LATER"), readable book covers or titles, signs, \
handwritten notes or letters, screens.

For each one give the tile number (1-{tiles}), the exact English text, its \
kind, needs_translation, and a natural Taiwan Traditional Chinese rendering \
(for a published book use its established Taiwan title).

Also list opening/closing credits, studio logos and watermarks if you see \
them, with needs_translation false. Do not list dialogue subtitles, text you \
cannot actually read, or the same text twice within one tile. If there is \
nothing, return an empty list.

Separately, in small_text, list the tiles that show an object which clearly \
carries text you cannot read at this size — a book cover held up, a page, a \
sign, a screen — with a few words on what it is. Those frames will be looked \
at again at full resolution."""


def mmss(sec: float) -> str:
    return f"{int(sec // 60)}:{int(sec % 60):02d}"


def make_grids(video: str, out: pathlib.Path, step: int) -> list[pathlib.Path]:
    out.mkdir(parents=True, exist_ok=True)
    if not any(out.glob("g*.jpg")):
        subprocess.run(
            ["ffmpeg", "-nostdin", "-v", "error", "-y", "-i", video,
             "-vf", f"fps=1/{step},scale=512:-2,tile={COLS}x{ROWS}", "-q:v", "4",
             str(out / "g%04d.jpg")],
            check=True,
        )
    return sorted(out.glob("g*.jpg"))


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("video")
    ap.add_argument("outdir")
    ap.add_argument("--model", default="claude-opus-5-5")
    ap.add_argument("--effort", default="low")
    ap.add_argument("--step", type=int, default=2)
    ap.add_argument("--budget", type=float, default=3.0, help="stop once spend reaches this many USD")
    ap.add_argument("--workers", type=int, default=4)
    args = ap.parse_args()

    key = os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("CLAUDE_API_KEY")
    client = anthropic.Anthropic(api_key=key)
    price_in, price_out = PRICES[args.model]
    out = pathlib.Path(args.outdir)
    grids = make_grids(args.video, out / "grids", args.step)

    lock = threading.Lock()
    spent = {"usd": 0.0, "in": 0, "out": 0}
    raw: dict[int, list] = {}

    def ask(data: bytes, prompt: str) -> dict:
        resp = client.messages.create(
            model=args.model,
            max_tokens=8000,
            output_config={"effort": args.effort, "format": {"type": "json_schema", "schema": SCHEMA}},
            messages=[{"role": "user", "content": [
                {"type": "image", "source": {"type": "base64", "media_type": "image/jpeg",
                                             "data": base64.standard_b64encode(data).decode("utf-8")}},
                {"type": "text", "text": prompt},
            ]}],
        )
        u = resp.usage
        with lock:
            spent["in"] += u.input_tokens
            spent["out"] += u.output_tokens
            spent["usd"] += u.input_tokens / 1e6 * price_in + u.output_tokens / 1e6 * price_out
        if resp.stop_reason == "refusal":
            print("refused", file=sys.stderr)
            return {"items": [], "small_text": []}
        text = next((b.text for b in resp.content if b.type == "text"), "")
        return json.loads(text) if text else {"items": [], "small_text": []}

    zoom_requests: list[tuple[int, str]] = []

    def run(i: int, path: pathlib.Path):
        with lock:
            if spent["usd"] >= args.budget:
                return
        first = i * TILES
        times = ", ".join(f"{k + 1}={mmss((first + k) * args.step)}" for k in range(TILES))
        got = ask(path.read_bytes(), PROMPT.format(cols=COLS, rows=ROWS, times=times, tiles=TILES))
        for it in got["items"]:
            it["sec"] = (first + max(1, min(TILES, it["tile"])) - 1) * args.step
            it["stage"] = "grid"
        with lock:
            raw[i] = got["items"]
            for st in got["small_text"]:
                zoom_requests.append(((first + max(1, min(TILES, st["tile"])) - 1) * args.step, st["what"]))

    def zoom(sec: int, what: str):
        with lock:
            if spent["usd"] >= args.budget:
                return
        frame = subprocess.run(
            ["ffmpeg", "-nostdin", "-v", "error", "-ss", str(sec), "-i", args.video,
             "-frames:v", "1", "-f", "image2", "-c:v", "mjpeg", "-q:v", "3", "-"],
            check=True, capture_output=True).stdout
        got = ask(frame, ZOOM_PROMPT.format(time=mmss(sec), what=what))
        for it in got["items"]:
            it["sec"] = sec
            it["stage"] = "zoom"
        with lock:
            raw[100000 + sec] = got["items"]

    with concurrent.futures.ThreadPoolExecutor(args.workers) as pool:
        futures = [pool.submit(run, i, p) for i, p in enumerate(grids)]
        for n, f in enumerate(concurrent.futures.as_completed(futures), 1):
            f.result()
            if n % 20 == 0:
                print(f"{n}/{len(grids)} grids, ${spent['usd']:.3f}", file=sys.stderr)

    # Stage 2: the frames the grid pass could see text in but not read.
    # One look per sampled frame. Not "one per shot": the first run skipped
    # 55:08 because 55:06 had been zoomed, and the second book in the scene
    # (1984) only shows up readable in the later frame.
    todo = sorted({sec: what for sec, what in zoom_requests}.items())
    print(f"zooming into {len(todo)} frames", file=sys.stderr)
    with concurrent.futures.ThreadPoolExecutor(args.workers) as pool:
        for f in [pool.submit(zoom, sec, what) for sec, what in todo]:
            f.result()

    # Merge the same text seen in frames less than 3 steps apart.
    flat = sorted((it for items in raw.values() for it in items), key=lambda x: x["sec"])
    findings = []
    for it in flat:
        norm = " ".join(it["text"].lower().split())
        prev = next((f for f in reversed(findings) if f["norm"] == norm), None)
        if prev and it["sec"] - prev["end"] <= 3 * args.step:
            prev["end"] = it["sec"]
            continue
        findings.append({**it, "norm": norm, "start": it["sec"], "end": it["sec"]})
    for f in findings:
        f["time"] = mmss(f["start"])

    result = {
        "model": args.model, "effort": args.effort, "step_seconds": args.step,
        "grids": len(grids), "zoomed_frames": len(todo),
        "input_tokens": spent["in"], "output_tokens": spent["out"], "usd": round(spent["usd"], 4),
        "findings": [{k: f[k] for k in ("time", "start", "end", "text", "kind", "needs_translation", "zh", "stage")} for f in findings],
    }
    (out / "findings.json").write_text(json.dumps(result, ensure_ascii=False, indent=1))
    print(json.dumps({k: v for k, v in result.items() if k != "findings"}, ensure_ascii=False))
    print(f"{len(findings)} findings → {out / 'findings.json'}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
