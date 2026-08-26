---
status: 'approved'
story: 'ux3-1-6-home-v3-summary-api (consumed by ux3-1-7 / ux3-1-8)'
author: 'Architect session, 2026-08-26 (facts by code survey; decisions per home-v3-identity-brief.md §7)'
inputDocuments:
  - _bmad-output/planning-artifacts/home-v3-identity-brief.md
  - _bmad-output/planning-artifacts/sprint-change-proposal-2026-08-26.md
---

# Tech Spec — Home v3 readout-band aggregate (`GET /api/v1/home-summary`)

Settles the three open decisions the brief forbids builders to invent: endpoint shape,
time anchor, spend source. Every field name below is verified against code (file:line
in §6), not guessed.

## D1. Endpoint: NEW `GET /api/v1/home-summary` (not an /activity extension)

`/activity` is the Activity hub's contract: in-memory job state + qBT counts + recent
parse events. The readout band needs DB `COUNT` aggregates (coverage, today, failures)
that belong to no existing section. Precedent for a page-scoped fail-soft aggregate
already exists: `GET /api/v1/status/summary` (`status_summary_service.go`). Follow it:
new `HomeSummaryService` + handler, **fail-soft per cell** (never a 500; a broken cell
reports `"unavailable"`, its siblings still render) — unlike `/library/stats`, which
500s on any repo error and is therefore NOT reused as the vehicle.

### Response contract `[@contract-v1]`

```json
{ "success": true, "data": {
  "coverage":        { "status": "ok", "covered": 42, "total": 55 },
  "processed_today": { "status": "ok", "count": 3 },
  "attention":       { "status": "ok", "failed_count": 2,
                       "spent_usd": 1.2, "budget_usd": 5.0, "spend_source": "live_batch" },
  "in_flight":       { "status": "ok", "count": 2 }
}}
```

- `attention.spent_usd` / `budget_usd` / `spend_source` are omitempty — absent when no
  spend datum exists yet. `spend_source`: `"live_batch"` (a generation batch is running
  now) | `"last_run"` (latest persisted completed/failed run). Backend resolves the
  precedence (live wins) so every client shares one rule.

- Envelope via existing `SuccessResponse` (`{success, data}`).
- Each cell: `status: "ok" | "unavailable"` + `error` (omitempty), same constants as
  `sectionOK`/`sectionUnavailable` in `status_summary_service.go:54-55`.
- FE honesty mapping (per brief §2/§5): `unavailable` → cell hides its number; `0` is
  rendered as 0; `failed_count == 0` → 「一切正常」.

## D2. Cell semantics and sources

### coverage — 「繁中字幕 42/55」(by 部: movies + series)
- `total` = `MovieRepository.Count` + `SeriesRepository.Count` (existing; same
  predicates as `/library/stats`'s `movie_count`/`tv_count`).
- `covered` = movies-with-zh-Hant + fully-covered-series. The canonical predicate is
  the INVERSE of `missingZhHantSubtitleWhere` (`movie_repository.go:905-909`) — it keys
  on **`subtitle_language = 'zh-Hant'`** (the tag the placer writes), NOT on
  `subtitle_status`, and requires a non-empty `file_path`:
  - NEW `MovieRepository.CountZhHantSubtitle`: `COUNT(*) WHERE subtitle_language =
    'zh-Hant' AND file_path IS NOT NULL AND file_path != '' AND (is_removed = 0 OR
    is_removed IS NULL)`.
  - NEW `SeriesRepository.CountZhHantCovered`: series (not-removed) having ≥1 episode
    with a file AND no episode matching `missingZhHantSubtitleEpisodeWhere`
    (`episode_repository.go:151-154`) — `EXISTS(...) AND NOT EXISTS(...)`. A series
    with zero on-disk episodes is NOT covered (no vacuous truth).
- Fileless items are neither covered nor missing → `covered ≤ total` and the ratio is
  honest. Episode-grain detail stays on the library page (brief §5).

### processed_today — 「今天處理 3 部」
- Time anchor ruling: **calendar day, server-local** (start-of-day computed in Go,
  passed as UTC instant; both tables store UTC). Copy 「今天」 stays truthful; no
  rolling-24h relabel needed.
- Count = distinct media completed today across `parse_jobs` ∪ `subtitle_runs`:
  - NEW `ParseJobRepository.CompletedMediaIDsSince(since)` — media ids of jobs with
    `status='completed' AND completed_at >= ?` (completed_at is set on every terminal
    transition, `parse_job_repository.go:152-158`).
  - NEW `SubtitleRunRepository.CompletedMediaRefsSince(since)` — media (id, type) of
    runs `status='completed' AND completed_at >= ?`.
  - Union/dedupe in Go. **No arbitrary LIMIT** (長解 ruling): the day window itself
    bounds the result set (a NAS processes at most hundreds of items/day); a silent
    500-cap would be the same defect /activity's `pending.parse_count` already has —
    do not replicate it.
- `completed_at` gets an index on BOTH tables in this story's migration (see D3's
  migration — one migration serves both needs). Same-day queries stay cheap forever
  instead of "fine at today's volumes".
- KNOWN LIMITATION (documented, not a workaround): `subtitle_runs` rows are only
  written on the pipeline path (`VIDO_SUBTITLE_PIPELINE_MODE=pipeline`; default
  `legacy` today). The legacy engine has no run bookkeeping to hook — porting it would
  be dead-end work since the pipeline epic (M-waves, in flight) is retiring legacy.
  The long-term fix is the planned pipeline-default flip, at which point this cell's
  data completes itself with zero further changes here.

### attention — 「N 部失敗」(+ spend, see D3)
- `failed_count` = NEW `ParseJobRepository.CountByStatus('failed')` + NEW
  `SubtitleRunRepository.CountByStatus('failed')` (both plain `COUNT(*)`; status
  enums verified: `ParseJobFailed`/`SubtitleRunFailed` = `"failed"`, index exists on
  `subtitle_runs.status`).
- Download errors are NOT folded in (already surfaced by sidebar strip + /downloads;
  folding them would double-report and couple this endpoint to qBT availability).

### in_flight — 「2 個任務」
- Same number the nav badge shows (`useInflightJobCount` = len(activeJobs.jobs) on
  /activity). Backend: reuse the SAME source wiring (`ActivityProgress()` sources:
  scanner, batch subtitle, generation batch, transcription) — inject the identical
  source slice into `HomeSummaryService` in `main.go`; do NOT re-derive from different
  services (one counting path, no drift).

## D3. Spend 「$1.2/$5」: persist per-run cost NOW (長解 ruling, 2026-08-26)

Fact: per-run cost today is in-memory only (`ai.Budget`), zero persisted spend columns;
the only computed spend-vs-budget is `GenerationBatchProgress.spent_usd/budget_usd`,
non-nil ONLY while a batch runs. An earlier draft deferred persistence and showed the
amount live-batch-only — **rejected as a short-term workaround** (Alexyu ruling:
architecture takes the long solution). The recording infrastructure all exists
(`ai.Budget.SpentUSD()` at run end, `subtitle_runs` terminal updates); persisting is a
column away, and Epic 16 (media statistics dashboard) will want cost history anyway.

Ruling — in scope for ux3-1-6:
- **Migration** (one migration, serves D2 too): `subtitle_runs` + `spent_usd REAL NULL`
  + `budget_usd REAL NULL`; index `idx_subtitle_runs_completed_at(completed_at)`;
  index `idx_parse_jobs_completed_at(completed_at)`.
- **Recorder**: `subtitle/process_item.go` terminal updates write
  `ai.BudgetFromContext(ctx).SpentUSD()` + the run's budget onto the run row (every
  terminal path: completed, failed, skipped — cost was incurred regardless of outcome).
- **Endpoint**: `attention` resolves spend as live-batch (`GenerationBatchProgress`,
  running now) → else latest persisted run with non-null `spent_usd`
  (NEW `SubtitleRunRepository.LatestWithSpend()`) → else fields absent. `spend_source`
  tells the client which it is; FE copy can say 執行中 vs 最近一次執行.
- Legacy-path caveat: same as D2 — legacy runs record nothing; fields stay absent until
  a pipeline run happens. Honest (absent ≠ $0), self-completing on the default flip.

## D4. Hero data for ux3-1-8: zero new API

Own-library hero = newest 3–5 items WITH backdrop: existing library list endpoints
(sort by added desc) + `backdrop_path` on movie/series models; filter + cap client-side.
Confirmed — 1-8 has no backend dependency and may run parallel to 1-7 after 1-6.

## §5. Non-goals

- No SSE push for band numbers in v1 (fetch on mount; nav badge already polls
  /activity — do not add a second live channel for the same numbers).
- No fix for /activity's capped `pending.parse_count` (known defect, separately
  tracked — but this endpoint does NOT replicate it, per D2).
- No porting of run bookkeeping into the legacy subtitle engine (dying code path;
  the pipeline-default flip completes the data).

## §6. Fact base (file:line)

- /activity shape + sources: `internal/services/activity_service.go:65-121, 187-251`;
  fail-soft constants `status_summary_service.go:54-55`.
- Missing-zh-Hant predicates: `movie_repository.go:905-949`,
  `episode_repository.go:148-186`; placer writes `zh-Hant`: `subtitle/placer.go:50,138`.
- Library counts: `library_service.go:583-660`, `movie_repository.go:532-540`,
  `series_repository.go:529-537`.
- Timestamps: `parse_job_repository.go:152-158`; `subtitle_runs` migration
  `030_create_subtitle_runs_table.go:46-79`; pipeline-mode gate `main.go:615-643`,
  `config/subtitle_pipeline.go:16,45-47`.
- Failure enums: `models/parse_job.go:9-13`, `models/subtitle_run.go:16-25`.
- Spend: `ai/budget.go:95-205`, `services/generation_batch.go:62-74,181-190`,
  `handlers/generation_batch_handler.go:45-47`, `config/config.go:77,138`.
