# Story: Search Enhancement & Unimplemented Page Labels

Status: ready-for-dev

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

- [ ] Task 1: Add `enabled` flag to settings categories (AC: #3)
  - [ ] 1.1 `apps/web/src/components/settings/SettingsLayout.tsx`: Add `enabled?: boolean` field to the settings category type definition
  - [ ] 1.2 Set `enabled: false` for `export` (line ~53) and `performance` (line ~58) items
  - [ ] 1.3 Default `enabled` to `true` for all other existing categories

- [ ] Task 2: Visual indicator for unimplemented pages (AC: #1, #2)
  - [ ] 2.1 `apps/web/src/components/settings/SettingsLayout.tsx`: Render a "Coming Soon" badge next to disabled menu items (lines 47-61, sidebar rendering)
  - [ ] 2.2 Style disabled items with reduced opacity and `cursor: not-allowed`
  - [ ] 2.3 When a disabled item is clicked, prevent navigation and show a toast or tooltip: "此功能尚未實作 / Coming soon"
  - [ ] 2.4 If user navigates directly to `/settings/export` or `/settings/performance` via URL, show a placeholder page with "Coming Soon" message

- [ ] Task 3: Clarify search scope in UI (AC: #4)
  - [ ] 3.1 Identify the search input component used on the library page (likely in `apps/web/src/components/library/` or search-related component)
  - [ ] 3.2 Update placeholder text to "搜尋媒體庫..." or "Search library..."
  - [ ] 3.3 If there is a search bar in the global nav, ensure it is clear that it searches the library only (not downloads)

- [ ] Task 4: Tests (AC: all)
  - [ ] 4.1 `apps/web/src/components/settings/SettingsLayout.spec.tsx`: Test that disabled items render with "Coming Soon" badge
  - [ ] 4.2 `apps/web/src/components/settings/SettingsLayout.spec.tsx`: Test that clicking disabled item does not navigate
  - [ ] 4.3 Update any existing search component tests to verify updated placeholder text

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

### Debug Log References

### Completion Notes List

### File List
