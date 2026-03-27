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

- [ ] 1.1 Update `movie.go` JSON tags: all camelCase → snake_case (match `db:` tags)
- [ ] 1.2 Update `series.go` JSON tags (including `SeriesBasicInfo`, `Network` embedded structs)
- [ ] 1.3 Update `season.go` JSON tags
- [ ] 1.4 Update `episode.go` JSON tags
- [ ] 1.5 Update `parse_job.go` JSON tags
- [ ] 1.6 Update `parse_status.go` JSON tags
- [ ] 1.7 Update `filename_mapping.go` JSON tags
- [ ] 1.8 Update `settings.go` JSON tags
- [ ] 1.9 Update `backup.go` JSON tags
- [ ] 1.10 Update `secret.go` JSON tags
- [ ] 1.11 Update `system_log.go` JSON tags
- [ ] 1.12 Update `connection_event.go` JSON tags
- [ ] 1.13 Update `degradation.go` JSON tags
- [ ] 1.14 Update Go handler tests that assert JSON field names (grep for camelCase field assertions in `*_handler_test.go`)

### Task 2: Frontend transform utility (AC2, AC7)

- [ ] 2.1 Create `apps/web/src/utils/caseTransform.ts` with `snakeToCamel(obj)` function
- [ ] 2.2 Handle edge cases: nested objects, arrays of objects, null/undefined, primitive values
- [ ] 2.3 Create `apps/web/src/utils/caseTransform.spec.ts` with comprehensive tests
- [ ] 2.4 Test edge cases: `tmdb_id` → `tmdbId`, `imdb_id` → `imdbId`, `created_at` → `createdAt`

### Task 3: Frontend TypeScript types → camelCase (AC4)

- [ ] 3.1 Update `apps/web/src/types/library.ts` — `LibraryMovie` and `LibrarySeries` interfaces: `poster_path` → `posterPath`, `release_date` → `releaseDate`, etc.
- [ ] 3.2 Update any other TypeScript type files that reference snake_case API fields
- [ ] 3.3 Update component references to use new camelCase field names

### Task 4: Frontend service integration (AC3)

- [ ] 4.1 Apply `snakeToCamel` in `libraryService.ts` response handling
- [ ] 4.2 Apply `snakeToCamel` in `mediaService.ts` response handling
- [ ] 4.3 Apply `snakeToCamel` in `downloadService.ts` response handling (if needed — check if download API uses Go models)
- [ ] 4.4 Apply in remaining service files that consume Go API responses
- [ ] 4.5 Update frontend unit tests (`.spec.ts/.spec.tsx`) to use camelCase mock data

### Task 5: Verification (AC5, AC6)

- [ ] 5.1 Run `go test ./apps/api/...` — all pass
- [ ] 5.2 Run `pnpm nx run web:test` — all pass
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

### Debug Log References

### Completion Notes List

### File List
