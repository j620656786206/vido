# Story 12.2: Season/Episode List — Expandable Accordion with Subtitle Status

Status: ready-for-dev

## Story

As a media library user viewing a TV series detail page,
I want to see an expandable season accordion with episode lists showing titles, air dates, and subtitle status per episode,
so that I can quickly browse episode details and know which episodes have subtitles available.

## Acceptance Criteria

1. **Given** a TV series with `tmdb_id > 0`, **when** the detail page loads, **then** a season accordion is displayed below the series metadata, with one collapsible section per season.
2. **Given** a season section is collapsed (default), **then** it shows: season poster thumbnail, season name (e.g., "第 1 季"), episode count, and air date.
3. **Given** a user clicks/taps a season header, **when** the accordion expands, **then** it fetches and displays the episode list for that season from the TMDB Season Details API.
4. **Given** an expanded season, **then** each episode row shows: episode number (S01E05 format), title, air date, runtime, and a subtitle status indicator icon.
5. **Given** an episode has a local file (`file_path` is set in the `episodes` table), **then** the subtitle status indicator reflects the episode's `subtitle_status` field (found=green, not_found=red, not_searched=gray, searching=amber spinner).
6. **Given** an episode has NO local file, **then** no subtitle indicator is shown (episode not in library).
7. **Given** the TMDB Season Details API is unavailable, **when** expansion is attempted, **then** a retry-able error message is shown inside the accordion body (graceful degradation).
8. **Given** a mobile viewport, **when** viewing the episode list, **then** episode rows stack with title/date on one line and metadata below, maintaining readability.

## Tasks / Subtasks

### Backend

- [ ] **Task 1: DB Migration 023 — Add episode-level subtitle tracking** (AC: #5)
  - [ ] 1.1 Create `023_add_episode_subtitle_fields.go` in `apps/api/internal/database/migrations/`
  - [ ] 1.2 Add to `episodes` table: `subtitle_status TEXT DEFAULT 'not_searched'`, `subtitle_path TEXT`, `subtitle_language TEXT`
  - [ ] 1.3 Create index: `idx_episodes_subtitle_status`
  - [ ] 1.4 Register migration in `registry.go`

- [ ] **Task 2: Extend Episode model with subtitle fields** (AC: #5)
  - [ ] 2.1 Add to `Episode` struct in `models/episode.go`: `SubtitleStatus SubtitleStatus`, `SubtitlePath NullString`, `SubtitleLanguage NullString`
  - [ ] 2.2 Add db/json tags matching convention

- [ ] **Task 3: TMDB GetSeasonDetails endpoint** (AC: #3)
  - [ ] 3.1 Add `GetSeasonDetails(ctx, tvID int, seasonNumber int) (*SeasonDetails, error)` to `tmdb/tv.go`
  - [ ] 3.2 Add `SeasonDetails` type to `tmdb/types.go` with `Episodes []EpisodeInfo` (TMDB episode data: number, title, overview, air_date, runtime, still_path, vote_average)
  - [ ] 3.3 Add `EpisodeInfo` type to `tmdb/types.go`
  - [ ] 3.4 Use `zh-TW` language parameter, cache response (24h TTL via existing cache layer)

- [ ] **Task 4: Season/Episode API endpoints** (AC: #1, #3, #4, #5)
  - [ ] 4.1 Add `GET /api/v1/series/:id/seasons` — returns season list from `SeasonsJSON` on series record (fast, no TMDB call)
  - [ ] 4.2 Add `GET /api/v1/series/:id/seasons/:seasonNumber/episodes` — merges TMDB episode data with local episode records (subtitle status, file_path)
  - [ ] 4.3 Merge logic: for each TMDB episode, LEFT JOIN with local `episodes` table by (series_id, season_number, episode_number) to attach `subtitle_status` and `file_path`
  - [ ] 4.4 Response format: `{ "success": true, "data": { "season": {...}, "episodes": [{...}] } }`
  - [ ] 4.5 Add to series handler, register routes in `RegisterRoutes`
  - [ ] 4.6 Write handler + service tests

- [ ] **Task 5: Episode repository — subtitle queries** (AC: #5)
  - [ ] 5.1 Add `FindBySeriesAndSeason(ctx, seriesID string, seasonNumber int) ([]Episode, error)` to episode repository
  - [ ] 5.2 Update existing episode SELECT queries to include new subtitle columns
  - [ ] 5.3 Add `UpdateEpisodeSubtitleStatus(ctx, episodeID string, status SubtitleStatus, path, language string) error`

### Frontend

- [ ] **Task 6: Season accordion component** (AC: #1, #2, #8)
  - [ ] 6.1 Create `apps/web/src/components/media/SeasonAccordion.tsx`
  - [ ] 6.2 Props: `seasons: SeasonSummary[]`, `seriesId: string`, `tmdbId: number`
  - [ ] 6.3 Collapsed state: poster thumbnail (56×84), season name, episode count badge, air date
  - [ ] 6.4 Click toggles expansion (one-at-a-time or multi, TBD — default: multi-open)
  - [ ] 6.5 Expansion triggers episode list fetch (lazy load)
  - [ ] 6.6 Tailwind responsive: horizontal row on desktop, stacked on mobile
  - [ ] 6.7 Write `SeasonAccordion.spec.tsx`

- [ ] **Task 7: Episode list component** (AC: #4, #5, #6, #8)
  - [ ] 7.1 Create `apps/web/src/components/media/EpisodeList.tsx`
  - [ ] 7.2 Props: `episodes: MergedEpisode[]` (TMDB data + local subtitle status)
  - [ ] 7.3 Each row: episode code (S01E05), title, air date, runtime, subtitle status icon
  - [ ] 7.4 Subtitle status icons: `✅` found (green), `❌` not_found (red), `⏳` searching (amber), `—` not_searched (gray), hidden if no local file
  - [ ] 7.5 Loading skeleton while fetching episodes
  - [ ] 7.6 Error state with retry button (AC: #7)
  - [ ] 7.7 Write `EpisodeList.spec.tsx`

- [ ] **Task 8: Integrate into TV detail page** (AC: #1, #3)
  - [ ] 8.1 In `apps/web/src/routes/media/$type.$id.tsx`, add `<SeasonAccordion />` below overview/credits when `type === 'tv'`
  - [ ] 8.2 Pass `localData.seasons` (from `SeasonsJSON`) and `localData.tmdbId`
  - [ ] 8.3 Add `useSeasonEpisodes(seriesId, seasonNumber)` TanStack Query hook that calls `GET /api/v1/series/:id/seasons/:seasonNumber/episodes`
  - [ ] 8.4 `staleTime: 60 * 60 * 1000` (1h — episodes change less frequently)

- [ ] **Task 9: TypeScript types & service methods** (AC: #3, #4)
  - [ ] 9.1 Add `SeasonSummary`, `MergedEpisode` types to `types/library.ts`
  - [ ] 9.2 Add `getSeriesSeasons(seriesId)` and `getSeasonEpisodes(seriesId, seasonNumber)` to `libraryService.ts`
  - [ ] 9.3 `MergedEpisode` type: TMDB fields + `subtitleStatus?: string`, `filePath?: string`, `hasLocalFile: boolean`

## Dev Notes

### Architecture Compliance

- **Rule 4 (Layered Architecture):** `SeriesHandler.GetSeasonEpisodes` → `SeriesService.GetSeasonEpisodes` → `EpisodeRepository.FindBySeriesAndSeason` + `TMDbClient.GetSeasonDetails`
- **Rule 5 (TanStack Query):** Episode data fetched via `useQuery` with lazy loading (only on accordion expand)
- **Rule 6 (Naming):** Endpoint: `/api/v1/series/:id/seasons/:seasonNumber/episodes`; Go: `FindBySeriesAndSeason`; TS: `getSeasonEpisodes`
- **Rule 10 (API Versioning):** All new endpoints under `/api/v1/`
- **Rule 13 (Error Handling):** TMDB failures return partial data (season list from DB, episodes unavailable); never swallow errors
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` for DOM, `toBeAttached()` for accordion transitions
- **Rule 18 (Case Transform):** Auto via `fetchApi`

### Cross-Stack Split Check

Backend: 5 tasks, Frontend: 4 tasks → Both > 3.

⚠️ **This story has 5 backend tasks and 4 frontend tasks.** Per Agreement 5 (Epic 8 Retro), cross-stack stories with >3 tasks on each side should be split. However, the backend and frontend are tightly coupled (merged episode data format), and splitting would create awkward API-only vs UI-only stories with a hard dependency. **Recommendation: keep as single story** with advisory note that implementation may take longer.

### Project Structure Notes

**Files to CREATE:**
- `apps/api/internal/database/migrations/023_add_episode_subtitle_fields.go`
- `apps/web/src/components/media/SeasonAccordion.tsx`
- `apps/web/src/components/media/SeasonAccordion.spec.tsx`
- `apps/web/src/components/media/EpisodeList.tsx`
- `apps/web/src/components/media/EpisodeList.spec.tsx`

**Files to MODIFY:**
- `apps/api/internal/models/episode.go` — add subtitle fields
- `apps/api/internal/tmdb/tv.go` — add `GetSeasonDetails`
- `apps/api/internal/tmdb/types.go` — add `SeasonDetails`, `EpisodeInfo` types
- `apps/api/internal/repository/episode_repository.go` — `FindBySeriesAndSeason`, subtitle queries
- `apps/api/internal/services/series_service.go` — `GetSeasonEpisodes` merge logic
- `apps/api/internal/handlers/series_handler.go` — register new routes
- `apps/api/internal/database/migrations/registry.go` — register 023
- `apps/web/src/routes/media/$type.$id.tsx` — add `<SeasonAccordion />`
- `apps/web/src/services/libraryService.ts` — add `getSeriesSeasons`, `getSeasonEpisodes`
- `apps/web/src/types/library.ts` — add `SeasonSummary`, `MergedEpisode` types

### Critical Implementation Details

1. **Season data already cached on Series record.** The `series.seasons` JSON column stores `SeasonSummary[]` from TMDB (set during initial metadata fetch). Use this for the collapsed accordion — NO extra API call needed for the season list.

2. **Episode data merge pattern.** The key complexity is merging TMDB episode metadata (title, overview, air_date from `GetSeasonDetails`) with local `episodes` table data (file_path, subtitle_status). Use a map-based merge: index local episodes by `(season_number, episode_number)` → iterate TMDB episodes → attach local data if match exists.

3. **Episode table already exists** (migration 006). It has records for series that have been scanned. Not all series will have local episode records — only those whose files were found during library scanning. TMDB provides the canonical episode list; local records provide file/subtitle enrichment.

4. **Subtitle status at episode level is NEW.** Migration 023 adds `subtitle_status` to episodes. Existing episodes will default to `not_searched`. The subtitle engine (Epic 8) currently operates at series level — future work will need to extend it to per-episode searches, but this story only displays the status.

5. **TMDB Season Details API:** `GET /3/tv/{tv_id}/season/{season_number}?language=zh-TW`. Returns episodes array with: episode_number, name, overview, air_date, runtime, still_path, vote_average. Cache for 24h.

6. **Accordion lazy loading:** Only fetch episode data when a season is expanded. Use `enabled: isExpanded` in TanStack Query to prevent premature fetches.

7. **Performance:** Season list from DB (instant). Episode fetch per season (~200ms cached, ~1s uncached TMDB). Detail page load target < 1.5s is met because seasons display without episode data initially.

### Existing Code References

- Series model with SeasonsJSON: `apps/api/internal/models/series.go` — `GetSeasons()`, `SetSeasons()`
- SeasonSummary type: `apps/api/internal/models/series.go` — `SeasonSummary` struct
- Episode model: `apps/api/internal/models/episode.go` — `GetSeasonEpisodeCode()` returns "S01E05"
- Season model: `apps/api/internal/models/season.go`
- Episode repository: `apps/api/internal/repository/episode_repository.go`
- TMDB TV client: `apps/api/internal/tmdb/tv.go` — `GetTVShowDetails` exists, `GetSeasonDetails` does NOT
- TMDB Season type: `apps/api/internal/tmdb/types.go` — `Season` struct (summary only, no episodes)
- Detail page TV handling: `apps/web/src/routes/media/$type.$id.tsx`
- Frontend LibrarySeries type: `apps/web/src/types/library.ts`

### UX Design Note

No dedicated UX design screen for season/episode accordion yet (Epic 12 not in `ux-design.pen`). Follow these patterns:
- Accordion: similar to collapsible sections in Settings page
- Episode rows: table-like layout inspired by Plex/Jellyfin episode lists
- Subtitle status icons: reuse color tokens from design system (`--success`, `--error`, `--warning`, `--muted`)
- Poster thumbnails: use `getImageUrl(posterPath, 'w92')` from TMDB service

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-2]
- [Source: project-context.md — Rules 4, 5, 6, 10, 13, 16, 18]
- [Source: apps/api/internal/models/episode.go — Episode struct]
- [Source: apps/api/internal/models/season.go — Season struct]
- [Source: apps/api/internal/models/series.go — SeasonSummary, GetSeasons()]
- [Source: apps/api/internal/tmdb/types.go — Season struct (TMDB)]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
