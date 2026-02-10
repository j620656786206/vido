# Story 6.4: Service Connection Status Dashboard

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **see connection status for all external services**,
So that **I can identify integration issues at a glance**.

## Acceptance Criteria

1. **Given** the user opens Settings > Status, **When** the status page loads, **Then** it shows connection status for: qBittorrent (Connected/Disconnected), TMDb API (Available/Rate Limited/Error), AI Service (Available/Error)
2. **Given** a service shows an error, **When** hovering or clicking on the status, **Then** detailed error message is shown and last successful connection time is displayed
3. **Given** the status page is open, **When** service status changes, **Then** the status updates in real-time and a notification indicates the change

## Tasks / Subtasks

- [ ] Task 1: Extend Health Check Service (AC: 1)
  - [ ] 1.1: Extend `/apps/api/internal/health/checker.go` to include ALL service checks (qBittorrent, TMDb, AI)
  - [ ] 1.2: Add `GetAllServiceStatuses(ctx) ([]ServiceStatus, error)` method
  - [ ] 1.3: Track last successful connection time per service
  - [ ] 1.4: Add rate-limit detection for TMDb (check response headers)
  - [ ] 1.5: Write unit tests (≥80% coverage)

- [ ] Task 2: Create Service Status Endpoints (AC: 1, 2)
  - [ ] 2.1: Create `/apps/api/internal/handlers/status_handler.go`
  - [ ] 2.2: `GET /api/v1/settings/services` → returns all service connection statuses
  - [ ] 2.3: `POST /api/v1/settings/services/:name/test` → manually test specific service connection
  - [ ] 2.4: Write handler tests (≥70% coverage)

- [ ] Task 3: Implement Real-time Status Updates (AC: 3)
  - [ ] 3.1: Use polling approach: frontend polls every 30 seconds (SSE/WebSocket is over-engineering for this)
  - [ ] 3.2: Backend health monitor already runs periodically - expose latest results

- [ ] Task 4: Create Status Dashboard UI (AC: 1, 2, 3)
  - [ ] 4.1: Create `/apps/web/src/components/settings/ServiceStatusDashboard.tsx` - main status view
  - [ ] 4.2: Create `/apps/web/src/components/settings/ServiceStatusCard.tsx` - individual service card
  - [ ] 4.3: Implement status indicators: 🟢 Connected, 🟡 Rate Limited, 🔴 Error/Disconnected
  - [ ] 4.4: Show detail panel on click/hover with error message and last success time
  - [ ] 4.5: Add "Test Connection" button per service

- [ ] Task 5: Create API Client & Hooks (AC: all)
  - [ ] 5.1: Add service status methods to `settingsService.ts`
  - [ ] 5.2: Create `/apps/web/src/hooks/useServiceStatus.ts` with `refetchInterval: 30000`

- [ ] Task 6: Wire Up (AC: all)
  - [ ] 6.1: Register status handler in `main.go`
  - [ ] 6.2: Write component tests

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
