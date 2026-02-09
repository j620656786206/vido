# Story 4.6: Connection Health Monitoring

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **see qBittorrent connection health status**,
So that **I know immediately when there are connectivity issues**.

## Acceptance Criteria

1. **AC1: Status Indicator Display**
   - Given qBittorrent is connected
   - When viewing the dashboard header
   - Then a status indicator shows: 🟢 Connected

2. **AC2: Disconnection Detection**
   - Given qBittorrent becomes unreachable
   - When the health check fails
   - Then the indicator changes to: 🔴 Disconnected
   - And shows: "Last success: 2 minutes ago"

3. **AC3: Auto-Recovery**
   - Given connection is lost
   - When automatic recovery is attempted
   - Then the system retries every 30 seconds (NFR-R6)
   - And reconnects automatically when available

4. **AC4: Connection Details**
   - Given the user clicks on the connection status
   - When viewing details
   - Then they see connection history and error logs

5. **AC5: Integration with Existing Health System**
   - Given the health monitoring system exists (Story 3-12)
   - When qBittorrent health is checked
   - Then it integrates with the existing service health monitor
   - And appears in the `/api/v1/health/services` response

## Tasks / Subtasks

- [ ] Task 1: Extend Health Monitor for qBittorrent (AC: 1, 2, 3, 5)
  - [ ] 1.1: Add qBittorrent to HealthChecker interface
  - [ ] 1.2: Implement `CheckQBittorrent(ctx) error` in health checker
  - [ ] 1.3: Add qBittorrent to monitored services list
  - [ ] 1.4: Configure 30-second retry interval (NFR-R6)
  - [ ] 1.5: Write health check tests

- [ ] Task 2: Create qBittorrent Health Check (AC: 1, 2, 3)
  - [ ] 2.1: Add health check endpoint call to qBittorrent client
  - [ ] 2.2: Implement `Ping(ctx) error` method on client
  - [ ] 2.3: Handle authentication refresh if needed
  - [ ] 2.4: Track last successful connection time
  - [ ] 2.5: Write client tests

- [ ] Task 3: Create Connection History Repository (AC: 4)
  - [ ] 3.1: Create `/apps/api/internal/repository/connection_history_repository.go`
  - [ ] 3.2: Add migration for `connection_history` table
  - [ ] 3.3: Implement `RecordEvent(ctx, event) error`
  - [ ] 3.4: Implement `GetHistory(ctx, service, limit) ([]ConnectionEvent, error)`
  - [ ] 3.5: Write repository tests

- [ ] Task 4: Create Connection History Types (AC: 4)
  - [ ] 4.1: Create `/apps/api/internal/models/connection_event.go`
  - [ ] 4.2: Define `ConnectionEvent` struct (ID, Service, Status, Message, Timestamp)
  - [ ] 4.3: Define `ConnectionEventType` enum (connected, disconnected, error, recovered)
  - [ ] 4.4: Write type tests

- [ ] Task 5: Create Connection History Handler (AC: 4)
  - [ ] 5.1: Create `GET /api/v1/health/services/qbittorrent/history` endpoint
  - [ ] 5.2: Return recent connection events
  - [ ] 5.3: Add Swagger documentation
  - [ ] 5.4: Write handler tests

- [ ] Task 6: Update Health Services Response (AC: 5)
  - [ ] 6.1: Add qBittorrent to `/api/v1/health/services` response
  - [ ] 6.2: Include last success time
  - [ ] 6.3: Include error count
  - [ ] 6.4: Update Swagger documentation

- [ ] Task 7: Create Connection Status Indicator Component (AC: 1, 2)
  - [ ] 7.1: Create `/apps/web/src/components/health/QBStatusIndicator.tsx`
  - [ ] 7.2: Show green/yellow/red indicator based on status
  - [ ] 7.3: Show tooltip with details on hover
  - [ ] 7.4: Show "Last success: X ago" when disconnected
  - [ ] 7.5: Write component tests

- [ ] Task 8: Create Connection History Modal (AC: 4)
  - [ ] 8.1: Create `/apps/web/src/components/health/ConnectionHistoryModal.tsx`
  - [ ] 8.2: Show history list with timestamps
  - [ ] 8.3: Color-code events by type
  - [ ] 8.4: Allow filtering by event type
  - [ ] 8.5: Write component tests

- [ ] Task 9: Add Status Indicator to Header (AC: 1, 2)
  - [ ] 9.1: Add QBStatusIndicator to main layout header
  - [ ] 9.2: Click opens ConnectionHistoryModal
  - [ ] 9.3: Poll status at regular intervals

- [ ] Task 10: Create Connection Health Hook (AC: 1, 2, 3)
  - [ ] 10.1: Create `/apps/web/src/hooks/useQBConnectionHealth.ts`
  - [ ] 10.2: Poll health status every 30 seconds
  - [ ] 10.3: Expose connected, lastSuccess, errorCount

- [ ] Task 11: E2E Tests (AC: all)
  - [ ] 11.1: Create `/e2e/connection-health.spec.ts`
  - [ ] 11.2: Test status indicator display
  - [ ] 11.3: Test disconnection detection
  - [ ] 11.4: Test history modal
  - [ ] 11.5: Test auto-recovery behavior

## Dev Notes

### Architecture Requirements

**FR33: Display connection health status**
- Visual indicator
- Connection history

**NFR-R6: Auto-recover from qBittorrent failures (30s reconnection)**
- 30-second retry interval
- Automatic reconnection

**ARCH-8: Health Check Scheduler**
- Periodic health checks
- Event logging

### Database Schema

```sql
-- Migration: create_connection_history_table.sql
CREATE TABLE IF NOT EXISTS connection_history (
    id TEXT PRIMARY KEY,
    service TEXT NOT NULL, -- 'qbittorrent', 'tmdb', 'ai', etc.
    event_type TEXT NOT NULL, -- 'connected', 'disconnected', 'error', 'recovered'
    status TEXT NOT NULL, -- 'healthy', 'degraded', 'down'
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_connection_history_service ON connection_history(service);
CREATE INDEX idx_connection_history_created_at ON connection_history(created_at);
```

### Backend Implementation

```go
// /apps/api/internal/models/connection_event.go
package models

import "time"

type ConnectionEventType string

const (
    EventConnected    ConnectionEventType = "connected"
    EventDisconnected ConnectionEventType = "disconnected"
    EventError        ConnectionEventType = "error"
    EventRecovered    ConnectionEventType = "recovered"
)

type ConnectionEvent struct {
    ID        string              `json:"id"`
    Service   string              `json:"service"`
    EventType ConnectionEventType `json:"eventType"`
    Status    string              `json:"status"` // healthy, degraded, down
    Message   string              `json:"message,omitempty"`
    CreatedAt time.Time           `json:"createdAt"`
}
```

```go
// /apps/api/internal/qbittorrent/client.go - Add Ping method

// Ping checks if qBittorrent is reachable and authenticated
func (c *Client) Ping(ctx context.Context) error {
    // Try to get app version as a lightweight health check
    _, err := c.getAppVersion(ctx)
    if err != nil {
        // Try to re-authenticate
        if authErr := c.Login(ctx); authErr != nil {
            return fmt.Errorf("qBittorrent unreachable: %w", authErr)
        }
        // Retry version check
        _, err = c.getAppVersion(ctx)
    }
    return err
}
```

```go
// /apps/api/internal/health/checker.go - Extend for qBittorrent

type HealthChecker interface {
    CheckTMDb(ctx context.Context) error
    CheckDouban(ctx context.Context) error
    CheckWikipedia(ctx context.Context) error
    CheckAI(ctx context.Context) error
    CheckQBittorrent(ctx context.Context) error // NEW
}

type HealthCheckerImpl struct {
    tmdbClient     *tmdb.Client
    aiService      AIServiceInterface
    qbService      QBittorrentServiceInterface
    historyRepo    repository.ConnectionHistoryRepositoryInterface
    logger         *slog.Logger
}

func (c *HealthCheckerImpl) CheckQBittorrent(ctx context.Context) error {
    config, err := c.qbService.GetConfig(ctx)
    if err != nil {
        return err
    }

    if config.Host == "" {
        return fmt.Errorf("qBittorrent not configured")
    }

    client := qbittorrent.NewClient(config, c.logger)
    return client.Ping(ctx)
}
```

```go
// /apps/api/internal/health/monitor.go - Add qBittorrent monitoring

const QBittorrentService = "qbittorrent"

func NewHealthMonitor(checker HealthChecker, historyRepo repository.ConnectionHistoryRepositoryInterface, logger *slog.Logger) *HealthMonitor {
    m := &HealthMonitor{
        services: make(map[string]*ServiceHealth),
        checker:  checker,
        historyRepo: historyRepo,
        logger:   logger,
    }

    // Initialize all services
    services := []string{"tmdb", "douban", "wikipedia", "ai", QBittorrentService}
    for _, svc := range services {
        m.services[svc] = &ServiceHealth{
            Name:      svc,
            Status:    "unknown",
            LastCheck: time.Time{},
        }
    }

    return m
}

func (m *HealthMonitor) checkAllServices(ctx context.Context) {
    checks := []struct {
        name    string
        checker func(context.Context) error
    }{
        {"tmdb", m.checker.CheckTMDb},
        {"douban", m.checker.CheckDouban},
        {"wikipedia", m.checker.CheckWikipedia},
        {"ai", m.checker.CheckAI},
        {QBittorrentService, m.checker.CheckQBittorrent}, // Added
    }

    for _, check := range checks {
        go func(name string, fn func(context.Context) error) {
            err := fn(ctx)
            m.updateServiceHealth(ctx, name, err)
        }(check.name, check.checker)
    }
}

func (m *HealthMonitor) updateServiceHealth(ctx context.Context, name string, err error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    svc := m.services[name]
    previousStatus := svc.Status
    svc.LastCheck = time.Now()

    if err == nil {
        svc.Status = "healthy"
        svc.LastSuccess = time.Now()
        svc.ErrorCount = 0
        svc.Message = ""
    } else {
        svc.ErrorCount++

        if svc.ErrorCount >= 3 {
            svc.Status = "down"
        } else {
            svc.Status = "degraded"
        }

        svc.Message = err.Error()
    }

    // Record status change events
    if previousStatus != svc.Status && previousStatus != "unknown" {
        var eventType models.ConnectionEventType
        if svc.Status == "healthy" {
            eventType = models.EventRecovered
        } else if svc.Status == "down" {
            eventType = models.EventDisconnected
        } else {
            eventType = models.EventError
        }

        event := &models.ConnectionEvent{
            ID:        uuid.New().String(),
            Service:   name,
            EventType: eventType,
            Status:    svc.Status,
            Message:   svc.Message,
            CreatedAt: time.Now(),
        }

        if err := m.historyRepo.Create(ctx, event); err != nil {
            m.logger.Error("Failed to record connection event",
                "service", name,
                "error", err,
            )
        }
    }
}

// StartQBMonitoring starts a dedicated monitor for qBittorrent with 30s interval
func (m *HealthMonitor) StartQBMonitoring(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second) // NFR-R6
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            err := m.checker.CheckQBittorrent(ctx)
            m.updateServiceHealth(ctx, QBittorrentService, err)
        }
    }
}
```

### API Response Format

**Health Services (Extended):**
```
GET /api/v1/health/services
```
Response:
```json
{
  "success": true,
  "data": {
    "degradationLevel": "partial",
    "services": {
      "tmdb": {
        "name": "TMDb API",
        "status": "healthy",
        "lastCheck": "2026-02-09T12:00:00Z",
        "lastSuccess": "2026-02-09T12:00:00Z"
      },
      "qbittorrent": {
        "name": "qBittorrent",
        "status": "down",
        "lastCheck": "2026-02-09T12:00:00Z",
        "lastSuccess": "2026-02-09T11:55:00Z",
        "errorCount": 4,
        "message": "connection refused"
      }
    }
  }
}
```

**Connection History:**
```
GET /api/v1/health/services/qbittorrent/history?limit=20
```
Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "event-1",
      "service": "qbittorrent",
      "eventType": "disconnected",
      "status": "down",
      "message": "connection refused",
      "createdAt": "2026-02-09T11:55:30Z"
    },
    {
      "id": "event-2",
      "service": "qbittorrent",
      "eventType": "connected",
      "status": "healthy",
      "message": "",
      "createdAt": "2026-02-09T11:00:00Z"
    }
  ]
}
```

### Frontend Implementation

```tsx
// /apps/web/src/components/health/QBStatusIndicator.tsx
interface QBStatusIndicatorProps {
  onClick?: () => void;
}

export function QBStatusIndicator({ onClick }: QBStatusIndicatorProps) {
  const { data: health, isLoading } = useQBConnectionHealth();

  const statusConfig = {
    healthy: {
      color: 'bg-green-500',
      label: 'qBittorrent 已連線',
      icon: CheckCircle,
    },
    degraded: {
      color: 'bg-yellow-500',
      label: 'qBittorrent 連線不穩定',
      icon: AlertCircle,
    },
    down: {
      color: 'bg-red-500',
      label: 'qBittorrent 未連線',
      icon: XCircle,
    },
    unknown: {
      color: 'bg-gray-400',
      label: 'qBittorrent 狀態未知',
      icon: HelpCircle,
    },
  };

  const status = health?.status || 'unknown';
  const config = statusConfig[status];
  const Icon = config.icon;

  const formatLastSuccess = (lastSuccess?: string) => {
    if (!lastSuccess) return '';
    const date = new Date(lastSuccess);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return '剛剛';
    if (diffMins < 60) return `${diffMins} 分鐘前`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours} 小時前`;
    return `${Math.floor(diffHours / 24)} 天前`;
  };

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={onClick}
            className="flex items-center gap-2 px-2 py-1 rounded-md hover:bg-muted transition-colors"
          >
            <div className={cn("w-2 h-2 rounded-full", config.color)} />
            <Icon className="h-4 w-4 text-muted-foreground" />
          </button>
        </TooltipTrigger>
        <TooltipContent>
          <p className="font-medium">{config.label}</p>
          {status === 'down' && health?.lastSuccess && (
            <p className="text-xs text-muted-foreground">
              上次連線：{formatLastSuccess(health.lastSuccess)}
            </p>
          )}
          {health?.message && (
            <p className="text-xs text-destructive">{health.message}</p>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
```

```tsx
// /apps/web/src/components/health/ConnectionHistoryModal.tsx
interface ConnectionHistoryModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ConnectionHistoryModal({ open, onOpenChange }: ConnectionHistoryModalProps) {
  const { data: history, isLoading } = useQuery({
    queryKey: ['health', 'qbittorrent', 'history'],
    queryFn: () => healthService.getQBHistory(),
    enabled: open,
  });

  const eventTypeConfig: Record<ConnectionEventType, { label: string; icon: React.ComponentType; color: string }> = {
    connected: { label: '已連線', icon: CheckCircle, color: 'text-green-500' },
    disconnected: { label: '已斷線', icon: XCircle, color: 'text-red-500' },
    error: { label: '錯誤', icon: AlertCircle, color: 'text-yellow-500' },
    recovered: { label: '已恢復', icon: RefreshCw, color: 'text-green-500' },
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>qBittorrent 連線記錄</DialogTitle>
          <DialogDescription>
            顯示最近 20 筆連線狀態變更記錄
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : history?.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              沒有連線記錄
            </div>
          ) : (
            <div className="space-y-2">
              {history?.map((event) => {
                const config = eventTypeConfig[event.eventType];
                const Icon = config.icon;

                return (
                  <div
                    key={event.id}
                    className="flex items-start gap-3 p-2 rounded-lg hover:bg-muted"
                  >
                    <Icon className={cn("h-4 w-4 mt-0.5", config.color)} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">{config.label}</p>
                      {event.message && (
                        <p className="text-xs text-muted-foreground truncate">
                          {event.message}
                        </p>
                      )}
                    </div>
                    <time className="text-xs text-muted-foreground">
                      {formatRelativeTime(event.createdAt)}
                    </time>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
```

```tsx
// /apps/web/src/hooks/useQBConnectionHealth.ts
export function useQBConnectionHealth() {
  return useQuery({
    queryKey: ['health', 'services', 'qbittorrent'],
    queryFn: async () => {
      const response = await healthService.getServiceHealth();
      return response.services.qbittorrent;
    },
    refetchInterval: 30000, // NFR-R6: 30 second polling
    staleTime: 25000,
  });
}
```

### Project Structure Notes

**Backend Files to Create/Modify:**
```
/apps/api/internal/models/
└── connection_event.go

/apps/api/internal/repository/
├── connection_history_repository.go
└── connection_history_repository_test.go

/apps/api/internal/health/
├── checker.go (modify - add qBittorrent)
├── monitor.go (modify - add qBittorrent, history)
└── monitor_test.go (update tests)

/apps/api/internal/qbittorrent/
├── client.go (modify - add Ping)
└── client_test.go (update tests)

/apps/api/internal/handlers/
└── health.go (modify - add history endpoint)

/apps/api/migrations/
└── XXX_create_connection_history_table.sql
```

**Frontend Files to Create:**
```
/apps/web/src/components/health/
├── QBStatusIndicator.tsx
├── QBStatusIndicator.spec.tsx
├── ConnectionHistoryModal.tsx
├── ConnectionHistoryModal.spec.tsx
└── index.ts

/apps/web/src/hooks/
└── useQBConnectionHealth.ts

/apps/web/src/services/
└── healthService.ts (extend with getQBHistory)
```

### Testing Strategy

**Backend Tests:**
1. Health checker qBittorrent tests
2. Monitor qBittorrent status tracking tests
3. Connection history repository tests
4. History endpoint tests

**Frontend Tests:**
1. QBStatusIndicator render tests
2. ConnectionHistoryModal render tests
3. Polling behavior tests

**E2E Tests:**
1. Status indicator changes on connection loss
2. History modal displays events
3. Auto-recovery detection

**Coverage Targets:**
- Backend health package: ≥80%
- Backend repository: ≥80%
- Frontend components: ≥70%

### Error Codes

- `QB_HEALTH_CHECK_FAILED` - Health check failed
- `CONNECTION_HISTORY_ERROR` - Failed to record/retrieve history

### Dependencies

**Story Dependencies:**
- Story 3-12 (Graceful Degradation) - Health monitor infrastructure
- Story 4-1 (Connection Configuration) - qBittorrent client

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.6]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR33]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-R6]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-8]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3-12 (Graceful Degradation):**
- Health monitor pattern established
- ServiceHealth struct can be reused
- DegradationLevel enum available
- `/api/v1/health/services` endpoint exists - extend it

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
