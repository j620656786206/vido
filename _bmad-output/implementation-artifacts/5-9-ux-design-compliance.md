# Story 5.9: UX Design Compliance — Epic 5 Design Review Fixes

Status: review

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

- [x] Task 1: Refactor FilterPanel as Left Sidebar (AC: 1, 2, 3)
  - [x] 1.1: Change FilterPanel layout from dropdown to sidebar — use `position: sticky`, ~200px width, right border only, full height
  - [x] 1.2: Update `library.tsx` layout to flex: `[sidebar | content]` when filter is open
  - [x] 1.3: Add "篩選條件" heading at top of sidebar panel
  - [x] 1.4: Replace genre HTML checkboxes with rounded-full chip toggle buttons — active state shows `Check` icon + `bg-blue-500/15 border-blue-500` styling
  - [x] 1.5: Replace year min/max number inputs with decade chip toggles: `2020s`, `2010s`, `2000s`, `1990s`, `更早` — multi-select, same chip style as genre
  - [x] 1.6: Move media type toggle (全部/電影/影集) INTO the FilterPanel as the first section — remove standalone type buttons from `library.tsx`
  - [x] 1.7: Update year section label "年份範圍" → "年份"
  - [x] 1.8: Update button labels: "套用篩選" → "套用", "清除" → "重置" — both full-width in sidebar
  - [x] 1.9: Update panel background/border: remove `rounded-lg`, use right border only, `bg-slate-900` or page-matching bg
  - [x] 1.10: Ensure sidebar collapses properly via toggle button — mobile: convert to bottom sheet or full-screen overlay (stretch goal)

- [x] Task 2: Fix Controls Row Layout (AC: 4, 5)
  - [x] 2.1: Add "全部媒體" section heading (`text-xl font-semibold text-white`) to left side of controls row in `library.tsx`
  - [x] 2.2: Swap controls order in `library.tsx`: render `SortSelector` before `FilterPanel` (Sort → Filter → ViewToggle)
  - [x] 2.3: Add `bg-slate-800/50` to table header `<tr>` in `LibraryTable.tsx`
  - [x] 2.4: Unify date column label — use "新增日期" in both `LibraryTable.tsx` and `SortSelector.tsx`

- [x] Task 3: Fix Search & Filter Interaction (AC: 6)
  - [x] 3.1: Change `LibrarySearchBar.tsx` border-radius from `rounded-lg` to `rounded-full`
  - [x] 3.2: In `library.tsx`, remove the `{!isSearchActive && (...)}` guard that hides FilterPanel during search — filter toggle button always visible, only SortSelector hidden during search

- [x] Task 4: Unify Filter Chip Styling (AC: 7)
  - [x] 4.1: In `FilterChips.tsx`, changed all chips to unified `bg-blue-600/20 text-blue-300` — replaced green year chips, also replaced inline SVG icons with Lucide `X` component

- [x] Task 5: Update Tests & Verify (AC: 1-7)
  - [x] 5.1: Rewrote FilterPanel.spec.tsx — 10 tests for sidebar layout, chip toggles, decade chips, labels, type section
  - [x] 5.2: Updated library.spec.tsx — type filter tabs test now opens sidebar, item count test → section heading test
  - [x] 5.3: LibraryTable header background verified (bg-slate-800/50 added)
  - [x] 5.4: Full `nx run web:test` — 82 files, 938 tests, all passed
  - [x] 5.5: `pnpm run test:cleanup` — no orphaned processes

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

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- ✅ Task 1: Completely refactored FilterPanel from dropdown to sidebar — chip toggles for genres, decade chips for year, type filter moved inside, "篩選條件" heading, "套用"/"重置" buttons
- ✅ Task 2: Controls row — added "全部媒體" heading, swapped Sort→Filter order, added table header bg, unified "新增日期" label
- ✅ Task 3: Search bar shape `rounded-lg` → `rounded-full`, filter toggle visible during search
- ✅ Task 4: Unified filter chip colors — all chips now use `bg-blue-600/20 text-blue-300`
- ✅ Task 5: Rewrote FilterPanel tests (10 tests), updated library route tests (2 tests), all 938 tests pass, no orphaned processes

### Change Log

- 2026-03-15: Story 5-9 UX design compliance — all 5 tasks completed, 20 subtasks checked

### File List

**Frontend (modified):**
- `apps/web/src/components/library/FilterPanel.tsx` — Major refactor: dropdown → sidebar content, chip toggles, decade chips, type section
- `apps/web/src/components/library/FilterPanel.spec.tsx` — Rewrote all tests for new component structure
- `apps/web/src/components/library/FilterChips.tsx` — Unified chip colors, replaced SVG with Lucide X icon
- `apps/web/src/components/library/LibrarySearchBar.tsx` — rounded-lg → rounded-full
- `apps/web/src/components/library/LibraryTable.tsx` — Header row bg-slate-800/50, label "加入日期" → "新增日期"
- `apps/web/src/routes/library.tsx` — Sidebar layout, controls row order, "全部媒體" heading, filter toggle, removed standalone type buttons
- `apps/web/src/routes/library.spec.tsx` — Updated type filter + section heading tests
