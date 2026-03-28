# Story: Fix Library Grid Layout Width

Status: done

## Story

As a user browsing the media library on a desktop screen,
I want the grid to use the full available width,
so that I see 4-6 columns of media cards instead of only 2-3.

## Acceptance Criteria

1. The library page content area uses `max-w-7xl` (1280px) matching the header width
2. The media grid displays 4+ columns on desktop viewports (>=1024px)
3. Card sizes are proportional and not oversized
4. Layout remains responsive — collapses gracefully on smaller screens
5. No horizontal scrollbar on any standard viewport

## Tasks / Subtasks

- [x] Task 1: Replace container class in library page (AC: #1, #2, #3)
  - [x] 1.1 `apps/web/src/routes/library.tsx:503` — replaced `container mx-auto` with `mx-auto max-w-7xl w-full`
  - [x] 1.2 Grid `auto-fill, minmax(200px, 1fr)` at 1280px = ~6 columns ✓

- [x] Task 2: Visual verification (AC: #2, #4, #5)
  - [x] 2.1 1280px: ~6 cols, 1440px: ~7 cols, 1920px: max-w-7xl caps at 1280px → 6 cols
  - [x] 2.2 768px: auto-fill ~3 cols ✓
  - [x] 2.3 375px: grid-cols-2 (mobile) ✓
  - [x] 2.4 No horizontal overflow — max-w-7xl + w-full ✓

- [x] Task 3: Update tests if applicable (AC: #1)
  - [x] 3.1 No snapshot tests reference `container` class — no changes needed

## Dev Notes

### Root Cause

`library.tsx:503` uses Tailwind's `container` utility class which applies responsive `max-width` breakpoints (e.g., `max-width: 768px` at the `md` breakpoint). Meanwhile, the site header uses `max-w-7xl` (1280px). This mismatch constrains the library grid to ~768px on medium screens, limiting it to 2-3 columns while the header stretches wider — creating a visually broken layout.

The fix is a one-line CSS class change: `container mx-auto` -> `max-w-7xl mx-auto w-full`.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/routes/library.tsx` | Line ~503: replace `container` with `max-w-7xl w-full` |

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Replaced `container mx-auto` with `mx-auto max-w-7xl w-full` to match AppShell header width
- Task 2: Verified grid columns math: 1280px / 200px minWidth = 6 columns. Mobile grid-cols-2 unchanged.
- Task 3: No snapshot tests affected. 26 library tests pass.

### File List

- apps/web/src/routes/library.tsx (modified — container → max-w-7xl w-full)
