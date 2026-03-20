# Story 6.4: Service Connection Status Dashboard

Status: review

## Story

As a **system administrator**,
I want to **see connection status for all external services**,
So that **I can identify integration issues at a glance**.

## Acceptance Criteria

1. **Given** the user opens Settings > Status, **When** the status page loads, **Then** it shows connection status for: qBittorrent (Connected/Disconnected), TMDb API (Available/Rate Limited/Error), AI Service (Available/Error)
2. **Given** a service shows an error, **When** hovering or clicking on the status, **Then** detailed error message is shown and last successful connection time is displayed
3. **Given** the status page is open, **When** service status changes, **Then** the status updates in real-time and a notification indicates the change

## Tasks / Subtasks

- [x] Task 1: Extend Health Check Service (AC: 1)
  - [x] 1.1: Extend `/apps/api/internal/health/checker.go` to include ALL service checks (qBittorrent, TMDb, AI)
  - [x] 1.2: Add `GetAllServiceStatuses(ctx) ([]ServiceStatus, error)` method
  - [x] 1.3: Track last successful connection time per service
  - [x] 1.4: Add rate-limit detection for TMDb (check response headers)
  - [x] 1.5: Write unit tests (≥80% coverage)

- [x] Task 2: Create Service Status Endpoints (AC: 1, 2)
  - [x] 2.1: Create `/apps/api/internal/handlers/status_handler.go`
  - [x] 2.2: `GET /api/v1/settings/services` → returns all service connection statuses
  - [x] 2.3: `POST /api/v1/settings/services/:name/test` → manually test specific service connection
  - [x] 2.4: Write handler tests (≥70% coverage)

- [x] Task 3: Implement Real-time Status Updates (AC: 3)
  - [x] 3.1: Use polling approach: frontend polls every 30 seconds (SSE/WebSocket is over-engineering for this)
  - [x] 3.2: Backend health monitor already runs periodically - expose latest results

- [x] Task 4: Create Status Dashboard UI (AC: 1, 2, 3)
  - [x] 4.1: Create `/apps/web/src/components/settings/ServiceStatusDashboard.tsx` - main status view
  - [x] 4.2: Create `/apps/web/src/components/settings/ServiceStatusCard.tsx` - individual service card
  - [x] 4.3: Implement status indicators: 🟢 Connected, 🟡 Rate Limited, 🔴 Error/Disconnected
  - [x] 4.4: Show detail panel on click/hover with error message and last success time
  - [x] 4.5: Add "Test Connection" button per service

- [x] Task 5: Create API Client & Hooks (AC: all)
  - [x] 5.1: Add service status methods to `settingsService.ts`
  - [x] 5.2: Create `/apps/web/src/hooks/useServiceStatus.ts` with `refetchInterval: 30000`

- [x] Task 6: Wire Up (AC: all)
  - [x] 6.1: Register status handler in `main.go`
  - [x] 6.2: Write component tests

## Dev Notes

### Architecture Requirements

**FR55: Display service connection status**
**NFR-M13:** System health status visible (service connection status)
**ARCH-8:** Health Check Scheduler - monitor qBittorrent, TMDb, AI APIs
**NFR-I2:** Connection health monitoring must detect failures within <10 seconds

### Existing Codebase Context

**Health module already exists:** `/apps/api/internal/health/` has `checker.go` and `monitor.go`. The health checker already runs periodic checks. Extend it to track more detailed status per service.

**Health endpoint exists:** The `/api/v1/health/services` endpoint was implemented in Story 3-12. Build on top of this.

### Service Status Model

```go
type ServiceStatus struct {
    Name               string     `json:"name"`            // "qbittorrent", "tmdb", "ai"
    DisplayName        string     `json:"displayName"`     // "qBittorrent", "TMDb API", "AI 服務"
    Status             string     `json:"status"`          // "connected", "rate_limited", "error", "disconnected", "unconfigured"
    Message            string     `json:"message"`         // Detailed status message
    LastSuccessAt      *time.Time `json:"lastSuccessAt"`   // Last successful check
    LastCheckAt        time.Time  `json:"lastCheckAt"`     // Last check attempt
    ResponseTimeMs     int64      `json:"responseTimeMs"`  // Latest response time
    ErrorMessage       string     `json:"errorMessage,omitempty"`
}
```

### Status Color Mapping (Frontend)

```tsx
const statusConfig = {
  connected:    { color: 'text-green-400', bg: 'bg-green-400/10', icon: CheckCircle, label: '已連線' },
  rate_limited: { color: 'text-yellow-400', bg: 'bg-yellow-400/10', icon: AlertTriangle, label: '速率限制' },
  error:        { color: 'text-red-400', bg: 'bg-red-400/10', icon: XCircle, label: '錯誤' },
  disconnected: { color: 'text-red-400', bg: 'bg-red-400/10', icon: WifiOff, label: '已斷線' },
  unconfigured: { color: 'text-gray-400', bg: 'bg-gray-400/10', icon: Settings, label: '未設定' },
};
```

### API Response Format

```json
// GET /api/v1/settings/services
{
  "success": true,
  "data": {
    "services": [
      {
        "name": "qbittorrent",
        "displayName": "qBittorrent",
        "status": "connected",
        "message": "已連線，版本 4.6.1",
        "lastSuccessAt": "2026-02-10T14:30:00Z",
        "lastCheckAt": "2026-02-10T14:30:00Z",
        "responseTimeMs": 45
      },
      {
        "name": "tmdb",
        "displayName": "TMDb API",
        "status": "rate_limited",
        "message": "速率限制中，35/40 請求已用",
        "lastSuccessAt": "2026-02-10T14:29:55Z",
        "lastCheckAt": "2026-02-10T14:30:00Z",
        "responseTimeMs": 230
      }
    ]
  }
}
```

### Error Codes

- `SERVICE_CHECK_FAILED` - Failed to check service status
- `SERVICE_NOT_FOUND` - Unknown service name

### Project Structure Notes

```
/apps/api/internal/handlers/
├── status_handler.go
└── status_handler_test.go

/apps/web/src/components/settings/
├── ServiceStatusDashboard.tsx
├── ServiceStatusDashboard.spec.tsx
├── ServiceStatusCard.tsx
└── ServiceStatusCard.spec.tsx
```

### Dependencies

- Story 3-12 (Graceful Degradation) - health check infrastructure
- `/apps/api/internal/health/` module

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.4]
- [Source: _bmad-output/planning-artifacts/prd.md#FR55]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-M13]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-8-Health-Check]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Extended `ServiceHealth` model with `ResponseTimeMs` field and `SetResponseTime()` method. Added `ServiceStatus` model with status mapping (connected/rate_limited/error/disconnected/unconfigured). Added `ToServiceStatus()` conversion with rate-limit detection (`isRateLimitError`) and unconfigured detection (`isUnconfiguredError`). Added `GetAllServiceStatuses()` to `HealthMonitor` with response time tracking during `CheckAllServices()` and `StartQBMonitoring()`. 14 model tests + 4 monitor tests pass.
- Task 2: Created `ServiceStatusService` (service layer) wrapping `HealthMonitor` and `HealthChecker`. Provides `GetAllStatuses()` and `TestService()` with response time measurement. Created `StatusHandler` with `GET /api/v1/settings/services` and `POST /api/v1/settings/services/:name/test`. Sentinel error `ErrServiceNotFound` for unknown service names. 7 service tests + 6 handler tests pass.
- Task 3: Frontend polling implemented via TanStack Query `refetchInterval: 30000` with page visibility detection to pause when tab is hidden.
- Task 4: Created `ServiceStatusDashboard.tsx` (main view with loading/error/empty states) and `ServiceStatusCard.tsx` (individual card with status icon, response time, expandable detail panel, test button). Status indicators use story-specified color scheme.
- Task 5: Created `serviceStatusService.ts` (API client) and `useServiceStatus.ts` hook (query + mutation with cache invalidation).
- Task 6: Registered `StatusHandler` in `main.go` with route ordering before settingsHandler. 13 component tests pass (6 dashboard + 7 card).
- 🎨 UX Verification: SKIPPED — no UX design screenshots for settings status page

### Change Log

- 2026-03-18: Implemented Story 6-4 Service Connection Status Dashboard — full backend (model, service, handler) + frontend (service, hook, dashboard UI, card component) with 44 new tests
- 2026-03-20: Code review fixes — fetchApi null guard, test error display, concurrent click prevention. TA expanded 16 tests + CR added 3 more tests.

### Review Follow-ups

- [ ] [AI-Review][HIGH] AC3 partial: Status change notification not implemented — polling updates data but no toast/notification when service status transitions (e.g. connected→error). Requires UX design decision before implementation.

### File List

- apps/api/internal/models/degradation.go (modified — added ServiceStatus model, ResponseTimeMs, status conversion)
- apps/api/internal/models/service_status_test.go (new — 14 tests)
- apps/api/internal/health/monitor.go (modified — response time tracking, GetAllServiceStatuses)
- apps/api/internal/health/monitor_test.go (modified — 4 new tests)
- apps/api/internal/services/service_status_service.go (new — service layer)
- apps/api/internal/services/service_status_service_test.go (new — 7 tests)
- apps/api/internal/handlers/status_handler.go (new — HTTP handler)
- apps/api/internal/handlers/status_handler_test.go (new — 6 tests)
- apps/api/cmd/api/main.go (modified — wired StatusHandler)
- apps/web/src/services/serviceStatusService.ts (modified — API client, added null guard for data field)
- apps/web/src/services/serviceStatusService.spec.ts (new — 6 tests)
- apps/web/src/hooks/useServiceStatus.ts (new — TanStack Query hooks)
- apps/web/src/components/settings/ServiceStatusDashboard.tsx (modified — added test error display, concurrent click guard)
- apps/web/src/components/settings/ServiceStatusDashboard.spec.tsx (modified — 13 tests)
- apps/web/src/components/settings/ServiceStatusCard.tsx (new — service card)
- apps/web/src/components/settings/ServiceStatusCard.spec.tsx (modified — 13 tests)
- apps/web/src/routes/settings/status.tsx (modified — replaced placeholder with dashboard)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified — status updated)
