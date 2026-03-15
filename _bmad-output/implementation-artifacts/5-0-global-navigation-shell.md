# Story 5.0: Global Navigation Shell

Status: done

## Story

As a **Vido user**,
I want a **consistent navigation shell across all pages**,
So that **I can easily switch between sections and always know where I am in the app**.

## Design Reference

- Desktop: `_bmad-output/screenshots/flow-a-browse-desktop/01-library-grid-desktop.png`
- Desktop (empty): `_bmad-output/screenshots/flow-a-browse-desktop/09a-empty-library-desktop.png`
- Mobile: `_bmad-output/screenshots/flow-d-browse-mobile/03-library-grid-mobile.png`
- Mobile (empty): `_bmad-output/screenshots/flow-d-browse-mobile/09a-m-empty-library-mobile.png`

## Acceptance Criteria

1. **AC1: Global Dark Theme Baseline**
   - Given any page in the app
   - When the page loads
   - Then the root layout uses `bg-slate-900` dark theme consistently
   - And all pages inherit this dark baseline without needing to override

2. **AC2: Top Header Bar (Desktop)**
   - Given the user is on any page (desktop viewport)
   - When the page loads
   - Then the header displays:
     - Left: "vido" text logo (blue/cyan accent color)
     - Center: Global search bar (rounded, with search icon, placeholder "搜尋電影或影集...")
     - Right: Settings gear icon
   - And the header is fixed/sticky at the top

3. **AC3: Top Header Bar (Mobile)**
   - Given the user is on any page (mobile viewport)
   - When the page loads
   - Then the header displays:
     - Left: "vido" text logo
     - Right: Search icon + Settings gear icon (compact)
   - And the search bar is hidden (accessible via search icon tap)

4. **AC4: Tab Navigation**
   - Given the header is displayed
   - When viewing the navigation tabs
   - Then four tabs are shown: 媒體庫, 下載中, 待解析, 設定
   - And the active tab has a visual indicator (underline or highlight)
   - And clicking a tab navigates to the corresponding route:
     - 媒體庫 → `/library`
     - 下載中 → `/downloads`
     - 待解析 → `/pending`
     - 設定 → `/settings`
   - And on mobile, tabs are horizontally scrollable if needed

5. **AC5: Active Route Highlighting**
   - Given the user is on a specific page
   - When viewing the navigation tabs
   - Then the corresponding tab is visually highlighted as active
   - And the highlight updates when navigating between pages

6. **AC6: Existing Pages Integration**
   - Given the navigation shell is implemented
   - When visiting existing pages (dashboard `/`, library `/library`, downloads `/downloads`, settings `/settings/*`)
   - Then all pages render correctly within the new shell
   - And no existing functionality is broken
   - And page-specific headers (e.g., "媒體庫" title in library) may be removed or adapted to avoid duplication

## Technical Notes

- Modify `apps/web/src/routes/__root.tsx` to include the navigation shell
- The shell replaces individual page headers — pages should no longer define their own top-level header
- Search bar in the shell may reuse or adapt `QuickSearchBar` component
- Settings gear may reuse `SettingsGearDropdown` from library page
- Use TanStack Router's `Link` with `activeProps` for active tab styling
- Must be responsive: desktop layout vs mobile layout per design screenshots

## Tasks / Subtasks

- [x] Task 1: Update Root Layout with Dark Theme (AC: 1)
  - [x] 1.1: Change `__root.tsx` background to `bg-slate-900` (already done as hot fix)
  - [x] 1.2: Ensure all child pages work correctly with dark baseline
  - [x] 1.3: Remove redundant `bg-slate-900` from individual page components (dashboard, etc.)

- [x] Task 2: Create AppShell Component (AC: 2, 3)
  - [x] 2.1: Create `apps/web/src/components/shell/AppShell.tsx` with header layout
  - [x] 2.2: Implement desktop header: vido logo (left), search bar (center), settings gear (right)
  - [x] 2.3: Implement mobile header: vido logo (left), search + settings icons (right)
  - [x] 2.4: Write component tests for AppShell (8 tests)

- [x] Task 3: Create Tab Navigation Component (AC: 4, 5)
  - [x] 3.1: Create `apps/web/src/components/shell/TabNavigation.tsx`
  - [x] 3.2: Implement four tabs with TanStack Router `Link` and route matching
  - [x] 3.3: Style active tab with underline indicator matching design
  - [x] 3.4: Implement mobile responsive tab layout (overflow-x-auto)
  - [x] 3.5: Write component tests for TabNavigation (6 tests)

- [x] Task 4: Integrate Shell into Root Layout (AC: 6)
  - [x] 4.1: Wrap `<Outlet />` with AppShell in `__root.tsx`
  - [x] 4.2: Remove/adapt individual page headers that conflict with shell
  - [x] 4.3: Ensure dashboard, library, downloads, and settings pages render correctly
  - [x] 4.4: Verify no existing tests are broken (819 tests passing)

- [x] Task 5: Design Verification
  - [x] 5.1: Compare running app against desktop design screenshots
  - [x] 5.2: Compare running app against mobile design screenshots
  - [x] 5.3: Document any deviations and get SM/UX/User approval

## Dev Agent Record

### Implementation Notes
- AppShell wraps all pages via `__root.tsx` providing consistent header, search, and tab navigation
- Removed redundant `min-h-screen bg-slate-900` from 6 route files since root provides it
- Removed dashboard page header (Vido title + QBStatusIndicator) — replaced by global shell
- Removed search page h1 title — global search bar in shell replaces it
- Removed library page h1 — tab navigation shows active "媒體庫" tab
- Dashboard QuickSearchBar and ConnectionHistoryPanel removed from dashboard — global search in shell
- TabNavigation uses `useRouterState` for route matching instead of `activeProps` for more control

### Decisions Made
- Used `useRouterState` + `startsWith` for tab active state instead of `activeProps` — allows matching nested routes (e.g., `/settings/qbittorrent` matches `/settings` tab)
- Settings link goes to `/settings/qbittorrent` since that's the only settings page currently
- Tab "待解析" links to `/pending` route (to be created in future story)
- Mobile search bar is expandable via toggle button rather than always visible
- Kept SettingsGearDropdown on library page (display preferences), separate from global settings gear

## File List

### Created Files
- `apps/web/src/components/shell/AppShell.tsx`
- `apps/web/src/components/shell/AppShell.spec.tsx`
- `apps/web/src/components/shell/TabNavigation.tsx`
- `apps/web/src/components/shell/TabNavigation.spec.tsx`
- `apps/web/src/components/shell/index.ts`

### Modified Files
- `apps/web/src/routes/__root.tsx` — integrated AppShell, dark theme baseline
- `apps/web/src/routes/index.tsx` — removed redundant header and bg, removed QuickSearchBar
- `apps/web/src/routes/library.tsx` — removed redundant bg and h1 header
- `apps/web/src/routes/search.tsx` — removed redundant bg and h1 title
- `apps/web/src/routes/media/$type.$id.tsx` — removed redundant bg
- `apps/web/src/routes/test/manual-search.tsx` — removed redundant bg
- `apps/web/src/routes/-search.spec.tsx` — updated test for removed h1 title
- `apps/web/src/components/dashboard/QuickSearchBar.tsx` — fixed form relative positioning (pre-existing bug)

## Changelog

| Change | Reason | Date |
|--------|--------|------|
| Story created | Global navigation shell missing from all pages, identified during 5-1 design review | 2026-03-15 |
| Tasks 1-4 implemented | AppShell + TabNavigation created, integrated into root, pages adapted, 819 tests passing | 2026-03-15 |
| Task 5 completed | Design verification done — fixed tab label 待處理→待解析 per design spec, all elements match | 2026-03-15 |
| Code Review fixes | H1: Added sticky header positioning (AC2). M1: Removed 4x bg-slate-900 redundancy. M2: Removed unused SettingsGearDropdown import. M3: Fixed AC4 typo. L1: Removed duplicate min-h-screen. | 2026-03-15 |
