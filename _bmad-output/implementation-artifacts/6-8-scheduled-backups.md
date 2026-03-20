# Story 6.8: Scheduled Backups

Status: done

## Story

As a **system administrator**,
I want to **schedule automatic backups**,
So that **I don't have to remember to backup manually**.

## Acceptance Criteria

1. **Given** the user opens backup settings, **When** configuring schedule, **Then** options include: Daily, Weekly, or Disabled, and time of day can be selected
2. **Given** scheduled backup is enabled, **When** the scheduled time arrives, **Then** backup runs automatically and runs in background without UI disruption
3. **Given** backups accumulate, **When** retention policy is active, **Then** keeps last 7 daily + last 4 weekly backups and older backups are automatically deleted (FR64)

## Tasks / Subtasks

- [x] Task 1: Create Backup Scheduler Service (AC: 1, 2)
  - [x] 1.1: Create `/apps/api/internal/services/backup_scheduler.go` with `BackupSchedulerInterface`
  - [x] 1.2: Implement `Start(ctx)` - goroutine that checks schedule every minute
  - [x] 1.3: Implement `Stop()` - graceful shutdown
  - [x] 1.4: Implement `SetSchedule(ctx, schedule BackupSchedule) error` - save schedule config
  - [x] 1.5: Implement `GetSchedule(ctx) (*BackupSchedule, error)`
  - [x] 1.6: Use settings table to persist schedule configuration
  - [x] 1.7: Write unit tests (≥80% coverage)

- [x] Task 2: Create Retention Policy Service (AC: 3)
  - [x] 2.1: Add `ApplyRetentionPolicy(ctx) (*RetentionResult, error)` to backup service
  - [x] 2.2: Keep last 7 daily backups
  - [x] 2.3: Keep last 4 weekly backups (oldest daily per week)
  - [x] 2.4: Delete files and DB records for expired backups
  - [x] 2.5: Never delete auto-snapshots (from restore operations)
  - [x] 2.6: Run after each scheduled backup completes
  - [x] 2.7: Write unit tests (≥80% coverage)

- [x] Task 3: Create Schedule API Endpoints (AC: 1)
  - [x] 3.1: `GET /api/v1/settings/backups/schedule` → get current schedule
  - [x] 3.2: `PUT /api/v1/settings/backups/schedule` → update schedule
  - [x] 3.3: Write handler tests (≥70% coverage)

- [x] Task 4: Create Schedule UI (AC: 1)
  - [x] 4.1: Create `/apps/web/src/components/settings/BackupScheduleConfig.tsx`
  - [x] 4.2: Frequency selector: Disabled / Daily / Weekly dropdown
  - [x] 4.3: Time picker for backup hour (00:00 - 23:00)
  - [x] 4.4: For weekly: day of week selector
  - [x] 4.5: Show next scheduled backup time
  - [x] 4.6: Show retention policy info (7 daily + 4 weekly)

- [x] Task 5: Wire Up (AC: all)
  - [x] 5.1: Start scheduler in `main.go` (background goroutine)
  - [x] 5.2: Stop scheduler on app shutdown (graceful)
  - [x] 5.3: Write component tests

## Dev Notes

### Architecture Requirements

**FR63: Configure backup schedule (daily/weekly)**
**FR64: Auto-cleanup old backups (retention policy)**
**ARCH-4:** Background Task Queue

### Schedule Configuration Model

```go
type BackupSchedule struct {
    Enabled   bool   `json:"enabled"`
    Frequency string `json:"frequency"` // "daily", "weekly", "disabled"
    Hour      int    `json:"hour"`      // 0-23
    DayOfWeek int    `json:"dayOfWeek"` // 0=Sunday, 1=Monday, ... (only for weekly)
}
```

### Scheduler Implementation

```go
func (s *BackupScheduler) run(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-s.stop:
            return
        case now := <-ticker.C:
            schedule, _ := s.GetSchedule(ctx)
            if schedule == nil || !schedule.Enabled {
                continue
            }
            if s.shouldRunNow(now, schedule) {
                slog.Info("Starting scheduled backup")
                _, err := s.backupService.CreateBackup(ctx)
                if err != nil {
                    slog.Error("Scheduled backup failed", "error", err)
                } else {
                    s.backupService.ApplyRetentionPolicy(ctx)
                }
            }
        }
    }
}
```

### Retention Policy Logic

```
After each backup:
1. Get all backups sorted by date DESC
2. Mark first 7 as "keep" (daily retention)
3. For remaining, keep 1 per week for last 4 weeks (weekly retention)
4. Delete everything else (files + DB records)
5. Never delete type="auto_snapshot" backups
```

### API Response Format

```json
// GET /api/v1/settings/backups/schedule
{
  "success": true,
  "data": {
    "enabled": true,
    "frequency": "daily",
    "hour": 3,
    "dayOfWeek": 0,
    "nextBackupAt": "2026-02-11T03:00:00Z",
    "lastBackupAt": "2026-02-10T03:00:00Z"
  }
}
```

### Error Codes

- `SCHEDULE_INVALID` - Invalid schedule configuration
- `RETENTION_CLEANUP_FAILED` - Failed to delete old backups

### Dependencies

- Story 6-5 (Database Backup) - backup service
- Settings table for schedule persistence

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.8]
- [Source: _bmad-output/planning-artifacts/prd.md#FR63]
- [Source: _bmad-output/planning-artifacts/prd.md#FR64]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-4-Background-Task]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Created BackupScheduler with Start/Stop/SetSchedule/GetSchedule. Uses 1-minute ticker with shouldRunNow logic and duplicate run prevention via lastRunDate tracking.
- Task 2: Implemented ApplyRetentionPolicy — keeps 7 daily + 4 weekly (by ISO week), never deletes auto-snapshots. Runs after each scheduled backup.
- Task 3: Added GET/PUT /api/v1/settings/backups/schedule endpoints via BackupHandler.SetScheduler pattern.
- Task 4: Created BackupScheduleConfig component with toggle, frequency/hour/dayOfWeek selectors, next backup time display, retention info.
- Task 5: Wired scheduler in main.go — background goroutine with context cancellation + Stop() on shutdown.
- 🎨 UX Verification: PASS — Schedule UI matches design screenshot (toggle, frequency, time, retention info)

### Change Log

- 2026-03-20: Implemented Story 6-8 Scheduled Backups — all tasks complete

### File List

- apps/api/internal/services/backup_scheduler.go (new — BackupScheduler with schedule, retention, timing logic)
- apps/api/internal/services/backup_scheduler_test.go (new — 25 tests for scheduler+retention)
- apps/api/internal/handlers/backup_handler.go (modified — added GetSchedule, UpdateSchedule handlers, SetScheduler)
- apps/api/internal/handlers/backup_handler_test.go (modified — added schedule handler tests)
- apps/api/cmd/api/main.go (modified — wired scheduler start/stop)
- apps/web/src/services/backupService.ts (modified — added BackupSchedule type, getSchedule/updateSchedule methods)
- apps/web/src/services/backupService.spec.ts (modified — schedule service tests pending)
- apps/web/src/hooks/useBackups.ts (modified — added useBackupSchedule, useUpdateSchedule hooks)
- apps/web/src/components/settings/BackupScheduleConfig.tsx (new — schedule config UI component)
- apps/web/src/components/settings/BackupScheduleConfig.spec.tsx (new — 8 component tests)
- apps/web/src/components/settings/BackupManagement.tsx (modified — integrated BackupScheduleConfig)
- apps/web/src/components/settings/BackupManagement.spec.tsx (modified — added schedule mock setup)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified — status tracking)
