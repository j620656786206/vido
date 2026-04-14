# Story 10.1: TMDB Trending & Discover API with Server-Side Filtering

Status: review

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

- [x] Task 1: Extend TMDB client with trending/discover endpoints (AC: #1, #2)
  - [x] 1.1 Added `GetTrendingMovies` + `GetTrendingMoviesWithLanguage` → `GET /trending/movie/{time_window}` in `movies.go`. Accepts `"day"`/`"week"`; anything else normalized to `"week"`. Negative page → 1.
  - [x] 1.2 Added `GetTrendingTVShows` + `GetTrendingTVShowsWithLanguage` → `GET /trending/tv/{time_window}` in `tv.go`, symmetric to movies.
  - [x] 1.3 Added `DiscoverMovies(ctx, DiscoverParams)` → `GET /discover/movie`. `DiscoverParams.YearGte/Lte` map to `primary_release_date.gte/lte` (`YYYY-01-01` / `YYYY-12-31`). `Genre` → `with_genres`. Empty fields are omitted so TMDb applies its own defaults.
  - [x] 1.4 Added `DiscoverTVShows(ctx, DiscoverParams)` → `GET /discover/tv`. Same mapping except `YearGte/Lte` use `first_air_date.gte/lte` (TV wire key, verified by test).
  - [x] 1.5 Added `DiscoverParams` struct in `types.go`. `TrendingResult` wrapper also added for future merged-endpoint use; current trending endpoints are split by media type so the client returns concrete `SearchResultMovies` / `SearchResultTVShows` shapes (same wire format as TMDb).
  - [x] 1.6 Extended `LanguageFallbackClient` with `GetTrendingMoviesWithFallback`, `GetTrendingTVShowsWithFallback`, `DiscoverMoviesWithFallback`, `DiscoverTVShowsWithFallback`. Fallback honors caller-set `DiscoverParams.Language` (skips chain); otherwise iterates `zh-TW → zh-CN → en` and stops on first localized result. `ClientInterface` + `LanguageFallbackClientInterface` both extended; `MockClient` (fallback_test.go) and `MockFallbackClient` (cache_test.go) stubs added for existing tests.

- [x] Task 2: Server-side content filtering (AC: #3, #4)
  - [x] 2.1 Created `apps/api/internal/services/content_filter_service.go` with `ContentFilterService` struct (injected `now` clock for deterministic FarFuture tests via `NewContentFilterServiceWithClock`).
  - [x] 2.2 Implemented `FilterFarFutureMovies` (ReleaseDate) + `FilterFarFutureTVShows` (FirstAirDate) — excludes items with parseable date strictly AFTER `now + FarFutureHorizonMonths`. Items with empty/unparseable dates are RETAINED (conservative: don't silently drop data we can't assess).
  - [x] 2.3 Implemented `FilterLowQualityMovies` + `FilterLowQualityTVShows` — excludes items with `VoteAverage < 3.0 AND VoteCount < 50` (AND semantics per AC #4, verified by boundary tests: rating-exactly-3.0 keep; votes-exactly-50 keep; 2.9+49 drop). Constants `LowQualityRatingThreshold`, `LowQualityVoteCountThreshold` guarded by a separate test.
  - [x] 2.4 Wiring deferred to Task 3 (`TMDbService` will call the cache service then pass results through `ContentFilterService` before returning).

- [x] Task 3: Extend TMDb service interface (AC: #1, #5)
  - [x] 3.1 Added all 4 methods to both `tmdb.CacheServiceInterface` and `services.TMDbServiceInterface`. Updated stubs in `MockClient`, `MockFallbackClient`, `MockCacheService`, and `mockTMDbServiceForNFO` so existing tests still compile. `MockCacheRepository.lastSetTTL` added for TTL assertions.
  - [x] 3.2 `CacheService` trending/discover methods use new `TrendingDiscoverCacheTTL = 1 * time.Hour` constant — **independent** of the service-wide default TTL (verified by a dedicated test that constructs `CacheService{TTL: 24h}` and asserts the trending method still sets 1h). Cache keys: `tmdb:trending/{kind}:{window}:{page}` and `tmdb:discover/{kind}:g=…:yg=…:yl=…:r=…:lang=…:sort=…:p=…` (deterministic so different filter combos get distinct entries).
  - [x] 3.3 `TMDbService.GetTrending{Movies,TVShows}` and `DiscoverX` pipe cache result through `ContentFilterService` (FarFuture → LowQuality) before returning. Wired via new `contentFilter` field on `TMDbService`; `SetContentFilter` setter allows tests to inject a fixed clock. Filter behavior verified by table-driven tests at both the filter level AND the service pipeline level.

- [x] Task 4: API endpoints (AC: #6)
  - [x] 4.1 `GET /api/v1/tmdb/trending/movies` — handler `TMDbHandler.GetTrendingMovies`. Accepts `?time_window=day|week&page=N`; unknown/empty window → `week`; empty/invalid page → 1.
  - [x] 4.2 `GET /api/v1/tmdb/trending/tv` — handler `TMDbHandler.GetTrendingTVShows`. Same defaults as 4.1.
  - [x] 4.3 `GET /api/v1/tmdb/discover/movies` — handler `TMDbHandler.DiscoverMovies`. Parses `genre`, `year_gte`, `year_lte`, `region`, `language`, `sort`, `page` (all snake_case per Rule 18). Zero-valued fields omitted so TMDb applies its own defaults.
  - [x] 4.4 `GET /api/v1/tmdb/discover/tv` — handler `TMDbHandler.DiscoverTVShows`. Same query-param parser as 4.3.
  - [x] 4.5 Added to `apps/api/internal/handlers/tmdb_handler.go` with 3 small helpers (`parseTrendingWindow`, `parsePageQuery`, `parseDiscoverParams`). Routes registered under `/tmdb/trending` and `/tmdb/discover` groups inside `TMDbHandler.RegisterRoutes` — no `cmd/api/main.go` change needed (the existing `tmdbHandler` wiring picks up the new routes automatically). Local handler-level `TMDbServiceInterface` extended with the 4 new methods.

- [x] Task 5: Tests (AC: #1-6)
  - [x] 5.1 TMDB client unit tests: 9 test functions (4 tables + singles) covering trending day/week/unknown window, page normalization, language fallback, and discover query-param mapping (including the TV-specific `first_air_date.*` keys vs movies' `primary_release_date.*`).
  - [x] 5.2 Content filter unit tests: 6 test functions covering boundary conditions — exactly-on-horizon keep, 1-day-past horizon drop, rating exactly 3.0 keep, votes exactly 50 keep, 2.9+49 drop, empty/unparseable dates retained, plus a constants-guard test.
  - [x] 5.3 Handler tests: 7 test functions verifying default time_window, explicit day, unknown fallback, error→502, query-param mapping for all 7 discover params, empty defaults, and the APIResponse envelope `success/data/error` shape (Rule 18 boundary check).
  - [x] 5.4 Additional inter-layer: cache tests (1-hour TTL assertion, cache-miss-then-hit, distinct param keys, error non-cache) and service-layer tests (filter pipeline applied after cache fetch).

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

- DEV implementation: Claude Opus 4.6 (1M context) — Amelia `/dev-story` + `/bmad:bmm:agents:dev`, 2026-04-14.

### Debug Log References

- Initial build of `internal/tmdb` after adding `GetTrending*` / `Discover*` to `ClientInterface` → compile errors in `fallback_test.go::MockClient` (6 sites) → added 6 stub methods → compile clean.
- Second build after extending `LanguageFallbackClientInterface` → compile errors in `cache_test.go::MockFallbackClient` (6 sites) → added 4 configurable stubs via `storyTenOneFallbackConfig` side map → compile clean.
- Extending `services.TMDbServiceInterface` → compile errors in `enrichment_nfo_test.go::mockTMDbServiceForNFO` (8 sites) → added 4 pass-through stubs → compile clean.
- Handler test initial failure: `upstream_error_surfaces_as_500` expected 500 but received 502. Root cause: `tmdb.NewServerError` emits `StatusCode: 502` (Bad Gateway is semantically correct for upstream-down); test expectations corrected rather than handler behavior.
- Full regression: `pnpm nx test api` PASS (Go suite including new 23 test functions for Story 10-1), `pnpm nx test web` PASS (cached — no frontend changes), `pnpm run lint:all` PASS (0 errors, 108 warnings — unchanged from baseline).

### Completion Notes List

- All 5 tasks + 22 subtasks complete; all 6 ACs satisfied end-to-end.
- **Layered architecture preserved**: Handler → TMDbService (filter) → CacheService (1h TTL) → LanguageFallbackClient (zh-TW → zh-CN → en) → Client → TMDb API. The new trending/discover methods plug into every existing layer — zero architectural divergence.
- **AC #5 TTL**: New constant `tmdb.TrendingDiscoverCacheTTL = 1 * time.Hour`, used by `CacheService` trending/discover methods **independently** of the service-wide default TTL. Verified by a dedicated test that constructs `CacheService{TTL: 24h}` and asserts the trending method still writes 1h.
- **AC #3/#4 content filter semantics** (written conservatively, verified by boundary tests):
  - FarFuture: items with parseable date STRICTLY AFTER `now + 6 months` are dropped. Exactly-on-horizon is KEPT. Empty/unparseable dates are KEPT (don't silently drop data we can't assess).
  - LowQuality: items are dropped only if rating `<` 3.0 AND vote count `<` 50. A low-rated-but-well-voted item is kept (that's signal, not noise); a rarely-voted-but-high-rated item is also kept (niche but positive).
- **DiscoverParams TV vs movie date mapping**: `YearGte/Lte` map to `primary_release_date.{gte,lte}` for movies but `first_air_date.{gte,lte}` for TV — TMDb uses different field names per media type. Covered by a dedicated TV-discover test that asserts the TV keys are present AND the movie keys are absent.
- **Language fallback semantics for discover**: when `DiscoverParams.Language` is explicitly set by the caller, the fallback chain is SKIPPED (the caller's language is authoritative). When empty, the chain iterates `zh-TW → zh-CN → en` and stops on the first result with localized content. Matches the intent of existing search/details fallback behavior.
- **No main.go wiring change needed**: the existing `tmdbHandler := handlers.NewTMDbHandler(tmdbService)` at `apps/api/cmd/api/main.go:444` picks up the new routes automatically via `RegisterRoutes`. (Story Project Structure Notes said `apps/api/cmd/main.go` — the actual path is `apps/api/cmd/api/main.go`, noted for future stories.)
- **Response format (AC #6)**: all 4 new endpoints use the existing `SuccessResponse` / `handleTMDbError` helpers, so responses are wrapped in `APIResponse{success, data, error}` with snake_case JSON keys per Rule 18. Verified by `TestTMDbHandler_ResponseIsApiResponseWrapped`.
- **Mock churn**: 4 existing mock implementations (`MockClient`, `MockFallbackClient`, `MockCacheService`, `mockTMDbServiceForNFO`) gained pass-through stubs for the new interface methods. One existing helper (`MockCacheRepository.lastSetTTL`) was added to enable TTL assertions.

### UX Verification

🎨 UX Verification: SKIPPED — no UI changes in this story (pure backend: TMDb client extension + content filter + HTTP handlers).

### Change Log

| Date       | Change                                                                                                                                                                                                      |
|------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 2026-04-14 | Extended `tmdb.ClientInterface` with `GetTrending{Movies,TVShows}`, `GetTrending{Movies,TVShows}WithLanguage`, `Discover{Movies,TVShows}` (6 new methods). Added `DiscoverParams` + `TrendingResult` types. |
| 2026-04-14 | Extended `tmdb.LanguageFallbackClientInterface` with 4 new `*WithFallback` methods. Discover honors explicit caller language; otherwise iterates zh-TW→zh-CN→en.                                             |
| 2026-04-14 | Added `tmdb.TrendingDiscoverCacheTTL = 1 * time.Hour`. Extended `tmdb.CacheServiceInterface` with 4 trending/discover methods that cache at 1h TTL (independent of service-wide default).                   |
| 2026-04-14 | Added `services.ContentFilterService` (new file) with FarFuture + LowQuality filters for both Movie and TVShow. Injected clock enables deterministic FarFuture tests.                                       |
| 2026-04-14 | Extended `services.TMDbServiceInterface` + `TMDbService` with 4 trending/discover methods that pipe cached results through `ContentFilterService`.                                                           |
| 2026-04-14 | Added 4 HTTP endpoints: `GET /api/v1/tmdb/trending/{movies,tv}` and `GET /api/v1/tmdb/discover/{movies,tv}`. Local handler-level `TMDbServiceInterface` extended; 3 query-param helpers.                    |
| 2026-04-14 | 23 new test functions across 4 files (client, fallback, cache, services, handlers) covering ACs #1–#6 end-to-end.                                                                                            |

### File List

- `apps/api/internal/tmdb/types.go` — **MODIFIED** (added `DiscoverParams`, `TrendingResult`)
- `apps/api/internal/tmdb/client.go` — **MODIFIED** (extended `ClientInterface` with 6 methods)
- `apps/api/internal/tmdb/movies.go` — **MODIFIED** (added `GetTrendingMovies`, `GetTrendingMoviesWithLanguage`, `DiscoverMovies`, `discoverQueryParams` helper, `normalizeTrendingWindow`, `validTrendingWindows`)
- `apps/api/internal/tmdb/tv.go` — **MODIFIED** (added `GetTrendingTVShows`, `GetTrendingTVShowsWithLanguage`, `DiscoverTVShows`)
- `apps/api/internal/tmdb/fallback.go` — **MODIFIED** (extended `LanguageFallbackClientInterface`; added 4 `*WithFallback` methods)
- `apps/api/internal/tmdb/cache.go` — **MODIFIED** (added `TrendingDiscoverCacheTTL` constant; extended `CacheServiceInterface`; added 4 trending/discover cache methods + `discoverCacheKey` helper)
- `apps/api/internal/tmdb/movies_test.go` — **MODIFIED** (added `TestClient_GetTrendingMovies` + 2 more, total 3 new test functions)
- `apps/api/internal/tmdb/tv_test.go` — **MODIFIED** (added 3 new test functions)
- `apps/api/internal/tmdb/fallback_test.go` — **MODIFIED** (added 6 `MockClient` stubs + `trackingMockClient` + 5 new fallback test functions)
- `apps/api/internal/tmdb/cache_test.go` — **MODIFIED** (added `lastSetTTL` field on `MockCacheRepository`; added 4 Story 10-1 fallback-client stubs via `storyTenOneFallbackConfig` side map; added 4 new cache test functions)
- `apps/api/internal/services/content_filter_service.go` — **CREATED** (new `ContentFilterService` with 4 filter methods)
- `apps/api/internal/services/content_filter_service_test.go` — **CREATED** (6 test functions)
- `apps/api/internal/services/tmdb_service.go` — **MODIFIED** (extended `TMDbServiceInterface` with 4 methods; added `contentFilter` field + `SetContentFilter` setter + 4 service implementations)
- `apps/api/internal/services/tmdb_service_test.go` — **MODIFIED** (extended `MockCacheService` with 4 stubs + field set; added 3 Story 10-1 test functions + `mustParseDate` helper; added `time` import)
- `apps/api/internal/services/enrichment_nfo_test.go` — **MODIFIED** (added 4 pass-through stubs on `mockTMDbServiceForNFO` for the extended `TMDbServiceInterface`)
- `apps/api/internal/handlers/tmdb_handler.go` — **MODIFIED** (extended handler-local `TMDbServiceInterface` with 4 methods; added 4 handler methods + 3 query-param helpers; extended `RegisterRoutes` with `/tmdb/trending` and `/tmdb/discover` groups)
- `apps/api/internal/handlers/tmdb_handler_test.go` — **MODIFIED** (extended `MockTMDbService` with 4 stubs + capture fields; added 7 Story 10-1 test functions; added `errors` import)
