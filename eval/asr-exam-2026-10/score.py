#!/usr/bin/env python3
"""聽寫考卷評分（spike-asr-exam-2026-10）。

答案 = 片子附的官方英文字幕（answer.en.srt）。四個分數，規則就是 story 裡那張表：

  亂補率   輸出的句子，有一半以上時間落在「官方沒人說話」的區間 → 亂補。亂補 ÷ 輸出句數。≤ 5% 及格。
  漏句率   官方句子完全沒有任何輸出句子跟它重疊 → 漏。漏 ÷ 官方句數。≤ 10% 及格。
  時間誤差 有對上的官方句子，和跟它重疊最多的輸出句子，開口時間差（輸出把兩句併成一句的不計）。中位 ≤ 0.5 秒、p90 ≤ 1.0 秒及格。
  錯字率   整份輸出對整份答案的 WER（正規化後），只排名用。

「沒人說話」= 官方句子前後各放寬 0.5 秒之外的時間。

用法：python3 score.py runs/*.json runs/*.srt
支援的輸入：OpenAI verbose_json（whisper-1、VoiceStudio）、Mistral Voxtral（segments）、
ElevenLabs Scribe（words，自己切句）、SRT。
"""
import json
import re
import statistics
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
PAD = 0.5           # 官方句子前後放寬
HALLU_SHARE = 0.5   # 一半以上時間在沒人說話處 → 亂補
LIMITS = {"hallu": 0.05, "missed": 0.10, "t_med": 0.5, "t_p90": 1.0}

MUSIC = re.compile(r"^[\s♪¶\-–—.…]*$")
TAG = re.compile(r"\[[^\]]*\]|\([^)]*\)")


def srt_time(s):
    h, m, rest = s.strip().split(":")
    sec, ms = rest.replace(".", ",").split(",")
    return int(h) * 3600 + int(m) * 60 + int(sec) + int(ms) / 1000


def load_srt(path):
    cues = []
    for block in Path(path).read_text(encoding="utf-8-sig").replace("\r", "").strip().split("\n\n"):
        lines = block.split("\n")
        if len(lines) < 3 or "-->" not in lines[1]:
            continue
        a, b = lines[1].split("-->")
        text = " ".join(lines[2:])
        cues.append((srt_time(a), srt_time(b), text))
    return cues


def load_output(path):
    """→ [(start, end, text)]，去掉空句、純音樂記號、音效標記。"""
    path = Path(path)
    if path.suffix == ".srt":
        cues = load_srt(path)
    else:
        d = json.loads(path.read_text(encoding="utf-8"))
        if "words" in d and "segments" not in d:          # ElevenLabs Scribe
            cues = scribe_cues(d["words"])
        else:                                              # OpenAI verbose_json / Voxtral
            cues = [(s["start"], s["end"], s.get("text", "")) for s in d.get("segments", [])]
    out = []
    for a, b, t in cues:
        t = TAG.sub("", t).strip()
        if not t or MUSIC.match(t):
            continue
        out.append((a, b, t))
    return out


def scribe_cues(words, gap=0.8):
    """Scribe 只給逐字；照句尾標點和 0.8 秒以上的空隙切成句子。"""
    cues, cur = [], []
    for w in words:
        if w.get("type") != "word":
            if w.get("type") == "audio_event" and cur:
                cues.append(cur); cur = []
            continue
        if cur and w["start"] - cur[-1]["end"] > gap:
            cues.append(cur); cur = []
        cur.append(w)
        if re.search(r"[.?!]$", w["text"].strip()):
            cues.append(cur); cur = []
    if cur:
        cues.append(cur)
    return [(c[0]["start"], c[-1]["end"], " ".join(w["text"].strip() for w in c)) for c in cues]


def overlap(a, b, windows):
    return sum(max(0.0, min(b, y) - max(a, x)) for x, y in windows)


def norm_words(text):
    text = TAG.sub("", text).lower().replace("’", "'")
    text = re.sub(r"[^a-z0-9' ]+", " ", text)
    return text.split()


def wer(ref, hyp):
    r, h = norm_words(ref), norm_words(hyp)
    prev = list(range(len(h) + 1))
    for i, rw in enumerate(r, 1):
        cur = [i]
        for j, hw in enumerate(h, 1):
            cur.append(min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + (rw != hw)))
        prev = cur
    return prev[-1] / max(1, len(r))


def score(output, answer):
    windows = [(a - PAD, b + PAD) for a, b, _ in answer]
    hallu = 0
    for a, b, _ in output:
        dur = b - a
        inside = overlap(a, b, windows) if dur > 0 else (PAD if any(x <= a <= y for x, y in windows) else 0)
        if dur > 0 and inside / dur < HALLU_SHARE:
            hallu += 1
        elif dur == 0 and inside == 0:
            hallu += 1
    # 時間誤差只量「一句對一句」的情況：跟官方句子重疊最多的那句輸出，
    # 如果同時也蓋到前一句官方句子（輸出把兩句併成一句），那是切句不同，不是時間錯，不計。
    missed, diffs, prev = 0, [], None
    for a, b, _ in answer:
        hits = [o for o in output if min(b, o[1]) - max(a, o[0]) > -PAD]
        if not hits:
            missed += 1
            prev = (a, b)
            continue
        # 官方一句常是輸出的好幾句（「Who's there? Answer me. Who's there?」）：
        # 拿真的壓到這句時間的輸出裡最早開口的那句來量。
        inside = [o for o in hits if min(b, o[1]) - max(a, o[0]) > 0]
        best = min(inside, key=lambda o: o[0]) if inside else max(hits, key=lambda o: min(b, o[1]) - max(a, o[0]))
        merged = prev is not None and min(prev[1], best[1]) - max(prev[0], best[0]) > 0
        if not merged:
            diffs.append(abs(best[0] - a))
        prev = (a, b)
    diffs.sort()
    n = len(diffs)
    p90 = diffs[min(n - 1, int(0.9 * n))] if n else float("nan")
    return {
        "cues": len(output),
        "hallu": hallu / max(1, len(output)),
        "missed": missed / max(1, len(answer)),
        "t_med": statistics.median(diffs) if diffs else float("nan"),
        "t_p90": p90,
        "wer": wer(" ".join(t for _, _, t in answer), " ".join(t for _, _, t in output)),
    }


def passed(s):
    return s["hallu"] <= LIMITS["hallu"] and s["missed"] <= LIMITS["missed"] \
        and s["t_med"] <= LIMITS["t_med"] and s["t_p90"] <= LIMITS["t_p90"]


def main(paths):
    answer = load_srt(HERE / "answer.en.srt")
    print(f"答案：{len(answer)} 句（官方英文字幕，前 10 分鐘）\n")
    print("| 檔案 | 句數 | 亂補率 | 漏句率 | 時間中位 | 時間 p90 | 錯字率 | 過？ |")
    print("|---|---|---|---|---|---|---|---|")
    groups = {}
    for p in sorted(paths):
        s = score(load_output(p), answer)
        ok = passed(s)
        name = Path(p).stem
        groups.setdefault(re.sub(r"-\d+$", "", name), []).append(ok)
        print(f"| {name} | {s['cues']} | {s['hallu']:.1%} | {s['missed']:.1%} | {s['t_med']:.2f}s | {s['t_p90']:.2f}s | {s['wer']:.1%} | {'✅' if ok else '❌'} |")
    print()
    for g, oks in groups.items():
        if len(oks) > 1:
            print(f"{g}：{sum(oks)}/{len(oks)} 次過 → {'✅ 過關' if all(oks) else '❌ 沒過'}")


if __name__ == "__main__":
    main(sys.argv[1:])
