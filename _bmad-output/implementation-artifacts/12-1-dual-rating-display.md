# Story 12.1: Dual Rating Display — TMDB + Douban Side-by-Side

Status: ready-for-dev

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

- [ ] **Task 1: DB Migration 022 — Add Douban rating columns** (AC: #5)
  - [ ] 1.1 Create `022_add_douban_rating_fields.go` in `apps/api/internal/database/migrations/`
  - [ ] 1.2 Add columns to `movies` table: `douban_id TEXT`, `douban_rating REAL`, `douban_vote_count INTEGER`
  - [ ] 1.3 Add same columns to `series` table
  - [ ] 1.4 Create index: `idx_movies_douban_id`, `idx_series_douban_id`
  - [ ] 1.5 Register migration in `registry.go`

- [ ] **Task 2: Extend Movie/Series models** (AC: #5, #6)
  - [ ] 2.1 Add to `Movie` struct in `models/movie.go`: `DoubanID NullString`, `DoubanRating NullFloat64`, `DoubanVoteCount NullInt64`
  - [ ] 2.2 Add same fields to `Series` struct in `models/series.go`
  - [ ] 2.3 Use db tags: `db:"douban_id"`, `db:"douban_rating"`, `db:"douban_vote_count"`
  - [ ] 2.4 Use json tags: `json:"douban_id,omitempty"`, `json:"douban_rating,omitempty"`, `json:"douban_vote_count,omitempty"`

- [ ] **Task 3: Repository SQL updates** (AC: #5)
  - [ ] 3.1 Update `movie_repository.go` — add new columns to SELECT, INSERT, UPDATE queries
  - [ ] 3.2 Update `series_repository.go` — same changes
  - [ ] 3.3 Add method `UpdateDoubanRating(ctx, id string, doubanID string, rating float64, voteCount int) error` to both repositories
  - [ ] 3.4 Update Scan calls to include new columns

- [ ] **Task 4: Douban rating enrichment endpoint** (AC: #2, #3, #4, #5)
  - [ ] 4.1 Add `EnrichDoubanRating(ctx, mediaID string, mediaType string) (*DoubanRatingResult, error)` to metadata service (or movie/series service)
  - [ ] 4.2 Logic: check if `douban_rating` already set → return cached; else search Douban by title+year via existing `douban.Client` → match → fetch detail → extract Rating+RatingCount+ID → persist via `UpdateDoubanRating` → return result
  - [ ] 4.3 Use existing `douban_cache` for scraper-level caching (already implemented in `douban/cache.go`)
  - [ ] 4.4 Handle errors gracefully: Douban blocked/timeout/not-found → return nil (no rating), log warning
  - [ ] 4.5 Add `GET /api/v1/movies/:id/douban-rating` and `GET /api/v1/series/:id/douban-rating` endpoints
  - [ ] 4.6 Response format: `{ "success": true, "data": { "douban_id": "1292052", "douban_rating": 9.7, "douban_vote_count": 2130000 } }` or `{ "success": true, "data": null }` if not found
  - [ ] 4.7 Write handler + service unit tests (testify)

### Frontend

- [ ] **Task 5: Extend TypeScript types** (AC: #1, #2, #6)
  - [ ] 5.1 Add to `LibraryMovie` and `LibrarySeries` types: `doubanId?: string`, `doubanRating?: number`, `doubanVoteCount?: number`
  - [ ] 5.2 Add `getMovieDoubanRating(id: string)` and `getSeriesDoubanRating(id: string)` to `libraryService.ts`
  - [ ] 5.3 Response type: `DoubanRatingResponse = { doubanId: string; doubanRating: number; doubanVoteCount: number } | null`

- [ ] **Task 6: DualRatingDisplay component** (AC: #1, #2, #6, #7)
  - [ ] 6.1 Create `apps/web/src/components/media/DualRatingDisplay.tsx`
  - [ ] 6.2 Props: `tmdbRating?: number`, `tmdbVoteCount?: number`, `doubanRating?: number`, `doubanVoteCount?: number`, `doubanLoading?: boolean`
  - [ ] 6.3 Layout: horizontal on desktop (flex-row), vertical stack on mobile (flex-col)
  - [ ] 6.4 Each rating badge: source label (TMDb / Douban), star icon, score `X.X`, vote count formatted (e.g., `2.5k`, `1.3M`)
  - [ ] 6.5 Douban slot: show skeleton when `doubanLoading=true`, hide entirely when no Douban data and not loading
  - [ ] 6.6 Tailwind styling consistent with existing detail page (check `$type.$id.tsx` line 233-237 for current rating style)
  - [ ] 6.7 Write `DualRatingDisplay.spec.tsx` co-located test

- [ ] **Task 7: Integrate into detail page** (AC: #1, #2, #3, #7)
  - [ ] 7.1 In `apps/web/src/routes/media/$type.$id.tsx`, replace the inline `⭐ {localData.voteAverage.toFixed(1)}` with `<DualRatingDisplay />`
  - [ ] 7.2 Add TanStack Query hook: `useDoubanRating(id, type)` that calls the new endpoint — enabled only when `tmdbId > 0`
  - [ ] 7.3 Pass local data's `voteAverage`/`voteCount` as TMDB props, query result as Douban props
  - [ ] 7.4 `staleTime: 24 * 60 * 60 * 1000` (24h, matching server cache TTL)
  - [ ] 7.5 Update existing tests in `$type.$id.spec.tsx` if any reference the old rating display

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

### Debug Log References

### Completion Notes List

### File List
