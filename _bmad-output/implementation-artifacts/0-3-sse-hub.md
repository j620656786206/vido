# Story 0-3: SSE Hub

## Status: done

## Story
As a developer, I need a global Server-Sent Events (SSE) hub so that Epic 7 (Scanner) and Epic 8 (Subtitle Engine) can push real-time progress events to the frontend.

## Acceptance Criteria
- [x] SSE hub package at `apps/api/internal/sse/`
- [x] Hub supports Register/Unregister/Broadcast/Close operations
- [x] Client channels buffered (capacity 100) with non-blocking send
- [x] Event types defined: scan_progress, subtitle_progress, notification
- [x] HTTP handler at `GET /api/v1/events` with proper SSE headers
- [x] 30-second keepalive pings to prevent connection timeout
- [x] Wired into main.go with defer cleanup
- [x] Unit tests pass (7/7)
- [x] `go build` passes

## Tasks
- [x] Task 1: Create `sse/hub.go` with Hub, Client, Event types
- [x] Task 2: Implement Hub.Run() select loop with register/unregister/broadcast
- [x] Task 3: Create `sse/handler.go` with Gin SSE handler
- [x] Task 4: Wire SSE hub into `cmd/api/main.go`
- [x] Task 5: Write unit tests in `sse/hub_test.go`

## Dev Agent Record

### Completion Notes
- Uses `github.com/google/uuid` for client IDs (already in go.mod)
- Uses `log/slog` for logging per project rules
- Non-blocking sends — drops events if client buffer full (with warning log)
- Handler uses `c.Stream()` with `c.SSEvent()` for proper SSE wire format
- `X-Accel-Buffering: no` header added for nginx compatibility
- 7 tests: RegisterUnregister, Broadcast, MultipleClients, UnregisterCloses, NonBlockingSend, ConcurrentBroadcast, ClientCount, Close

### File List
| Action | File |
|--------|------|
| CREATE | `apps/api/internal/sse/hub.go` |
| CREATE | `apps/api/internal/sse/handler.go` |
| CREATE | `apps/api/internal/sse/hub_test.go` |
| MODIFY | `apps/api/cmd/api/main.go` |

### Change Log
- 2026-03-23: Implemented SSE hub + handler + tests. All 7 tests pass. Build passes.
