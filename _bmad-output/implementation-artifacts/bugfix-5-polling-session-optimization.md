# Story: Polling & Session Optimization

Status: ready-for-dev

## Story

As a user running Vido in production,
I want the frontend to avoid unnecessary duplicate polling and the backend to reuse qBittorrent sessions,
so that the system uses fewer resources and does not overwhelm external services with redundant requests.

## Acceptance Criteria

1. Only ONE polling mechanism actively polls `/scanner/status` at any given time (no duplicate requests visible in Network tab)
2. qBittorrent `Login()` is only called when the session is expired or missing (lazy auth), not on every API call
3. Frontend qBittorrent polling at 5s interval reuses an existing authenticated session without re-authenticating
4. When qBittorrent returns 401/403, the client automatically re-authenticates and retries the failed request once
5. Scanner polling frequency: 3s when scanning, 30s when idle (single poller)
6. No user-visible behavior change — scanner progress and download status continue to update correctly

## Tasks / Subtasks

- [ ] Task 1: Consolidate scanner polling into a single hook (AC: #1, #5)
  - [ ] 1.1 `apps/web/src/hooks/useScanProgress.ts`: Remove the 3s fallback polling and 10s idle polling — this hook should only provide SSE-based progress
  - [ ] 1.2 `apps/web/src/hooks/useScanner.ts`: Make this the single source of truth for polling `/scanner/status` (3s scanning, 30s idle)
  - [ ] 1.3 `apps/web/src/components/settings/ScannerSettings.tsx`: Ensure component uses only one hook for polling, remove any redundant hook usage
  - [ ] 1.4 Verify no other components import `useScanProgress` for polling purposes

- [ ] Task 2: Implement lazy auth in qBittorrent client (AC: #2, #3, #4)
  - [ ] 2.1 `apps/api/internal/qbt/client.go`: Remove unconditional `Login()` call from `GetTorrents()` and `GetTorrentDetails()`
  - [ ] 2.2 Add a `ensureAuth()` method that checks if session cookie exists in CookieJar before calling Login()
  - [ ] 2.3 Add retry-on-401/403 logic: if API call returns 401/403, call `Login()` once, then retry the original request
  - [ ] 2.4 Add a `lastLogin` timestamp field; skip re-login if last successful login was within a configurable TTL (e.g., 30 minutes)

- [ ] Task 3: Tests (AC: all)
  - [ ] 3.1 `apps/web/src/hooks/useScanProgress.spec.ts`: Update tests to confirm no polling timer is set by this hook
  - [ ] 3.2 `apps/web/src/hooks/useScanner.spec.ts`: Add test confirming single polling interval at correct frequencies
  - [ ] 3.3 `apps/api/internal/qbt/client_test.go`: Add test for lazy auth — Login() not called when session exists
  - [ ] 3.4 `apps/api/internal/qbt/client_test.go`: Add test for 401 retry — Login() called on 401, then request retried

## Dev Notes

### Root Cause Analysis

**Scanner double-polling:** Two independent hooks both poll `/scanner/status`:
- `useScanProgress.ts` has a 3s fallback polling interval and a 10s idle interval, originally as SSE fallback
- `useScanner.ts` polls at 3s when scanning, 30s otherwise
- `ScannerSettings.tsx` uses both hooks, causing doubled requests to the same endpoint

**qBittorrent session waste:** `client.go` calls `Login()` unconditionally at the start of every `GetTorrents()` and `GetTorrentDetails()` invocation. The frontend polls every 5s, meaning Login() is called every 5s. A `CookieJar` is already configured on the HTTP client, but `Login()` overwrites the session cookie each time, wasting a round-trip and creating unnecessary load on the qBittorrent Web API.

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/hooks/useScanProgress.ts` | Remove polling fallback, keep SSE-only |
| `apps/web/src/hooks/useScanner.ts` | Single source of polling truth |
| `apps/web/src/components/settings/ScannerSettings.tsx` | Use single polling hook |
| `apps/api/internal/qbt/client.go` | Lazy auth with retry-on-401 |

### References

- [Source: apps/web/src/hooks/useScanProgress.ts] — 3s/10s fallback polling
- [Source: apps/web/src/hooks/useScanner.ts] — 3s/30s polling
- [Source: apps/web/src/components/settings/ScannerSettings.tsx] — uses both hooks
- [Source: apps/api/internal/qbt/client.go] — Login() called in GetTorrents/GetTorrentDetails

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
