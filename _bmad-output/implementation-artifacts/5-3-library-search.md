# Story 5.3: Library Search

Status: review

## Story

As a **media collector**,
I want to **search within my saved media library**,
So that **I can quickly find specific titles in my collection**.

## Acceptance Criteria

1. **AC1: Real-Time Search Filtering**
   - Given the user is on the Library page
   - When they type in the search box (≥2 characters)
   - Then results filter in real-time with 500ms debounce
   - And both Chinese and English titles are searched

2. **AC2: Search Result Highlighting**
   - Given a search query is entered
   - When results are displayed
   - Then matching terms are highlighted (bold or mark)
   - And search completes within 500ms (NFR-SC8)

3. **AC3: No Results State**
   - Given no results match the query
   - When the search completes
   - Then a friendly message shows: "找不到相關結果。試試不同的關鍵字或新增媒體。"
   - And suggestions list: different keywords, use zh-TW or English, check spelling

4. **AC4: Search UX**
   - Given the search box is displayed
   - When interacting with search
   - Then search icon (🔍) is left-aligned
   - And clear button (✕) appears when input has value
   - And result count shows: "找到 15 個結果"
   - And Ctrl+K keyboard shortcut focuses search

## Tasks / Subtasks

- [x] Task 1: Create Library Search API Endpoint (AC: 1, 2)
  - [x] 1.1: Add `GET /api/v1/library/search?q=X&page=1&page_size=20&type=all` endpoint
  - [x] 1.2: Use existing `LibraryService.SearchLibrary()` which calls FTS5
  - [x] 1.3: Return combined movie + series results with pagination
  - [x] 1.4: Ensure FTS5 searches title, original_title, overview fields
  - [x] 1.5: Write handler tests

- [x] Task 2: Create Library Search Hook (AC: 1, 2)
  - [x] 2.1: Add `useLibrarySearch(query, params)` to `/apps/web/src/hooks/useLibrary.ts`
  - [x] 2.2: Query key: `['library', 'search', query, params]`
  - [x] 2.3: Only trigger when query.length ≥ 2
  - [x] 2.4: staleTime: 60s, gcTime: 5min
  - [x] 2.5: Add `searchLibrary(query, params)` to libraryService.ts

- [x] Task 3: Create Library Search Bar Component (AC: 1, 4)
  - [x] 3.1: Create `/apps/web/src/components/library/LibrarySearchBar.tsx`
  - [x] 3.2: Search icon left, clear button right (appears when value exists)
  - [x] 3.3: Placeholder: "搜尋媒體標題..."
  - [x] 3.4: 500ms debounce on input change
  - [x] 3.5: Add Ctrl+K global shortcut to focus search
  - [x] 3.6: Show result count below: "找到 N 個結果"
  - [x] 3.7: Write component tests

- [x] Task 4: Create No Results Component (AC: 3)
  - [x] 4.1: Create `/apps/web/src/components/library/EmptySearchResults.tsx`
  - [x] 4.2: Show 🔍 icon + "找不到相關結果" message
  - [x] 4.3: Show search query context: "搜尋「{query}」沒有找到匹配的電影或影集"
  - [x] 4.4: Bullet suggestions: different keywords, zh-TW/English, check spelling
  - [x] 4.5: Clear search button
  - [x] 4.6: Fade in after 500ms delay

- [x] Task 5: Integrate Search into Library Page (AC: 1, 2, 3, 4)
  - [x] 5.1: Add `q` search param to library route
  - [x] 5.2: When search is active, use `useLibrarySearch` instead of `useLibraryList`
  - [x] 5.3: Pass search results to same LibraryGrid/LibraryTable components
  - [x] 5.4: Show EmptySearchResults when no matches

## Dev Notes

### Architecture Requirements

**FR5:** Search within saved media library
**NFR-SC8:** SQLite FTS5 full-text search, <500ms on 10,000 items

### Existing Code to Reuse (DO NOT Reinvent)

**Backend — FTS5 already built:**
- `MovieRepository.FullTextSearch(ctx, query, params)` in `movie_repository.go` — FTS5 query with ranking
- `SeriesRepository.FullTextSearch(ctx, query, params)` in `series_repository.go`
- `LibraryService.SearchLibrary(ctx, query, params)` in `library_service.go` — unified search
- FTS5 virtual tables: `movies_fts`, `series_fts` (created in migration 006)
- Triggers auto-sync FTS on INSERT/UPDATE/DELETE

**Frontend — Search patterns exist:**
- `SearchBar` in `/apps/web/src/components/search/SearchBar.tsx` — reference for debounce pattern
- `useSearchMovies` / `useSearchTVShows` in `hooks/useSearchMedia.ts` — query pattern reference
- Debounce pattern already used: 500ms, min 2 chars, cancel previous request

### Backend API

```
GET /api/v1/library/search?q=駭客&page=1&page_size=20&type=all

Response:
{
  "success": true,
  "data": {
    "movies": [...],
    "series": [...],
    "pagination": { "page": 1, "page_size": 20, "total_results": 15, "total_pages": 1 }
  }
}
```

### Frontend Debounce Pattern

```typescript
import { useMemo, useEffect, useState } from 'react';
import { useDebouncedValue } from '@/hooks/useDebouncedValue'; // or inline

function useLibrarySearch(query: string, params: LibraryParams) {
  const debouncedQuery = useDebouncedValue(query, 500);

  return useQuery({
    queryKey: libraryKeys.search(debouncedQuery, params),
    queryFn: () => libraryService.searchLibrary(debouncedQuery, params),
    enabled: debouncedQuery.length >= 2,
    staleTime: 60 * 1000,
  });
}
```

### Keyboard Shortcut Pattern

```typescript
useEffect(() => {
  const handler = (e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      searchInputRef.current?.focus();
    }
  };
  document.addEventListener('keydown', handler);
  return () => document.removeEventListener('keydown', handler);
}, []);
```

### Project Structure Notes

```
Backend (extend):
/apps/api/internal/handlers/library_handler.go ← ADD SearchLibrary handler

Frontend (new):
/apps/web/src/components/library/LibrarySearchBar.tsx        ← NEW
/apps/web/src/components/library/LibrarySearchBar.spec.tsx   ← NEW
/apps/web/src/components/library/EmptySearchResults.tsx      ← NEW
/apps/web/src/components/library/EmptySearchResults.spec.tsx ← NEW

Frontend (modify):
/apps/web/src/routes/library.tsx        ← ADD q param, search integration
/apps/web/src/hooks/useLibrary.ts       ← ADD useLibrarySearch
/apps/web/src/services/libraryService.ts ← ADD searchLibrary
```

### Dependencies

- Story 5-1 (Media Library Grid View) — library page and API must exist

### Testing Strategy

- Backend: FTS5 search returns ranked results, handles Chinese + English, pagination
- LibrarySearchBar: debounce fires after 500ms, clear resets, Ctrl+K focuses
- EmptySearchResults: renders message with query, shows suggestions
- Integration: search filters grid results in real-time

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.3]
- [Source: _bmad-output/planning-artifacts/prd.md#FR5]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-SC8]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Real-time-Search-Pattern]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- **Task 1:** Added `SearchLibrary` handler to `library_handler.go` with `GET /api/v1/library/search` route. Validates query ≥2 chars, type filter (all/movie/tv), pagination. Delegates to existing `LibraryService.SearchLibrary()` which uses FTS5. 8 handler tests written and passing.
- **Task 2:** Added `useLibrarySearch` hook to `useLibrary.ts` with query key `['library', 'search', query, params]`, enabled when query ≥2 chars, staleTime 60s, gcTime 5min. Added `searchLibrary()` to `libraryService.ts` and `LibrarySearchResponse` type.
- **Task 3:** Created `LibrarySearchBar` component with Search icon (left), clear button (right, conditional), 500ms debounce, Ctrl+K/Cmd+K focus shortcut, result count display. 16 tests written and passing.
- **Task 4:** Created `EmptySearchResults` component with search icon, "找不到相關結果" message, query context, bullet suggestions, clear button, fade-in animation. 6 tests written and passing.
- **Task 5:** Integrated search into library route — added `q` search param, conditional `useLibrarySearch` vs `useLibraryList`, search result → LibraryItem conversion for grid/table reuse, EmptySearchResults display. All 16 existing library route tests pass without regression.

### File List

- `apps/api/internal/handlers/library_handler.go` — MODIFIED (added SearchLibrary handler + route)
- `apps/api/internal/handlers/library_handler_test.go` — MODIFIED (added 8 search handler tests)
- `apps/web/src/components/library/LibrarySearchBar.tsx` — NEW
- `apps/web/src/components/library/LibrarySearchBar.spec.tsx` — NEW (16 tests)
- `apps/web/src/components/library/EmptySearchResults.tsx` — NEW
- `apps/web/src/components/library/EmptySearchResults.spec.tsx` — NEW (6 tests)
- `apps/web/src/routes/library.tsx` — MODIFIED (search integration, q param)
- `apps/web/src/hooks/useLibrary.ts` — MODIFIED (added useLibrarySearch, search keys)
- `apps/web/src/services/libraryService.ts` — MODIFIED (added searchLibrary)
- `apps/web/src/types/library.ts` — MODIFIED (added LibrarySearchResult, LibrarySearchResponse)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — MODIFIED (5-3 status)
- `_bmad-output/implementation-artifacts/5-3-library-search.md` — MODIFIED (this file)

## Change Log

- 2026-03-15: Implemented all 5 tasks for Story 5-3 Library Search. Backend search endpoint, frontend hook/service, search bar component with debounce/keyboard shortcut, no-results component, and full library page integration. 30 new tests (8 backend + 22 frontend) all passing.
