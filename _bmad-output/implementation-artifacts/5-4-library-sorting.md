# Story 5.4: Library Sorting

Status: ready-for-dev

## Story

As a **media collector**,
I want to **sort my library by different criteria**,
So that **I can organize my view based on what I'm looking for**.

## Acceptance Criteria

1. **AC1: Sort Dropdown Options**
   - Given the library is displayed
   - When the user opens the sort dropdown
   - Then options include:
     - 新增日期（最新/最舊）— Date Added (newest/oldest)
     - 標題（A-Z / Z-A）— Title (A-Z / Z-A)
     - 年份（最新/最舊）— Year (newest/oldest)
     - 評分（最高/最低）— Rating (highest/lowest)

2. **AC2: Sort Application**
   - Given a sort option is selected
   - When the sort is applied
   - Then the library reorders immediately
   - And the current sort is indicated in the UI (label + direction arrow)

3. **AC3: Sort Preference Persistence**
   - Given the user's sort preference
   - When they return to the library
   - Then their last used sort is applied
   - And sort state is persisted in both URL params and localStorage

## Tasks / Subtasks

- [ ] Task 1: Create Sort Selector Component (AC: 1, 2)
  - [ ] 1.1: Create `/apps/web/src/components/library/SortSelector.tsx`
  - [ ] 1.2: Dropdown with sort options (zh-TW labels)
  - [ ] 1.3: Each option includes direction toggle (asc/desc)
  - [ ] 1.4: Show current sort label + arrow indicator
  - [ ] 1.5: Accessible: proper aria-label, keyboard navigation
  - [ ] 1.6: Write component tests

- [ ] Task 2: Add Sort Params to Library Route (AC: 2, 3)
  - [ ] 2.1: Add `sortBy` and `sortOrder` to library route SearchParams
  - [ ] 2.2: Map frontend sort keys to backend: `created_at`, `title`, `release_date`, `vote_average`
  - [ ] 2.3: Pass sort params to `useLibraryList` / `useLibrarySearch` hooks
  - [ ] 2.4: Reset to page 1 when sort changes

- [ ] Task 3: Persist Sort Preference (AC: 3)
  - [ ] 3.1: Save to localStorage key `vido:library:sort` as JSON `{ sortBy, sortOrder }`
  - [ ] 3.2: Initialize from localStorage when no URL sort params present
  - [ ] 3.3: URL params take priority over localStorage

- [ ] Task 4: Ensure Backend Sort Support (AC: 2)
  - [ ] 4.1: Verify `ListParams` validates allowed sort fields
  - [ ] 4.2: Add database indexes if missing for sort columns
  - [ ] 4.3: Handle combined movie+series sort (interleave by sort field)
  - [ ] 4.4: Write backend sort tests

## Dev Notes

### Architecture Requirements

**FR6:** Sort media library by date added, title, year, rating

### Existing Code to Reuse (DO NOT Reinvent)

- `ListParams.SortBy` / `ListParams.SortOrder` — already validated in repository.go
- `parseListParams(c)` handler helper — already parses `sort_by`, `sort_order` query params
- Backend repos already sort by any column passed in SortBy
- Library route SearchParams from Story 5-1 — extend with sort params

### Sort Field Mapping

| Frontend Label | `sortBy` Value | Backend Column | Notes |
|---|---|---|---|
| 新增日期 | `created_at` | `created_at` | Default sort |
| 標題 | `title` | `title` | Alphabetical (Chinese stroke order in SQLite) |
| 年份 | `year` | `release_date` / `first_air_date` | Movies use release_date, series use first_air_date |
| 評分 | `rating` | `vote_average` | TMDb rating |

### localStorage Pattern

```typescript
const SORT_STORAGE_KEY = 'vido:library:sort';

interface SortPreference {
  sortBy: string;
  sortOrder: 'asc' | 'desc';
}

const DEFAULT_SORT: SortPreference = { sortBy: 'created_at', sortOrder: 'desc' };
```

### Project Structure Notes

```
Frontend (new):
/apps/web/src/components/library/SortSelector.tsx       ← NEW
/apps/web/src/components/library/SortSelector.spec.tsx  ← NEW

Frontend (modify):
/apps/web/src/routes/library.tsx         ← ADD sortBy, sortOrder params
/apps/web/src/hooks/useLibrary.ts        ← PASS sort params to query
/apps/web/src/services/libraryService.ts ← PASS sort params to API
```

### Dependencies

- Story 5-1 (Media Library Grid View) — library page and API must exist

### Testing Strategy

- SortSelector: renders options, click changes sort, shows current sort indicator
- Sort integration: changing sort updates URL params and re-fetches with new order
- Persistence: sort saved to localStorage, restored on page load

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.4]
- [Source: _bmad-output/planning-artifacts/prd.md#FR6]
- [Source: project-context.md#Rule-5-TanStack-Query]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
