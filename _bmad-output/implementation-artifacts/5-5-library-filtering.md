# Story 5.5: Library Filtering

Status: done

## Story

As a **media collector**,
I want to **filter my library by genre, year, and media type**,
So that **I can narrow down to specific categories**.

## Acceptance Criteria

1. **AC1: Filter Panel Options**
   - Given the library is displayed
   - When the user opens the filter panel
   - Then filter options include:
     - Genre (multi-select checkboxes)
     - Year range (min/max inputs or slider)
     - Media Type (Movie, TV Show — segmented control)

2. **AC2: Filter Application**
   - Given filters are applied
   - When the library updates
   - Then only matching items are displayed
   - And the filter count is shown: "顯示 45 / 500 項"

3. **AC3: Active Filter Chips**
   - Given multiple filters are active
   - When viewing the filter status
   - Then active filters are shown as removable chips/tags
   - And a "清除全部篩選" (Clear all filters) option is available

4. **AC4: Combined Filters**
   - Given multiple filters are selected
   - When applied together
   - Then filters combine with AND logic
   - And URL state reflects all active filters for sharing/bookmarking

## Tasks / Subtasks

- [x] Task 1: Add Filter Support to Library API (AC: 1, 2, 4)
  - [x] 1.1: Extend `ListParams.Filters` map to support: `genres`, `year_min`, `year_max`
  - [x] 1.2: Implement genre filtering in MovieRepository.List() and SeriesRepository.List()
  - [x] 1.3: Implement year range filtering (WHERE substr(release_date,1,4) >= ? AND <= ?)
  - [x] 1.4: Add `GET /api/v1/library/genres` endpoint — return distinct genres from movies + series
  - [x] 1.5: Add `GET /api/v1/library/stats` endpoint — return year range, total counts by type
  - [x] 1.6: Write repository filter tests (integration tests with real DB)
  - [x] 1.7: Write handler tests (mock service)

- [x] Task 2: Create Filter Panel Component (AC: 1, 3)
  - [x] 2.1: Create `/apps/web/src/components/library/FilterPanel.tsx`
  - [x] 2.2: Genre section: multi-select checkboxes, dynamically populated from API
  - [x] 2.3: Year range section: min/max number inputs
  - [x] 2.4: Media type section: handled via existing type tabs (all/movie/tv)
  - [x] 2.5: "套用篩選" (Apply) and "清除" (Clear) buttons
  - [x] 2.6: Collapsible panel (toggle open/close)
  - [x] 2.7: Write component tests (9 tests)

- [x] Task 3: Create Filter Chips Component (AC: 3)
  - [x] 3.1: Create `/apps/web/src/components/library/FilterChips.tsx`
  - [x] 3.2: Render active filters as removable chip/tag elements
  - [x] 3.3: Click ✕ on chip removes that filter
  - [x] 3.4: "清除全部篩選" button clears all filters
  - [x] 3.5: Write component tests (7 tests)

- [x] Task 4: Create Filter Hooks & Service (AC: 1, 2, 4)
  - [x] 4.1: Add `useLibraryGenres()` hook — fetches available genres (5min stale)
  - [x] 4.2: Add `useLibraryStats()` hook — fetches year range, counts (1min stale)
  - [x] 4.3: Add filter params (genres, yearMin, yearMax) to `useLibraryList` query key
  - [x] 4.4: Add `getGenres()` and `getStats()` to libraryService.ts

- [x] Task 5: Integrate Filters into Library Route (AC: 2, 4)
  - [x] 5.1: Add filter SearchParams: `genres`, `yearMin`, `yearMax`
  - [x] 5.2: Serialize genres as comma-separated in URL: `?genres=科幻,動作`
  - [x] 5.3: Pass filter params to library hooks
  - [x] 5.4: Reset to page 1 when filters change
  - [x] 5.5: Show "顯示 N / Total 項" count header (already existed)

## Dev Notes

### Architecture Requirements

**FR7:** Filter media library by genre, year, media type
Filters work in combination (AND logic)
Filter state persisted in URL for sharing

### Existing Code to Reuse (DO NOT Reinvent)

- `ListParams.Filters` map — already supports arbitrary key-value filters
- `MovieRepository.List()` / `SeriesRepository.List()` — extend WHERE clauses
- Genres are stored as JSON array in movies/series tables — use JSON extraction for filtering
- Library route SearchParams pattern from Stories 5-1 through 5-4

### Backend Filter Implementation

```go
// In movie_repository.go List() method, extend query building:
if genre, ok := params.Filters["genre"]; ok {
    // SQLite JSON: WHERE json_each.value = ?
    // JOIN json_each(genres) ON json_each.value LIKE '%genre%'
    query += " AND genres LIKE ?"
    args = append(args, "%"+genre+"%")
}
if yearMin, ok := params.Filters["year_min"]; ok {
    query += " AND substr(release_date, 1, 4) >= ?"
    args = append(args, yearMin)
}
if yearMax, ok := params.Filters["year_max"]; ok {
    query += " AND substr(release_date, 1, 4) <= ?"
    args = append(args, yearMax)
}
```

```go
// GET /api/v1/library/genres
// Returns distinct genres across all library items
func (h *LibraryHandler) GetGenres(c *gin.Context) {
    genres, err := h.service.GetDistinctGenres(c.Request.Context())
    // Returns: ["科幻", "動作", "劇情", "恐怖", ...]
    SuccessResponse(c, genres)
}
```

### Frontend URL State Pattern

```typescript
type LibrarySearchParams = {
  page?: number;
  pageSize?: number;
  type?: 'all' | 'movie' | 'tv';
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  q?: string;
  genres?: string;   // comma-separated: "科幻,動作"
  yearMin?: number;
  yearMax?: number;
};
```

### Project Structure Notes

```
Backend (extend):
/apps/api/internal/repository/movie_repository.go  ← ADD filter WHERE clauses
/apps/api/internal/repository/series_repository.go ← ADD filter WHERE clauses
/apps/api/internal/services/library_service.go     ← ADD GetDistinctGenres, GetStats
/apps/api/internal/handlers/library_handler.go     ← ADD GetGenres, GetStats endpoints

Frontend (new):
/apps/web/src/components/library/FilterPanel.tsx       ← NEW
/apps/web/src/components/library/FilterPanel.spec.tsx  ← NEW
/apps/web/src/components/library/FilterChips.tsx       ← NEW
/apps/web/src/components/library/FilterChips.spec.tsx  ← NEW

Frontend (modify):
/apps/web/src/routes/library.tsx         ← ADD filter params
/apps/web/src/hooks/useLibrary.ts        ← ADD useLibraryGenres, useLibraryStats, filter params
/apps/web/src/services/libraryService.ts ← ADD getGenres, getStats, filter params
```

### Dependencies

- Story 5-1 (Media Library Grid View) — library page and API must exist

### Testing Strategy

- Backend: genre filter returns matching items, year range works, combined filters use AND logic
- FilterPanel: renders genre checkboxes, year inputs, applies/clears
- FilterChips: renders active filters, remove chip updates filters
- Integration: filter changes update URL and re-fetch filtered results

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.5]
- [Source: _bmad-output/planning-artifacts/prd.md#FR7]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Advanced-Filtering-Sorting]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Backend API — Extended repository List() methods with genre (JSON LIKE) and year range (substr) filtering. Added GetDistinctGenres(), GetYearRange(), Count() to both MovieRepository and SeriesRepository. Added GetDistinctGenres() and GetLibraryStats() to LibraryService. Added GET /api/v1/library/genres and GET /api/v1/library/stats endpoints. Handler parses genres (comma-separated), year_min, year_max query params with validation. All 20 backend test packages pass.
- Task 2: FilterPanel component — Collapsible panel with genre multi-select checkboxes (API-driven), year range number inputs, apply/clear buttons. Shows active filter count badge. 9 unit tests.
- Task 3: FilterChips component — Renders genre chips (blue) and year range chips (green) with ✕ remove buttons. "清除全部篩選" clear-all button. Returns null when no filters active. 7 unit tests.
- Task 4: Hooks & Service — useLibraryGenres (5min stale), useLibraryStats (1min stale), filter params in useLibraryList query key. getGenres() and getStats() in libraryService.
- Task 5: Route integration — Filter state persisted in URL (genres=comma-separated, yearMin, yearMax). Filter panel and chips rendered in toolbar. Reset page to 1 on filter change. Recently-added section hidden when filters active.

### Change Log

- 2026-03-15: Implemented Story 5-5 Library Filtering — full-stack genre/year/type filtering with URL state persistence
- 2026-03-15: Code Review fixes (H1: listAll pagination, H2: AC2 filter count display, H3: useLibraryStats wired, M1: decade auto-fill, M2: year validation)

## Senior Developer Review (AI)

**Reviewer:** Amelia (Dev Agent) — 2026-03-15
**Outcome:** Approved with fixes applied

### Issues Found & Fixed (5/7)

| ID | Severity | Description | Status |
|----|----------|-------------|--------|
| H1 | HIGH | `listAll` pagination broken for page>1 — repos paginated independently before merge | FIXED |
| H2 | HIGH | AC2 "顯示 N / Total 項" filter count not displayed anywhere | FIXED |
| H3 | HIGH | `useLibraryStats` hook defined but never consumed | FIXED (wired to filter count) |
| M1 | MEDIUM | Non-contiguous decade selection merged as contiguous range | FIXED (auto-fill gaps) |
| M2 | MEDIUM | year_min/year_max accepts invalid values (negative, inverted range) | FIXED (range 1888-2100, order check) |
| M3 | MEDIUM | GetDistinctGenres full table scan O(N) | DEFERRED (acceptable at NAS scale) |
| L1 | LOW | Genre LIKE matching edge cases with special chars | ACCEPTED (impractical in production) |

### Files Changed in Review

- apps/api/internal/services/library_service.go — Fixed listAll pagination (fetch page*pageSize then slice)
- apps/api/internal/handlers/library_handler.go — Added year range validation (1888-2100, min<=max)
- apps/api/internal/handlers/library_handler_test.go — Added 3 new validation tests
- apps/web/src/components/library/FilterPanel.tsx — Added normalizeDecadeSelection auto-fill
- apps/web/src/components/library/FilterPanel.spec.tsx — Added auto-fill decade test
- apps/web/src/routes/library.tsx — Added useLibraryStats + "顯示 N / Total 項" display

### File List

Backend (modified):
- apps/api/internal/repository/movie_repository.go — genre/year filter WHERE clauses, GetDistinctGenres, GetYearRange, Count
- apps/api/internal/repository/series_repository.go — genre/year filter WHERE clauses, GetDistinctGenres, GetYearRange, Count
- apps/api/internal/repository/interfaces.go — added 3 new methods to MovieRepositoryInterface and SeriesRepositoryInterface
- apps/api/internal/services/library_service.go — GetDistinctGenres, GetLibraryStats, LibraryStats type
- apps/api/internal/handlers/library_handler.go — GetGenres, GetStats handlers, filter param parsing, route registration
- apps/api/internal/handlers/library_handler_test.go — mock update + 12 new tests (filters, genres, stats)
- apps/api/internal/services/library_service_test.go — 13 new integration tests (filter, genres, stats, combined)
- apps/api/internal/services/movie_service_test.go — mock update (3 new methods)
- apps/api/internal/services/series_service_test.go — mock update (3 new methods)
- apps/api/internal/services/parse_queue_service_test.go — mock update (3 new methods each)

Frontend (new):
- apps/web/src/components/library/FilterPanel.tsx — collapsible filter panel
- apps/web/src/components/library/FilterPanel.spec.tsx — 9 tests
- apps/web/src/components/library/FilterChips.tsx — removable filter chips
- apps/web/src/components/library/FilterChips.spec.tsx — 7 tests

Frontend (modified):
- apps/web/src/types/library.ts — LibraryListParams (genres, yearMin, yearMax), LibraryStats type
- apps/web/src/services/libraryService.ts — filter params in listLibrary, getGenres, getStats
- apps/web/src/hooks/useLibrary.ts — useLibraryGenres, useLibraryStats, query key with filters
- apps/web/src/routes/library.tsx — filter URL params, FilterPanel/FilterChips integration, handlers
