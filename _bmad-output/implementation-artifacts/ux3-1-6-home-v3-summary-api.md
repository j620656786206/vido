# Story ux3-1-6 — Home v3 readout-band aggregate (`GET /api/v1/home-summary`)

**Epic:** ux3-home-v3 (UX Redesign Phase 3) · **Status:** ready-for-dev · **Type:** backend
**Spec:** `tech-spec-ux3-home-summary.md` (D1–D4 are BINDING — endpoint shape, cell
semantics, spend persistence, no-cap rule) · **Pairs with:** ux3-1-5 design (H1-D-v3
讀數帶) · FE consumers = ux3-1-7 (band) / ux3-1-8 (hero, no BE dep — D4)

## What

The cheap aggregate behind the Home v3 readout band's four cells. New
`GET /api/v1/home-summary`, modeled on `status_summary`/`activity`: reads existing
services + a handful of NEW `COUNT` queries, **fail-soft per cell** (always 200; a
broken cell is `"unavailable"`, siblings render). Includes the 長解 groundwork:
per-run spend persisted on `subtitle_runs`, `completed_at` indexed on both job tables.

## Contract — per tech-spec D1 `[@contract-v1]`

```jsonc
{ "success": true, "data": {
  "coverage":        { "status": "ok", "covered": 42, "total": 55 },
  "processed_today": { "status": "ok", "count": 3 },
  "attention":       { "status": "ok", "failed_count": 2,
                       "spent_usd": 1.2, "budget_usd": 5.0, "spend_source": "live_batch" },
  "in_flight":       { "status": "ok", "count": 2 }
}}
```

snake_case on the wire (Rule 18). Spend trio omitempty; `spend_source` ∈
`"live_batch" | "last_run"`, backend resolves precedence (live wins). Copy-free
backend — all human copy lives on the web client.

## Acceptance criteria

1. `GET /api/v1/home-summary` returns the D1 contract; **never** an error envelope —
   each cell degrades alone to `{"status":"unavailable","error":"…"}`.
2. `coverage.covered` counts movies via the INVERSE of `missingZhHantSubtitleWhere`
   (keyed `subtitle_language = 'zh-Hant'`, non-empty `file_path`, not-removed) plus
   series that have ≥1 on-disk episode AND no episode matching
   `missingZhHantSubtitleEpisodeWhere`. `coverage.total` = movie + series counts
   (same predicates as `/library/stats`). Fileless items: in `total`, never `covered`.
3. `processed_today.count` = distinct media completed since server-local start-of-day
   across `parse_jobs` ∪ `subtitle_runs` (union/dedupe in Go, **no LIMIT** — D2).
4. `attention.failed_count` = `COUNT(*)` of `parse_jobs.status='failed'` +
   `subtitle_runs.status='failed'`. Download errors NOT folded in (D2 rationale).
5. `in_flight.count` uses the SAME source wiring as `/activity`'s `active_jobs`
   (scanner + batch subtitle + generation batch + transcription) — one counting path,
   no drift with the nav badge.
6. Migration 032: `subtitle_runs.spent_usd REAL NULL` + `budget_usd REAL NULL`;
   indexes `idx_subtitle_runs_completed_at`, `idx_parse_jobs_completed_at`.
7. `subtitle/process_item.go` writes `spent_usd`/`budget_usd` on EVERY terminal run
   update (completed, failed, skipped — cost incurred regardless of outcome).
8. Spend resolution: live `GenerationBatchProgress` → else
   `SubtitleRunRepository.LatestWithSpend()` → else trio absent (absent ≠ $0).
9. Tests: service tests cover each cell's fail-soft + nil-source degradation +
   spend-precedence (live over last_run over absent); repo tests for each new query
   (incl. series-coverage vacuous-truth guard: zero-episode series is NOT covered);
   migration runs green on existing DB.
10. Gates: `go build ./...`, `go vet`, `gofmt` clean, `pnpm nx test api` green.

## Tasks

1. Migration `032_add_run_spend_and_completed_at_indexes.go` (AC 6).
2. Recorder: thread run budget + `ai.Budget` spend into terminal updates in
   `subtitle/process_item.go` (AC 7).
3. New repo queries + interface decls + mocks:
   `MovieRepository.CountZhHantSubtitle`, `SeriesRepository.CountZhHantCovered`,
   `ParseJobRepository.CountByStatus` + `CompletedMediaIDsSince`,
   `SubtitleRunRepository.CountByStatus` + `CompletedMediaRefsSince` +
   `LatestWithSpend` (ACs 2–4, 8).
4. `services/home_summary_service.go` (+ `_test.go`) — four cells, fail-soft,
   start-of-day anchor computed server-local in Go (AC 1–5, 8, 9).
5. `handlers/home_summary_handler.go` + route registration + wiring in
   `cmd/api/main.go` (share the /activity source slice — AC 5).
6. Run gates (AC 10).

## Real vs. greenfield (Rule 24)

| Cell | Source | Notes |
|---|---|---|
| coverage | NEW counts over existing predicates | predicates live at `movie_repository.go:905-949` / `episode_repository.go:151-186`; series-grain counter is net-new (no series missing-counter exists) |
| processed_today | `parse_jobs.completed_at` (always written) + `subtitle_runs.completed_at` (pipeline mode only) | lower bound until pipeline default flips — documented limitation, NOT padded |
| attention.failed | NEW `COUNT` by status | enums verified `"failed"` on both tables |
| attention.spend | live batch OR newly-persisted run spend | absent until first pipeline run after migration |
| in_flight | existing `ActivityProgress()` sources | reuse, do not re-derive |

## Next

ux3-1-7 (FE band) consumes this contract; ux3-1-8 (hero+tail) needs no BE (D4).
FE camelCases at the `fetchApi` boundary. E2E: seeded env should gain a deterministic
home-summary fixture path when 1-7 lands.
