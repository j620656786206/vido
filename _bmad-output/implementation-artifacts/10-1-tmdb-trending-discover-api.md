# Story 10.1: TMDB Trending & Discover API with Server-Side Filtering

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user browsing the homepage,
I want trending and discovery content to be pre-filtered for zh-TW relevance,
so that I see content relevant to my region without far-future or low-quality noise.

## Acceptance Criteria

1. Given the TMDB client, when trending endpoints are called, then the system returns trending movies and TV shows with zh-TW language metadata
2. Given the TMDB discover endpoint, when called with custom filters (genre, year, region, language), then results respect all filter parameters simultaneously
3. Given trending results, when server-side filtering is applied, then items with release dates >6 months in the future are excluded (P2-004)
4. Given trending results, when quality filtering is applied, then items with TMDB rating <3 AND vote count <50 are excluded (P2-005)
5. Given the trending API, when called, then results are cached with 1-hour TTL to respect TMDB rate limits
6. Given the new endpoints, when called without authentication issues, then responses follow existing `ApiResponse<T>` wrapper format

## Tasks / Subtasks

- [ ] Task 1: Extend TMDB client with trending/discover endpoints (AC: #1, #2)
  - [ ] 1.1 Add `GetTrendingMovies(ctx, timeWindow, page)` → `GET /trending/movie/{time_window}` to `apps/api/internal/tmdb/movies.go`
  - [ ] 1.2 Add `GetTrendingTVShows(ctx, timeWindow, page)` → `GET /trending/tv/{time_window}` to `apps/api/internal/tmdb/tv.go`
  - [ ] 1.3 Add `DiscoverMovies(ctx, params)` → `GET /discover/movie` with genre, year, region, language, sort params
  - [ ] 1.4 Add `DiscoverTVShows(ctx, params)` → `GET /discover/tv` with same params
  - [ ] 1.5 Add response types: `TrendingResult`, `DiscoverParams` structs
  - [ ] 1.6 All methods use existing `LanguageFallbackClient` for zh-TW priority

- [ ] Task 2: Server-side content filtering (AC: #3, #4)
  - [ ] 2.1 Create `apps/api/internal/services/content_filter_service.go`
  - [ ] 2.2 `FilterFarFuture(results)` — exclude items with release_date > now + 6 months
  - [ ] 2.3 `FilterLowQuality(results)` — exclude items with vote_average < 3 AND vote_count < 50
  - [ ] 2.4 Filters applied post-fetch, before returning to frontend

- [ ] Task 3: Extend TMDb service interface (AC: #1, #5)
  - [ ] 3.1 Add `GetTrendingMovies`, `GetTrendingTVShows`, `DiscoverMovies`, `DiscoverTVShows` to `TMDbServiceInterface`
  - [ ] 3.2 Implement with existing `CacheService` layer (1-hour TTL) (AC: #5)
  - [ ] 3.3 Apply content filters to all trending/discover results

- [ ] Task 4: API endpoints (AC: #6)
  - [ ] 4.1 `GET /api/v1/tmdb/trending/movies?time_window=week&page=1`
  - [ ] 4.2 `GET /api/v1/tmdb/trending/tv?time_window=week&page=1`
  - [ ] 4.3 `GET /api/v1/tmdb/discover/movies?genre=28&year_gte=2024&region=TW&sort=popularity.desc`
  - [ ] 4.4 `GET /api/v1/tmdb/discover/tv?genre=18&language=zh&sort=popularity.desc`
  - [ ] 4.5 Add to `apps/api/internal/handlers/tmdb_handler.go`

- [ ] Task 5: Tests (AC: #1-6)
  - [ ] 5.1 TMDB client unit tests with mock HTTP server
  - [ ] 5.2 Content filter unit tests: far-future exclusion, low-quality exclusion, edge cases
  - [ ] 5.3 Handler tests: verify response format, query param parsing

## Dev Notes

### Architecture Compliance

- **TMDB client:** Extend existing `apps/api/internal/tmdb/` — do NOT create new HTTP clients
- **Caching:** Use existing `CacheService` (tiered in-memory + SQLite, 1-hour TTL per architecture decision #4)
- **Language fallback:** All new methods must use `LanguageFallbackClient` (zh-TW → zh-CN → en)
- **API response format:** Use existing `ApiResponse<T>` wrapper, snake_case JSON (Rule 18)

### Project Structure Notes

- Modified files:
  - `apps/api/internal/tmdb/movies.go` (add trending/discover methods)
  - `apps/api/internal/tmdb/tv.go` (add trending/discover methods)
  - `apps/api/internal/services/tmdb_service.go` (extend interface + impl)
  - `apps/api/internal/handlers/tmdb_handler.go` (new endpoints)
  - `apps/api/cmd/main.go` (register new routes)
- New files:
  - `apps/api/internal/services/content_filter_service.go`
  - Tests for each

### References

- [Source: apps/api/internal/tmdb/client.go] — Existing TMDB client
- [Source: apps/api/internal/tmdb/movies.go] — Movie methods pattern
- [Source: apps/api/internal/services/tmdb_service.go] — Service interface
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.4] — P2-001~P2-005

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
