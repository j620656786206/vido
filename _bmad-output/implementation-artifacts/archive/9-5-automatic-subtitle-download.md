# Story 9.5: Automatic Subtitle Download

Status: ready-for-dev

## Story

As a **user**,
I want **subtitles to download automatically when new media is added**,
so that **I don't have to search manually every time**.

## Acceptance Criteria

1. **AC1: Auto-Download Settings** — Given the user enables "Auto-download subtitles" in Settings, when configuring preferences, then they can select: preferred languages (ordered list), minimum rating threshold, preferred format (SRT/ASS/Any).
2. **AC2: Automatic Trigger** — Given auto-download is enabled, when new media is added to the library, then the system automatically searches for subtitles and downloads the best match based on preferences.
3. **AC3: Auto-Download Status** — Given a subtitle is auto-downloaded, when viewing the media detail, then status shows "Auto-downloaded" and the user can reject and search for alternatives.
4. **AC4: Failure Handling** — Given no suitable subtitle is found, when auto-download fails, then the media shows "No subtitle found" and manual search remains available.

## Tasks / Subtasks

- [ ] Task 1: Add auto-download settings (AC: 1)
  - [ ] 1.1 Add settings keys via SettingsService: `subtitle_auto_download_enabled` (bool, default false), `subtitle_auto_download_languages` (JSON array), `subtitle_auto_download_min_rating` (int), `subtitle_auto_download_format` (string)
  - [ ] 1.2 Add GET/PUT `/api/v1/settings/subtitle-auto-download` endpoints
  - [ ] 1.3 Validate settings values (language codes, rating 0-10, format in allowed list)
  - [ ] 1.4 Write handler tests

- [ ] Task 2: Create auto-download subtitle task (AC: 2, 3, 4)
  - [ ] 2.1 Create `apps/api/internal/subtitle/auto_download_task.go` with AutoDownloadSubtitleTask struct
  - [ ] 2.2 Implement Execute(ctx) method: load preferences → search (OpenSubtitles + Zimuku) → filter by preferences → download best match
  - [ ] 2.3 Best match logic: first result matching language priority + above min_rating + preferred format
  - [ ] 2.4 On success: save file via SubtitleFileManager, create DB record with source="auto"
  - [ ] 2.5 On failure: log with slog.Warn, set subtitle_status="unavailable" for the movie
  - [ ] 2.6 Respect rate limits: queue requests, don't blast APIs
  - [ ] 2.7 Write unit tests

- [ ] Task 3: Integrate with media add event (AC: 2)
  - [ ] 3.1 Hook into existing media creation flow — when a new movie/series is added, check if auto-download is enabled
  - [ ] 3.2 If enabled, submit AutoDownloadSubtitleTask to background task queue
  - [ ] 3.3 Use existing task queue infrastructure (`apps/api/internal/tasks/` or `apps/api/internal/retry/`)
  - [ ] 3.4 Task includes: movie_id, title, language preferences
  - [ ] 3.5 Retry with exponential backoff: 1s → 2s → 4s → 8s (ARCH-4 pattern)
  - [ ] 3.6 Write integration tests

- [ ] Task 4: Add auto-download status tracking (AC: 3, 4)
  - [ ] 4.1 Add `auto_download_status` field to subtitle tracking: "pending", "downloading", "completed", "failed", "rejected"
  - [ ] 4.2 Add GET `/api/v1/movies/{id}/subtitle-status` endpoint for auto-download status
  - [ ] 4.3 Add POST `/api/v1/subtitles/{id}/reject` endpoint — marks auto-downloaded subtitle as rejected, triggers manual search
  - [ ] 4.4 Write handler tests

- [ ] Task 5: Create frontend auto-download settings UI (AC: 1)
  - [ ] 5.1 Create `apps/web/src/components/subtitles/AutoDownloadSettings.tsx` — toggle switch + language reorder + rating slider + format dropdown
  - [ ] 5.2 Create `apps/web/src/hooks/useAutoDownloadSettings.ts` — TanStack Query for settings CRUD
  - [ ] 5.3 Integrate into Settings page (subtitle section)
  - [ ] 5.4 Write component tests

- [ ] Task 6: Update media detail for auto-download status (AC: 3, 4)
  - [ ] 6.1 Show "Auto-downloaded" badge on subtitles with source="auto"
  - [ ] 6.2 Add "Reject & Search Manually" action for auto-downloaded subtitles
  - [ ] 6.3 Show "No subtitle found — Search manually" when auto-download failed
  - [ ] 6.4 Show "Searching for subtitles..." spinner when task is pending/downloading
  - [ ] 6.5 Use TanStack Query polling (refetchInterval: 5000) for active auto-download tasks
  - [ ] 6.6 Write component tests

## Dev Notes

### Architecture Compliance

- **Background tasks**: Use existing worker pool (3-5 goroutines) from ARCH-4. Goroutines + channels, NO external queue.
- **Retry**: Exponential backoff 1s → 2s → 4s → 8s.
- **Rate limiting**: Respect per-source limits (OpenSubtitles plan-dependent, Zimuku 1 req/2s).
- **Settings**: Use existing settings key-value table (no migration needed).
- **Error codes**: SUBTITLE_AUTO_DOWNLOAD_FAILED, SUBTITLE_AUTO_DOWNLOAD_DISABLED.

### Existing Patterns to Follow

- **Background tasks**: Follow existing retry queue at `apps/api/internal/retry/` — retry_executor.go, retry_queue.go.
- **Event-driven**: Follow event emitter pattern at `apps/api/internal/events/` for triggering on media add.
- **Settings**: Follow existing subtitle preferences pattern from Story 9.2.

### Task Queue Design

```
Media Added Event
  → Check settings: subtitle_auto_download_enabled
  → If enabled: Submit AutoDownloadSubtitleTask {movie_id, title, prefs}
  → Worker picks up task
  → Search OpenSubtitles + Zimuku (reuse Story 9.1 service)
  → Filter by preferences (reuse Story 9.2 sorting)
  → Download best match (reuse Story 9.3 download)
  → Save result and update status
```

### Dependencies

- **Depends on Story 9.1**: SubtitleService.Search for subtitle lookup.
- **Depends on Story 9.2**: SortByPriority for best match selection.
- **Depends on Story 9.3**: SubtitleFileManager + Download for file saving.
- **Background task queue**: Must exist (ARCH-4). Check if `apps/api/internal/tasks/` or use retry infrastructure.

### Project Structure Notes

- New file: `apps/api/internal/subtitle/auto_download_task.go`
- Extend: `apps/api/internal/services/subtitle_service.go`
- Extend: `apps/api/internal/handlers/subtitle_handler.go`
- New frontend: `apps/web/src/components/subtitles/AutoDownloadSettings.tsx`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.5]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-4 Background Tasks]
- [Source: _bmad-output/planning-artifacts/architecture.md#Retry Strategy]
- [Source: project-context.md#Rule 4 - Layered Architecture]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
