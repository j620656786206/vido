# Story 6.6: Backup Integrity Verification

Status: ready-for-dev

## Story

As a **system administrator**,
I want **backup integrity to be verified**,
So that **I know my backups are reliable**.

## Acceptance Criteria

1. **Given** a backup is created, **When** the backup completes, **Then** a SHA-256 checksum is calculated and stored alongside the backup file
2. **Given** a backup file exists, **When** the user clicks "Verify Backup", **Then** the checksum is recalculated and compared against the stored checksum
3. **Given** verification fails, **When** the checksum doesn't match, **Then** the backup is marked as "Potentially Corrupted" and the user is warned before attempting restore

## Tasks / Subtasks

- [ ] Task 1: Add Checksum to Backup Service (AC: 1)
  - [ ] 1.1: Extend `BackupService.CreateBackup()` to compute SHA-256 after tar.gz creation
  - [ ] 1.2: Store checksum in `backups` table `checksum` column
  - [ ] 1.3: Write `.sha256` sidecar file alongside the backup: `{filename}.sha256`
  - [ ] 1.4: Write unit tests for checksum generation

- [ ] Task 2: Create Verification Service (AC: 2, 3)
  - [ ] 2.1: Add `VerifyBackup(ctx, backupID string) (*VerificationResult, error)` to `BackupServiceInterface`
  - [ ] 2.2: Recalculate SHA-256 of backup file on disk
  - [ ] 2.3: Compare with stored checksum in DB and `.sha256` file
  - [ ] 2.4: Return verification status: `verified`, `corrupted`, `missing`
  - [ ] 2.5: If corrupted, update backup record status to `corrupted`
  - [ ] 2.6: Write unit tests (≥80% coverage)

- [ ] Task 3: Create Verification Endpoint (AC: 2, 3)
  - [ ] 3.1: `POST /api/v1/settings/backups/:id/verify` → trigger verification
  - [ ] 3.2: Return verification result with details
  - [ ] 3.3: Write handler tests (≥70% coverage)

- [ ] Task 4: Update Backup UI (AC: 2, 3)
  - [ ] 4.1: Add "Verify" button to each BackupCard
  - [ ] 4.2: Show verification status badge: ✅ Verified / ⚠️ Corrupted / ❓ Unverified
  - [ ] 4.3: Show warning dialog when backup is corrupted
  - [ ] 4.4: Write component tests

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
