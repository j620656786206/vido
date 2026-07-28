# Story sub-1.4: Extract, SDH-filter, and route embedded subtitles

Status: done

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🔴 HIGH** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.4 · architecture **P0** (the `und` rule) + **P5/P6/P7** + § Data flow
**Depends on:** **sub-1-3 merged** (consumes `ErrSubtitleExtractFailed` from `internal/subtitle/errors.go`; `ErrSubtitleNoTextSource` is NOT referenced here — "no text source" is expressed as the `RouteNoTextSource` verdict, and the sentinel's first consumer is 1.5b's status mapping — CR 2026-07-28 L3). **Zero overlap with sub-1-1 and sub-1-2** — parallel-safe with both.
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

- [x] **Task 1 — Probe index (AC #2)**: `ffprobeStream.Index` + `SubtitleTrack.StreamIndex` + populate + tests; verify the git-diff constraint.
- [x] **Task 2 — Extractor (AC #3)**: `extractor.go` + `buildExtractArgs` + availability/timeout scaffolding + sentinel wrapping + tests.
- [x] **Task 3 — SDH filter (AC #4)**: pure `FilterSDH` + full rule/preservation test table.
- [x] **Task 4 — Router (AC #1, #5)**: `TechProber` iface, candidate filter, single-pass extraction call, selection heuristic, detector integration, the nine-row verdict matrix + `[@contract-v1]` stamp; `router_test.go`.
- [x] **Task 5 — Sync + gates (AC #6, #7)**: architecture delta-tree marker; full `go test ./...` + `pnpm lint:all`.
  - [x] 5.1 **(added in-flight — Rule 24 lane ①)** Same-document truth sweep while applying the AC #7 marker: the delta tree still described the extractor as `ffmpeg -map 0:s -c copy` (:423) and the M1 data flow as `extractor (ffmpeg -c copy)` (:520), both falsified by Finding 2; the requirements-to-structure table still carried `ffprobe_service.go 🔒` (:487); the file tally (:481) still counted it as untouched. All four corrected in `subtitle-pipeline-architecture.md`. Absorbed rather than deferred because AC #7 *is* the architecture micro-sync and leaving a contradiction inside the very document being edited is the "claims an amendment it never made" defect IR-r2 F6 already caught once.

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

`claude-opus-5[1m]` (Amelia — BMM dev-story workflow, 2026-07-28)

### Debug Log References

- **Task 1 RED** — `go test ./internal/services/ -run "SubtitleStreamIndex|StreamIndexJSONRoundTrip"` → build failure naming exactly the 5 new symbol uses (`SubtitleTrack has no field or method StreamIndex`). GREEN after the two struct fields + one assignment.
- **AC #2 git-diff constraint verified** — `git diff apps/api/internal/services/ffprobe_service.go` = **10 insertions / 3 deletions**: `SubtitleTrack.StreamIndex` (+ doc comment), `ffprobeStream.Index`, and the `StreamIndex: stream.Index` assignment. The 3 deletions are gofmt key realignment of the pre-existing composite literal (`Language:`/`Format:`/`External:` values byte-identical), not behaviour.

### Completion Notes List

**🔗 AC Drift: FOUND — `9c-3-ffprobe-integration` AC #2 — `subtitle_tracks` contains `{language, format, external:false}` → `{language, format, external:false, stream_index}`.**
Additive extension: the three original keys keep byte-identical names and values (asserted by `TestSubtitleTrack_StreamIndexJSONRoundTrip`). Every frontend consumer reads the column via `JSON.parse` + named keys (`apps/web/src/utils/libraryStatus.ts:78`, `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:99`) and the TS type is the opaque `subtitleTracks?: string` (`apps/web/src/types/library.ts:47,112`) — an unknown extra key is fail-soft, so no frontend change is owed. Grep run: `grep -rln "SubtitleTrack" _bmad-output/implementation-artifacts/` → 9c-1 / 9c-2 / 9c-3 / this story; 9c-1 AC #1 (the TEXT column) and 9c-2 (NFO reader) are REUSE, not drift.

**📎 Contract Stamps: FOUND (1 stamped AC produced by this story, 0 consumed).**
Produced: AC #1 `[@contract-v1]` — `RouteKind` (5 wire literals) / `RouteDecision` / `ExtractedTrack` / `SelectAndRoute`, stamped in `router.go` and guarded by `TestRouteKindWireValues`. Declared consumers: sub-1-5a, sub-1-5b, sub-1-6; `sub-1-5b-cache-serialize-deliver.md:131` already carries the matching `confirmed against [@contract-v1] sub-1-4 AC #1` ack line.
Consumed: **none owed**. Upstream refs are all unstamped → implicit v0 under the Rule 20 forward-only retrofit — `grep -nE '\[@contract-v[0-9]+\]' sub-1-3-error-codes-sse-stages.md` shows the stamp sits on **AC #1 (SSE stages)** only, while the `errors.go` sentinels this story consumes are **AC #2, unstamped** (the same reading `sub-1-6-wire-triggering-gating.md:113` records: "the 4 `SUBTITLE_` sentinels via sub-1-3 AC #2 (**unstamped** — registry codes only, no ack owed)"); `srt_parser.go`, `detector.go` and `ffprobe_service.go` are all pre-Rule-20.
Bump-side obligation: **not triggered** — v1 is a first stamp, not a bump, so no downstream stale-marking is owed.

**🎭 A11y Pre-Flight: N/A (100% backend — no `apps/web/` files touched).**
Verified by `git diff --name-only` — every changed path is under `apps/api/`, `_bmad-output/`, or the planning artifacts.

**🎨 UX Verification: SKIPPED — no UI changes in this story** (BACKEND-ONLY; frontend tasks = 0 per the story header).

**Pre-existing failure — FIX OR FILE (Epic 9c Retro AI-2): option 2, entry already existed → ENRICHED, not duplicated.**
`TestScannerService_SSEBroadcast_ScanCancelled` (`apps/api/internal/services/scanner_service_test.go:467`) failed intermittently during the Step-7 gates. Tracked entry: **`preexisting-fail-scanner-sse-scan-cancelled-flake`** (filed 2026-05-04 by bugfix-10-1; this story appended the recurrence, the pinpointed root cause, and the decisive proof).
Evidence it is pre-existing and unrelated to sub-1-4:
- With this story's changes **stashed** (clean `main`), `pnpm nx test api --skip-nx-cache` failed run 1 on this exact test and passed run 2 — the repo's own gate target is flaky on `main` today.
- Isolated: `go test -run TestScannerService_SSEBroadcast_ScanCancelled -count=10` → green. Package-scoped: 5× green on clean tree, 3× green with changes.
- Root cause pinpointed (new information this story contributes): `scanner_service_test.go:439-442` races a `time.Sleep(1*time.Millisecond)` goroutine calling `CancelScan()` against a real filesystem walk; the `-v` flag in the `api:test` target widens the loss window.
- Not fixed here because the fix needs a synchronization hook inside `ScannerService` — `CancelScan()` returns `ErrScanNotActive` while `!isScanning`, so pre-cancelling cannot work — which AC #8's scope fence excludes and which the backlog entry already classifies as non-trivial.

**Gate results (Step 7).**
- `cd apps/api && go test ./...` → **exit 0, 34 packages ok** (re-verified after the final change; the only ever-red test across all runs is the pre-existing flake above).
- `pnpm nx run api:lint` (`go vet ./...` + `staticcheck@2026.1 ./...`) → **clean**.
- `pnpm lint:all` → **0 errors** (120 pre-existing warnings, none in files this story touched), `prettier --check .` → all files conform.
- `pnpm nx test web` → **225 files / 2457 tests passed** (0 `apps/web/` files changed, so the Nx cache hit is a legitimate green).
- `pnpm test:cleanup` → **no orphaned test processes**.
- `gofmt -l` on the six **new** Go files (`extractor{,_test}.go`, `sdh_filter{,_test}.go`, `router{,_test}.go`) → clean. `ffprobe_service.go` and `ffprobe_service_test.go` remain gofmt-dirty, but **verifiably so before this story**: with the changes stashed, `gofmt -l` on clean `main` lists both files. The drift is struct-tag/map-literal alignment across blocks this story does not own (`ffprobeStream`, `videoCodecMap`, `audioCodecMap`), and running gofmt on them would reformat unrelated lines — directly violating AC #2's "the git diff shows **only** the two struct fields + one assignment + tests" constraint. Left untouched under the existing `backlog-repo-wide-gofmt-drift` entry, alongside `detector.go` / `manager.go` / `scorer.go` / `providers/*`.

**Deliberate design notes.**
- `RouteKind` is intentionally NOT `models.SubtitleStatus`: routing is a decision, persistence is a side effect. 1.5b maps `RouteSkip → skipped` and `RouteNoTextSource → no_text_source` when it writes.
- The extractor has **no internal semaphore** — the orchestrator's fixed concurrency of 2 (NFR-P3) is the only bound, and a second limiter here would silently halve it.
- `FilterSDH` runs inside the router (P6 — before translation), never in 1.5a, and survivors keep their original `Index` (P7) so 1.5a can hash cue content and assert timestamps.
- `ParseSRT` never returns a non-nil error today, so "candidate failed to parse" is implemented as *read error, parse error, **or** zero cues parsed* — otherwise a garbage track would silently look like a successful empty extraction.

### Discovery Triage

- **Two discoveries, both absorbed (lane ①) at authoring time:**
  - **① Persisted `subtitle_tracks` lacks a stream index** (Finding 1) → absorbed as **AC #2** (additive `StreamIndex` + fresh-probe ruling + 🔒-deviation constraint).
  - **① Epic's literal `-c copy` can't extract ass/mov_text** (Finding 2) → absorbed as **AC #3**'s uniform `-c:s srt` ruling with the FR2 no-media-re-encode reading.
- **One clarification propagated:** sub-1-2's "1.4 writes skipped/no_text_source" shorthand corrected in sub-1-2 (decide vs write split — 1.5b owns the writes).

**Discoveries triaged during implementation (2026-07-28):**

| # | Discovery | Lane | Tracked as |
|---|---|---|---|
| 1 | `subtitle-pipeline-architecture.md` still described the extractor as `-map 0:s -c copy` (:423, :520), still marked `ffprobe_service.go 🔒` in the requirements-to-structure table (:487), and still counted it as untouched in the file tally (:481) — all falsified by this story | ① expand-scope-in-place | **sub-task 5.1** (same document AC #7 already scopes) |
| 2 | Six upstream planning-doc lines still say `ffmpeg -map 0:s -c copy` — `epics-subtitle-pipeline.md:234`, `prd.md:173,207`, `vido-subtitle-pipeline-spec.md:54,82,147`. Non-blocking (sub-1-4 AC #3 is the governing ruling and the code matches it); amending PRD/epic/spec is an SM/PM act | ③ backlog-with-carry-forward-link | **`backlog-planning-docs-c-copy-wording`** |
| 3 | `TestScannerService_SSEBroadcast_ScanCancelled` intermittently red under the full-suite / `nx test api` gates; proven pre-existing on clean `main` | ③ (entry already existed — enriched with recurrence + root cause + proof, NOT duplicated) | **`preexisting-fail-scanner-sse-scan-cancelled-flake`** |
| 4 | Repo-wide `gofmt` drift still present, and it covers `ffprobe_service{,_test}.go` too (verified on clean `main`) — formatting them would reformat unrelated blocks and break AC #2's diff constraint, so they were left as-is; the six new Go files are clean | ③ (entry already existed — no action needed) | **`backlog-repo-wide-gofmt-drift`** (filed by sub-1-3) |

- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
|---|---|
| `apps/api/internal/services/ffprobe_service.go` | AC #2 — `SubtitleTrack.StreamIndex int` (`json:"stream_index"`, no omitempty) + `ffprobeStream.Index int` + one assignment in the subtitle case. Additive only |
| `apps/api/internal/services/ffprobe_service_test.go` | +3 tests — multi-stream index parse (subtitles at absolute 2/5, not slice positions), index-0 survival, and the JSON round-trip proving the three pre-existing keys are byte-identical |
| `apps/api/internal/subtitle/extractor.go` | **new** — text/image codec sets + `SelectCandidates` (embedded ∧ text ∧ eng/en), `Extractor` (LookPath availability, 10-min default per-call timeout, no internal semaphore), single-invocation `Extract` returning `map[streamIndex]path`, `buildExtractArgs` (uniform `-c:s srt`), `trackOutputPath`, `extractFailure`/`stderrTail` |
| `apps/api/internal/subtitle/router.go` | **new** — `RouteKind` 5-value enum + `RouteDecision` + `ExtractedTrack` (all `[@contract-v1]`), the `TechProber` / `TrackExtractor` ports, `Router`/`NewRouter`, `SelectAndRoute` (fresh probe → candidate filter → one-pass extract → parse/filter/select → detect → verdict), `verdictWithoutTrack` (FR9 skip vs FR5 no-text-source), `pickBestCandidate` + `betterCandidate` heuristic, `routeForVariant`, `cueText` |
| `apps/api/internal/subtitle/router_test.go` | **new** — fake `TechProber` + fake extractor (writes real temp files); all nine AC #5 matrix rows incl. the 4-language `und`/`jpn`/`zh`/`chi` skip sub-table, the selection heuristic (post-filter cue count, not raw), the lowest-index tie-break, P7 numbering, fresh-probe + error-propagation, sidecar exclusion, the wire-literal guard for the stamped enum, and compile-time port conformance |
| `apps/api/internal/subtitle/sdh_filter.go` | **new** — pure `FilterSDH(blocks) (kept, removed)` + `filterSDHLine`, `isWholeLineAnnotation`, depth-walking `isWrappedInBrackets`, `isMusicLine`, `stripSpeakerLabel` |
| `apps/api/internal/subtitle/sdh_filter_test.go` | **new** — 25-row rule table (every AC #4 rule + the conservative-preservation cases), the P7 Index/timestamp-preservation assertion (survivor byte-equal to input), input-immutability, and `removed`-counts-cues semantics |
| `apps/api/internal/subtitle/extractor_test.go` | **new** — 10-row `SelectCandidates` table (incl. `und`-never-English, image codecs, sidecars), `buildExtractArgs` table + the never-beside-the-media and uniform-`-c:s srt` guards, construction defaults, sentinel classification, and two ffmpeg-availability-gated integration cases (real mux → extract → `ParseSRT` round-trip; missing-input → `errors.Is` sentinel) |
| `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md` | AC #7 + sub-task 5.1 — delta-tree `ffprobe_service.go` 🔒 → ✏️ `+stream_index (additive, 1.4)`; `extractor.go` line and the M1 data-flow diagram corrected from `-c copy` to `-c:s srt` (Finding 2); requirements-to-structure table FR1 marker synced; file tally re-scored ~20 → ~21 modified / ~14 → ~13 untouched |
| `_bmad-output/implementation-artifacts/9c-3-ffprobe-integration.md` | (AC drift reference — see Completion Notes) `subtitle_tracks` JSON shape additively extended by this story; file itself unmodified |
| `_bmad-output/implementation-artifacts/sub-1-4-extract-filter-route.md` | this file — task checkboxes, Dev Agent Record, Discovery Triage, File List, Change Log, Status |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | `sub-1-4` ready-for-dev → in-progress → review; **new** lane-③ entry `backlog-planning-docs-c-copy-wording`; `preexisting-fail-scanner-sse-scan-cancelled-flake` enriched with the 2026-07-28 recurrence, root cause and clean-tree proof |

### Code Review Record (2026-07-28, adversarial CR — Fable 5)

**Verdict: APPROVED after fixes — 0 High / 2 Medium / 5 Low, all fixed in-review.**
Checks: Git-vs-story discrepancies 0 (the sub-1-2 same-day-correction claim verified as landed at drafting time via #180/#182) · 🔒 Rule 7 PASS (0 new error-code constants; sentinels consumed via `errors.Is` only) · 🔒 Rule 20 N/A (AC #1 is a first v1 stamp, no bump tokens in the diff) · 🔒 Rule 25 N/A (mega-line untouched). Independently re-verified: `ffprobe_service.go` diff = exactly 2 fields + 1 assignment (AC #2 constraint holds), all nine AC #5 matrix rows present in `router_test.go`, subtitle+services suites green, gofmt/prettier clean.

- **M1 (fixed)** `extractor.go` — `extractFailure` chained the cause with `%v`, so `errors.Is(err, context.Canceled)` failed and 1.5b could not tell a shutdown from a real ffmpeg failure (the doc comment even claimed `%w`). Now a second `%w`; lock-in test `TestExtractFailure_CauseStaysInTheChain`.
- **M2 (fixed)** `extractor.go` — parent-ctx cancellation was reported as "ffmpeg timed out after 10m" (`extractCtx.Err() != nil` is true for both). Now `errors.Is(…, DeadlineExceeded)` splits timeout vs "ffmpeg cancelled"; lock-in test `TestExtract_ParentCancellationIsNotReportedAsTimeout`.
- **L1 (fixed)** `sdh_filter.go` — speaker-label regex accepted mismatched brackets (`[JOHN:` / `JOHN]:`), an over-strip against AC #4's under-strip doctrine. Pattern rebuilt as a matched-bracket alternation; 2 new table rows.
- **L2 (fixed)** `extractor.go` — `stderrTail` could cut mid-UTF-8-rune (non-ASCII filenames in ffmpeg stderr). Cut now advances to a rune boundary; test `TestStderrTail_RespectsRuneBoundaries`.
- **L3 (fixed)** story header — claimed consuming `ErrSubtitleNoTextSource`; grep shows 0 references (the verdict enum covers it; 1.5b's status mapping is the sentinel's first consumer). Header corrected.
- **L4 (fixed)** `router.go` — `pickBestCandidate`'s missing-output branch is unreachable with the production Extractor (it errors when any output is missing per AC #3.5); documented as port-contract defense so the fake-only test isn't misread as production behavior.
- **L5 (fixed)** `router.go` — `routeForVariant`'s `default:` sent any FUTURE unknown detector variant to the LLM; now explicit `case LangUndetermined` + fail-closed `default:` → `RouteSkip` (P0 philosophy); test `TestRouteForVariant_UnknownVariantFailsClosed`.

Gates re-run post-fix: `go test ./...` exit 0 / 34 packages (flake did not fire), `nx run api:lint` (vet + staticcheck) clean, gofmt clean on all six touched Go files, prettier clean on this file.

### Change Log

| Date | Change |
|---|---|
| 2026-07-28 | **Adversarial code review — APPROVED after fixes (0H/2M/5L, all fixed).** M1 `%v`→`%w` cause chain in `extractFailure` + M2 cancel-vs-timeout split in `Extract` (both `extractor.go`); L1 matched-bracket speaker-label regex (`sdh_filter.go`); L2 rune-boundary `stderrTail`; L3 story-header sentinel claim corrected; L4 defensive-branch comment; L5 fail-closed unknown-variant default (`router.go`). +5 lock-in tests (extractor 30→…, sdh 25→27 rows, router +1 func). Rule 7 PASS · Rule 20 N/A · Rule 25 N/A. Full gates re-run green. Status review → done. |
| 2026-07-28 | **Task 5 (AC #6, #7)** — architecture micro-sync: delta-tree `ffprobe_service.go` 🔒 → ✏️ `+stream_index (additive, 1.4)`, plus the sub-task 5.1 same-document truth sweep (`-c copy` → `-c:s srt` in the tree line and the M1 data flow; FR1 marker in the requirements-to-structure table; file tally re-scored). Filed lane-③ `backlog-planning-docs-c-copy-wording` for the six remaining PRD/epic/spec lines. Gates: `go test ./...` exit 0 / 34 packages, `api:lint` (vet + staticcheck) clean, `pnpm lint:all` 0 errors + prettier clean, `nx test web` 225 files / 2457 tests, `test:cleanup` no orphans. The single ever-red test is the pre-existing `TestScannerService_SSEBroadcast_ScanCancelled` flake — proven on clean `main` with this story stashed, tracked and enriched under `preexisting-fail-scanner-sse-scan-cancelled-flake`. |
| 2026-07-28 | **Task 4 (AC #1, #5)** — `router.go`: the `[@contract-v1]` front-half contract (`RouteKind` 5 values / `RouteDecision` / `ExtractedTrack` / `SelectAndRoute`) plus the `TechProber` and `TrackExtractor` ports. Verdict matrix implemented exactly: no streams → `no_text_source`; image-only → `no_text_source`; text-but-not-eng/en (incl. `und`) → `skip`; eng+zh-Hant → `deliver_direct`; eng+zh-Hans → `convert_then_deliver`; eng+ambiguous-zh → `convert_then_deliver`; eng+und → `translate`; 0 cues after SDH → `no_text_source`; per-candidate parse failure falls back, all-fail → `ErrSubtitleExtractFailed`. Selection = highest **post-**`FilterSDH` cue count, tie-break lowest `StreamIndex`. Variant comes from `detector.Detect` on cue CONTENT, never the tag (FR6). Writes nothing, broadcasts nothing, calls neither `converter.go` nor `placer.go` (AC #8). |
| 2026-07-28 | **Task 3 (AC #4)** — pure `FilterSDH`: whole-line `[…]`/`(…)` annotations and `♪…♪`/`#…#` music lines dropped, leading ALL-CAPS `NAME:` / `[NAME]:` labels (≤20 chars) stripped, cue dropped when every line went. Conservative guards proven by test: mid-line parentheticals survive, `(quietly) he said (loudly)` is NOT a wrapped annotation (depth walk), `He said: hello` / `John: Hello` / `12:30 tomorrow` are not speaker labels, an unwrapped `♪ dramatic music` keeps its line. Survivors keep the original `Index` + timestamps (P7) and the input slice is never mutated. `removed` counts **cues dropped**, not lines stripped. |
| 2026-07-28 | **Task 2 (AC #3)** — `extractor.go`: candidate filter (embedded ∧ text codec ∧ eng/en, case-insensitive; image codecs enumerated explicitly so the exclusion is auditable), ONE ffmpeg invocation for all candidates (FR3), uniform `-c:s srt` per Finding 2, output confined to the caller's temp dir (D3 — `placer.go` is the sole media-folder writer, and a stray `.eng.srt` would be auto-detected by Plex/Jellyfin), `NewExtractor` availability + 10-min default timeout with **no** internal semaphore (the orchestrator's concurrency 2 is the only bound), and `ErrSubtitleExtractFailed` wrapping with an ffmpeg stderr tail. Real-ffmpeg integration cases pass locally and skip cleanly where ffmpeg is absent. |
| 2026-07-28 | **Task 1 (AC #2)** — fresh-probe stream indexes: `ffprobeStream` now parses ffprobe's `index`, `services.SubtitleTrack` carries `StreamIndex` (`json:"stream_index"`, deliberately no `omitempty` — index 0 is a legal embedded index), populated for subtitle streams only. Additive extension of the 9c-3 `subtitle_tracks` JSON shape; FE reads the column via `JSON.parse` + named keys (`libraryStatus.ts:78`, `ManageSubtitleDialogV2.tsx:99`) and its type is an opaque `subtitleTracks?: string`, so the new key is fail-soft. RED observed first. |
