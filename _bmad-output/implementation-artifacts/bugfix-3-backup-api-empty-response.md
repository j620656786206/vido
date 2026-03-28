# Story: Harden Backup API Response Handling

Status: ready-for-dev

## Story

As a user visiting the Settings > Backup page,
I want the backup list to load without errors even when no backups exist,
so that I see an empty state instead of a JSON parse crash.

## Acceptance Criteria

1. `backupService.ts` does not throw when the API returns an empty body or non-JSON response
2. When the backend returns an empty backup list (valid JSON `[]` or `{"backups":[]}`), the UI renders an empty state
3. `fetchApi` utility gracefully handles non-JSON and empty responses across all callers
4. No unhandled promise rejections in the console on the backup settings page

## Tasks / Subtasks

- [ ] Task 1: Harden fetchApi response parsing (AC: #1, #3)
  - [ ] 1.1 In the shared fetch utility, check `Content-Length` header or response body before calling `.json()`
  - [ ] 1.2 If body is empty or content-type is not JSON, return a safe default (e.g., `null` or `{}`)
  - [ ] 1.3 Wrap `.json()` in try/catch as a secondary safety net

- [ ] Task 2: Fix backupService caller (AC: #1, #2)
  - [ ] 2.1 `apps/web/src/services/backupService.ts:82` — add null/empty guard before accessing `response.json()` result
  - [ ] 2.2 Return empty array `[]` when response body is absent or parsing fails

- [ ] Task 3: Add tests (AC: #1, #3, #4)
  - [ ] 3.1 Unit test fetchApi with empty response body — no throw
  - [ ] 3.2 Unit test fetchApi with non-JSON content-type — no throw
  - [ ] 3.3 Unit test backupService.listBackups with empty array response — returns `[]`

## Dev Notes

### Root Cause

`backupService.ts:82` calls `response.json()` unconditionally on every API response. While the backend `ListBackups` handler does return valid JSON (an empty array `[]`), the frontend fetch utility has no resilience for edge cases where the response body might be empty or non-JSON (e.g., 204 No Content, network errors, proxy interference). This makes the entire backup page fragile.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/services/backupService.ts` | Guard `.json()` call at line ~82 |
| `apps/web/src/utils/fetchApi.ts` (or equivalent) | Add safe JSON parsing |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
