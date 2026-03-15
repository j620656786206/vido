# Story 5.2: Library List View Toggle

Status: done

## Story

As a **media collector**,
I want to **switch between grid and list views**,
So that **I can choose the display format that suits my preference**.

## Acceptance Criteria

1. **AC1: View Toggle Control**
   - Given the library is displayed in grid view
   - When the user clicks the "List View" toggle button
   - Then the display switches to a table/list format
   - And the toggle visually indicates the current mode

2. **AC2: List View Columns**
   - Given list view is active
   - When viewing the table
   - Then columns include: poster thumbnail (small), title, year, genre, rating, date added
   - And rows are compact for efficient scanning

3. **AC3: Column Sorting**
   - Given list view is active
   - When the user clicks a column header
   - Then the list sorts by that column
   - And ascending/descending toggle is available
   - And sort indicator (arrow) shows current direction

4. **AC4: View Preference Persistence**
   - Given the user's view preference (grid or list)
   - When they return to the library later
   - Then their preferred view is remembered
   - And persisted in localStorage

## Tasks / Subtasks

- [x] Task 1: Create View Toggle Component (AC: 1)
  - [x] 1.1: Create `/apps/web/src/components/library/ViewToggle.tsx`
  - [x] 1.2: Two icon buttons: grid icon + list icon
  - [x] 1.3: Active state styling (filled vs outline)
  - [x] 1.4: Accessible: `role="radiogroup"`, `aria-label="切換檢視模式"`
  - [x] 1.5: Write component tests

- [x] Task 2: Create Library Table Component (AC: 2, 3)
  - [x] 2.1: Create `/apps/web/src/components/library/LibraryTable.tsx`
  - [x] 2.2: Columns: thumbnail (48x72px), title (zh-TW primary, original secondary), year, genre tags, rating star, date added
  - [x] 2.3: Sortable column headers with click handler
  - [x] 2.4: Sort indicator arrows (▲/▼)
  - [x] 2.5: Hover row highlight
  - [x] 2.6: Click row → navigate to detail (same as grid card click)
  - [x] 2.7: Skeleton loading rows for loading state
  - [x] 2.8: Write component tests

- [x] Task 3: Integrate View Toggle into Library Page (AC: 1, 4)
  - [x] 3.1: Add `view` to library route SearchParams (`grid` | `list`)
  - [x] 3.2: Add ViewToggle to library toolbar area
  - [x] 3.3: Conditionally render LibraryGrid or LibraryTable based on view
  - [x] 3.4: Persist preference to localStorage key `vido:library:view`
  - [x] 3.5: Initialize from localStorage on page load

- [x] Task 4: Extend Backend Sort Support (AC: 3)
  - [x] 4.1: Ensure `ListParams.SortBy` supports: `created_at`, `title`, `release_date`/`first_air_date`, `vote_average`
  - [x] 4.2: Update `ListLibrary` service to pass sort params to repositories
  - [x] 4.3: Add `sort_by` and `sort_order` query params to library API
  - [x] 4.4: Write tests for sort options

## Dev Notes

### Architecture Requirements

**FR8:** Toggle between grid and list view
**UX:** View preference stored in localStorage

### Existing Code to Reuse (DO NOT Reinvent)

- `LibraryGrid` from Story 5-1 — grid view already done
- `Pagination` component — works for both views
- `useLibraryList` hook from Story 5-1 — same data source
- `ListParams` backend struct — already supports SortBy, SortOrder
- `PosterCard` click behavior — reuse navigation logic
- lucide-react icons: `LayoutGrid`, `List` for toggle buttons

### Frontend Implementation Pattern

```tsx
// /apps/web/src/components/library/ViewToggle.tsx
interface ViewToggleProps {
  view: 'grid' | 'list';
  onViewChange: (view: 'grid' | 'list') => void;
}

export function ViewToggle({ view, onViewChange }: ViewToggleProps) {
  return (
    <div role="radiogroup" aria-label="切換檢視模式" className="flex gap-1">
      <button
        role="radio"
        aria-checked={view === 'grid'}
        onClick={() => onViewChange('grid')}
        className={cn('p-2 rounded', view === 'grid' ? 'bg-primary text-white' : 'text-muted-foreground')}
      >
        <LayoutGrid size={18} />
      </button>
      <button
        role="radio"
        aria-checked={view === 'list'}
        onClick={() => onViewChange('list')}
        className={cn('p-2 rounded', view === 'list' ? 'bg-primary text-white' : 'text-muted-foreground')}
      >
        <List size={18} />
      </button>
    </div>
  );
}
```

```tsx
// /apps/web/src/components/library/LibraryTable.tsx
// Sortable table with columns: thumbnail, title, year, genre, rating, date added
// Use <table> with proper semantic HTML
// Click row → navigate to /media/{type}/{id}
```

### localStorage Pattern

```typescript
const STORAGE_KEY = 'vido:library:view';

function getStoredView(): 'grid' | 'list' {
  return (localStorage.getItem(STORAGE_KEY) as 'grid' | 'list') || 'grid';
}

function setStoredView(view: 'grid' | 'list') {
  localStorage.setItem(STORAGE_KEY, view);
}
```

### Project Structure Notes

```
Frontend (new):
/apps/web/src/components/library/ViewToggle.tsx       ← NEW
/apps/web/src/components/library/ViewToggle.spec.tsx   ← NEW
/apps/web/src/components/library/LibraryTable.tsx      ← NEW
/apps/web/src/components/library/LibraryTable.spec.tsx ← NEW

Frontend (modify):
/apps/web/src/routes/library.tsx                       ← ADD view param, toggle, conditional render
```

### Dependencies

- Story 5-1 (Media Library Grid View) — grid component and library API must exist first

### Testing Strategy

- ViewToggle: render both states, click toggles, ARIA attributes
- LibraryTable: render with data, sort click changes order, loading state, click row navigates
- Integration: toggle switches between grid and table in library page

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR8]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Grid-View-List-View]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Created ViewToggle component with LayoutGrid/List icons, radiogroup ARIA, active/inactive styling. 9 tests pass.
- Task 2: Created LibraryTable component with 6 columns (poster, title, year, genre, rating, date added), sortable headers with ArrowUp/ArrowDown indicators, hover highlight, row click navigation via Link, skeleton loading. 20 tests pass.
- Task 3: Integrated ViewToggle into library toolbar, added `view` search param, conditional rendering of LibraryGrid vs LibraryTable, localStorage persistence with `vido:library:view` key, column sort handler with asc/desc toggle via URL params.
- Task 4: Added `vote_average` to valid sort columns in movie and series repositories. Updated `listAll` service method to sort by any requested field (title, release_date, rating/vote_average, created_at) instead of hardcoded created_at. Added `release_date` alias to series repository. 3 new backend tests for sort (title ASC, title DESC, vote_average).

### File List

- apps/web/src/components/library/ViewToggle.tsx (NEW)
- apps/web/src/components/library/ViewToggle.spec.tsx (NEW)
- apps/web/src/components/library/LibraryTable.tsx (NEW)
- apps/web/src/components/library/LibraryTable.spec.tsx (NEW)
- apps/web/src/routes/library.tsx (MODIFIED)
- apps/api/internal/repository/movie_repository.go (MODIFIED)
- apps/api/internal/repository/series_repository.go (MODIFIED)
- apps/api/internal/services/library_service.go (MODIFIED)
- apps/api/internal/services/library_service_test.go (MODIFIED)
- _bmad-output/implementation-artifacts/sprint-status.yaml (MODIFIED)
- _bmad-output/implementation-artifacts/5-2-library-list-view-toggle.md (MODIFIED)

### Senior Developer Review (AI)

**Reviewer:** Amelia (Dev Agent) — 2026-03-15
**Outcome:** Approved with fixes applied

**Issues Found & Fixed (5):**
1. **[HIGH] release_date sort crashes on series table** — series table has `first_air_date`, not `release_date`. Fixed by converting validSortColumns map to sortColumnMap with aliases in series_repository.go.
2. **[HIGH] rating vs vote_average sort inconsistency** — Frontend sends `rating` but TMDb data populates `vote_average`. Fixed by aliasing `rating` → `vote_average` in both movie_repository.go and series_repository.go sortColumnMap.
3. **[HIGH] Navigation handlers drop sort & view params** — handlePageChange, handleTypeChange, handleColumnSort all lost sortBy/sortOrder/view on navigate. Fixed all handlers to preserve current search params.
4. **[MEDIUM] View toggle doesn't sync to URL** — handleViewChange only set state+localStorage. Fixed to also navigate with view param.
5. **[MEDIUM] Thumbnail size doesn't match spec** — Was 36x54px, spec says 48x72px. Fixed to w-12 h-[72px].

### Change Log

- 2026-03-15: Implemented Story 5-2 Library List View Toggle — all 4 tasks complete, 29 new frontend tests + 3 new backend tests, all 854 frontend tests and all backend tests pass.
- 2026-03-15: Code Review — Fixed 3 HIGH + 2 MEDIUM issues (sort column aliasing, navigation param preservation, thumbnail size). All 64 related tests pass.
