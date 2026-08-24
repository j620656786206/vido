# Story: surface single-episode/movie transcription jobs on the Activity page

Status: done

## Story

As someone who clicked "生成字幕" on a single episode or movie and then closed the progress modal,
I want the Activity page to show that a subtitle-generation job is still running,
so that closing the modal doesn't make it feel like the job vanished into a black hole.

## Evidence (from a 2-lane parallel investigation, 2026-08-24, all citations verified by reading code)

- Reproduced scenario: user clicks 生成字幕 on 龍族前傳 (House of the Dragon) S03E01, the modal shows the live SSE stage tracker with "關閉後生成會在背景繼續", user clicks 關閉, navigates to `/activity` — the page shows only the Downloads card, nothing about the job.
- **The job genuinely keeps running server-side.** `TranscriptionService.StartTranscription` (`transcription_service.go:295-319`) spawns `go func(){ ...context.Background()... }()`, detached from the HTTP request context — confirmed, not speculative.
- **But nothing outside that goroutine can discover it exists.** The only record is `TranscriptionService.inProgress` (`transcription_service.go:154`), currently `map[string]string` (mediaID → jobID), consulted only via the single-key `IsInProgress(mediaID)` used internally for a 409-dedup check — **never exposed as a list**.
- `GET /api/v1/activity`'s active-jobs aggregate (`ActivityService.activeJobsSection()`, `activity_service.go:135-166`) is wired to exactly 3 in-flight sources today — `scan`, `subtitle_batch` (`s.batch`), `generation_batch` (`s.generation`) — via a shared `batchJobSource` interface:

  ```go
  type batchJobSource interface {
      ActivityProgress() (active bool, percentDone, current, total int, currentItem string)
  }
  ```

  `TranscriptionService` implements none of this. `main.go:899` wires `NewActivityService(scannerService, batchProcessor, generationBatchProcessor, downloadService, repos.ParseJobs)` — no transcription arg exists to pass.
- The modal's own SSE subscription is correctly torn down on close (`ManageSubtitleDialogV2.tsx:236`, comment: *"Closing only stops WATCHING — a running job continues server-side"*) — this is intentional, working as designed. The gap is entirely on the discovery side, not the teardown side.
- The frontend already half-anticipated this: `ActiveJob['kind']` is typed `'scan' | 'subtitle_batch' | 'generation_batch' | string` (`activityService.ts:19`), and `useGenerationJobsFeed.ts:14-20` carries a comment citing this exact gap by name (`disc-2026-07-transcription-active-jobs`) — this story closes that citation.
- `TranscriptionService` is constructed at `main.go:542`, well before `NewActivityService` at `main.go:899` — so it is already in scope to pass as a new constructor argument with no reordering needed.

## Acceptance Criteria

1. `TranscriptionService` tracks, per in-flight job, a human-readable title in addition to the job ID — resolved **once**, at job-start, via existing readers (`episodeReader` + `seriesReader` for an episode → `"{series title} S{season:02d}E{episode:02d}"`; `stateReader` for a movie → the movie's title). Resolution runs **before** acquiring the single-flight lock (no DB I/O while holding `s.mu`). On any lookup failure, fail soft: log a `Warn` (mirroring the existing `series metadata lookup failed` pattern at `transcription_service.go:1052-1055`) and fall back to the raw `mediaID` — never fabricate a title, never fail the transcription itself over a display-string lookup.
2. `TranscriptionService` implements the existing `batchJobSource` shape exactly — `ActivityProgress() (active bool, percentDone, current, total int, currentItem string)` — with a compile-time interface assertion. `percentDone` and `total` are always `0` (this service tracks discrete stages, not a fractional/bounded count — reporting a fabricated percent would be dishonest); `current` = number of concurrent in-flight jobs; `currentItem` = one representative job's title (arbitrary pick among concurrent jobs is acceptable — see Dev Notes) or `""` when `active` is `false`.
3. `ActivityService` gains a 6th active-jobs source, wired exactly like the existing two (`Kind: "transcription"`, `PercentDone/Detail/Current/Total` sourced from the new method) in `activeJobsSection()`. `NewActivityService`'s signature grows a `transcription batchJobSource` parameter; `main.go:899` passes the already-in-scope `transcriptionService`.
4. Frontend: `ActiveJob['kind']` (`activityService.ts:19`) is widened to include `'transcription'` explicitly (not left to the `| string` fallback). `ActivityHub.tsx`'s `ACTIVE_META` map gains a `transcription` entry (icon + zh-TW label, matching the existing 3-entry convention) so the row renders with real copy instead of the generic kind-as-title fallback.
5. **Explicitly out of scope for this story** (documented, not silently dropped): no deep-link from the new row into the `?view=generation` single-job workspace (that link today requires a `generation_batch` job to exist, per `ActivityHub.tsx:119-133`); no per-stage progress text (提取音訊/轉錄中/翻譯中/…) surfaced on this row — the row proves the job *exists*, live stage detail stays the modal's job. No SSE replay/backfill mechanism added to the Hub.
6. Regression corpus: `ActivityProgress()` covered for 0/1/N concurrent jobs; `resolveActivityTitle`-equivalent covered for episode success, movie success, and lookup-failure fallback; `activeJobsSection()` covered end-to-end showing the new `transcription` kind appears with the right shape when a job is in flight and disappears when none are. Frontend: `ActivityHub` renders the new kind with its own icon/label, not the generic fallback.
7. Gates: `pnpm nx test api` green, `pnpm nx test web` green, `pnpm run lint:all` clean, `pnpm run format:check` green. Manually verified against the live NAS: trigger a real single-episode generation, confirm it appears on `/activity`, confirm it disappears once the job completes.

## Tasks / Subtasks

- [x] Task 1 — Track title + widen the in-flight map (AC: #1)
  - [x] 1.1 `type soloTranscriptionJob struct { JobID, Title string }`; change `inProgress` to `map[string]*soloTranscriptionJob`
  - [x] 1.2 `resolveActivityTitle(ctx, mediaType, mediaID string) string` — fail-soft per AC #1
  - [x] 1.3 Call it in `StartTranscription` and `RunTranscription` before `acquireJob`; extend `acquireJob(mediaID, title string)` to store both; `runPipeline`'s existing `delete(s.inProgress, mediaID)` cleanup is unchanged (still keyed by mediaID)
- [x] Task 2 — `ActivityProgress()` (AC: #2, #6)
  - [x] 2.1 Implement matching the exact `batchJobSource` signature + compile-time assertion
  - [x] 2.2 Table-driven tests: 0 jobs (`active=false`), 1 job (title surfaces verbatim), N jobs (count correct, some title surfaces, `percentDone`/`total` stay 0)
  - [x] 2.3 Tests for `resolveActivityTitle`: episode success shape, movie success shape, lookup-failure → mediaID fallback + Warn logged
- [x] Task 3 — Wire into ActivityService (AC: #3, #6)
  - [x] 3.1 Add `transcription batchJobSource` field + constructor param; add the block in `activeJobsSection()` (copy the existing `subtitle_batch`/`generation_batch` block shape exactly, `Kind: "transcription"`)
  - [x] 3.2 Update `main.go:899` call site
  - [x] 3.3 Test: `activeJobsSection()` / `GetActivity()` shows the `transcription` job when the (mocked) source reports one active, and shows nothing when it reports none
- [x] Task 4 — Frontend surface (AC: #4, #6)
  - [x] 4.1 Widen `ActiveJob['kind']` type in `activityService.ts`
  - [x] 4.2 Add `transcription` entry to `ACTIVE_META` in `ActivityHub.tsx` (icon + zh-TW label — pick something distinct from the existing 字幕批次/生成批次 labels so a user can tell them apart, e.g. "字幕生成中")
  - [x] 4.3 Test: rendering a mocked `activeJobs.jobs` entry with `kind: 'transcription'` shows the real icon/label, not the generic fallback
- [x] Task 5 — Gates + live verification (AC: #7)

## Dev Notes

- **Reuse `batchJobSource`, do not invent a parallel interface.** This is the whole point of matching the signature exactly (AC #2) — `ActivityService.activeJobsSection()`'s new block should read as a copy-paste of the `subtitle_batch`/`generation_batch` blocks with the field names changed, not a bespoke branch.
- **Do not fabricate `percentDone`.** This codebase has a strong, repeatedly-demonstrated discipline against reporting numbers that aren't really tracked (see today's AI pricing/metering work). Mapping "current stage" to a guessed percentage would be exactly that. Leave it `0`; the live per-stage detail already exists in the modal via SSE — this row's job is only to prove existence.
- **"Arbitrary pick" for `currentItem` when N>1 is a deliberate simplification, not an oversight.** Go map iteration order is unspecified; picking whichever job the iteration reaches first is fine because ANY currently-running job's title is honestly representative of "something is generating right now." Do not add ordering/sorting machinery for this — it isn't worth the complexity for a cosmetic "which title shows" nicety. Note this explicitly in a code comment so a future reader doesn't mistake it for a bug.
- **Do not put a Chinese sentence into `Detail`.** (E.g. do NOT do `fmt.Sprintf("%d 部作品字幕生成中", count)`.) The established convention (`activity_service.go` comment: *"Kind drives the web client's icon + label... backend stays copy-free"*) plus the fact that every existing `Detail` value across this file is raw data (a filename, an item name), never a constructed sentence, means `Detail` here should be a real title (or the `mediaID` fallback), and `Current`/`Total` carry the count — exactly like `scan` already does (`Current: p.FilesFound`, no `Total` set).
- **Keep DB I/O out of the mutex-protected section.** `resolveActivityTitle` runs before `acquireJob` is called, not inside it — `acquireJob` stays a pure, fast lock/map operation as it is today.
- Rule 7 / 10: N/A (no new error codes, no new routes — `GET /api/v1/activity`'s existing shape just gains one more entry in an already-open array). Rule 20: N/A — no `[@contract-v*]` stamp exists on this endpoint's shape to bump; if a future story wants to formalize the `ActivitySummary` JSON contract, that's separate. Rule 23: N/A on the Go side; the touched frontend file (`ActivityHub.tsx`) reads no wall-clock, only server-reported job data.
- Cross-stack split check: backend ~4 tasks (1-3), frontend 1 task (4) → well under the >3-per-side threshold, no split needed.

### Time-dependent visual coverage

- N/A — `ActivityHub.tsx`'s touched surface (`ACTIVE_META` lookup) reads no `Date.now()`/`new Date()`.

### References

- [Source: apps/api/internal/services/transcription_service.go:154, 284-289, 295-325, 360-378 — the in-flight map, `IsInProgress`, `StartTranscription`, `acquireJob`]
- [Source: apps/api/internal/services/transcription_service.go:1020-1060 — the existing fail-soft series/movie title lookup pattern to mirror]
- [Source: apps/api/internal/services/activity_service.go:23-36, 49, 135-166 — `ActivityService` struct, `batchJobSource`, `NewActivityService`, `activeJobsSection()`]
- [Source: apps/api/cmd/api/main.go:542, 899 — `transcriptionService` construction order and the `NewActivityService` call site]
- [Source: apps/web/src/services/activityService.ts:19 — `ActiveJob['kind']` type]
- [Source: apps/web/src/components/activity/ActivityHub.tsx:39-44, 86-138 — `ACTIVE_META`, the "進行中" section render condition]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:14-20 — the frontend comment that originally cited this exact gap]
- [Source: apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:233-246 — modal SSE teardown-on-close, confirmed correct/intentional]

## Dev Agent Record

### Agent Model Used

claude-sonnet-5 (Claude Sonnet 5). Root cause was located by a 2-lane parallel Workflow investigation (independent frontend + backend read-only lanes, then a synthesis pass) before any story was written — the diagnosis that this is an *observability gap*, not a UI bug, came from that, and shaped the whole design.

### Debug Log References

- **RED (backend):** new `transcription_activity_test.go` failed to compile — `resolveActivityTitle undefined`, `ActivityProgress undefined`, `acquireJob` arity. Confirms the tests exercise genuinely-absent surface.
- **RED (frontend):** temporarily deleted the `transcription` entry from `ACTIVE_META` — the new spec failed with `Expected: 字幕生成中 / Received: transcription龍族前傳 S03E01進行中`, i.e. it correctly caught the row falling back to the generic kind-as-title. Restored → 14/14 green.
- **Live NAS verification (AC #7)** in a throwaway container (`vido-verify2`: own DB copy, own port 8098, media read-only, real ASR key so a genuine job ran). Production instance untouched throughout; container + temp files removed afterwards.
  - Baseline: `active_jobs` = `{"status":"ok","jobs":[]}`
  - `POST /api/v1/episodes/6edf8c64…/transcribe` on the exact episode from the bug report (龍族前傳 S03E01) → job started
  - 3s later: `{"kind":"transcription","percent_done":0,"detail":"龍族前傳 S03E01","current":1}` — the exact scenario that previously showed nothing
  - Polled to completion: visible continuously for ~200s, then cleanly absent (`transcription_jobs=0`) once the job finished
- Gates: `pnpm nx test api` green · `pnpm nx test web` green · `pnpm run lint:all` 0 errors (119 pre-existing jsx-a11y warnings, same count as prior stories today) · `pnpm run format:check` green.

### Completion Notes List

- **A design flaw was caught mid-implementation and fixed before it shipped.** The story as drafted would have double-counted: `RunTranscription` (the batch/pipeline entry) shares the *same* single-flight map as `StartTranscription` (the solo entry), so a batch item would have appeared BOTH in its own `generation_batch` row AND in the new `transcription` row — one real job rendered as two. Found by grepping the actual callers (`generation_batch_runner.go:38`, `cmd/api/asr_adapter.go:33`) rather than trusting the story's framing. Fix: `soloTranscriptionJob.Solo` flag — the map still serves its original dedup purpose for *both* paths (a batch item and a solo click on the same media still correctly 409 each other), but only solo jobs count toward `ActivityProgress()`. Pinned by two dedicated tests (`TestActivityProgress_NonSoloJob_DoesNotCount`, `TestActivityProgress_MixOfSoloAndBatchJobs_CountsSoloOnly`).
- **A second honesty issue surfaced during frontend work, also fixed.** The existing row renderer would have shown a literal `0%` for this kind's entire runtime (since the backend deliberately reports no percent), which reads as *stuck* — worse than showing nothing. Added `NO_PERCENT_KINDS`: renders a static `進行中` label and suppresses the progress bar entirely, rather than drawing a permanently-empty bar. This keeps the backend's "don't fabricate a number" discipline visible all the way to the pixel.
- **Batch path pays no cost.** `RunTranscription` passes an empty title and skips `resolveActivityTitle` entirely — the DB lookup doesn't just get hidden later, it never runs on the batch hot path.
- Title resolution runs strictly *before* `acquireJob`, so the mutex-protected section stays a pure, I/O-free map operation as it was.
- 8 pre-existing `NewActivityService(...)` test call sites updated for the new parameter; 3 pre-existing tests that wrote `inProgress` directly updated for the new struct value. All are mechanical signature/shape updates — no existing assertion's *meaning* changed.
- 🔗 AC Drift: **NONE.** No prior story specifies `ActiveJob.kind` as a closed set — the frontend type already carried a `| string` fallback and `ActivityHub` already had a generic-row fallback path, both of which still work (pinned by a new defensive test). This widens an open enum; it does not redefine a contract.
- 📎 Contract Stamps: **NONE** — no `[@contract-v*]` stamp exists on `GET /api/v1/activity`'s shape to bump. The change is additive (one more entry in an already-open array), so no consumer breaks.
- 🎭 A11y Pre-Flight: **PASS** — the touched frontend surface adds one row variant reusing the existing `ActivityRow` component (unchanged); suppressing `progress` removes a `role="progressbar"` element rather than adding an unlabeled one. 0 new jsx-a11y warnings.
- 🎨 UX Verification: the new row reuses the established `ActivityRow` shape (icon chip + title + detail + right slot) already validated for the other three kinds; no new visual pattern introduced.

### Discovery Triage

- **N/A — no out-of-scope work discovered.** The two issues found mid-implementation (batch double-counting, the misleading `0%`) were both *inside* this story's own scope — defects in the drafted design, corrected before shipping and recorded above, not separate findings to hand off.
- Note on scope already declared up-front (AC #5, not a new discovery): the deep-link from this row into the `?view=generation` single-job workspace, per-stage detail on the row, and SSE replay/backfill all remain deliberately unbuilt.

### File List

- apps/api/internal/services/transcription_service.go
- apps/api/internal/services/transcription_activity_test.go (new)
- apps/api/internal/services/activity_service.go
- apps/api/internal/services/activity_service_test.go
- apps/api/internal/services/transcription_service_test.go
- apps/api/internal/services/transcription_generation_test.go
- apps/api/cmd/api/main.go
- apps/web/src/services/activityService.ts
- apps/web/src/components/activity/ActivityHub.tsx
- apps/web/src/components/activity/ActivityHub.spec.tsx
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/disc-2026-07-transcription-active-jobs.md

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Task 1: `soloTranscriptionJob` record (JobID/Title/Solo) replaces the bare jobID string in `inProgress`; `resolveActivityTitle` added with fail-soft fallback to mediaID; `acquireJob` takes title+solo. |
| 2026-08-24 | Task 2: `ActivityProgress()` implemented against the existing `batchJobSource` shape, counting solo jobs only; 13 tests including the batch-double-count guard. |
| 2026-08-24 | Task 3: `ActivityService` gains a 6th source wired identically to the existing two; `main.go:899` updated; 2 new integration tests + 8 existing call sites updated. |
| 2026-08-24 | Task 4: frontend `kind` type widened, `ACTIVE_META` + `NO_PERCENT_KINDS` entries added (static 進行中 label instead of a misleading 0%); 2 new specs, RED-GREEN verified. |
| 2026-08-24 | Task 5: gates green; live NAS verification on the exact bug-report episode — job appears within 3s, stays ~200s, disappears on completion. |
