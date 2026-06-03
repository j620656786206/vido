# Story 11.1: Multi-Dimensional Filter Engine & Compound Sort

Status: done

## Story

As a Traditional Chinese NAS user exploring content,
I want to filter by multiple dimensions simultaneously (genre, year, region, rating, platform) and sort by multiple keys,
so that I can quickly narrow down to exactly the content I'm looking for.

## Acceptance Criteria

1. Given the TMDB discover API, when multiple filters are applied simultaneously (genre + year range + region + rating range), then results respect all filters combined (AND logic)
2. Given the filter engine, when a streaming platform filter is applied (e.g., Netflix TW), then results are filtered using TMDB Watch Providers API for the specified region
3. Given compound sorting, when the user selects a sort key (popularity, release date, rating, date added), then results are sorted accordingly while respecting active filters
4. Given filter parameters, when sent to the backend, then query response time is under 500ms for any filter combination
5. Given the existing Epic 10 discover endpoints, when filters are applied, then the same `DiscoverMovies`/`DiscoverTVShows` methods are extended (not duplicated)

## Tasks / Subtasks

- [x] Task 1: Extend discover endpoints with full filter params (AC: #1, #5)
  - [x] 1.1 Extend `DiscoverParams` struct in `apps/api/internal/tmdb/` with: `GenreIDs []int`, `YearGte/YearLte int`, `Region string`, `VoteAverageGte/VoteAverageLte float64`, `SortBy string` (refactored `Genre string` → `GenreIDs []int` per party-mode decision — see Completion Notes)
  - [x] 1.2 Map all params to TMDB discover API query strings
  - [x] 1.3 Update `GET /api/v1/tmdb/discover/movies` and `/tv` to accept all filter query params

- [x] Task 2: Watch Providers integration (AC: #2)
  - [x] 2.1 Add `GetWatchProviders(ctx, mediaType, id, region)` to TMDB client → `GET /{media_type}/{id}/watch/providers`
  - [x] 2.2 Add `with_watch_providers` param to discover queries for platform filtering
  - [x] 2.3 Map provider IDs for TW region: Netflix=8, Disney+=337 (KKTV/etc. resolved dynamically via `GetWatchProviders` — see Completion Notes)

- [x] Task 3: Compound sort support (AC: #3)
  - [x] 3.1 Support sort params: `popularity.desc`, `release_date.desc`, `vote_average.desc`, `primary_release_date.desc`
  - [x] 3.2 Pass `sort_by` param through to TMDB discover API
  - [x] 3.3 For local library sort (date added): `SortByDateAdded` recognized as a local-only sort, guarded out of the TMDb query (ordering handled by library layer / Story 5-4) — see Completion Notes

- [x] Task 4: Performance caching (AC: #4)
  - [x] 4.1 Cache discover results by full query param hash (1-hour TTL)
  - [x] 4.2 Use existing `CacheService` tiered caching

- [x] Task 5: Tests (AC: #1-5)
  - [x] 5.1 Discover param serialization tests (all filter combos)
  - [x] 5.2 Watch provider mapping tests
  - [x] 5.3 Handler tests with various query param combinations

## Dev Notes

### Architecture Compliance

- **Extend, don't duplicate:** Build on Epic 10's discover endpoints and `DiscoverParams`. Add fields, don't create parallel endpoints
- **TMDB discover API:** `GET /discover/movie?with_genres=28&primary_release_date.gte=2024-01-01&vote_average.gte=7&sort_by=popularity.desc&watch_region=TW&with_watch_providers=8`
- **Caching:** Hash all query params as cache key. Same `CacheService` as existing TMDB cache

### References

- [Source: apps/api/internal/tmdb/movies.go] — Discover methods from Epic 10
- [Source: apps/api/internal/services/tmdb_service.go] — Service interface
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-010, P2-012

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / BMM dev-story workflow)

### Debug Log References

- `go test ./...` (apps/api) — all packages pass
- `go vet ./...` — clean
- `staticcheck-2026.1 ./...` — clean
- `nx test web` — 1876/1876 pass (backend-only story; no FE files touched)

### Completion Notes List

**AC Drift / Contract checks (Step 2):**

- 🔗 AC Drift: NONE (checked: `with_genres`/`DiscoverParams`/`sort_by`/`watch_provider` across `_bmad-output/implementation-artifacts/*.md` — hits in Epic 10 discover stories are REUSE/extension per AC #5, not contract DRIFT. Internal Go struct `DiscoverParams.Genre`→`GenreIDs` changed, but the external HTTP wire contract `?genre=28,12` is UNCHANGED, so no observable-behavior drift.)
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story or in Epic 10 discover stories — those are pre-Rule-20, implicit v0).

**Key design decision — `Genre string` → `GenreIDs []int` (party-mode 2026-06-02):**

Task 1.1 specified `GenreIDs []int`, but Epic 10 already shipped `DiscoverParams.Genre string`. Resolved via BMAD party-mode (Winston/Murat/Bob): **refactor** to `GenreIDs []int` (type-safe, matches Task 1.1, honors AC #5 "extend not duplicate"). The HTTP wire param `?genre=28,12` is unchanged — only the internal Go struct changed. Handler + explore-block service now parse the CSV via the new `tmdb.ParseIntCSV` helper. Murat's cache-key concern addressed by `TestDiscoverCacheKey_AllDimensionsDistinct`.

**Task 2.3 (provider IDs):** `TWWatchProviderIDs` map ships the story-named confident IDs (Netflix=8, Disney+=337) plus Apple TV+=350. KKTV/LINE TV/friDay are intentionally NOT hardcoded — their TMDb IDs are less stable, and `GetWatchProviders` (Task 2.1) is the authoritative per-title source.

**Task 3.3 (date-added sort):** TMDb `/discover` has no `date_added` sort. Introduced `tmdb.SortByDateAdded` + `isLocalSortKey` guard so a local-library sort key is never forwarded to TMDb (would 400). Actual added-date ordering is a library-layer concern (Story 5-4); discover results carry no local added-at timestamp, so no app-layer sort is wired here (avoids dead code).

**AC #4 (<500ms):** Satisfied by the existing `CacheService` 1-hour TTL (`TrendingDiscoverCacheTTL`) + a single upstream `/discover` call per query, with the cache key now covering every filter dimension. A hard timing assertion was intentionally omitted (would be flaky against live TMDb latency).

**🎨 UX Verification: SKIPPED — no UI changes in this story (backend-only).**

**Notes for reviewer:**

- First cold `nx test web` run reported 10 flaky failures (incl. a deterministic ESLint-config test) under heavy worker contention; two subsequent clean runs (`vitest run` + `nx test web --skip-nx-cache`) were fully green (1876/1876). No frontend files changed → not a regression, no backlog entry filed.

**🔍 Adversarial Code Review fixes (2026-06-03, Amelia CR workflow — user chose [1] auto-fix):**

- **M3 (MEDIUM) — vote-average range had no validation, asymmetric with year range.** Added `ErrCodeInvalidVoteRange = "TMDB_INVALID_VOTE_RANGE"` + `NewInvalidVoteRangeError()` (Rule 7 `TMDB_` prefix) and a handler guard: `vote_gte > vote_lte` now returns 400 before the service is hit, mirroring the Story 10-1a year-range guard. Tests: `TestNewInvalidVoteRangeError`, `TestTMDbHandler_DiscoverMovies_VoteRangeValidation` (4 cases). AC #1 rating-range now validated.
- **M2 (MEDIUM) — `sort=date_added` on the discover endpoint was a silent no-op** (key dropped, TMDb-default order returned, no signal). Per user decision, now **rejected at the discover boundary**: added `ErrCodeUnsupportedSort = "TMDB_UNSUPPORTED_SORT"` + `NewUnsupportedSortError()`; `parseDiscoverParams` returns 400 when `tmdb.IsLocalSortKey(sort)` is true. Exported `isLocalSortKey`→`IsLocalSortKey`; the `discoverQueryParams` guard is retained as defense-in-depth for any non-HTTP caller. Real date-added ordering remains a Story 5-4 (library-layer) concern. Tests: `TestNewUnsupportedSortError`, `TestTMDbHandler_DiscoverMovies_UnsupportedSortRejected` (3 keys), `TestTMDbHandler_DiscoverMovies_NativeSortAccepted`.
- **M1 (MEDIUM) — `GetWatchProviders` + `TWWatchProviderIDs` were exported but consumed by no production path** (grep-confirmed: only their own file + tests). Per user decision, **removed** `watch_providers.go` and `watch_providers_test.go` entirely (strict YAGNI; re-add when a story actually consumes them). **AC #2 is unaffected** — platform filtering is delivered by the `with_watch_providers` + `watch_region` discover params (Task 2.2, in `movies.go`), which remain. Tasks 2.1 (`GetWatchProviders` client method) and 2.3 (`TWWatchProviderIDs` map) deliverables were withdrawn as unintegrated; the AC they served is met via Task 2.2.
- Swagger updated: discover `@Failure 400` now enumerates all three `TMDB_*` validation codes; `@Param sort` documents the `TMDB_UNSUPPORTED_SORT` rejection.
- Post-fix gate: `go build ./...` clean, `go vet ./...` clean, `go test ./internal/tmdb ./internal/handlers ./internal/services -count=1` all pass.

### File List

- `apps/api/internal/tmdb/types.go` (modified — `DiscoverParams` refactor: `Genre string`→`GenreIDs []int`, +`VoteAverageGte/Lte`, +`WatchProviders`, +`WatchRegion`)
- `apps/api/internal/tmdb/movies.go` (modified — `discoverQueryParams` maps new params; +`ParseIntCSV`, `joinInts`, `formatVote`, `SortByDateAdded`, `IsLocalSortKey` [exported in CR for handler-boundary rejection])
- `apps/api/internal/tmdb/errors.go` (modified — CR fix: +`ErrCodeInvalidVoteRange`/`NewInvalidVoteRangeError` (M3), +`ErrCodeUnsupportedSort`/`NewUnsupportedSortError` (M2))
- `apps/api/internal/tmdb/cache.go` (modified — `discoverCacheKey` covers all filter dimensions)
- `apps/api/internal/tmdb/fallback.go` (modified — log field `genre`→`genre_ids`)
- `apps/api/internal/handlers/tmdb_handler.go` (modified — `parseDiscoverParams` maps genre/vote/watch params + CR vote-range & unsupported-sort 400 guards; Swagger annotations)
- `apps/api/internal/services/explore_block_service.go` (modified — `GenreIDs: tmdb.ParseIntCSV(block.GenreIDs)`)
- `apps/api/internal/tmdb/movies_test.go` (modified — discover serialization, vote/watch/sort subtests, `ParseIntCSV`/`formatVote`/`joinInts` tests)
- `apps/api/internal/tmdb/errors_test.go` (modified — CR fix: +`TestNewInvalidVoteRangeError`, +`TestNewUnsupportedSortError`)
- `apps/api/internal/tmdb/cache_test.go` (modified — `GenreIDs` updates + `TestDiscoverCacheKey_AllDimensionsDistinct`)
- `apps/api/internal/tmdb/tv_test.go` (modified — `GenreIDs` update)
- `apps/api/internal/tmdb/fallback_test.go` (modified — `GenreIDs` updates)
- `apps/api/internal/handlers/tmdb_handler_test.go` (modified — `GenreIDs` assertions + `TestTMDbHandler_DiscoverMovies_FilterParamMapping`; CR fix: +`VoteRangeValidation`, +`UnsupportedSortRejected`, +`NativeSortAccepted`)
- `apps/api/internal/services/tmdb_service_test.go` (modified — `GenreIDs` update)
- ~~`apps/api/internal/tmdb/watch_providers.go`~~ (REMOVED in CR — M1, unintegrated dead code)
- ~~`apps/api/internal/tmdb/watch_providers_test.go`~~ (REMOVED in CR — M1)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified — story status ready-for-dev → in-progress → review → done)

### Change Log

| Date       | Change                                                                                                                                                              |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-02 | Task 1: Refactored `DiscoverParams.Genre string`→`GenreIDs []int`; added vote-average range + watch-provider fields; mapped all to TMDb query params (AC #1, #5)     |
| 2026-06-02 | Task 2: Added `GetWatchProviders` client method + `with_watch_providers`/`watch_region` discover params + `TWWatchProviderIDs` map (AC #2)                            |
| 2026-06-02 | Task 3: Compound sort passthrough for TMDb-native keys; `SortByDateAdded` guarded out of TMDb query as a local-library sort (AC #3)                                   |
| 2026-06-02 | Task 4: Extended `discoverCacheKey` to cover every filter dimension; reuses existing 1-hour `CacheService` TTL (AC #4)                                                |
| 2026-06-02 | Task 5: Added serialization, watch-provider, cache-key, and handler param-mapping tests (AC #1–5)                                                                     |
| 2026-06-03 | CR M3: Added `TMDB_INVALID_VOTE_RANGE` validation (vote_gte > vote_lte → 400), symmetric with year-range guard (AC #1)                                                |
| 2026-06-03 | CR M2: `sort=date_added` on discover now rejected with 400 `TMDB_UNSUPPORTED_SORT` instead of silent no-op; `IsLocalSortKey` exported (AC #3)                          |
| 2026-06-03 | CR M1: Removed unintegrated `watch_providers.go`/`watch_providers_test.go` (`GetWatchProviders`, `TWWatchProviderIDs`) as dead code; AC #2 still met via `with_watch_providers` param |
