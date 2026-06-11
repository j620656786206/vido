# Story 12.1: Dual Rating Display — TMDB + Douban Side-by-Side

Status: done

## Story

As a media library user,
I want to see both TMDB and Douban ratings side-by-side on the media detail page,
so that I get a more complete quality signal from both Western and Chinese-language rating communities.

## Acceptance Criteria

1. **Given** a movie/series with a `tmdb_id`, **when** the detail page loads, **then** the TMDB rating (`vote_average`) and vote count are displayed.
2. **Given** a movie/series that has Douban data cached (via `douban_cache` table), **when** the detail page loads, **then** the Douban rating and vote count are displayed alongside the TMDB rating.
3. **Given** a movie/series with `tmdb_id > 0` but no Douban data yet, **when** the detail page loads, **then** a background enrichment job is triggered to look up the Douban rating by title+year, and the UI shows a loading skeleton in the Douban rating slot.
4. **Given** the Douban scraper is unavailable or returns no results, **when** enrichment is attempted, **then** only the TMDB rating is shown (graceful degradation), no error is displayed to the user, and the failure is logged with `slog.Warn`.
5. **Given** Douban data is fetched successfully, **when** the enrichment completes, **then** `douban_id`, `douban_rating`, and `douban_vote_count` are persisted to the movie/series record for future loads.
6. **Given** both ratings are displayed, **then** each rating shows: source logo/label, numeric score (0-10 scale, 1 decimal), and vote count (abbreviated: 1.2k, 15k).
7. **Given** a mobile viewport, **when** viewing the detail page, **then** the dual ratings stack vertically and remain readable.

## Tasks / Subtasks

### Backend

- [x] **Task 1: DB Migration 024 — Add Douban rating columns** (AC: #5) — ⚠️ number corrected 022→024 (022/023 already taken)
  - [x] 1.1 Create `024_add_douban_rating_fields.go` in `apps/api/internal/database/migrations/`
  - [x] 1.2 Add columns to `movies` table: `douban_id TEXT`, `douban_rating REAL`, `douban_vote_count INTEGER`
  - [x] 1.3 Add same columns to `series` table
  - [x] 1.4 Create index: `idx_movies_douban_id`, `idx_series_douban_id`
  - [x] 1.5 Migration auto-registers via its own `init()` + the blank import in `cmd/api/main.go` (no `registry.go` edit needed — that file holds no central list)

- [x] **Task 2: Extend Movie/Series models** (AC: #5, #6)
  - [x] 2.1 Add to `Movie` struct in `models/movie.go`: `DoubanID NullString`, `DoubanRating NullFloat64`, `DoubanVoteCount NullInt64`
  - [x] 2.2 Add same fields to `Series` struct in `models/series.go`
  - [x] 2.3 Use db tags: `db:"douban_id"`, `db:"douban_rating"`, `db:"douban_vote_count"`
  - [x] 2.4 Use json tags: `json:"douban_id,omitempty"`, `json:"douban_rating,omitempty"`, `json:"douban_vote_count,omitempty"`

- [x] **Task 3: Repository SQL updates** (AC: #5)
  - [x] 3.1 Update `movie_repository.go` — add new columns to SELECT, INSERT, UPDATE queries
  - [x] 3.2 Update `series_repository.go` — same changes
  - [x] 3.3 Add method `UpdateDoubanRating(ctx, id string, doubanID string, rating float64, voteCount int) error` to both repositories
  - [x] 3.4 Update Scan calls to include new columns

- [x] **Task 4: Douban rating enrichment endpoint** (AC: #2, #3, #4, #5)
  - [x] 4.1 Add `EnrichDoubanRating(ctx, mediaID string, mediaType string) (*DoubanRatingResult, error)` to metadata service (or movie/series service)
  - [x] 4.2 Logic: check if `douban_rating` already set → return cached; else search Douban by title+year via existing `douban.Client` → match → fetch detail → extract Rating+RatingCount+ID → persist via `UpdateDoubanRating` → return result
  - [x] 4.3 Use existing `douban_cache` for scraper-level caching (already implemented in `douban/cache.go`)
  - [x] 4.4 Handle errors gracefully: Douban blocked/timeout/not-found → return nil (no rating), log warning
  - [x] 4.5 Add `GET /api/v1/movies/:id/douban-rating` and `GET /api/v1/series/:id/douban-rating` endpoints
  - [x] 4.6 Response format: `{ "success": true, "data": { "douban_id": "1292052", "douban_rating": 9.7, "douban_vote_count": 2130000 } }` or `{ "success": true, "data": null }` if not found
  - [x] 4.7 Write handler + service unit tests (testify)

### Frontend

- [x] **Task 5: Extend TypeScript types** (AC: #1, #2, #6)
  - [x] 5.1 Add to `LibraryMovie` and `LibrarySeries` types: `doubanId?: string`, `doubanRating?: number`, `doubanVoteCount?: number`
  - [x] 5.2 Add `getMovieDoubanRating(id: string)` and `getSeriesDoubanRating(id: string)` to `libraryService.ts`
  - [x] 5.3 Response type: `DoubanRatingResponse = { doubanId: string; doubanRating: number; doubanVoteCount: number } | null`

- [x] **Task 6: DualRatingDisplay component** (AC: #1, #2, #6, #7)
  - [x] 6.1 Create `apps/web/src/components/media/DualRatingDisplay.tsx`
  - [x] 6.2 Props: `tmdbRating?: number`, `tmdbVoteCount?: number`, `doubanRating?: number`, `doubanVoteCount?: number`, `doubanLoading?: boolean`
  - [x] 6.3 Layout: horizontal on desktop (flex-row), vertical stack on mobile (flex-col)
  - [x] 6.4 Each rating badge: source label (TMDb / Douban), star icon, score `X.X`, vote count formatted (e.g., `2.5k`, `1.3M`)
  - [x] 6.5 Douban slot: show skeleton when `doubanLoading=true`, hide entirely when no Douban data and not loading
  - [x] 6.6 Tailwind styling consistent with existing detail page (check `$type.$id.tsx` line 233-237 for current rating style)
  - [x] 6.7 Write `DualRatingDisplay.spec.tsx` co-located test

- [x] **Task 7: Integrate into detail page** (AC: #1, #2, #3, #7)
  - [x] 7.1 In `apps/web/src/routes/media/$type.$id.tsx`, replace the inline `⭐ {localData.voteAverage.toFixed(1)}` with `<DualRatingDisplay />`
  - [x] 7.2 Add TanStack Query hook: `useDoubanRating(id, type)` that calls the new endpoint — enabled only when `tmdbId > 0`
  - [x] 7.3 Pass local data's `voteAverage`/`voteCount` as TMDB props, query result as Douban props
  - [x] 7.4 `staleTime: 24 * 60 * 60 * 1000` (24h, matching server cache TTL)
  - [x] 7.5 Update existing tests in `$type.$id.spec.tsx` if any reference the old rating display

## Dev Notes

### Architecture Compliance

- **Rule 4 (Layered Architecture):** Handler → Service → Repository. The new endpoint follows this: `DoubanRatingHandler.Get` → `MovieService.EnrichDoubanRating` → `MovieRepository.UpdateDoubanRating` + `douban.Client.Search/GetDetail`
- **Rule 5 (TanStack Query):** Douban rating fetched via `useQuery` hook, NOT stored in Zustand
- **Rule 6 (Naming):** Go: `douban_rating` (snake_case DB column), `DoubanRating` (PascalCase struct field); TS: `doubanRating` (camelCase after `snakeToCamel`)
- **Rule 7 (Error Codes):** Reuse existing `DOUBAN_TIMEOUT`, `DOUBAN_NOT_FOUND`, `DOUBAN_BLOCKED` from `douban/types.go`
- **Rule 10 (API Versioning):** Endpoints: `/api/v1/movies/:id/douban-rating`, `/api/v1/series/:id/douban-rating`
- **Rule 13 (Error Handling):** Douban errors logged + returned as nil data (graceful degradation), NOT swallowed silently
- **Rule 16 (Test Assertions):** Use `toBeInTheDocument()` for DOM presence, `toEqual` for structured data
- **Rule 18 (Case Transform):** Response passes through `snakeToCamel` in `fetchApi` — no extra work needed

### Project Structure Notes

**Files to CREATE:**
- `apps/api/internal/database/migrations/022_add_douban_rating_fields.go`
- `apps/web/src/components/media/DualRatingDisplay.tsx`
- `apps/web/src/components/media/DualRatingDisplay.spec.tsx`

**Files to MODIFY:**
- `apps/api/internal/models/movie.go` — add 3 Douban fields to Movie struct
- `apps/api/internal/models/series.go` — add 3 Douban fields to Series struct
- `apps/api/internal/repository/movie_repository.go` — SQL queries + `UpdateDoubanRating`
- `apps/api/internal/repository/series_repository.go` — SQL queries + `UpdateDoubanRating`
- `apps/api/internal/services/movie_service.go` — `EnrichDoubanRating` method
- `apps/api/internal/services/series_service.go` — `EnrichDoubanRating` method (or shared metadata service)
- `apps/api/internal/handlers/movie_handler.go` — register `GET /movies/:id/douban-rating` route
- `apps/api/internal/handlers/series_handler.go` — register `GET /series/:id/douban-rating` route
- `apps/api/main.go` — wire up new routes (Rule 15)
- `apps/api/internal/database/migrations/registry.go` — register migration 022
- `apps/web/src/services/libraryService.ts` — add `getMovieDoubanRating`, `getSeriesDoubanRating`
- `apps/web/src/routes/media/$type.$id.tsx` — replace inline rating with `<DualRatingDisplay />`
- Frontend type files — add `doubanId`, `doubanRating`, `doubanVoteCount` fields

### Critical Implementation Details

1. **Douban client already exists** at `apps/api/internal/douban/` with full scraper, cache, and rate limiting (0.5 req/s). Do NOT create a new client. Use the existing `douban.Client` interface.

2. **Douban cache table already exists** (`douban_cache`, migration 008). The scraper caches full `DetailResult` there. Story 12-1 adds denormalized rating fields directly on `movies`/`series` tables for fast reads without joining.

3. **Metadata provider exists** at `apps/api/internal/metadata/douban_provider.go`. The enrichment service should use this provider's search capability rather than calling `douban.Client` directly — this respects the circuit breaker and health check patterns in the metadata layer.

4. **Vote count formatting:** Use a shared utility (or create one) for abbreviating numbers:
   - `< 1000` → show as-is (e.g., "856")
   - `>= 1000` → "1.2k", "15k"
   - `>= 1000000` → "1.3M"
   - Douban movies commonly have 100k-2M+ ratings

5. **Rating scale:** Both TMDB and Douban use 0-10 scale. Display with 1 decimal place.

6. **Enrichment trigger:** The frontend calls the Douban rating endpoint lazily on detail page load. The backend either returns cached data or triggers a lookup. This avoids bulk-scraping Douban for the entire library.

7. **Performance target:** Detail page < 1.5s (Epic 12 success criteria). TMDB data comes from local DB (instant). Douban endpoint should return < 500ms from cache, < 3s for fresh scrape.

### Existing Code References

- Current rating display: `apps/web/src/routes/media/$type.$id.tsx` lines 233-237
- Douban scraper types: `apps/api/internal/douban/types.go` — `DetailResult.Rating`, `DetailResult.RatingCount`
- Douban cache: `apps/api/internal/douban/cache.go` — `GetByDoubanID`, `Store`
- Douban metadata provider: `apps/api/internal/metadata/douban_provider.go`
- Movie model: `apps/api/internal/models/movie.go` line 104-169
- MetadataSource priority: `apps/api/internal/models/movie.go` lines 33-40 (Douban = 50)
- TMDB service frontend: `apps/web/src/services/tmdb.ts`
- Library service frontend: `apps/web/src/services/libraryService.ts`

### UX Design Note

No UX design screen exists yet for dual rating display (Epic 12 designs not yet in `ux-design.pen`). Implementation should follow the existing detail page visual style:
- Current: `⭐ {voteAverage.toFixed(1)}` in `text-[var(--warning)]` color
- New: Two rating badges side-by-side, each with source label + star + score + vote count
- Use CSS variables from the design system for consistent theming
- Refer to existing `TechBadgeGroup.tsx` pattern for badge-style components

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-1]
- [Source: project-context.md — Rules 4, 5, 6, 7, 10, 13, 16, 18]
- [Source: apps/api/internal/douban/types.go — DetailResult struct]
- [Source: apps/api/internal/models/movie.go — Movie struct, MetadataSource]
- [Source: apps/api/internal/metadata/douban_provider.go — DoubanProvider]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / dev-story workflow)

### Debug Log References

- Full Go suite (`go test ./...` from `apps/api`): PASS (run after the vote_count read-back additions).
- Full web suite (`npx vitest run` under `apps/web`): 2027 passed; 16 PRE-EXISTING failures across 3 `eslint.config.mjs` wiring specs (filed — see Completion Notes).
- `go vet ./...`, eslint (touched files), prettier (touched files): clean.

### Completion Notes List

**Architecture decision (Task 4):** Per user selection (option ①), enrichment lives in a dedicated `DoubanRatingService` + `DoubanRatingHandler`, but reuses the SAME `*metadata.DoubanProvider` that `MetadataService` already owns — exposed via a new `MetadataService.DoubanProvider()` getter. This keeps a single rate limiter / circuit breaker / cache for douban.com (Rule 27 ① / Rule 14) while keeping the service independently unit-testable. `main.go` guards the typed-nil-in-interface trap (passes a genuine nil `DoubanSearcher` when Douban is disabled) so the service degrades gracefully instead of dereferencing.

**Deviations from story text (factual corrections, not scope changes):**
- Migration number `022` → **024** (`022_create_explore_blocks.go` and `023_create_filter_presets.go` already exist).
- `main.go` is at `apps/api/cmd/api/main.go`, not `apps/api/main.go`.
- Task 1.5 "register in registry.go" — migrations auto-register via their own `init()` + the blank import in `cmd/api/main.go`; `registry.go` holds no central list, so no edit was needed.

**Scope expansion (Rule 24 ① expand-in-place, under Task 7):** AC #1/#6 require the **TMDB vote count**, and Task 7.3 passes `voteCount` — but `vote_count` was absent from the movie/series SELECT-column lists and `LibraryMovie`/`LibrarySeries` types. Added `vote_count` read-back to `movieSelectColumns`/`scanMovie` + `seriesSelectColumns`/`scanSeries` (additive — purely populates a field that was previously null in API responses) and `voteCount?: number` to both TS types. Series repo test schema gained the `vote_count` column to match.

🔗 **AC Drift: NONE** (checked `voteAverage|vote_average|⭐|rating` across `_bmad-output/implementation-artifacts/*.md` — Story 2-4 AC 4.5 "display rating" is the only prior coverage; the TMDB rating display is preserved and the Douban rating is layered on additively, so no prior contract is broken — REUSE, not DRIFT. Story 2-4 is pre-Rule-20.)

📎 **Contract Stamps: NONE** (no `[@contract-v*]` stamps in this story or its upstream refs; Story 2-4 / the detail route are pre-Rule-20 — this is normal for a story that defines no cross-story wire contract.)

🎭 **A11y Pre-Flight: PASS** (1 component checked — `DualRatingDisplay`; 0 jsx-a11y warnings on touched files under `components/`, 0 introduced. The route file `routes/media/$type.$id.tsx` is OUTSIDE the jsx-a11y `components/**` scope — manual check: the loading skeleton carries `role=status` + `aria-live=polite` for the async-revealed Douban rating [Story 10-4 L1 class], the `Star` icon is `aria-hidden`; no `<img>` / modal / combobox added, so the responsive-image / focus-trap / custom-widget classes are N/A.)

⚠️ **Pre-existing failure FILED (Epic 9c Retro AI-2, option 2):** `preexisting-fail-eslint-config-wiring-vitest` — 16 tests in `implements-pen-node-id.spec.ts` / `jsx-a11y-config.spec.ts` / `time-dependent-fixture-stability.spec.ts` fail under `nx test web` (vitest cannot resolve the `eslint.config.mjs` blocks they introspect). Confirmed pre-existing (fails on clean `main` via `git stash`; Story 12-1 touched no eslint config/rule files) and the config is correct (plain `node` import resolves all expected blocks) → vitest ESM-interop regression, non-trivial, unrelated to dual-rating.

🎨 **UX Verification: N/A** — no `.pen` design screen exists for Epic 12 dual rating (Dev Note "No UX design screen exists yet"). Implementation follows the existing detail-page rating style (`var(--warning)` star/score, design-system CSS vars) per the story's UX Design Note. Component header uses the accepted `// Design ref: ux-design.pen Screen 4 Detail Panel Desktop (RgSxQ)` marker (Rule 21).

### Senior Developer Review (AI) — 2026-06-11

Adversarial code review (CR workflow). 🔒 Rule 7 Wire Format: PASS (0 new error-code constants; handler reuses registered `VALIDATION_REQUIRED_FIELD`). Rule 20 Contract Bump: N/A. Rule 25 Mega-line: N/A. Git vs File List: 0 discrepancies. Findings — 1 High / 2 Medium / 3 Low; HIGH+MEDIUM auto-fixed (option [1]):

- **H1 (AC #3 "by title+year") FIXED** — `lookup` no longer blindly takes `Items[0]`. New `pickBestMatch(items, year)` prefers an EXACT `Year` match over Douban's first hit (skipping zero-rating subjects), eliminating the wrong-same-title-film risk. The record's year was already plumbed into `SearchRequest` but was ignored at selection time. (+2 service tests: year-match wins, no-year falls back to first rated.)
- **M1 (sync scrape can hang the request) FIXED** — `lookup` now wraps the Douban search in `context.WithTimeout(ctx, doubanLookupTimeout=10s)`; a slow/hung scrape degrades to TMDb-only (AC #4) instead of blocking the detail-page request indefinitely.
- **M2 (handler 404-masks infra errors) FIXED** — repo `FindByID` (movie+series) now wraps `sql.ErrNoRows` with `%w`; service maps a genuine not-found to the new sentinel `services.ErrMediaNotFound` (→ 404) and propagates any other error unchanged (→ 500). Handler branches accordingly. (+2 service tests, handler test split into 404 + 500 cases.)
- **L1 (`formatVoteCount` boundary) FIXED** — 999_500–999_999 rounded to `"1000k"`; now promotes to `"1M"`. (+1 spec case.)
- **L2 (series cache-hit untested) FIXED** — added `TestEnrichDoubanRating_SeriesCacheHit`.
- **L3 (NOT 12-1)** — `TestScannerService_SSEBroadcast_ScanCancelled` is intermittently flaky under parallel-package load (scanner service untouched by this story). Pre-existing timing flake; recommend filing separately.

Post-fix verification: `go build ./...` clean; `go vet` clean; services/handlers/repository/migrations/models suites green (modulo the pre-existing L3 flake); web `formatVoteCount`/`DualRatingDisplay`/`useDoubanRating` (19) + detail-page spec (42) green; prettier clean.

### File List

**Created (backend):**
- `apps/api/internal/database/migrations/024_add_douban_rating_fields.go`
- `apps/api/internal/database/migrations/024_add_douban_rating_fields_test.go`
- `apps/api/internal/services/douban_rating_service.go`
- `apps/api/internal/services/douban_rating_service_test.go`
- `apps/api/internal/handlers/douban_rating_handler.go`
- `apps/api/internal/handlers/douban_rating_handler_test.go`

**Created (frontend):**
- `apps/web/src/components/media/DualRatingDisplay.tsx`
- `apps/web/src/components/media/DualRatingDisplay.spec.tsx`
- `apps/web/src/utils/formatVoteCount.ts`
- `apps/web/src/utils/formatVoteCount.spec.ts`
- `apps/web/src/hooks/useDoubanRating.ts`
- `apps/web/src/hooks/useDoubanRating.spec.tsx`

**Modified (backend):**
- `apps/api/internal/models/movie.go` — Douban fields on Movie
- `apps/api/internal/models/series.go` — Douban fields on Series
- `apps/api/internal/repository/movie_repository.go` — select columns + scan (+ `vote_count`, douban), `UpdateDoubanRating`
- `apps/api/internal/repository/series_repository.go` — select columns + scan (+ `vote_count`, douban), `UpdateDoubanRating`
- `apps/api/internal/repository/interfaces.go` — `UpdateDoubanRating` on both interfaces
- `apps/api/internal/repository/movie_repository_test.go` — douban columns in test schema + round-trip test
- `apps/api/internal/repository/series_repository_test.go` — `vote_count` + douban columns in test schema
- `apps/api/internal/services/metadata_service.go` — `doubanProvider` field + `DoubanProvider()` getter
- `apps/api/internal/testutil/mocks.go` — `UpdateDoubanRating` on Mock{Movie,Series}Repository + default expectations
- `apps/api/internal/services/enrichment_nfo_test.go` — mock `UpdateDoubanRating`
- `apps/api/internal/services/parse_queue_service_test.go` — mock `UpdateDoubanRating` (movie + series)
- `apps/api/cmd/api/main.go` — wire DoubanRatingService + handler + routes

**Modified (frontend):**
- `apps/web/src/types/library.ts` — `voteCount`/`doubanId`/`doubanRating`/`doubanVoteCount` on `LibraryMovie`+`LibrarySeries`; `DoubanRating`/`DoubanRatingResponse` types
- `apps/web/src/services/libraryService.ts` — `getMovieDoubanRating` / `getSeriesDoubanRating`
- `apps/web/src/routes/media/$type.$id.tsx` — `<DualRatingDisplay>` (Local + TMDb views), `useDoubanRating` hook
- `apps/web/src/routes/media/-$type.$id.spec.tsx` — douban service stubs in libraryService mock

**Modified (tracking):**
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — story → review; filed `preexisting-fail-eslint-config-wiring-vitest`

### Change Log

| Date | Change |
|------|--------|
| 2026-06-10 | Task 1: migration 024 adds `douban_id`/`douban_rating`/`douban_vote_count` to movies+series + `idx_{movies,series}_douban_id` (idempotent; +4 tests). |
| 2026-06-10 | Task 2: Douban fields added to Movie + Series models (db/json tags). |
| 2026-06-10 | Task 3: movie+series repositories — Douban (and `vote_count`) columns in SELECT/scan, new `UpdateDoubanRating` method + interface entries + mocks; round-trip repo test. |
| 2026-06-10 | Task 4: `DoubanRatingService` (shared DoubanProvider via `MetadataService.DoubanProvider()`) + `DoubanRatingHandler` (`GET /api/v1/{movies,series}/:id/douban-rating`); graceful degradation to `data: null`; wired in main.go; +11 service / +4 handler tests. |
| 2026-06-10 | Task 5: TS types (`voteCount`, douban fields, `DoubanRatingResponse`) + `getMovieDoubanRating`/`getSeriesDoubanRating` service methods. |
| 2026-06-10 | Task 6: `DualRatingDisplay` component (responsive, skeleton, graceful hide) + `formatVoteCount` util; +8 / +6 tests. |
| 2026-06-10 | Task 7: integrated `<DualRatingDisplay>` into Local + TMDb detail views via `useDoubanRating` (24h staleTime, gated on tmdbId>0); +4 hook tests; spec mock stubs. |
| 2026-06-10 | Filed pre-existing `preexisting-fail-eslint-config-wiring-vitest` (16 unrelated config-wiring spec failures). Story status → review. |
| 2026-06-11 | CR fixes: H1 `pickBestMatch` year-disambiguation (no more blind Items[0]); M1 10s lookup timeout; M2 `ErrMediaNotFound` sentinel → 404 vs 500 (repo wraps `sql.ErrNoRows`); L1 `formatVoteCount` 1M boundary; L2 series cache-hit test. +6 tests. Status → done. |
