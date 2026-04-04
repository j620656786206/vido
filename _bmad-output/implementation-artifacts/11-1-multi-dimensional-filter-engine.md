# Story 11.1: Multi-Dimensional Filter Engine & Compound Sort

Status: ready-for-dev

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

- [ ] Task 1: Extend discover endpoints with full filter params (AC: #1, #5)
  - [ ] 1.1 Extend `DiscoverParams` struct in `apps/api/internal/tmdb/` with: `GenreIDs []int`, `YearGte/YearLte int`, `Region string`, `VoteAverageGte/VoteAverageLte float64`, `SortBy string`
  - [ ] 1.2 Map all params to TMDB discover API query strings
  - [ ] 1.3 Update `GET /api/v1/tmdb/discover/movies` and `/tv` to accept all filter query params

- [ ] Task 2: Watch Providers integration (AC: #2)
  - [ ] 2.1 Add `GetWatchProviders(ctx, mediaType, id, region)` to TMDB client → `GET /{media_type}/{id}/watch/providers`
  - [ ] 2.2 Add `with_watch_providers` param to discover queries for platform filtering
  - [ ] 2.3 Map provider IDs for TW region: Netflix=8, Disney+=337, KKTV, etc.

- [ ] Task 3: Compound sort support (AC: #3)
  - [ ] 3.1 Support sort params: `popularity.desc`, `release_date.desc`, `vote_average.desc`, `primary_release_date.desc`
  - [ ] 3.2 Pass `sort_by` param through to TMDB discover API
  - [ ] 3.3 For local library sort (date added): sort in application layer after fetch

- [ ] Task 4: Performance caching (AC: #4)
  - [ ] 4.1 Cache discover results by full query param hash (1-hour TTL)
  - [ ] 4.2 Use existing `CacheService` tiered caching

- [ ] Task 5: Tests (AC: #1-5)
  - [ ] 5.1 Discover param serialization tests (all filter combos)
  - [ ] 5.2 Watch provider mapping tests
  - [ ] 5.3 Handler tests with various query param combinations

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

### Debug Log References

### Completion Notes List

### File List
