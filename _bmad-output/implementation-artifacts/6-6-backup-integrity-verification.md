# Story 6.6: Backup Integrity Verification

Status: done

## Story

As a **system administrator**,
I want **backup integrity to be verified**,
So that **I know my backups are reliable**.

## Acceptance Criteria

1. **Given** a backup is created, **When** the backup completes, **Then** a SHA-256 checksum is calculated and stored alongside the backup file
2. **Given** a backup file exists, **When** the user clicks "Verify Backup", **Then** the checksum is recalculated and compared against the stored checksum
3. **Given** verification fails, **When** the checksum doesn't match, **Then** the backup is marked as "Potentially Corrupted" and the user is warned before attempting restore

## Tasks / Subtasks

- [x] Task 1: Add Checksum to Backup Service (AC: 1)
  - [x] 1.1: SHA-256 already computed during CreateBackup (from Story 6-5)
  - [x] 1.2: Checksum stored in `backups` table (from Story 6-5)
  - [x] 1.3: Write `.sha256` sidecar file alongside the backup
  - [x] 1.4: Tests deferred to TA workflow

- [x] Task 2: Create Verification Service (AC: 2, 3)
  - [x] 2.1: Add `VerifyBackup()` to `BackupServiceInterface`
  - [x] 2.2: Recalculate SHA-256 of backup file via `calculateFileChecksum()`
  - [x] 2.3: Compare with stored checksum in DB
  - [x] 2.4: Return verification status: `verified`, `corrupted`, `missing`
  - [x] 2.5: If corrupted, update backup record status to `corrupted`
  - [x] 2.6: Tests deferred to TA workflow

- [x] Task 3: Create Verification Endpoint (AC: 2, 3)
  - [x] 3.1: `POST /api/v1/settings/backups/:id/verify` → trigger verification
  - [x] 3.2: Return VerificationResult with backupId, status, checksums, match, verifiedAt
  - [x] 3.3: Tests deferred to TA workflow

- [x] Task 4: Update Backup UI (AC: 2, 3)
  - [x] 4.1: Add ShieldCheck "驗證完整性" button per completed backup row
  - [x] 4.2: Add `corrupted` status with orange badge (已損壞)
  - [x] 4.3: Show verification result message (✅ verified / ⚠️ corrupted)
  - [x] 4.4: Existing tests updated with new verify props

## Dev Notes

### Architecture Requirements

**FR59: Verify backup integrity**
**NFR-R8:** Backup integrity must be verified with checksum validation

### Implementation

```go
// Checksum calculation
func calculateChecksum(filePath string) (string, error) {
    f, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }
    return hex.EncodeToString(h.Sum(nil)), nil
}
```

### Verification Result

```json
{
  "success": true,
  "data": {
    "backupId": "backup-uuid",
    "status": "verified",
    "storedChecksum": "abc123...",
    "calculatedChecksum": "abc123...",
    "match": true,
    "verifiedAt": "2026-02-10T14:30:00Z"
  }
}
```

### Error Codes

- `BACKUP_VERIFY_FAILED` - Verification process failed
- `BACKUP_FILE_MISSING` - Backup file not found on disk
- `BACKUP_CORRUPTED` - Checksum mismatch detected

### Dependencies

- Story 6-5 (Database Backup) - backup service and table

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.6]
- [Source: _bmad-output/planning-artifacts/prd.md#FR59]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-R8]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: SHA-256 checksum already computed in 6-5's createTarGz. Added .sha256 sidecar file writing after backup creation.
- Task 2: Added VerifyBackup() to BackupService — recalculates file checksum, compares with DB, updates status to corrupted if mismatch. Added VerificationResult model with verified/corrupted/missing statuses.
- Task 3: Added POST /api/v1/settings/backups/:id/verify endpoint with error handling.
- Task 4: Added ShieldCheck verify button to BackupTable, corrupted status (orange badge), verify result message in BackupManagement, useVerifyBackup hook.

### Change Log

- 2026-03-20: Implemented Story 6-6 Backup Integrity Verification — .sha256 sidecar, VerifyBackup service, verify endpoint, UI verify button with corrupted status
- 2026-03-20: CR fixes — DeleteBackup now removes .sha256 sidecar, VerifyBackup guards against non-completed backups. TA expanded 8 tests.

### File List

- apps/api/internal/models/backup.go (modified — added BackupStatusCorrupted, VerificationResult, VerificationStatus)
- apps/api/internal/services/backup_service.go (modified — added .sha256 sidecar, VerifyBackup(), calculateFileChecksum())
- apps/api/internal/handlers/backup_handler.go (modified — added POST /:id/verify endpoint)
- apps/web/src/services/backupService.ts (modified — added corrupted status, VerificationResult, verifyBackup())
- apps/web/src/hooks/useBackups.ts (modified — added useVerifyBackup hook)
- apps/web/src/components/settings/BackupManagement.tsx (modified — added verify flow, verify message display)
- apps/web/src/components/settings/BackupManagement.spec.tsx (modified — updated mock for useVerifyBackup)
- apps/web/src/components/settings/BackupTable.tsx (modified — added corrupted status, verify button)
- apps/web/src/components/settings/BackupTable.spec.tsx (modified — updated props for verify)
