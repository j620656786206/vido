# Story 3.1: Wire the ASR fallback leg into the automatic pipeline

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner,
I want the automatic subtitle pipeline to fall through to speech recognition when a media file carries no usable embedded text subtitle track,
so that the ~68% of my library that has no extractable subtitle actually gets one, instead of silently terminating at `no_text_source`.

## Context — why this story exists, and why it is the ONLY M2 story

`_bmad-output/implementation-artifacts/spike-2026-08-06-pipeline-ordering-evidence.md` (PR #207) settled the M2 ordering question with measurements:

- **68.3%** of the library (142/208 sample, `adults/` excluded) reaches this layer — no Chinese text track, no foreign text track, no sidecar. Today `ProcessItem` **terminates** on all of them.
- The **online-search layer is cancelled** for the automatic pipeline. End-to-end usable rate against the real fall-through population was **1/14 (7%)**, and that single "pass" was the **wrong episode** (`Supernatural.S15E01` aligned to an `S15E04` audio track). 13/14 returned zero Chinese results — the fall-through population is **77% Chinese-native content**, where Assrt hit rate is 6%.
- `alass` cannot serve as an automatic acceptance gate: cross-show negatives aligned to **2.81s / 5.21s**, overlapping the **≤9.08s** range where genuinely-correct-but-offset subtitles live. **20% failure rate**, and the failure mode is a user watching a different show's dialogue.
- ASR is measured and cheap: on the target NAS (i5-12400) `faster-whisper small int8` runs **22.2× realtime → ~2 min per 45-min episode**; the transcribe→translate→place chain was **run end-to-end on 2026-08-06** (50 cues / 24s / $0.029).

So M2 = **one leg**: `no_text_source → ASR`. There is no M2b.

## Acceptance Criteria

1. **[@contract-v2→v3] `no_text_source` becomes an INTERMEDIATE verdict.** `models/movie.go`'s stamped D2 media-row status contract is bumped v2→v3 with the semantic change recorded inline (the value set is UNCHANGED at 10 — only `no_text_source`'s terminality changes). `worker_pool.go`'s `terminalPipelineVerdict()` no longer treats `no_text_source` as terminal; `skipped` is untouched and REMAINS terminal (it is a deliberate routing decision, not a recoverable gap).

2. **`ProcessItem` continues into ASR on `RouteNoTextSource`.** The `case RouteNoTextSource:` arm (`process_item.go:106`) no longer calls `recordSkip` unconditionally. When the ASR port is present and available it runs transcription for that media, and the run row + media status reflect the ASR outcome rather than `no_text_source`. `RouteSkip` behaviour is byte-unchanged.
   **⚠️ Scope narrowed at implementation (CR 2026-08-07, M1):** the leg is **MOVIE-only** — `TranscriptionService`'s status writer and resume reader are movie-repository-bound, so non-movie refs degrade via the AC #4 path and the episode sweep loop keeps filtering `no_text_source`. Tracked: `backlog-episode-asr-fallback`.

3. **ASR is a narrow injected port, not a concrete dependency.** A new interface on the `subtitle` package side (Rule 11) exposes only what the leg needs — the existing synchronous `TranscriptionService.RunTranscription(ctx, mediaID, filePath, mediaDir, opts...) error` seam (added by 9R-16) plus an availability probe. It is wired via a functional option like every other item-flow port and validated in `requireItemPorts()`. Rule 19: `subtitle → services` is a legal direction (engine.go precedent).

4. **ASR unavailable is a graceful degradation, not a regression.** When the port is nil (translate-only Pipeline construction, sub-1-5a) OR the service reports unavailable (no ASR key / no FFmpeg), the item records `no_text_source` exactly as it does today. A deployment without an ASR key must behave identically to before this story.

5. **The feature flag covers the new leg.** `VIDO_SUBTITLE_PIPELINE_MODE=legacy` produces byte-identical behaviour to today. Only `pipeline` mode gets the ASR leg.

6. **One budget, not two.** The ASR call rides the ctx-attached `ai.Budget` (9R-11 / 9R-16 `resolveBudget`) so ASR + LLM share the run ceiling. `ai.ErrBudgetExceeded` propagates out of the leg as a pause, not a failure, and does NOT get recorded as `no_text_source`.

7. **SSE vocabulary is decided and documented.** D2 pipeline stages (`PipelineStage`) and the transcription service's `transcription_*` events are DISTINCT contracts sharing vocabulary — they must NOT be deduplicated (project-context.md, sub-1-3 entry). The story records which stream the frontend observes for an ASR-fallback item and why, and emits the D2 stage consistently with that decision.

8. **Downstream stale-mark completed (Rule 20).** Every consumer of the D2 status contract is audited against the terminality change and the result recorded: `apps/web/src/utils/libraryStatus.ts`, `apps/web/src/components/media/EpisodeList.tsx`, the library status filter, and `apps/api/internal/subtitle/worker_pool.go`'s enqueue sweep. Consumers that need no change are listed as verified-no-change, not silently skipped.

9. **Regression + new coverage.** Tests pin: (a) `RouteNoTextSource` + available ASR → transcription invoked, media status is the ASR outcome (movie-scoped per the AC #2 narrowing; episode-degrade pinned separately); (b) `RouteNoTextSource` + nil port → `no_text_source` recorded, ASR never called; (c) `RouteNoTextSource` + unavailable service → same as (b); (d) `RouteSkip` unchanged; (e) legacy flag mode unchanged; (f) budget-exceeded propagates as pause; (g) `terminalPipelineVerdict` no longer terminal for `no_text_source` but still terminal for `skipped`.

## Tasks / Subtasks

- [x] **Task 1 — Contract bump + terminality change (AC: #1, #8)**
  - [x] Bump `models/movie.go`'s `[@contract-v2]` stamp to v3 with an inline note that the CHANGE is `no_text_source` terminality, not the value set
  - [x] `worker_pool.go`: `terminalPipelineVerdict()` drops `no_text_source`, keeps `skipped`
  - [x] Run the mandatory Rule 20 downstream stale-mark grep; record every consumer and its verdict (changed / verified-no-change) in Dev Notes → recorded in Dev Agent Record → Completion Notes (all acked consumers are `done` = frozen; AC #8 code-consumer audit table there too)

- [x] **Task 2 — ASR port (AC: #3, #4)**
  - [x] Define the narrow interface in the `subtitle` package (name it for what the pipeline needs, not after the concrete service) → `subtitle.SpeechTranscriber` (pipeline.go)
  - [x] Add the functional option + `requireItemPorts()` handling — note the port is OPTIONAL (nil = no ASR leg), unlike the five existing required ports, so it must NOT be added to the missing-ports error → `WithSpeechTranscriber` + explicit do-not-"fix" comment on `requireItemPorts`
  - [x] Adapter on the `services` side (or in `cmd/api/main.go` wiring) over `TranscriptionService` → `cmd/api/asr_adapter.go` (`pipelineASRAdapter`, the `subtitlePlacerAdapter` placement precedent)

- [x] **Task 3 — The leg itself (AC: #2, #5, #6, #7)**
  - [x] Rewrite the `case RouteNoTextSource:` arm: availability check → run ASR → map outcome to run row + media status; fall back to today's `recordSkip` path on nil/unavailable → `transcribeFallback` (process_item.go); per-item availability delegated to the service's richer resume-aware entry gate via `ErrTranscriptionDisabled` mapping (see Completion Notes)
  - [x] Confirm the ctx Budget threads through (`resolveBudget` already reads it) and `ErrBudgetExceeded` is classified as pause, not failure → confirmed; `pauseASRItem` closes the run row and leaves the media row untouched, sentinel propagates for `errors.Is` classification
  - [x] Gate on the existing pipeline-mode flag → port + sweep gate wired only inside main.go's `if cfg.SubtitlePipelineEnabled()` block; legacy mode never constructs the pipeline (flag semantics pinned by existing config tests)
  - [x] Decide + document the SSE stage emitted for this leg; do NOT touch the `transcription_*` event set → decision recorded in Completion Notes (D6 `extracting`/`complete`/`failed`/`skipped`, no new stage value; `transcription_*` untouched)

- [x] **Task 4 — Tests (AC: #9)**
  - [x] Seven cases from AC #9 in `process_item_test.go` / `worker_pool_test.go` → `process_item_asr_test.go` (a×2, b+e, c, d, f, plus ordinary-failure and episode-degrade) + `worker_pool_test.go` (g + two sweep-gate tests)
  - [x] Fake ASR port (Rule 11 test-fake precedent: `TranscriptionServiceInterface` in `transcription_handler.go`) → `fakeSpeechTranscriber` with service-writeback simulation hook

- [x] **Task 5 — Planning-doc reconciliation (AC: #1)**
  - [x] `vido-subtitle-pipeline-spec.md:21` — keep order A, but REPLACE the unverified reason ("線上搜尋命中率低") with the measured numbers + a link to the spike doc
  - [x] `vido-subtitle-pipeline-spec.md:159` — M2 checklist: mark the ASR-client item done (`ai.ASRProvider` + `ASR_BASE_URL`/`ASR_MODEL` shipped in 9R-9, re-verified 2026-08-06), rewrite the pipeline-wiring item as this story, and record that the search-reorder item is CANCELLED with the reason
  - [x] `subtitle-pipeline-architecture.md:37` — the pre-agreed premise says `Engine.Process` becomes the "final search fallback"; annotate that the automatic path no longer includes it (ruling 2026-08-06)
  - [x] `sprint-status.yaml:374` — the "Assrt token unobtainable" tombstone is factually wrong; update it (token obtainable as of 2026-08-06)
  - [x] `backlog-dialog-helper-verb-drift`'s ruling text contains an "M2 rewires to the unified 抽>ASR>搜尋 entry" note — annotate that the search half is cancelled

## Dev Notes

### The seam already exists — do not build a new one

`TranscriptionService.RunTranscription(ctx, mediaID, filePath, mediaDir, opts ...TranscriptionOption) error` (`transcription_service.go:249`) is the **synchronous, error-returning** entry added by 9R-16 precisely for callers that own the loop. It derives its timeout from the caller ctx (NOT the async `Background` detach) and `runPipeline` already reuses a ctx-attached Budget. **This is the seam. Do not call `StartTranscription` (async, job-managed) from the pipeline.**

Note its own availability gate is already resume-aware (`IsAvailable() || (translate && CanResumeTranslateOnly)`, relaxed by CR sub-2-2a M2) — the port's availability probe should not duplicate or contradict that logic.

### Port wiring pattern

`Pipeline` uses functional options with ports validated in `requireItemPorts()` (`process_item.go:441`). The five existing item-flow ports (`media`, `runs`, `router`, `placer`, `cache`) are **required** — a nil one produces a precise "which port is missing" error rather than a nil panic. **The ASR port is deliberately different: nil is a legal, supported state** (AC #4), so it must not join that required list. Make that asymmetry explicit in a comment or the next reader will "fix" it.

### `recordSkip` is still needed

`recordSkip` (`process_item.go:319`) writes run status `skipped` + `ErrorMessage = decision.Reason` + media status + `StageSkipped` progress. It stays the path for `RouteSkip` and for the nil/unavailable ASR fallback. Do not delete or repurpose it — add a sibling path.

### D2 vs D6 — the trap

The media-row status enum (D2) and the SSE stage set (D6) share vocabulary but are **DISTINCT stamped contracts**. project-context.md's sub-1-3 entry explicitly bans deduplicating them, and sub-2-2a's story text repeats the warning. This story touches D2 only. If you find yourself editing `PipelineStage` to "match" a status value, stop.

### What NOT to do

- **Do not add the online-search layer.** Ruled out 2026-08-06 with measurements; the spike doc is the audit trail.
- **Do not rewire `ManageSubtitleDialogV2`'s 生成字幕 button.** Party-mode ruling 2026-08-06 (`backlog-dialog-helper-verb-drift`): it stays on Route C ASR. That ruling's "when M2 lands" clause referred to the unified three-layer entry, which no longer exists.
- **Do not fix the Assrt provider here.** It is genuinely broken (search response parsing fails on every real payload — two struct type errors plus an undocumented second response schema), but it has been re-scoped to the **manual search dialog** and needs its own story. Filed as a lane-③ discovery below.

### Project Structure Notes

Backend-only story. Touched packages:

- `apps/api/internal/subtitle/` — `pipeline.go` (port + option), `process_item.go` (the leg), `worker_pool.go` (terminality)
- `apps/api/internal/models/movie.go` — contract stamp only, no value-set change, **no migration** (018 defines `subtitle_status TEXT` with no CHECK constraint — Winston verified during sub-2-2a)
- `apps/api/cmd/api/main.go` — wiring
- Frontend: **audit only** (AC #8). If the audit finds a real rendering gap, that is a lane-② or ③ discovery, not silent scope expansion.

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.` This story is backend-only; the frontend work is a read-only Rule 20 stale-mark audit that adds and modifies no components.

### References

- [Source: `_bmad-output/implementation-artifacts/spike-2026-08-06-pipeline-ordering-evidence.md`] — the ordering ruling and every number quoted above
- [Source: `apps/api/internal/subtitle/process_item.go:106`] — the `RouteNoTextSource` arm this story rewrites
- [Source: `apps/api/internal/subtitle/process_item.go:319`] — `recordSkip`, retained for `RouteSkip` + degraded path
- [Source: `apps/api/internal/subtitle/process_item.go:441`] — `requireItemPorts()` port-validation pattern
- [Source: `apps/api/internal/subtitle/worker_pool.go`] — `terminalPipelineVerdict()`
- [Source: `apps/api/internal/subtitle/router.go`] — `verdictWithoutTrack()`; its comment "what P2's ASR can later recover" is the design intent this story finally delivers
- [Source: `apps/api/internal/services/transcription_service.go:249`] — `RunTranscription`, the sync seam
- [Source: `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md:37`] — Option B strangler-wrapper premise (needs the annotation in Task 5)
- [Source: `project-context.md`] — Rule 11 (narrow interfaces), Rule 19 (package direction), Rule 20 (contract bump + stale-mark), Rule 24 (Discovery Triage)

## Senior Developer Review (AI)

**Date:** 2026-08-07 · **Outcome:** Approve (after same-session fixes) · **Findings:** 0 High / 2 Medium / 3 Low

**Mandatory checks:**

- 🔒 Rule 7 Wire Format: **PASS** (0 new error-code constants in scope; grep over all in-review Go files returned none — existing `SUBTITLE_`/`TRANSCRIPTION_` sentinels reused, untouched)
- 🔒 Rule 20 Contract Bump: **PASS** (1 bump `[@contract-v2→v3]`; backtick-tolerant downstream-ack grep → consumers sub-1-5b / sub-1-7b / sub-2-2a / sub-2-2b all `done` = frozen; 0 not-done consumers, scan recorded in Change Log row)
- 🔒 Rule 25 Mega-line: **N/A** (project-context.md untouched)
- Git vs Story File List: **0 discrepancies** (13 = 13)
- AC validation: all 9 IMPLEMENTED (AC #2/#9a as movie-scoped, annotated under M1 below); Task audit: all [x] verified with file:line evidence

**Action Items:**

- [x] **[M1]** AC #2 / #9(a) were implemented MOVIE-only but the AC text read as all-media — contract-taxonomy mismatch (19-8 CR re-stamp precedent). **Fixed:** movie-scope narrowing annotated inline on AC #2 + AC #9(a) with the `backlog-episode-asr-fallback` pointer.
- [x] **[M2]** `transcribeFallback` provenance mis-attribution: under `Force`, a pre-existing acceptable zh sidecar surviving an `untranslated` outcome was claimed as this run's `OutputPath`/`CueCount` (run says `completed`+sidecar while the media row says `untranslated`). **Fixed:** pre-run `os.Stat` snapshot + `sidecarWrittenSince` guard (`process_item.go`) — attribute only a new or rewritten file; two regression subtests added (stale-not-attributed / overwritten-attributed).
- [x] **[L1]** `models/movie.go` `AllSubtitleStatuses` doc comment still cited "[@contract-v1] stamp" (stale since v2, doubly stale at v3). **Fixed:** reworded version-neutral so it cannot re-stale.
- [x] **[L2]** `SpeechTranscriber.Available()` has no interface-typed consumer (the pool takes a plain `func() bool` via `WithASRAvailability`; main.go passes the adapter's bound method). **No change:** the probe is AC #3-mandated, the port doc already declares it sweep-side, and collapsing it would couple the pool to the port for zero gain. Recorded as accepted.
- [x] **[L3]** A mid-ASR budget ceiling leaves the media row at `extracting` until the next sweep (transient phantom in-flight state). **No change:** deliberate, documented trade-off in `transcribeFallback`/`pauseASRItem` — the alternative (`not_searched` revert) destroys the `untranslated` resume marker (sub-2-2a headline-bug class); self-heals on the next sweep.

**Post-fix verification:** `go build` ✅ · subtitle + models suites ✅ (incl. 2 new CR subtests) · full `nx test api` + `nx test web` ✅ · gofmt/vet/prettier clean.

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — dev-story workflow, 2026-08-07

### Debug Log References

- RED confirmed before implementation: `WithSpeechTranscriber`/`WithASRAvailability` undefined (build fail) + `TestSubtitleStatus_IsTerminal` failing on `no_text_source` — then GREEN after the leg landed.
- Full regression: `pnpm nx test api` ✅ · `pnpm nx test web` ✅ (228 files / 2547 tests) · vitest cleanup verified "No test processes found" · `go vet` clean · `gofmt -l` clean on every touched file (the 10 listed files are pre-existing unformatted files this story did not touch) · prettier clean on all touched docs.

### Completion Notes List

- **🔗 AC Drift: FOUND (by design — this story IS the drift)** — the story's whole point is changing sub-1-2 AC #2's `no_text_source` terminality. Grep `no_text_source|terminalPipelineVerdict` across `_bmad-output/implementation-artifacts/*.md`: prior-story hits are sub-1-2 (defines the value), sub-1-4 (routes to it), sub-1-5b (writes it), sub-1-6 (sweep filters it), sub-1-7a/7b + sub-2-2b (render it). The DRIFT is confined to terminality (value set unchanged); it is stamped `[@contract-v2→v3]` and Change-Logged below. All other hits are REUSE.
- **📎 Contract Stamps: FOUND (1 bump produced, 2 upstream acks recorded)** — this story bumps `models/movie.go` `[@contract-v2→v3]` (sub-1-2 AC #2 lineage). Bump-side stale-mark grep (`confirmed against \`?\[@contract-v` + sub-1-2 AC #2): consumers sub-1-5b, sub-1-7b, sub-2-2a, sub-2-2b — **all `done` in sprint-status.yaml = FROZEN, no stale-marks owed; no not-done downstream consumers**. Acks this story records as consumer: confirmed against [@contract-v1] sub-1-5b AC #1 (`ProcessItem`/`MediaRef`/`ProcessOutcome` — signature and outcome shape untouched; a new verdict-arm behavior behind an optional port, no bump) · confirmed against [@contract-v2] 9R-16 AC 12 (as amended by sub-2-2a — the service's found/untranslated writeback IS the media-status truth the leg relies on) · `PipelineStage` (sub-1-3 AC #1 [@contract-v1]) consumed with ZERO new values.
- **🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)**. Frontend surfaces were audited read-only (AC #8, next bullet); no component added or modified.
- **AC #8 downstream consumer audit (Rule 20 stale-mark, code side)** — every consumer of the D2 status contract checked against the terminality change:
  | Consumer | Verdict |
  |---|---|
  | `apps/web/src/utils/libraryStatus.ts:155` | **verified-no-change** — value set unchanged; 無字幕源 label + "only P2 ASR can change it" comment become literally true |
  | `apps/web/src/components/media/EpisodeList.tsx:79` | **verified-no-change** — label/icon unchanged; for EPISODES `no_text_source` remains effectively terminal until backlog-episode-asr-fallback, so its "Terminal" comment stays truthful in its scope |
  | `apps/web/src/routes/library.tsx` subtitleStatus filter | **verified-no-change** — pass-through string param over an unchanged value set |
  | `apps/api/internal/subtitle/worker_pool.go` enqueue sweep | **changed by this story** — `terminalPipelineVerdict` drops `no_text_source`; movie loop re-enumerates it only behind `WithASRAvailability`; episode loop keeps filtering it |
  | `models.SubtitleStatus.IsTerminal()` | **changed by this story** — `no_text_source` removed (v3 semantic); zero production callers, test updated |
- **Availability-probe deviation (AC #3, documented)**: the leg does NOT call `Available()` per item. The service's entry gate is RICHER — `IsAvailable() || (translate && CanResumeTranslateOnly)` (CR sub-2-2a M2) — and pre-probing with the plain capability fact would have blocked ASR-less translate-only resumes and clobbered their `untranslated` marker with `no_text_source`. The leg calls `Transcribe` and maps `services.ErrTranscriptionDisabled` → today's degrade path; `Available()` exists for the SWEEP gate (worker pool re-enumeration), where no per-media nuance applies. This honors the Dev Notes instruction that the probe "should not duplicate or contradict" the service gate.
- **SSE decision (AC #7)**: for an ASR-fallback item the frontend observes the **`subtitle_progress` (D6 `PipelineStage`) stream**, same as every other automatic-pipeline item — the leg emits the EXISTING `extracting` stage (ASR's first phase is literally an ffmpeg audio extraction, and the media row already reads `extracting`), then `complete`/`failed`/`skipped` via the existing paths. **No new D6 value** — adding one would bump the stamped sub-1-3 contract this story deliberately leaves untouched. The service's `transcription_*` events still fire during the leg; they are the manual Route-C dialog's job-keyed contract (DISTINCT stream, dedup banned per project-context.md sub-1-3) and no automatic-pipeline UI subscribes to them.
- **Budget pause semantics (AC #6)**: `RunTranscription` → `resolveBudget` reuses a ctx-attached `ai.Budget` and creates the per-run envelope otherwise, so ASR + LLM always share one ceiling — nothing extra to wire. On `ai.ErrBudgetExceeded` the leg runs `pauseASRItem`: run row closes `failed` with the `AI_BUDGET_EXCEEDED` message (the run-status enum has no `paused` member and adding one would be an unscoped contract change — the sentinel in the message + `errors.Is` classification carry the pause semantics), the media row is **left untouched** (a mid-translate ceiling means the service already wrote `untranslated` + EN path = the resume marker; reverting to `not_searched` would re-pay the whole ASR — sub-2-2a's headline bug class), and the error propagates wrapping the sentinel. A mid-ASR ceiling leaves the row at `extracting`, which the next sweep re-enumerates — transiently dishonest in the UI, self-healing, and preferable to destroying resume markers; trade-off documented in code.
- **Movie-only scope (discovered at implementation, lane ③)**: `TranscriptionService`'s status writer/resume reader are movie-repository-bound (`SetSubtitleStatusWriter`/`SetSubtitleStateReader` take `repos.Movies`; `UpdateSubtitleStatus` on an episode id errors at 0 rows), so an episode through the leg would pay full ASR+LLM then fail at writeback. The leg degrades non-movie refs to `recordSkip(no_text_source)` and the episode sweep loop keeps filtering `no_text_source` — episodes keep exact pre-M2 behavior. Filed as `backlog-episode-asr-fallback` (see Discovery Triage).
- **AC #4 nuance beyond the story text**: making `no_text_source` non-terminal in the sweep UNCONDITIONALLY would have violated AC #4 — an ASR-less deployment would re-probe all 142+ items every scan, appending a run row + re-broadcasting 已略過 each time (the exact CR H1 churn `terminalPipelineVerdict` was built to stop). Hence the sweep-side `WithASRAvailability` gate: `no_text_source` movies re-enter enumeration only while the ASR service is live; with no key the sweep is byte-identical to pre-M2 (pinned by test).
- **Bonus honesty fix en route**: an `untranslated` movie swept in pipeline mode used to dead-end at `recordSkip(no_text_source)` (clobbering its resume marker); with a live ASR port it now flows into `RunTranscription`'s translate-only resume and completes without re-paying ASR.
- **🎨 UX Verification: SKIPPED — no UI changes in this story** (backend-only; frontend surfaces audited read-only under AC #8).

### Change Log

| Date | Change |
|---|---|
| 2026-08-07 | [@contract-v2→v3] AC #1 (models/movie.go SubtitleStatus, sub-1-2 AC #2 lineage): what changed — `no_text_source` terminality flips from PERMANENT verdict to INTERMEDIATE (ASR-recoverable); value set unchanged at 10; `IsTerminal()` drops it; `terminalPipelineVerdict()` narrows to `skipped`. What breaks downstream — any consumer treating `no_text_source` as "will never change again" (sweep filters, badge steady-state assumptions, run-row churn assumptions); bump-side grep found consumers sub-1-5b / sub-1-7b / sub-2-2a / sub-2-2b, all `done` = frozen, **no not-done downstream consumers**. |
| 2026-08-07 | Task 1–4 implementation: `subtitle.SpeechTranscriber` optional port + `WithSpeechTranscriber`; `transcribeFallback`/`pauseASRItem`/`sidecarCueCount` in process_item.go; `WithASRAvailability` sweep gate + movie/episode loop split in worker_pool.go; `pipelineASRAdapter` (cmd/api/asr_adapter.go) wired in main.go pipeline-mode block; 9 new tests + 2 amended (RED→GREEN), full api+web regression green. |
| 2026-08-07 | Task 5 planning-doc reconciliation: spec §1 ordering rationale replaced with spike measurements + link; spec §8 M2 checklist rewritten (ASR client done, wiring = this story, search-reorder CANCELLED); architecture Pre-agreed Premise Option B amended (automatic path no longer includes `Engine.Process`); sprint-status Epic-9R Assrt tombstone corrected (token obtainable 2026-08-06); backlog-dialog-helper-verb-drift annotated (unified three-layer entry no longer exists). |
| 2026-08-07 | Rule 24 lane ③ filed at implementation: `backlog-episode-asr-fallback` (episodes out of ASR reach — movie-bound TranscriptionService writer/reader). |
| 2026-08-07 | Senior Developer Review (AI): 0H/2M/3L — M1 AC #2/#9(a) movie-scope annotation; M2 `sidecarWrittenSince` provenance guard + 2 regression subtests; L1 version-neutral comment fix; L2/L3 accepted with rationale. Status review → done. |

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at implementation time (2026-08-07):
    - **③ backlog-with-carry-forward-link** — `backlog-episode-asr-fallback`: `TranscriptionService`'s status writer and resume reader are movie-repository-bound (`SetSubtitleStatusWriter`/`SetSubtitleStateReader` take `repos.Movies`; `MovieRepository.UpdateSubtitleStatus` errors on 0 rows for an episode id), so an episode through the ASR leg would pay full ASR+LLM and then fail at the writeback. This story scopes the leg to movies: `transcribeFallback` degrades non-movie refs to `recordSkip(no_text_source)` and the episode sweep loop keeps filtering `no_text_source` (pre-M2 behavior). Both code sites carry comments pointing at the backlog entry. Picked up = media-type-dispatched writer/reader on the service, then remove the two guards. Note: the spike's 68.3% fall-through sample includes series — initial M2 recovers only its movie share.
  - **YES** — filed at authoring time:
    - **③ backlog-with-carry-forward-link** — `backlog-assrt-search-response-parsing`: the Assrt provider's search-response structs cannot unmarshal ANY real payload (`lang` declared `string` but is an object; `subs` is `{}` not `[]` when empty), and the API returns a **second, undocumented schema** (`m_langn`/`m_lang`/`sub_name`/`fileid`) that the struct does not model at all. `engine.go:249` only `slog.Warn`s, so Assrt has been contributing zero results silently. Also missing: format normalisation on the download path (`convertIfNeeded` does OpenCC only; an ASS download is written verbatim into a `.zh-Hant.srt`). Re-scoped by the 2026-08-06 ruling from "pipeline layer" to "manual search dialog data source". Compatibility map available in `dyphire/mpv-sub-assrt` (`sub-assrt.lua:514`). Evidence: spike doc §4.
    - **③ backlog-with-carry-forward-link** — `backlog-compute-aware-asr-default`: the 9R-S2 benchmark finally ran (kit had four mutually-masking bugs, fixed 2026-08-06) but on an i5-12400 / 31 GiB NAS, which is far above the J4125/N5095 class the spec's tiering assumes. The measured numbers cannot set a product default. Either re-run on representative hardware or promote spec §6's compute-aware selection.

### File List

- `apps/api/internal/models/movie.go` — [@contract-v2→v3] stamp bump; `no_text_source`/`skipped` doc comments; `IsTerminal()` drops `no_text_source`
- `apps/api/internal/models/movie_test.go` — `TestSubtitleStatus_IsTerminal` updated to the v3 semantic
- `apps/api/internal/subtitle/pipeline.go` — `SpeechTranscriber` port + `asr` field + `WithSpeechTranscriber` option
- `apps/api/internal/subtitle/process_item.go` — `RouteNoTextSource` arm → `transcribeFallback`; `pauseASRItem`; `sidecarCueCount`; `requireItemPorts` optional-port note; imports (errors/filepath/ai/services)
- `apps/api/internal/subtitle/process_item_asr_test.go` — NEW: fake ASR port + AC #9 cases (a×2, b+e, c, d, f, ordinary-failure, episode-degrade)
- `apps/api/internal/subtitle/worker_pool.go` — `terminalPipelineVerdict` narrows to `skipped`; `asrAvailable` field + `WithASRAvailability`; movie-loop ASR gate; episode-loop `no_text_source` filter
- `apps/api/internal/subtitle/worker_pool_test.go` — AC #9 (g) terminality test + two sweep-gate tests; CR H1 test comment amended
- `apps/api/cmd/api/asr_adapter.go` — NEW: `pipelineASRAdapter` over `TranscriptionService.RunTranscription` (sync seam, `WithTranslation`)
- `apps/api/cmd/api/main.go` — pipeline-mode wiring: `WithSpeechTranscriber(pipelineASR)` + `WithASRAvailability(pipelineASR.Available)`
- `_bmad-output/planning-artifacts/vido-subtitle-pipeline-spec.md` — §1 ordering rationale (measured), §8 M2 checklist rewrite (Task 5)
- `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md` — Pre-agreed Premise Option B amendment (Task 5)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — sub-3-1 status transitions; Epic-9R Assrt tombstone correction; backlog-dialog-helper-verb-drift annotation; NEW `backlog-episode-asr-fallback` entry
- `_bmad-output/implementation-artifacts/sub-3-1-asr-fallback-leg.md` — this story file (checkboxes, Dev Agent Record, File List, Change Log, Status)
