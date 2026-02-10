# Story 5.2: Library List View Toggle

Status: ready-for-dev

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

- [ ] Task 1: Create View Toggle Component (AC: 1)
  - [ ] 1.1: Create `/apps/web/src/components/library/ViewToggle.tsx`
  - [ ] 1.2: Two icon buttons: grid icon + list icon
  - [ ] 1.3: Active state styling (filled vs outline)
  - [ ] 1.4: Accessible: `role="radiogroup"`, `aria-label="切換檢視模式"`
  - [ ] 1.5: Write component tests

- [ ] Task 2: Create Library Table Component (AC: 2, 3)
  - [ ] 2.1: Create `/apps/web/src/components/library/LibraryTable.tsx`
  - [ ] 2.2: Columns: thumbnail (48x72px), title (zh-TW primary, original secondary), year, genre tags, rating star, date added
  - [ ] 2.3: Sortable column headers with click handler
  - [ ] 2.4: Sort indicator arrows (▲/▼)
  - [ ] 2.5: Hover row highlight
  - [ ] 2.6: Click row → navigate to detail (same as grid card click)
  - [ ] 2.7: Skeleton loading rows for loading state
  - [ ] 2.8: Write component tests

- [ ] Task 3: Integrate View Toggle into Library Page (AC: 1, 4)
  - [ ] 3.1: Add `view` to library route SearchParams (`grid` | `list`)
  - [ ] 3.2: Add ViewToggle to library toolbar area
  - [ ] 3.3: Conditionally render LibraryGrid or LibraryTable based on view
  - [ ] 3.4: Persist preference to localStorage key `vido:library:view`
  - [ ] 3.5: Initialize from localStorage on page load

- [ ] Task 4: Extend Backend Sort Support (AC: 3)
  - [ ] 4.1: Ensure `ListParams.SortBy` supports: `created_at`, `title`, `release_date`/`first_air_date`, `vote_average`
  - [ ] 4.2: Update `ListLibrary` service to pass sort params to repositories
  - [ ] 4.3: Add `sort_by` and `sort_order` query params to library API
  - [ ] 4.4: Write tests for sort options

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
