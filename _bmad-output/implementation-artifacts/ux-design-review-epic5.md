# UX Design Review Report — Epic 5 (Media Library)

**Reviewer:** Sally (UX Designer Agent)
**Date:** 2026-03-15
**Scope:** Stories 5-2 through 5-5 (per-story detailed comparison)
**Reference:** `_bmad-output/screenshots/` design specifications

---

## Executive Summary

| Story | ✅ Match | ⚠️ Minor Gap | ❌ Mismatch | Key Issues |
|-------|---------|-------------|------------|------------|
| 5-2 List View | 13 | 7 | 0 | Controls order, missing section heading, table header bg |
| 5-3 Search | 12 | 6 | 1 | Dual search bars, search bar shape mismatch |
| 5-4 Sorting | 11 | 5 | 1 | No mobile bottom sheet, label inconsistency |
| 5-5 Filtering | 6 | 5 | 6 | Sidebar vs dropdown, chip controls, decade chips, type placement |
| **Total** | **42** | **23** | **8** | |

---

# Story 5-2: Library List View Toggle

**Design Reference:** `flow-a-browse-desktop/06-list-view-desktop.png`

## Findings

### 5-2-01 ⚠️ Controls Row — Left Side Content
- **Design:** Section heading "全部媒體" on the left of controls row
- **Implementation:** Shows pagination count `顯示 1-20 / 100 項` instead
- **Fix:** Add "全部媒體" as section heading; move count to secondary position

### 5-2-02 ⚠️ Controls Row — Element Order
- **Design:** Right side order: Sort → Filter → View Toggle
- **Implementation:** Order is Filter → Sort → View Toggle
- **Fix:** Swap `FilterPanel` and `SortSelector` in `library.tsx` controls row

### 5-2-03 ✅ View Toggle Button Styling
- Active: `bg-blue-600 text-white`, Inactive: `text-slate-400` — matches design

### 5-2-04 ⚠️ Table Header Row Background
- **Design:** Header row has slightly lighter background than body for visual separation
- **Implementation:** No explicit background on header `<tr>` — inherits page bg
- **Fix:** Add `bg-slate-800/50` to header `<tr>`

### 5-2-05 ✅ Table Column Labels
- 標題, 年份, 類型, 評分, 加入日期 — all match design

### 5-2-06 ✅ Poster Thumbnail Size
- `w-12` (48px) × `h-[72px]` (72px) with `rounded` — matches design

### 5-2-07 ✅ Title Column (Primary + Secondary)
- White primary `text-sm font-medium`, muted `text-xs text-slate-500` secondary — matches

### 5-2-08 ✅ Year Column
- Centered, `text-sm text-slate-400` — matches

### 5-2-09 ✅ Genre Tags/Chips
- `rounded bg-slate-700 px-1.5 py-0.5 text-xs text-slate-300`, max 3 — matches

### 5-2-10 ✅ Rating Display
- `text-yellow-400` star + numeric value, centered — matches

### 5-2-11 ⚠️ Date Format
- **Design:** Appears to use shorter date format (e.g., "3月 22")
- **Implementation:** Full date `2026/03/15` via `toLocaleDateString('zh-TW')`
- **Fix:** Consider shorter format to match design; low priority

### 5-2-12 ✅ Table Row Borders
- `border-b border-slate-800 hover:bg-slate-800/50` — matches design separators

### 5-2-13 ✅ Table Row Height
- ~88px (72px poster + padding) — matches compact design

### 5-2-14 ✅ Sort Indicator Arrows on Headers
- `ArrowUp`/`ArrowDown` icons at size 14, shown only for active column — matches

### 5-2-15 ⚠️ Top Navigation Tabs Labels
- **Design:** "全部影片", "下載中", "待處理", "追蹤"
- **Implementation:** Tab nav component handles this globally; type filter "全部/電影/影集" is separate
- **Fix:** Verify tab labels in `TabNavigation.tsx` match design (may be global scope issue)

### 5-2-16 ⚠️ Missing "全部媒體" Section Heading
- **Design:** "全部媒體" as prominent section heading
- **Implementation:** Only pagination count string, no section title
- **Fix:** Add `<h2 className="text-xl font-semibold text-white">全部媒體</h2>` before controls

### 5-2-17 ⚠️ Redundant Sort Controls in List View
- **Design:** Table headers provide sorting; controls row also shows sort selector
- **Implementation:** Both SortSelector dropdown AND column headers sort simultaneously
- **Fix:** Consider hiding SortSelector when `currentView === 'list'` to avoid redundancy

### 5-2-18 ✅ Table Width and Column Proportions
- `w-full` with fixed widths for year/rating/date, flex for title/genre — matches

### 5-2-19 ✅ Poster Column Width
- `w-16` (64px) accommodates `w-12` (48px) image + padding — correct

### 5-2-20 ⚠️ Loading Skeleton (List View)
- **Design:** No specific list-view skeleton screenshot
- **Implementation:** Generic bars `h-14 animate-pulse rounded bg-slate-800` instead of table-row-shaped
- **Fix:** Low priority; could improve skeleton to mimic table structure

---

# Story 5-3: Library Search

**Design Reference:** `flow-a-browse-desktop/01-library-grid-desktop.png`, `flow-c-search-filter-settings-desktop/07-search-filter-desktop.png`

## Findings

### 5-3-01 ⚠️ Dual Search Bars
- **Design:** Single search bar in the HEADER only. No separate library-level search.
- **Implementation:** TWO search bars — global in header (`AppShell.tsx`, navigates to `/search`) + library-specific in content area (`LibrarySearchBar.tsx`)
- **Fix:** Either (a) repurpose header search on library page to do library search, or (b) hide header search when on library page. Dual bars may confuse users.

### 5-3-02 ⚠️ Library Search Bar Shape
- **Design:** Header search is `rounded-full` (pill-shaped)
- **Implementation:** LibrarySearchBar uses `rounded-lg` (rectangle with rounded corners)
- **Fix:** Change `rounded-lg` to `rounded-full` on LibrarySearchBar for consistency

### 5-3-03 ⚠️ Search Bar Width & Position
- **Design:** ~256px wide, centered in header
- **Implementation:** LibrarySearchBar is `max-w-md` (448px), left-aligned in content area
- **Fix:** If keeping separate, narrow to `max-w-sm` (~384px) and consider centering

### 5-3-04 ✅ Placeholder Text
- Library: "搜尋媒體標題..." — context-appropriate
- Header: "搜尋電影或影集..." — matches design

### 5-3-05 ✅ Search Icon Position
- Left-aligned `absolute left-3 top-1/2` with `h-5 w-5 text-slate-400` — matches

### 5-3-06 ✅ Clear Button (X)
- Conditionally shown when input has value, `absolute right-3` — matches

### 5-3-07 ✅ Result Count Display
- "找到 {resultCount} 個結果" in `text-sm text-slate-400` — matches AC4

### 5-3-08 ✅ Search Highlighting
- `<mark className="bg-yellow-500/30 text-inherit rounded-sm px-0.5">` — satisfies AC2

### 5-3-09 ✅ Empty Search — Icon
- `Search` (magnifying glass) icon `h-12 w-12 text-slate-500` — matches

### 5-3-10 ✅ Empty Search — Heading
- "找不到相關結果" + contextual description — matches AC3

### 5-3-11 ✅ Empty Search — Suggestions
- Three bullet suggestions matching AC3 requirements — matches

### 5-3-12 ✅ Empty Search — Fade-in Animation
- `animate-in fade-in duration-500 delay-500 fill-mode-backwards` — matches

### 5-3-13 ✅ Keyboard Shortcut (Cmd+K)
- Global keydown listener for `(metaKey || ctrlKey) && 'k'` — matches AC4

### 5-3-14 ✅ Debounce (500ms, ≥2 chars)
- `useDebouncedCallback` 500ms, `query.length >= 2` gate — matches AC1

### 5-3-15 ❌ Filter Sidebar During Search
- **Design:** 07-search-filter-desktop.png shows filter sidebar visible during search
- **Implementation:** Filter panel is HIDDEN when `isSearchActive` is true
- **Fix:** When search is active, filter sidebar should remain accessible (ties into 5-5 sidebar refactor)

### 5-3-16 ✅ Sort Hidden During Search
- Sort is hidden during active search — matches design (FTS sorts by relevance)

### 5-3-17 ⚠️ No Keyboard Shortcut Indicator
- **Design:** Some UIs show ⌘K badge in the search bar for discoverability
- **Implementation:** No visual indicator, only functional
- **Fix:** Add subtle `⌘K` badge on right side of search bar (when empty)

### 5-3-18 ⚠️ Search Bar in List View
- Same dual-search-bar issue as 5-3-01 — design shows only header search

### 5-3-19 ⚠️ Highlight Coverage
- Grid PosterCard highlights title only (originalTitle not shown on card)
- Table highlights both title + originalTitle
- Acceptable since grid card intentionally omits originalTitle for space

---

# Story 5-4: Library Sorting

**Design Reference:** `flow-a-browse-desktop/06-list-view-desktop.png`, `flow-d-browse-mobile/03a-m-sort-bottom-sheet.png`

## Findings

### 5-4-01 ⚠️ Sort Button Label Format
- **Design:** Combined phrase like "按新到舊" (by newest to oldest)
- **Implementation:** Field name "新增日期" + separate arrow icon
- **Fix:** Consider combined phrase format; current approach is functional but differs

### 5-4-02 ✅ Sort Selector Position
- Between filter and view toggle in controls row — matches design grouping

### 5-4-03 ✅ Sort Selector Icon
- `ArrowUpDown` icon (size 16) + label + direction arrow — standard sort indicator

### 5-4-04 ✅ Sort Selector Button Styling
- `rounded-lg bg-slate-800 px-3 py-2 text-sm text-slate-300` — matches dark theme

### 5-4-05 ⚠️ Active Sort Item Highlight
- **Design (mobile):** Active item shows blue arrow indicator only, normal background
- **Implementation:** Active item has full `bg-blue-600 text-white` row highlight
- **Fix:** Consider changing to normal bg + blue arrow indicator to match mobile design

### 5-4-06 ✅ Sort Options Labels
- 新增日期, 標題, 年份, 評分 — all 4 options match

### 5-4-07 ✅ Sortable Column Selection
- 標題, 年份, 評分, 加入日期 sortable; poster, 類型 not — correct

### 5-4-08 ✅ Column Header Sort Indicators
- `ArrowUp`/`ArrowDown` (size 14) inline with header text — matches

### 5-4-09 ✅ Column Header Hover
- `hover:text-white` transition from `text-slate-400` — reasonable

### 5-4-10 ⚠️ Column Sort Default Direction
- **SortSelector:** Uses field's `defaultOrder` (e.g., `desc` for dates/ratings)
- **Column header click:** Always starts at `asc` for new columns
- **Fix:** Align column header sort to use `defaultOrder` from SORT_OPTIONS

### 5-4-11 ✅ Grid View Sort Indicator
- Sort state visible in SortSelector button, not on individual cards — correct

### 5-4-12 ⚠️ Sort During Search
- **Design:** 07-search-filter-desktop.png may show sort controls during search
- **Implementation:** SortSelector hidden when `isSearchActive` (deliberate — FTS sorts by rank)
- **Fix:** Verify with UX designer; hiding is arguably better UX for relevance-based search

### 5-4-13 ✅ Default Sort
- `created_at` / `desc` (newest first) — matches spec

### 5-4-14 ✅ Sort Persistence
- URL params + localStorage, URL takes priority — matches AC3

### 5-4-15 ⚠️ Date Column Label Inconsistency
- **Table column:** "加入日期"
- **Sort dropdown:** "新增日期"
- **Fix:** Unify to one label (both mean "date added" but use different Chinese phrasing)

### 5-4-16 ✅ Non-active Items — No Arrow
- Arrow icon only shown for active sort field — matches

### 5-4-17 ❌ Mobile Sort — No Bottom Sheet
- **Design (mobile):** Bottom sheet with "排序方式" title + "完成" button + radio-style list
- **Implementation:** Same dropdown component on all viewports
- **Fix:** Implement responsive sort: dropdown on desktop, bottom sheet on mobile

---

# Story 5-5: Library Filtering

**Design Reference:** `flow-c-search-filter-settings-desktop/07-search-filter-desktop.png`

## Findings

### 5-5-01 ❌ Filter Panel Layout — Sidebar vs Dropdown
- **Design:** Persistent LEFT SIDEBAR (~200px wide) coexisting alongside grid
- **Implementation:** Dropdown `div` below toggle button, overlays content
- **Fix:** Redesign as sidebar with `flex` layout: `[sidebar | content]`

### 5-5-02 ❌ Missing Panel Title
- **Design:** "篩選條件" as panel heading
- **Implementation:** No panel heading; goes straight to filter sections
- **Fix:** Add `<h3>篩選條件</h3>` at top of panel

### 5-5-03 ❌ Genre Controls — Checkboxes vs Chips
- **Design:** Rounded-full pill chip toggles with checkmark icon for active state
- **Implementation:** HTML `<input type="checkbox">` in `rounded-md bg-slate-700` containers
- **Fix:** Replace checkboxes with chip-style toggle buttons (`rounded-full`), active state = checkmark icon + blue bg/border

### 5-5-04 ✅ Genre Section Label
- "類型" — matches design

### 5-5-05 ❌ Year Filter — Number Inputs vs Decade Chips
- **Design:** Decade chip toggles: `[2020s]` `[2010s]` `[2000s]` `[1990s]` `[更早]`
- **Implementation:** Two `<input type="number">` fields for min/max year
- **Fix:** Replace number inputs with decade chip toggles (same chip style as genre). Selecting `2020s` = years 2020-2029.

### 5-5-06 ⚠️ Year Section Label
- **Design:** "年份"
- **Implementation:** "年份範圍"
- **Fix:** Change to "年份"

### 5-5-07 ❌ Type Filter Placement
- **Design:** Type toggle (全部/電影/影集) is INSIDE filter sidebar as a section
- **Implementation:** Separate tab buttons OUTSIDE filter panel in main content area
- **Fix:** Move type toggle into FilterPanel as a section

### 5-5-08 ⚠️ Language Filter Section
- **Design:** Screenshot shows a 4th section (likely 語言/Language)
- **Implementation:** Not implemented
- **Fix:** Story AC1 doesn't require Language — document as future work. If in design scope, add Language section with checkboxes (日語, 韓語, 英語, 華語)

### 5-5-09 ⚠️ Apply/Reset Button Labels
- **Design:** "套用" + "重置"
- **Implementation:** "套用篩選" + "清除"
- **Fix:** Rename to match design labels

### 5-5-10 ⚠️ Filter Chip Color Coding
- **Design:** Uniform `--bg-tertiary` styling for all chips
- **Implementation:** Blue for genre (`bg-blue-600/20`), green for year (`bg-green-600/20`)
- **Fix:** Unify to single color scheme for all filter chips

### 5-5-11 ⚠️ Filter Count Display
- **Design AC2:** "顯示 45 / 500 項" (filtered / total)
- **Implementation:** "顯示 1-20 / 500 項" (pagination range / filtered count)
- **Fix:** Show filtered-vs-unfiltered count when filters are active

### 5-5-12 ✅ "清除全部篩選" Placement
- Inline text button with filter chips, `text-slate-400` — matches spec

### 5-5-13 ❌ Panel Background/Border Styling
- **Design:** Full-height sidebar, right border only, matches page bg
- **Implementation:** `rounded-lg border border-slate-700 bg-slate-800 p-4` card-style dropdown
- **Fix:** When converting to sidebar: full-height, right border only, no rounded corners

### 5-5-14 ✅ URL State Persistence
- Genres as comma-separated URL params, yearMin/yearMax — matches AC4

### 5-5-15 ✅ AND Logic
- All filter params sent simultaneously, backend applies AND — matches AC4

### 5-5-16 ✅ Page Reset on Filter Change
- `page: 1` set on filter apply/clear/chip remove — correct

### 5-5-17 ✅ Genre Active State (tied to 5-5-03)
- Needs checkmark icon on chip-style toggles when converting from checkboxes

---

## Cross-Story Issues (Affect Multiple Stories)

### XS-1: ❌ Filter Panel Architecture (5-2, 5-3, 5-5)
The filter panel should be a left sidebar. This affects:
- **5-2:** Controls row layout changes when sidebar is present
- **5-3:** Search should work with sidebar visible (not hidden)
- **5-5:** Core filter panel structure

### XS-2: ⚠️ Controls Row Order (5-2, 5-4)
Current: Filter → Sort → View Toggle
Design: Sort → Filter → View Toggle
Affects both grid and list views.

### XS-3: ⚠️ Missing "全部媒體" Section Heading (5-2)
No section title for the main content area in either view.

### XS-4: ❌ No Mobile-Specific Components (5-4, 5-5)
- Sort: No bottom sheet on mobile
- Filter: No bottom sheet on mobile
Both need responsive variants.

---

## Implementation Priority for Dev Agent

### Priority 1 — Critical Structural Fix (Story-level effort)
| ID | Issue | Files |
|----|-------|-------|
| 5-5-01 | Filter Panel → Sidebar layout | `FilterPanel.tsx`, `library.tsx` |
| 5-5-03 | Genre → Chip toggles with checkmark | `FilterPanel.tsx` |
| 5-5-05 | Year → Decade chip toggles | `FilterPanel.tsx` |
| 5-5-07 | Type filter → Inside sidebar | `FilterPanel.tsx`, `library.tsx` |
| 5-5-02 | Add "篩選條件" panel title | `FilterPanel.tsx` |
| 5-5-13 | Panel styling → Full-height sidebar | `FilterPanel.tsx` |

### Priority 2 — High (Quick fixes)
| ID | Issue | Files |
|----|-------|-------|
| XS-2 / 5-2-02 | Controls row order: Sort → Filter → Toggle | `library.tsx` |
| 5-2-01 / 5-2-16 | Add "全部媒體" section heading | `library.tsx` |
| 5-2-04 | Table header row background | `LibraryTable.tsx` |
| 5-3-15 | Keep filter accessible during search | `library.tsx` |
| 5-4-15 | Unify "加入日期" / "新增日期" labels | `LibraryTable.tsx`, `SortSelector.tsx` |

### Priority 3 — Medium (Styling tweaks)
| ID | Issue | Files |
|----|-------|-------|
| 5-3-02 | Search bar shape `rounded-lg` → `rounded-full` | `LibrarySearchBar.tsx` |
| 5-5-06 | Year label "年份範圍" → "年份" | `FilterPanel.tsx` |
| 5-5-09 | Button labels "套用篩選"→"套用", "清除"→"重置" | `FilterPanel.tsx` |
| 5-5-10 | Unify filter chip colors | `FilterChips.tsx` |
| 5-4-05 | Active sort item styling | `SortSelector.tsx` |
| 5-4-10 | Column sort default direction | `library.tsx` |

### Priority 4 — Low (Polish / Future)
| ID | Issue | Files |
|----|-------|-------|
| 5-3-01 | Dual search bars architecture | `AppShell.tsx`, `LibrarySearchBar.tsx` |
| 5-3-17 | ⌘K keyboard shortcut badge | `LibrarySearchBar.tsx` |
| 5-4-17 | Mobile sort bottom sheet | `SortSelector.tsx` |
| XS-4 | Mobile-specific filter/sort components | Multiple |
| 5-2-20 | List view loading skeleton | `LibraryTable.tsx` |
| 5-5-08 | Language filter (not in story scope) | Future story |

---

## Reference Screenshots

| Screenshot | Path | Relevant Stories |
|------------|------|-----------------|
| Grid View | `flow-a-browse-desktop/01-library-grid-desktop.png` | 5-2, 5-3, 5-4 |
| List View | `flow-a-browse-desktop/06-list-view-desktop.png` | 5-2, 5-4 |
| Search+Filter | `flow-c-search-filter-settings-desktop/07-search-filter-desktop.png` | 5-3, 5-5 |
| Empty State | `flow-a-browse-desktop/09a-empty-library-desktop.png` | 5-3 |
| Loading | `flow-a-browse-desktop/09b-loading-skeleton-desktop.png` | 5-2 |
| Mobile Sort | `flow-d-browse-mobile/03a-m-sort-bottom-sheet.png` | 5-4 |
| Mobile Filter | `flow-d-browse-mobile/10-filter-bottom-sheet-mobile.png` | 5-5 |
