# Story 9R-16: Route C batch-generation endpoint — orchestrator, shared budget, batch SSE

Status: ready-for-dev

> **Pair note (SM Bob, 2026-07-05):** authored as the BE half of the PH3-M5 batch slice, paired with `ux3-subtitle-v2-batch` (FE) which consumes this story's [@contract-v1] endpoint + SSE shapes (13-7a/b pairing precedent). Epic: 9R (Route C). Filed by ux3-subtitle-v2 create-story Discovery Triage (Rule 24 ③) — the design's BE-gaps note (c) assumed "batch generation = 9R-10 + 9R-11" but neither shipped a batch trigger; `/api/v1/subtitles/batch` is the Epic 8 provider-FETCH batch.

## Story

As a Vido user with a library full of media missing 繁中 subtitles,
I want one batch action that runs the Route C generation pipeline across all missing items (or my selected ones) under a shared cost ceiling,
so that the library heals in bulk, spending stops at my budget, and the remainder picks up next run — a normal outcome, not an error.

## Acceptance Criteria

1. **`POST /api/v1/subtitles/generation-batch` [@contract-v1] — start.** Body (snake_case): `{"scope": "missing"|"selected", "media_ids": [<int64>...]}` (`media_ids` required iff `scope=selected`; movies only — see AC 8). Responses: **202** `{"success":true,"data":{"batch_id":"<uuid>","total_items":<n>,"items":[{"media_id":<int64>,"title":"<zh title>"}...]}}` (the enumerated queue, in run order — FE renders the F8 row list from this); **409** `TRANSCRIPTION_BATCH_RUNNING` with current progress in the error body (mirror `SUBTITLE_BATCH_RUNNING`, `subtitle_handler.go:423-434`); **503** `TRANSCRIPTION_DISABLED` (reuse — `IsAvailable()` gate); **400** `VALIDATION_*` (bad scope / empty or missing `media_ids` for selected / `media_ids` given with scope=missing); **200-empty-scope**: `scope=missing` resolving to 0 items returns 200 `{"data":{"total_items":0,"items":[]}}` — nothing to do is not an error.
2. **`GET /api/v1/subtitles/generation-batch/status` [@contract-v1]** → 200 `{"running":<bool>,"progress":<BatchProgress|null>}` (recovery probe, mirrors fetch-batch `GetBatchStatus`). **`POST /api/v1/subtitles/generation-batch/cancel`** → 200 `{"cancelled":<bool>,"running":<bool>}`, idempotent (F8 `全部取消`): in-flight item's pipeline ctx is cancelled, queued items never start, status → `cancelled`.
3. **`GET /api/v1/subtitles/generation-batch/preview?scope=missing` [@contract-v1]** → 200 `{"total_items":<n>}` — the F8 idle-dialog count (`缺字幕的項目 38`) BEFORE starting. (`scope=selected` needs no preview — FE knows its selection count.)
4. **Missing-zh-Hant enumeration (new repo finder).** NEW `MovieRepository` query for "needs Route C generation": `subtitle_language IS NULL OR subtitle_language != 'zh-Hant'`, AND `file_path` non-empty/on-record — deliberately BROADER than the fetch-batch's `FindBySubtitleStatus(not_searched|not_found)`: a movie with a found ENGLISH subtitle still lacks zh-Hant and is in scope. ⚠️ `subtitle_language` is written `"zh-Hant"` by **AC 13 of THIS story** — verified 2026-07-05: NOTHING on the transcription path writes it today (only the fetch engine/download handler call `UpdateSubtitleStatus`); without AC 13 this enumeration never shrinks. Used by scope=missing AND by preview. Resume-for-free corollary (holds ONLY with AC 13): completed items stop matching, so **`下次繼續` (F9) = simply start a new `scope=missing` batch — NO batch persistence table, NO migration** (in-memory single-flight state only, fetch-batch precedent; YAGNI ruled).
5. **Orchestrator — `GenerationBatchProcessor`.** Mirror `subtitle/batch.go`'s shape (global single-flight mutex + `activeBatch` + `context.WithCancel(context.Background())` detached from the HTTP request + collect-outside-lock + double-check): sequential loop, one item at a time (matches F8: one 轉錄中, rest 排隊中; the shared `ai.Governor` is the real throttle). Per item it calls a NEW **synchronous** TranscriptionService entry (AC 6) with `WithTranslation()`. Per-item error tolerance: an item failure (including per-media 409 `ErrTranscriptionInProgress` — user ran that item from the detail dialog mid-batch) increments `fail_count`, logs `slog.Warn`, and the loop CONTINUES. Location: `internal/services/` or `internal/subtitle/` — pick whichever avoids a Rule 19 cycle (orchestrator needs TranscriptionService which lives in services; putting it in services is the safe default; verify imports).
6. **Synchronous pipeline entry + shared-budget threading (TranscriptionService change).** (a) NEW exported synchronous method (e.g. `RunTranscription(ctx, mediaID, filePath, mediaDir, opts...) error`) that registers/deregisters the same per-media `inProgress` map (single-flight consistency with the async path), runs `runPipeline` inline, and **RETURNS the pipeline error** (the async path reports via `failJob` SSE only). ⚠️ CTX TRAP: the async precedent (`:191-195`) deliberately detaches — `context.WithTimeout(context.Background(), s.timeout)`; the sync path MUST derive from the CALLER's ctx (`context.WithTimeout(ctx, s.timeout)`) or the batch's shared Budget (a ctx value) and cancel propagation silently break. `StartTranscription` keeps its fire-and-forget behavior, now delegating to the same core. (b) Budget threading: `runPipeline` currently creates a fresh per-run `ai.Budget` (`transcription_service.go:227`) — change to **reuse a ctx-attached Budget when present** (`ai.BudgetFromContext`), else create per-run as today. The batch attaches ONE shared Budget (ceiling = `AI_RUN_BUDGET_USD`) to the batch ctx → the whole batch spends from one envelope. (c) Budget-sentinel propagation: `runPipeline` currently SWALLOWS translate-stage errors (`:308-319`, non-fatal by design — English SRT preserved). That stays — EXCEPT `errors.Is(err, ai.ErrBudgetExceeded)`, which MUST propagate out of the translate phase so AC 7's mid-item pause is implementable (otherwise a budget hit mid-translate counts the item as success and the orchestrator only notices one item late). Per-item async behavior otherwise unchanged (no [@contract] stamps exist on the pipeline — verified).
7. **Budget-ceiling state (F9 批次生成—預算上限) [@contract-v1].** Before starting EACH item, check `budget.Exceeded()`; if an in-flight item errors with `errors.Is(err, ai.ErrBudgetExceeded)`, treat the SAME way: that item and all remaining queued items are marked **paused** (NOT failed — design: `已暫停 — 下次繼續`, "partial completion is a normal outcome, not an error"), batch terminal status → **`budget_ceiling`**. Completed items stay done. No auto-resume — resume is the user starting a new batch (AC 4).
8. **Movies-only capability honor.** Series/episode Route C generation does not exist (9R-10a ready-for-dev). `scope=selected` with a non-movie id → that id is rejected at validation (400 with zh-TW message) or filtered out with the response noting it — pick ONE, document in Swagger. Enumeration for `scope=missing` queries movies only. (Series batch inclusion rides 9R-10a; tracked on the `ux3-subtitle-v2-batch` entry — no new entry.)
9. **Batch SSE event [@contract-v1].** NEW hub-level const `generation_batch_progress` (in `sse/hub.go` const block, fetch-batch precedent). Broadcast on every state change + per-item transition. Payload (snake_case, inner `data` of the standard double-nested envelope): `{"batch_id","total_items","current_index","current_media_id","current_item","success_count","fail_count","paused_count","status","spent_usd","budget_usd"}` where `status` ∈ `"running"|"complete"|"cancelled"|"error"|"budget_ceiling"` and `spent_usd`/`budget_usd` come from the shared Budget (`Snapshot()`/`SpentUSD()`) — **this is what makes the F8/F9 cost line (`本次用量：$0.42 / 上限 $5.00`) buildable without waiting for 9R-17**. Per-item STAGE detail is NOT duplicated here — FE joins the existing `transcription_*` events by `current_media_id` (frozen stage vocabulary untouched).
10. **Activity hub aggregation.** `GET /api/v1/activity` `active_jobs` gains kind **`generation_batch`** (`{kind:"generation_batch", current, total, detail}`) sourced from the processor's `GetProgress()` via a narrow source interface (mirror `scanStateSource`/`downloadCountSource`, fail-soft per section — ux3-2-2 pattern). Consumer: FE `ACTIVE_META` (ux3-subtitle-v2-batch).
11. **Rule 7.** New codes under EXISTING prefixes only (no new prefix, no CR-workflow sync): `TRANSCRIPTION_BATCH_RUNNING` (409), `TRANSCRIPTION_BATCH_START_FAILED` (500). Reuse `TRANSCRIPTION_DISABLED` (503), `VALIDATION_*` (400). `AI_BUDGET_EXCEEDED` stays internal (surfaces as `status:"budget_ceiling"`, not an HTTP error). Swagger annotations on all four routes; `swag init` if changed (Rule 15: verify route registration in main.go wiring).
12. **DB writeback on generation success (the resume enabler + badge truth) [@contract-v1].** After a successful zh-Hant place — on BOTH the sync batch path and the async per-item path — update the movie row via the EXISTING `MovieRepository.UpdateSubtitleStatus` (`movie_repository.go:823`): `subtitle_status='found'`, `subtitle_path=<zh-Hant path>`, `subtitle_language='zh-Hant'`. Verified gap (2026-07-05): the transcription path performs ZERO repo writes today — only the fetch engine (`subtitle/engine.go:342-361`) and download handler call `UpdateSubtitleStatus`. Without this: the missing-scope enumeration never shrinks (batch re-transcribes finished movies and double-spends budget), preview counts never drop, and library poster badges stay 缺字幕 after generation until a rescan (breaks ux3-subtitle-v2 AC 6's badge-refresh expectation). Rule 19-safe: services already imports repository; give TranscriptionService a narrow updater interface (Rule 11). en-only runs (`translate` absent) do NOT write zh-Hant fields; failures write nothing. This absorbs the pre-existing per-item writeback gap (Rule 24 lane ① — this AC is the tracking).
13. **Tests + gates.** Repo: new finder matrix (NULL / en / zh-Hant / no-file). Service: sync-entry single-flight consistency (async+sync share the map), ctx-budget reuse vs fresh. Orchestrator: sequential order, per-item fail-continue, per-media-409 skip, cancel mid-item + mid-queue, budget-ceiling pre-check AND mid-item `ErrBudgetExceeded` → paused counts + `budget_ceiling` status, SSE payload field assertions, 409 single-flight, empty-scope. Handler httptest: 202/200/400/409/503 + preview + status + cancel. Activity: new kind fail-soft. Full Go suite + staticcheck + `pnpm lint:all` green.

## Tasks / Subtasks

- [ ] Task 1: Repo finder (AC: 4)
  - [ ] NEW movie query "missing zh-Hant + has file" + count variant; real-sqlite tests
- [ ] Task 2: TranscriptionService — sync entry + budget threading + writeback (AC: 6, 12)
  - [ ] `RunTranscription` synchronous export sharing `inProgress` registration, error-returning, caller-ctx-derived timeout; `StartTranscription` delegates
  - [ ] `runPipeline` reuses ctx Budget when present; `ErrBudgetExceeded` propagates out of translate phase (other translate errors stay non-fatal); tests for all paths
  - [ ] `UpdateSubtitleStatus` writeback on zh-Hant place success (both paths, narrow updater iface); tests: success writes / en-only no-write / failure no-write
- [ ] Task 3: `GenerationBatchProcessor` (AC: 5, 7)
  - [ ] Single-flight state machine + sequential loop + cancel + budget-ceiling (pre-check + mid-item sentinel) + paused accounting
  - [ ] SSE broadcasts (AC 9) with cost fields; orchestrator tests
- [ ] Task 4: Handler + routes (AC: 1, 2, 3, 8, 11)
  - [ ] `/subtitles/generation-batch` group: POST start / GET status / POST cancel / GET preview; validation; Swagger; main.go wiring (Rule 15 grep)
  - [ ] httptest matrix
- [ ] Task 5: Activity aggregation (AC: 10)
  - [ ] `generation_batch` active-job source (narrow iface, fail-soft) + tests
- [ ] Task 6: Gates (AC: 13)
  - [ ] `go test ./...`, vet, staticcheck, `pnpm lint:all`, prettier on touched md/yaml

**Cross-stack split check:** backend tasks = 6, frontend tasks = 0 → single story. ✓ (FE = `ux3-subtitle-v2-batch`.)

## Dev Notes

### Code anchors (verified 2026-07-05)

- **Single-flight:** `transcription_service.go:78-79` (`mu` + `inProgress map[int64]string`), `:181-197` (`StartTranscription` → goroutine + own 5-min ctx `:191-195`), cleanup `:217-221`, `ErrTranscriptionInProgress` `:49`, `IsInProgress` `:160-165`, `IsAvailable` `:155-157`, `WithTranslation` `:200-212`.
- **Route C chain:** `runPipeline` `:216` → extract `:248-272` → transcribe `:282` → `.en.srt` `:289-295` → translate `:298-319` → `translateSRT` `:436` (glossary `:466` → `TranslateWithGlossary` `:470` → OpenCC s2twp fail-soft `:481-488` → atomic place `:494-505`).
- **Budget:** created at `:227-228` (`ai.NewBudget(s.runBudgetUSD)` + `ai.WithBudget`) — the line AC 6(b) changes. `ai/budget.go:39-53` (per-run doc), `Exceeded()` `:62-69`, `SpentUSD()` `:110`, `Snapshot()` `:131-142` (fields have NO json tags — batch SSE builds its own map, don't marshal the struct raw). `ai/governor.go:68-80` (`governed` pre-checks budget → `ErrBudgetExceeded` `types.go:18-21`). Governor built once `cmd/api/main.go:489`.
- **Fetch-batch template (mirror, do NOT touch):** `subtitle/batch.go` — `BatchProcessor` `:87-97`, `Start` collect-outside-lock + double-check `:159-198`, sequential `process` `:204` with ctx-cancel checks `:213-225`/`:277-289`, per-item fail-continue `:246-270`, SSE map payload `:361-377`, `GetProgress` copy `:131-140`, `Cancel` `:115-121`. Handler 409-with-progress `subtitle_handler.go:423-434`, status `:452-473`, cancel `:479-499`. Wiring `main.go:635-637`.
- **Enumeration fields:** `models/movie.go:146-149` (`subtitle_status`/`subtitle_path`/`subtitle_language`); existing `FindBySubtitleStatus` `movie_repository.go:859-882`; `List` has NO subtitle filter `:315-343` (the FE-side list filter is separately tracked: `disc-2026-06-library-subtitle-status-filter`).
- **SSE:** hub consts `sse/hub.go:12-34` (add `generation_batch_progress` here), `Event` `:37-41`, `Broadcast` `:141-148` (non-blocking). Transcription event consts live in the service `transcription_service.go:39-45` — per-item events UNCHANGED (frozen stage vocabulary: FE maps `extracting/transcribing/translating/complete/failed`; ux3-subtitle-v2 consumes them).
- **Activity precedent:** ux3-2-2 — `/api/v1/activity` aggregates via narrow sources (`scanStateSource`, `downloadCountSource`), fail-soft per section. FE `ActiveJob.kind` union today: `'scan'|'subtitle_batch'` (`activityService.ts:18`).

### Rulings (do not re-litigate)

- **No persistence table.** Resume = re-enumerate scope; completed items self-exclude via `subtitle_language='zh-Hant'` — which AC 12's writeback makes true (it is NOT true on today's code). `下次繼續` is an FE re-start call.
- **ID types:** the wire contract uses `media_id` int64 (consistent with `/movies/:id/transcribe` + `transcription_*` SSE payloads), but `models.Movie.ID` is a STRING (`movie.go:106`) and repo methods take string ids — the handler/orchestrator converts both ways (enumeration → `items[].media_id`, `media_ids` → repo lookups). FE selection is `Set<string>` → `Number()` on its side (noted in ux3-subtitle-v2-batch).
- **Sequential, not parallel.** Design shows one active item; Governor still guards AI concurrency if this ever changes.
- **Cost on the batch SSE, not 9R-17.** Batch-scoped `spent_usd/budget_usd` ride `generation_batch_progress`. 9R-17 (general AI-usage endpoint, per-item cost slot) stays a separate backlog entry — do NOT absorb it.
- **Fetch-batch untouched.** `/subtitles/batch` (Epic 8) keeps working; the two processors are independent single-flights (a fetch batch and a generation batch CAN run simultaneously — acceptable, they share no state; the Governor bounds AI spend, fetch uses no AI).
- **zh-TW messages** for all user-facing errors (Rule 3); slog for internals (Rule 2/13).

### Project Structure Notes

- All under `apps/api/internal/{services,subtitle?,handlers,repository,sse}` + `cmd/api/main.go` wiring. NO migration. NO FE. Rule 19: orchestrator in `services` (needs TranscriptionService); do not import `subtitle` from `services` (Known Cycle) — the orchestrator does NOT need the subtitle engine, only TranscriptionService, so this stays clean.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: sprint-status.yaml `9R-16-batch-generation-endpoint` seed (Rule 24 ③ from ux3-subtitle-v2, 2026-07-05)]
- [Source: _bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md — F8/F9 batch surfaces; "partial completion is a normal outcome, not an error"]
- [Source: ux-design.pen F8 `i9Nun1`/`H717g` (scope segments, `已完成 12/38`, `本次用量 $0.42/$5.00`, `全部取消`), F9 `JMqPg` (`已達本次預算上限（$5.00）— 已完成12部，剩餘26部下次繼續`, `已暫停 — 下次繼續`, `關閉`/`下次繼續`)]
- [Source: _bmad-output/implementation-artifacts/9R-10-pipeline-orchestration.md + 9R-11-ai-cost-quota.md — single flow, Governor/Budget internals, "batch = 9R-16" gap]
- [Source: project-context.md Rules 2/3/4/7/11/13/14/15/19/24; §8 SSE]

## Dev Agent Record

### Agent Model Used

(fill at dev time)

### Debug Log References

### Completion Notes List

### Discovery Triage

- Authoring-time (SM Bob, 2026-07-05): no NEW entries filed by this story — the FE-side gaps found in the same research pass are filed on `ux3-subtitle-v2-batch` (v2 multi-select) and pre-existing entries (`disc-2026-06-library-subtitle-status-filter`, `9R-17-ai-usage-endpoint`, `9R-10a`).
- **① expand-scope-in-place (validation pass, 2026-07-05): per-item generation DB-writeback gap** — the transcription path never updates `movies.subtitle_status/subtitle_path/subtitle_language` after a successful generation (badge stays 缺字幕 until rescan; missing-scope enumeration never shrinks). Absorbed as **AC 12** + Task 2 subtask (this AC is the tracked absorption). Cross-story note: ux3-subtitle-v2's AC-6 badge-refresh expectation depends on it — annotated on that story + its sprint entry.
- (Dev: add in-flight discoveries per Rule 24 before marking done.)

### File List
