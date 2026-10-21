# Sidecar alignment exam (disc-2026-10-generate-keeps-user-en-srt-b1, AC #5)

Date: 2026-10-20. Run on the production NAS (Unraid, x86_64) inside a throwaway container of the production image, with the media mounted read-only. The production container was never touched.

This file names no library file. Films are labelled by kind. The one exception is Tears of Steel, which is an open film (CC BY 3.0, © Blender Foundation).

## How to rerun

```sh
cd apps/api
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c -o sidecarcheck.test ./internal/subtitle/sidecarcheck/
# copy sidecarcheck.test and an exam.json to the NAS, then:
docker run --rm -e VIDO_ALIGN_EXAM=/exam/exam.json \
  -v /mnt/user/data/media:/media:ro -v <dir>:/exam \
  --entrypoint /exam/sidecarcheck.test ghcr.io/j620656786206/vido:main \
  -test.run TestAlignmentExam -test.v
```

`exam.json` lists `{"label", "media", "stream", "subtitle" | "subtitle_stream"}`. `stream` is the absolute ffmpeg index of the English audio track. An embedded `subtitle_stream` is extracted once and kept beside `exam.json`. Extracting it reads the whole file, which takes about ten minutes for a 99 GB remux. Optional settings:

- `VIDO_ALIGN_WINDOWS=n` tries another window count.
- `VIDO_ALIGN_DUMP=path.json` saves every per-window score curve, for offline tuning.

## Films

| Label | Kind | Length | Audio | English subtitle (the "original") |
|---|---|---|---|---|
| A | Tears of Steel, short open film, music almost throughout | 12 min | stereo MP3 | official `.en.srt` beside the file |
| B | 1080p feature | 2 h 37 min | stereo AC-3 | embedded SubRip track |
| C | 4K action film, WEBRip | 2 h 05 min | 5.1 AAC | embedded SubRip track |
| D | 4K Blu-ray remux, 99 GB | 2 h 25 min | 5.1 AC-3 | embedded SubRip track ("Full") |
| E | TV episode, dialogue-heavy, partly in another language | 51 min | 5.1 E-AC-3 | embedded SubRip track (not Forced, not SDH) |

## Variants and expected verdicts

| Variant | Made by | Expected |
|---|---|---|
| original | the film's own English subtitle | `aligned` |
| +3s | every cue 3 s later | `offset`, within ±0.3 s of +3 |
| -2s | every cue 2 s earlier | `offset`, within ±0.3 s of −2 |
| fps | every time × 25/23.976 (made for another frame rate) | `mismatch` |
| fps 24 | every time × 24/23.976 (a 0.1% stretch, added after code review) | `mismatch` when the film drifts more than 1 s end to end; for a shorter drift either `aligned` or `mismatch` |
| other film | the next film's original | `mismatch` |

There are 5 films × 6 variants = 30 cells. The story asked for 20 cells: it counted the two shifts as one row, and it had no slow-drift row.

The "fps 24" row was added by me after code review. The expected answer depends on the film's length. Over Tears of Steel's 12 minutes, the subtitle moves 0.7 s from end to end. Translating it would be harmless, so `aligned` is accepted there. Over a feature, the drift is several seconds, so only `mismatch` is right. The story's original 20 cells were not changed.

## Result with the shipped settings: 30 / 30

Settings:

- 12 windows of 90 s, with the centre channel only on 5.1 audio.
- The window curves are z-scored and summed. `minGlobalPeak` is 4.9.
- The first half and the second half of the film are compared to catch a slow drift.
- The English check has no words shared with other languages and no accented letters.

| Film | original | +3s | −2s | fps | fps 24 | other film | check time |
|---|---|---|---|---|---|---|---|
| A | aligned −0.1 s (10.9) | offset +2.9 (10.4) | offset −2.1 (10.0) | mismatch (3.35) | aligned +0.3 (10.0)¹ | mismatch (3.02) | 1.3–1.6 s |
| B | aligned +0.2 (6.8) | offset +3.2 (9.0) | offset −1.8 (6.4) | mismatch (2.06) | mismatch (3.61) | mismatch (2.06) | 2.2–4.5 s |
| C | aligned +0.2 (7.4) | offset +3.0 (7.6) | offset −1.8 (7.3) | mismatch (2.87) | mismatch (4.12) | mismatch (1.83) | 2.8–8.0 s |
| D | aligned 0.0 (9.9) | offset +3.0 (9.4) | offset −2.0 (10.3) | mismatch (2.01) | mismatch (2.97) | mismatch (3.32) | 10.5 s warm, **23–37 s cold** |
| E | aligned +0.1 (8.7) | offset +3.1 (7.1) | offset −1.9 (9.4) | mismatch (2.75) | mismatch (4.59) | mismatch (2.47) | 4.4–11 s |

¹ A 0.7 s drift over a 12-minute short. Both answers are accepted, as explained above.

The number in brackets is the prominence of the summed curve, in standard deviations.

- Weakest real match: 6.40.
- Strongest wrong match: 3.35 without the slow drift. With the slow drift it is 4.59 (episode E).
- The threshold is 4.9. The slow drift leaves the least margin: 0.31 below the threshold, against 1.5 above it for real matches.

## How the settings were reached

| Round | Change | Result |
|---|---|---|
| 1 | 6 windows, whole-mix audio, raw sum, threshold 5 | 19 / 25. B's shifts scored 4.9. C was never found (2.3). One window in E found +14.7 s at the edge of the search and overruled the rest. |
| 2 | Centre channel only on 5.1 layouts. A best shift within 1 s of ±15 s is not "found". | 19 / 25. D and E got much stronger. B was still just under the threshold. C was still lost. In E's −2 s cell, one window found +12.8 s. |
| 3 | Threshold 4.4. Disagreeing windows overrule only when they outnumber agreeing ones. | 22 / 25 with 6 windows. C still failed. With 10 windows, B regressed. |
| 4 | Saved the curves for 6, 8, 10 and 12 windows and compared aggregations offline (below) | — |
| 5 | 12 windows, z-scored sum, threshold 4.9 | **25 / 25** |
| 6 | Code review fixes: half-vs-half drift check, English word list without shared words, accented-letter limit, finite scores, tunables in the cache key. Added the "fps 24" row. | 29 / 30. A's fps 24 was judged aligned at 0.7 s of drift. |
| 7 | "fps 24" expectation made to depend on the drift length (see above) | **30 / 30** |

Gap between the weakest real match and the strongest wrong one (positive = the two groups separate):

| Windows | raw sum | z-scored sum | positive part of z |
|---|---|---|---|
| 6 | −1.45 | −1.36 | −2.62 |
| 8 | −0.24 | +0.45 | +0.58 |
| 10 | +0.10 | −0.31 | −1.33 |
| 12 | +1.87 | **+3.06** | +2.39 |

## Known limits

- **Cold reads of a large remux are slow.** A 90 s window of a 99 GB remux reads all of its interleaved 4K video. Twelve windows read about 11 GB. That took 37 s on a cold cache and 10.5 s warm, against the story's ≤ 15 s target.
  - The verdict is cached per file pair, so this cost is paid once per file.
  - `-b2` should run the check in the background rather than block the estimate.
  - A shorter window was not explored. It is the next thing to try if this matters.
- **The margin is measured on 5 films, and the thresholds were tuned on the same cells.** Code review called this fitted, and I agree there is a risk. Before `-b2` relies on the check, about 10 more films should go through the same exam, especially quiet dramas and anime. The slow-drift row is the closest call (4.59 against 4.9).
- The exam material (extracted subtitles, curve dumps, the test binary) is kept on the NAS at `/mnt/user/data/tmp-b1exam/`, so a rerun does not have to extract the subtitles again.

## Second exam: 10 more films (disc-2026-10-generate-keeps-user-en-srt-b2, AC #0)

Date: 2026-10-20 to 10-21. Same NAS, same throwaway-container setup, same test binary built from `main` (the algorithm and settings are unchanged from the first exam). Thresholds were not touched before or after this run.

### Films

| Label | Kind | English subtitle (the "original") |
|---|---|---|
| F | quiet drama, 4K | embedded SubRip track |
| G | comedy-drama, 4K | embedded SubRip track |
| H | drama, Blu-ray remux, 85 GB | embedded SubRip track |
| I | animated feature | embedded SubRip track |
| J | mystery film | embedded SubRip track |
| K | action film whose only English subtitle is SDH | embedded SubRip track (SDH) |
| L | Japanese anime episode, English dub on the second audio track | embedded SubRip track |
| M | 30-minute comedy episode | embedded SubRip track |
| N | documentary, remux, 7.1 audio | embedded SubRip track |
| O | dialogue-heavy TV episode | embedded SubRip track |

Each film got the same six variants as the first exam. "Other film" is the next film's original (O uses F's).

### Result: 57 / 60

| Film | original | +3s | −2s | fps | fps 24 | other film | first check |
|---|---|---|---|---|---|---|---|
| F | aligned 0.0 (11.6) | offset +3.0 (10.5) | offset −2.0 (11.2) | mismatch (2.43) | mismatch (4.28) | mismatch (2.85) | 25 s |
| G | aligned +0.1 (11.7) | offset +3.1 (12.3) | offset −1.9 (11.1) | mismatch (2.38) | mismatch (3.68) | mismatch (2.45) | 45 s |
| H | aligned 0.0 (8.2) | offset +3.0 (8.8) | offset −2.0 (7.6) | mismatch (1.89) | mismatch (2.70) | mismatch (1.89) | **196 s** |
| I | aligned 0.0 (10.7) | offset +3.0 (10.7) | offset −2.0 (11.0) | mismatch (2.51) | mismatch (3.70) | mismatch (2.74) | 78 s |
| J | aligned 0.0 (10.5) | offset +3.0 (10.0) | offset −2.0 (10.7) | mismatch (3.25) | mismatch (2.15) | mismatch (2.52) | 88 s |
| K | aligned 0.0 (10.7) | offset +3.0 (10.6) | offset −2.0 (10.5) | mismatch (2.83) | mismatch (2.44) | mismatch (3.82) | 29 s |
| L | aligned −0.1 (11.7) | offset +2.9 (9.8) | offset −2.1 (12.2) | mismatch (2.23) | **offset +0.7 (9.72) ✗** | mismatch (3.56) | 25 s |
| M | aligned +0.1 (7.7) | offset +3.1 (7.8) | offset −1.9 (7.8) | mismatch (3.06) | **offset +1.1 (6.31) ✗** | mismatch (4.67) | 119 s |
| N | aligned +0.1 (7.2) | offset +3.1 (7.1) | offset −1.9 (7.3) | mismatch (2.20) | **offset +1.8 (5.30) ✗** | mismatch (1.98) | 124 s (241 s for one later cell) |
| O | aligned +0.1 (6.4) | offset +3.1 (6.5) | offset −1.9 (6.3) | mismatch (2.01) | mismatch (4.14) | mismatch (3.12) | 36 s |

Later cells of the same film usually took 4–30 s, because the file was then in the page cache. The production Vido container was running at the same time (load average about 8), so the first-check times are pessimistic.

- Every original was `aligned`. Every shift was found within ±0.2 s.
- No wrong subtitle was judged `aligned`. The three errors all say `offset` where `mismatch` was expected.
- Weakest real match: 6.25 (O, −2 s). Strongest wrong match that was not a slow drift: 4.67 (M, other film), 0.23 below the threshold.

### Why the three "fps 24" cells failed

The slow-drift check compares the first half of the film with the second half and calls it a drift when they disagree by more than 1 s (`agreeMS`), and only when each half on its own clears 4.9 (`minHalfPeak`).

- L (an episode of about 24 min) drifts 0.7 s end to end. The two halves differ by well under 1 s, so the whole curve is one blurred peak at the average, +0.7 s.
- M (30 min) drifts about 1.7 s. The halves differ by about 0.9 s, just under the 1 s line.
- N (documentary) drifts about 2.8 s, but its windows are weak (quiet narration), so the halves do not each clear 4.9 and the drift check is skipped.

### Ruling

⚖️ Alexyu, 2026-10-21, chose A: wire it up as it is.

- In `-b2`, `offset` and `mismatch` take the same path: the sidecar is not used and the estimate stays at the transcription price. So these three errors cost money but never translate a drifting subtitle.
- The risk moves to `-b3`. The dialog would say "off by about 1 s" for a subtitle that actually drifts. If `-b3` ever auto-corrects an offset, shifting such a file by the average leaves it up to ±1.4 s off at the ends. That has to be fixed before any auto-correction ships.
- First checks of large files took up to 4 minutes on a busy NAS. The estimate polls for about 3 minutes, then keeps the transcription price. Nothing is overcharged; the user just sees the higher price first.

The exam material is kept on the NAS at `/mnt/user/data/tmp-b2exam/` (extracted subtitles, `exam.json`, `out.log`, the test binary).
