# Story 9.4: Manual Subtitle Upload

Status: ready-for-dev

## Story

As a **user**,
I want to **upload my own subtitle files**,
so that **I can use subtitles from other sources**.

## Acceptance Criteria

1. **AC1: Upload Dialog** — Given the user opens a media detail page, when clicking "Upload Subtitle", then a file picker dialog opens and accepts: .srt, .ass, .ssa, .sub, .vtt formats.
2. **AC2: Language Selection & Save** — Given a subtitle file is selected, when uploading, then the user can select the language from a dropdown, and the file is copied to the media folder.
3. **AC3: Upload Status** — Given the upload succeeds, when the subtitle is saved, then it appears in the "Available Subtitles" list and status shows "Manually uploaded".
4. **AC4: Edit/Delete** — Given the subtitle needs editing, when the user clicks "Rename/Delete", then they can change the language tag and delete the subtitle file.

## Tasks / Subtasks

- [ ] Task 1: Create subtitle upload endpoint (AC: 1, 2, 3)
  - [ ] 1.1 Add POST `/api/v1/subtitles/upload` multipart form endpoint to subtitle handler
  - [ ] 1.2 Accept multipart file upload with fields: `file` (binary), `movie_id` (string), `language` (string)
  - [ ] 1.3 Validate file extension against allowed formats: .srt, .ass, .ssa, .sub, .vtt
  - [ ] 1.4 Validate file size (max 5MB — reasonable for subtitle files)
  - [ ] 1.5 Call SubtitleService.Upload for processing
  - [ ] 1.6 Return uploaded subtitle metadata
  - [ ] 1.7 Write handler tests

- [ ] Task 2: Implement upload logic in subtitle service (AC: 2, 3)
  - [ ] 2.1 Add Upload(ctx, movieID, fileBytes, filename, language) method to SubtitleService
  - [ ] 2.2 Detect file encoding using charset detection (e.g., `golang.org/x/net/html/charset` or `github.com/saintfish/chardet`)
  - [ ] 2.3 Convert to UTF-8 if not already UTF-8 encoded
  - [ ] 2.4 Delegate file saving to SubtitleFileManager (from 9.3) with source="manual"
  - [ ] 2.5 Create SubtitleFile record in database with status "downloaded", source "manual"
  - [ ] 2.6 Write unit tests (>80% coverage)

- [ ] Task 3: Create subtitle update/delete endpoints (AC: 4)
  - [ ] 3.1 Add PUT `/api/v1/subtitles/{id}` to update language tag
  - [ ] 3.2 Implement rename: change language suffix in filename, update database record
  - [ ] 3.3 Add DELETE `/api/v1/subtitles/{id}` to delete subtitle file
  - [ ] 3.4 Implement delete: remove file from disk, remove database record
  - [ ] 3.5 Error codes: SUBTITLE_FORMAT_INVALID, SUBTITLE_ENCODING_ERROR
  - [ ] 3.6 Write handler tests

- [ ] Task 4: Add update/delete methods to subtitle service (AC: 4)
  - [ ] 4.1 Add UpdateLanguage(ctx, subtitleID, newLanguage) method
  - [ ] 4.2 Rename physical file: change language code in filename
  - [ ] 4.3 Add Delete(ctx, subtitleID) method
  - [ ] 4.4 Remove physical file and database record
  - [ ] 4.5 Invalidate caches on update/delete
  - [ ] 4.6 Write unit tests

- [ ] Task 5: Create frontend upload UI (AC: 1, 2, 3)
  - [ ] 5.1 Create `apps/web/src/components/subtitles/SubtitleUpload.tsx` — file picker + language dropdown + upload button
  - [ ] 5.2 Create `apps/web/src/hooks/useSubtitleUpload.ts` — TanStack Query mutation for POST upload
  - [ ] 5.3 Validate file extension client-side before upload
  - [ ] 5.4 Show upload progress indicator
  - [ ] 5.5 Invalidate subtitle list query on successful upload
  - [ ] 5.6 Show toast on success/failure
  - [ ] 5.7 Write component tests

- [ ] Task 6: Create frontend edit/delete UI (AC: 4)
  - [ ] 6.1 Add edit/delete actions to each subtitle item in SubtitleList
  - [ ] 6.2 Create inline language tag editor (dropdown)
  - [ ] 6.3 Add delete confirmation dialog
  - [ ] 6.4 Use TanStack Query mutations for PUT/DELETE operations
  - [ ] 6.5 Write component tests

## Dev Notes

### Architecture Compliance

- **File upload**: Use Gin's multipart form handling `c.FormFile("file")`.
- **Encoding detection**: Use charset detection library. Convert all uploaded files to UTF-8.
- **Layered**: Handler (validates + extracts file) → Service (encoding + business logic) → FileManager (disk ops) + Repository (DB).
- **API**: POST `/api/v1/subtitles/upload`, PUT/DELETE `/api/v1/subtitles/{id}` (Rule 10).

### Supported Formats

```
.srt  — SubRip (most common)
.ass  — Advanced SubStation Alpha
.ssa  — SubStation Alpha
.sub  — MicroDVD
.vtt  — WebVTT
```

### Encoding Handling

- Detect encoding of uploaded file (common: UTF-8, BIG5, GB2312, GB18030, EUC-JP)
- Convert to UTF-8 for consistency
- BIG5 is common for Traditional Chinese subtitle files
- GB2312/GB18030 common for Simplified Chinese

### Existing Patterns to Follow

- **Multipart upload**: Gin `c.FormFile` for file extraction, `c.PostForm` for fields.
- **File operations**: Reuse SubtitleFileManager from Story 9.3.
- **CRUD pattern**: Follow movie handler for PUT/DELETE patterns.

### Dependencies

- **Depends on Story 9.1**: Uses SubtitleFile model, SubtitleRepository.
- **Depends on Story 9.3**: Uses SubtitleFileManager for file saving.

### Security Considerations

- Validate file content, not just extension (check for magic bytes if possible)
- Limit file size to prevent abuse (5MB max)
- Sanitize filename to prevent injection
- Validate language code against known list

### Project Structure Notes

- Extend: `apps/api/internal/handlers/subtitle_handler.go`
- Extend: `apps/api/internal/services/subtitle_service.go`
- New frontend: `apps/web/src/components/subtitles/SubtitleUpload.tsx`
- Go dependency: charset detection library

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.4]
- [Source: _bmad-output/planning-artifacts/architecture.md#File Handling]
- [Source: project-context.md#Rule 6 - Naming Conventions]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
