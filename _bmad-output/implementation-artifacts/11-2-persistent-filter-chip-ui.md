# Story 11.2: Persistent Filter Chip UI

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user browsing content,
I want active filters displayed as persistent pill/chip elements at the top of the page,
so that I can see what's currently filtered and easily add, remove, or clear filters.

## Acceptance Criteria

1. Given active filters, when displayed, then each filter is a removable chip/pill at the top of the content area (not hidden in dropdowns)
2. Given filter chips, when a user clicks the X on a chip, then that filter is removed and results update immediately
3. Given multiple active filters, when displayed, then a "清除全部" button appears to remove all filters at once
4. Given filter state, when the user navigates away and returns (back button), then the filter state is preserved via URL query params
5. Given a filter panel, when the user selects genre/year/region/rating/platform options, then chips are added immediately and results update
6. Given mobile viewport, when filters are active, then chips wrap to multiple lines and the filter panel is accessible via a bottom sheet

## Tasks / Subtasks

- [ ] Task 1: Filter state management via URL (AC: #4)
  - [ ] 1.1 Create `apps/web/src/hooks/useFilterState.ts` — sync filter state with URL search params
  - [ ] 1.2 Params: `genre`, `year_gte`, `year_lte`, `region`, `rating_gte`, `platform`, `sort_by`
  - [ ] 1.3 Use TanStack Router's `useSearch` for type-safe URL state
  - [ ] 1.4 Back/forward navigation preserves filter state

- [ ] Task 2: FilterChipBar component (AC: #1, #2, #3)
  - [ ] 2.1 Create `apps/web/src/components/search/FilterChipBar.tsx`
  - [ ] 2.2 Render active filters as removable pill chips (reuse existing `Component/FilterChip` design pattern)
  - [ ] 2.3 Each chip shows filter label + value (e.g., "類型: 動畫")
  - [ ] 2.4 X button per chip to remove individual filter
  - [ ] 2.5 "清除全部" button when ≥2 filters active

- [ ] Task 3: Filter panel (AC: #5)
  - [ ] 3.1 Create `apps/web/src/components/search/FilterPanel.tsx`
  - [ ] 3.2 Genre multi-select (from TMDB genre list, zh-TW labels)
  - [ ] 3.3 Year range: min/max year inputs
  - [ ] 3.4 Region: TW, JP, KR, US, CN quick-select chips
  - [ ] 3.5 Rating: slider or min rating input
  - [ ] 3.6 Platform: Netflix, Disney+, KKTV chip toggles
  - [ ] 3.7 Sort dropdown: popularity, date, rating

- [ ] Task 4: Mobile filter bottom sheet (AC: #6)
  - [ ] 4.1 On mobile: filter panel renders as a bottom sheet (triggered by filter icon button)
  - [ ] 4.2 Chips still visible above content (condensed, horizontal scroll)
  - [ ] 4.3 Follow existing bottom sheet pattern from `apps/web/src/components/`

- [ ] Task 5: Integration with discover results (AC: #1, #5)
  - [ ] 5.1 Create `apps/web/src/hooks/useDiscoverResults.ts` — TanStack Query consuming filter state
  - [ ] 5.2 Call `GET /api/v1/tmdb/discover/movies` or `/tv` with current filter params
  - [ ] 5.3 Create browse/discover page or integrate into existing search route
  - [ ] 5.4 Results grid reuses `PosterCard` component

- [ ] Task 6: Tests (AC: #1-6)
  - [ ] 6.1 FilterChipBar: render, remove, clear all
  - [ ] 6.2 useFilterState: URL sync, back button preservation
  - [ ] 6.3 FilterPanel: selection interactions
  - [ ] 6.4 Mobile: bottom sheet open/close

## Dev Notes

### Architecture Compliance

- **URL state:** Filter state in URL search params — no Zustand/Redux needed. TanStack Router handles it
- **Chip design:** Follow existing `Component/FilterChip` from ux-design.pen (cornerRadius:100, bg-tertiary, text-secondary + X icon)
- **Genre list:** Fetch from `GET /genre/movie/list` and `/genre/tv/list` TMDB endpoints (may need to add these to client)
- **Bottom sheet:** Follow existing pattern from Flow D mobile filter (Screen 10)

### References

- [Source: apps/web/src/components/] — Existing component patterns
- [Source: ux-design.pen Component/FilterChip (jD7gF)] — Filter chip design
- [Source: ux-design.pen Screen 7 (rsAxf)] — Desktop search + filter layout
- [Source: ux-design.pen Screen 10 (oypj1)] — Mobile filter bottom sheet
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-011

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
