# Story 9.6: Subtitle Availability Status

Status: ready-for-dev

## Story

As a **user**,
I want to **see subtitle availability at a glance in the library and detail pages**,
so that **I know which media has subtitles**.

## Acceptance Criteria

1. **AC1: Library Indicators** — Given the user views the media library, when looking at media cards/list items, then a subtitle indicator shows: green (has zh-TW), yellow (has zh-CN or en only), white/grey (no subtitle).
2. **AC2: Detail Page Subtitle List** — Given the user opens a media detail page, when viewing the subtitle section, then all available subtitles are listed with: language flag/code, format (SRT, ASS), source (OpenSubtitles, Zimuku, Manual).
3. **AC3: Online Availability** — Given subtitles exist online but not downloaded, when viewing the detail page, then "Available online" count is shown and one-click "Download best match" is available.

## Tasks / Subtasks

- [ ] Task 1: Add subtitle status to movie/series list API (AC: 1)
  - [ ] 1.1 Add `subtitle_status` field to movie/series list response: "has_zh_tw", "has_other", "none"
  - [ ] 1.2 Compute from subtitle_files table: JOIN to check for zh-TW first, then any language, then none
  - [ ] 1.3 Use subquery or LEFT JOIN for efficient batch computation (avoid N+1 queries)
  - [ ] 1.4 Cache computed status in Tier 1 (invalidate on subtitle add/delete)
  - [ ] 1.5 Write repository tests for the JOIN query

- [ ] Task 2: Create subtitle list endpoint for detail page (AC: 2)
  - [ ] 2.1 Ensure GET `/api/v1/movies/{id}/subtitles` returns all downloaded subtitles (from 9.1)
  - [ ] 2.2 Response includes: id, language, format, source, status, file_path, created_at
  - [ ] 2.3 Sort by language priority (zh-TW first) using Story 9.2 sorting logic
  - [ ] 2.4 Write handler tests

- [ ] Task 3: Add online availability check (AC: 3)
  - [ ] 3.1 Add GetOnlineAvailability(ctx, movieID, title) method to SubtitleService
  - [ ] 3.2 Check cached search results first (Tier 2 cache, 24h TTL)
  - [ ] 3.3 If no cache: perform lightweight search (title only, no full results)
  - [ ] 3.4 Return count of available online subtitles not yet downloaded
  - [ ] 3.5 Add `online_count` field to subtitle status response
  - [ ] 3.6 Add POST `/api/v1/subtitles/download-best` — one-click download best match using Story 9.2 priority + Story 9.3 download
  - [ ] 3.7 Write unit tests

- [ ] Task 4: Create frontend library subtitle indicators (AC: 1)
  - [ ] 4.1 Update `apps/web/src/components/media/PosterCard.tsx` — add subtitle indicator dot/icon
  - [ ] 4.2 Indicator colors: green (🟢 has zh-TW), yellow (🟡 has zh-CN/en), grey (⚪ none)
  - [ ] 4.3 Use Tailwind CSS classes: `bg-green-500`, `bg-yellow-500`, `bg-gray-400` with small dot
  - [ ] 4.4 Show tooltip on hover with subtitle language info
  - [ ] 4.5 Update MediaGrid list view if applicable
  - [ ] 4.6 Write component tests

- [ ] Task 5: Create frontend subtitle detail section (AC: 2, 3)
  - [ ] 5.1 Create `apps/web/src/components/subtitles/SubtitleStatus.tsx` — full subtitle section for media detail page
  - [ ] 5.2 Display downloaded subtitles list with language flag, format badge, source label
  - [ ] 5.3 Show "Available online: N subtitles" with "Download best match" button
  - [ ] 5.4 Create `apps/web/src/hooks/useSubtitleStatus.ts` — TanStack Query for subtitle list + online count
  - [ ] 5.5 Integrate into media detail page (below existing content)
  - [ ] 5.6 Write component tests

## Dev Notes

### Architecture Compliance

- **Performance**: Use JOIN/subquery for batch subtitle status — avoid N+1 queries in library list.
- **Caching**: Cache subtitle status in Tier 1 (in-memory). Invalidate on any subtitle add/remove/update.
- **Online check**: Cache in Tier 2 (SQLite, 24h TTL) to minimize external API calls.
- **Frontend**: TanStack Query for all server state (Rule 5). Zustand ONLY for UI state.

### Visual Indicator Design

```
Library Card/Grid:
  ┌──────────────┐
  │  [Poster]  ● │  ← Small colored dot (top-right corner)
  │  Title       │
  └──────────────┘

Colors (Tailwind):
  🟢 bg-green-500  — Has Traditional Chinese (zh-TW) subtitle
  🟡 bg-yellow-500 — Has Simplified Chinese or English only
  ⚪ bg-gray-400   — No subtitle

Detail Page Section:
  ## 字幕 (Subtitles)
  ┌─────────────────────────────────────────┐
  │ 🇹🇼 繁體中文 (SRT) — Downloaded from Zimuku    │
  │ 🇨🇳 簡體中文 (ASS) — Manually uploaded         │
  │ 🇺🇸 English (SRT)  — Downloaded from OpenSubtitles │
  ├─────────────────────────────────────────┤
  │ 📡 2 more available online              │
  │    [Download Best Match]                │
  └─────────────────────────────────────────┘
```

### Existing Patterns to Follow

- **Library card**: Modify existing `apps/web/src/components/media/PosterCard.tsx`.
- **Detail page**: Extend existing media detail page layout.
- **Batch queries**: Follow movie_repository.go List method with JOINs for efficient queries.
- **TanStack Query keys**: Follow pattern `['subtitles', 'status', movieId]`.

### Dependencies

- **Depends on Story 9.1**: SubtitleRepository, SubtitleFile model, search service.
- **Depends on Story 9.2**: Priority sorting for "best match" download.
- **Depends on Story 9.3**: Download functionality for "Download best match" action.

### Performance Considerations

- Library list may show 50+ items — subtitle status must be computed in batch, not per-item
- Online availability check should be lazy-loaded (only on detail page, not library)
- Use staleTime in TanStack Query to prevent excessive refetching

### Project Structure Notes

- Modify: `apps/web/src/components/media/PosterCard.tsx` (add indicator)
- Extend: `apps/api/internal/repository/subtitle_repository.go` (batch status query)
- New frontend: `apps/web/src/components/subtitles/SubtitleStatus.tsx`
- New hook: `apps/web/src/hooks/useSubtitleStatus.ts`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 9.6]
- [Source: _bmad-output/planning-artifacts/architecture.md#Caching Strategy]
- [Source: project-context.md#Rule 5 - TanStack Query]
- [Source: project-context.md#Rule 1 - Single Backend Location]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
