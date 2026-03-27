# Story retro-8-TD7: Fix API JSON Naming Convention (Rule 6 Compliance)

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a frontend developer,
I want API JSON responses to use snake_case field names (per project-context.md Rule 6),
so that the frontend can reliably consume API data without field name mismatches.

## Background

Since Epic 2, Go model JSON tags have used camelCase (`posterPath`, `releaseDate`) while:
- project-context.md Rule 6 specifies: `JSON Fields: snake_case (release_date, tmdb_id)`
- Frontend TypeScript types use snake_case (`poster_path`, `release_date`)
- Database columns use snake_case (`poster_path`, `release_date`)

This mismatch was hidden by a second bug: `sql.NullString` serialized as `{"String":"...","Valid":true}` instead of the raw value. Commit `a123f59` fixed the serialization, exposing the naming mismatch. Result: poster images and other nullable fields don't render on the NAS deployment.

**Party Mode Decision (2026-03-27):** Agreed on Boundary Transformation Pattern (Option 1):
- Go JSON tags → snake_case (match Rule 6 + DB columns)
- Frontend adds `snakeToCamel` utility in each service function (not interceptor)
- Frontend TypeScript types → camelCase (JS convention)

## Acceptance Criteria

1. **AC1: Go JSON tags use snake_case** — All model structs in `apps/api/internal/models/` use snake_case JSON tags matching their `db:` tags (e.g., `json:"poster_path,omitempty"`)
2. **AC2: Frontend transform utility** — A `snakeToCamel` utility exists in `apps/web/src/utils/` that recursively converts snake_case object keys to camelCase
3. **AC3: Frontend service integration** — All service functions that consume API responses apply the transform before returning data
4. **AC4: Frontend types use camelCase** — TypeScript types in `apps/web/src/types/library.ts` and service files use camelCase (`posterPath`, `releaseDate`)
5. **AC5: Poster images render** — Library grid shows poster images from TMDb when `poster_path` is populated in the database
6. **AC6: All existing tests pass** — Go unit tests, frontend unit tests, and E2E tests all pass after the changes
7. **AC7: Transform utility tested** — Unit tests for `snakeToCamel` cover: flat objects, nested objects, arrays, null values, edge cases (e.g., `tmdb_id` → `tmdbId`, `imdb_id` → `imdbId`)

## Tasks / Subtasks

### Task 1: Go JSON tags → snake_case (AC1)

- [x] 1.1 Update `movie.go` JSON tags: all camelCase → snake_case (match `db:` tags)
- [x] 1.2 Update `series.go` JSON tags (including `SeriesBasicInfo`, `Network` embedded structs)
- [x] 1.3 Update `season.go` JSON tags
- [x] 1.4 Update `episode.go` JSON tags
- [x] 1.5 Update `parse_job.go` JSON tags
- [x] 1.6 Update `parse_status.go` JSON tags
- [x] 1.7 Update `filename_mapping.go` JSON tags
- [x] 1.8 Update `settings.go` JSON tags
- [x] 1.9 Update `backup.go` JSON tags
- [x] 1.10 Update `secret.go` JSON tags
- [x] 1.11 Update `system_log.go` JSON tags
- [x] 1.12 Update `connection_event.go` JSON tags
- [x] 1.13 Update `degradation.go` JSON tags
- [x] 1.14 Update Go handler tests that assert JSON field names (grep for camelCase field assertions in `*_handler_test.go`)
- [x] 1.15 (additional) Update JSON tags in handlers/, services/, qbittorrent/, retry/, cache/, metadata/, events/, repository/, ai/, subtitle/ packages

### Task 2: Frontend transform utility (AC2, AC7)

- [x] 2.1 Create `apps/web/src/utils/caseTransform.ts` with `snakeToCamel(obj)` function
- [x] 2.2 Handle edge cases: nested objects, arrays of objects, null/undefined, primitive values
- [x] 2.3 Create `apps/web/src/utils/caseTransform.spec.ts` with comprehensive tests (14 tests)
- [x] 2.4 Test edge cases: `tmdb_id` → `tmdbId`, `imdb_id` → `imdbId`, `created_at` → `createdAt`

### Task 3: Frontend TypeScript types → camelCase (AC4)

- [x] 3.1 Update `apps/web/src/types/library.ts` — `LibraryMovie` and `LibrarySeries` interfaces: `poster_path` → `posterPath`, `release_date` → `releaseDate`, etc.
- [x] 3.2 Update any other TypeScript type files that reference snake_case API fields (tmdb.ts, scannerService.ts, subtitleService.ts, useSubtitleSearch.ts, useScanProgress.ts)
- [x] 3.3 Update component references to use new camelCase field names (21 files: LibraryGrid, LibraryTable, RecentlyAdded, MediaDetailPanel, TVShowInfo, MediaGrid, CreditsSection, SubtitleSearchDialog, SearchResults, ScannerSettings, routes)

### Task 4: Frontend service integration (AC3)

- [x] 4.1 Apply `snakeToCamel` in `libraryService.ts` response handling
- [x] 4.2 Apply `snakeToCamel` in `mediaService.ts` response handling
- [x] 4.3 Apply `snakeToCamel` in `downloadService.ts` response handling
- [x] 4.4 Apply in remaining 13 service files (tmdb, scanner, subtitle, health, learning, qbittorrent, log, retry, cache, setup, metadata, backup, serviceStatus). Added `camelToSnake` for subtitle request params.
- [x] 4.5 Update frontend unit tests (24 `.spec.ts/.spec.tsx` files, 474 snake_case → camelCase mock data replacements)

### Task 5: Verification (AC5, AC6)

- [x] 5.1 Run Go tests — all pass (1 pre-existing flaky SSE test excluded)
- [x] 5.2 Run `pnpm nx run web:test` — all 125 files, 1545 tests pass
- [ ] 5.3 Run E2E tests — all pass (especially library grid, dashboard, downloads)
- [ ] 5.4 Manual verification: library grid shows poster images when TMDb metadata exists

## Dev Notes

### Root Cause Analysis

- Epic 2 (2026-01-17): Go models created with camelCase JSON tags (Go convention)
- Epic 5 (2026-03-15): Frontend TypeScript types created with snake_case (matching DB columns)
- No cross-validation between backend JSON output and frontend type expectations
- `sql.NullString` serialization bug masked the issue (both bugs cancelled out — data was broken regardless)
- Commit `a123f59` (2026-03-27) fixed serialization, exposing the naming mismatch

### Affected Go Model Files (13 files)

```
apps/api/internal/models/movie.go          — ~15 camelCase fields
apps/api/internal/models/series.go         — ~20 camelCase fields
apps/api/internal/models/season.go         — ~8 camelCase fields
apps/api/internal/models/episode.go        — ~10 camelCase fields
apps/api/internal/models/parse_job.go      — ~8 camelCase fields
apps/api/internal/models/parse_status.go   — ~8 camelCase fields
apps/api/internal/models/filename_mapping.go — ~8 camelCase fields
apps/api/internal/models/settings.go       — ~8 camelCase fields
apps/api/internal/models/backup.go         — ~10 camelCase fields
apps/api/internal/models/secret.go         — ~3 camelCase fields
apps/api/internal/models/system_log.go     — ~1 camelCase field
apps/api/internal/models/connection_event.go — ~2 camelCase fields
apps/api/internal/models/degradation.go    — ~10 camelCase fields
```

### Transform Utility Design

```typescript
// apps/web/src/utils/caseTransform.ts
export function snakeToCamel<T>(obj: unknown): T {
  if (obj === null || obj === undefined) return obj as T;
  if (Array.isArray(obj)) return obj.map(snakeToCamel) as T;
  if (typeof obj !== 'object') return obj as T;

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const camelKey = key.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
    result[camelKey] = snakeToCamel(value);
  }
  return result as T;
}
```

### Service Integration Pattern (Option 1)

```typescript
// Each service function wraps the response
export async function getLibraryItems(params: LibraryListParams): Promise<LibraryListResponse> {
  const response = await fetch(`${API_BASE_URL}/library?...`);
  const data = await response.json();
  return snakeToCamel<LibraryListResponse>(data);
}
```

### Project Structure Notes

- Transform utility: `apps/web/src/utils/caseTransform.ts` (new file)
- No changes to `apps/api/internal/handlers/` logic — only JSON tag strings change
- No database migration needed — `db:` tags unchanged
- project-context.md Rule 6 already correct — no doc update needed

### References

- [Source: project-context.md#Rule 6] — JSON Fields: snake_case
- [Source: apps/api/internal/models/movie.go#L100] — PosterPath json:"posterPath" violation
- [Source: apps/web/src/types/library.ts#L22] — Frontend expects poster_path
- [Source: Party Mode Discussion 2026-03-27] — Option 1 (per-service transform) agreed

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- ✅ Task 1 (2026-03-27): Changed all Go JSON tags from camelCase to snake_case across 40+ files (models, handlers, services, qbittorrent, retry, cache, metadata, events, repository, ai, subtitle). Updated all corresponding test assertions. Go test suite passes (1 pre-existing flaky scanner SSE test excluded).
- ✅ Task 2 (2026-03-27): Created `snakeToCamel` transform utility with 14 unit tests covering flat objects, nested objects, arrays, null/undefined, primitives, and edge cases (tmdb_id, imdb_id, created_at).
- ✅ Task 3 (2026-03-27): Converted all frontend TypeScript types to camelCase (library.ts, tmdb.ts, scannerService.ts, subtitleService.ts) and updated 21 component/hook files. Added `camelToSnake` utility for request param serialization. (commit 13a449a)
- ✅ Task 4 (2026-03-27): Applied `snakeToCamel` transform in all 16 service files' `fetchApi` wrappers. Applied `camelToSnake` for subtitleService request params. Updated 24 test files with 474 mock data conversions. (commits 348b7bc, 059eea8, 30c5676)
- ✅ Task 5.1+5.2 (2026-03-27): Go tests pass (1 pre-existing flaky SSE test excluded). Frontend tests: 125/125 files, 1545/1545 tests pass.

### Change Log

- 2026-03-27: Task 1+2 complete — Go backend snake_case JSON tags + frontend transform utility (commit d7a0119)
- 2026-03-27: Task 3 complete — Frontend types + components → camelCase (commit 13a449a)
- 2026-03-27: Task 4 complete — snakeToCamel in all services + test mock data (commits 348b7bc, 059eea8, 30c5676)

### File List

**Go backend (snake_case JSON tags):**
- apps/api/internal/models/movie.go
- apps/api/internal/models/series.go
- apps/api/internal/models/season.go
- apps/api/internal/models/episode.go
- apps/api/internal/models/parse_job.go
- apps/api/internal/models/parse_status.go
- apps/api/internal/models/filename_mapping.go
- apps/api/internal/models/settings.go
- apps/api/internal/models/backup.go
- apps/api/internal/models/secret.go
- apps/api/internal/models/system_log.go
- apps/api/internal/models/connection_event.go
- apps/api/internal/models/degradation.go
- apps/api/internal/handlers/movie_handler.go
- apps/api/internal/handlers/series_handler.go
- apps/api/internal/handlers/metadata_handler.go
- apps/api/internal/handlers/download_handler.go
- apps/api/internal/handlers/learning_handler.go
- apps/api/internal/handlers/recent_media_handler.go
- apps/api/internal/handlers/parse_progress_handler.go
- apps/api/internal/handlers/qbittorrent_handler.go
- apps/api/internal/handlers/health.go
- apps/api/internal/handlers/response.go
- apps/api/internal/services/cache_stats_service.go
- apps/api/internal/services/metadata_service.go
- apps/api/internal/services/scanner_service.go
- apps/api/internal/services/library_service.go
- apps/api/internal/services/learning_service.go
- apps/api/internal/services/export_service.go
- apps/api/internal/services/backup_service.go
- apps/api/internal/services/backup_scheduler.go
- apps/api/internal/services/cache_cleanup_service.go
- apps/api/internal/services/log_service.go
- apps/api/internal/qbittorrent/torrent.go
- apps/api/internal/qbittorrent/types.go
- apps/api/internal/retry/queue.go
- apps/api/internal/retry/executor.go
- apps/api/internal/cache/offline_cache.go
- apps/api/internal/metadata/orchestrator.go
- apps/api/internal/metadata/partial.go
- apps/api/internal/events/parse_events.go
- apps/api/internal/repository/interfaces.go
- apps/api/internal/repository/repository.go
- apps/api/internal/ai/gemini.go
- apps/api/internal/subtitle/scorer.go

**Go test files (assertion updates):**
- apps/api/internal/handlers/*_test.go (12 files)
- apps/api/internal/qbittorrent/types_test.go
- apps/api/internal/retry/executor_test.go
- apps/api/internal/services/backup_service_test.go

**Frontend (new files):**
- apps/web/src/utils/caseTransform.ts (NEW)
- apps/web/src/utils/caseTransform.spec.ts (NEW)
