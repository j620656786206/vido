# Story 6.9: Metadata Export (JSON/YAML/NFO)

Status: ready-for-dev

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

- [ ] Task 1: Create Export Service (AC: 1, 2, 3)
  - [ ] 1.1: Create `/apps/api/internal/services/export_service.go` with `ExportServiceInterface`
  - [ ] 1.2: Implement `ExportJSON(ctx) (string, error)` - export all media metadata to JSON
  - [ ] 1.3: Implement `ExportYAML(ctx) (string, error)` - export all media metadata to YAML
  - [ ] 1.4: Implement `ExportNFO(ctx) (*ExportResult, error)` - create .nfo files alongside media
  - [ ] 1.5: Write unit tests (≥80% coverage)

- [ ] Task 2: Implement NFO Format (AC: 3)
  - [ ] 2.1: Create `/apps/api/internal/services/nfo_generator.go`
  - [ ] 2.2: Implement Kodi-compatible movie NFO format (`<movie>` root element)
  - [ ] 2.3: Implement Kodi-compatible TV show NFO format (`<tvshow>` root element)
  - [ ] 2.4: Place .nfo files: `{media_dir}/{filename}.nfo`
  - [ ] 2.5: Include: title, year, plot, genres, directors, actors, TMDb ID, poster URL
  - [ ] 2.6: Write unit tests

- [ ] Task 3: Create Export API Endpoints (AC: 1, 4)
  - [ ] 3.1: Create `/apps/api/internal/handlers/export_handler.go`
  - [ ] 3.2: `POST /api/v1/settings/export` → trigger export (body: `{ "format": "json|yaml|nfo" }`)
  - [ ] 3.3: `GET /api/v1/settings/export/:id/download` → download exported file
  - [ ] 3.4: `GET /api/v1/settings/export/status` → check export progress
  - [ ] 3.5: Write handler tests (≥70% coverage)

- [ ] Task 4: Create Export UI (AC: 1, 4)
  - [ ] 4.1: Create `/apps/web/src/components/settings/MetadataExport.tsx`
  - [ ] 4.2: Format selector with descriptions for each format
  - [ ] 4.3: "Export" button with loading/progress state
  - [ ] 4.4: Download link when export completes

- [ ] Task 5: Wire Up & Tests (AC: all)
  - [ ] 5.1: Register services and handlers in `main.go`
  - [ ] 5.2: Write component tests

## Dev Notes

### Architecture Requirements

**FR60, FR62: Export to JSON/YAML/NFO**
- NFO follows Kodi standard format
- Export runs asynchronously

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
  <director>Lilly Wachowski</director>
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
  "media": [
    {
      "title": "駭客任務",
      "originalTitle": "The Matrix",
      "year": 1999,
      "mediaType": "movie",
      "tmdbId": 603,
      "genres": ["科幻", "動作"],
      "overview": "...",
      "posterUrl": "...",
      "filePath": "/media/movies/The.Matrix.1999.mkv",
      "addedAt": "2026-01-15T10:00:00Z"
    }
  ]
}
```

### Error Codes

- `EXPORT_FORMAT_INVALID` - Unknown export format
- `EXPORT_FAILED` - Export process failed
- `EXPORT_NO_MEDIA` - No media to export

### Dependencies

- Story 2-6 (Media Entity) - media repository
- Media repository for fetching all library items

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.9]
- [Source: _bmad-output/planning-artifacts/prd.md#FR60]
- [Source: _bmad-output/planning-artifacts/prd.md#FR62]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
