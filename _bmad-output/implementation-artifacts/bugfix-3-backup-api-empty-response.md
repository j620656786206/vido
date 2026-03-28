# Story: Harden Backup API Response Handling

Status: done

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

- [x] Task 1: Harden fetchApi response parsing (AC: #1, #3)
  - [x] 1.1 Handle 204 No Content — return safe default without calling .json()
  - [x] 1.2 If .json() throws (empty/malformed body), return safe default for ok responses
  - [x] 1.3 Wrap `.json()` in try/catch as secondary safety net

- [x] Task 2: Fix backupService caller (AC: #1, #2)
  - [x] 2.1 `fetchApi` now handles empty/malformed responses — all callers benefit automatically
  - [x] 2.2 Returns safe default `{}` (via snakeToCamel) when parsing fails on ok responses

- [x] Task 3: Add tests (AC: #1, #3, #4)
  - [x] 3.1 Unit test fetchApi with 204 No Content — no throw
  - [x] 3.2 Unit test fetchApi with malformed JSON — no throw
  - [x] 3.3 Unit test fetchApi with empty JSON body — returns safe default
  - [x] 3.4 Unit test fetchApi throws on non-ok + unparseable body

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

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Hardened backupService's fetchApi — handles 204 No Content and wraps .json() in try/catch
- Task 2: All callers benefit from fetchApi resilience — no individual caller changes needed
- Task 3: Added 4 resilience tests: 204, malformed JSON, empty body, non-ok + unparseable. 126 files / 1564 tests pass.

### File List

- apps/web/src/services/backupService.ts (modified — hardened fetchApi response parsing)
- apps/web/src/services/backupService.spec.ts (modified — added 4 resilience tests)
