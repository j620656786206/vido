# Story 6.8: Scheduled Backups

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **schedule automatic backups**,
So that **I don't have to remember to backup manually**.

## Acceptance Criteria

1. **Given** the user opens backup settings, **When** configuring schedule, **Then** options include: Daily, Weekly, or Disabled, and time of day can be selected
2. **Given** scheduled backup is enabled, **When** the scheduled time arrives, **Then** backup runs automatically and runs in background without UI disruption
3. **Given** backups accumulate, **When** retention policy is active, **Then** keeps last 7 daily + last 4 weekly backups and older backups are automatically deleted (FR64)

## Tasks / Subtasks

- [ ] Task 1: Create Backup Scheduler Service (AC: 1, 2)
  - [ ] 1.1: Create `/apps/api/internal/services/backup_scheduler.go` with `BackupSchedulerInterface`
  - [ ] 1.2: Implement `Start(ctx)` - goroutine that checks schedule every minute
  - [ ] 1.3: Implement `Stop()` - graceful shutdown
  - [ ] 1.4: Implement `SetSchedule(ctx, schedule BackupSchedule) error` - save schedule config
  - [ ] 1.5: Implement `GetSchedule(ctx) (*BackupSchedule, error)`
  - [ ] 1.6: Use settings table to persist schedule configuration
  - [ ] 1.7: Write unit tests (≥80% coverage)

- [ ] Task 2: Create Retention Policy Service (AC: 3)
  - [ ] 2.1: Add `ApplyRetentionPolicy(ctx) (*RetentionResult, error)` to backup service
  - [ ] 2.2: Keep last 7 daily backups
  - [ ] 2.3: Keep last 4 weekly backups (oldest daily per week)
  - [ ] 2.4: Delete files and DB records for expired backups
  - [ ] 2.5: Never delete auto-snapshots (from restore operations)
  - [ ] 2.6: Run after each scheduled backup completes
  - [ ] 2.7: Write unit tests (≥80% coverage)

- [ ] Task 3: Create Schedule API Endpoints (AC: 1)
  - [ ] 3.1: `GET /api/v1/settings/backups/schedule` → get current schedule
  - [ ] 3.2: `PUT /api/v1/settings/backups/schedule` → update schedule
  - [ ] 3.3: Write handler tests (≥70% coverage)

- [ ] Task 4: Create Schedule UI (AC: 1)
  - [ ] 4.1: Create `/apps/web/src/components/settings/BackupScheduleConfig.tsx`
  - [ ] 4.2: Frequency selector: Disabled / Daily / Weekly dropdown
  - [ ] 4.3: Time picker for backup hour (00:00 - 23:00)
  - [ ] 4.4: For weekly: day of week selector
  - [ ] 4.5: Show next scheduled backup time
  - [ ] 4.6: Show retention policy info (7 daily + 4 weekly)

- [ ] Task 5: Wire Up (AC: all)
  - [ ] 5.1: Start scheduler in `main.go` (background goroutine)
  - [ ] 5.2: Stop scheduler on app shutdown (graceful)
  - [ ] 5.3: Write component tests

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
