# Story 10.1a: Discover YearRange Input Validation

Status: ready-for-dev

## Story

As a Vido backend API consumer (web client or deeplink),
I want TMDb discover endpoints to reject invalid year ranges with HTTP 400,
so that reversed year filters surface as explicit errors instead of being silently
transparent to TMDb (where behavior is undefined) or being cached as empty results.

## Acceptance Criteria

1. Given `GET /api/v1/tmdb/discover/movies` with `year_gte > year_lte` (both non-zero), when the handler parses query params, then the response is HTTP 400 with error code `INVALID_YEAR_RANGE`.
2. Given `GET /api/v1/tmdb/discover/tv` with `year_gte > year_lte` (both non-zero), when the handler parses query params, then the response is HTTP 400 with error code `INVALID_YEAR_RANGE`.
3. Given `year_gte=0` or `year_lte=0` (either or both), when the handler parses query params, then validation is SKIPPED and the request proceeds — zero retains the existing "unlimited" semantics from Story 10-1.
4. Given `year_gte == year_lte` (same year, both non-zero), when the handler parses query params, then the request proceeds — same-year is valid (maps to single calendar year).
5. Given the 400 response, when the client inspects it, then the body follows the existing `ApiResponse<T>` envelope: `{success: false, error: {code: "INVALID_YEAR_RANGE", message: "year_gte must be <= year_lte"}}`.
6. Given the validation, when it fires, then it runs in the HTTP handler layer (`parseDiscoverParams`) — NOT in TMDb client, service, or cache layers. Client/service remain permissive for internal callers.

## Tasks / Subtasks

- [ ] Task 1: Error code + validation (AC: #1, #2, #5)
  - [ ] 1.1 Add `ErrCodeInvalidYearRange = "INVALID_YEAR_RANGE"` to `apps/api/internal/tmdb/errors.go` alongside existing codes.
  - [ ] 1.2 Add helper `NewInvalidYearRangeError()` returning `*TMDbError` with `StatusCode: 400`, message `"year_gte must be <= year_lte"` (mirror `NewBadRequestError` shape).
  - [ ] 1.3 In `apps/api/internal/handlers/tmdb_handler.go::parseDiscoverParams`, after parsing `year_gte`/`year_lte`, validate: if both > 0 AND `year_gte > year_lte`, return the error.
  - [ ] 1.4 Update `DiscoverMovies` and `DiscoverTVShows` handler methods to surface parse errors via existing `handleTMDbError` helper.

- [ ] Task 2: Tests (AC: #1–6)
  - [ ] 2.1 Handler test (movies): `year_gte=2030&year_lte=2020` → 400 + code `INVALID_YEAR_RANGE`.
  - [ ] 2.2 Handler test (tv): `year_gte=2030&year_lte=2020` → 400 + code `INVALID_YEAR_RANGE`. (Separate test confirms both handlers route through the shared `parseDiscoverParams` validation — guards against future regressions where only one handler enforces it.)
  - [ ] 2.3 Handler test: `year_gte=2024&year_lte=2024` → 200 (same-year boundary is valid).
  - [ ] 2.4 Handler test: `year_gte=0&year_lte=2024` → 200 (zero-gte = unlimited lower bound).
  - [ ] 2.5 Handler test: `year_gte=2024&year_lte=0` → 200 (zero-lte = unlimited upper bound).
  - [ ] 2.6 Handler test: `year_gte=2024&year_lte=2025` → 200 (normal range, sanity baseline).
  - [ ] 2.7 Errors package test (`errors_test.go`): `TestNewInvalidYearRangeError` asserts `Code == ErrCodeInvalidYearRange`, `StatusCode == 400`, and `Message` contains `"year_gte"`.

## Dev Notes

### Architecture Compliance

- **Validation location:** Handler layer ONLY — keeps TMDb client / CacheService / TMDbService reusable by internal code paths that trust their inputs. This is an explicit AC (#6), not an implementation detail, to prevent future PRs from pushing validation down into lower layers.
- **Error wrapping:** Use existing `*TMDbError` pattern. Story 10-1 left `handleTMDbError` as the translation path; do NOT introduce a parallel error type.
- **Zero-value semantics:** Preserve. `DiscoverParams.YearGte == 0` already means "omit `primary_release_date.gte`" per Story 10-1. Validation MUST skip when either field is zero.
- **No client / service / cache changes** — enforced by the fact that all new tests live in `handlers/` and `tmdb/errors_test.go`.

### Project Structure Notes

- Modified files:
  - `apps/api/internal/tmdb/errors.go` — add error code constant + constructor
  - `apps/api/internal/tmdb/errors_test.go` — new constructor test
  - `apps/api/internal/handlers/tmdb_handler.go` — add validation to `parseDiscoverParams`
  - `apps/api/internal/handlers/tmdb_handler_test.go` — 6 new handler cases (table-driven recommended)
- New files: none

### References

- [Source: apps/api/internal/handlers/tmdb_handler.go] — `parseDiscoverParams` (Story 10-1 addition)
- [Source: apps/api/internal/tmdb/errors.go] — `TMDbError` + `NewBadRequestError` pattern
- [Source: _bmad-output/implementation-artifacts/10-1-tmdb-trending-discover-api.md] — Parent story, AC #2 (Discover endpoint query-param parsing)
- [Source: _bmad-output/planning-artifacts/epics/epic-10-homepage-tv-wall.md] — Epic 10 context

## Dev Agent Record

### Estimated Effort

- ~15 minutes implementation + tests (per Amelia's estimate in party-mode session 2026-04-14).
- Does NOT block Story 10-2 (Hero Banner) or downstream Discover consumers.

### Change Log

| Date       | Change |
|------------|--------|
| 2026-04-14 | Story drafted in party-mode follow-up to Story 10-1 review. Decision: option (A) 400 + `INVALID_YEAR_RANGE` over (B) auto-swap / (C) passthrough. Validation restricted to handler layer. Zero-value "unlimited" semantics preserved from Story 10-1. |
