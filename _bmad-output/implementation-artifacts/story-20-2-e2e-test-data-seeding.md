# Story 20-2: E2E test-data seeding (kill the false-green self-skips)

Status: done

> Epic 20 — Test Quality Hardening · Story 2 of 3. Surfaced alongside bugfix-20-1
> during the Phase-2 v2 pilot: the season bug slipped past CI because the E2E
> tests that *should* cover the detail page silently self-skip on the empty CI DB.

## Problem

Playwright's `webServer` boots a **fresh, empty backend** in CI
(`VIDO_DATA_DIR: ./vido-data`). Data-dependent specs opened with
`const movie = (await api.listMovies()).data?.items?.[0]` then
`test.skip(!movie, 'No movies available')`. On the empty DB every such test
self-skips → the suite goes **green without exercising anything** (false
confidence — the exact blind spot that let the season-accordion bug ship).

Three concrete debts:

1. **`media-detail.spec.ts`** — ~10 tests guarded by `test.skip(!movie/!series)`.
   The `[P0]` one even *tried* to `api.createMovie(...)` first, but
   (a) it passed camelCase `{releaseDate}` while the Go endpoint binds
   **snake_case `release_date` (required)** → 400 → `data` undefined → skip; and
   (b) it `return`ed without ever asserting. A no-op either way.
2. **`manual-search.api.spec.ts`** — 4 `/metadata/apply` tests perma-skipped
   "until we have proper test data seeding".
3. **`api.spec.ts`** — 3 `describe.skip` blocks using the stale `.results`
   response shape, fully superseded by `health.api.spec.ts` / `movies.api.spec.ts`
   / `search.spec.ts`.

## Decision (Alexyu, 2026-06-14)

- **Seeding mechanism = fixture-layer API seeding + cleanup** (Option A of 4).
  Reuse the already-proven `POST /movies` / `POST /series` endpoints
  (`movies.api.spec.ts` CRUD is green), seed per-test, delete in `afterEach`.
  Zero backend surface, tests isolated, identical local/CI.
- **Season-accordion E2E = deferred** (Tier 2). Neither `POST /series` nor the
  local seed script writes the `seasons` table (only `parse_queue_service` does),
  so the accordion stays out of E2E reach. It is already guarded by bugfix-20-1's
  real-sqlite Go integration test. Tracked as `story-20-4-season-accordion-e2e`.

## Acceptance Criteria

1. **Seed helper.** `tests/support/helpers/seed-helpers.ts` exposes
   `seedMovie` / `seedSeries` (snake_case payloads matching
   `CreateMovieRequest` / `CreateSeriesRequest`, asserts `success` + `id`) and
   `deleteMovies` / `deleteSeries` cleanup. POST raw via the generic `api.post`
   (bypasses the mistyped camelCase `api.createMovie`).
2. **`media-detail.spec.ts` seeds, never self-skips.** Every data-dependent test
   seeds the row it needs (`tmdbId > 0` for the full detail view; no `tmdbId` for
   the no-metadata fallback) and asserts. Cleanup in `afterEach`.
3. **Fallback UI (Story 5-11) coverage restored where seedable.** A no-metadata
   movie (`tmdb_id=0`, `parse_status=''`) renders ColorPlaceholder + FallbackFailed
   → `color-placeholder`, `failed state`, `search CTA` now run. The `pending`
   state is **not seedable** via the create endpoint (no `parse_status` field) →
   kept as a single honest `test.skip` with the precise reason + follow-up, NOT a
   runtime self-skip.
4. **`/metadata/apply` un-skipped (3 of 4).** The real blocker was misdiagnosed:
   `cmd/api/main.go` never calls `SetMediaUpdaters`, so the running backend takes
   the nil/no-op apply branch (returns success, no DB write). The 3 happy-path
   tests un-skip and assert the **request/response envelope contract** (seeded a
   real `media_id` for forward-compat). The `non-existent → NOT_FOUND` test stays
   skipped with the precise root cause (NOT_FOUND only fires when updaters are
   wired) + follow-up.
5. **Dead `api.spec.ts` deleted** (Rule 24 superseded-mechanism — the corollary
   added in Story 20-3). No unique coverage lost.
6. **No silent caps.** Everything that remains uncovered (pending state, NOT_FOUND,
   season accordion) is an explicit `test.skip` with a reason or a tracked
   sprint-status follow-up.

## Tasks / Subtasks

- [x] **T1** `seed-helpers.ts` — `seedMovie` / `seedSeries` / `deleteMovies` /
  `deleteSeries`; snake_case payloads, assert created.
- [x] **T2** Rewrite `media-detail.spec.ts` — seed-then-assert; per-describe
  `afterEach` cleanup; `pending` kept as honest skip.
- [x] **T3** `manual-search.api.spec.ts` — un-skip 3 apply tests (seed real
  media), keep NOT_FOUND skip with precise root cause + comment.
- [x] **T4** Delete `api.spec.ts` (3 dead `describe.skip`).
- [x] **T5** Local validation against the running backend.

## Verification (localhost, 2026-06-14)

- `manual-search.api.spec.ts` — **56 passed / 4 skipped / 0 failed** (apply
  movie/series/learnPattern run + pass; seeding creates + cleans real rows).
- `media-detail.spec.ts` (chromium, `new_shell_enabled=false` to mirror the CI
  legacy shell) — **12 passed / 1 skipped (pending) / 0 failed**. Previously these
  self-skipped on an empty DB.
- Flag flipped back to `true` after the run (Phase-2 manual V2 testing intact).

## Follow-ups (tracked in sprint-status)

- `story-20-4-season-accordion-e2e` — Tier-2 seed path that writes the `seasons`
  table (test-only seed endpoint or `POST /series` extension) → E2E the accordion
  + the `pending` parse-status fallback.
- `wire-set-media-updaters-test-harness` — wire `SetMediaUpdaters` so the
  `/metadata/apply` NOT_FOUND test and the real DB-mutation path become testable.
