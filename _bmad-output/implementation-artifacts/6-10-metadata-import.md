# Story 6.10: Metadata Import

Status: ready-for-dev

## Story

As a **power user**,
I want to **import metadata from JSON/YAML files**,
So that **I can restore or migrate my library data**.

## Acceptance Criteria

1. **Given** the user has a JSON/YAML export file, **When** they select "Import Metadata", **Then** they can upload or provide path to the file
2. **Given** an import file is provided, **When** import runs, **Then** metadata is merged with existing library and conflicts are handled: Skip / Overwrite / Ask
3. **Given** import completes, **When** viewing results, **Then** summary shows: Added X, Updated Y, Skipped Z items

## Tasks / Subtasks

- [ ] Task 1: Create Import Service (AC: 1, 2, 3)
  - [ ] 1.1: Create `/apps/api/internal/services/import_service.go` with `ImportServiceInterface`
  - [ ] 1.2: Implement `ValidateImportFile(ctx, filePath string) (*ImportPreview, error)` - parse and validate
  - [ ] 1.3: Implement `ImportMetadata(ctx, config ImportConfig) (*ImportResult, error)` - execute import
  - [ ] 1.4: Support JSON and YAML formats (detect from file extension or content)
  - [ ] 1.5: Implement conflict resolution strategies: `skip`, `overwrite`, `ask`
  - [ ] 1.6: Write unit tests (≥80% coverage)

- [ ] Task 2: Implement Merge Logic (AC: 2)
  - [ ] 2.1: Match imported items to existing by: TMDb ID > file path > title+year
  - [ ] 2.2: For `skip` strategy: don't update existing items
  - [ ] 2.3: For `overwrite` strategy: replace existing with imported data
  - [ ] 2.4: Track counters: added, updated, skipped, errored
  - [ ] 2.5: Wrap in transaction for rollback on critical failures

- [ ] Task 3: Create Import API Endpoints (AC: 1, 2, 3)
  - [ ] 3.1: Create `/apps/api/internal/handlers/import_handler.go`
  - [ ] 3.2: `POST /api/v1/settings/import/preview` → upload file, return preview (item count, conflicts)
  - [ ] 3.3: `POST /api/v1/settings/import` → execute import with conflict strategy
  - [ ] 3.4: `GET /api/v1/settings/import/status` → check import progress
  - [ ] 3.5: Write handler tests (≥70% coverage)

- [ ] Task 4: Create Import UI (AC: 1, 2, 3)
  - [ ] 4.1: Create `/apps/web/src/components/settings/MetadataImport.tsx`
  - [ ] 4.2: File upload area (drag & drop or file picker, accept .json/.yaml/.yml)
  - [ ] 4.3: Preview screen showing: total items, new items, conflicts
  - [ ] 4.4: Conflict strategy selector (Skip All / Overwrite All)
  - [ ] 4.5: Import progress bar
  - [ ] 4.6: Results summary with counts

- [ ] Task 5: Wire Up & Tests (AC: all)
  - [ ] 5.1: Register services and handlers in `main.go`
  - [ ] 5.2: Write component tests

## Dev Notes

### Architecture Requirements

**FR61: Import metadata from JSON/YAML**
- Supports incremental import (merge)
- Validates file format before processing

### Import Config

```go
type ImportConfig struct {
    FilePath         string `json:"filePath"`
    ConflictStrategy string `json:"conflictStrategy"` // "skip", "overwrite"
}

type ImportResult struct {
    TotalItems   int      `json:"totalItems"`
    Added        int      `json:"added"`
    Updated      int      `json:"updated"`
    Skipped      int      `json:"skipped"`
    Errors       int      `json:"errors"`
    ErrorDetails []string `json:"errorDetails,omitempty"`
}
```

### Import Preview

```json
// POST /api/v1/settings/import/preview
{
  "success": true,
  "data": {
    "totalItems": 500,
    "newItems": 350,
    "conflicts": 150,
    "formatVersion": "1.0",
    "sourceDate": "2026-02-10T14:30:00Z"
  }
}
```

### API Response Format

```json
// POST /api/v1/settings/import
{
  "success": true,
  "data": {
    "totalItems": 500,
    "added": 350,
    "updated": 100,
    "skipped": 50,
    "errors": 0
  }
}
```

### Error Codes

- `IMPORT_FILE_INVALID` - File format not recognized or malformed
- `IMPORT_VERSION_UNSUPPORTED` - Export version not supported
- `IMPORT_FAILED` - Import process failed
- `IMPORT_PARTIAL` - Some items failed to import

### Dependencies

- Story 6-9 (Metadata Export) - export format definition
- Story 2-6 (Media Entity) - media repository

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.10]
- [Source: _bmad-output/planning-artifacts/prd.md#FR61]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
