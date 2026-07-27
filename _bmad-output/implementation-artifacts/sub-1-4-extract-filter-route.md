# Story sub-1.4: Extract, SDH-filter, and route embedded subtitles

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🔴 HIGH** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.4 · architecture **P0** (the `und` rule) + **P5/P6/P7** + § Data flow
**Depends on:** **sub-1-3 merged** (consumes `ErrSubtitleExtractFailed` / `ErrSubtitleNoTextSource` from `internal/subtitle/errors.go`). **Zero overlap with sub-1-1 and sub-1-2** — parallel-safe with both.
**Blocks:** sub-1-5a (translates the routed English cues), sub-1-5b (delivers direct/converted routes via placer), sub-1-6 (calls the front half via the orchestrator).
**Cross-stack split check:** backend tasks = 5, frontend tasks = **0** → single story.

---

## Story

As a NAS owner,
I want embedded text subtitles detected, extracted, SDH-filtered, and routed by language,
so that Traditional/Simplified embedded subs are delivered directly and English is queued for translation.

---

## 🔎 Codebase findings that reshape the epic AC (verified 2026-07-27)

### Finding 1 — the persisted `subtitle_tracks` JSON **cannot drive `ffmpeg -map`**

`services.SubtitleTrack` is `{Language, Format, External}` (`ffprobe_service.go:30-34`) — **no stream index**, and `ffprobeStream` (`:142-151`) doesn't even parse ffprobe's `index` field. The architecture's *"subtitle_tracks already persisted"* reuse assumption holds for **eligibility/enumeration** (1.6's batch scope), not for **extraction targeting**. Ruling (Rule 24 lane ①, absorbed as AC #2):

- `ffprobeStream` gains `Index int` + `SubtitleTrack` gains `StreamIndex int` (**additive**; meaningful only when `External == false`; FE types are loose strings/objects → fail-soft).
- The pipeline **re-probes fresh at run time** via the existing `FFprobeService.Probe(ctx, path)` — files move/change after scan, and pre-existing rows lack the index. Persisted JSON stays the *scan-time* signal; the fresh probe is the *run-time* truth.
- `ffprobe_service.go` is 🔒 in the delta tree — same deviation shape as sub-1-3's `engine.go` ruling: **additive only**, `git diff` must show exactly the two struct fields + one assignment (+ tests). AC #7 syncs the tree marker.

### Finding 2 — the epic's literal `-c copy` cannot extract `ass`/`mov_text` to `.srt`

`-c:s copy` into an `.srt` muxer works **only** when the source codec is already `subrip`. ASS/SSA and MP4 `mov_text` tracks require the subtitle-codec transform **`-c:s srt`**. Ruling: the extractor uses **uniform `-c:s srt`** for every candidate track. This does **not** violate FR2's "no re-encode" — that clause protects the **media** (video/audio are never touched; `-map 0:{n}` selects subtitle streams only); a text-format transform is not a re-encode. The epic AC's `-c copy` reads as "the media is not re-encoded".

### Finding 3 — M1 routing is **eng/en-gated first, content-routed second** (P0, and why FR7/FR8 still fire)

P0's ruling is literal: *"M1 accepts only tracks tagged `eng`/`en`. Everything else routes to skip. `und` is NEVER treated as English."* So a zh-tagged embedded track **skips in M1**. FR7/FR8 (Traditional pass-through / Simplified → OpenCC) fire on the **mislabeled-track path**: an `eng`-tagged track whose *content* turns out Traditional/Simplified — exactly the Bazarr mislabel bug FR6 exists to fix. That is why `detector.Detect` runs **after** extraction in the data flow. Do not "fix" the order.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` The front-half contract: `Router.SelectAndRoute`

**Given** the orchestrator (1.5a/1.5b) needs one clean entry point for everything before translation, **then** `internal/subtitle/router.go` exposes:

```go
// RouteKind classifies what the pipeline should do with a media item.
// [@contract-v1] — consumed by sub-1-5a (RouteTranslate), sub-1-5b (delivery
// of RouteDeliverDirect/RouteConvertThenDeliver + terminal status writes),
// sub-1-6 (orchestration). Changing kinds/fields = Rule 20 bump + stale-mark.
type RouteKind string

const (
    RouteDeliverDirect      RouteKind = "deliver_direct"       // FR7 — content is already zh-Hant
    RouteConvertThenDeliver RouteKind = "convert_then_deliver" // FR8 — content is zh-Hans → OpenCC (caller runs it)
    RouteTranslate          RouteKind = "translate"            // FR10 — English → LLM
    RouteSkip               RouteKind = "skip"                 // FR9/P0 — no eng/en-tagged text track
    RouteNoTextSource       RouteKind = "no_text_source"       // FR5 — no usable text source at all
)

type RouteDecision struct {
    Kind            RouteKind
    Track           *ExtractedTrack // nil for RouteSkip / RouteNoTextSource
    DetectedVariant string          // detector result for the chosen track ("zh-Hant"/"zh-Hans"/"zh"/"und")
    Reason          string          // human-readable, for logs + the orchestrator's status message
}

type ExtractedTrack struct {
    StreamIndex int    // absolute ffmpeg stream index
    Language    string // the ffprobe tag that admitted it ("eng"/"en")
    Codec       string // source codec (subrip/ass/mov_text/…)
    Path        string // extracted .srt in the caller-owned temp dir
    Blocks      []SubtitleBlock // parsed + SDH-filtered cues (original numbering — P7)
}

// SelectAndRoute probes fresh, extracts every candidate track in ONE ffmpeg
// pass, SDH-filters, detects the CJK variant, and returns the routing verdict.
func (r *Router) SelectAndRoute(ctx context.Context, mediaPath, tmpDir string) (RouteDecision, error)
```

**This story writes NOTHING to the DB and broadcasts NOTHING.** Components are pure; `subtitle_status` writes and SSE stages are the orchestrator's (1.5b/1.6). It also never calls `converter.go` or `placer.go` — the verdict tells the caller to.

### AC #2 — Fresh probe with stream indexes (Finding 1)

**Given** `FFprobeService.Probe`, **then** `ffprobeStream` parses `index` and `SubtitleTrack` carries `StreamIndex int` (`json:"stream_index"` — no `omitempty`; index 0 is legal; doc-comment: meaningful only when `External == false`). `Probe` populates it for subtitle streams. The router consumes the service through a narrow local interface (`TechProber`) so unit tests inject a fake:

```go
type TechProber interface {
    Probe(ctx context.Context, filePath string) (*services.MediaTechInfo, error)
}
```

(`subtitle → services` is a legal Rule 19 direction — the `engine.go` precedent.) `git diff apps/api/internal/services/ffprobe_service.go` shows **only**: the two struct fields, the one assignment in the subtitle case, and test additions.

### AC #3 — Extraction: one pass, temp-dir only, text codecs only (FR2/FR3)

**Given** a media file and its probed subtitle tracks, **then** `internal/subtitle/extractor.go`:

1. **Candidate filter:** `External == false` **and** codec ∈ text set `{subrip, srt, ass, ssa, mov_text, webvtt}` **and** language tag ∈ `{eng, en}` (case-insensitive). Image codecs `{hdmv_pgs_subtitle, dvd_subtitle, dvb_subtitle, xsub}` are never candidates.
2. **Single invocation, multiple outputs** (FR3 — one read pass): `ffmpeg -nostdin -y -i {media} -map 0:{idx} -c:s srt {tmp}/track_{idx}.srt` repeated per candidate **within one command**. Uniform `-c:s srt` per Finding 2.
3. **Output goes to the caller-supplied temp dir, NEVER beside the media.** Two reasons, both load-bearing: `placer.go` is the **sole writer** into the media folder (D3), and a stray `.eng.srt` sidecar would be auto-detected by Plex/Jellyfin/Video Station (the same hazard the architecture's pilot-variant isolation rule guards).
4. Construction mirrors `FFprobeService`: `exec.LookPath("ffmpeg")` availability at `NewExtractor`, `IsAvailable()`, per-call `context.WithTimeout` (default **10 min** — a 20 GB remux demux on a DS920+ is minutes of pure I/O; configurable). No internal semaphore — the orchestrator's fixed concurrency 2 (NFR-P3) is the bound.
5. ffmpeg exit ≠ 0 or a missing output file → wrap **`ErrSubtitleExtractFailed`** (sub-1-3 sentinel) with stderr tail in the message (Rule 13).
6. Args assembly is a pure, tested function (`buildExtractArgs`) — the real-ffmpeg path is integration-tested behind an availability skip (the `ffprobe_service_test.go` precedent).

### AC #4 — SDH filter: strip annotations, keep numbering (FR4, P6/P7)

**Given** parsed `[]SubtitleBlock`, **then** `internal/subtitle/sdh_filter.go` exposes a pure `FilterSDH(blocks []SubtitleBlock) (kept []SubtitleBlock, removed int)`:

| Rule | Example removed |
|---|---|
| Line is entirely `[...]` or `(...)` | `[door slams]` · `(sighs)` |
| Line is wrapped in music marks `♪…♪` / `#…#` | `♪ dramatic music ♪` |
| Leading speaker label `NAME:` (ALL-CAPS ≤ 20 chars, optional `[]`) is stripped from the line | `JOHN: Hello` → `Hello` |
| Cue whose every line was removed → **cue dropped** | — |

- **Conservative by design:** a parenthetical *inside* a dialogue line stays (`He said (quietly) yes` is untouched) — only whole-line annotations and leading labels are handled. Over-stripping loses dialogue; under-stripping costs tokens. M1 errs to under-strip.
- **P7:** survivors keep their original `Index` and timestamps untouched (P2) — no renumbering, gaps are correct. FR17's later equality assertion depends on this.
- Filtering runs here — **before** translation (P6) — never in 1.5a.

### AC #5 — Routing verdict matrix (FR5/6/7/8/9 · P0)

**Given** `SelectAndRoute`, **then** the verdict follows this matrix **exactly** (each row is a required test case):

| Probed state | Verdict |
|---|---|
| No subtitle streams at all | `RouteNoTextSource` |
| Only image-codec tracks (PGS/VobSub…) | `RouteNoTextSource` (FR5 — "image-only or none") |
| Text tracks exist, **none tagged eng/en** (incl. `und`, `jpn`, `zh`, `chi`) | `RouteSkip` — **`und` is NEVER English (P0)**; zh-tagged tracks skip in M1 (Finding 3) |
| eng-tagged text track, content detects **zh-Hant** | `RouteDeliverDirect` (FR7 — mislabel case) |
| eng-tagged, content detects **zh-Hans** | `RouteConvertThenDeliver` (FR8) |
| eng-tagged, content detects **zh** (ambiguous 30–70%) | `RouteConvertThenDeliver` — s2twp is idempotent on already-Traditional text, so converting is the safe branch (§9b co-production precedent: default to convert) |
| eng-tagged, content detects **und** (no CJK = real English) | `RouteTranslate` |
| Chosen track parses but **0 cues survive `FilterSDH`** | `RouteNoTextSource` — FR5's word is *usable*; a pure-SDH track is not a usable text source |
| First candidate fails `ParseSRT` | fall back to the next candidate; **all** candidates fail → `ErrSubtitleExtractFailed` |

**Multi-candidate selection (two-devs-diverge point, ruled):** when several eng/en text tracks survive extraction, choose the one with the **highest post-`FilterSDH` cue count** (forced-narrative tracks have few cues; SDH variants converge with full tracks after filtering). Deterministic tie-break: lowest `StreamIndex`.

- CJK-variant decision comes from **`detector.Detect` on content** — never the tag (FR6). `detector.go` stays 🔒 (its scope-narrowing per P0 is a *usage* convention: this story simply never asks it for source language).

### AC #6 — Tests (Rule 9 co-located, Rule 16 exact)

1. `extractor_test.go` — `buildExtractArgs` table (multi-track single command, `-c:s srt`, `-map 0:{idx}`, temp-dir paths); availability-gated integration case (skip pattern from `ffprobe_service_test.go`); `ErrSubtitleExtractFailed` classification via `errors.Is`.
2. `sdh_filter_test.go` — table over every AC #4 rule + the preservation cases (mid-line parenthetical kept; `Index`/timestamps of survivors byte-equal originals — the P7 assertion).
3. `router_test.go` — **all nine matrix rows** of AC #5 via fake `TechProber` + fake extractor, plus the selection heuristic + tie-break. The `und`-never-English and zh-tagged-skip rows are non-negotiable.
4. `ffprobe_service_test.go` — `stream_index` parse + populate.
5. `go test ./...` green; `pnpm lint:all` green.

### AC #7 — Architecture micro-sync

**Given** the delta tree, **then** `ffprobe_service.go` marker `🔒` → `✏️ +stream_index (additive, 1.4)`; the story's temp-dir + no-media-folder-write rule needs no tree change (extractor was always 🆕).

### AC #8 — Scope fence

- ❌ No orchestrator / `pipeline.go`, no DB writes, no `subtitle_status`, no SSE, no stage broadcasts — 1.5a/1.5b/1.6.
- ❌ No OpenCC invocation, no `placer.go` call — the verdict instructs the caller (D3: placer stays sole writer, orchestrator sole automatic caller).
- ❌ No Claude/LLM anything — 1.5a.
- ❌ No external-sidecar handling (`DetectExternalSubtitles` untouched — M1 pipeline is embedded-only; sidecars remain the search engine's domain).
- ❌ No forced/disposition flag parsing (the cue-count heuristic covers it for M1; revisit if the pilot shows mis-selection).
- ❌ No `batch.go`, no feature flag, no frontend, no `.pen`.

---

## Tasks / Subtasks

- [ ] **Task 1 — Probe index (AC #2)**: `ffprobeStream.Index` + `SubtitleTrack.StreamIndex` + populate + tests; verify the git-diff constraint.
- [ ] **Task 2 — Extractor (AC #3)**: `extractor.go` + `buildExtractArgs` + availability/timeout scaffolding + sentinel wrapping + tests.
- [ ] **Task 3 — SDH filter (AC #4)**: pure `FilterSDH` + full rule/preservation test table.
- [ ] **Task 4 — Router (AC #1, #5)**: `TechProber` iface, candidate filter, single-pass extraction call, selection heuristic, detector integration, the nine-row verdict matrix + `[@contract-v1]` stamp; `router_test.go`.
- [ ] **Task 5 — Sync + gates (AC #6, #7)**: architecture delta-tree marker; full `go test ./...` + `pnpm lint:all`.

---

## Dev Notes

- **Reuse, don't rebuild:** `ParseSRT`/`SerializeSRT` + `SubtitleBlock` (`srt_parser.go:11-23` — the cue contract), `detector.Detect` (`detector.go:61` — returns `DetectionResult.Language` ∈ zh-Hant/zh-Hans/zh/und), `FFprobeService.Probe` (`ffprobe_service.go:82`). The text/image codec sets are the only new vocabulary.
- **P1 heads-up for your successors (not this story):** cue identity for caching/retry is `hash(content)`, never index — SDH filtering shifts positions. Your `ExtractedTrack.Blocks` keeps original `Index` per P7 precisely so 1.5a can hash content and assert timestamps.
- **Why `RouteKind` is not `SubtitleStatus`:** routing is a decision, persistence is a side effect. The orchestrator maps `RouteSkip → SubtitleStatusSkipped`, `RouteNoTextSource → SubtitleStatusNoTextSource` when it writes (1.5b). Keeping the enums separate prevents the DB vocabulary leaking into pure functions — and note sub-1-2's blocks-line shorthand ("1.4 writes skipped/…") is corrected in that file today: 1.4 *decides*, 1.5b *writes*.
- **Rule 13:** every ffmpeg/probe error wrapped with context (`%w`), never swallowed; the router logs the per-candidate fallback at `slog.Warn` with track index + reason.
- **Rule 14:** one `Extractor` constructed at startup wiring (by 1.6), reused; never per item.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. Rule 23 does not apply.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.4] · [Source: architecture #P0 (:319-341) — the ruling text + "fails closed rather than mistranslating"] · [#P5/P6/P7 (:368-378)] · [#Data flow (:504-520)] · [#Pilot variant isolation (:123) — why extraction output never touches the media folder]
- [Source: `apps/api/internal/services/ffprobe_service.go`:30-34,82,142-151,188-198] — the missing-index evidence.
- [Source: `apps/api/internal/subtitle/detector.go`:12-61 · `srt_parser.go`:11-23 · `errors.go` (sub-1-3)]
- [Source: `project-context.md`#Rule 13, #Rule 14, #Rule 19, #Rule 20, #Rule 24]

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Two discoveries, both absorbed (lane ①) at authoring time:**
  - **① Persisted `subtitle_tracks` lacks a stream index** (Finding 1) → absorbed as **AC #2** (additive `StreamIndex` + fresh-probe ruling + 🔒-deviation constraint).
  - **① Epic's literal `-c copy` can't extract ass/mov_text** (Finding 2) → absorbed as **AC #3**'s uniform `-c:s srt` ruling with the FR2 no-media-re-encode reading.
- **One clarification propagated:** sub-1-2's "1.4 writes skipped/no_text_source" shorthand corrected in sub-1-2 (decide vs write split — 1.5b owns the writes).
- Reference: `project-context.md` Rule 24.

### File List
