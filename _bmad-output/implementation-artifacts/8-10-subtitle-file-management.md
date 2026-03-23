# Story 8.10: Subtitle File Management

Status: ready-for-dev

## Story

As a **media collector**,
I want **subtitle files to be properly named, placed alongside my media files, and tracked in the database**,
so that **my media player automatically picks up the correct Traditional Chinese subtitle with a standardized filename**.

## Acceptance Criteria

1. **Given** a subtitle file is ready to be placed,
   **When** the placer processes it,
   **Then** the file is saved as `{media_filename}.zh-Hant.srt` using IETF BCP 47 language tags;
   **And** the output directory is the same directory as the media file.

2. **Given** a media file named `Movie.2024.1080p.mkv`,
   **When** a subtitle is placed,
   **Then** the subtitle file is named `Movie.2024.1080p.zh-Hant.srt`;
   **And** `.ass` subtitles produce `Movie.2024.1080p.zh-Hant.ass`;
   **And** the media file extension is stripped and replaced with the language tag + subtitle extension.

3. **Given** a subtitle file already exists at the target path,
   **When** a new subtitle is placed,
   **Then** the existing file is backed up as `{filename}.zh-Hant.srt.bak` (default behavior);
   **Or** the existing file is overwritten if configured;
   **And** the behavior is controlled by a configurable setting.

4. **Given** a subtitle is successfully placed,
   **When** the database is updated,
   **Then** `subtitle_path` is set to the absolute path of the placed file;
   **And** `subtitle_language` is set to the IETF BCP 47 tag (e.g., "zh-Hant");
   **And** `subtitle_status` is set to `found`;
   **And** `subtitle_search_score` is set to the score from the scorer;
   **And** `subtitle_last_searched` is updated.

5. **Given** a media item is deleted from the library,
   **When** cleanup runs,
   **Then** the associated subtitle file is also deleted from disk;
   **And** any `.bak` backup of the subtitle is also cleaned up;
   **And** the subtitle DB fields are cleared.

6. **Given** subtitle content bytes and a target media file path,
   **When** the placer writes the file,
   **Then** the content is written atomically (write to temp file, then rename);
   **And** file permissions are set to 0644;
   **And** write errors are returned without partial files left behind.

7. **Given** various subtitle formats from providers,
   **When** the extension is determined,
   **Then** `.srt` and `.ass` are the supported output formats;
   **And** the format is detected from the subtitle content or provider metadata;
   **And** unsupported formats return an error.

8. **Given** a language tag from the detector,
   **When** the extension is normalized,
   **Then** `zh-Hant`, `zh-TW` → `.zh-Hant.{ext}`;
   **And** `zh-Hans`, `zh-CN` → `.zh-Hans.{ext}` (pre-conversion);
   **And** the mapping follows IETF BCP 47 standard.

## Tasks / Subtasks

### Task 1: Define Placer Types (AC: #1, #2, #3)
- [ ] 1.1 Create `apps/api/internal/subtitle/placer.go`
- [ ] 1.2 Define `Placer` struct with config: `backupExisting bool`
- [ ] 1.3 Define `PlaceRequest` struct: `MediaFilePath string`, `SubtitleData []byte`, `Language string`, `Format string`, `Score float64`
- [ ] 1.4 Define `PlaceResult` struct: `SubtitlePath string`, `Language string`, `BackupPath string` (empty if no backup)
- [ ] 1.5 Define `NewPlacer(config PlacerConfig) *Placer` constructor

### Task 2: Implement Extension Normalization (AC: #1, #2, #8)
- [ ] 2.1 Create `normalizeLanguageTag(lang string) string` — map zh-TW→zh-Hant, zh-CN→zh-Hans, etc.
- [ ] 2.2 Create `buildSubtitleFilename(mediaPath, langTag, subtitleExt string) string`
- [ ] 2.3 Strip media extension, append `.{langTag}.{subtitleExt}`
- [ ] 2.4 Handle edge cases: media files with multiple dots, no extension

### Task 3: Implement Format Detection (AC: #7)
- [ ] 3.1 Create `detectFormat(data []byte, hintFormat string) (string, error)`
- [ ] 3.2 Check hint format first (from provider metadata)
- [ ] 3.3 Content-based detection: look for `[Script Info]` (ASS) or digit+timestamp pattern (SRT)
- [ ] 3.4 Supported formats: `.srt`, `.ass`
- [ ] 3.5 Return error for unsupported formats

### Task 4: Implement Atomic File Write (AC: #6)
- [ ] 4.1 Create `writeFileAtomic(path string, data []byte, perm os.FileMode) error`
- [ ] 4.2 Write to temp file in same directory (`{path}.tmp.{random}`)
- [ ] 4.3 Set permissions to 0644
- [ ] 4.4 Rename temp file to target path
- [ ] 4.5 Clean up temp file on any error

### Task 5: Implement Backup Logic (AC: #3)
- [ ] 5.1 Create `backupExistingFile(path string) (string, error)` method
- [ ] 5.2 If file exists and backupExisting is true, rename to `{path}.bak`
- [ ] 5.3 If file exists and backupExisting is false, allow overwrite
- [ ] 5.4 If `.bak` already exists, overwrite the backup

### Task 6: Implement Place Method (AC: #1, #2, #4, #6)
- [ ] 6.1 Implement `Place(req PlaceRequest) (*PlaceResult, error)` method
- [ ] 6.2 Validate media file path exists
- [ ] 6.3 Detect or validate subtitle format
- [ ] 6.4 Normalize language tag
- [ ] 6.5 Build target filename
- [ ] 6.6 Backup existing subtitle if present
- [ ] 6.7 Write subtitle file atomically
- [ ] 6.8 Return `PlaceResult` with final path and language

### Task 7: Implement DB Update Helper (AC: #4)
- [ ] 7.1 Create `apps/api/internal/subtitle/manager.go`
- [ ] 7.2 Define `Manager` struct wrapping `Placer` + repository dependencies
- [ ] 7.3 Implement `PlaceAndRecord(ctx, mediaID, mediaType string, req PlaceRequest) error`
- [ ] 7.4 Call `placer.Place()` then `UpdateSubtitleStatus` on appropriate repository
- [ ] 7.5 Set all subtitle fields: path, language, status=found, score, last_searched

### Task 8: Implement Cleanup on Media Deletion (AC: #5)
- [ ] 8.1 Create `Cleanup(mediaFilePath, subtitlePath string) error` method on Manager
- [ ] 8.2 Delete subtitle file at `subtitlePath` if it exists
- [ ] 8.3 Delete `.bak` file if it exists
- [ ] 8.4 Log cleanup actions at `slog.Info` level
- [ ] 8.5 Create `ClearSubtitleFields(ctx, mediaID, mediaType)` to reset DB fields
- [ ] 8.6 Integrate cleanup call in media deletion flow (hook into existing delete handler)

### Task 9: Write Tests (AC: #1–#8)
- [ ] 9.1 Create `apps/api/internal/subtitle/placer_test.go`
- [ ] 9.2 Create `apps/api/internal/subtitle/manager_test.go`
- [ ] 9.3 Test filename generation: `Movie.2024.1080p.mkv` → `Movie.2024.1080p.zh-Hant.srt`
- [ ] 9.4 Test filename with `.ass` format
- [ ] 9.5 Test language tag normalization: zh-TW→zh-Hant, zh-CN→zh-Hans
- [ ] 9.6 Test atomic write: file appears only after successful write
- [ ] 9.7 Test atomic write cleanup: no temp files left on error
- [ ] 9.8 Test backup: existing file is renamed to `.bak`
- [ ] 9.9 Test overwrite mode: existing file is replaced without backup
- [ ] 9.10 Test format detection: SRT content, ASS content, unsupported
- [ ] 9.11 Test cleanup: subtitle + backup files deleted
- [ ] 9.12 Test cleanup: missing files handled gracefully (no error)
- [ ] 9.13 Test PlaceAndRecord: DB fields updated correctly (mock repository)
- [ ] 9.14 Test media file path validation (non-existent media dir)
- [ ] 9.15 Ensure >80% coverage on placer.go and manager.go

## Dev Notes

### Architecture & Patterns
- **Placer** handles pure file operations (no DB) — testable with temp directories
- **Manager** wraps Placer + repository for the full place-and-record flow
- Atomic write pattern (temp + rename) prevents corrupt subtitle files on crash/power loss
- BCP 47 normalization is centralized in `normalizeLanguageTag` — single source of truth for the project
- Format detection is best-effort from content; provider hint takes priority
- Cleanup integration should use a hook/callback pattern rather than direct coupling to delete handlers

### Project Structure Notes
- Placer: `apps/api/internal/subtitle/placer.go` (file operations)
- Manager: `apps/api/internal/subtitle/manager.go` (placer + DB coordination)
- Tests: `apps/api/internal/subtitle/placer_test.go`, `manager_test.go`
- Repository: `UpdateSubtitleStatus` from Story 0-2 in `apps/api/internal/repository/`
- Media deletion: currently in `apps/api/internal/handlers/` and `apps/api/internal/services/movie_service.go`, `series_service.go`

### References
- PRD: P1-013 (Extension normalization — merged into this story)
- Gate 2A decision: Extension normalization uses `.zh-Hant.srt` (IETF BCP 47)
- Story 8-7: Engine calls `placer.Place()` as final pipeline step
- Story 0-2: `UpdateSubtitleStatus` repository method
- Story 0-1: Subtitle DB fields (subtitle_path, subtitle_language, subtitle_status, subtitle_search_score, subtitle_last_searched)
- IETF BCP 47: https://www.rfc-editor.org/info/bcp47

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
