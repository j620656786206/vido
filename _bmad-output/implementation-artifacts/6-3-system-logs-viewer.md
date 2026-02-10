# Story 6.3: System Logs Viewer

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **view system logs**,
So that **I can troubleshoot issues and monitor system health**.

## Acceptance Criteria

1. **Given** the user opens Settings > Logs, **When** logs are displayed, **Then** entries show: timestamp, level (ERROR/WARN/INFO/DEBUG), message, and logs are color-coded by level
2. **Given** many log entries exist, **When** viewing the log list, **Then** pagination or infinite scroll is available and newest logs are shown first
3. **Given** logs are displayed, **When** the user filters by level (e.g., "ERROR only"), **Then** only matching entries are shown and search by keyword is available
4. **Given** any log entry, **When** it contains sensitive information, **Then** API keys are masked (NFR-S4) and error hints are provided (NFR-U9)

## Tasks / Subtasks

- [ ] Task 1: Create Log Storage System (AC: 1, 2)
  - [ ] 1.1: Create migration `013_create_system_logs_table.go` - table: `system_logs` (id, level, message, context_json, created_at)
  - [ ] 1.2: Create `/apps/api/internal/repository/log_repository.go` with `LogRepositoryInterface`
  - [ ] 1.3: Implement `GetLogs(ctx, filter LogFilter) ([]SystemLog, int, error)` with pagination and total count
  - [ ] 1.4: Implement `CreateLog(ctx, log *SystemLog) error` for persisting logs
  - [ ] 1.5: Write repository tests (≥80% coverage)

- [ ] Task 2: Create slog Database Handler (AC: 1, 4)
  - [ ] 2.1: Create `/apps/api/internal/logger/db_handler.go` - custom `slog.Handler` that writes to `system_logs` table
  - [ ] 2.2: Implement sensitive data filtering (mask API keys, passwords matching patterns)
  - [ ] 2.3: Buffer logs and batch-write to DB (every 5s or 100 entries) to avoid performance impact
  - [ ] 2.4: Keep existing stdout handler, add DB handler as multi-handler
  - [ ] 2.5: Write unit tests

- [ ] Task 3: Create Logs API Endpoints (AC: 1, 2, 3)
  - [ ] 3.1: Create `/apps/api/internal/handlers/log_handler.go`
  - [ ] 3.2: `GET /api/v1/settings/logs` → paginated logs with filters (level, keyword, date range)
  - [ ] 3.3: `DELETE /api/v1/settings/logs` → clear logs older than N days
  - [ ] 3.4: Write handler tests (≥70% coverage)

- [ ] Task 4: Create Logs Viewer UI (AC: 1, 2, 3, 4)
  - [ ] 4.1: Create `/apps/web/src/components/settings/LogsViewer.tsx` - main log viewer component
  - [ ] 4.2: Create `/apps/web/src/components/settings/LogEntry.tsx` - individual log entry with color-coding
  - [ ] 4.3: Create `/apps/web/src/components/settings/LogFilters.tsx` - level filter chips + keyword search
  - [ ] 4.4: Implement infinite scroll with TanStack Query `useInfiniteQuery`
  - [ ] 4.5: Add troubleshooting hints for ERROR entries (NFR-U9)

- [ ] Task 5: Create API Client & Hooks (AC: all)
  - [ ] 5.1: Add log methods to `/apps/web/src/services/settingsService.ts`
  - [ ] 5.2: Create `/apps/web/src/hooks/useLogs.ts` - infinite query hook with filters

- [ ] Task 6: Wire Up (AC: all)
  - [ ] 6.1: Register DB handler in slog setup (main.go)
  - [ ] 6.2: Register log handler routes
  - [ ] 6.3: Write component tests

## Dev Notes

### Architecture Requirements

**FR54: View system logs**
**NFR-M11:** Severity-level logging (ERROR, WARN, INFO, DEBUG)
**NFR-S4:** API keys must NEVER appear in logs
**NFR-U9:** Error logs with actionable troubleshooting hints

### Existing Codebase Context

**Logging already uses slog:** The project uses Go `log/slog` (project-context.md Rule 2). Need to add a database handler alongside existing stdout handler.

**Sensitive data filtering:** Already have secrets management in `/apps/api/internal/secrets/`. Reuse pattern matching for filtering API keys in logs.

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS system_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level TEXT NOT NULL,           -- ERROR, WARN, INFO, DEBUG
    message TEXT NOT NULL,
    source TEXT,                   -- package/module name
    context_json TEXT,             -- JSON of structured fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_system_logs_level ON system_logs(level);
CREATE INDEX idx_system_logs_created_at ON system_logs(created_at DESC);
```

### Log Level Color Coding

| Level | Color | Tailwind Class |
|---|---|---|
| ERROR | Red | `text-red-400 bg-red-400/10` |
| WARN | Yellow | `text-yellow-400 bg-yellow-400/10` |
| INFO | Blue | `text-blue-400 bg-blue-400/10` |
| DEBUG | Gray | `text-gray-400 bg-gray-400/10` |

### Troubleshooting Hints Map

```go
var troubleshootingHints = map[string]string{
    "TMDB_TIMEOUT":     "檢查網路連線，或稍後重試。TMDb API 可能暫時不可用。",
    "AI_QUOTA_EXCEEDED": "AI API 配額已用完。請檢查帳戶或等待配額重置。",
    "DB_QUERY_FAILED":  "資料庫查詢失敗。請檢查磁碟空間是否充足。",
    "QBT_CONNECTION":   "無法連線到 qBittorrent。請確認服務是否正在運行。",
}
```

### API Response Format

```json
// GET /api/v1/settings/logs?level=ERROR&keyword=tmdb&page=1&per_page=50
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": 1234,
        "level": "ERROR",
        "message": "Failed to fetch metadata from TMDb",
        "source": "tmdb",
        "context": { "movie_id": "12345", "error_code": "TMDB_TIMEOUT" },
        "hint": "檢查網路連線，或稍後重試。TMDb API 可能暫時不可用。",
        "createdAt": "2026-02-10T14:30:00Z"
      }
    ],
    "total": 150,
    "page": 1,
    "perPage": 50
  }
}
```

### Error Codes

- `LOGS_QUERY_FAILED` - Failed to query logs
- `LOGS_CLEAR_FAILED` - Failed to clear old logs

### Project Structure Notes

```
/apps/api/internal/database/migrations/
└── 013_create_system_logs_table.go

/apps/api/internal/repository/
├── log_repository.go
└── log_repository_test.go

/apps/api/internal/logger/
├── db_handler.go
└── db_handler_test.go

/apps/api/internal/handlers/
├── log_handler.go
└── log_handler_test.go

/apps/web/src/components/settings/
├── LogsViewer.tsx
├── LogsViewer.spec.tsx
├── LogEntry.tsx
├── LogFilters.tsx
└── LogFilters.spec.tsx
```

### Dependencies

- Existing slog setup in main.go
- Settings table for log retention config

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.3]
- [Source: _bmad-output/planning-artifacts/prd.md#FR54]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-M11]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-S4]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-U9]
- [Source: project-context.md#Rule-2-Logging-with-slog]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
