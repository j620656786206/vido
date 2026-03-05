# Story 5.1: Media Library Grid View

Status: ready-for-dev

## Story

As a **media collector**,
I want to **browse my media library in a visual grid**,
So that **I can enjoy seeing my collection with beautiful posters**.

## Acceptance Criteria

1. **AC1: Library Page with Responsive Grid**
   - Given the user opens the Library page
   - When the page loads
   - Then media items display in a responsive grid (2 cols mobile, 3-4 tablet, 5-6 desktop)
   - And each card shows: poster (2:3 ratio), title (zh-TW), year, rating

2. **AC2: Virtual Scrolling for Large Libraries**
   - Given the library has more than 1,000 items
   - When scrolling through the grid
   - Then virtual scrolling is enabled (NFR-SC6)
   - And scrolling maintains 60 FPS (NFR-P10)

3. **AC3: Poster Card Hover States**
   - Given the grid is displayed
   - When hovering over a card (desktop)
   - Then card scales to 1.05 with shadow-xl
   - And overlay shows: rating, genre, description preview, metadata source badge
   - And animation completes within 300ms ease-out

4. **AC4: Pagination Support**
   - Given the library has many items
   - When viewing the grid
   - Then pagination controls are available (page, page_size via URL params)
   - And total count is displayed ("顯示 1-20 / 500 項")

5. **AC5: Empty State**
   - Given the library has no media
   - When the page loads
   - Then a welcoming empty state shows with setup guidance
   - And includes CTA to navigate to search/parse

6. **AC6: Loading States**
   - Given the library is loading
   - When data is being fetched
   - Then skeleton cards with pulsing animation are displayed
   - And the grid layout is preserved during loading

7. **AC7: PosterCard Context Menu**
   - Given the grid is displayed
   - When the user clicks the `...` (three-dot) icon on a poster card (appears on hover, top-right)
   - Then a context menu opens with the following items (Epic 5 scope):
     - View Details (Lucide: `Eye`) — opens Detail Panel (Story 5.6)
     - Re-parse Metadata (Lucide: `RefreshCw`) — re-parse this item (FR40)
     - Export Metadata (Lucide: `Download`) — export this item (FR40)
     - *(separator)*
     - Delete (Lucide: `Trash2`, `--error` red color) — remove from library, requires confirmation dialog
   - And the menu dismisses when clicking outside
   - And on mobile, the context menu triggers via long-press and presents as a bottom sheet
   - And single-item operations (re-parse, export, delete) call individual API endpoints

8. **AC8: Settings Gear Dropdown**
   - Given the library toolbar is displayed
   - When the user clicks the Settings gear icon (Lucide: `Settings`)
   - Then a dropdown shows library display preferences:
     - Poster Size / Density — Small / Medium / Large (adjusts grid columns)
     - Default Sort Preference — remember preferred sort order
     - Title Display Language — zh-TW priority / Original title priority
   - And preferences persist across sessions (localStorage)
   - And changes apply immediately to the library view

## Tasks / Subtasks

- [ ] Task 1: Create Library API Endpoints (AC: 1, 4)
  - [ ] 1.1: Register library routes in `/apps/api/main.go` router setup
  - [ ] 1.2: Create `GET /api/v1/library` unified endpoint in existing `handlers/` directory
  - [ ] 1.3: Implement `LibraryHandler` using existing `LibraryServiceInterface`
  - [ ] 1.4: Support query params: `page`, `page_size`, `type` (all|movie|tv)
  - [ ] 1.5: Return `PaginatedResponse` with movies + series combined
  - [ ] 1.6: Write handler tests

- [ ] Task 2: Extend Library Service (AC: 1, 4)
  - [ ] 2.1: Add `ListLibrary(ctx, params) (*LibraryListResult, error)` to `LibraryServiceInterface`
  - [ ] 2.2: Implement combined movie + series listing with pagination
  - [ ] 2.3: Support type filtering (all, movie, tv)
  - [ ] 2.4: Default sort: created_at DESC (newest first)
  - [ ] 2.5: Write service tests (≥80% coverage)

- [ ] Task 3: Create Library Route (AC: 1, 5, 6)
  - [ ] 3.1: Create `/apps/web/src/routes/library.tsx` with TanStack Router
  - [ ] 3.2: Define SearchParams: `page`, `pageSize`, `type`
  - [ ] 3.3: Add route to navigation (horizontal tab bar pattern per UX spec)
  - [ ] 3.4: Implement empty state component per UX spec

- [ ] Task 4: Create Library API Service & Hooks (AC: 1, 4)
  - [ ] 4.1: Create `/apps/web/src/services/libraryService.ts`
  - [ ] 4.2: Implement `listLibrary(params): Promise<LibraryResponse>`
  - [ ] 4.3: Create `/apps/web/src/hooks/useLibrary.ts`
  - [ ] 4.4: Implement `useLibraryList(params)` with TanStack Query
  - [ ] 4.5: Define query keys: `['library', 'list', { page, pageSize, type }]`
  - [ ] 4.6: Set staleTime: 30s (NFR-P9: updates within 30 seconds)

- [ ] Task 5: Create Library Grid Component (AC: 1, 2, 3, 6)
  - [ ] 5.1: Create `/apps/web/src/components/library/LibraryGrid.tsx`
  - [ ] 5.2: Reuse existing `PosterCard` from `/components/media/PosterCard.tsx`
  - [ ] 5.3: Grid layout: `grid-template-columns: repeat(auto-fill, minmax(200px, 1fr))`, gap 16px
  - [ ] 5.4: Implement virtual scrolling with `@tanstack/react-virtual` for >1000 items
  - [ ] 5.5: Add skeleton loading state using existing `PosterCardSkeleton`
  - [ ] 5.6: Write component tests

- [ ] Task 6: Enhance PosterCard for Library Context (AC: 3, 7)
  - [ ] 6.1: Add metadata source badge to `PosterCard` hover state
  - [ ] 6.2: Add library-specific props (date added, metadata source)
  - [ ] 6.3: Ensure click navigates to detail panel/page
  - [ ] 6.4: Add `...` (MoreHorizontal) icon to hover overlay at top-right position
  - [ ] 6.5: On `...` click, open PosterCardMenu (stopPropagation to prevent card click)
  - [ ] 6.6: Write updated PosterCard tests

- [ ] Task 7: Create Library Types (AC: 1)
  - [ ] 7.1: Add library types to `/apps/web/src/types/library.ts`
  - [ ] 7.2: Define `LibraryItem` (unified movie + series type)
  - [ ] 7.3: Define `LibraryListResponse` with pagination

- [ ] Task 8: Create PosterCardMenu Component (AC: 7)
  - [ ] 8.1: Create `/apps/web/src/components/library/PosterCardMenu.tsx`
  - [ ] 8.2: Menu items with Lucide icons: Eye (View Details), RefreshCw (Re-parse), Download (Export), Trash2 (Delete)
  - [ ] 8.3: Delete uses `--error` red color, separated by divider, appears last
  - [ ] 8.4: Delete triggers confirmation dialog (reuse pattern from Story 5.7 BatchConfirmDialog)
  - [ ] 8.5: Re-parse calls `POST /api/v1/library/{type}/{id}/reparse` (single-item endpoint)
  - [ ] 8.6: Export calls `POST /api/v1/library/{type}/{id}/export` (single-item endpoint)
  - [ ] 8.7: Mobile: long-press trigger with bottom sheet menu presentation
  - [ ] 8.8: Menu dismisses on outside click
  - [ ] 8.9: Write component tests

- [ ] Task 9: Create Settings Gear Dropdown Component (AC: 8)
  - [ ] 9.1: Create `/apps/web/src/components/library/SettingsGearDropdown.tsx`
  - [ ] 9.2: Trigger icon: Lucide `Settings` in library toolbar
  - [ ] 9.3: Poster Size / Density selector (Small / Medium / Large) — adjusts grid column min-width
  - [ ] 9.4: Default Sort Preference — dropdown to select and remember sort order
  - [ ] 9.5: Title Display Language toggle — zh-TW priority vs. Original title priority
  - [ ] 9.6: Persist preferences in localStorage; apply immediately on change
  - [ ] 9.7: Write component tests

- [ ] Task 10: Create Single-Item API Endpoints for Context Menu (AC: 7)
  - [ ] 10.1: Add `POST /api/v1/library/movies/:id/reparse` endpoint
  - [ ] 10.2: Add `POST /api/v1/library/series/:id/reparse` endpoint
  - [ ] 10.3: Add `POST /api/v1/library/movies/:id/export` endpoint
  - [ ] 10.4: Add `POST /api/v1/library/series/:id/export` endpoint
  - [ ] 10.5: Add `DELETE /api/v1/library/movies/:id` endpoint
  - [ ] 10.6: Add `DELETE /api/v1/library/series/:id` endpoint
  - [ ] 10.7: Write handler and service tests

## Dev Notes

### Architecture Requirements

**FR38:** Browse complete media library collection
**FR40:** Single-item operations via context menu (delete, re-parse, export metadata)
**NFR-SC6:** Virtual scrolling when library >1,000 items
**NFR-P10:** Grid scrolling maintains 60 FPS
**UX-9:** Appreciation Loop — browsing library is the most frequent daily action
**PRD UI Component Interaction Specs:** Settings Gear Dropdown (#1), PosterCard Context Menu (#2)

### Existing Code to Reuse (DO NOT Reinvent)

**Backend — Already exists:**
- `LibraryService` in `/apps/api/internal/services/library_service.go` — has `SearchLibrary`, needs `ListLibrary`
- `MovieRepository.List()` in `/apps/api/internal/repository/movie_repository.go` — pagination built-in
- `SeriesRepository.List()` in `/apps/api/internal/repository/series_repository.go`
- `ListParams` struct with Page, PageSize, SortBy, SortOrder, Filters validation
- `PaginationResult` struct with TotalResults, TotalPages
- `response.go` helpers: `SuccessResponse()`, `PaginatedResponse` struct

**Frontend — Already exists:**
- `MediaGrid` in `/apps/web/src/components/media/MediaGrid.tsx` — responsive grid
- `PosterCard` in `/apps/web/src/components/media/PosterCard.tsx` — card with hover
- `PosterCardSkeleton` — loading state
- `Pagination` in `/apps/web/src/components/ui/Pagination.tsx` — smart pagination
- `SidePanel` in `/apps/web/src/components/ui/SidePanel.tsx` — detail panel
- `tmdb.ts` `fetchApi<T>()` wrapper pattern — reuse for library service
- Route pattern from `search.tsx` — SearchParams, pagination, query hooks

### Backend Implementation Pattern

```go
// /apps/api/internal/services/library_service.go (extend existing)
type LibraryListResult struct {
    Movies     []models.Movie  `json:"movies"`
    Series     []models.Series `json:"series"`
    Pagination *repository.PaginationResult `json:"pagination"`
}

func (s *LibraryService) ListLibrary(ctx context.Context, params repository.ListParams) (*LibraryListResult, error) {
    // Based on type filter, list from one or both repos
    // Combine results with pagination metadata
}
```

```go
// /apps/api/internal/handlers/library_handler.go (new)
type LibraryHandler struct {
    service services.LibraryServiceInterface
}

// GET /api/v1/library?page=1&page_size=20&type=all
func (h *LibraryHandler) ListLibrary(c *gin.Context) {
    params := parseListParams(c) // reuse existing helper
    // Add type filter from query param
    result, err := h.service.ListLibrary(c.Request.Context(), params)
    // ...
    SuccessResponse(c, result)
}
```

### Frontend Implementation Pattern

```tsx
// /apps/web/src/routes/library.tsx
import { createFileRoute } from '@tanstack/react-router';

type LibrarySearchParams = {
  page?: number;
  pageSize?: number;
  type?: 'all' | 'movie' | 'tv';
};

export const Route = createFileRoute('/library')({
  validateSearch: (search: Record<string, unknown>): LibrarySearchParams => ({
    page: Number(search.page) || 1,
    pageSize: Number(search.pageSize) || 20,
    type: (search.type as LibrarySearchParams['type']) || 'all',
  }),
  component: LibraryPage,
});
```

```tsx
// /apps/web/src/hooks/useLibrary.ts
const libraryKeys = {
  all: ['library'] as const,
  lists: () => [...libraryKeys.all, 'list'] as const,
  list: (params: LibraryParams) => [...libraryKeys.lists(), params] as const,
};

export function useLibraryList(params: LibraryParams) {
  return useQuery({
    queryKey: libraryKeys.list(params),
    queryFn: () => libraryService.listLibrary(params),
    staleTime: 30 * 1000, // NFR-P9: 30s freshness
  });
}
```

### UX Requirements (from UX Design Spec)

- **Grid**: `repeat(auto-fill, minmax(200px, 1fr))`, gap 16px, 2:3 poster ratio
- **Desktop**: 5-6 columns at 1440px+, hover shows scale(1.05) + shadow-xl + overlay
- **Tablet**: 3-4 columns (768-1439px)
- **Mobile**: 2 columns (<768px)
- **Hover overlay**: Rating star, play icon, status badge — staggered 50ms animation
- **Click**: Opens Spotify-style right slide-in panel (500ms ease-out, 400-500px width)
- **Empty state**: 📚🎬 icon + welcome message + setup CTA
- **Tab nav**: Horizontal tabs — Library | Downloading | To Parse | Settings

### Project Structure Notes

```
Backend (extend existing):
/apps/api/internal/services/library_service.go  ← ADD ListLibrary method
/apps/api/internal/handlers/library_handler.go   ← NEW
/apps/api/internal/handlers/library_handler_test.go ← NEW

Frontend (new + reuse):
/apps/web/src/routes/library.tsx                 ← NEW
/apps/web/src/services/libraryService.ts         ← NEW
/apps/web/src/hooks/useLibrary.ts                ← NEW
/apps/web/src/types/library.ts                   ← NEW
/apps/web/src/components/library/LibraryGrid.tsx ← NEW
/apps/web/src/components/library/LibraryGrid.spec.tsx ← NEW
/apps/web/src/components/library/EmptyLibrary.tsx ← NEW
/apps/web/src/components/library/PosterCardMenu.tsx      ← NEW
/apps/web/src/components/library/PosterCardMenu.spec.tsx ← NEW
/apps/web/src/components/library/SettingsGearDropdown.tsx      ← NEW
/apps/web/src/components/library/SettingsGearDropdown.spec.tsx ← NEW
```

### Testing Strategy

- Backend services: ≥80% coverage (mock repositories)
- Backend handlers: ≥70% coverage (mock services)
- Frontend components: ≥70% coverage (render tests, empty/loading/data states)
- Tests co-located: `*_test.go`, `*.spec.tsx`

### Error Codes

- `LIBRARY_FETCH_FAILED` — Failed to load library
- `LIBRARY_EMPTY` — No items in library (informational, not error)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR38]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-SC6]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-P10]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Media-Library-Page]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [Source: project-context.md#Rule-5-TanStack-Query]
- [Source: _bmad-output/planning-artifacts/prd.md#UI-Component-Interaction-Specifications]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
