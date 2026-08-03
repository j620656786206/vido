# Story sub-1.6: Wire triggering, gating, and progress

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟡 MEDIUM (wiring + one HTTP surface)** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.6 · architecture **D5** + **V2/V3/V6** + **P8** · IR rulings **F4** (G2/G4 bars live here) + **F9** (deployment-doc half)
**Depends on (merged):** **sub-1-5b** (`ProcessItem` — acked below) · sub-1-3 (stages) transitively. This is the story where the pipeline becomes **user-reachable**.
**Blocks:** nothing in M1 — this closes the backend chain. (sub-1-7b and Epic 2 are parallel tracks.)
**Cross-stack split check:** backend tasks = 6, frontend tasks = **0** → single story (F2/F3/F5 surfaces already render `message`/`stage` fail-soft — verified in 1-3).

---

## Story

As a NAS owner,
I want the pipeline triggered automatically on media-add and on demand, gated when unconfigured, with live progress,
so that subtitles appear without manual steps and failures are always visible.

---

## 🔎 Codebase findings (verified 2026-07-27)

1. **The scan-complete hook already exists — and is already occupied.** `ScannerService.SetOnScanComplete(fn func())` (`scanner_service.go:82`, fired at `:314` when files were created/updated) is wired at **`main.go:422`**. FR13 therefore needs **zero `scanner_service.go` edits** — main.go composes the existing callback (`old(); pipeline.EnqueueMissing(ctx)`). The architecture's V3 "adds `scanner_service.go ✏️`" is unnecessary; AC #7 corrects the delta tree (the third such correction this epic).
2. **The flag seam is exactly one call:** `bp.engine.Process(...)` at `batch.go:~244`. D5's `legacy | pipeline` conditional wraps that single call site — nowhere else.
3. **Movie enumeration exists:** `FindMissingZhHantSubtitle` (`movie_repository.go:892`, Route C heritage — "movies with a media file but no zh-Hant subtitle"). Episode-grain: locate the 9R-16 generation-batch enumeration at implementation (Task 2 decision tree).
4. **`docs/deployment.zh-TW.md` does not exist** — every other user-facing doc has a zh-TW twin (sse-event-types, testsprite-local-dev, unraid-installation-guide); deployment.md never got one. Pre-existing Rule 17 debt, discovered now → **lane ③** entry `backlog-deployment-doc-zh-tw-twin` (filed with this story); F9's AC here edits EN and the twin inherits the section when it materializes. *(Alexyu may overrule: translate the full twin inside this story — see Open Questions.)*

---

## Acceptance Criteria

### AC #1 — D5: the feature flag gates exactly one seam

**Given** `VIDO_SUBTITLE_PIPELINE_MODE` = `legacy` (default) | `pipeline` (config.go `loadString` pattern + validation; env-var, not a settings row — the migration-029 cleanup of `new_shell_enabled` is the cautionary precedent for table-backed flags), **when** set to `pipeline`, **then** the `bp.engine.Process(...)` call at `batch.go:~244` becomes `pipeline.ProcessItem(ctx, MediaRef{...}, ProcessOptions{})`; `legacy` behaves byte-identically to today. **One conditional, one call site** — the flag appears nowhere inside pipeline stages, handlers, or the scanner path (D5's ban). Flag read once at startup into the wiring, not per item.

### AC #2 — FR13: auto-enqueue on scan complete (zero scanner edits)

**Given** a completed scan with created/updated files, **then** main.go's composed `onScanComplete` callback calls `pipeline.EnqueueMissing(ctx)`, which enumerates eligible items (movies via `FindMissingZhHantSubtitle`; episodes via the located 9R-16 path) and enqueues them. Eligibility re-checks are cheap here — `ProcessItem`'s P5 pre-flight (1.5b) is the authoritative gate, so over-enumeration is safe.

- Enqueue is honoured **only in `pipeline` mode** — in `legacy` mode the callback composition still runs the pre-existing `:422` behaviour untouched.
- The existing `:422` callback body is preserved exactly (wrap, don't replace).

### AC #3 — Worker pool: fixed concurrency 2, graceful lifecycle

**Given** AD #5 + NFR-P3 (M1), **then** the pipeline owns a pool: **2 workers** (const `PipelineConcurrencyM1 = 2`), buffered channel (cap 1024), **non-blocking enqueue with drop-and-`slog.Warn`** on overflow (fail-soft — the next scan re-enqueues; mirrors the SSE hub's drop discipline), **in-flight dedup** by `MediaRef` key (bounded map, Rule 14 — an item already queued/running is not double-enqueued), `Start(ctx)`/`Stop()` lifecycle in main.go's goroutine zone + graceful-shutdown block (the `retry/scheduler.go` pattern, per the 13-4a precedent).

### AC #4 — FR12: `POST /api/v1/subtitles/pipeline/run`

**Given** F2-D-v2's 生成字幕 button (V2), **then** a **new** `handlers/subtitle_pipeline_handler.go` (the existing `subtitle_handler.go` stays 🔒 — manual search path unchanged, D3):

- Request: `{media_id, media_type, force?}` (`media_type` ∈ movie|series|episode — sub-1-2 AC #1 vocabulary).
- Behaviour: capability gate (AC #5) → dedup-aware enqueue → **`202 Accepted`** with `{success, data: {status: "queued"|"already_queued", media_id}}`. Never synchronous — a translate run is minutes; progress flows over SSE (AC #6).
- Rule 3 envelope · Rule 10 versioning · **Swagger annotations + `swag init`** (Rule 15 — this story adds the epic's only new HTTP surface) · `RegisterRoutes` called from main.go and **verified** (Rule 15's route↔client check; the 10-2 precedent).
- 400 on unknown `media_type`; 404 when the item doesn't exist (repo lookup); gate failure per AC #5.

### AC #5 — FR23: one capability gate, three entry points (V6)

**Given** no configured translation key, **then** a single check owned by the `Pipeline` (`configured func() bool` field, wired from `cfg.HasClaudeKey` — V6's "top of pipeline.go") governs **all three** entry paths (endpoint, batch seam, scanner enqueue):

- Endpoint → `409` with **`AI_NOT_CONFIGURED`** (reuse — `ErrAINotConfigured` exists; **zero new Rule 7 codes**, zero registry edits) + zh-TW message: `尚未設定翻譯服務金鑰` + suggestion `請設定 CLAUDE_API_KEY 環境變數後重啟（設定頁面將於 M1.5 提供）` — matching F5-D-v2's 尚未設定 framing and J3's env-var reality.
- Scanner/batch paths → gate short-circuits **before enqueue** with one `slog.Info` (not per-item spam) — no silent failure, no queued work that can only fail.
- **Ruling (scope-honest):** the gate closes the **whole** pipeline entry in M1, including keyless extract-only routes — matches F5's wholesale-gated UX and V6's single-gate design. Keyless zh-extraction is a noted P2 candidate, not built.

### AC #6 — FR33/P8: SSE progress wiring

**Given** 1.5b's nil-safe progress hook, **then** main.go connects it to the existing SSE hub: `subtitle_progress` events (**event type unchanged** — `sse/hub.go` stays 🔒) carrying `{media_id, media_type, stage, message}` with 1-3's stage values (`probing`/`extracting`/`translating`/`skipped` now reach the wire for the first time). Cadence is already P8-correct (once per chunk + stage transitions — the hook's call sites, 1.5b). Messages are zh-TW user-facing strings composed at the wiring layer (e.g. `抽取內嵌字幕中…`, `翻譯中（第 N/M 段）`). Frontend consumes fail-soft today (`useSubtitleSearch.ts:21` — verified in 1-3); richer stage UI is F3-D-v2's existing surface reading `message`.

### AC #7 — [F4 ruling] The G2/G4 measurable bars — **✅ ALL THREE CONFIRMED (Alexyu, 2026-07-27)**

**Given** M1's purpose is validating trust on real hardware, **then** these bars are citable ACs of M1 (measured at the pilot on the DS920+; per-item timing comes free from `subtitle_runs.started_at/completed_at`):

| # | Bar | Confirmed threshold |
|---|---|---|
| (a) | **NFR-P1 resource bound** during one item | pipeline-attributable sustained CPU ≤ **1 core** (≈25% of J4125) and incremental RSS ≤ **256 MB**; concurrent playback (Video Station/Plex) stays functional — verified via `docker stats` per the pilot procedure noted in the run log |
| (b) | **time-to-`.zh-Hant.srt`** | translate path: ~600-cue episode ≤ **3 min** · ~1,200-cue movie ≤ **6 min** (2× scale, same per-cue rate) · direct/convert path (no LLM) ≤ **60 s** |
| (c) | **trust bar** | ≥ **90%** of a **20-item** pilot sample accepted **without hand-editing** (protocol: skim + spot-play 3 random cues per item; recorded in pilot notes) |

**✅ Confirmed (Alexyu, 2026-07-27):** (b) episode 5 → **3 min** (「太慢，我覺得3分鐘可以接受」), movie scaled to **6 min**; **(a) and (c) confirmed as proposed** (「a跟c可以」). These are now citable M1 acceptance bars, not proposals.

**Engineering implication of the 3-min bar (for the pilot's eyes):** ~600 cues ÷ 10-cue chunks ≈ 60 sequential requests ⇒ **average ≤ 3 s per chunk including quality-gate retries**. Feasible on `claude-haiku-4-5` but tight. If the pilot misses it, the two levers — in order — are ① a larger chunk size (transport unit only; the retry unit stays the cue per P3, and `TranslationMaxTokens=4096` has headroom) and ② post-D10-warm chunk parallelism. Both are pilot-informed follow-ups, **not** current scope — do not pre-optimize.

**⚠️ Discovery under (b):** the PRD's *"tens of seconds … one translation call"* assumed a single call; 1.5a ships sequential chunk=10 → a 600-cue episode is ~60 chunks = **minutes, not tens of seconds**. The bar above is the honest number; the PRD's prose estimate is flagged (F2-class, optional edit — Open Questions).

### AC #8 — [F9 ruling] Deployment docs: EN section now, zh-TW twin as filed debt

**Given** `Dockerfile:47` (ffmpeg bundled) and `docker.yml:80` (amd64+arm64) are shipped infrastructure, **then** `docs/deployment.md` gains a short section (under § Prerequisites): the image **bundles ffmpeg/ffprobe** (no host install; required by the subtitle pipeline — absence degrades silently, the 2026-06 audit), the image is **multi-arch**, and the `VIDO_SUBTITLE_PIPELINE_MODE` + `CLAUDE_API_KEY` env vars are documented under § Configuration. **`docs/deployment.zh-TW.md` does not exist** (Finding 4) → the twin is `backlog-deployment-doc-zh-tw-twin` (lane ③, filed with this story); it inherits this section. NFR-S3's HTTPS half stays with Story 2.1.

### AC #9 — Tests + scope fence

**Tests (Rule 9/16):** flag seam (both modes, byte-identical legacy path via a spy engine); enqueue dedup + overflow-drop + mode-gating; pool start/stop with ctx cancel (no goroutine leak — `goleak`-style or WaitGroup assertion); handler table (202 queued / already_queued / 409 gate / 400 / 404) + envelope shape; gate short-circuit before enqueue (spy: zero enqueues when unconfigured); SSE wiring emits stage+message per hook call (fake hub). `go test ./...` + `pnpm lint:all` green.

**Fence:** ❌ no frontend (F2's button already POSTs? — no: the FE call-site wiring to the new endpoint is **sub-1-7b-adjacent but NOT here**; M1's button wiring rides the existing generation dialog surface — if a FE edit turns out to be required to point 生成字幕 at the new endpoint, that is a **lane ② discovery**, stop and file it) · ❌ no batch-scope UI (FR34 P2) · ❌ no cost estimate (FR14 P2) · ❌ no key-config UI (2-1) · ❌ no new Rule 7 codes · ❌ no `scanner_service.go` / `sse/hub.go` / `subtitle_handler.go` edits.

### AC #10 — Architecture micro-sync

Delta tree: `scanner_service.go ✏️` → `🔒 (hook existed; composed in main.go — corrected at 1.6 drafting)` · add `handlers/subtitle_pipeline_handler.go 🆕` · main.go already ✏️.

---

### AC #11 — [Rule 24 lane ①, added at implementation 2026-08-03] Episode-grain enumeration

Task 2's decision tree fired its **absent** branch: the 9R-16 episode enumeration **does not exist**. 9R-16 shipped movies-only — its `generationCandidateFinder` (`services/generation_batch.go:81`) declares only movie methods, and `EpisodeRepository` had no missing-subtitle query at all. Enumerating episodes via any existing call was therefore impossible, and silently inventing a query is what the decision tree forbids.

**Therefore:** `EpisodeRepository.FindMissingZhHantSubtitle(ctx)` is added, MIRRORING the movie predicate (`missingZhHantSubtitleWhere`, `movie_repository.go:883`):

1. Predicate = "no zh-Hant on record **AND** a media file present". Deliberately broader than a `subtitle_status` filter, for the same reason as movies: an episode with a found **English** subtitle still lacks zh-Hant and is in scope. Completed items self-exclude once generation writes `subtitle_language='zh-Hant'` — which is what makes a re-scan free.
2. **No `is_removed` clause** — unlike `movies`, the `episodes` table has no such column (migration 006 models removal by deleting the row). Mirroring it blindly would have been a query against a column that does not exist.
3. Ordered `series_id, season_number, episode_number` — load-bearing, not cosmetic: grouping a show's episodes is what lets sub-1-5b's D10 latch warm ONE prompt prefix per show instead of re-warming it every time the queue interleaves two series.
4. Registered on `EpisodeRepositoryInterface` (Rule 11) and covered by real-`:memory:`-SQLite tests over the migrated schema (Rule 15).

---

## Tasks / Subtasks

- [x] **Task 1 — Flag + config (AC #1):** `VIDO_SUBTITLE_PIPELINE_MODE` in config.go (+validation+test); the one-conditional seam at `batch.go:~244` with a spy-engine byte-identity test for legacy.
- [~] **Task 2 — Pool + enqueue (AC #2, #3, #11):** _(in progress — pool, enumeration, MediaStore adapter and the search-column fix are done + tested; main.go callback composition remains)_  pool with lifecycle/dedup/overflow; `EnqueueMissing` (movies via `FindMissingZhHantSubtitle`; **decision tree:** locate the 9R-16 episode enumeration — found ⇒ reuse; absent ⇒ expand scope with a new AC per Rule 24 lane ①, do not silently build a new query); main.go callback composition preserving `:422`.
- [ ] **Task 3 — Endpoint (AC #4):** new handler + routes + Swagger + `swag init` + main.go wiring + Rule 15 route verification.
- [ ] **Task 4 — Gate (AC #5):** `configured` wiring, three-entry-point coverage, zh-TW messages, tests.
- [ ] **Task 5 — SSE + docs (AC #6, #8):** progress-hook→hub adapter + zh-TW stage messages; deployment.md EN section; file `backlog-deployment-doc-zh-tw-twin` in sprint-status (verify not already filed).
- [ ] **Task 6 — Sync + gates (AC #7, #9, #10):** record the pilot-bar numbers as confirmed (or adjusted) by Alexyu; delta-tree corrections; full test + lint gates; Rule 20 ack recorded.

---

## Dev Notes

- **Rule 20 acks (record verbatim):** `confirmed against [@contract-v1] sub-1-5b AC #1` (`ProcessItem`/`MediaRef`/`ProcessItemOptions`) · `confirmed against [@contract-v1] (Story sub-1-3 AC #1)` — the 12-value `PipelineStage` set this story broadcasts · the 4 `SUBTITLE_` sentinels via sub-1-3 AC #2 (**unstamped** — registry codes only, no ack owed) · `MediaRef.MediaType` vocabulary per sub-1-2 AC #1.
  - ⚠️ **Corrected 2026-07-28 by sub-1-5b at implementation time** (Rule 24 lane ①). Three shipped-surface facts differ from what sub-1-5b's AC text predicted. **No re-architecture is implied and no Rule 20 bump is owed** (sub-1-5b AC #1 is a NEW `[@contract-v1]`, not a bump) — but this story's wiring must be written against the shipped names:
    1. **`ProcessOptions` → `ProcessItemOptions`.** `subtitle.ProcessOptions` was already taken by the SEARCH engine (`engine.go:125`, constructed at `batch.go:238` — the very call site AC #1's flag seam wraps). Shape is unchanged: one `Force bool`.
    2. **The progress hook carries the `MediaRef`:** `func(ref MediaRef, stage PipelineStage, message string)`, installed with `subtitle.WithProgress(...)`. AC #6 needs `{media_id, media_type}` in every `subtitle_progress` payload, and ONE Pipeline serves both pool workers — the two-argument form in sub-1-5b's AC text could not have attributed an event to an item. Cadence is already P8-correct: stage transitions plus once per chunk.
    3. **`MediaStore` (AC #1's status-writer port) must NOT stamp the search columns.** `MovieRepository.UpdateSubtitleStatus:817` and `SeriesRepository.UpdateSubtitleStatus:800` unconditionally write `subtitle_last_searched = now`, which violates sub-1-5b AC #6.3 (`subtitle_search_score` / `subtitle_last_searched` stay unset — extract/translate is not a search). `EpisodeRepository.UpdateEpisodeSubtitleStatus:273` is already correct. Tracked as `backlog-subtitle-status-writer-search-columns`; the cheap fix is a narrow repo method mirroring the episode one. **This story owns the adapter, so it owns the fix.**
    4. **Wiring surface:** `subtitle.NewPipeline(translator, converter, logger, WithRouter, WithPlacer, WithMediaStore, WithRunStore, WithSegmentCache, WithModelID, WithProgress)`. `ProcessItem` returns a named error listing any port left nil, so an incomplete wiring fails loudly at the first item rather than nil-panicking a worker. `MediaItem.ShowKey` (series id for series/episode rows, empty for movies) is what drives the D10 gate — an episode's `MediaRef.ID` is the EPISODE id and would gate nothing.
  - ⚠️ **Corrected 2026-07-28 by sub-1-3 at implementation time** (Rule 24 lane ①, sub-1-3 sub-task 4.3). This line previously read "stage constants + sentinels via sub-1-3 (registry codes, no stamp)", which was true of the sentinels but **false of the stage constants**: sub-1-3 AC #1 stamps the 12-value `PipelineStage` set `[@contract-v1]`. Without the ack line above, this story would have shipped a HIGH-severity Rule 20 consumer-side gap at its own CR. No re-architecture is implied — the contract is exactly the 12 values this story already planned to broadcast.
- **This story stamps nothing new** — it is pure composition; the HTTP request/response shape is v0 until Epic 2/FE work needs it stamped.
- **main.go touch points:** flag read → pipeline construction (deps: repos, cache, placer, converter, translation service, `modelID` from `cfg.GetClaudeModel()`/default, `configured` from `HasClaudeKey`) → pool Start + shutdown hook → scan-callback composition → handler registration → SSE adapter. Rule 15's main.go-wiring check applies to every one.
- **Why 202-async:** a translate item is minutes (AC #7b); holding HTTP open invites proxy timeouts and duplicate submits. SSE is the progress channel (J2), the run row is the record.
- **Rule 12/13/14/17** all in play; zh-TW strings live at the wiring/handler layer only (Rule 3's `message`/`suggestion` discipline).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. Rule 23 does not apply.

### References

- [architecture #D5 (:255-257) · #V2/V3/V6 (:546-557) · #P8 (:380-382) · § M1 acceptance (spec §8)]
- [`apps/api/internal/subtitle/batch.go`:238-246 · `services/scanner_service.go`:78-83,312-315 · `cmd/api/main.go`:422 · `repository/movie_repository.go`:889-918 · `internal/config/config.go`:105-131]
- [IR report 2026-07-27 — F4/F9 rulings + the Dockerfile:47/docker.yml:80 verification]
- [`project-context.md`#Rule 3/10/13/14/15/17/24 · AD #5 · NFR-P3]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) · Claude Opus 5 (1M context), effort xhigh · 2026-08-03

### Debug Log References

RED verified before every task (build failure naming exactly the new symbols, or a failing assertion):

| Task | RED signal |
|---|---|
| 1 | `vet: subtitle_pipeline_test.go:25: undefined: SubtitlePipelineModeLegacy` · `vet: batch_seam_test.go:94: undefined: batchEngine` |

Falsification checks (deliberately break the implementation, confirm the test catches it, restore):

| Guard | Falsification | Result |
|---|---|---|
| D5 seam routes to the pipeline | `if bp.item != nil` forced to `if false` | `TestBatchSeam_PipelineModeRoutesToProcessItem` FAILS |
| Pipeline error is a failed item | `failItem` path returns `EngineResult{Success: true}` | `TestBatchSeam_PipelineFailureCountsAsAFailedItem` FAILS |

### Completion Notes List

- 🔗 **AC Drift: NONE** (checked `batch.go` / `engine.Process` / `SetOnScanComplete` across `_bmad-output/implementation-artifacts/*.md` — the only prior behavioural hit is **8-9 sub-task 2.8** "Call `engine.Process()` with ProcessOptions", and it still holds: the D5 flag defaults to `legacy`, which routes to `engine.Process` with byte-identical arguments (pinned by `TestBatchSeam_LegacyModeIsByteIdentical`, which asserts all six captured arguments including the CN-policy `ProductionCountry`). The `pipeline` branch is a NEW opt-in path, not a redefinition of the shipped one.)
- 📎 **Contract Stamps: FOUND** (3 ack references in this story; 0 produced). This story is pure composition and stamps nothing new — the HTTP request/response shape stays v0 until Epic 2/FE needs it stamped. Consumed, all greped at v1 and all reconciling:
  - `confirmed against [@contract-v1] sub-1-5b AC #1` — `ProcessItem` / `MediaRef` / `ProcessItemOptions` / `ProcessOutcome`. Consumed unchanged; the batch seam calls the stamped signature directly.
  - `confirmed against [@contract-v1] (Story sub-1-3 AC #1)` — the 12-value `PipelineStage` set this story broadcasts.
  - `confirmed against [@contract-v1] sub-1-2 AC #1` — `MediaRef.MediaType` vocabulary (movie|series|episode).
  - Upstream bump scan: the only `[@contract-vN→vN+1]` token anywhere in the upstream set is sub-1-5b's Open-Question **hypothetical** ("*Alternative: a dedicated failed media status — but that's a `[@contract-v1→v2]` bump*"), not an actual bump. No stale-mark is owed to this story.

### Discovery Triage

- **Pre-recorded at authoring:**
  - **① scanner hook already exists + occupied** → absorbed as AC #2's zero-edit composition ruling + AC #10's delta-tree correction.
  - **③ `docs/deployment.zh-TW.md` missing entirely** (pre-existing Rule 17 debt) → `backlog-deployment-doc-zh-tw-twin` filed with this story (bidirectional); AC #8 edits EN only.
  - **① PRD "tens of seconds" vs chunked reality** → absorbed as AC #7(b)'s honest bar + flagged as an optional PRD edit (Open Questions).
  - Episode-enumeration decision tree (Task 2) may add a lane-① AC at implementation.
- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
|---|---|
| `apps/api/internal/config/subtitle_pipeline.go` | **new** — D5 flag: `SubtitlePipelineModeLegacy`/`…Pipeline` constants, closed-set `validateSubtitlePipelineMode`, and the single `Config.SubtitlePipelineEnabled()` predicate the wiring reads |
| `apps/api/internal/config/config.go` | **modified** — `SubtitlePipelineMode` field + `loadString("VIDO_SUBTITLE_PIPELINE_MODE", "legacy")` with startup validation (an unknown value fails the process rather than silently staying legacy) |
| `apps/api/internal/config/subtitle_pipeline_test.go` | **new** — 5-row mode table (default / explicit legacy / pipeline / unknown / wrong case) + the `SubtitlePipelineEnabled` predicate incl. the zero-value guard |
| `apps/api/internal/subtitle/batch.go` | **modified** — `batchEngine` port (so the legacy path is spy-provable), `ItemProcessor` seam + `BatchProcessorOption`/`WithItemProcessor` on a variadic `NewBatchProcessor`, the ONE D5 conditional at the former `:244`, and `processViaPipeline` mapping `ProcessOutcome` onto the batch's `EngineResult` bookkeeping |
| `apps/api/internal/subtitle/batch_seam_test.go` | **new** — 4 tests: legacy byte-identity over all six captured engine arguments, pipeline routing with the engine fully bypassed, a pipeline error counting as a failed item, and a P5 pre-flight early-exit counting as a success |

### Change Log

| Date | Change |
|---|---|
| 2026-08-03 | **Task 1 (AC #1) — RED first** (`undefined: SubtitlePipelineModeLegacy`, then `undefined: batchEngine`). The flag is an env var with a CLOSED value set validated at startup: an unknown `VIDO_SUBTITLE_PIPELINE_MODE` fails the process instead of silently falling back to legacy, because a typo'd `pipelien` that quietly kept the old behaviour looks exactly like "the pipeline is broken" to an operator who believes they enabled it. Nothing downstream of main.go compares the raw string — `SubtitlePipelineEnabled()` is the single read (D5's ban on a flag that spreads). The seam needed a `batchEngine` port before it could be tested at all: with `bp.engine` typed as the concrete `*Engine`, "legacy is byte-identical" could only ever be a claim about a diff; it is now an assertion over all six captured arguments including the CN-policy `ProductionCountry`. `processViaPipeline` maps `ProcessOutcome` onto `EngineResult` so the batch's success/fail bookkeeping reads the same either side of the seam — with two deliberate mappings: Score/ProviderUsed stay zero (a generated subtitle was never scored against providers, sub-1-5b AC #6.3), and a nil-`Run` outcome (the P5 pre-flight early-exit) is a SUCCESS, since "this item already has its sidecar" is the desired result of a re-scan. |

---

## Open Questions for Alexyu (AC #7 numbers need your confirmation — the rest proceed on stated rulings)

1. ~~**G2/G4 bars (AC #7).**~~ ✅ **RESOLVED 2026-07-27 (Alexyu): all three confirmed** — (b) episode ≤ **3 min** (5 was 太慢) / movie ≤ **6 min** / direct ≤ **60 s**; (a) ≤ 1 core / ≤ 256 MB and (c) **90%** over 20 items confirmed verbatim (「a跟c可以」).
2. **PRD prose:** "time-to-`.zh-Hant.srt` on the order of tens of seconds (… one translation call)" is inconsistent with chunk=10 sequential reality. Edit the PRD line, or leave it and let the pilot report supersede?
3. **Deployment zh-TW twin:** filed as backlog (lane ③). If you'd rather 1.6 create the full translation now, say so and Task 5 expands.
