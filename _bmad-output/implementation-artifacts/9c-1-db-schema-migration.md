# Story 9c-1: DB Schema Migration — Tech Info Fields & Data Source Priority

Status: ready-for-dev

## Story

As a **developer**,
I want **the database schema extended with technical info columns, series file_size, and metadata source priority constants**,
So that **subsequent stories (NFO reader, FFprobe, badges UI) have the data foundation they need**.

## Acceptance Criteria

1. **Given** the application starts with an existing database
   **When** migration #021 runs
   **Then** `movies` table gains columns: `video_codec TEXT`, `video_resolution TEXT`, `audio_codec TEXT`, `audio_channels INTEGER`, `subtitle_tracks TEXT`, `hdr_format TEXT`
   **And** `series` table gains the same 6 columns plus `file_size INTEGER`

2. **Given** migration #021 completes
   **When** querying existing movie/series records
   **Then** all new columns default to NULL (no data loss, no breakage)

3. **Given** the `MetadataSource` type in `models/movie.go`
   **When** the new constants are added
   **Then** `MetadataSourceNFO = "nfo"` and `MetadataSourceAI = "ai"` are available
   **And** existing constants (tmdb, douban, wikipedia, manual) are unchanged

4. **Given** the `ShouldOverwrite(current, incoming MetadataSource) bool` function
   **When** called with various source combinations
   **Then** it returns true when incoming priority >= current priority
   **And** returns true when current is empty (first data)
   **And** priority order is: manual(100) > nfo(80) > tmdb(60) > douban(50) > wikipedia(40) > ai(20)

5. **Given** the Movie and Series Go models
   **When** the new fields are added
   **Then** JSON serialization uses snake_case (`video_codec`, `audio_channels`, etc.)
   **And** repository INSERT/UPDATE SQL includes all new fields

## Tasks / Subtasks

- [ ] Task 1: Create migration `021_media_tech_info.go` (AC: #1, #2)
  - [ ] 1.1 Create `apps/api/internal/database/migrations/021_media_tech_info.go`
  - [ ] 1.2 ALTER TABLE `movies` ADD COLUMN for 6 columns (video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format)
  - [ ] 1.3 ALTER TABLE `series` ADD COLUMN for 7 columns (same 6 + file_size)
  - [ ] 1.4 Register migration in `registry.go`

- [ ] Task 2: Add MetadataSource constants + ShouldOverwrite (AC: #3, #4)
  - [ ] 2.1 Add `MetadataSourceNFO MetadataSource = "nfo"` and `MetadataSourceAI MetadataSource = "ai"` to `models/movie.go`
  - [ ] 2.2 Add `metadataSourcePriority` map with all 6 sources and their numeric priorities
  - [ ] 2.3 Implement `ShouldOverwrite(current, incoming MetadataSource) bool` function

- [ ] Task 3: Update Movie model (AC: #5)
  - [ ] 3.1 Add fields to `Movie` struct: `VideoCodec`, `VideoResolution`, `AudioCodec`, `AudioChannels`, `SubtitleTracks`, `HDRFormat` — all `NullString`/`NullInt64` with proper `db:` and `json:` tags

- [ ] Task 4: Update Series model (AC: #5)
  - [ ] 4.1 Add same 6 fields + `FileSize NullInt64` to `Series` struct

- [ ] Task 5: Update movie repository SQL (AC: #5)
  - [ ] 5.1 Update `Create()` INSERT to include new columns
  - [ ] 5.2 Update `Update()` SET to include new columns
  - [ ] 5.3 Update `BulkCreate()` INSERT to include new columns
  - [ ] 5.4 Update `Upsert()` INSERT/UPDATE to include new columns
  - [ ] 5.5 Update all scan destinations (FindByID, List, etc.) to include new fields

- [ ] Task 6: Update series repository SQL (AC: #5)
  - [ ] 6.1 Same updates as Task 5 for series repository

- [ ] Task 7: Write tests (AC: #1-5)
  - [ ] 7.1 Migration test: tables altered correctly, NULL defaults, no data loss
  - [ ] 7.2 `ShouldOverwrite()` unit tests: all priority combinations, empty current, same source
  - [ ] 7.3 Model serialization tests: JSON field names are snake_case
  - [ ] 7.4 Repository tests: Create/Update with new fields round-trip correctly

## Dev Notes

### Architecture Compliance

- **Rule 4**: Repository pattern — only repository layer touches SQL
- **Rule 6**: Naming — snake_case for DB columns, PascalCase for Go fields, snake_case for JSON
- **Rule 11**: Interfaces in repository package — update `MovieRepositoryInterface` and `SeriesRepositoryInterface` if method signatures change (they shouldn't — same methods, just more columns)
- **Rule 13**: Error handling — propagate all migration errors
- **Rule 15**: Pre-commit verification — migration + model fields + repository SQL ALL in sync

### Project Structure Notes

- Migration file: `apps/api/internal/database/migrations/021_media_tech_info.go`
- Migration registry: `apps/api/internal/database/migrations/registry.go`
- Movie model: `apps/api/internal/models/movie.go` (lines 19-26 have existing MetadataSource constants)
- Series model: `apps/api/internal/models/series.go`
- Movie repository: `apps/api/internal/repository/movie_repository.go`
- Series repository: `apps/api/internal/repository/series_repository.go`
- Repository interfaces: `apps/api/internal/repository/interfaces.go`

### Critical Implementation Details

- **SQLite ALTER TABLE ADD COLUMN** is O(1) — no table rewrite, safe for large databases
- **subtitle_tracks** is stored as TEXT (JSON string), not a separate table — consistent with existing pattern (credits, production_countries are JSON columns)
- **NullString/NullInt64** — use existing `models.NullString` and `models.NullInt64` types (already used for file_size on Movie)
- **Existing MetadataSource** already has: tmdb, douban, wikipedia, manual — just adding nfo and ai
- **ShouldOverwrite** uses `>=` (not `>`) so same-source updates are allowed (idempotent re-scans)
- **Migration pattern**: Follow `020_create_media_libraries.go` structure exactly

### Existing Migration Pattern Reference

```go
// Follow this pattern from 020:
func init() {
    Register(&Migration{
        Version:     21,
        Description: "Add media tech info columns",
        Up: func(db *sql.DB) error {
            // ALTER TABLE statements here
        },
    })
}
```

### References

- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 2: Database Schema]
- [Source: project-context.md#Rule 6: Naming Conventions]
- [Source: project-context.md#Rule 15: Pre-commit Self-verification]
- [Source: models/movie.go#MetadataSource constants (lines 19-26)]
- [Source: migrations/020_create_media_libraries.go — migration pattern]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
