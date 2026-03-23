# Story 7.2: Scheduled Scan Service

Status: ready-for-dev

## Story

As a **NAS media collector**,
I want **Vido to automatically re-scan my media folders on a schedule**,
so that **new downloads are discovered without manual intervention**.

## Acceptance Criteria

1. **Given** scan schedule is configured (hourly/daily/manual-only), **When** the scheduled time arrives, **Then** ScannerService.StartScan() is called automatically
2. **Given** an incremental scan runs, **When** comparing current filesystem with DB records, **Then** only new files (not in DB) are inserted, changed files (mtime differs) are updated, and removed files (in DB but not on disk) are marked as removed
3. **Given** scan schedule is set to "manual only", **When** no schedule timer is active, **Then** no automatic scans occur
4. **Given** a scheduled scan triggers while a manual scan is running, **When** mutex detects conflict, **Then** the scheduled scan is skipped (logged, not queued)
5. **Given** scan settings are changed via API, **When** new schedule is saved, **Then** the scheduler restarts with the new interval immediately

## Tasks / Subtasks

- [ ] Task 1: Implement scan scheduler using Go ticker/cron (AC: 1, 3, 5)
  - [ ] 1.1: Create `apps/api/internal/services/scan_scheduler.go` with `ScanScheduler` struct
  - [ ] 1.2: Implement schedule configuration (hourly/daily/manual-only) using `time.Ticker`
  - [ ] 1.3: Implement `Start(ctx context.Context)` goroutine that calls `ScannerService.StartScan()` on tick
  - [ ] 1.4: Implement `Reconfigure(interval string)` — stop old ticker, start new one based on interval
  - [ ] 1.5: Implement `Stop()` for graceful shutdown
  - [ ] 1.6: Load schedule preference on startup via `SettingsService.GetString(ctx, "scan_schedule")` (default: "manual")
  - [ ] 1.7: Write unit tests for scheduler start/stop/reconfigure (≥80% coverage)

- [ ] Task 2: Implement incremental scan logic (AC: 2)
  - [ ] 2.1: Extend `ScannerService.StartScan()` to support incremental mode (compare file mtime with DB record's UpdatedAt)
  - [ ] 2.2: New files (not in DB by FilePath): insert with ParseStatus=pending
  - [ ] 2.3: Changed files (mtime > DB UpdatedAt): update FileSize, reset ParseStatus=pending
  - [ ] 2.4: Removed files (in DB but not on disk): set soft-delete flag (IsRemoved=true), do NOT hard-delete
  - [ ] 2.5: Add `IsRemoved bool` field to media models if not present (migration if needed)
  - [ ] 2.6: Write tests for each case: new file detected, changed file updated, removed file soft-deleted (≥80% coverage)

- [ ] Task 3: Handle scheduled scan conflict with mutex (AC: 4)
  - [ ] 3.1: In scheduler tick handler, call `ScannerService.IsScanActive()` before starting
  - [ ] 3.2: If scan is active, log `slog.Info("Scheduled scan skipped — manual scan in progress")` and return
  - [ ] 3.3: Do NOT queue skipped scheduled scans — next tick will try again
  - [ ] 3.4: Write test verifying scheduled scan is skipped when manual scan is active

- [ ] Task 4: Add schedule API endpoints (AC: 5)
  - [ ] 4.1: Add `GET /api/v1/scanner/schedule` to ScannerHandler — return current schedule config from SettingsService
  - [ ] 4.2: Add `PUT /api/v1/scanner/schedule` to ScannerHandler — update schedule (body: `{interval: "hourly"|"daily"|"manual"}`)
  - [ ] 4.3: Handler validates interval value, persists via `SettingsService.SetString(ctx, "scan_schedule", interval)`, then calls `ScanScheduler.Reconfigure(interval)`
  - [ ] 4.4: Return 400 with SCANNER_SCHEDULE_INVALID for unrecognized interval values
  - [ ] 4.5: Write handler tests (≥70% coverage)

- [ ] Task 5: Wire into main.go and verify (AC: all)
  - [ ] 5.1: Initialize `ScanScheduler` in main.go with `ScannerService` + `SettingsService` dependencies
  - [ ] 5.2: Start scheduler goroutine after ScannerService initialization: `go scanScheduler.Start(schedulerCtx)`
  - [ ] 5.3: Add graceful shutdown: call `scanScheduler.Stop()` in shutdown sequence (before DB close)
  - [ ] 5.4: Run `go build ./cmd/api/` — verify build passes
  - [ ] 5.5: Run `go test ./...` — verify all tests pass
  - [ ] 5.6: Manual verification: set schedule to "hourly", observe scan triggers; change to "manual", observe no triggers

## Dev Notes

### Gate 2A Decisions (Mandatory)
- **Incremental scan:** Use file mtime for change detection (compare against DB record's UpdatedAt)
- **Mutex:** Scheduled scan skipped if manual scan is active (don't queue)
- **Concurrency:** Only one scan at a time (reuse ScannerService mutex from Story 7-1)

### Phase 0 Infrastructure Available
- `ScannerService` from Story 7-1 with `StartScan()`, `IsScanActive()`, `CancelScan()`
- `SettingsService` with `GetString(ctx, key)` / `SetString(ctx, key, value)` for persistent config
- SSE Hub for broadcasting scan_progress events (already wired in Story 7-1)
- `FindByFilePath()` on repositories for existing file lookup
- `BulkCreate()` for batch insertion

### Scope Boundaries — DO NOT Implement
- ❌ UI for schedule configuration (Story 7-3)
- ❌ Scan progress UI (Story 7-4)
- ❌ Filename parsing or TMDB matching (already done in Epics 2-3)
- ❌ Subtitle search (Epic 8)
- ❌ Folder path editing via API (configured via VIDO_MEDIA_DIRS env var)

### Service Pattern to Follow
```
Handler → Service → Repository → Database
ScanScheduler → ScannerService → repos
```
- `ScanScheduler` depends on: `*ScannerService`, `*SettingsService`
- Schedule endpoints added to existing `ScannerHandler` (not a new handler)
- Reference implementation: `services/backup_scheduler.go` for similar scheduling pattern (Start/Stop with context)

### Error Codes
- `SCANNER_SCHEDULE_INVALID` — Unrecognized schedule interval (not hourly/daily/manual)
- `SCANNER_ALREADY_RUNNING` — Reuse from Story 7-1 (409 Conflict)

### Project Structure Notes
- Scan scheduler: `apps/api/internal/services/scan_scheduler.go`
- Tests co-located: `scan_scheduler_test.go` in same directory
- Schedule endpoints: added to existing `apps/api/internal/handlers/scanner_handler.go`
- All logging: `log/slog` (NOT zerolog, NOT fmt.Println)
- API responses: `{success: true/false, data/error}` format
- Default schedule: "manual" (no auto-scan until user configures)

### References
- [Source: project-context.md] — Mandatory rules (logging, error codes, layered architecture)
- [Source: epic-7-media-library-scanner.md] — Epic scope, success criteria, Gate 2A decisions
- [Source: prd/functional-requirements.md#P1-005] — Scheduled scan requirements
- [Source: 7-1-recursive-folder-scanner.md] — ScannerService API, mutex pattern, SSE integration

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
