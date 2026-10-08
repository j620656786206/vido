#!/usr/bin/env python3
"""On-screen text spike (spike-onscreen-text-vision.md).

Question: can a vision model, shown sampled frames of an episode, find the
on-screen text a viewer needs translated (time cards, book covers, signs,
notes) — without a local OCR — and what does one episode cost?

Pipeline (everything a NAS container can also run: ffmpeg + one API):
  1. ffmpeg: one frame every STEP seconds. Optionally drop frames that look
     like the last one kept (perceptual hash, --dedupe), then tile the rest
     N x N per image (--grid, --tile-width).
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
      [--grid 4 --tile-width 384 --dedupe 10]     # the second run's settings
  The first run (2026-10-08) was the defaults: --grid 3 --tile-width 512, no dedupe.
  Needs: pip install anthropic pillow
"""

import argparse
import base64
import concurrent.futures
import io
import json
import os
import pathlib
import subprocess
import sys
import threading

import anthropic
from PIL import Image

# $ per million tokens (input, output) — claude-api skill price table, cached 2026-09-25.
PRICES = {
    "claude-opus-5-5": (4.00, 20.00),
    "claude-sonnet-5-5": (2.00, 10.00),
    "claude-haiku-4-5": (1.00, 5.00),
}

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


def extract_frames(video: str, out: pathlib.Path, step: int, width: int) -> list[pathlib.Path]:
    """One frame every `step` seconds at `width` px; frame i is at i*step s."""
    out.mkdir(parents=True, exist_ok=True)
    if not any(out.glob("f*.jpg")):
        subprocess.run(
            ["ffmpeg", "-nostdin", "-v", "error", "-y", "-i", video, "-an", "-sn",
             "-vf", f"fps=1/{step},scale={width}:-2", "-q:v", "4", str(out / "f%05d.jpg")],
            check=True,
        )
    return sorted(out.glob("f*.jpg"))


def dhash(path: pathlib.Path) -> int:
    """8x8 difference hash: a frame's rough shape, blind to small detail."""
    img = Image.open(path).convert("L").resize((9, 8))
    px = list(img.getdata())
    h = 0
    for r in range(8):
        for c in range(8):
            h = (h << 1) | (px[r * 9 + c] > px[r * 9 + c + 1])
    return h


def pick_frames(frames: list[pathlib.Path], threshold: int, safety_every: int) -> list[int]:
    """Indexes worth showing the model: a frame that looks different enough
    from the last one kept (perceptual-hash dedupe), plus one every
    `safety_every` frames so a slow fade-in inside a still shot is not lost.
    threshold 0 keeps everything (the first run)."""
    if threshold <= 0:
        return list(range(len(frames)))
    hashes = [dhash(f) for f in frames]
    kept = [0]
    for i in range(1, len(frames)):
        if bin(hashes[i] ^ hashes[kept[-1]]).count("1") > threshold or i % safety_every == 0:
            kept.append(i)
    return kept


def make_grid(paths: list[pathlib.Path], cols: int) -> bytes:
    tiles = [Image.open(p) for p in paths]
    w, h = tiles[0].size
    rows = (len(tiles) + cols - 1) // cols
    sheet = Image.new("RGB", (w * cols, h * rows))
    for k, t in enumerate(tiles):
        sheet.paste(t, ((k % cols) * w, (k // cols) * h))
    buf = io.BytesIO()
    sheet.save(buf, "JPEG", quality=80)
    return buf.getvalue()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("video")
    ap.add_argument("outdir")
    ap.add_argument("--model", default="claude-opus-5-5")
    ap.add_argument("--effort", default="low")
    ap.add_argument("--step", type=int, default=2)
    ap.add_argument("--budget", type=float, default=3.0, help="stop once spend reaches this many USD")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--grid", type=int, default=3, help="tiles per side (3 → 3x3)")
    ap.add_argument("--tile-width", type=int, default=512)
    ap.add_argument("--dedupe", type=int, default=0,
                    help="perceptual-hash distance (bits of 64) below which a frame counts as a repeat; 0 = keep all")
    ap.add_argument("--safety-every", type=int, default=10, help="with --dedupe, still keep every Nth frame")
    args = ap.parse_args()
    cols = args.grid
    tiles_per = cols * cols

    key = os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("CLAUDE_API_KEY")
    client = anthropic.Anthropic(api_key=key)
    price_in, price_out = PRICES[args.model]
    out = pathlib.Path(args.outdir)
    frames = extract_frames(args.video, out / f"frames-{args.step}s-{args.tile_width}", args.step, args.tile_width)
    kept = pick_frames(frames, args.dedupe, args.safety_every)
    grids = [kept[i:i + tiles_per] for i in range(0, len(kept), tiles_per)]
    print(f"{len(frames)} frames, {len(kept)} kept, {len(grids)} grids", file=sys.stderr)

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

    def run(i: int, idxs: list[int]):
        with lock:
            if spent["usd"] >= args.budget:
                return
        secs = [k * args.step for k in idxs]
        times = ", ".join(f"{n + 1}={mmss(sec)}" for n, sec in enumerate(secs))
        got = ask(make_grid([frames[k] for k in idxs], cols),
                  PROMPT.format(cols=cols, rows=(len(idxs) + cols - 1) // cols, times=times, tiles=len(idxs)))
        for it in got["items"]:
            it["sec"] = secs[max(1, min(len(secs), it["tile"])) - 1]
            it["stage"] = "grid"
        with lock:
            raw[i] = got["items"]
            for st in got["small_text"]:
                zoom_requests.append((secs[max(1, min(len(secs), st["tile"])) - 1], st["what"]))

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
        futures = [pool.submit(run, i, g) for i, g in enumerate(grids)]
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
        "grid": f"{cols}x{cols}", "tile_width": args.tile_width, "dedupe": args.dedupe,
        "frames": len(frames), "frames_kept": len(kept),
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
