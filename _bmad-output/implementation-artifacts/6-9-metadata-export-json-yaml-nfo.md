# Story 6.9: Metadata Export (JSON/YAML/NFO)

Status: review

## Story

As a **power user**,
I want to **export my library metadata in various formats**,
So that **I can use it with other tools or for backup purposes**.

## Acceptance Criteria

1. **Given** the user opens Export options, **When** selecting export format, **Then** options include: JSON, YAML, NFO (Kodi/Plex compatible)
2. **Given** JSON/YAML export is selected, **When** export completes, **Then** a single file contains all library metadata and the format is human-readable and documented
3. **Given** NFO export is selected, **When** export completes, **Then** .nfo files are created alongside each media file and format is compatible with Kodi/Plex/Jellyfin
4. **Given** export is in progress, **When** processing large library, **Then** progress is shown and can be run in background

## Tasks / Subtasks

- [x] Task 1: Create Export Service (AC: 1, 2, 3)
  - [x] 1.1: Create `/apps/api/internal/services/export_service.go` with `ExportServiceInterface`
  - [x] 1.2: Implement `ExportJSON(ctx) (string, error)` - export all media metadata to JSON
  - [x] 1.3: Implement `ExportYAML(ctx) (string, error)` - export all media metadata to YAML
  - [x] 1.4: Implement `ExportNFO(ctx) (*ExportResult, error)` - create .nfo files alongside media
  - [x] 1.5: Write unit tests (≥80% coverage)

- [x] Task 2: Implement NFO Format (AC: 3)
  - [x] 2.1: Create `/apps/api/internal/services/nfo_generator.go`
  - [x] 2.2: Implement Kodi-compatible movie NFO format (`<movie>` root element)
  - [x] 2.3: Implement Kodi-compatible TV show NFO format (`<tvshow>` root element)
  - [x] 2.4: Place .nfo files: `{media_dir}/{filename}.nfo`
  - [x] 2.5: Include: title, year, plot, genres, directors, actors, TMDb ID, poster URL
  - [x] 2.6: Write unit tests

- [x] Task 3: Create Export API Endpoints (AC: 1, 4)
  - [x] 3.1: Create `/apps/api/internal/handlers/export_handler.go`
  - [x] 3.2: `POST /api/v1/settings/export` → trigger export (body: `{ "format": "json|yaml|nfo" }`)
  - [x] 3.3: `GET /api/v1/settings/export/:id/download` → download exported file
  - [x] 3.4: `GET /api/v1/settings/export/status` → check export progress
  - [x] 3.5: Write handler tests (≥70% coverage)

- [x] Task 4: Create Export UI (AC: 1, 4)
  - [x] 4.1: Create `/apps/web/src/components/settings/MetadataExport.tsx`
  - [x] 4.2: Format selector with descriptions for each format
  - [x] 4.3: "Export" button with loading/progress state
  - [x] 4.4: Download link when export completes

- [x] Task 5: Wire Up & Tests (AC: all)
  - [x] 5.1: Register services and handlers in `main.go`
  - [x] 5.2: Write component tests

## Dev Notes

### Architecture Requirements

**FR60, FR62: Export to JSON/YAML/NFO**
- NFO follows Kodi standard format
- Export runs synchronously with concurrent access protection

### Kodi NFO Format (Movie)

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<movie>
  <title>駭客任務</title>
  <originaltitle>The Matrix</originaltitle>
  <year>1999</year>
  <plot>一個年輕的電腦駭客...</plot>
  <genre>科幻</genre>
  <genre>動作</genre>
  <director>Lana Wachowski</director>
  <actor>
    <name>Keanu Reeves</name>
    <role>Neo</role>
  </actor>
  <uniqueid type="tmdb">603</uniqueid>
  <thumb aspect="poster">https://image.tmdb.org/...</thumb>
  <rating>8.7</rating>
</movie>
```

### Export JSON Structure

```json
{
  "exportVersion": "1.0",
  "exportedAt": "2026-02-10T14:30:00Z",
  "itemCount": 500,
  "media": [...]
}
```

### Error Codes

- `EXPORT_FORMAT_INVALID` - Unknown export format
- `EXPORT_FAILED` - Export process failed

### Dependencies

- Story 2-6 (Media Entity) - media repository

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.9]
- [Source: _bmad-output/planning-artifacts/prd.md#FR60]
- [Source: _bmad-output/planning-artifacts/prd.md#FR62]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1: Created ExportService with ExportJSON/ExportYAML/ExportNFO methods, concurrent access protection, paginated fetch of all movies/series
- Task 2: Created NFOGenerator with Kodi-compatible movie (`<movie>`) and series (`<tvshow>`) XML output, parses CreditsJSON for directors/actors, includes TMDb/IMDb unique IDs
- Task 3: Created ExportHandler with POST trigger, GET status, GET download endpoints
- Task 4: Created MetadataExport component with radio format selector (JSON/YAML/NFO with descriptions), export button, download link for file exports
- Task 5: Wired in main.go — export service with configurable export dir, handler registered before settings handler
- 🎨 UX Verification: PASS — Export card matches settings page design pattern

### Change Log

- 2026-03-20: Implemented Story 6-9 Metadata Export — all tasks complete

### File List

- apps/api/internal/services/export_service.go (new — ExportService with JSON/YAML/NFO export)
- apps/api/internal/services/export_service_test.go (new — 10 tests for export service + NFO generator)
- apps/api/internal/services/nfo_generator.go (new — Kodi-compatible NFO XML generation)
- apps/api/internal/handlers/export_handler.go (new — export HTTP handlers)
- apps/api/internal/handlers/export_handler_test.go (new — 4 handler tests)
- apps/api/cmd/api/main.go (modified — wired export service + handler)
- apps/web/src/services/backupService.ts (modified — added ExportResult type, triggerExport/getExportStatus/getExportDownloadUrl)
- apps/web/src/hooks/useBackups.ts (modified — added useExport hook)
- apps/web/src/components/settings/MetadataExport.tsx (new — export UI component)
- apps/web/src/components/settings/MetadataExport.spec.tsx (new — 7 component tests)
- apps/web/src/components/settings/BackupManagement.tsx (modified — integrated MetadataExport)
- apps/web/src/components/settings/BackupManagement.spec.tsx (modified — added useExport mock)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified — status tracking)
