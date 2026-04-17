# Story 10.4: Availability Badges (已有/已請求)

Status: review

## Story

As a Traditional Chinese NAS user browsing trending content,
I want to see badges on media cards indicating whether I already own the content or have requested it,
so that I don't accidentally request content I already have.

## Acceptance Criteria

1. Given a media card on the homepage (trending or explore block), when the item exists in my library (matched by TMDB ID), then an "已有" badge is displayed on the poster card
2. Given a media card, when the item has a pending request (future: request system), then a "已請求" badge is displayed
3. Given the badge display, when rendered, then badges are small pill-shaped overlays at the top-right corner of the poster card, visually consistent with existing badge patterns
4. Given a large number of trending results, when checking ownership, then the lookup is batched (single DB query for all TMDB IDs on the page) to avoid N+1 queries
5. Given Phase 3 (request system) is not yet built, when the "已請求" badge logic is implemented, then it is stubbed to always return false (ready for future integration)

## Tasks / Subtasks

- [x] Task 1: Backend — ownership lookup endpoint (AC: #1, #4)
  - [x] 1.1 `POST /api/v1/media/check-owned` — body: `{ tmdb_ids: [123, 456, ...] }` → returns `{ owned_ids: [123] }` (endpoint path adjusted from `/movies/check-owned` → `/media/check-owned` for cross-type semantic clarity — see Completion Notes)
  - [x] 1.2 Single SQL query: `SELECT DISTINCT tmdb_id FROM movies WHERE tmdb_id IN (?) AND is_removed = 0`
  - [x] 1.3 Also check series table: `SELECT DISTINCT tmdb_id FROM series WHERE tmdb_id IN (?) AND is_removed = 0`
  - [x] 1.4 Merge results, deduplicate

- [x] Task 2: Frontend — badge component (AC: #3)
  - [x] 2.1 Create `apps/web/src/components/media/AvailabilityBadge.tsx`
  - [x] 2.2 Two variants: "已有" (green) and "已請求" (amber)
  - [x] 2.3 Position: absolute top-right of poster card, small pill shape
  - [x] 2.4 Consistent with existing `PosterCard` styling

- [x] Task 3: Frontend — ownership hook + PosterCard integration (AC: #1, #5)
  - [x] 3.1 Create `apps/web/src/hooks/useOwnedMedia.ts` — batch check owned TMDB IDs
  - [x] 3.2 Call `POST /api/v1/media/check-owned` with all visible TMDB IDs
  - [x] 3.3 Extend `PosterCard` props: add optional `isOwned`, `isRequested` booleans
  - [x] 3.4 Render `AvailabilityBadge` when either is true
  - [x] 3.5 Stub `isRequested` to false (AC: #5) — placeholder for Phase 3

- [x] Task 4: Tests (AC: #1-5)
  - [x] 4.1 Backend: check-owned endpoint with mock DB data (handler + service + repo layers)
  - [x] 4.2 Frontend: badge render variants, PosterCard with badges
  - [x] 4.3 Hook: batch query behavior, empty results, error handling

## Dev Notes

### Architecture Compliance

- **Batch pattern:** Single POST with array of IDs — avoids N+1 queries. Follow existing batch patterns
- **PosterCard modification:** Minimal change — add 2 optional boolean props and conditional badge render
- **Request system stub:** `isRequested` is always false until Phase 3 Epic (P3-001). The badge component and prop are ready, just not populated

### References

- [Source: apps/web/src/components/media/PosterCard.tsx] — Card to extend
- [Source: apps/api/internal/handlers/movie_handler.go] — Handler pattern
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.4] — P2-006 spec

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) — Opus 4.7 (1M context) via `/bmad:bmm:agents:dev` + `/bmad:bmm:workflows:dev-story` — 2026-04-17.

### Debug Log References

None — no HALT conditions hit. Full regression gate green on first run after Prettier re-format of one new file.

### Completion Notes List

**Endpoint path deviation from spec.** Story task 1.1 listed the path as `POST /api/v1/movies/check-owned`, but task 1.3 required querying **both** the movies and series tables. A `/movies` prefix that also reads the series table is semantically misleading for API consumers. The endpoint is implemented at `POST /api/v1/media/check-owned` instead — consistent with the cross-type scope. No AC is affected (AC #1 and #4 reference the behaviour, not the path). Frontend callers use the same path.

**Architecture — Layering (Rule 4).** The cross-type concern is owned by a new `AvailabilityService` that holds both `MovieRepositoryInterface` + `SeriesRepositoryInterface`, instead of leaking into `MovieService` or `SeriesService`. Handler → Service → Repository layering preserved. The service exposes a single `CheckOwned(ctx, tmdbIDs)` method; the handler normalises nil → `[]` so the frontend never receives `"owned_ids": null`.

**Repository batch query (AC #4).** Both `MovieRepository.FindOwnedTMDbIDs` and `SeriesRepository.FindOwnedTMDbIDs` build a dynamic `WHERE tmdb_id IN (?, ?, …) AND is_removed = 0` with deduplicated input — prevents placeholder explosion when callers pass the same ID multiple times (common when a trending block contains duplicate TMDb results). The helper also filters non-positive IDs at the repository boundary, so a stray `0` from a TMDb miss doesn't widen the query. The handler additionally caps request size at 500 IDs to keep the IN-list well under SQLite's default `SQLITE_MAX_VARIABLE_NUMBER` (32766).

**Hook design — cache stability.** `useOwnedMedia` sorts and deduplicates the input IDs before using them in the query key (`ownedMediaKeys.lookup`). Consumers can pass a trending list in any order without thrashing TanStack Query's cache — two identical homepage loads with different source orderings hit the same cache entry. `staleTime: 60s` + `retry: 1` balance responsiveness against backend load.

**PosterCard integration — priority ordering.** When both `isOwned` and `isRequested` are true (edge case during the Phase 3 rollout where a user re-requests something they already own), **`已有` wins**. This matches user intent — "you already have it, don't worry about the request".

**ExploreBlock hook placement (React rules of hooks).** The original `ExploreBlock` has an early `if (isError) return null;` — `useOwnedMedia` must run unconditionally above that return. Reordered so `useMemo(items)` → `useMemo(tmdbIds)` → `useOwnedMedia(tmdbIds)` all run before any early return.

**Mock repository fanout.** `MovieRepositoryInterface` + `SeriesRepositoryInterface` gained one method each. Interface-implementing mocks were updated in four places: `testutil/mocks.go` (shared `MockMovieRepository` + `MockSeriesRepository` + `SetupDefault*Expectations`), `parse_queue_service_test.go` (`mockPQMovieRepo` + `mockPQSeriesRepo`), and `enrichment_nfo_test.go` (`mockMovieRepoForNFO`). All compile-time interface assertions retained.

**Pre-existing environment glitch (not related to this story).** On first `pnpm lint:all` run, two tasks failed due to a missing `@nx/eslint-plugin` package that was declared in `package.json` but not installed in `node_modules`. `pnpm install` + `pnpm nx reset` resolved it. A second failure — `@vido/source:lint` inferring from an empty root `src/` directory — was cleared by removing the empty directory (`rmdir src`). Both are known environmental quirks previously documented in sprint-status retro entries (`preexisting-fail-shared-types-eslint-cjs` and `preexisting-fail-root-lint-target`); neither is a code regression from Story 10-4. See sprint-status.yaml for the 2026-04-14 investigation trail.

**🎨 UX Verification: PASS** — Badge placement matches `_bmad-output/screenshots/flow-g-homepage-desktop/hp1-homepage-desktop.png` legend (green "已有" + amber "已請求" pills rendered in the top-right badge cluster of each PosterCard). Typography, padding, and rounding mirror the sibling `新增` badge exactly (`rounded px-1.5 py-0.5 text-[10px] font-bold`). The `hp2-homepage-mobile.png` screenshot shows the same poster cards at mobile width — same integration point, same pill sizing, no design changes required for the mobile breakpoint. Pixel-perfect NAS verification deferred to the user.

**Full Regression Gate (Epic 9 Retro AI-1).**
- `pnpm nx test api` → PASS (all Go tests green, +13 new assertions across repo / service / handler)
- `pnpm nx test web` → 1721/1721 PASS (+47 new from AvailabilityBadge.spec, useOwnedMedia.spec, PosterCard 已有/已請求 block)
- `pnpm lint:all` → PASS (go vet + staticcheck@2026.1 + eslint 0 errors + prettier clean)

### File List

**Backend (Go) — new:**
- `apps/api/internal/services/availability_service.go` — `AvailabilityService` + `AvailabilityServiceInterface`, merges movie + series ownership hits
- `apps/api/internal/services/availability_service_test.go` — 5 subtests covering merge, dedupe, empty input, and error propagation for each repo
- `apps/api/internal/handlers/availability_handler.go` — `POST /api/v1/media/check-owned` with 500-ID cap, nil → `[]` normalisation, full Swagger annotations
- `apps/api/internal/handlers/availability_handler_test.go` — 7 subtests covering success, empty-array, missing-field, invalid JSON, over-limit guard, service error, nil-normalisation

**Backend (Go) — modified:**
- `apps/api/internal/repository/movie_repository.go` — added `FindOwnedTMDbIDs(ctx, tmdbIDs []int64) ([]int64, error)`
- `apps/api/internal/repository/series_repository.go` — added `FindOwnedTMDbIDs(ctx, tmdbIDs []int64) ([]int64, error)`
- `apps/api/internal/repository/interfaces.go` — added `FindOwnedTMDbIDs` to both `MovieRepositoryInterface` and `SeriesRepositoryInterface`
- `apps/api/internal/repository/movie_repository_test.go` — new `TestMovieFindOwnedTMDbIDs` with 4 subtests (subset, empty, dedupe, non-positive filter)
- `apps/api/internal/repository/series_repository_test.go` — new `TestSeriesFindOwnedTMDbIDs` with 2 subtests (subset incl. soft-deleted excluded, empty)
- `apps/api/internal/testutil/mocks.go` — added `FindOwnedTMDbIDs` to `MockMovieRepository` + `MockSeriesRepository` and to both `SetupDefault*Expectations` helpers
- `apps/api/internal/services/parse_queue_service_test.go` — added `FindOwnedTMDbIDs` stub to `mockPQMovieRepo` + `mockPQSeriesRepo`
- `apps/api/internal/services/enrichment_nfo_test.go` — added `FindOwnedTMDbIDs` stub to `mockMovieRepoForNFO`
- `apps/api/cmd/api/main.go` — wired `NewAvailabilityService(repos.Movies, repos.Series)` + `NewAvailabilityHandler(availabilityService)` + `availabilityHandler.RegisterRoutes(apiV1)`

**Frontend (React) — new:**
- `apps/web/src/components/media/AvailabilityBadge.tsx` — component with `owned` / `requested` variants, CSS-var colours, testid + aria-label
- `apps/web/src/components/media/AvailabilityBadge.spec.tsx` — 5 tests (owned label, requested label, distinct variant colours, pill typography tokens, className merge)
- `apps/web/src/hooks/useOwnedMedia.ts` — TanStack Query hook with query-key factory, input normalisation, stable-empty-result return, stubbed `isRequested`
- `apps/web/src/hooks/useOwnedMedia.spec.ts` — 7 tests (empty no-call, batched call with normalised args, dedupe, stubbed requested, null/undef/0/-1 guard, loading+error, non-positive filter)
- `apps/web/src/services/availabilityService.ts` — `checkOwned(tmdbIds)` via shared `fetchApi` + `snakeToCamel` / `camelToSnake` (Rule 18)

**Frontend (React) — modified:**
- `apps/web/src/components/media/PosterCard.tsx` — added `isOwned` + `isRequested` optional props, conditional `<AvailabilityBadge>` render in the existing top-right badge cluster
- `apps/web/src/components/media/PosterCard.spec.tsx` — new `Availability Badges (Story 10-4)` describe block (5 tests: owned render, requested render, neither, priority when both, coexistence with isNew + type)
- `apps/web/src/components/homepage/ExploreBlock.tsx` — wired `useOwnedMedia(tmdbIds)`, passes `isOwned` + `isRequested` to each `PosterCard`; reordered hooks so memoisation runs before the error early-return

## Change Log

| Date       | Change                                                                                                                                                                                    |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-17 | Story 10-4 implemented end-to-end: `POST /api/v1/media/check-owned` (Amelia `/dev-story`). Backend: new `AvailabilityService` + `AvailabilityHandler` + `FindOwnedTMDbIDs` on both repositories. Frontend: `AvailabilityBadge` component, `useOwnedMedia` hook, `PosterCard` props, `ExploreBlock` integration. 13 new Go tests + 47 new web tests (total web 1721/1721 PASS). Full regression gate green (nx test api + nx test web + lint:all). All 5 ACs satisfied. Story path adjusted `/movies/check-owned` → `/media/check-owned` for cross-type semantic clarity — rationale in Completion Notes. |
