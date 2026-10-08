"""Round 9 of the on-screen text spike: a second look at "background" verdicts.

Round 8 read the text well but judged focus on a 128-px thumbnail, and called
a held-up book and a phone screen background. This takes read_tracks.py's
findings, picks the books, notes and screens judged background (one per
distinct text), and asks again on a 768-px whole frame with the box drawn.

  python refocus.py FRAMES_DIR TRACKS.json OUTDIR   # OUTDIR = read_tracks.py's
"""
import base64, io, json, os, pathlib, sys
import anthropic
from PIL import Image, ImageDraw

sys.path.insert(0, os.path.dirname(__file__))
from read_tracks import PRICE_IN, PRICE_OUT, STORY_KINDS, COLS, ROWS, build_cells

RECHECK_KINDS = {"book", "note", "screen"}
WIDTH = 768
SCHEMA = {"type": "object", "properties": {"focus": {"type": "boolean"}}, "required": ["focus"],
          "additionalProperties": False}
PROMPT = """This is a frame from a TV episode. The text "{text}" is inside the red box.

Is the SHOT showing this text to the viewer? True for a close-up or insert of \
it, or when it is held up, near the centre, or the thing the shot is about \
(a phone screen someone is looking at, a book a character is showing). False \
for background text the camera just happens to catch — signs, plates, \
posters, book spines on a shelf, words on clothing."""


def main():
    frames, tracks_path, out = sys.argv[1], sys.argv[2], pathlib.Path(sys.argv[3])
    res = json.loads((out / "findings.json").read_text())
    tracks = sorted(json.load(open(tracks_path))["track_list"], key=lambda t: t["best"])
    cells = build_cells(frames, tracks)
    per = COLS * ROWS

    def track_of(f):  # findings written before "box" was stored
        k = int(f["sheet"][1:4]) - 1
        t = cells[k * per + f["cell"] - 1][0]
        assert t["best"] == f["sec"], "cells rebuilt differently from round 8"
        return t

    todo = {}
    for f in res["findings"]:
        if f["kind"] in RECHECK_KINDS and not f["focus"]:
            t = track_of(f)
            a = (t["box"][2] - t["box"][0]) * (t["box"][3] - t["box"][1])
            key = " ".join(f["text"].lower().split())
            if key not in todo or a > todo[key][1]:
                todo[key] = (f, a, t)

    client = anthropic.Anthropic(api_key=os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("CLAUDE_API_KEY"))
    usd, flipped = 0.0, set()
    for key, (f, _, t) in todo.items():
        im = Image.open(f"{frames}/f{t['best'] // 2 + 1:05d}.jpg").convert("RGB")
        ImageDraw.Draw(im).rectangle(t["box"], outline=(255, 0, 0), width=4)
        im = im.resize((WIDTH, int(WIDTH * im.height / im.width)))
        buf = io.BytesIO(); im.save(buf, "JPEG", quality=88)
        r = client.messages.create(
            model="claude-opus-5-5", max_tokens=200,
            output_config={"effort": "low", "format": {"type": "json_schema", "schema": SCHEMA}},
            messages=[{"role": "user", "content": [
                {"type": "image", "source": {"type": "base64", "media_type": "image/jpeg",
                                             "data": base64.standard_b64encode(buf.getvalue()).decode()}},
                {"type": "text", "text": PROMPT.format(text=f["text"])}]}])
        usd += r.usage.input_tokens / 1e6 * PRICE_IN + r.usage.output_tokens / 1e6 * PRICE_OUT
        focus = json.loads(next(b.text for b in r.content if b.type == "text"))["focus"]
        print(f"{f['sec'] // 60}:{f['sec'] % 60:02d}  {f['text'][:40]!r} → focus {focus}", file=sys.stderr)
        if focus:
            flipped.add(key)

    for f in res["findings"]:
        if " ".join(f["text"].lower().split()) in flipped and f["kind"] in RECHECK_KINDS:
            f["focus"], f["refocused"] = True, True
            f["translate"] = f["kind"] in STORY_KINDS
    res["refocus"] = {"checked": len(todo), "flipped": len(flipped), "usd": round(usd, 4)}
    res["usd_total"] = round(res["usd"] + usd, 4)
    (out / "findings2.json").write_text(json.dumps(res, ensure_ascii=False, indent=1))
    print(json.dumps({"refocus": res["refocus"], "usd_total": res["usd_total"]}))


if __name__ == "__main__":
    sys.exit(main())
