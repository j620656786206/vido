> **STATUS: COMPLETED (v3 PRD)**
> This epic was completed under the v3 PRD structure. Its work is fully integrated
> into the v4 codebase. See [Completed Work Registry](./completed-work-registry.md)
> for the mapping to v4 feature IDs.

# Epic 5: Media Library Management

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can browse, search, filter, and manage their complete media library collection.

**Design Reference:** `ux-design.pen` (Pencil app) | Screenshots: `_bmad-output/screenshots/`

---

## Development Order & Dependencies

Stories MUST be developed in the following order due to UI and data dependencies:

```
Phase 0: Global Shell (prerequisite — all pages depend on this)
  5-0 (Navigation Shell)                  ← Global header, tabs, dark theme

Phase 1: Foundation
  5-1 (Grid View) + 5-8 (Recently Added)  ← Same page, develop together (DONE)

Phase 2: View Options
  5-2 (List View Toggle)                  ← Depends on 5-1 page skeleton

Phase 3: Data Operations (parallelizable)
  5-3 (Search)   ← Depends on 5-1 grid/list rendering
  5-4 (Sorting)  ← Depends on 5-1 grid/list rendering

Phase 4: Filtering
  5-5 (Filtering) ← Depends on 5-1 + needs genre API endpoint

Phase 5: Detail & Batch
  5-6 (Detail Page)     ← Depends on 5-1 card click interaction
  5-7 (Batch Operations) ← Depends on 5-1 card selection mechanism
```

**Key notes:**
- 5-0 is the global app shell — provides consistent navigation, header, and dark theme for all pages
- 5-1 is the foundation — all other stories depend on its page skeleton, grid rendering, and API integration
- 5-8 shares the same page layout as 5-1 (Recently Added section above the main grid), so they should be developed together
- 5-3 and 5-4 are independent of each other and can be developed in parallel
- 5-6 and 5-7 can also be parallelized if needed, but both are larger stories

**Development Workflow Rule (added 2026-03-15):**
- Dev Agent must verify UI matches design screenshots after every story completion
- Deviations must be flagged to SM (Bob) + UX (Sally) + User (Alexyu) for scope decision

---

## Story 5.1: Media Library Grid View

As a **media collector**,
I want to **browse my media library in a visual grid**,
So that **I can enjoy seeing my collection with beautiful posters**.

**Acceptance Criteria:**

**Given** the user opens the Library page
**When** the page loads
**Then** media items display in a responsive grid
**And** each card shows: poster, title (zh-TW), year, rating

**Given** the library has more than 1,000 items
**When** scrolling through the grid
**Then** virtual scrolling is enabled (NFR-SC6)
**And** scrolling maintains 60 FPS (NFR-P10)

**Given** the grid is displayed
**When** hovering over a card (desktop)
**Then** additional info appears: genre, description preview, metadata source
**And** a `...` (three-dot) icon appears at the top-right of the card
**And** the card has a subtle animation effect

**Given** the user clicks the `...` icon on a poster card
**When** the context menu opens
**Then** menu items include: View Details, Re-parse Metadata, Export Metadata, Delete
**And** Delete appears last with `--error` (red) color and requires confirmation dialog
**And** each menu item is prefixed with a Lucide icon
**And** on mobile, the context menu triggers via long-press and presents as a bottom sheet

**Given** the library toolbar is displayed
**When** the user clicks the Settings gear icon
**Then** a dropdown shows library display preferences:
- Poster Size / Density (Small / Medium / Large)
- Default Sort Preference
- Title Display Language (zh-TW priority / Original title priority)

**Technical Notes:**
- Implements FR38: Browse complete media library
- Implements FR40: Single-item operations via context menu (delete, re-parse, export)
- Implements NFR-SC6, NFR-P10: Virtual scrolling, 60 FPS
- Implements UX-9: Appreciation Loop
- Implements PRD UI Component Interaction Specifications (Settings Gear, PosterCard Context Menu)

---

## Story 5.2: Library List View Toggle

As a **media collector**,
I want to **switch between grid and list views**,
So that **I can choose the display format that suits my preference**.

**Acceptance Criteria:**

**Given** the library is displayed in grid view
**When** the user clicks the "List View" toggle
**Then** the display switches to a table/list format
**And** columns include: poster thumbnail, title, year, genre, rating, date added

**Given** list view is active
**When** the user clicks a column header
**Then** the list sorts by that column
**And** ascending/descending toggle is available

**Given** the user's view preference
**When** they return to the library later
**Then** their preferred view (grid/list) is remembered

**Technical Notes:**
- Implements FR8: Toggle between grid and list view
- View preference stored in localStorage
- Table component with sortable columns

---

## Story 5.3: Library Search

As a **media collector**,
I want to **search within my saved media library**,
So that **I can quickly find specific titles in my collection**.

**Acceptance Criteria:**

**Given** the user is on the Library page
**When** they type in the search box
**Then** results filter in real-time as they type
**And** both Chinese and English titles are searched

**Given** a search query is entered
**When** results are displayed
**Then** matching terms are highlighted
**And** search completes within 500ms (NFR-SC8)

**Given** no results match the query
**When** the search completes
**Then** a friendly message suggests: "No results found. Try a different search term or add new media."

**Technical Notes:**
- Implements FR5: Search within saved media library
- Implements NFR-SC8: SQLite FTS5 full-text search
- Debounced input for performance

---

## Story 5.4: Library Sorting

As a **media collector**,
I want to **sort my library by different criteria**,
So that **I can organize my view based on what I'm looking for**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user opens the sort dropdown
**Then** options include:
- Date Added (newest/oldest)
- Title (A-Z / Z-A)
- Year (newest/oldest)
- Rating (highest/lowest)

**Given** a sort option is selected
**When** the sort is applied
**Then** the library reorders immediately
**And** the current sort is indicated in the UI

**Given** the user's sort preference
**When** they return to the library
**Then** their last used sort is applied

**Technical Notes:**
- Implements FR6: Sort media library
- Sort state persisted in URL and localStorage
- Efficient sorting with database indexes

---

## Story 5.5: Library Filtering

As a **media collector**,
I want to **filter my library by genre, year, and media type**,
So that **I can narrow down to specific categories**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user opens the filter panel
**Then** filter options include:
- Genre (multi-select)
- Year range (slider or inputs)
- Media Type (Movie, TV Show)

**Given** filters are applied
**When** the library updates
**Then** only matching items are displayed
**And** the filter count is shown: "Showing 45 of 500 items"

**Given** multiple filters are active
**When** viewing the filter status
**Then** active filters are shown as removable chips
**And** a "Clear all filters" option is available

**Technical Notes:**
- Implements FR7: Filter media library
- Filters work in combination (AND logic)
- Filter state persisted in URL for sharing

---

## Story 5.6: Media Detail Page (Full Version)

As a **media collector**,
I want to **view comprehensive details about media in my library**,
So that **I can access all information including cast, trailers, and metadata source**.

**Acceptance Criteria:**

**Given** the user clicks on a library item
**When** the detail page opens
**Then** it displays all information from Story 2.4 plus:
- Full cast list with roles
- Embedded trailer (YouTube)
- Metadata source indicator (TMDb/Douban/Wikipedia/AI/Manual)
- File information (filename, size, quality)
- Date added to library

**Given** trailers are available
**When** the user clicks "Watch Trailer"
**Then** the YouTube video plays in an embedded player
**And** doesn't navigate away from the page

**Given** the metadata source is displayed
**When** hovering over the source badge
**Then** details show: "Fetched from TMDb on 2026-01-10"

**Given** the detail panel is open
**When** the user clicks the `...` icon (top-right, next to close button)
**Then** a context menu opens with: Re-parse Metadata, Export Metadata, Delete
**And** Delete appears last with `--error` (red) color and requires confirmation dialog
**And** each menu item is prefixed with a Lucide icon
**And** on mobile, the menu presents as a bottom sheet

**Technical Notes:**
- Implements FR39: View media detail pages with cast/trailers
- Implements FR40: Single-item operations via context menu (delete, re-parse, export)
- Implements FR42: Display metadata source indicators
- Implements PRD UI Component Interaction Specifications (Detail Panel Context Menu)
- YouTube embed with privacy-enhanced mode

---

## Story 5.7: Batch Operations

As a **power user**,
I want to **perform batch operations on multiple media items**,
So that **I can efficiently manage large numbers of files**.

**Acceptance Criteria:**

**Given** the library is displayed
**When** the user enters "selection mode"
**Then** checkboxes appear on each item
**And** a toolbar shows available batch actions

**Given** multiple items are selected
**When** batch actions are available
**Then** options include:
- Delete selected
- Re-parse selected
- Export metadata

**Given** the user selects "Delete selected"
**When** confirming the action
**Then** a confirmation dialog shows item count
**And** upon confirmation, items are removed from library

**Given** a batch operation is in progress
**When** processing multiple items
**Then** a progress indicator shows: "Processing 5 of 20..."
**And** errors are collected and shown at the end

**Technical Notes:**
- Implements FR40: Batch operations (delete, re-parse)
- Confirmation required for destructive operations
- Progress tracking for large batches

---

## Story 5.8: Recently Added Section

As a **media collector**,
I want to **see recently added media prominently**,
So that **I can quickly access my newest additions**.

**Acceptance Criteria:**

**Given** the user opens the Library page
**When** the page loads
**Then** a "Recently Added" section shows the newest 10-20 items
**And** items are sorted by date added (newest first)

**Given** new media is added
**When** the library updates within 30 seconds (NFR-P9)
**Then** the new item appears at the top of "Recently Added"
**And** a subtle animation highlights the new addition

**Given** the user clicks "See All"
**When** navigating to the full library
**Then** the sort is set to "Date Added (newest)"
**And** all items are visible

**Technical Notes:**
- Implements FR41: View recently added media items
- Implements NFR-P9: Library updates <30 seconds
- Section appears on Library page and Dashboard

---
