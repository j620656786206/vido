# Story 9.1: Subtitle Search Integration

Status: ready-for-dev

## Story

As a **Traditional Chinese user**,
I want to **search for subtitles from multiple sources (OpenSubtitles + Zimuku)**,
so that **I can find subtitles for my media**.

## Acceptance Criteria

1. **AC1: Search Button** — Given the user opens a media detail page, when subtitles are not attached, then a "Search Subtitles" button is available.
2. **AC2: Multi-Source Search** — Given the user clicks "Search Subtitles", when the search executes, then it queries OpenSubtitles API first, then queries Zimuku (web scraping), and results are combined and deduplicated.
3. **AC3: Result Display** — Given search results are displayed, when viewing the list, then each result shows: Language, Format, Source, Rating/Downloads, and Traditional Chinese subtitles are highlighted.
4. **AC4: Partial Results** — Given an error occurs with one source, when the other source succeeds, then results from the working source are shown and an error message indicates partial results.

## Tasks / Subtasks

- [ ] Task 1: Create subtitle database schema and migration (AC: all)
  - [ ] 1.1 Create migration `013_create_subtitle_files_table.go` with columns: id, movie_id/series_id, language, format, source, file_path, rating, download_count, status, created_at, updated_at
  - [ ] 1.2 Create index on movie_id and language for fast lookups
  - [ ] 1.3 Add subtitle_status enum values: "downloaded", "online", "unavailable"

- [ ] Task 2: Create subtitle models (AC: all)
  - [ ] 2.1 Create `apps/api/internal/models/subtitle.go` with SubtitleFile struct (follow movie.go patterns: `db` + `json` tags, sql.Null* for optionals)
  - [ ] 2.2 Create SubtitleSearchResult struct for API responses (Language, Format, Source, Rating, DownloadCount, DownloadURL)
  - [ ] 2.3 Create SubtitleSource constants: "opensubtitles", "zimuku", "manual"
  - [ ] 2.4 Create SubtitleFormat constants: "srt", "ass", "ssa", "sub", "vtt"

- [ ] Task 3: Create OpenSubtitles API client (AC: 2, 4)
  - [ ] 3.1 Create `apps/api/internal/subtitle/opensubtitles.go` with OpenSubtitlesClient struct
  - [ ] 3.2 Implement Search(ctx, title, language) method with HTTP client, API key auth, rate limiting
  - [ ] 3.3 Implement response parsing to SubtitleSearchResult
  - [ ] 3.4 Add circuit breaker: 5 failures → 1 min cooldown
  - [ ] 3.5 Add configurable timeout (default 10s)
  - [ ] 3.6 Write unit tests with mocked HTTP responses

- [ ] Task 4: Create Zimuku web scraper (AC: 2, 4)
  - [ ] 4.1 Create `apps/api/internal/subtitle/zimuku.go` with ZimukuScraper struct
  - [ ] 4.2 Implement Search(ctx, title) using goquery (follow douban/scraper.go patterns)
  - [ ] 4.3 Add User-Agent rotation and rate limiting (1 req/2s)
  - [ ] 4.4 Parse HTML results to SubtitleSearchResult
  - [ ] 4.5 Add circuit breaker matching OpenSubtitles pattern
  - [ ] 4.6 Write unit tests with HTML fixture files

- [ ] Task 5: Create subtitle service with multi-source orchestration (AC: 2, 3, 4)
  - [ ] 5.1 Create `apps/api/internal/services/subtitle_service.go` with SubtitleService
  - [ ] 5.2 Define SubtitleServiceInterface in services package (Rule 11)
  - [ ] 5.3 Implement Search(ctx, movieID, title) that queries both sources, combines, deduplicates
  - [ ] 5.4 Handle partial failures: return successful results + error info
  - [ ] 5.5 Cache search results in Tier 1 (in-memory, 2h TTL) and Tier 2 (SQLite, 24h TTL)
  - [ ] 5.6 Write unit tests (>80% coverage)

- [ ] Task 6: Create subtitle repository (AC: all)
  - [ ] 6.1 Create `apps/api/internal/repository/subtitle_repository.go`
  - [ ] 6.2 Define SubtitleRepositoryInterface in repository/interfaces.go
  - [ ] 6.3 Implement CRUD: Create, FindByMovieID, FindBySeriesID, Delete
  - [ ] 6.4 Add to repository factory in NewRepositoriesWithCache
  - [ ] 6.5 Write unit tests

- [ ] Task 7: Create subtitle handler and API endpoints (AC: 1, 3)
  - [ ] 7.1 Create `apps/api/internal/handlers/subtitle_handler.go`
  - [ ] 7.2 Implement GET `/api/v1/subtitles/search?movie_id={id}&title={title}&language={lang}`
  - [ ] 7.3 Implement GET `/api/v1/movies/{id}/subtitles` to list available subtitles
  - [ ] 7.4 Use standard response helpers (SuccessResponse, ErrorResponse)
  - [ ] 7.5 Register routes in handler RegisterRoutes method
  - [ ] 7.6 Wire handler in main.go
  - [ ] 7.7 Write handler tests (>70% coverage)

- [ ] Task 8: Create frontend subtitle search UI (AC: 1, 3, 4)
  - [ ] 8.1 Create `apps/web/src/services/subtitle.ts` API client
  - [ ] 8.2 Create `apps/web/src/hooks/useSubtitleSearch.ts` with TanStack Query
  - [ ] 8.3 Create `apps/web/src/components/subtitles/SubtitleSearch.tsx` — search button + results modal
  - [ ] 8.4 Create `apps/web/src/components/subtitles/SubtitleList.tsx` — results display with language highlighting
  - [ ] 8.5 Integrate into media detail page
  - [ ] 8.6 Handle loading, error, partial-result states
  - [ ] 8.7 Write component tests with Vitest + RTL

## Dev Notes

### Architecture Compliance

- **Layered architecture**: Handler → Service → Repository (Rule 4). Service orchestrates OpenSubtitles + Zimuku clients.
- **Logging**: Use `slog` exclusively (Rule 2). Never zerolog or fmt.Println.
- **API format**: All responses use `{success: true/false, data/error}` wrapper (Rule 3).
- **Error codes**: Use `SUBTITLE_*` prefix — `SUBTITLE_API_TIMEOUT`, `SUBTITLE_NOT_FOUND`, `SUBTITLE_RATE_LIMIT` (Rule 7).
- **Endpoints**: `/api/v1/subtitles/search` (Rule 10 — versioned, plural).
- **Naming**: Go files snake_case, structs PascalCase, JSON fields snake_case (Rule 6).

### Existing Patterns to Follow

- **Web scraping**: Follow `apps/api/internal/douban/scraper.go` — uses goquery, CSS selectors, strings.TrimSpace, regex extraction.
- **External API client**: Follow `apps/api/internal/tmdb/` — HTTP client with timeout, rate limiting, error wrapping.
- **Model pattern**: Follow `apps/api/internal/models/movie.go` — `db` + `json` tags, sql.Null* for optionals, time.Time for timestamps.
- **Handler pattern**: Follow `apps/api/internal/handlers/movie_handler.go` — interface injection, RegisterRoutes, response helpers.
- **Repository pattern**: Follow `apps/api/internal/repository/movie_repository.go` — parameterized SQL, context-aware, interface in interfaces.go.
- **Service wiring**: Follow `apps/api/cmd/api/main.go` — constructor injection, repos → services → handlers.
- **Frontend hooks**: Follow `apps/web/src/hooks/useSearchMedia.ts` — TanStack Query pattern.
- **Frontend services**: Follow `apps/web/src/services/tmdb.ts` — API client pattern.

### Caching Strategy

- **Tier 1 (in-memory)**: Cache search results for 2h. Key: `subtitle:search:{query_hash}:{language}`.
- **Tier 2 (SQLite)**: Persist availability for 24h. Key: `subtitle:availability:{movie_id}`.
- Use existing `apps/api/internal/cache/` infrastructure and CacheRepositoryInterface.

### Circuit Breaker

- 5 consecutive failures → circuit OPEN (1 min cooldown)
- During OPEN → return cached data or empty with error message
- After 1 min → HALF_OPEN (test 1 request)
- Follow retry pattern from `apps/api/internal/retry/`

### Latest Migration Number

- Current latest: `012_create_retry_stats_table`
- New migration: `013_create_subtitle_files_table`

### Project Structure Notes

- All backend code → `/apps/api` (Rule 1)
- New package: `/apps/api/internal/subtitle/` for OpenSubtitles + Zimuku clients
- New frontend: `/apps/web/src/components/subtitles/` for subtitle UI components

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#Caching Strategy]
- [Source: _bmad-output/planning-artifacts/architecture.md#Error Handling]
- [Source: project-context.md#Rules 1-11]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
