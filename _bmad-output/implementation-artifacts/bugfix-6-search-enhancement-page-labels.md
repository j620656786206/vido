# Story: Search Enhancement & Unimplemented Page Labels

Status: done

## Story

As a user navigating the Vido interface,
I want to clearly see which settings pages are not yet implemented and understand that search only covers my library,
so that I do not waste time clicking on non-functional menu items or wondering why downloads do not appear in search results.

## Acceptance Criteria

1. Unimplemented settings pages (`export`, `performance`) show a visible "Coming Soon" or "尚未實作" badge/tag next to their menu items in the sidebar
2. Clicking an unimplemented settings menu item either shows a placeholder page with explanation or disables the link with a tooltip
3. The `SETTINGS_CATEGORIES` data structure supports an `enabled` flag to control which items are active
4. The library search bar includes a subtle hint or placeholder text indicating it searches the local library (e.g., "搜尋媒體庫..." / "Search library...")
5. No changes to actual search functionality — this is a UX clarity improvement only

## Tasks / Subtasks

- [x] Task 1: Add `enabled` flag to settings categories (AC: #3)
  - [x] 1.1 Added `enabled?: boolean` to SettingsCategory interface
  - [x] 1.2 Set `enabled: false` for export and performance items
  - [x] 1.3 All other categories default to true (checked via `cat.enabled !== false`)

- [x] Task 2: Visual indicator for unimplemented pages (AC: #1, #2)
  - [x] 2.1 Disabled items render as `<span>` with "Coming Soon" badge (desktop + mobile)
  - [x] 2.2 `cursor-not-allowed` + `text-slate-600` styling for disabled items
  - [x] 2.3 Disabled items are `<span>` not `<Link>` — no navigation possible + title tooltip
  - [x] 2.4 Routes already have placeholder pages (SettingsPlaceholder) — verified

- [x] Task 3: Clarify search scope in UI (AC: #4)
  - [x] 3.1 Identified: AppShell.tsx, QuickSearchBar.tsx, SearchBar.tsx
  - [x] 3.2 Updated all placeholder text from "搜尋電影或影集..." to "搜尋媒體庫..."
  - [x] 3.3 LibrarySearchBar already uses "搜尋媒體標題..." — clear enough

- [x] Task 4: Tests (AC: all)
  - [x] 4.1 Updated SettingsLayout tests: disabled items have Coming Soon badge, cursor-not-allowed, text-slate-600
  - [x] 4.2 Updated href tests: disabled items render as SPAN (no href), verified via tagName check
  - [x] 4.3 Updated all search placeholder tests: AppShell, SearchBar, QuickSearchBar, -search specs

## Dev Notes

### Root Cause Analysis

**Search scope confusion:** The search endpoint (`/api/v1/library/search`) only queries the local library database. Downloads are a separate system accessed via `/api/v1/downloads/`. There is no combined search. Users may expect search to cover everything, so the UI should clarify scope.

**Unimplemented pages:** `SettingsLayout.tsx` lines 47-61 define `SETTINGS_CATEGORIES` which includes `export` and `performance` items. These render as normal clickable menu items but have no corresponding route or component. The data structure has no `enabled` flag, so there is no way to distinguish implemented from unimplemented items.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/components/settings/SettingsLayout.tsx` | Add `enabled` flag, badge, disable logic (lines 47-61) |
| `apps/web/src/components/library/` (search component) | Update search placeholder text |

### References

- [Source: apps/web/src/components/settings/SettingsLayout.tsx:47-61] — SETTINGS_CATEGORIES with export/performance
- [Source: apps/api/internal/handlers/library_handler.go] — /api/v1/library/search endpoint
- [Source: apps/api/internal/handlers/download_handler.go] — separate downloads system

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Added `enabled?: boolean` flag to SettingsCategory, set false for export/performance
- Task 2: Disabled items render as `<span>` with "Coming Soon" badge, no navigation. Desktop + mobile.
- Task 3: Updated search placeholder from "搜尋電影或影集..." to "搜尋媒體庫..." across 3 source files
- Task 4: Updated 37 SettingsLayout tests + 9 search placeholder tests. 126 files / 1562 tests pass.

### File List

- apps/web/src/components/settings/SettingsLayout.tsx (modified — enabled flag, Coming Soon badge)
- apps/web/src/components/settings/SettingsLayout.spec.tsx (modified — tests for disabled items)
- apps/web/src/components/shell/AppShell.tsx (modified — search placeholder)
- apps/web/src/components/shell/AppShell.spec.tsx (modified — updated placeholder assertion)
- apps/web/src/components/dashboard/QuickSearchBar.tsx (modified — search placeholder)
- apps/web/src/components/dashboard/QuickSearchBar.spec.tsx (modified — updated placeholder)
- apps/web/src/components/search/SearchBar.tsx (modified — search placeholder default)
- apps/web/src/components/search/SearchBar.spec.tsx (modified — updated placeholder)
- apps/web/src/routes/-search.spec.tsx (modified — updated placeholder)
