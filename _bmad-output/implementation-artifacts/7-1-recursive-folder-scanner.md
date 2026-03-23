# Story 7.1: Recursive Folder Scanner

Status: ready-for-dev

## Story

As a **NAS media collector**,
I want **Vido to recursively scan my configured media library folders and detect all video files**,
so that **my media collection is automatically discovered and registered for metadata parsing and subtitle search**.

## Acceptance Criteria

1. **Given** media library paths are configured (VIDO_MEDIA_DIRS), **When** a scan is triggered via POST /api/v1/scanner/scan, **Then** the scanner recursively walks all configured paths and detects video files (mkv, mp4, avi, rmvb)
2. **Given** symlinks exist in the media library, **When** scanning encounters a symlink, **Then** it follows the symlink but deduplicates by resolved absolute file path (same physical file enters DB only once)
3. **Given** video files are detected, **When** creating scan records, **Then** each file is inserted into the database with ParseStatus="pending", resolved absolute FilePath, and FileSize
4. **Given** a scan is already in progress, **When** another scan is triggered, **Then** the API returns error code SCANNER_ALREADY_RUNNING with HTTP 409
5. **Given** a directory is not accessible (permission denied), **When** the scanner encounters it, **Then** it logs a warning with SCANNER_PERMISSION_DENIED and continues scanning other paths
6. **Given** 1,000 video files exist, **When** a full scan is triggered, **Then** the scan completes in less than 1 minute (filesystem walk + DB insert)
7. **Given** a scan is in progress, **When** SSE clients are connected to /api/v1/events, **Then** they receive scan_progress events with filesFound, currentFile, percentDone, and errorCount
8. **Given** a file was already scanned (exists in DB by FilePath), **When** a re-scan detects it, **Then** the existing record is preserved (no duplicate created), and file metadata (size) is updated if changed

## Tasks / Subtasks

- [ ] Task 1: Create ScannerService with filesystem walk (AC: 1, 2, 5, 6)
  - [ ] 1.1: Create `apps/api/internal/services/scanner_service.go` with `ScannerService` struct
  - [ ] 1.2: Implement `StartScan(ctx context.Context) (*ScanResult, error)` — recursive filepath.WalkDir across all MediaDirs
  - [ ] 1.3: Implement video file detection (case-insensitive extension check: .mkv, .mp4, .avi, .rmvb)
  - [ ] 1.4: Implement symlink resolution via `filepath.EvalSymlinks` + deduplication by resolved absolute path
  - [ ] 1.5: Implement permission error handling (log SCANNER_PERMISSION_DENIED, continue scanning)
  - [ ] 1.6: Write unit tests for filesystem walk logic using `t.TempDir()` (≥80% coverage)

- [ ] Task 2: Implement concurrency control with mutex (AC: 4)
  - [ ] 2.1: Add `sync.Mutex` + `isScanning bool` + `cancelChan chan struct{}` to ScannerService
  - [ ] 2.2: Implement `IsScanActive() bool` for status checking
  - [ ] 2.3: Implement `CancelScan()` for scan cancellation
  - [ ] 2.4: Write tests for concurrent scan prevention and cancellation

- [ ] Task 3: Implement database record creation (AC: 3, 8)
  - [ ] 3.1: Create scan records with ParseStatus=pending, resolved FilePath, FileSize
  - [ ] 3.2: Use `FindByFilePath()` for duplicate detection before insert
  - [ ] 3.3: Use `BulkCreate()` for batch insertion in single transaction
  - [ ] 3.4: Handle re-scan: update FileSize if file changed, skip if unchanged
  - [ ] 3.5: Write tests for duplicate detection and bulk create logic

- [ ] Task 4: Implement SSE progress broadcasting (AC: 7)
  - [ ] 4.1: Accept `*sse.Hub` dependency in ScannerService constructor
  - [ ] 4.2: Define `ScanProgressEvent` struct (FilesFound, CurrentFile, PercentDone, ErrorCount)
  - [ ] 4.3: Broadcast scan_progress events during walk (every 10 files or every 1 second)
  - [ ] 4.4: Broadcast final completion/error event
  - [ ] 4.5: Write tests verifying SSE events are broadcast

- [ ] Task 5: Create ScannerHandler with HTTP endpoints (AC: 1, 4)
  - [ ] 5.1: Create `apps/api/internal/handlers/scanner_handler.go` with `ScannerHandler` struct
  - [ ] 5.2: Implement `POST /api/v1/scanner/scan` — trigger scan (returns 202 Accepted or 409 Conflict)
  - [ ] 5.3: Implement `GET /api/v1/scanner/status` — return current scan progress or last result
  - [ ] 5.4: Implement `POST /api/v1/scanner/cancel` — cancel active scan
  - [ ] 5.5: Use ApiResponse format for all responses (project-context.md Rule 3)
  - [ ] 5.6: Write handler tests (≥70% coverage)

- [ ] Task 6: Wire into main.go and verify (AC: all)
  - [ ] 6.1: Initialize ScannerService with repos, settingsService, sseHub in main.go
  - [ ] 6.2: Initialize ScannerHandler and register routes on apiV1
  - [ ] 6.3: Run `go build ./cmd/api/` — verify build passes
  - [ ] 6.4: Run `go test ./...` — verify all tests pass
  - [ ] 6.5: Manual verification: start server, trigger scan, observe SSE events

## Dev Notes

### Gate 2A Decisions (Mandatory)
- **Symlinks:** Follow symlinks, deduplicate by resolved absolute file path
- **Concurrency:** Mutex lock — only one scan at a time
- **Incremental scan:** Use file mtime for change detection (Story 7-2 will implement)
- **Video formats:** .mkv, .mp4, .avi, .rmvb (case-insensitive)

### Phase 0 Infrastructure Available
- `BulkCreate()` method on MovieRepository and SeriesRepository
- `FindByFilePath()` for duplicate detection
- `FindByParseStatus()` for finding pending items
- SSE Hub at `apps/api/internal/sse/` wired to `GET /api/v1/events`
- `ParseStatus` enum: pending, parsing, success, needs_ai, failed
- `SubtitleStatus` enum: not_searched, searching, found, not_found

### Scope Boundaries — DO NOT Implement
- ❌ Filename parsing (Stories 2-3, 3-1 already done)
- ❌ TMDB matching (Story 2-4 already done)
- ❌ Subtitle search (Epic 8)
- ❌ Scheduled/cron scanning (Story 7-2)
- ❌ Scan UI buttons (Story 7-3)
- ❌ Scan progress UI (Story 7-4)

### Service Pattern to Follow
```
Handler → Service → Repository → Database
```
- ScannerService depends on: repos.Movies, repos.Series, config.MediaDirs, *sse.Hub
- ScannerHandler depends on: *ScannerService
- Reference implementation: `services/export_service.go` for similar bulk operation patterns

### Error Codes
- `SCANNER_PERMISSION_DENIED` — Cannot access path
- `SCANNER_PATH_NOT_FOUND` — Configured path doesn't exist
- `SCANNER_ALREADY_RUNNING` — Scan already in progress
- `SCANNER_PARSE_FAILED` — Failed to scan directory

### Project Structure Notes

- Scanner service: `apps/api/internal/services/scanner_service.go`
- Scanner handler: `apps/api/internal/handlers/scanner_handler.go`
- Tests co-located: `*_test.go` in same directory
- All logging: `log/slog` (NOT zerolog, NOT fmt.Println)
- API responses: `{success: true/false, data/error}` format

### References

- [Source: project-context.md] — Mandatory rules (logging, error codes, layered architecture)
- [Source: architecture/core-architectural-decisions.md#Decision-8] — SSE Hub architecture
- [Source: architecture/phase0-prerequisites-spec.md] — BulkCreate, FindByFilePath, SSE Hub
- [Source: epic-7-media-library-scanner.md] — Epic scope, success criteria, Gate 2A decisions
- [Source: prd/functional-requirements.md#P1-001] — Folder scanning requirements
- [Source: scanner-ui-design-brief.md] — UI context (for endpoint design)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
