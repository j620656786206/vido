# Story 6.5: Database Backup

Status: ready-for-dev

## Story

As a **system administrator**,
I want to **backup my Vido database and configuration**,
So that **I can restore my data if something goes wrong**.

## Acceptance Criteria

1. **Given** the user opens Settings > Backup, **When** they click "Create Backup Now", **Then** an atomic backup is created using SQLite .backup and backup includes: database, configuration, learned mappings
2. **Given** a backup is created, **When** the backup completes, **Then** it is saved to `/vido-backups` volume and filename format: `vido-backup-YYYYMMDD-HHMMSS-v{schema}.tar.gz`
3. **Given** backup is in progress, **When** viewing the progress, **Then** a progress indicator is shown and backup for 10,000 items completes in <5 minutes

## Tasks / Subtasks

- [ ] Task 1: Create Backup Service (AC: 1, 2)
  - [ ] 1.1: Create `/apps/api/internal/services/backup_service.go` with `BackupServiceInterface`
  - [ ] 1.2: Implement `CreateBackup(ctx) (*BackupInfo, error)` using SQLite `.backup` command (NFR-R7)
  - [ ] 1.3: Package database + config files into tar.gz archive
  - [ ] 1.4: Generate filename: `vido-backup-YYYYMMDD-HHMMSS-v{schema_version}.tar.gz`
  - [ ] 1.5: Save to configurable backup directory (default: `./backups/`)
  - [ ] 1.6: Write unit tests (≥80% coverage)

- [ ] Task 2: Create Backup Repository (AC: 1, 2)
  - [ ] 2.1: Create `/apps/api/internal/repository/backup_repository.go` with `BackupRepositoryInterface`
  - [ ] 2.2: Create migration `013_create_backups_table.go` - table: `backups` (id, filename, size_bytes, schema_version, checksum, status, created_at)
  - [ ] 2.3: Implement `Create`, `List`, `GetByID`, `Delete` methods
  - [ ] 2.4: Write repository tests

- [ ] Task 3: Implement SQLite Atomic Backup (AC: 1)
  - [ ] 3.1: Use `sqlite3_backup_init`, `sqlite3_backup_step`, `sqlite3_backup_finish` via Go SQLite driver
  - [ ] 3.2: Ensure WAL mode compatibility (NFR-R7)
  - [ ] 3.3: Copy to temp file first, then move to final location (atomic)
  - [ ] 3.4: Include settings, learned mappings, and all user data

- [ ] Task 4: Create Backup API Endpoints (AC: 1, 2, 3)
  - [ ] 4.1: Create `/apps/api/internal/handlers/backup_handler.go`
  - [ ] 4.2: `POST /api/v1/settings/backups` → trigger backup creation
  - [ ] 4.3: `GET /api/v1/settings/backups` → list all backups
  - [ ] 4.4: `GET /api/v1/settings/backups/:id` → get backup details
  - [ ] 4.5: `DELETE /api/v1/settings/backups/:id` → delete a backup
  - [ ] 4.6: `GET /api/v1/settings/backups/:id/download` → download backup file
  - [ ] 4.7: Write handler tests (≥70% coverage)

- [ ] Task 5: Create Backup UI (AC: 1, 2, 3)
  - [ ] 5.1: Create `/apps/web/src/components/settings/BackupManagement.tsx` - main backup view
  - [ ] 5.2: Create `/apps/web/src/components/settings/BackupList.tsx` - list of existing backups
  - [ ] 5.3: Create `/apps/web/src/components/settings/BackupCard.tsx` - individual backup entry
  - [ ] 5.4: Add "Create Backup Now" button with loading state
  - [ ] 5.5: Show backup progress indicator during creation
  - [ ] 5.6: Add download and delete buttons per backup

- [ ] Task 6: Create API Client & Hooks (AC: all)
  - [ ] 6.1: Add backup methods to `settingsService.ts`
  - [ ] 6.2: Create `/apps/web/src/hooks/useBackups.ts` - TanStack Query hooks

- [ ] Task 7: Wire Up (AC: all)
  - [ ] 7.1: Register services and handlers in `main.go`
  - [ ] 7.2: Write component tests

## Dev Notes

### Architecture Requirements

**FR57: Backup database and configuration**
**NFR-R7:** Database backups must use SQLite `.backup` command (atomic consistency in WAL mode)
**ARCH-4:** Background Task Queue - backup as background task

### SQLite Backup Implementation

```go
// Using go-sqlite3 driver backup API
func (s *BackupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
    // 1. Create temp file for backup
    tmpFile, err := os.CreateTemp("", "vido-backup-*.db")

    // 2. Open destination connection
    destDB, err := sql.Open("sqlite3", tmpFile.Name())

    // 3. Get raw connections for backup API
    srcConn, _ := s.db.Conn(ctx)
    destConn, _ := destDB.Conn(ctx)

    // 4. Use sqlite3 backup API (via raw connection)
    // This is atomic and safe with WAL mode

    // 5. Package: db file + config files → tar.gz
    // 6. Move to backup directory
    // 7. Record in backups table
}
```

### Backup Contents

The tar.gz archive should contain:
- `vido.db` - Complete SQLite database backup
- `config.json` - Application configuration (excluding secrets)
- `manifest.json` - Backup metadata (schema version, date, item counts)

### API Response Format

```json
// POST /api/v1/settings/backups
{
  "success": true,
  "data": {
    "id": "backup-uuid",
    "filename": "vido-backup-20260210-143000-v12.tar.gz",
    "sizeBytes": 52428800,
    "schemaVersion": 12,
    "status": "completed",
    "createdAt": "2026-02-10T14:30:00Z"
  }
}

// GET /api/v1/settings/backups
{
  "success": true,
  "data": {
    "backups": [ ... ],
    "totalSizeBytes": 157286400
  }
}
```

### Error Codes

- `BACKUP_CREATE_FAILED` - Failed to create backup
- `BACKUP_NOT_FOUND` - Backup ID not found
- `BACKUP_DISK_FULL` - Insufficient disk space for backup
- `BACKUP_IN_PROGRESS` - Another backup is already running

### Project Structure Notes

```
/apps/api/internal/database/migrations/
└── 013_create_backups_table.go

/apps/api/internal/repository/
├── backup_repository.go
└── backup_repository_test.go

/apps/api/internal/services/
├── backup_service.go
└── backup_service_test.go

/apps/api/internal/handlers/
├── backup_handler.go
└── backup_handler_test.go

/apps/web/src/components/settings/
├── BackupManagement.tsx
├── BackupManagement.spec.tsx
├── BackupList.tsx
├── BackupCard.tsx
└── BackupCard.spec.tsx
```

### Note on Migration Numbering

Check existing migration count before creating. If Story 6.3 creates migration 013, this story should use 014.

### Dependencies

- Story 1-1 (Repository Pattern) - database access
- SQLite WAL mode (already configured)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.5]
- [Source: _bmad-output/planning-artifacts/prd.md#FR57]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-R7]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-4-Background-Task]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
