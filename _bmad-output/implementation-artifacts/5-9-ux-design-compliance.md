# Story 5.9: UX Design Compliance — Epic 5 Design Review Fixes

Status: ready-for-dev

<!-- Note: This story addresses all UX design deviations found during the Epic 5 design review (2026-03-15). -->
<!-- Source: _bmad-output/implementation-artifacts/ux-design-review-epic5.md -->

## Story

As a **media collector**,
I want the **library UI to precisely match the approved UX design specifications**,
So that **the interface is consistent, polished, and provides the interaction patterns that were designed for optimal usability**.

## Acceptance Criteria

1. **AC1: Filter Panel Sidebar Layout**
   - Given the user clicks the filter toggle
   - When the filter panel opens
   - Then it appears as a **left sidebar** (~200px wide) alongside the grid/list content
   - And the main content area shrinks to accommodate the sidebar
   - And the sidebar is persistent (not a dropdown overlay)
   - And the sidebar has a "篩選條件" heading at the top
   - And the sidebar has a right border only (no rounded corners, no all-around border)
   - Reference: `flow-c-search-filter-settings-desktop/07-search-filter-desktop.png`

2. **AC2: Filter Controls — Chip-Style Toggles**
   - Given the filter sidebar is open
   - When viewing genre options
   - Then genres are displayed as **rounded-full pill chip toggles** (not HTML checkboxes)
   - And active chips show a **checkmark icon** + blue background/border
   - And year filter uses **decade chip toggles**: `2020s`, `2010s`, `2000s`, `1990s`, `更早`
   - And media type toggle (全部/電影/影集) is **inside the sidebar** as a section (not separate buttons in content area)

3. **AC3: Filter Panel Labels & Buttons**
   - Given the filter sidebar is open
   - When viewing section labels and action buttons
   - Then year section label is "年份" (not "年份範圍")
   - And apply button text is "套用" (not "套用篩選")
   - And reset button text is "重置" (not "清除")
   - And both buttons are full-width within the sidebar

4. **AC4: Controls Row Layout**
   - Given the library page is displayed
   - When viewing the controls bar
   - Then the left side shows "全部媒體" as a section heading
   - And the right side order is: Sort → Filter → View Toggle (not Filter → Sort → View Toggle)
   - And the standalone type filter buttons (全部/電影/影集) are removed from the content area (moved into sidebar per AC2)

5. **AC5: List View Polish**
   - Given the user is in list view
   - When viewing the table
   - Then the table header row has a subtle background (`bg-slate-800/50` or similar)
   - And the date column label is unified — same term used in both table header and sort dropdown

6. **AC6: Search Bar & Filter Interaction**
   - Given the user is searching
   - When the search is active
   - Then the library search bar uses `rounded-full` shape (matching header search style)
   - And the filter sidebar remains accessible (not hidden) during active search

7. **AC7: Filter Chip Styling**
   - Given filters are active
   - When viewing filter chips above the grid
   - Then all chips use a **unified color scheme** (not blue for genre + green for year)

## Tasks / Subtasks

- [ ] Task 1: Refactor FilterPanel as Left Sidebar (AC: 1, 2, 3)
  - [ ] 1.1: Change FilterPanel layout from dropdown to sidebar — use `position: sticky`, ~200px width, right border only, full height
  - [ ] 1.2: Update `library.tsx` layout to flex: `[sidebar | content]` when filter is open
  - [ ] 1.3: Add "篩選條件" heading at top of sidebar panel
  - [ ] 1.4: Replace genre HTML checkboxes with rounded-full chip toggle buttons — active state shows `Check` icon + `bg-blue-500/15 border-blue-500` styling
  - [ ] 1.5: Replace year min/max number inputs with decade chip toggles: `2020s`, `2010s`, `2000s`, `1990s`, `更早` — multi-select, same chip style as genre
  - [ ] 1.6: Move media type toggle (全部/電影/影集) INTO the FilterPanel as the first section — remove standalone type buttons from `library.tsx`
  - [ ] 1.7: Update year section label "年份範圍" → "年份"
  - [ ] 1.8: Update button labels: "套用篩選" → "套用", "清除" → "重置" — both full-width in sidebar
  - [ ] 1.9: Update panel background/border: remove `rounded-lg`, use right border only, `bg-slate-900` or page-matching bg
  - [ ] 1.10: Ensure sidebar collapses properly via toggle button — mobile: convert to bottom sheet or full-screen overlay (stretch goal)

- [ ] Task 2: Fix Controls Row Layout (AC: 4, 5)
  - [ ] 2.1: Add "全部媒體" section heading (`text-xl font-semibold text-white`) to left side of controls row in `library.tsx`
  - [ ] 2.2: Swap controls order in `library.tsx`: render `SortSelector` before `FilterPanel` (Sort → Filter → ViewToggle)
  - [ ] 2.3: Add `bg-slate-800/50` to table header `<tr>` in `LibraryTable.tsx`
  - [ ] 2.4: Unify date column label — use same term in `LibraryTable.tsx` column header and `SortSelector.tsx` option label (choose either "加入日期" or "新增日期" consistently)

- [ ] Task 3: Fix Search & Filter Interaction (AC: 6)
  - [ ] 3.1: Change `LibrarySearchBar.tsx` border-radius from `rounded-lg` to `rounded-full`
  - [ ] 3.2: In `library.tsx`, remove the `{!isSearchActive && (...)}` guard that hides FilterPanel during search — filter sidebar should remain accessible during active search

- [ ] Task 4: Unify Filter Chip Styling (AC: 7)
  - [ ] 4.1: In `FilterChips.tsx`, change year chips from `bg-green-600/20 text-green-300` to match genre chip color (`bg-blue-600/20 text-blue-300` or a unified neutral color like `bg-slate-600/30 text-slate-300`)

- [ ] Task 5: Update Tests & Verify (AC: 1-7)
  - [ ] 5.1: Update existing FilterPanel tests for new sidebar layout and chip-style controls
  - [ ] 5.2: Update existing library route tests for new controls row order and removed type buttons
  - [ ] 5.3: Update LibraryTable tests for header background
  - [ ] 5.4: Run full `nx run web:test` suite — all tests must pass
  - [ ] 5.5: Run `pnpm run test:cleanup` to verify no orphaned processes

## Dev Notes

### Architecture Requirements

- Follow Tailwind CSS v3.x utility-first classes (project-context.md Rule 1)
- Frontend components in `apps/web/src/components/library/` (co-located with tests)
- Use TanStack Query for any server state (project-context.md Rule 5)
- Test assertions use specific matchers: `toBeInTheDocument`, `toBeAttached` (Rule 16)

### Existing Code to Reuse

**DO NOT reinvent — modify these existing files:**
- `apps/web/src/components/library/FilterPanel.tsx` — the main file to refactor
- `apps/web/src/components/library/FilterChips.tsx` — chip color unification
- `apps/web/src/components/library/LibrarySearchBar.tsx` — border-radius fix
- `apps/web/src/components/library/LibraryTable.tsx` — header bg + label fix
- `apps/web/src/components/library/SortSelector.tsx` — label unification
- `apps/web/src/routes/library.tsx` — layout restructure, controls order, remove type buttons

### Filter Panel Sidebar Layout Pattern

```tsx
// library.tsx layout when filter is open:
<div className="flex gap-0">
  {isFilterOpen && (
    <aside className="sticky top-16 h-[calc(100vh-4rem)] w-[200px] flex-shrink-0 overflow-y-auto border-r border-slate-700 p-4">
      <FilterPanel ... />
    </aside>
  )}
  <div className="flex-1 min-w-0">
    {/* existing grid/list content */}
  </div>
</div>
```

### Chip Toggle Button Pattern

```tsx
// Genre/Year chip toggle (replaces checkbox):
<button
  onClick={() => toggleGenre(genre)}
  className={cn(
    "inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm transition-colors",
    isSelected
      ? "bg-blue-500/15 border border-blue-500 text-blue-300"
      : "bg-slate-700 text-slate-300 hover:bg-slate-600"
  )}
>
  {isSelected && <Check className="h-3.5 w-3.5" />}
  {genre}
</button>
```

### Decade Chip Mapping

```tsx
const DECADE_OPTIONS = [
  { label: '2020s', min: 2020, max: 2029 },
  { label: '2010s', min: 2010, max: 2019 },
  { label: '2000s', min: 2000, max: 2009 },
  { label: '1990s', min: 1990, max: 1999 },
  { label: '更早', min: 0, max: 1989 },
];
// Multi-select: combine selected decades into yearMin/yearMax range
// e.g., selecting 2020s + 2000s → yearMin=2000, yearMax=2029
```

### Design Screenshots Reference

| Screenshot | Path | What to Match |
|------------|------|---------------|
| Filter Sidebar | `flow-c-search-filter-settings-desktop/07-search-filter-desktop.png` | Sidebar layout, sections, chips |
| List View | `flow-a-browse-desktop/06-list-view-desktop.png` | Controls row, table header |
| Grid View | `flow-a-browse-desktop/01-library-grid-desktop.png` | Controls row, section heading |

### UX Design Review Source

Full comparison details: `_bmad-output/implementation-artifacts/ux-design-review-epic5.md`

### Project Structure Notes

Files to modify (all existing):
```
apps/web/src/
├── routes/
│   └── library.tsx                    # Layout restructure, controls order, remove type buttons
├── components/
│   └── library/
│       ├── FilterPanel.tsx            # Major refactor: sidebar + chips + sections
│       ├── FilterPanel.spec.tsx       # Update tests for new layout
│       ├── FilterChips.tsx            # Unify chip colors
│       ├── LibrarySearchBar.tsx       # rounded-full fix
│       ├── LibraryTable.tsx           # Header bg, label fix
│       └── SortSelector.tsx           # Label unification
```

No new files should be needed — this is a refactor of existing components.

### Dependencies

- Stories 5-1 through 5-5 must be done (all ✅)
- No backend changes required (decade chips translate to existing yearMin/yearMax API params)

### Testing Strategy

- **Component tests:** Update existing FilterPanel tests for chip toggles and sidebar rendering
- **Route tests:** Update library route tests for controls order and layout changes
- **Table tests:** Update LibraryTable tests for header background class
- **No new E2E tests** — this is a visual compliance fix, existing integration tests cover functionality

### References

- [UX Design Review Report: _bmad-output/implementation-artifacts/ux-design-review-epic5.md]
- [Design Screenshots: _bmad-output/screenshots/]
- [project-context.md: Rules 1, 5, 9, 16]
- [Filter Panel Design: flow-c-search-filter-settings-desktop/07-search-filter-desktop.png]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Change Log

### File List
