# Story 9.3: Subtitle Download

Status: ready-for-dev

## Story

As a **user**,
I want to **download subtitles directly from search results**,
so that **I can use them with my media player**.

## Acceptance Criteria

1. **AC1: Download Action** — Given subtitle search results are displayed, when the user clicks "Download" on a result, then the subtitle file is downloaded and saved to the same folder as the media file.
2. **AC2: Naming Convention** — Given the subtitle is downloaded, when naming the file, then it matches the media filename with language suffix. Format: `MediaName.zh-TW.srt` or `MediaName.zh-CN.ass`.
3. **AC3: Conflict Handling** — Given a subtitle already exists for that language, when downloading another, then user is prompted: "Replace existing or keep both?", and "Keep both" adds a suffix: `.v2.srt`.
4. **AC4: Status Update** — Given download succeeds, when viewing the media detail page, then the subtitle appears in the "Available Subtitles" list and status shows "Downloaded from [Source]".

## Tasks / Subtasks

- [ ] Task 1: Implement subtitle file download in OpenSubtitles client (AC: 1)
  - [ ] 1.1 Add Download(ctx, downloadURL) ([]byte, error) method to OpenSubtitlesClient
  - [ ] 1.2 Handle compressed responses (zip/gzip) — extract subtitle file from archive
  - [ ] 1.3 Add timeout and retry logic
  - [ ] 1.4 Write unit tests with mock HTTP responses

- [ ] Task 2: Implement subtitle file download in Zimuku scraper (AC: 1)
  - [ ] 2.1 Add Download(ctx, downloadURL) ([]byte, error) method to ZimukuScraper
  - [ ] 2.2 Handle compressed responses and anti-scraping measures
  - [ ] 2.3 Write unit tests

- [ ] Task 3: Create subtitle file manager (AC: 1, 2, 3)
  - [ ] 3.1 Create `apps/api/internal/subtitle/file_manager.go` with SubtitleFileManager struct
  - [ ] 3.2 Implement SaveSubtitle(ctx, mediaFilePath, subtitleBytes, language, format) (string, error)
  - [ ] 3.3 Generate filename: extract media basename, append `.{lang}.{format}` — e.g., `Movie Title.zh-TW.srt`
  - [ ] 3.4 Implement conflict detection: check if file exists at target path
  - [ ] 3.5 Implement versioning: if conflict + keep-both, find next available `.vN.{format}` suffix
  - [ ] 3.6 Verify write permissions to media folder before writing
  - [ ] 3.7 Write unit tests for naming, conflict detection, versioning

- [ ] Task 4: Add download methods to subtitle service (AC: 1, 2, 3, 4)
  - [ ] 4.1 Add Download(ctx, movieID, subtitleResult, conflictAction) method to SubtitleService
  - [ ] 4.2 Resolve media file path from movie/series record in database
  - [ ] 4.3 Call appropriate client (OpenSubtitles or Zimuku) Download method based on source
  - [ ] 4.4 Delegate to SubtitleFileManager for file saving
  - [ ] 4.5 Create SubtitleFile record in database with status "downloaded", source info
  - [ ] 4.6 Invalidate subtitle cache for this movie
  - [ ] 4.7 Write unit tests (>80% coverage)

- [ ] Task 5: Create download API endpoint (AC: 1, 3)
  - [ ] 5.1 Add POST `/api/v1/subtitles/download` endpoint to subtitle handler
  - [ ] 5.2 Request body: `{movie_id, download_url, source, language, format, conflict_action: "replace"|"keep_both"}`
  - [ ] 5.3 Return downloaded subtitle file metadata
  - [ ] 5.4 Error codes: SUBTITLE_FILE_WRITE_ERROR, SUBTITLE_PERMISSION_DENIED, SUBTITLE_DUPLICATE_FOUND
  - [ ] 5.5 Write handler tests

- [ ] Task 6: Create frontend download UI (AC: 1, 3, 4)
  - [ ] 6.1 Add download button to each SubtitleList item (from 9.1)
  - [ ] 6.2 Create `apps/web/src/components/subtitles/SubtitleDownload.tsx` — download action + conflict dialog
  - [ ] 6.3 Create `apps/web/src/hooks/useSubtitleDownload.ts` — TanStack Query mutation for POST download
  - [ ] 6.4 Show conflict prompt modal when SUBTITLE_DUPLICATE_FOUND returned
  - [ ] 6.5 Invalidate subtitle list query on successful download (queryClient.invalidateQueries)
  - [ ] 6.6 Show toast notification on success/failure
  - [ ] 6.7 Write component tests

## Dev Notes

### Architecture Compliance

- **File operations**: Subtitle files saved to same directory as media file. Respect media folder permissions.
- **Error codes**: SUBTITLE_FILE_WRITE_ERROR, SUBTITLE_PERMISSION_DENIED, SUBTITLE_DUPLICATE_FOUND (Rule 7).
- **API**: POST `/api/v1/subtitles/download` (Rule 10).
- **Layered**: Handler → SubtitleService → SubtitleFileManager + Repository (Rule 4).

### File Naming Convention

```
Pattern: {MediaBaseName}.{language_code}.{format}
Examples:
  Movie Title (2024).zh-TW.srt
  Movie Title (2024).zh-CN.ass
  Movie Title (2024).en.vtt

Versioning (conflict - keep both):
  Movie Title (2024).zh-TW.v2.srt
  Movie Title (2024).zh-TW.v3.srt
```

### Existing Patterns to Follow

- **File path resolution**: Media file paths stored in movies/series tables. Use repository to fetch.
- **Archive handling**: May need `archive/zip` stdlib for compressed subtitle downloads.
- **Error wrapping**: `fmt.Errorf("failed to save subtitle: %w", err)` pattern.

### Dependencies

- **Depends on Story 9.1**: Uses subtitle search results, OpenSubtitles/Zimuku clients, SubtitleService.
- **Media file paths**: Requires movies/series records to have file_path stored.

### Security Considerations

- Validate file write paths to prevent path traversal attacks
- Only write to configured media directories
- Sanitize downloaded filenames

### Project Structure Notes

- New file: `apps/api/internal/subtitle/file_manager.go`
- Extend: `apps/api/internal/services/subtitle_service.go`
- Extend: `apps/api/internal/handlers/subtitle_handler.go`
- New frontend: `apps/web/src/components/subtitles/SubtitleDownload.tsx`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.3]
- [Source: _bmad-output/planning-artifacts/architecture.md#File Handling]
- [Source: project-context.md#Rule 3 - API Response Format]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
