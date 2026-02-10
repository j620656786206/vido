# Story 5.3: Library Search

Status: ready-for-dev

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

- [ ] Task 1: Create Library Search API Endpoint (AC: 1, 2)
  - [ ] 1.1: Add `GET /api/v1/library/search?q=X&page=1&page_size=20&type=all` endpoint
  - [ ] 1.2: Use existing `LibraryService.SearchLibrary()` which calls FTS5
  - [ ] 1.3: Return combined movie + series results with pagination
  - [ ] 1.4: Ensure FTS5 searches title, original_title, overview fields
  - [ ] 1.5: Write handler tests

- [ ] Task 2: Create Library Search Hook (AC: 1, 2)
  - [ ] 2.1: Add `useLibrarySearch(query, params)` to `/apps/web/src/hooks/useLibrary.ts`
  - [ ] 2.2: Query key: `['library', 'search', query, params]`
  - [ ] 2.3: Only trigger when query.length ≥ 2
  - [ ] 2.4: staleTime: 60s, gcTime: 5min
  - [ ] 2.5: Add `searchLibrary(query, params)` to libraryService.ts

- [ ] Task 3: Create Library Search Bar Component (AC: 1, 4)
  - [ ] 3.1: Create `/apps/web/src/components/library/LibrarySearchBar.tsx`
  - [ ] 3.2: Search icon left, clear button right (appears when value exists)
  - [ ] 3.3: Placeholder: "搜尋媒體標題..."
  - [ ] 3.4: 500ms debounce on input change
  - [ ] 3.5: Add Ctrl+K global shortcut to focus search
  - [ ] 3.6: Show result count below: "找到 N 個結果"
  - [ ] 3.7: Write component tests

- [ ] Task 4: Create No Results Component (AC: 3)
  - [ ] 4.1: Create `/apps/web/src/components/library/EmptySearchResults.tsx`
  - [ ] 4.2: Show 🔍 icon + "找不到相關結果" message
  - [ ] 4.3: Show search query context: "搜尋「{query}」沒有找到匹配的電影或影集"
  - [ ] 4.4: Bullet suggestions: different keywords, zh-TW/English, check spelling
  - [ ] 4.5: Clear search button
  - [ ] 4.6: Fade in after 500ms delay

- [ ] Task 5: Integrate Search into Library Page (AC: 1, 2, 3, 4)
  - [ ] 5.1: Add `q` search param to library route
  - [ ] 5.2: When search is active, use `useLibrarySearch` instead of `useLibraryList`
  - [ ] 5.3: Pass search results to same LibraryGrid/LibraryTable components
  - [ ] 5.4: Show EmptySearchResults when no matches

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
