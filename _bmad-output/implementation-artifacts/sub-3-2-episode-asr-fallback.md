# Story 3.2: Episode ASR fallback — media-type-aware TranscriptionService

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner with TV libraries,
I want the automatic subtitle pipeline's ASR fallback to also serve episodes,
so that the series share of the 68.3% no-text-source population gets subtitles instead of being silently filtered at the sweep — sub-3-1 recovered only movies.

## Context — why this story exists

sub-3-1 (PR #208, merged 2026-08-07) wired the `no_text_source → ASR` leg but discovered at implementation that **`TranscriptionService`'s status writer and resume reader are movie-repository-bound**: `SetSubtitleStatusWriter`/`SetSubtitleStateReader` both take `repos.Movies`, and `MovieRepository.UpdateSubtitleStatus` errors at 0 rows for an episode id. An episode through the leg would have paid full ASR+LLM and then failed at the writeback. sub-3-1 therefore scoped the leg to movies with **two explicit guards, both commented with a pointer to `backlog-episode-asr-fallback`** (this story's origin entry):

1. `process_item.go` `transcribeFallback` — non-movie refs degrade to `recordSkip(no_text_source)`
2. `worker_pool.go` episode sweep loop — `no_text_source` episodes unconditionally filtered

This story makes the service media-type aware, then removes both guards. The infrastructure is otherwise ready: episodes already carry `subtitle_status`/`subtitle_path`/`subtitle_language` (Story 12-2, migration mirror of the movie columns), `EpisodeRepository.UpdateEpisodeSubtitleStatus` + `FindByID` exist, the worker pool already enumerates episodes (`EpisodeGenerationFinder`), and the pipeline's own `MediaStore` already dispatches movie/series/episode.

## Acceptance Criteria

1. **`TranscriptionService` gains media-type awareness via an ADDITIVE option.** A new `WithMediaType(mediaType string)` `TranscriptionOption` (vocabulary: `models.SubtitleRunMediaMovie` / `models.SubtitleRunMediaEpisode` — the internal movie|series|episode set, sub-1-2 AC #1, NEVER the TMDB movie|tv pair). Default is **movie**, so every existing caller (`transcription_handler.go`, `generation_batch.go`) is byte-unchanged with zero call-site edits. `RunTranscription`/`StartTranscription` signatures are untouched — the 9R-16 sync seam is consumed, not bumped.

2. **Episode outcomes write through the episode repository.** With `WithMediaType(episode)`, `translateAndPersist`'s three writeback sites (zh-success → `found`+zh path, translate-expected-but-absent → `untranslated`+EN path, budget-pause pre-write) dispatch to `EpisodeRepository.UpdateEpisodeSubtitleStatus(ctx, id, status, path, language)`. Note the shape difference from the movie writer: **different method name, no `score` param, and no `subtitle_search_score`/`subtitle_last_searched` columns on episodes** — the dispatch must absorb this, not change the movie path. Movie writes stay byte-identical.

3. **Episode translate-only resume works.** `resumeSource`/`CanResumeTranslateOnly`/`tryTranslateOnlyResume` dispatch on the media type: an `untranslated` episode row whose recorded EN SRT is still on disk resumes translate-only (no re-ASR), exactly mirroring the movie behavior (sub-2-2a AC #3 semantics; status-bound — a bare on-disk `.srt` must never trigger a resume). The current `SubtitleStateReader` returns a concrete `*models.Movie`; the episode read needs its own narrow interface or a media-neutral internal shape — do NOT widen the existing movie interface (Rule 11).

4. **The `SpeechTranscriber` port carries the media type.** `Transcribe(ctx, mediaID, filePath, mediaDir string)` grows a media-type argument (or takes `MediaRef`), and `pipelineASRAdapter` maps it to `WithMediaType`. The port is a sub-3-1-fresh, unstamped, in-repo-only contract (consumers: `Pipeline` + `cmd/api/asr_adapter.go` + the test fake) — record the change in the Change Log; no Rule 20 bump obligations exist.

5. **`transcribeFallback`'s movie-only guard flips to movie|episode.** Episode refs run the leg; `series` (and any unknown media type) still degrades to `recordSkip(no_text_source)` — a series row is a container, not a media file. The guard comment pointing at `backlog-episode-asr-fallback` is updated to the series-only rationale.

6. **The episode sweep loop joins the ASR gate.** `worker_pool.go`'s episode loop drops its unconditional `no_text_source` filter and uses the same `asrRecoverable()` gate as the movie loop (unify — do not duplicate the condition). AC-#4-of-sub-3-1 invariant holds for episodes too: an ASR-less deployment's sweep is byte-identical to today (zero re-probes, zero appended run rows).

7. **Partial wiring fails loudly, not silently.** If media type is `episode` but the episode writer/reader is not wired (nil), the writeback/resume-check fails with a precise "episode subtitle writer not wired" error — never the misleading movie-repo "movie with id X not found". `cmd/api/main.go` wires `repos.Episodes` unconditionally (both transcription-service construction branches, mirroring the movie pair at main.go:531-534).

8. **Known limitation recorded, not solved:** `loadGlossary` uses `LookupByMedia(mediaID)` and `show_glossary` is keyed by the movie/show media id — an episode-id lookup returns empty, so episode ASR translations run glossary-less (fail-soft already handles this: nil pairs → plain translation). Record in Dev Agent Record → Completion Notes as a known limitation with a pointer to spec §6.5 (名詞庫 auto-harvest, 順序無關); resolving show-key glossary resolution is explicitly OUT of scope.

9. **Regression + new coverage.** Tests pin: (a) `RouteNoTextSource` + episode ref + available ASR → `Transcribe` invoked with the episode media type, run row completed, media status = ASR outcome (via the fake's service-writeback simulation); (b) sweep: `no_text_source` episodes enqueued when `asrRecoverable()`, filtered when not (byte-identical pre-M2 behavior without ASR); (c) service: `WithMediaType(episode)` writes found+zh / untranslated+en through the EPISODE writer, movie writer untouched; (d) episode translate-only resume: `untranslated` episode + EN SRT on disk → no ASR call, translation runs; (e) default media type is movie — omitted option hits the movie writer/reader (pins backward compat for handler + generation batch); (f) episode media type + nil episode writer → precise wiring error; (g) series ref still degrades to `no_text_source`.

## Tasks / Subtasks

- [x] **Task 1 — Service media-type plumbing (AC: #1, #2, #3, #7)**
  - [x] `WithMediaType` option + `transcriptionConfig.mediaType` (default `models.SubtitleRunMediaMovie`) → `newTranscriptionConfig` centralizes the default
  - [x] Narrow episode interfaces in `services` (Rule 11; do NOT widen the movie ones): `EpisodeSubtitleStatusWriter` + `EpisodeSubtitleStateReader` — `*repository.EpisodeRepository` satisfies both structurally
  - [x] `SetEpisodeSubtitleStatusWriter` / `SetEpisodeSubtitleStateReader` setters (mirror the movie pair)
  - [x] Internal dispatch: `writeSubtitleStatus` (three `translateAndPersist` sites incl. the budget-pause pre-write) + `loadSubtitleRowState` (behind `resumeSource`); `mediaType` threaded through `runPipeline` → `translateAndPersist`/`tryTranslateOnlyResume`; public `CanResumeTranslateOnly` stays movie-scoped (handler surface), entry gates use the internal `canResumeTranslateOnly`
  - [x] Precise nil-wiring error for the episode path (AC #7): "episode subtitle writer not wired — cannot persist {status} for episode {id}"

- [x] **Task 2 — Port + adapter + the leg (AC: #4, #5)**
  - [x] `SpeechTranscriber.Transcribe` now takes `MediaRef`; `pipelineASRAdapter` maps `ref.MediaType` → `WithMediaType`; test fake updated
  - [x] `transcribeFallback`: guard flips to movie|episode (`transcribable` predicate); series/unknown still degrades; guard comment rewritten with the series-only rationale

- [x] **Task 3 — Sweep gate unification (AC: #6)**
  - [x] `sweepEligible()` extracted; BOTH loops use it (movie/episode conditions can no longer drift); episode-filter comment removed
  - [x] Sweep tests amended: ReEnumerates now asserts episodes enqueued (+ ep-skipped never), Unavailable covers both loops

- [x] **Task 4 — Wiring (AC: #7)**
  - [x] `cmd/api/main.go`: `SetEpisodeSubtitleStatusWriter(repos.Episodes)` + `SetEpisodeSubtitleStateReader(repos.Episodes)` unconditionally, beside the movie pair (single wiring site after both construction branches)

- [x] **Task 5 — Tests + reconciliation (AC: #8, #9)**
  - [x] The seven AC #9 cases: (a)(g) in `process_item_asr_test.go` (episode-runs replaces episode-degrade; series-degrade added), (b) in `worker_pool_test.go` sweep pair, (c)(d)(e)(f) in NEW `transcription_episode_test.go` (episode writer found/untranslated, movie-writer-untouched cross-assertions, precise nil-writer error, episode resume, movie default, episode nil-reader no-cross-resume)
  - [x] AC #8 glossary limitation recorded in Completion Notes
  - [x] `sprint-status.yaml`: promoted-tombstone + bidirectional link verified
  - [x] `vido-subtitle-pipeline-spec.md` §8 M2 checklist line updated (sub-3-2 episode leg)
  - [x] Full regression gate: `pnpm nx test api` ✅ + `pnpm nx test web` ✅ (228 files / 2547 tests)

## Dev Notes

### The dispatch goes INSIDE the service, not at the call sites

`RunTranscription(ctx, mediaID, filePath, mediaDir, opts...)` is the 9R-16 seam consumed by three callers (manual handler, generation batch, pipeline adapter). Only the pipeline adapter knows about episodes today. Making the media type an OPTION with a movie default means zero existing call-site edits and no signature change — the exact additive pattern `WithTranslation()` set. Do not add a mediaType parameter to `RunTranscription` and do not make the writer "try movies then episodes" (two queries + cross-table id ambiguity).

### Shape differences to absorb (verified 2026-08-07)

- `SubtitleStatusWriter.UpdateSubtitleStatus(ctx, id, status, path, language, score)` (movie) vs `EpisodeRepository.UpdateEpisodeSubtitleStatus(ctx, id, status, path, language)` — different name, no score, and the episode SQL touches only `subtitle_status/path/language` (episodes have no search-score columns). Score is always 0 in every transcription writeback today, so nothing is lost — but do not "unify" the repos; adapt in the service dispatch.
- `SubtitleStateReader.FindByID` returns `*models.Movie` concretely. `EpisodeRepository.FindByID` returns `*models.Episode`. `resumeSource` only reads `SubtitleStatus` + `SubtitlePath` — both models carry them (`episode.go:31-33`). A separate narrow episode-reader interface (or an internal `(status, path, ok)` normalization) keeps Rule 11 intact.
- The single-flight `inProgress` map keys on bare mediaID — episode UUIDs collide with nothing; no change needed.

### sub-3-1 left you breadcrumbs

Both guards carry comments naming `backlog-episode-asr-fallback`:

- `process_item.go` `transcribeFallback`: `if p.asr == nil || ref.MediaType != models.SubtitleRunMediaMovie` — the second clause becomes "movie or episode" (series still out).
- `worker_pool.go` episode loop: the `if e.SubtitleStatus == models.SubtitleStatusNoTextSource { continue }` block — replace with the movie loop's gated condition; extract a helper so the two loops cannot drift.

Also update `EpisodeList.tsx:77-78`'s "Terminal: … only P2 ASR can change this one" comment IF you touch that file — otherwise leave it: it remains truthful (ASR now does change it) and this story is otherwise backend-only. Frontend is audit-only; a rendering gap is a lane-②/③ discovery, not silent scope expansion.

### Contract posture (Rule 20)

- D2 media-row status contract: **no bump.** `[@contract-v3]` (sub-3-1) already declares `no_text_source` INTERMEDIATE with no media-type carve-out — the episode filter was an implementation guard, not a contract clause. Record: `confirmed against [@contract-v3] sub-1-2 AC #2 (as bumped by sub-3-1)`.
- 9R-16 `RunTranscription` seam: consumed additively (new option), signatures untouched — record the ack, no bump.
- `SpeechTranscriber`: unstamped sub-3-1 port with only in-repo consumers — signature change is legal; record in Change Log.
- `terminalPipelineVerdict`: unchanged by this story (already skipped-only).

### What NOT to do

- **Do not touch the manual dialog or `/movies/{id}/transcribe` endpoint.** The manual Route-C surface stays movie-only; episode manual generation is a separate product decision (would need its own endpoint/UI — not this story).
- **Do not make the 9R-16 generation batch episode-aware.** `GenerationBatchProcessor` enumerates movies by design; the worker-pool sweep is the episode path.
- **Do not solve show-key glossary resolution** (AC #8 records the limitation). No `show_glossary` schema or lookup changes.
- **Do not touch the `transcription_*` SSE event set or D6 `PipelineStage`** — same posture as sub-3-1 (D2/D6 are distinct stamped contracts; dedup banned).

### Project Structure Notes

Backend-only story. Touched packages:

- `apps/api/internal/services/transcription_service.go` — option, interfaces, setters, dispatch
- `apps/api/internal/subtitle/` — `pipeline.go` (port signature), `process_item.go` (guard flip), `worker_pool.go` (episode loop gate)
- `apps/api/cmd/api/asr_adapter.go` + `main.go` — adapter + wiring
- Frontend: none (comment-level audit only)

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.` Backend-only; no components added or modified.

### References

- [Source: `_bmad-output/implementation-artifacts/sub-3-1-asr-fallback-leg.md`] — the movie-scoped leg, both guards, CR findings (M2 `sidecarWrittenSince` guard applies unchanged to episodes)
- [Source: `apps/api/internal/services/transcription_service.go:23-37`] — the two movie-bound interfaces this story dispatches around
- [Source: `apps/api/internal/services/transcription_service.go:452-506`] — `resumeSource`/`CanResumeTranslateOnly`/`tryTranslateOnlyResume`
- [Source: `apps/api/internal/services/transcription_service.go:524-590`] — `translateAndPersist`'s three writeback sites
- [Source: `apps/api/internal/repository/episode_repository.go:107,309`] — `FindByID` + `UpdateEpisodeSubtitleStatus`
- [Source: `apps/api/internal/models/episode.go:29-33`] — episode subtitle columns (Story 12-2)
- [Source: `apps/api/internal/subtitle/process_item.go`] — `transcribeFallback` guard
- [Source: `apps/api/internal/subtitle/worker_pool.go`] — episode sweep loop + `asrRecoverable()`
- [Source: `project-context.md`] — Rule 11 (narrow interfaces), Rule 19 (services must not import subtitle), Rule 20 (contract posture above), Rule 24 (Discovery Triage)

## Senior Developer Review (AI)

**Date:** 2026-08-07 · **Outcome:** Approve (after same-session fixes) · **Findings:** 0 High / 1 Medium / 2 Low

**Mandatory checks:**

- 🔒 Rule 7 Wire Format: **PASS** (0 new error-code constants; the AC #7 wiring error is inline prose matching the movie-repo precedent, not a coded sentinel)
- 🔒 Rule 20 Contract Bump: **PASS** (1 bump — authored BY this review's M1 fix: `transcription_*` SSE `media_id` stamp `[@contract-v1→v2]`; downstream-ack grep → 9R-16 / ux3-ai-2-workspace-frontend / ux3-subtitle-v2-batch all `done` = FROZEN, 0 not-done consumers, scan recorded in the Change Log row below)
- 🔒 Rule 25 Mega-line: **N/A** (project-context.md untouched)
- Git vs Story File List: **0 discrepancies** (13 = 13 at review start; +1 file from the M1 fix, File List updated)
- AC validation: all 9 IMPLEMENTED with file:line evidence; Task audit: all [x] verified

**Action Items:**

- [x] **[M1]** `transcription_*` SSE events' `[@contract-v1]` stamp (9R-18) declared `media_id` = "the MOVIE row id" — this story makes episode UUIDs flow into that stream for the first time, a semantic widening of the stamped id domain shipped without a bump. **Fixed:** stamp bumped `[@contract-v1→v2]` at `transcription_service.go` (movie-or-episode media row id; the 9R-18 UUID-string clause unchanged) + the FE hook's descriptive header (`useGenerationProgress.ts`) updated in step (comment-only, zero behavior). Verified harmless at runtime: both consumers filter by strict equality against their own tracked id (`useGenerationProgress.ts:180`), so episode events pass unobserved.
- [x] **[L2]** `TestRunTranscription_EpisodeNilReaderSkipsResumeCheck` asserted only `require.Error` — it could pass for the wrong reason (e.g., a gate rejection). **Fixed:** added `NotErrorIs(err, ErrTranscriptionDisabled)` to pin the failure mode (full run attempted, not gate-rejected).
- [x] **[L1]** `WithMediaType` silently maps unknown values (e.g. a future `series` caller) onto the MOVIE table path. **No change:** the leg's `transcribable` guard is the authoritative filter, callers are in-repo, and the option's doc comment declares the defensive fallback. Recorded as accepted.

**Post-fix verification:** subtitle + services + models suites ✅ · full `nx test api` + `nx test web` ✅ · gofmt/vet/prettier clean.

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — dev-story workflow, 2026-08-07

### Debug Log References

- RED confirmed before implementation: services side `SetEpisodeSubtitleStateReader`/`WithMediaType` undefined; subtitle side fake/port signature mismatch (`have Transcribe(ctx, MediaRef, ...) want Transcribe(ctx, string, ...)`). GREEN after implementation.
- Full regression: `pnpm nx test api` ✅ · `pnpm nx test web` ✅ (228 files / 2547 tests) · `go vet` clean · `gofmt -l` clean on all 10 touched Go files · prettier clean on touched docs.

### Completion Notes List

- **🔗 AC Drift: FOUND (by design — this story removes sub-3-1's annotated carve-out).** sub-3-1 AC #2 carries the CR-M1 "Scope narrowed at implementation: MOVIE-only" annotation with a pointer to `backlog-episode-asr-fallback` — this story IS that pointer's resolution, extending the same behavior to episodes. sub-3-1 is `done` = frozen; its annotation stays as history and explicitly names this work. Grep basis: `backlog-episode-asr-fallback` across `_bmad-output/implementation-artifacts/*.md` — every hit is the sub-3-1 story, the backlog entry, or this story (bidirectional, no third-party consumer). All other prior-AC hits are REUSE, not drift.
- **📎 Contract Stamps: FOUND (0 bumps produced, acks recorded):** confirmed against [@contract-v3] sub-1-2 AC #2 (as bumped by sub-3-1) — the D2 terminality contract already declares `no_text_source` INTERMEDIATE with no media-type carve-out; the episode filter was an implementation guard, so removing it needs NO bump. Confirmed against the 9R-16 `RunTranscription` seam — consumed ADDITIVELY (`WithMediaType` option; signatures untouched, movie default keeps every legacy call site byte-unchanged). `SpeechTranscriber.Transcribe(ctx, MediaRef, ...)` signature change: unstamped sub-3-1 port, in-repo consumers only (Pipeline / main adapter / test fake) — recorded in Change Log per the story's AC #4.
- **🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched).**
- **🎨 UX Verification: SKIPPED — no UI changes in this story** (frontend untouched; `EpisodeList.tsx`'s "only P2 ASR can change this one" comment is now literally true for episodes as well — audited, no edit needed).
- **AC #8 known limitation (recorded, not solved):** `loadGlossary` → `LookupByMedia(mediaID)` keys `show_glossary` on the movie/show media id, so an EPISODE-id lookup returns empty and episode ASR translations run glossary-less. Fail-soft already covers it (nil pairs → plain translation; a glossary miss never blocks generation). Resolving show-key glossary resolution for episodes is out of scope — candidate future work per spec §6.5 (名詞庫 auto-harvest, 順序無關) if episode translation-consistency complaints materialize.
- **Design notes as implemented:** dispatch lives INSIDE the service (`writeSubtitleStatus` / `loadSubtitleRowState`) keyed on `transcriptionConfig.mediaType` — zero existing call-site edits; the movie branch preserves 9R-16's nil-writer skip semantics EXACTLY while the episode branch fails loudly (AC #7 — no pre-sub-3-2 episode behavior exists to preserve, and the movie-repo fallback would report a phantom "movie not found"). The episode resume is table-honest: an episode run consults ONLY the episode reader (pinned by the no-cross-resume test) — resuming off the movie table's row for the same UUID would be wrong even though UUIDs never collide in practice.
- **Movie-scope regression safety:** the eight pre-existing `translateAndPersist` test call sites were mechanically updated to pass the movie media type (signature gained the param); every behavioral assertion is unchanged and green — plus the new `TestTranslateAndPersist_DefaultMediaTypeHitsMovieWriter` and the untouched handler/generation-batch suites pin the default path end-to-end.

### Change Log

| Date | Change |
| ---- | ------ |
| 2026-08-07 | Tasks 1–4: `WithMediaType` TranscriptionOption (movie default) + `EpisodeSubtitleStatusWriter`/`EpisodeSubtitleStateReader` narrow interfaces + setters + `writeSubtitleStatus`/`loadSubtitleRowState` dispatch (transcription_service.go); `SpeechTranscriber.Transcribe` now takes `MediaRef` (unstamped in-repo port — consumers: Pipeline, `pipelineASRAdapter`, test fake, all updated in the same change); `transcribeFallback` guard → movie\|episode (series still degrades); `sweepEligible()` unifies both worker-pool loops; main.go wires `repos.Episodes` pair. |
| 2026-08-07 | Task 5: seven AC #9 cases (NEW `transcription_episode_test.go` + leg/sweep test amendments, RED→GREEN); spec §8 M2 checklist line updated; glossary limitation recorded (AC #8); full api+web regression green. No contract bumps at implementation ([@contract-v3] ack + additive 9R-16 seam consumption); the CR-authored SSE bump is the row below. |
| 2026-08-07 | [@contract-v1→v2] `transcription_*` SSE `media_id` (9R-18 lineage, CR M1): what changed — the id domain widens from "the movie row id" to "the media row id (movie OR episode)"; UUID-string format and payload shape unchanged. What breaks downstream — any consumer assuming every `transcription_*` event resolves to a MOVIE row (e.g., a movie lookup on unfiltered events); bump-side grep found acked consumers 9R-16 / ux3-ai-2-workspace-frontend / ux3-subtitle-v2-batch, all `done` = frozen, **no not-done downstream consumers**; both live FE consumers filter by strict own-id equality and observe no episode events. |
| 2026-08-07 | Senior Developer Review (AI): 0H/1M/2L — M1 SSE stamp bump (row above) + FE hook comment truth-fix; L2 failure-mode assertion added; L1 accepted with rationale. Status review → done. |

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - Filed at authoring time: NONE beyond the two recorded non-goals — episode manual-generation UI (product decision, unfiled by design until requested) and show-key glossary resolution for episodes (AC #8 known limitation; candidate future entry if episode translation quality complaints materialize).
  - Discovered at implementation: **NONE** — the work landed exactly inside the authored scope; no new sprint-status entries owed.

### File List

- `apps/api/internal/services/transcription_service.go` — episode interfaces + setters, `WithMediaType`/`newTranscriptionConfig`, mediaType threading, `writeSubtitleStatus`/`loadSubtitleRowState` dispatch, movie-scoped public `CanResumeTranslateOnly` + internal variant
- `apps/api/internal/services/transcription_episode_test.go` — NEW: episode writer/reader fakes + AC #9 (c)(d)(e)(f) cases
- `apps/api/internal/services/transcription_generation_test.go` — eight `translateAndPersist` call sites updated to the new signature (movie media type; assertions unchanged)
- `apps/api/internal/subtitle/pipeline.go` — `SpeechTranscriber.Transcribe` takes `MediaRef` (doc updated)
- `apps/api/internal/subtitle/process_item.go` — `transcribeFallback` movie|episode guard + series rationale comment
- `apps/api/internal/subtitle/process_item_asr_test.go` — fake signature, episode-runs test (replaces episode-degrade), series-degrade test
- `apps/api/internal/subtitle/worker_pool.go` — `sweepEligible()` shared filter; both loops unified
- `apps/api/internal/subtitle/worker_pool_test.go` — sweep pair amended for episodes
- `apps/api/cmd/api/asr_adapter.go` — adapter maps `ref.MediaType` → `WithMediaType`
- `apps/api/cmd/api/main.go` — `SetEpisodeSubtitleStatusWriter/Reader(repos.Episodes)` wiring
- `_bmad-output/planning-artifacts/vido-subtitle-pipeline-spec.md` — §8 M2 checklist line (episode leg via sub-3-2)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — sub-3-2 status transitions; backlog promotion tombstone (at create-story)
- `apps/web/src/hooks/useGenerationProgress.ts` — CR M1: header comment truth-fix only (media_id domain note; zero behavior, zero rendering)
- `_bmad-output/implementation-artifacts/sub-3-2-episode-asr-fallback.md` — this story file
