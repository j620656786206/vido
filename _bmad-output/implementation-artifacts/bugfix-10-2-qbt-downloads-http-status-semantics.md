# Story: Bugfix 10.2 — qBT Downloads HTTP Status Code Semantics + Polling Gate

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a developer monitoring browser console / DevTools Network panel,
I want the `/api/v1/downloads*` endpoints to return semantically correct HTTP status codes when qBittorrent is not configured / auth-failed / unreachable, AND the frontend to stop polling those endpoints until qBT is configured,
so that an init-race during app startup does not burst the console with misleading "400 Bad Request" errors and so that downstream tooling (TanStack Query retry, Sentry, oncall dashboards) can rely on standard HTTP semantics.

## Acceptance Criteria

1. [@contract-v1] Given a qBittorrent connection error returned to a `download_handler.go` endpoint, when the error is mapped to an HTTP response, then it MUST follow this contract:

   | qBT Error Code                       | HTTP Status | Reason                                                        |
   | ------------------------------------ | ----------- | ------------------------------------------------------------- |
   | `QBITTORRENT_NOT_CONFIGURED`         | **503**     | Service Unavailable — feature requires upstream setup         |
   | `QBITTORRENT_AUTH_FAILED`            | **502**     | Bad Gateway — upstream rejected the proxied credentials       |
   | `QBITTORRENT_TIMEOUT`                | **504**     | Gateway Timeout — upstream did not respond in time            |
   | `QBITTORRENT_CONNECTION_FAILED` (default) | **502** | Bad Gateway — upstream unreachable                            |
   | `QBITTORRENT_TORRENT_NOT_FOUND`      | 404         | Not Found — unchanged (already correct in current code)       |

2. Given the three qBT-error switch blocks in `apps/api/internal/handlers/download_handler.go` (lines ~80-87 in `ListDownloads`, ~175-184 in `GetDownloadDetails`, ~209-218 in `GetDownloadCounts`), when this story lands, then ALL three blocks MUST use the same status-code mapping (defined once in a single helper — e.g. `qbtErrorToHTTPStatus(code string) (httpStatus int)` — to prevent drift). No call site may use `400` for the four qBT codes listed in AC #1.

3. Given the `503 QBITTORRENT_NOT_CONFIGURED` response, when the response body is serialized, then the `suggestion` field MUST end with the string `"SETUP_REQUIRED"` (exact substring) so the frontend can branch on it without parsing zh-TW copy. Existing zh-TW message ("請先設定 qBittorrent 連線。") MUST remain in `message` for user display.

4. Given Swagger annotations on `ListDownloads`, `GetDownloadCounts`, `GetDownloadDetails`, when re-generated via `swag init`, then `@Failure` lines MUST list `502`, `503`, `504` (in addition to existing `404`/`500`); the obsolete `@Failure 400 {object} APIResponse` MUST be removed for these three endpoints. Run `swag init` from `apps/api/` and commit the regenerated `docs/swagger.*` files (Rule 15).

5. Given the frontend `useDownloads(filter, sort, order, page, pageSize)` hook in `apps/web/src/hooks/useDownloads.ts`, when invoked anywhere in the app, then it MUST internally call `useQBittorrentConfig()` and pass `enabled: configData?.configured === true` to its `useQuery`. While `configured !== true` (including the loading state), the hook MUST NOT issue any network request to `/api/v1/downloads`.

6. Given the frontend `useDownloadCounts(enabled)` hook, when invoked, then it MUST AND its existing `enabled` parameter with `configData?.configured === true` so the gate is applied even when callers pass `enabled = true`. The existing `enabled` parameter contract is preserved for callers that want to additionally suppress polling.

7. Given app startup with qBittorrent unconfigured (`/api/v1/settings/qbittorrent` returns `{configured: false}`), when the user lands on the homepage, dashboard, or `/downloads` route, then the browser DevTools Network panel MUST show ZERO requests to `/api/v1/downloads`, `/api/v1/downloads/counts`, or `/api/v1/downloads/:hash` — the gate fully suppresses pre-config traffic. Verified by a co-located test that mocks `useQBittorrentConfig` and asserts `downloadService.getDownloads` was NOT called.

8. Given regression coverage, when the full test suite runs, then `nx test api` (existing handler tests) AND `nx test web` (existing hook tests) MUST stay green. New/updated tests MUST cover:
   - Backend: the 4 qBT error codes each return their mapped status code (2xx test cases per current pattern + 4 new cases × 3 endpoints = 12 assertions minimum, but use a table-driven approach so it's a single `t.Run` block per endpoint).
   - Frontend: `useDownloads` + `useDownloadCounts` do NOT fire when `useQBittorrentConfig().data?.configured !== true` (loading + false branches both covered).

9. Given the existing `code-review` Rule 7 wire-format check, when this story lands, then ZERO new error codes are introduced (existing `QBITTORRENT_*` codes are reused unchanged). Confirmed against [@contract-v0] (project-context.md Rule 7 prefix list — implicit v0 since the prefix list itself is not [@contract-vN]-stamped).

10. Given the optional out-of-scope items, when DEV evaluates them, then they MUST remain UNCHANGED unless explicitly elevated:
    - `qbittorrent_handler.go:129` (`TestConnection` returns 400) — intentional: the endpoint exists to test arbitrary user-supplied config, where 400 conveys "the test config failed" and the test failure IS the response payload. NOT updated by this story.
    - `qbittorrent_handler.go` `AddDownload`/`PauseDownload`/etc. mutation endpoints — out of scope (this story addresses GET endpoints first, where the init-race burst originates). If similar 400-on-qBT-error patterns exist there, file a follow-up.

## Tasks / Subtasks

- [ ] Task 1: Backend status-code mapping helper + handler updates (AC: #1, #2, #10)
  - [ ] 1.1 In `apps/api/internal/handlers/download_handler.go`, add unexported helper `qbtErrorToHTTPStatus(code string) int` returning `503` for `ErrCodeNotConfigured`, `502` for `ErrCodeAuthFailed` and `ErrCodeConnectionFailed` (default), `504` for `ErrCodeTimeout`. The helper lives in this file (single-package usage); do NOT promote to a shared util until a second handler needs it (YAGNI).
  - [ ] 1.2 Replace the three `ErrorResponse(c, 400, ...)` switch-arms in `ListDownloads` (~line 80-87), `GetDownloadDetails` (~line 175-184), `GetDownloadCounts` (~line 209-218) with calls that use `qbtErrorToHTTPStatus(connErr.Code)` for the status. Keep the `TorrentNotFound → NotFoundError(c, "torrent")` branch in `GetDownloadDetails` unchanged.
  - [ ] 1.3 Append `" SETUP_REQUIRED"` to the suggestion string for `ErrCodeNotConfigured` (e.g. `"請先設定 qBittorrent 連線。SETUP_REQUIRED"`) so frontend can branch programmatically without parsing zh-TW (AC #3). Other suggestion strings unchanged.
  - [ ] 1.4 Do NOT touch `qbittorrent_handler.go:TestConnection` or any mutation endpoint (AC #10).

- [ ] Task 2: Backend test updates (AC: #8 backend half)
  - [ ] 2.1 In `apps/api/internal/handlers/download_handler_test.go`, find the existing `connErr` test cases. Add a table-driven sub-test per endpoint (`ListDownloads`, `GetDownloadDetails`, `GetDownloadCounts`) iterating the 4 qBT error codes and asserting the expected HTTP status (per AC #1 table). Use a small fixture map `{code → expectedStatus}` so the table mirrors the contract.
  - [ ] 2.2 Assert `response.Body` contains `SETUP_REQUIRED` for the `NotConfigured` case (AC #3).
  - [ ] 2.3 Run `nx test api` from repo root (Rule 12) — must be GREEN.
  - [ ] 2.4 Run `pnpm lint:all` — must be GREEN.

- [ ] Task 3: Swagger regeneration (AC: #4)
  - [ ] 3.1 Update `@Failure` annotations on `ListDownloads`, `GetDownloadCounts`, `GetDownloadDetails`: remove `@Failure 400 {object} APIResponse`, add `@Failure 502 {object} APIResponse`, `@Failure 503 {object} APIResponse`, `@Failure 504 {object} APIResponse`.
  - [ ] 3.2 From `apps/api/`, run `swag init -g cmd/api/main.go --parseInternal --parseDependency` (or the project's standard swag invocation — check `package.json` script if any). Commit `apps/api/docs/swagger.json`, `apps/api/docs/swagger.yaml`, `apps/api/docs/docs.go` if regenerated.
  - [ ] 3.3 Verify the regenerated swagger renders correctly via `nx serve api` + visiting `http://localhost:8080/swagger/index.html` (manual smoke). If swag tool is not in DEV environment, document in Completion Notes and let CI handle verification.

- [ ] Task 4: Frontend hook gating (AC: #5, #6)
  - [ ] 4.1 In `apps/web/src/hooks/useDownloads.ts`, import `useQBittorrentConfig` from `./useQBittorrent`. Inside `useDownloads`, call `const { data: qbtConfig } = useQBittorrentConfig();` and OR `enabled: qbtConfig?.configured === true` into the `useQuery` options.
  - [ ] 4.2 In the same file, update `useDownloadCounts(enabled = true)` so the effective `enabled` is `enabled && qbtConfig?.configured === true`.
  - [ ] 4.3 In `useDownloadDetails(hash)`, ALSO gate on `qbtConfig?.configured === true` for consistency (existing `enabled: !!hash` becomes `enabled: !!hash && qbtConfig?.configured === true`).
  - [ ] 4.4 Document the gate with a single-line comment near each `enabled:` clause: `// bugfix-10-2: skip polling until qBT config check confirms configured; prevents init-race 503 burst`.

- [ ] Task 5: Frontend test updates (AC: #7, #8 frontend half)
  - [ ] 5.1 In `apps/web/src/hooks/useDownloads.spec.ts`, mock `useQBittorrentConfig` (via `vi.mock('./useQBittorrent', ...)` at top of file) so the existing tests still pass when the new gate is in place. The mock default should return `{ data: { configured: true }, isLoading: false }` to avoid breaking existing happy-path tests.
  - [ ] 5.2 Add new test: when `useQBittorrentConfig` mock returns `{ data: { configured: false } }`, then `mockGetDownloads` is called 0 times after a render + 200ms wait. Same for `mockGetDownloadCounts`, `mockGetDownloadDetails`.
  - [ ] 5.3 Add new test: when `useQBittorrentConfig` mock returns `{ data: undefined, isLoading: true }`, then `mockGetDownloads` is called 0 times.
  - [ ] 5.4 Run `nx test web` from repo root — must be GREEN.

- [ ] Task 6: Manual smoke + audit (AC: #7)
  - [ ] 6.1 With qBT unconfigured (or `/api/v1/settings/qbittorrent` mock-returning `{configured: false}`), open the dashboard. Verify in DevTools Network: ZERO `/api/v1/downloads*` requests fire.
  - [ ] 6.2 Configure qBT (or flip mock to `{configured: true}`). Verify polling resumes (5s cadence) at the next `useQBittorrentConfig` invalidation cycle.
  - [ ] 6.3 Trigger an actual qBT failure (e.g., stop the qBT container). Verify the response status in DevTools Network is `502` (not `400`), and the existing `DownloadPanel.DisconnectedState` UI still appears.
  - [ ] 6.4 Document the smoke results in Completion Notes (one-line each).

## Dev Notes

### Root Cause Analysis

Two independent issues compound into the user-visible "console burst on app start":

**(a) Wrong HTTP status codes (server-side bug):**
`download_handler.go` returns `400 Bad Request` for `QBITTORRENT_NOT_CONFIGURED`, `QBITTORRENT_AUTH_FAILED`, and connection/timeout errors. `400` semantically means "the client sent a malformed request" — but the request was perfectly fine; the failure is upstream. Standard HTTP semantics:

- `502 Bad Gateway` — proxy received a bad response from upstream
- `503 Service Unavailable` — service is temporarily unable to handle the request (often used for "feature not configured" or "in maintenance")
- `504 Gateway Timeout` — proxy timed out waiting for upstream

Using `400` makes downstream tooling (TanStack Query retry policies, Sentry filters, oncall alerting) misclassify the errors. It also confuses developers reading the Network tab.

**(b) Frontend polls before config check resolves (client-side bug):**
`useDownloads` and `useDownloadCounts` start polling on mount. `useQBittorrentConfig` (which reports `{configured: true|false}`) resolves a few hundred ms later. During that window the download endpoints fire and (for un-configured users) hit a 400 wall — bursting the console at every page load. `DownloadPanel` already gates its UI on `config.configured`, but the hook itself doesn't gate. Centralizing the gate inside the hooks is cleaner than asking every caller to remember.

### Why this is Epic 4 tech debt (not Epic 11 or Epic 10 regression)

- The `400` status was the original wire format from Story 4.x (qBT integration). It was never re-examined against HTTP semantics.
- The frontend init-race is amplified by Epic 10's homepage layout (multiple panels mount simultaneously) but the underlying race existed since Epic 4.
- Sprint-status comment confirmed: "Confirmed unrelated to Epic 11" — Epic 11 (advanced filters) does not touch downloads.

### Status Code Map (the contract)

| qBT Error Code                       | Current | New     | RFC 7231 §6 Justification                             |
| ------------------------------------ | ------- | ------- | ----------------------------------------------------- |
| `QBITTORRENT_NOT_CONFIGURED`         | 400     | **503** | Server is unable; setup needed. Frontend keys on `SETUP_REQUIRED` substring (AC #3) |
| `QBITTORRENT_AUTH_FAILED`            | 400     | **502** | Upstream rejected the proxied credentials             |
| `QBITTORRENT_TIMEOUT`                | 400     | **504** | Upstream timeout                                      |
| `QBITTORRENT_CONNECTION_FAILED` (default) | 400 | **502** | Upstream unreachable                                  |
| `QBITTORRENT_TORRENT_NOT_FOUND`      | 404     | 404     | Unchanged — already correct                           |

### Why centralize the gate inside the hooks (vs. consumer-driven)

Three call sites today (`DownloadPanel`, `/downloads` route × 2 hooks). All three want the same behavior. Putting the gate in the hook means:

1. Future callers can't forget — single source of truth.
2. Test surface shrinks: one `useQBittorrentConfig` mock, not three.
3. Reads as documentation: anyone grepping `useDownloads` sees the gate inline.

Trade-off: the hook now has a transitive dependency on `useQBittorrent`. Acceptable — both are in the same module, and `useQBittorrentConfig` is already widely consumed (`DownloadPanel`, `QBittorrentForm`, etc.).

### Architecture Compliance

- **Rule 1 (Single Backend):** All Go changes in `apps/api/`. No `/cmd` or root `/internal` touched.
- **Rule 3 (API Response Format):** Existing `ApiResponse<T>` envelope unchanged; only HTTP status codes change.
- **Rule 4 (Layered Architecture):** Handler-only change. Service layer (`DownloadService`) and `qbittorrent.ConnectionError` types unchanged.
- **Rule 5 (TanStack Query):** Frontend changes stay within TanStack Query's `enabled` contract — no Zustand additions.
- **Rule 7 (Error Codes):** ZERO new codes (AC #9). Existing `QBITTORRENT_*` codes reused. The Rule 7 prefix grep in `code-review/instructions.xml` Step 3 stays valid.
- **Rule 11 (Interface Location):** No interface changes.
- **Rule 12 (CI Lint):** Tasks 2.4 + 5.4 enforce `pnpm lint:all` + `nx test {api,web}`.
- **Rule 13 (Error Handling):** Backend errors continue to be propagated via `ConnectionError`; only the HTTP mapping changes. Frontend hook errors continue to be surfaced via `useQuery.error`.
- **Rule 15 (Pre-commit Self-verification):** Swagger MUST be regenerated (Task 3.2) — explicitly enforced by AC #4.
- **Rule 16 (Test Assertions):** Use `expect(mockGetDownloads).not.toHaveBeenCalled()` (specific) for the gate tests, not `toBeFalsy()`.
- **Rule 18 (Case Transformation):** No new fields cross the boundary.
- **Rule 20 (AC Contract Versioning):** AC #1 stamped `[@contract-v1]` because the status-code map IS the contract this story creates. Future error-handling features may want to reference it.

### Cross-Stack Split Check

- Backend tasks: 3 (Tasks 1, 2, 3)
- Frontend tasks: 3 (Tasks 4, 5, 6)
- Verdict: **Single story** — both sides ≤ 3 (split rule triggers when BOTH > 3).

### Key Files

| File | Change |
|------|--------|
| `apps/api/internal/handlers/download_handler.go` | Add `qbtErrorToHTTPStatus` helper; replace 3 switch arms (lines ~80, ~175, ~209); update Swagger `@Failure` annotations |
| `apps/api/internal/handlers/download_handler_test.go` | Add table-driven status-code tests + `SETUP_REQUIRED` substring assertion |
| `apps/api/docs/swagger.json` + `swagger.yaml` + `docs.go` | Regenerated by `swag init` |
| `apps/web/src/hooks/useDownloads.ts` | Gate `useDownloads`, `useDownloadCounts`, `useDownloadDetails` on `useQBittorrentConfig().data?.configured` |
| `apps/web/src/hooks/useDownloads.spec.ts` | Mock `useQBittorrentConfig`; add gate-coverage tests |
| `apps/api/internal/handlers/qbittorrent_handler.go` | NO change — out of scope (AC #10) |
| `apps/api/internal/qbittorrent/types.go` | NO change — error codes reused |
| `apps/api/internal/services/download_service.go` | NO change — service layer untouched |
| `apps/web/src/components/dashboard/DownloadPanel.tsx` | NO change — already gates UI on `config.configured`; the hook gate just makes the gate complete |
| `apps/web/src/routes/downloads.tsx` | NO change — hooks now self-gate |

### Backend implementation sketch

```go
// qbtErrorToHTTPStatus maps a qBittorrent ConnectionError.Code to the
// semantically correct HTTP status. Used by all three GET endpoints in
// download_handler.go to keep the contract single-sourced (bugfix-10-2).
func qbtErrorToHTTPStatus(code string) int {
    switch code {
    case qbittorrent.ErrCodeNotConfigured:
        return http.StatusServiceUnavailable // 503
    case qbittorrent.ErrCodeTimeout:
        return http.StatusGatewayTimeout // 504
    case qbittorrent.ErrCodeAuthFailed, qbittorrent.ErrCodeConnectionFailed:
        return http.StatusBadGateway // 502
    default:
        return http.StatusBadGateway // 502 — conservative for any future qBT code
    }
}
```

Each switch arm becomes:

```go
case qbittorrent.ErrCodeNotConfigured:
    ErrorResponse(c, qbtErrorToHTTPStatus(connErr.Code), connErr.Code,
        "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。SETUP_REQUIRED")
case qbittorrent.ErrCodeAuthFailed:
    ErrorResponse(c, qbtErrorToHTTPStatus(connErr.Code), connErr.Code,
        "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
default:
    ErrorResponse(c, qbtErrorToHTTPStatus(connErr.Code), connErr.Code,
        "無法連線到 qBittorrent", connErr.Error())
```

### Frontend implementation sketch

```typescript
import { useQBittorrentConfig } from './useQBittorrent';

export function useDownloads(...) {
  const isVisible = usePageVisibility();
  const { data: qbtConfig } = useQBittorrentConfig();
  const isConfigured = qbtConfig?.configured === true;

  return useQuery<PaginatedDownloads, Error>({
    queryKey: downloadKeys.list(...),
    queryFn: () => downloadService.getDownloads(...),
    // bugfix-10-2: skip polling until qBT config check confirms configured;
    // prevents init-race 503 burst at app start.
    enabled: isConfigured,
    refetchInterval: isVisible && isConfigured ? 5000 : false,
    refetchOnWindowFocus: true,
  });
}
```

### Test pattern (backend, table-driven)

```go
tests := []struct {
    name           string
    qbtCode        string
    expectedStatus int
}{
    {"not_configured", qbittorrent.ErrCodeNotConfigured, http.StatusServiceUnavailable},
    {"auth_failed",    qbittorrent.ErrCodeAuthFailed,    http.StatusBadGateway},
    {"timeout",        qbittorrent.ErrCodeTimeout,       http.StatusGatewayTimeout},
    {"conn_failed",    qbittorrent.ErrCodeConnectionFailed, http.StatusBadGateway},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // arrange mock service to return ConnectionError with tt.qbtCode
        // act: call handler
        // assert: w.Code == tt.expectedStatus
    })
}
```

### Project Structure Notes

- Helper `qbtErrorToHTTPStatus` lives in `download_handler.go` (single-package use). If a second handler later needs the same mapping, promote it to a small package like `apps/api/internal/qbittorrent/httpstatus.go`. Until then, YAGNI.
- No new files. No new packages. No migration needed.

### References

- [Source: project-context.md Rule 7 — `QBITTORRENT_*` prefix authoritative list]
- [Source: project-context.md Rule 12 — `pnpm lint:all` mirrors CI]
- [Source: project-context.md Rule 15 — Swagger must be regenerated]
- [Source: project-context.md Rule 20 — AC contract versioning]
- [Source: apps/api/internal/handlers/download_handler.go:80-87, 175-184, 209-218 — three switch sites]
- [Source: apps/api/internal/qbittorrent/types.go:42-47 — error code constants]
- [Source: apps/web/src/hooks/useDownloads.ts:44-86 — three hooks to gate]
- [Source: apps/web/src/hooks/useQBittorrent.ts:21-27 — gate signal source]
- [Source: apps/web/src/components/dashboard/DownloadPanel.tsx:24,29 — existing UI gate, complementary]
- [Source: RFC 7231 §6.6 — 5xx status code semantics]

### Change Log

| Date       | Change |
|------------|--------|
| 2026-05-04 | [@contract-v1] AC #1: New status-code contract for `download_handler.go` qBT errors — `400` retired in favor of `502`/`503`/`504`. Downstream consumers (TanStack Query retry, Sentry classifiers, oncall dashboards) MUST be updated if they currently key on `400` for these endpoints. AC #3 introduces the `SETUP_REQUIRED` substring marker for programmatic frontend branching. |

## Dev Agent Record

### Agent Model Used

(to be filled by dev-story)

### Debug Log References

(to be filled by dev-story)

### Completion Notes List

(to be filled by dev-story — must include "🔗 AC Drift: NONE|FOUND|N/A" per retro-10-AI2)

### File List

(to be filled by dev-story)
