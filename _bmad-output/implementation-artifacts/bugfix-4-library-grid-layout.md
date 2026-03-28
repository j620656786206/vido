# Story: Fix Library Grid Layout Width

Status: ready-for-dev

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

- [ ] Task 1: Replace container class in library page (AC: #1, #2, #3)
  - [ ] 1.1 `apps/web/src/routes/library.tsx:503` — replace `container mx-auto` with `max-w-7xl mx-auto w-full`
  - [ ] 1.2 Verify the grid's `grid-cols-*` responsive breakpoints produce 4+ columns at 1280px width

- [ ] Task 2: Visual verification (AC: #2, #4, #5)
  - [ ] 2.1 Check layout at 1280px, 1440px, and 1920px viewports — expect 4-6 columns
  - [ ] 2.2 Check layout at 768px (tablet) — expect 2-3 columns
  - [ ] 2.3 Check layout at 375px (mobile) — expect 1-2 columns
  - [ ] 2.4 Confirm no horizontal overflow at any breakpoint

- [ ] Task 3: Update tests if applicable (AC: #1)
  - [ ] 3.1 If snapshot tests exist for library page, update expected class names

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

### Debug Log References

### Completion Notes List

### File List
