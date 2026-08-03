# Story sub-1.6: Wire triggering, gating, and progress

Status: done

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
- [x] **Task 2 — Pool + enqueue (AC #2, #3, #11):** pool with lifecycle/dedup/overflow; `EnqueueMissing` (movies via `FindMissingZhHantSubtitle`; **decision tree:** locate the 9R-16 episode enumeration — found ⇒ reuse; absent ⇒ expand scope with a new AC per Rule 24 lane ①, do not silently build a new query); main.go callback composition preserving `:422`.
- [x] **Task 3 — Endpoint (AC #4):** new handler + routes + Swagger + `swag init` + main.go wiring + Rule 15 route verification.
- [x] **Task 4 — Gate (AC #5):** `configured` wiring, three-entry-point coverage, zh-TW messages, tests.
- [x] **Task 5 — SSE + docs (AC #6, #8):** progress-hook→hub adapter + zh-TW stage messages; deployment.md EN section; file `backlog-deployment-doc-zh-tw-twin` in sprint-status (verify not already filed).
- [x] **Task 6 — Sync + gates (AC #7, #9, #10):** record the pilot-bar numbers as confirmed (or adjusted) by Alexyu; delta-tree corrections; full test + lint gates; Rule 20 ack recorded.

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
| 2 | `vet: worker_pool_test.go: undefined: NewWorkerPool` · `vet: media_store_test.go: undefined: NewMediaStore` · `vet: scan_callback_test.go:41: undefined: ComposeScanCallback` |
| 3 | `vet: subtitle_pipeline_handler_test.go:58: undefined: SubtitlePipelineHandler` · `vet: worker_pool_test.go:352: pool.EnqueueItem undefined` |
| 4 | `vet: batch_seam_test.go:229: undefined: WithPipelineGate` |
| 5 | `vet: progress_sse_test.go:30: undefined: NewSSEProgressHook` |

Falsification checks (deliberately break the implementation, confirm the test catches it, restore):

| Guard | Falsification | Result |
|---|---|---|
| D5 seam routes to the pipeline | `if bp.item != nil` forced to `if false` | `TestBatchSeam_PipelineModeRoutesToProcessItem` FAILS |
| Pipeline error is a failed item | `failItem` path returns `EngineResult{Success: true}` | `TestBatchSeam_PipelineFailureCountsAsAFailedItem` FAILS |
| FR23 gate on the batch seam | `pipelineAllowed`'s predicate check forced to `if true` | `TestBatchSeam_UnconfiguredGateFallsBackToLegacy` FAILS |
| FR23 gate on the endpoint | the handler's `configured` check forced to `if false` | `TestRunPipeline_409AndNoEnqueueWhenTranslationKeyMissing` FAILS |
| `force` survives the queue | `process` reverted to `ProcessItemOptions{}` instead of `queued.opts` | `TestWorkerPool_EnqueueItemCarriesOptionsToProcessItem` FAILS |
| SSE payload carries the item identity | `media_type` hardcoded to `""` | `TestSSEProgressHook_EmitsSubtitleProgressWithTheItemIdentity` FAILS |
| Enrichment runs BEFORE the enqueue | `prev()` moved after `EnqueueMissing` | **first attempt STILL PASSED** — the original assertion only proved both halves ran, not their order (two buffered channel receives are order-blind to a swap). Test rewritten to record `enrichment / enumerate / observed` into one inline slice; the same falsification now FAILS. A dropped `prev()` was caught either way (the test blocks, then fails) — but "wrap, don't replace" is an ORDER claim and now reads as one |

### Completion Notes List

- 🔗 **AC Drift: NONE** (checked `batch.go` / `engine.Process` / `SetOnScanComplete` across `_bmad-output/implementation-artifacts/*.md` — the only prior behavioural hit is **8-9 sub-task 2.8** "Call `engine.Process()` with ProcessOptions", and it still holds: the D5 flag defaults to `legacy`, which routes to `engine.Process` with byte-identical arguments (pinned by `TestBatchSeam_LegacyModeIsByteIdentical`, which asserts all six captured arguments including the CN-policy `ProductionCountry`). The `pipeline` branch is a NEW opt-in path, not a redefinition of the shipped one.)
- 📎 **Contract Stamps: FOUND** (3 ack references in this story; 0 produced). This story is pure composition and stamps nothing new — the HTTP request/response shape stays v0 until Epic 2/FE needs it stamped. Consumed, all greped at v1 and all reconciling:
  - `confirmed against [@contract-v1] sub-1-5b AC #1` — `ProcessItem` / `MediaRef` / `ProcessItemOptions` / `ProcessOutcome`. Consumed unchanged; the batch seam calls the stamped signature directly.
  - `confirmed against [@contract-v1] (Story sub-1-3 AC #1)` — the 12-value `PipelineStage` set this story broadcasts.
  - `confirmed against [@contract-v1] sub-1-2 AC #1` — `MediaRef.MediaType` vocabulary (movie|series|episode).
  - Upstream bump scan: the only `[@contract-vN→vN+1]` token anywhere in the upstream set is sub-1-5b's Open-Question **hypothetical** ("*Alternative: a dedicated failed media status — but that's a `[@contract-v1→v2]` bump*"), not an actual bump. No stale-mark is owed to this story.
- 🎭 **A11y Pre-Flight: N/A** (100% backend — zero `apps/web/` files touched; the story's cross-stack split check recorded frontend tasks = 0 at drafting and that held).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story. The M1 badge/rendering work is sub-1-7a (design) and sub-1-7b (frontend).
- ✅ **AC #9 fence held — zero frontend edits.** The lane-② escape hatch ("if a FE edit turns out to be required to point 生成字幕 at the new endpoint, stop and file it") was **not** triggered: nothing in `apps/web/` calls `/subtitles/pipeline/run`, and M1's button keeps riding the existing generation-dialog surface. Also untouched, as fenced: `scanner_service.go`, `sse/hub.go`, `subtitle_handler.go`; zero new **registered** Rule 7 codes (both 409s reuse `AI_NOT_CONFIGURED`), so `project-context.md`'s prefix list and `code-review/instructions.xml` need **zero** edits — verified, not assumed. *(Amended at CR 2026-08-03: the M1 fix added ONE inline literal `SUBTITLE_QUEUE_FULL` (503) under the already-registered `SUBTITLE_` prefix, following the shipped `subtitle_handler.go` inline-literal precedent — `SUBTITLE_PROVIDER_NOT_FOUND`/`SUBTITLE_PLACE_FAILED`/etc. are likewise not in the registry list. Registry and CR-workflow still untouched.)*
- 📊 **AC #7 bars are recorded, not measured here.** All three were confirmed by Alexyu on 2026-07-27 and stand as citable M1 acceptance bars for the DS920+ pilot: (a) ≤1 core sustained + ≤256 MB incremental RSS with playback still working, (b) ≤3 min per ~600-cue episode / ≤6 min per ~1,200-cue movie / ≤60 s for the no-LLM path, (c) ≥90% of a 20-item sample accepted without hand-editing. Per-item timing needs no new instrumentation — `subtitle_runs.started_at/completed_at` already carry it. The 3-min bar implies ≈3 s average per 10-cue chunk including quality-gate retries; if the pilot misses it, the two levers in order are a larger chunk size then post-D10 chunk parallelism, and **neither is in scope now**.
- 📘 **`swag init` was NOT run — it is a no-op in this repo.** `apps/api` has no swaggo dependency and no `docs` package (backend-consolidation Phase 1 Step 1.2 is still open, per `project-context.md`), so there is no Swagger artifact to regenerate. The handler carries full `@Summary`/`@Param`/`@Failure`/`@Router` annotations matching every other handler in the package, and they will be picked up when Swagger lands. Pre-existing and already tracked by the consolidation plan — no new entry filed.
- ⚠️ **Pre-existing lint noise, not a repo failure:** `pnpm lint:all` reports 2 errors, both in `tmp/shot.mjs` and `tmp/verify-posters.mjs` — local scratch files under the gitignored `tmp/` directory (`.gitignore:74`). They do not exist in a CI checkout, so CI lint is unaffected; nothing outside `tmp/` errors (120 warnings, all pre-existing, none introduced by this story — which touched zero JS/TS). No fix and no backlog entry: there is no repo defect to track.
- ✅ **Closes `backlog-subtitle-status-writer-search-columns`** (filed by sub-1-5b at discovery). The `MediaStore` adapter is the sole status writer for generated subtitles, and movies/series now have the narrow `UpdateSubtitleGenerationStatus` that episodes already had. sprint-status entry updated to `done` with the bidirectional link.

### Code Review (AI) — 2026-08-03, adversarial CR + auto-fix

**Findings: 1 HIGH / 3 MEDIUM / 3 LOW. All HIGH+MEDIUM fixed in-story; falsification-verified.** Mandatory checks: 🔒 Rule 7 Wire Format **PASS** (0 new error-code constants; in-scope grep clean) · 🔒 Rule 20 Contract Bump **N/A** (no stamp bumps — the only arrow token in the diff is a quoted hypothetical) · 🔒 Rule 25 Mega-line **N/A** (project-context.md untouched) · Git vs File List discrepancies: 0.

| # | Sev | Finding | Fix |
|---|---|---|---|
| H1 | HIGH | Terminal-verdict items (`no_text_source`/`skipped`) keep `subtitle_language` NULL, so `EnqueueMissing` re-enumerated them on EVERY scan; the P5 pre-flight cannot gate them (no sidecar exists) ⇒ each sweep re-probed the file and appended a fresh `skipped` run row, without bound — contradicting ProcessItem's own "no row per scan" design note. | `terminalPipelineVerdict` filter in `EnqueueMissing` (`worker_pool.go`), deliberately in Go not SQL — the movie query is shared with 9R-16 Route C, where `no_text_source` IS the ASR recovery scope. `not_found`/`found` stay enumerable (search verdicts, still lack zh-Hant). Manual endpoint bypasses the sweep, so an operator can still re-run a verdict item. Test: `TestWorkerPool_EnqueueMissingSkipsTerminalVerdicts`. |
| M1 | MED | `EnqueueItem` collapsed dup / overflow-drop / stopped-pool into one `false`, which the endpoint answered as 202 `already_queued` — promising work that was in fact discarded. | `EnqueueOutcome` enum (`Accepted`/`Duplicate`/`QueueFull`/`Stopped`); handler maps Duplicate→202 `already_queued`, QueueFull→**503 `SUBTITLE_QUEUE_FULL`** (inline literal under the registered prefix, `subtitle_handler.go` precedent), Stopped→the existing 409. Tests: `TestRunPipeline_QueueFullIs503NotAlreadyQueued`, `TestRunPipeline_PoolStoppedMidRequestIs409`, `TestWorkerPool_EnqueueItemReportsDistinctOutcomes`. |
| M2 | MED | `Stop()` early-returned without `wg.Wait()` when a ctx-cancel had already flipped `running` via `stopFromWorker` — and main.go's shutdown order is exactly `cancel()` then `Stop()`, so Stop could return while a worker was still mid-item, racing its `failItem` cleanup writes against the rest of shutdown. | `wg.Wait()` moved outside the running check — Stop always waits. Test: `TestWorkerPool_StopAfterContextCancelStillWaitsForInFlightWork` (busy worker parked through the cancel, like a `WithoutCancel` cleanup write). |
| M3 | MED | A Go select picks randomly among ready cases, so a worker whose stopCh had closed could keep pulling buffered items — one multi-minute translate per pull, holding Stop's `wg.Wait` open arbitrarily long. | Loop-head stop-priority pre-check in `run()`. Test: `TestWorkerPool_StopDoesNotDrainBufferedItems` (also pins that buffered items stay queued, not silently consumed). |

Falsification (same discipline as the implementation tasks — break the fix, watch the guard fire, restore): H1 filter forced to `return false` → `SkipsTerminalVerdicts` FAILS · M2 `wg.Wait` moved back inside the branch → `StillWaitsForInFlightWork` FAILS · M3 pre-check deleted → `StopDoesNotDrainBufferedItems` FAILS 3/3 runs. All restored; `go vet` + full `go test ./...` green, `-race` clean on the pool/handler tests, `gofmt -l` clean on every touched file.

**LOW (recorded, not fixed — accepted debt):** L1 `media_store.go:151` — the parent-series load error is discarded with a comment but no log line; a `logger.Debug` would aid diagnosing absent metadata (needs a logger on the adapter — constructor ripple not worth it here). L2 `progress_sse.go` — `已略過：`/`字幕生成失敗：` carry the English machine reason verbatim (mixed-language user string; deliberate per the code comment, flagged as zh-TW copy debt for the F3 surface). L3 `scan_callback.go:40` — `EnqueueMissing` runs on `context.Background()`; a scan finishing mid-shutdown runs non-cancellable enumeration queries (bounded: two indexed queries + non-blocking sends).

### Discovery Triage

- **Pre-recorded at authoring:**
  - **① scanner hook already exists + occupied** → absorbed as AC #2's zero-edit composition ruling + AC #10's delta-tree correction.
  - **③ `docs/deployment.zh-TW.md` missing entirely** (pre-existing Rule 17 debt) → `backlog-deployment-doc-zh-tw-twin` filed with this story (bidirectional); AC #8 edits EN only.
  - **① PRD "tens of seconds" vs chunked reality** → absorbed as AC #7(b)'s honest bar + flagged as an optional PRD edit (Open Questions).
  - Episode-enumeration decision tree (Task 2) may add a lane-① AC at implementation.
- **Triaged AT implementation (2026-08-03):**
  - **① 9R-16's episode enumeration does not exist** (the decision tree's `absent` branch fired) → absorbed as **AC #11** + `EpisodeRepository.FindMissingZhHantSubtitle`, exactly as the tree required; nothing invented silently.
  - **① Architecture V6's "single check at the top of `pipeline.go`" is not implementable** — `NewPipeline` panics on a nil translator, so the unconfigured case has no Pipeline object to gate → absorbed into AC #5's implementation (one `func() bool` in main.go feeding three injection points) + the V6 entry corrected in `subtitle-pipeline-architecture.md`.
  - **① Delta tree missing 7 shipped files** (`process_item.go`, `worker_pool.go`, `media_store.go`, `scan_callback.go`, `progress_sse.go`, `segment_cache.go`, `show_gate.go`) → absorbed into AC #10's sync pass, counts re-tallied.
  - **✅ closed, not deferred:** `backlog-subtitle-status-writer-search-columns` → fixed by this story's `MediaStore` adapter + the two new narrow repository writers; sprint-status entry moved to `done`.
  - **Pre-existing, already tracked elsewhere — no new entry:** `swag init` is a no-op (no swaggo in `apps/api`; backend-consolidation Phase 1 Step 1.2 owns it) · 2 `pnpm lint:all` errors live in the gitignored `tmp/` scratch directory and do not exist in a CI checkout.
  - **Unchanged from authoring:** `backlog-deployment-doc-zh-tw-twin` stays `backlog` (verified filed at `sprint-status.yaml:978`); AC #8 edited the EN doc only.
- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
|---|---|
| `apps/api/internal/config/subtitle_pipeline.go` | **new** — D5 flag: `SubtitlePipelineModeLegacy`/`…Pipeline` constants, closed-set `validateSubtitlePipelineMode`, and the single `Config.SubtitlePipelineEnabled()` predicate the wiring reads |
| `apps/api/internal/config/config.go` | **modified** — `SubtitlePipelineMode` field + `loadString("VIDO_SUBTITLE_PIPELINE_MODE", "legacy")` with startup validation (an unknown value fails the process rather than silently staying legacy) |
| `apps/api/internal/config/subtitle_pipeline_test.go` | **new** — 5-row mode table (default / explicit legacy / pipeline / unknown / wrong case) + the `SubtitlePipelineEnabled` predicate incl. the zero-value guard |
| `apps/api/internal/subtitle/batch.go` | **modified** — `batchEngine` port (so the legacy path is spy-provable), `ItemProcessor` seam + `BatchProcessorOption`/`WithItemProcessor` on a variadic `NewBatchProcessor`, the ONE D5 conditional at the former `:244`, and `processViaPipeline` mapping `ProcessOutcome` onto the batch's `EngineResult` bookkeeping |
| `apps/api/internal/subtitle/batch_seam_test.go` | **new** — 6 tests: legacy byte-identity over all six captured engine arguments, pipeline routing with the engine fully bypassed, a pipeline error counting as a failed item, a P5 pre-flight early-exit counting as a success, and (Task 4) both directions of the capability gate |
| `apps/api/internal/subtitle/worker_pool.go` | **new** — AC #3 pool: `PipelineConcurrencyM1 = 2`, 1024-cap queue with non-blocking drop-and-warn, `MediaRef`-keyed in-flight dedup (reservation taken BEFORE the send), `Start`/`Stop`/ctx-cancel lifecycle, and `EnqueueMissing` running the FR23 gate before enumeration |
| `apps/api/internal/subtitle/worker_pool_test.go` | **new** — lifecycle, dedup, overflow-drop, per-item failure isolation, ctx-cancel teardown, gate short-circuit |
| `apps/api/internal/subtitle/media_store.go` | **new** — the `MediaStore` port over the three media repositories; episodes load the PARENT SERIES metadata (a per-episode `MetadataHash` would split the segment cache per episode) and key `ShowKey` on the series id so D10 gates a show, not a row |
| `apps/api/internal/subtitle/media_store_test.go` | **new** — per-type load/dispatch, the series-metadata inheritance, the missing-parent fail-soft, and unknown-media-type errors |
| `apps/api/internal/repository/episode_repository.go` | **modified** — AC #11 `FindMissingZhHantSubtitle` mirroring the movie predicate, minus the `is_removed` clause the `episodes` table does not have, ordered `series_id, season_number, episode_number` so a show's episodes arrive contiguously for D10 |
| `apps/api/internal/repository/episode_generation_test.go` | **new** — real `:memory:` SQLite over the migrated schema (Rule 15): predicate coverage, the English-subtitle-still-in-scope case, and the show-contiguous ordering |
| `apps/api/internal/repository/subtitle_generation_status.go` | **new** — `UpdateSubtitleGenerationStatus` for movies + series: writes status/path/language ONLY, closing `backlog-subtitle-status-writer-search-columns` (the shipped `UpdateSubtitleStatus` unconditionally stamps `subtitle_last_searched`, which would make a generated sidecar look like a provider hit) |
| `apps/api/internal/repository/subtitle_generation_status_test.go` | **new** — asserts the search columns stay NULL after a generation write, against a real migrated DB |
| `apps/api/internal/repository/interfaces.go` | **modified** — Rule 11 registration of the three new repository methods |
| `apps/api/internal/repository/movie_repository.go` · `series_repository.go` | **modified** — shared select-column/scan plumbing for the generation-status writer |
| `apps/api/internal/subtitle/scan_callback.go` | **new** — `ComposeScanCallback`: WRAPS the single-slot scan-complete callback (AC #2). A nil pool returns `prev` untouched, which is what makes legacy mode a no-op rather than a wrapper that has to be reasoned about |
| `apps/api/internal/subtitle/scan_callback_test.go` | **new** — enrichment-runs-first ordering, legacy passthrough, nil-prev, and enumeration errors reaching the observer |
| `apps/api/internal/testutil/mocks.go` + 6 `*_test.go` files | **modified** — mock/fake conformance for the three new interface methods |
| `apps/api/internal/handlers/subtitle_pipeline_handler.go` | **new** — FR12 `POST /api/v1/subtitles/pipeline/run`: the AC #5 gate checked before any DB work, 400/404/409/500 table, Swagger annotations, Rule 3 envelope. `subtitle_handler.go` untouched (D3). **CR 2026-08-03 (M1):** enqueue outcomes answered distinctly — 202 `queued`/`already_queued`, overflow → 503 `SUBTITLE_QUEUE_FULL`, stopped-pool race → 409 |
| `apps/api/internal/handlers/subtitle_pipeline_handler_test.go` | **new** — 13 tests over the full status table incl. `force` reaching the queue, `tv` rejected as a media type, the two distinct 409 messages, and (CR 2026-08-03) queue-full → 503 + stopped-mid-request → 409 |
| `apps/api/internal/subtitle/media_store.go` | **modified** — `ErrMediaNotFound` sentinel so the endpoint can tell "no such id" (404) from "database is locked" (500) |
| `apps/api/internal/subtitle/worker_pool.go` | **modified** — `EnqueueItem(ref, opts)` carries FR32's `force` through the queue; dedup still keys on the `MediaRef` alone. **CR 2026-08-03:** `terminalPipelineVerdict` filter in `EnqueueMissing` (H1), 4-value `EnqueueOutcome` (M1), `Stop()` always `wg.Wait`s (M2), loop-head stop-priority in `run()` (M3) |
| `apps/api/internal/subtitle/batch.go` | **modified (Task 4)** — `WithPipelineGate` + `pipelineAllowed()`: AC #5's third entry point, denial logged through a `sync.Once` and falling back to the provider search rather than failing every item |
| `apps/api/internal/subtitle/progress_sse.go` | **new** — the AC #6 bridge: progress hook → `subtitle_progress` (event type and payload shape unchanged) with zh-TW messages composed at the wiring layer, and the shared `translateChunkProgressFormat` const |
| `apps/api/internal/subtitle/progress_sse_test.go` | **new** — 7 tests: identity in every payload, per-stage zh-TW mapping, chunk counters surviving the translation, the unparseable fallback, terminal-stage reasons, nil-hub safety, and the `WithProgress` wiring |
| `apps/api/internal/subtitle/pipeline.go` | **modified** — the translate-progress literal replaced by `translateChunkProgressFormat` so the emitter and the parser cannot drift |
| `apps/api/internal/subtitle/worker_pool_test.go` | **modified (Tasks 3–4)** — `EnqueueItem` options/dedup coverage and the gate short-circuit spy (`spyFinder` proves enumeration never runs when unconfigured). **CR 2026-08-03:** +4 guards — terminal-verdict filter, distinct enqueue outcomes, Stop-waits-after-cancel, no-post-stop-drain |
| `apps/api/internal/subtitle/scan_callback_test.go` | **modified (Task 6)** — the ordering assertion rewritten after falsification showed a reorder slipped through; now records `enrichment / enumerate / observed` into one inline slice |
| `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md` | **modified** — AC #10 delta-tree sync: `scanner_service.go` ✏️ → 🔒 with the reason, `handlers/subtitle_pipeline_handler.go` 🆕, the 7 previously-unlisted 1.5b/1.6 files added, counts re-tallied, and the **V3** + **V6** validation entries corrected in place |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `sub-1-6-wire-triggering-gating` → `review` + implementation summary; `backlog-subtitle-status-writer-search-columns` → `done` with the bidirectional resolution note |
| `docs/deployment.md` | **modified** — AC #8: "What the image already includes" (bundled ffmpeg/ffprobe + the silent-degradation warning; multi-arch amd64/arm64) under § Prerequisites, and a "Subtitle Generation Variables" table (`VIDO_SUBTITLE_PIPELINE_MODE`, `CLAUDE_API_KEY`) under § Configuration |
| `apps/api/cmd/api/main.go` | **modified** — the whole wiring: `postScanEnrichment` hoisted out of the `SetOnScanComplete` call so it can be composed instead of replaced; pipeline + pool construction behind `cfg.SubtitlePipelineEnabled()`; the D5 batch seam via `WithItemProcessor`; pool `Start` in the goroutine zone and `Stop` in the graceful-shutdown block |

### Change Log

| Date | Change |
|---|---|
| 2026-08-03 | **Adversarial CR (code-review workflow) — 4 fixes, story → done.** H1: `EnqueueMissing` now filters `no_text_source`/`skipped` (`terminalPipelineVerdict`, Go-side so 9R-16 Route C's shared movie query keeps its ASR scope) — ends the per-scan re-probe + unbounded `subtitle_runs` growth for permanently-declined items. M1: `EnqueueItem` returns a 4-value `EnqueueOutcome`; the endpoint answers overflow with 503 `SUBTITLE_QUEUE_FULL` (inline literal, registered prefix, registry untouched) instead of a lying 202 `already_queued`. M2: `Stop()` always `wg.Wait`s (the cancel-then-Stop shutdown order could skip it). M3: worker loop-head stop-priority (select randomness no longer lets a stopping worker drain buffered multi-minute items). All four falsification-verified; 3 LOWs recorded as accepted debt. Full `go test ./...` + `go vet` + `-race` green. |
| 2026-08-03 | **Task 6 (AC #7, #9, #10) — sync + gates.** AC #7's three bars need real hardware, so they are recorded as CITABLE M1 acceptance bars to be measured at the DS920+ pilot, not as anything this story can assert; per-item timing comes free from `subtitle_runs.started_at/completed_at`. Architecture sync went further than AC #10's three items: the delta tree was ALSO missing seven files that 1.5b and 1.6 actually shipped (`process_item.go`, `worker_pool.go`, `media_store.go`, `scan_callback.go`, `progress_sse.go`, `segment_cache.go`, `show_gate.go`), and a tree that omits half the package it describes is stale in exactly the way AC #10 exists to fix — absorbed as Rule 24 lane ①, counts re-tallied. Two architecture VALIDATION entries were corrected in place rather than left to contradict the tree: **V3**'s "Adds `scanner_service.go` ✏️" is struck (the hook already existed) and **V6**'s "a single check at the top of `pipeline.go`" is struck (impossible — `NewPipeline` panics on a nil translator, so the unconfigured case has no Pipeline to gate). Gates: `go build ./...` + `go test ./...` green across the whole api module; `pnpm nx test web` 2457/2457 green; `gofmt -l` clean on every touched Go file; `prettier --check` clean on all three touched markdown files. |
| 2026-08-03 | **Task 5 (AC #6, #8) — RED first** (`undefined: NewSSEProgressHook`). The bridge is a package-level function, not a closure in main.go, precisely so AC #9's "SSE wiring emits stage+message per hook call (fake hub)" is testable. Event type and payload shape are byte-identical to what `engine.broadcastStatus` already sends, so `sse/hub.go` stays locked and the frontend keeps its listener — the four new stage values reach the wire here for the first time. One non-obvious coupling had to be made explicit: the translate message carries the chunk counters (`第 7/60 段`), which are the ONLY progress signal during the multi-minute half of a run, so the bridge parses them back out of the pipeline's English line. Two independent literals would have drifted silently and degraded the counter to a static 「翻譯中…」 with nothing failing anywhere — hence the shared `translateChunkProgressFormat` const, with pipeline.go now formatting through it. zh-TW composition lives at the wiring layer (Rule 3): the pipeline's own strings stay English log lines. Docs: verified against the shipped infrastructure before writing (`Dockerfile:72` `apk add … ffmpeg`, `docker.yml:82` `linux/amd64,linux/arm64` — the story's cited line numbers had drifted, the facts had not). `backlog-deployment-doc-zh-tw-twin` confirmed already filed at `sprint-status.yaml:978`; EN only, per AC #8. |
| 2026-08-03 | **Task 4 (AC #5) — RED first** (`undefined: WithPipelineGate`). The AC's "single check owned by the Pipeline" could not be taken literally: `NewPipeline` PANICS on a nil translator, so an unconfigured install has no Pipeline object to hang a gate on, and a gate that only exists when the thing it gates exists is not a gate. The predicate therefore lives one level up — `subtitleCapabilityGate := cfg.HasClaudeKey`, declared ONCE in main.go and read by all three entry points: the endpoint (409 `AI_NOT_CONFIGURED`, before any DB work), the enqueue sweep (`WithCapabilityGate` — short-circuits BEFORE enumeration, so an unconfigured install pays no query per scan), and the batch seam (`WithPipelineGate`). The batch gate is belt-and-braces over main.go's structural gate (no key ⇒ no `WithItemProcessor` ⇒ legacy), and it falls back to the provider search rather than failing items: a user who asked for subtitles should get the ones that are still reachable. Its denial is logged through a `sync.Once` — a 400-episode batch must not write 400 identical lines. Both 409 paths deliberately reuse `AI_NOT_CONFIGURED` (the fence forbids new Rule 7 codes) with DIFFERENT zh-TW messages, because the operator's next action differs: set `CLAUDE_API_KEY`, versus set `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`. |
| 2026-08-03 | **Task 3 (AC #4) — RED first** (`undefined: SubtitlePipelineHandler`). Two shapes the AC text did not spell out, both forced by the code: (1) the pool's queue had to start carrying `ProcessItemOptions`, because `force` is a REQUEST field and a queue of bare `MediaRef`s would have accepted `force: true` and silently run the cached path — dedup still keys on the ref alone, since two submits of one item are the same work whatever their options; (2) `subtitle.ErrMediaNotFound` became a sentinel, because "no such media id" (404) and "database is locked" (500) were previously the same formatted string, and collapsing them would send an operator hunting for a row that is actually there. The route is registered in EVERY mode — an API surface that changes shape with an env var is worse than one that answers 409 honestly — and legacy mode reaches the handler with a nil queue, which is why `IsRunning()` is part of the narrow interface. `already_queued` is a 202, not a 409: the caller wanted the item processed and it IS being processed. **`swag init` is a no-op here and was NOT run**: `apps/api` has no swaggo dependency and no `docs` package (consolidation-plan Phase 1 Step 1.2 is still open), so the annotations were written to match every other handler in the package and will be picked up when Swagger lands. |
| 2026-08-03 | **Task 2 (AC #2, #3, #11) — RED first** (`undefined: ComposeScanCallback`). Task 2's decision tree fired its **absent** branch — 9R-16's episode enumeration does not exist (it shipped movies-only) — so per Rule 24 lane ① the scope expanded with a tracked AC (#11) rather than a silently-invented query. The episode predicate mirrors the movie one with two deliberate differences: **no `is_removed` clause** (the `episodes` table has no such column — mirroring blindly would have been a query against a column that does not exist) and an explicit `series_id, season_number, episode_number` ordering, which is load-bearing rather than cosmetic: contiguous episodes are what let sub-1-5b's D10 latch warm ONE prompt prefix per show instead of re-warming it every time the queue interleaves two series. The pool drops on overflow instead of blocking because the alternative is stalling the scanner's completion path, and the next scan re-enqueues for free (the P5 pre-flight makes re-enumeration cheap) — the dedup reservation is taken BEFORE the channel send and released if the send fails, so an overflowed item stays re-enqueueable instead of being lost permanently. The `MediaStore` adapter owns the fix for `backlog-subtitle-status-writer-search-columns`: the shipped `UpdateSubtitleStatus` unconditionally stamps `subtitle_last_searched`, which would have made every generated sidecar indistinguishable from a provider hit in the library UI (sub-1-5b AC #6.3), so movies and series got the narrow generation-only writer episodes already had. main.go composes rather than re-registers the scan callback — `SetOnScanComplete` holds ONE function, and calling the setter twice would compile, pass every test, and silently stop enriching newly-scanned media. |
| 2026-08-03 | **Task 1 (AC #1) — RED first** (`undefined: SubtitlePipelineModeLegacy`, then `undefined: batchEngine`). The flag is an env var with a CLOSED value set validated at startup: an unknown `VIDO_SUBTITLE_PIPELINE_MODE` fails the process instead of silently falling back to legacy, because a typo'd `pipelien` that quietly kept the old behaviour looks exactly like "the pipeline is broken" to an operator who believes they enabled it. Nothing downstream of main.go compares the raw string — `SubtitlePipelineEnabled()` is the single read (D5's ban on a flag that spreads). The seam needed a `batchEngine` port before it could be tested at all: with `bp.engine` typed as the concrete `*Engine`, "legacy is byte-identical" could only ever be a claim about a diff; it is now an assertion over all six captured arguments including the CN-policy `ProductionCountry`. `processViaPipeline` maps `ProcessOutcome` onto `EngineResult` so the batch's success/fail bookkeeping reads the same either side of the seam — with two deliberate mappings: Score/ProviderUsed stay zero (a generated subtitle was never scored against providers, sub-1-5b AC #6.3), and a nil-`Run` outcome (the P5 pre-flight early-exit) is a SUCCESS, since "this item already has its sidecar" is the desired result of a re-scan. |

---

## Open Questions for Alexyu (AC #7 numbers need your confirmation — the rest proceed on stated rulings)

1. ~~**G2/G4 bars (AC #7).**~~ ✅ **RESOLVED 2026-07-27 (Alexyu): all three confirmed** — (b) episode ≤ **3 min** (5 was 太慢) / movie ≤ **6 min** / direct ≤ **60 s**; (a) ≤ 1 core / ≤ 256 MB and (c) **90%** over 20 items confirmed verbatim (「a跟c可以」).
2. **PRD prose:** "time-to-`.zh-Hant.srt` on the order of tens of seconds (… one translation call)" is inconsistent with chunk=10 sequential reality. Edit the PRD line, or leave it and let the pilot report supersede?
3. **Deployment zh-TW twin:** filed as backlog (lane ③). If you'd rather 1.6 create the full translation now, say so and Task 5 expands.
