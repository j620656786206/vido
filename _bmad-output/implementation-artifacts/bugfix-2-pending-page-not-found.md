# Story: Fix Pending Page Not Found

Status: ready-for-dev

## Story

As a user navigating via the tab bar,
I want the /pending route to load a valid page,
so that I can see pending download items without getting a "Not Found" error.

## Acceptance Criteria

1. Navigating to `/pending` renders a Pending page component without error
2. The TabNavigation link to `/pending` resolves correctly
3. The page displays a placeholder or real pending-downloads list (depending on backend readiness)
4. No console errors or router warnings on navigation

## Tasks / Subtasks

- [ ] Task 1: Create the pending route file (AC: #1, #2)
  - [ ] 1.1 Create `apps/web/src/routes/pending.tsx` with a basic route definition using TanStack Router file-based routing conventions
  - [ ] 1.2 Export a Pending page component (can be a placeholder with "Pending Downloads" heading)

- [ ] Task 2: Verify TabNavigation integration (AC: #2, #4)
  - [ ] 2.1 Confirm `apps/web/src/components/layout/TabNavigation.tsx:13` link href matches the new route path
  - [ ] 2.2 Manually verify no console errors on navigation

- [ ] Task 3: Add route test (AC: #1, #4)
  - [ ] 3.1 Create `apps/web/src/routes/pending.spec.tsx` with basic render test

## Dev Notes

### Root Cause

`TabNavigation.tsx:13` contains a link to `/pending`, but no corresponding route file exists at `apps/web/src/routes/pending.tsx`. TanStack Router's file-based routing requires a physical file to register the route, so navigation to `/pending` results in a "Not Found" page.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/routes/pending.tsx` | **Create** — new route file |
| `apps/web/src/components/layout/TabNavigation.tsx` | Read-only — verify link at line 13 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
