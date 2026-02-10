# Story 5.5: Library Filtering

Status: ready-for-dev

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
     - Media Type (Movie, TV Show, Anime — radio/checkbox)

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

- [ ] Task 1: Add Filter Support to Library API (AC: 1, 2, 4)
  - [ ] 1.1: Extend `ListParams.Filters` map to support: `genre`, `year_min`, `year_max`, `media_type`
  - [ ] 1.2: Implement genre filtering in MovieRepository.List() and SeriesRepository.List()
  - [ ] 1.3: Implement year range filtering (WHERE release_date >= ? AND release_date <= ?)
  - [ ] 1.4: Add `GET /api/v1/library/genres` endpoint — return distinct genres from movies + series
  - [ ] 1.5: Add `GET /api/v1/library/stats` endpoint — return year range, total counts by type
  - [ ] 1.6: Write repository filter tests
  - [ ] 1.7: Write handler tests

- [ ] Task 2: Create Filter Panel Component (AC: 1, 3)
  - [ ] 2.1: Create `/apps/web/src/components/library/FilterPanel.tsx`
  - [ ] 2.2: Genre section: multi-select checkboxes, dynamically populated from API
  - [ ] 2.3: Year range section: min/max number inputs
  - [ ] 2.4: Media type section: checkboxes for Movie / TV Show
  - [ ] 2.5: "套用篩選" (Apply) and "清除" (Clear) buttons
  - [ ] 2.6: Collapsible panel (toggle open/close)
  - [ ] 2.7: Write component tests

- [ ] Task 3: Create Filter Chips Component (AC: 3)
  - [ ] 3.1: Create `/apps/web/src/components/library/FilterChips.tsx`
  - [ ] 3.2: Render active filters as removable chip/tag elements
  - [ ] 3.3: Click ✕ on chip removes that filter
  - [ ] 3.4: "清除全部" button clears all filters
  - [ ] 3.5: Write component tests

- [ ] Task 4: Create Filter Hooks & Service (AC: 1, 2, 4)
  - [ ] 4.1: Add `useLibraryGenres()` hook — fetches available genres
  - [ ] 4.2: Add `useLibraryStats()` hook — fetches year range, counts
  - [ ] 4.3: Add filter params to `useLibraryList` query key
  - [ ] 4.4: Add `getGenres()` and `getStats()` to libraryService.ts

- [ ] Task 5: Integrate Filters into Library Route (AC: 2, 4)
  - [ ] 5.1: Add filter SearchParams: `genres`, `yearMin`, `yearMax`, `mediaType`
  - [ ] 5.2: Serialize genres as comma-separated in URL: `?genres=科幻,動作`
  - [ ] 5.3: Pass filter params to library hooks
  - [ ] 5.4: Reset to page 1 when filters change
  - [ ] 5.5: Show "顯示 N / Total 項" count header

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
