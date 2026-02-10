# Story 6.7: Data Restore

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **restore data from a backup**,
So that **I can recover from data loss or migration**.

## Acceptance Criteria

1. **Given** backup files exist, **When** the user opens Settings > Backup > Restore, **Then** available backups are listed with date, size, and version
2. **Given** the user selects a backup, **When** they click "Restore", **Then** a confirmation dialog warns: "This will replace current data" and an auto-snapshot of current state is created first (NFR-R9)
3. **Given** restore is confirmed, **When** the restore process runs, **Then** progress is shown and the application restarts with restored data
4. **Given** restore fails, **When** an error occurs, **Then** the auto-snapshot is used to recover and an error message explains what happened

## Tasks / Subtasks

- [ ] Task 1: Create Restore Service (AC: 2, 3, 4)
  - [ ] 1.1: Add `RestoreBackup(ctx, backupID string) error` to `BackupServiceInterface`
  - [ ] 1.2: Implement pre-restore auto-snapshot (NFR-R9): create automatic backup before restore
  - [ ] 1.3: Verify backup integrity before restoring (call VerifyBackup first)
  - [ ] 1.4: Extract tar.gz to temp directory
  - [ ] 1.5: Check schema version compatibility
  - [ ] 1.6: Replace current database with backup database using SQLite backup API
  - [ ] 1.7: Restore configuration files
  - [ ] 1.8: On failure, restore from auto-snapshot
  - [ ] 1.9: Write unit tests (≥80% coverage)

- [ ] Task 2: Create Restore Endpoint (AC: 2, 3, 4)
  - [ ] 2.1: `POST /api/v1/settings/backups/:id/restore` → trigger restore
  - [ ] 2.2: Return immediate response with restore job ID (async operation)
  - [ ] 2.3: `GET /api/v1/settings/restore/status` → check restore progress
  - [ ] 2.4: Write handler tests (≥70% coverage)

- [ ] Task 3: Create Restore UI (AC: 1, 2, 3, 4)
  - [ ] 3.1: Add "Restore" button to each BackupCard
  - [ ] 3.2: Create `/apps/web/src/components/settings/RestoreConfirmDialog.tsx` - warning dialog
  - [ ] 3.3: Show restore progress with status updates
  - [ ] 3.4: Handle app restart gracefully (redirect to loading page)
  - [ ] 3.5: Show error with recovery info if restore fails

- [ ] Task 4: Wire Up & Tests (AC: all)
  - [ ] 4.1: Register restore endpoint
  - [ ] 4.2: Write component tests for RestoreConfirmDialog

## Dev Notes

### Architecture Requirements

**FR58: Restore data from backup**
**NFR-R9:** System must automatically create snapshot before any restore operation

### Restore Flow

```
User clicks Restore → Confirmation Dialog → Auto-Snapshot → Verify Backup →
Extract Archive → Schema Version Check → Replace Database → Restore Config →
Signal App Restart → Success/Failure
```

### Schema Version Compatibility

```go
type RestoreCompatibility struct {
    BackupSchemaVersion  int  `json:"backupSchemaVersion"`
    CurrentSchemaVersion int  `json:"currentSchemaVersion"`
    Compatible           bool `json:"compatible"`
    NeedsMigration       bool `json:"needsMigration"`
}

// Rules:
// - Same version: direct restore
// - Backup older: restore + run pending migrations
// - Backup newer: INCOMPATIBLE (user needs to upgrade Vido first)
```

### Auto-Snapshot Naming

```
vido-auto-snapshot-before-restore-YYYYMMDD-HHMMSS.tar.gz
```
- Tagged as `auto_snapshot` type in backups table
- Not counted toward retention policy limits

### API Response Format

```json
// POST /api/v1/settings/backups/:id/restore
{
  "success": true,
  "data": {
    "restoreId": "restore-uuid",
    "status": "in_progress",
    "snapshotId": "snapshot-backup-uuid",
    "message": "自動快照已建立，正在還原..."
  }
}
```

### Error Codes

- `RESTORE_INCOMPATIBLE_VERSION` - Backup schema version newer than current
- `RESTORE_SNAPSHOT_FAILED` - Failed to create pre-restore snapshot
- `RESTORE_VERIFY_FAILED` - Backup integrity check failed
- `RESTORE_EXTRACT_FAILED` - Failed to extract backup archive
- `RESTORE_DB_FAILED` - Failed to restore database
- `RESTORE_ROLLBACK_SUCCESS` - Restore failed but rollback to snapshot succeeded
- `RESTORE_ROLLBACK_FAILED` - Both restore and rollback failed (critical)

### Dependencies

- Story 6-5 (Database Backup) - backup service
- Story 6-6 (Backup Verification) - integrity check

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.7]
- [Source: _bmad-output/planning-artifacts/prd.md#FR58]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-R9]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
