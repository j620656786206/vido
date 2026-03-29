# Story 7b-5: Scanner Library Integration

Status: backlog

## Story

As a **user who has configured media libraries**,
I want to **the scanner to use my library configuration from the database instead of environment variables**,
So that **scanned media is correctly classified by content type (movie/series) based on which library folder it belongs to**.

## Acceptance Criteria

1. **Given** media libraries are configured in the database
   **When** a scan is triggered
   **Then** the scanner iterates through each library's paths
   **And** assigns the library's `content_type` to determine movie vs series classification
   **And** sets `library_id` on each scanned media record

2. **Given** no libraries in the database but `VIDO_MEDIA_DIRS` is set
   **When** the application starts
   **Then** a default library is created from the env var paths with type "movie"
   **And** a deprecation warning is logged
   **And** scanning proceeds normally

3. **Given** a library path is inaccessible during scan
   **When** the scanner encounters it
   **Then** the path status is updated to the appropriate error status
   **And** the scanner continues with other accessible paths (graceful degradation)

4. **Given** the scanner completes
   **When** results are reported
   **Then** scan progress and results include library context (which library each file belongs to)

## Tasks / Subtasks

### Task 1: Modify MediaService (AC: #1, #2)
- [ ] 1.1 Update `apps/api/internal/services/media_service.go`
- [ ] 1.2 Change constructor: `NewMediaService(repo MediaLibraryRepository, fallbackDirs []string)`
- [ ] 1.3 Primary source: read libraries from `MediaLibraryRepository`
- [ ] 1.4 Fallback: if DB empty and `fallbackDirs` provided, create default library records
- [ ] 1.5 Log deprecation warning when fallback is used
- [ ] 1.6 `GetConfiguredDirectories()` returns paths with library context (library_id, content_type)

### Task 2: Modify ScannerService (AC: #1, #3, #4)
- [ ] 2.1 Update `apps/api/internal/services/scanner_service.go`
- [ ] 2.2 Change constructor: accept `MediaLibraryRepository` instead of `[]string`
- [ ] 2.3 `StartScan`: iterate libraries, then paths within each library
- [ ] 2.4 When creating movie/series records, set `library_id` from current library
- [ ] 2.5 Use library `content_type` to determine whether to create movie or series record
- [ ] 2.6 Update path status on access errors (call `repo.UpdatePathStatus`)
- [ ] 2.7 Include library name in scan progress events

### Task 3: Update main.go Initialization (AC: #1, #2)
- [ ] 3.1 Update `apps/api/cmd/api/main.go`
- [ ] 3.2 Create `MediaLibraryRepository` instance
- [ ] 3.3 Pass repository to `MediaService` instead of `cfg.MediaDirs`
- [ ] 3.4 Pass repository to `ScannerService` instead of `cfg.MediaDirs`
- [ ] 3.5 Keep `cfg.MediaDirs` as fallback parameter

### Task 4: Update Config Deprecation (AC: #2)
- [ ] 4.1 Update `apps/api/internal/config/config.go` — add deprecation log for `VIDO_MEDIA_DIRS`
- [ ] 4.2 Keep parsing `VIDO_MEDIA_DIRS` but mark as fallback-only in comments

### Task 5: Write Tests (AC: #1–#4)
- [ ] 5.1 MediaService: test DB-based library reading
- [ ] 5.2 MediaService: test env var fallback creates default library
- [ ] 5.3 ScannerService: test scan with library context (library_id assignment)
- [ ] 5.4 ScannerService: test content_type determines movie vs series
- [ ] 5.5 ScannerService: test graceful degradation on inaccessible paths
- [ ] 5.6 Integration: test full scan flow with libraries from DB

## Dev Notes

- This is the critical integration story — ties everything together
- Depends on: 7b-1 (repository), 7b-2 (API for fallback library creation)
- Follow Rule 13: propagate ALL errors from repository calls
- Follow Rule 14: `MediaService` reuses repository instance (no per-request creation)
- Scanner content_type logic: if library type is "movie" → create movie record; if "series" → create series record
- Existing media without `library_id` should still display (NULL library_id is valid for pre-migration data)
- SSE events: include `library_name` field in `scan_progress` event data for UI display
