# Story 9.2: Traditional Chinese Subtitle Priority

Status: ready-for-dev

## Story

As a **Traditional Chinese user**,
I want **Traditional Chinese subtitles prioritized in search results**,
so that **I see the most relevant results first**.

## Acceptance Criteria

1. **AC1: Language Priority Sorting** — Given subtitle search results are returned, when displaying the list, then zh-TW subtitles appear first, zh-CN second, en third, and other languages follow.
2. **AC2: User-Configurable Priority** — Given user preferences are set, when configuring subtitle language priority, then users can customize the priority order, and preferences persist across sessions.
3. **AC3: Within-Group Sorting** — Given multiple Traditional Chinese subtitles exist, when sorting within the priority group, then higher-rated/more-downloaded subtitles appear first, and format preferences (SRT > ASS) can be configured.

## Tasks / Subtasks

- [ ] Task 1: Add subtitle preference settings to database (AC: 2)
  - [ ] 1.1 Add subtitle preference keys to settings table via SettingsRepositoryInterface: `subtitle_language_priority` (JSON array), `subtitle_preferred_format` (string), `subtitle_min_rating` (int)
  - [ ] 1.2 Set defaults: `["zh-TW","zh-CN","en"]`, `"srt"`, `0`
  - [ ] 1.3 Use existing SettingsService Get/Set pattern (no new migration needed — settings table is key-value)

- [ ] Task 2: Implement language priority sorting in subtitle service (AC: 1, 3)
  - [ ] 2.1 Add SortByPriority(results []SubtitleSearchResult, prefs SubtitlePreferences) method to SubtitleService
  - [ ] 2.2 Primary sort: language priority order (index in preference array)
  - [ ] 2.3 Secondary sort within same language: rating descending, then download_count descending
  - [ ] 2.4 Tertiary sort: format preference (user's preferred format first)
  - [ ] 2.5 Write comprehensive unit tests for sorting logic with edge cases

- [ ] Task 3: Create subtitle preferences API endpoints (AC: 2)
  - [ ] 3.1 Add GET `/api/v1/settings/subtitle-preferences` to return current preferences
  - [ ] 3.2 Add PUT `/api/v1/settings/subtitle-preferences` to update preferences
  - [ ] 3.3 Request body: `{language_priority: string[], preferred_format: string, min_rating: number}`
  - [ ] 3.4 Validate language codes, format values, rating range (0-10)
  - [ ] 3.5 Use existing settings handler or add to subtitle handler
  - [ ] 3.6 Write handler tests

- [ ] Task 4: Update search endpoint to apply priority sorting (AC: 1, 3)
  - [ ] 4.1 Load user subtitle preferences before returning search results
  - [ ] 4.2 Apply SortByPriority to combined search results from Story 9.1
  - [ ] 4.3 Filter out results below min_rating threshold
  - [ ] 4.4 Add `priority_rank` field to response for frontend highlighting

- [ ] Task 5: Create frontend subtitle preferences UI (AC: 2)
  - [ ] 5.1 Create `apps/web/src/components/subtitles/SubtitlePreferences.tsx` — language priority reorder list + format dropdown + min rating slider
  - [ ] 5.2 Create `apps/web/src/hooks/useSubtitlePreferences.ts` — TanStack Query for GET/PUT preferences
  - [ ] 5.3 Integrate into Settings page
  - [ ] 5.4 Write component tests

- [ ] Task 6: Update SubtitleList to show priority highlighting (AC: 1)
  - [ ] 6.1 In SubtitleList.tsx (from 9.1), highlight zh-TW results with green accent
  - [ ] 6.2 Show language group separators or badges
  - [ ] 6.3 Show format badge (SRT, ASS, etc.)
  - [ ] 6.4 Write component tests

## Dev Notes

### Architecture Compliance

- **Settings storage**: Use existing `settings` table (key-value). No new migration needed. Use `SettingsRepositoryInterface.Get/Set` methods with typed helpers (GetString, SetInt).
- **Sorting logic**: Lives in SubtitleService (business logic layer), NOT in handler or repository.
- **API format**: Standard `{success, data/error}` wrapper (Rule 3).
- **Endpoints**: `/api/v1/settings/subtitle-preferences` (Rule 10).

### Existing Patterns to Follow

- **Settings storage**: Follow `apps/api/internal/repository/settings_repository.go` — Get/Set with JSON serialization for complex values.
- **Settings service**: Follow `apps/api/internal/services/settings_service.go` — typed convenience methods.
- **Frontend preferences**: Follow existing settings page patterns if available, or create new settings section.

### Dependencies

- **Depends on Story 9.1**: Uses SubtitleSearchResult type and search service.
- **Settings table**: Already exists (migration 003). No new migration needed.

### Language Code Standards

```
zh-TW  — Traditional Chinese (highest priority default)
zh-CN  — Simplified Chinese
en     — English
ja     — Japanese
ko     — Korean
```

### Project Structure Notes

- Backend: Extend SubtitleService in `apps/api/internal/services/subtitle_service.go`
- Frontend: New `SubtitlePreferences.tsx` in `apps/web/src/components/subtitles/`
- Settings: Use existing settings infrastructure, no new tables

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.2]
- [Source: _bmad-output/planning-artifacts/architecture.md#Settings Pattern]
- [Source: project-context.md#Rule 5 - TanStack Query]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
