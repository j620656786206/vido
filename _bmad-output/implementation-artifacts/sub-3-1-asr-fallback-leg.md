# Story 3.1: Wire the ASR fallback leg into the automatic pipeline

Status: ready-for-dev

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

3. **ASR is a narrow injected port, not a concrete dependency.** A new interface on the `subtitle` package side (Rule 11) exposes only what the leg needs — the existing synchronous `TranscriptionService.RunTranscription(ctx, mediaID, filePath, mediaDir, opts...) error` seam (added by 9R-16) plus an availability probe. It is wired via a functional option like every other item-flow port and validated in `requireItemPorts()`. Rule 19: `subtitle → services` is a legal direction (engine.go precedent).

4. **ASR unavailable is a graceful degradation, not a regression.** When the port is nil (translate-only Pipeline construction, sub-1-5a) OR the service reports unavailable (no ASR key / no FFmpeg), the item records `no_text_source` exactly as it does today. A deployment without an ASR key must behave identically to before this story.

5. **The feature flag covers the new leg.** `VIDO_SUBTITLE_PIPELINE_MODE=legacy` produces byte-identical behaviour to today. Only `pipeline` mode gets the ASR leg.

6. **One budget, not two.** The ASR call rides the ctx-attached `ai.Budget` (9R-11 / 9R-16 `resolveBudget`) so ASR + LLM share the run ceiling. `ai.ErrBudgetExceeded` propagates out of the leg as a pause, not a failure, and does NOT get recorded as `no_text_source`.

7. **SSE vocabulary is decided and documented.** D2 pipeline stages (`PipelineStage`) and the transcription service's `transcription_*` events are DISTINCT contracts sharing vocabulary — they must NOT be deduplicated (project-context.md, sub-1-3 entry). The story records which stream the frontend observes for an ASR-fallback item and why, and emits the D2 stage consistently with that decision.

8. **Downstream stale-mark completed (Rule 20).** Every consumer of the D2 status contract is audited against the terminality change and the result recorded: `apps/web/src/utils/libraryStatus.ts`, `apps/web/src/components/media/EpisodeList.tsx`, the library status filter, and `apps/api/internal/subtitle/worker_pool.go`'s enqueue sweep. Consumers that need no change are listed as verified-no-change, not silently skipped.

9. **Regression + new coverage.** Tests pin: (a) `RouteNoTextSource` + available ASR → transcription invoked, media status is the ASR outcome; (b) `RouteNoTextSource` + nil port → `no_text_source` recorded, ASR never called; (c) `RouteNoTextSource` + unavailable service → same as (b); (d) `RouteSkip` unchanged; (e) legacy flag mode unchanged; (f) budget-exceeded propagates as pause; (g) `terminalPipelineVerdict` no longer terminal for `no_text_source` but still terminal for `skipped`.

## Tasks / Subtasks

- [ ] **Task 1 — Contract bump + terminality change (AC: #1, #8)**
  - [ ] Bump `models/movie.go`'s `[@contract-v2]` stamp to v3 with an inline note that the CHANGE is `no_text_source` terminality, not the value set
  - [ ] `worker_pool.go`: `terminalPipelineVerdict()` drops `no_text_source`, keeps `skipped`
  - [ ] Run the mandatory Rule 20 downstream stale-mark grep; record every consumer and its verdict (changed / verified-no-change) in Dev Notes

- [ ] **Task 2 — ASR port (AC: #3, #4)**
  - [ ] Define the narrow interface in the `subtitle` package (name it for what the pipeline needs, not after the concrete service)
  - [ ] Add the functional option + `requireItemPorts()` handling — note the port is OPTIONAL (nil = no ASR leg), unlike the five existing required ports, so it must NOT be added to the missing-ports error
  - [ ] Adapter on the `services` side (or in `cmd/api/main.go` wiring) over `TranscriptionService`

- [ ] **Task 3 — The leg itself (AC: #2, #5, #6, #7)**
  - [ ] Rewrite the `case RouteNoTextSource:` arm: availability check → run ASR → map outcome to run row + media status; fall back to today's `recordSkip` path on nil/unavailable
  - [ ] Confirm the ctx Budget threads through (`resolveBudget` already reads it) and `ErrBudgetExceeded` is classified as pause, not failure
  - [ ] Gate on the existing pipeline-mode flag
  - [ ] Decide + document the SSE stage emitted for this leg; do NOT touch the `transcription_*` event set

- [ ] **Task 4 — Tests (AC: #9)**
  - [ ] Seven cases from AC #9 in `process_item_test.go` / `worker_pool_test.go`
  - [ ] Fake ASR port (Rule 11 test-fake precedent: `TranscriptionServiceInterface` in `transcription_handler.go`)

- [ ] **Task 5 — Planning-doc reconciliation (AC: #1)**
  - [ ] `vido-subtitle-pipeline-spec.md:21` — keep order A, but REPLACE the unverified reason ("線上搜尋命中率低") with the measured numbers + a link to the spike doc
  - [ ] `vido-subtitle-pipeline-spec.md:159` — M2 checklist: mark the ASR-client item done (`ai.ASRProvider` + `ASR_BASE_URL`/`ASR_MODEL` shipped in 9R-9, re-verified 2026-08-06), rewrite the pipeline-wiring item as this story, and record that the search-reorder item is CANCELLED with the reason
  - [ ] `subtitle-pipeline-architecture.md:37` — the pre-agreed premise says `Engine.Process` becomes the "final search fallback"; annotate that the automatic path no longer includes it (ruling 2026-08-06)
  - [ ] `sprint-status.yaml:374` — the "Assrt token unobtainable" tombstone is factually wrong; update it (token obtainable as of 2026-08-06)
  - [ ] `backlog-dialog-helper-verb-drift`'s ruling text contains an "M2 rewires to the unified 抽>ASR>搜尋 entry" note — annotate that the search half is cancelled

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

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time:
    - **③ backlog-with-carry-forward-link** — `backlog-assrt-search-response-parsing`: the Assrt provider's search-response structs cannot unmarshal ANY real payload (`lang` declared `string` but is an object; `subs` is `{}` not `[]` when empty), and the API returns a **second, undocumented schema** (`m_langn`/`m_lang`/`sub_name`/`fileid`) that the struct does not model at all. `engine.go:249` only `slog.Warn`s, so Assrt has been contributing zero results silently. Also missing: format normalisation on the download path (`convertIfNeeded` does OpenCC only; an ASS download is written verbatim into a `.zh-Hant.srt`). Re-scoped by the 2026-08-06 ruling from "pipeline layer" to "manual search dialog data source". Compatibility map available in `dyphire/mpv-sub-assrt` (`sub-assrt.lua:514`). Evidence: spike doc §4.
    - **③ backlog-with-carry-forward-link** — `backlog-compute-aware-asr-default`: the 9R-S2 benchmark finally ran (kit had four mutually-masking bugs, fixed 2026-08-06) but on an i5-12400 / 31 GiB NAS, which is far above the J4125/N5095 class the spec's tiering assumes. The measured numbers cannot set a product default. Either re-run on representative hardware or promote spec §6's compute-aware selection.

### File List
