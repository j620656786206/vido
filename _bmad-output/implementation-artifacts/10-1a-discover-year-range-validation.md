# Story 10.1a: Discover YearRange Input Validation

Status: review

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

- [x] Task 1: Error code + validation (AC: #1, #2, #5)
  - [x] 1.1 Add `ErrCodeInvalidYearRange = "INVALID_YEAR_RANGE"` to `apps/api/internal/tmdb/errors.go` alongside existing codes.
  - [x] 1.2 Add helper `NewInvalidYearRangeError()` returning `*TMDbError` with `StatusCode: 400`, message `"year_gte must be <= year_lte"` (mirror `NewBadRequestError` shape).
  - [x] 1.3 In `apps/api/internal/handlers/tmdb_handler.go::parseDiscoverParams`, after parsing `year_gte`/`year_lte`, validate: if both > 0 AND `year_gte > year_lte`, return the error.
  - [x] 1.4 Update `DiscoverMovies` and `DiscoverTVShows` handler methods to surface parse errors via existing `handleTMDbError` helper.

- [x] Task 2: Tests (AC: #1–6)
  - [x] 2.1 Handler test (movies): `year_gte=2030&year_lte=2020` → 400 + code `INVALID_YEAR_RANGE`.
  - [x] 2.2 Handler test (tv): `year_gte=2030&year_lte=2020` → 400 + code `INVALID_YEAR_RANGE`. (Separate test confirms both handlers route through the shared `parseDiscoverParams` validation — guards against future regressions where only one handler enforces it.)
  - [x] 2.3 Handler test: `year_gte=2024&year_lte=2024` → 200 (same-year boundary is valid).
  - [x] 2.4 Handler test: `year_gte=0&year_lte=2024` → 200 (zero-gte = unlimited lower bound).
  - [x] 2.5 Handler test: `year_gte=2024&year_lte=0` → 200 (zero-lte = unlimited upper bound).
  - [x] 2.6 Handler test: `year_gte=2024&year_lte=2025` → 200 (normal range, sanity baseline).
  - [x] 2.7 Errors package test (`errors_test.go`): `TestNewInvalidYearRangeError` asserts `Code == ErrCodeInvalidYearRange`, `StatusCode == 400`, and `Message` contains `"year_gte"`.

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

### Implementation Plan (2026-04-15)

- **Validation site:** single branch inside `parseDiscoverParams` after the existing `year_gte`/`year_lte` parse blocks. Condition: `p.YearGte > 0 && p.YearLte > 0 && p.YearGte > p.YearLte`. This is the only code location that needs to change to satisfy ACs #1, #2, #6 — the existing `strconv.Atoi(v)` guard already discards `year_gte=0` into `p.YearGte=0` (same for `year_lte`), so "unlimited" semantics fall out for free.
- **Signature change:** `parseDiscoverParams` returns `(tmdb.DiscoverParams, error)` instead of `tmdb.DiscoverParams`. Both callers (`DiscoverMovies`, `DiscoverTVShows`) check the error and call the existing `handleTMDbError(c, err, ...)` helper, which translates `*tmdb.TMDbError`'s `StatusCode: 400` + `Code: INVALID_YEAR_RANGE` into the `ApiResponse<T>` error envelope (AC #5). No new error-translation plumbing added.
- **Service / client / cache layers untouched** — enforces AC #6 structurally (not by convention): validation cannot be duplicated lower in the stack because the lower layers never see the request.
- **Test structure:** 1 new table-driven test in `tmdb_handler_test.go` covering movies (ACs #1, #3, #4, #5, #6 on the movies path); 1 dedicated test for the TV handler (AC #2 — kept separate by design so a regression wiring only one handler through the error return cannot silently pass); 1 new constructor test in `errors_test.go` (Task 2.7).

### UX Verification

🎨 UX Verification: SKIPPED — no UI changes in this story (backend-only: error code constant + handler-layer validation).

### Completion Notes

- **AC coverage:** All 6 ACs satisfied by the 7-case test matrix (2.1 + 2.3–2.6 table-driven on movies; 2.2 standalone on TV; 2.7 constructor assertion on errors package).
- **Architecture compliance:** Validation lives exclusively in `handlers/tmdb_handler.go::parseDiscoverParams` (Rule: Handler → Service → Repository). The `tmdb.TMDbError` wire format was reused (Rule 7 error-code shape + Rule 3 `ApiResponse<T>` envelope via existing `handleTMDbError` branch).
- **Zero-value semantics preserved:** The existing `strconv.Atoi(v)` + `n > 0` gate from Story 10-1 already clamps negatives/zeros to `p.YearGte = 0` / `p.YearLte = 0`; the new validation branch then short-circuits when either is zero. No behavioral change for existing "unlimited upper/lower bound" callers (AC #3).
- **Full regression gate (Epic 9 Retro AI-1):** `pnpm nx test api` → PASS (Go backend). `pnpm nx test web` → 1629/1629 PASS (React frontend). `pnpm lint:all` → 0 ESLint errors (108 pre-existing warnings unrelated to this change), `go vet` / `staticcheck@2026.1` / `prettier --check` all green.
- **No pre-existing failures encountered** — both test suites passed cleanly, so no `preexisting-fail-*` backlog entries added.
- **No orphaned test processes** after test runs (verified via `pgrep`).

### File List

- `apps/api/internal/tmdb/errors.go` — added `ErrCodeInvalidYearRange` constant + `NewInvalidYearRangeError()` constructor
- `apps/api/internal/tmdb/errors_test.go` — added `TestNewInvalidYearRangeError` (Task 2.7)
- `apps/api/internal/handlers/tmdb_handler.go` — `parseDiscoverParams` now returns `(DiscoverParams, error)`; `DiscoverMovies` and `DiscoverTVShows` thread the parse error through `handleTMDbError`
- `apps/api/internal/handlers/tmdb_handler_test.go` — added `TestTMDbHandler_DiscoverMovies_YearRangeValidation` (table-driven, 5 cases: Tasks 2.1, 2.3, 2.4, 2.5, 2.6) + `TestTMDbHandler_DiscoverTVShows_YearRangeValidation_Reversed` (Task 2.2)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — status transitions: `ready-for-dev` → `in-progress` → `review`

### Change Log

| Date       | Change |
|------------|--------|
| 2026-04-14 | Story drafted in party-mode follow-up to Story 10-1 review. Decision: option (A) 400 + `INVALID_YEAR_RANGE` over (B) auto-swap / (C) passthrough. Validation restricted to handler layer. Zero-value "unlimited" semantics preserved from Story 10-1. |
| 2026-04-15 | **Task 1 complete** — `ErrCodeInvalidYearRange` constant + `NewInvalidYearRangeError()` added to `tmdb/errors.go`; `parseDiscoverParams` signature changed to `(DiscoverParams, error)` with the 3-condition guard (`YearGte > 0 && YearLte > 0 && YearGte > YearLte`); both `DiscoverMovies` and `DiscoverTVShows` thread the parse error through `handleTMDbError`. |
| 2026-04-15 | **Task 2 complete** — 7 new test cases: `TestNewInvalidYearRangeError` (errors package, 2.7) + table-driven `TestTMDbHandler_DiscoverMovies_YearRangeValidation` (2.1, 2.3-2.6) + dedicated `TestTMDbHandler_DiscoverTVShows_YearRangeValidation_Reversed` (2.2, kept separate by design). All 6 ACs covered; full regression suite PASS; lint:all PASS; Status → `review`. |
