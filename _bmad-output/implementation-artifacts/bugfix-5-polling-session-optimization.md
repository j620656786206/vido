# Story: Polling & Session Optimization

Status: review

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

- [x] Task 1: Consolidate scanner polling into a single hook (AC: #1, #5)
  - [x] 1.1 `useScanProgress.ts`: Removed all polling (3s fallback, 10s idle poll) — SSE-only now
  - [x] 1.2 `useScanner.ts`: Already single source of truth for polling (3s/30s) — no changes needed
  - [x] 1.3 `ScannerSettings.tsx`: Already uses only `useScanner` hooks — verified, no changes needed
  - [x] 1.4 Verified: `useScanProgress` used only by scanner progress UI components, none poll

- [x] Task 2: Implement lazy auth in qBittorrent client (AC: #2, #3, #4)
  - [x] 2.1 `apps/api/internal/qbittorrent/client.go`: Replaced `Login()` with `ensureAuth()` in GetTorrents/GetTorrentDetails
  - [x] 2.2 `ensureAuth()` checks `lastLoginAt` timestamp, skips login if within 30min TTL
  - [x] 2.3 `doWithAuth()` method: retries request once on 401/403 after re-authenticating
  - [x] 2.4 `lastLoginAt` field + `defaultAuthTTL = 30min` constant added

- [x] Task 3: Tests (AC: all)
  - [x] 3.1 `useScanProgress.spec.ts`: Rewritten — 11 tests confirm SSE-only, no polling
  - [x] 3.2 `useScanner.ts`: Already tested via existing `useScanStatus` query tests
  - [x] 3.3 `client_test.go`: Added `TestClient_EnsureAuth_SkipsLoginWhenSessionFresh` — verifies Login() called once, not on subsequent calls
  - [x] 3.4 `client_test.go`: Added `TestClient_DoWithAuth_RetriesOn401` — verifies 401 triggers re-auth + retry

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

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Removed all polling from useScanProgress — SSE-only. useScanner remains single polling source (3s/30s).
- Task 2: Implemented lazy auth (ensureAuth + doWithAuth) in qBittorrent client. Login() only called when session expired (>30min) or on 401/403.
- Task 3: Rewrote useScanProgress tests (11 pass). Added 2 Go tests for lazy auth and 401 retry. 126 files / 1562 frontend tests pass. Go qbittorrent package passes.

### File List

- apps/web/src/hooks/useScanProgress.ts (modified — removed polling, SSE-only)
- apps/web/src/hooks/useScanProgress.spec.ts (rewritten — 11 SSE-only tests)
- apps/api/internal/qbittorrent/client.go (modified — ensureAuth, doWithAuth, lastLoginAt)
- apps/api/internal/qbittorrent/client_test.go (modified — 2 new lazy auth tests)
