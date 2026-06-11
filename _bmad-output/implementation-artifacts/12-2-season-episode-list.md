# Story 12.2: Season/Episode List — Expandable Accordion with Subtitle Status

Status: done

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

- [x] **Task 1: DB Migration 025 — Add episode-level subtitle tracking** (AC: #5)
  - [x] 1.1 Create `025_add_episode_subtitle_fields.go` in `apps/api/internal/database/migrations/` (⚠️ migration 023/024 already taken — bumped to 025)
  - [x] 1.2 Add to `episodes` table: `subtitle_status TEXT DEFAULT 'not_searched'`, `subtitle_path TEXT`, `subtitle_language TEXT`
  - [x] 1.3 Create index: `idx_episodes_subtitle_status`
  - [x] 1.4 Register migration via `init()` self-registration (project pattern — `registry.go` is not hand-edited)

- [x] **Task 2: Extend Episode model with subtitle fields** (AC: #5)
  - [x] 2.1 Add to `Episode` struct in `models/episode.go`: `SubtitleStatus SubtitleStatus`, `SubtitlePath NullString`, `SubtitleLanguage NullString`
  - [x] 2.2 Add db/json tags matching convention

- [x] **Task 3: TMDB GetSeasonDetails endpoint** (AC: #3)
  - [x] 3.1 Add `GetSeasonDetails(ctx, tvID int, seasonNumber int) (*SeasonDetails, error)` to `tmdb/tv.go` (+ `GetSeasonDetailsWithLanguage`, threaded through Client→fallback→cache→TMDbService chain)
  - [x] 3.2 Add `SeasonDetails` type to `tmdb/types.go` with `Episodes []EpisodeInfo`
  - [x] 3.3 Add `EpisodeInfo` type to `tmdb/types.go`
  - [x] 3.4 Use `zh-TW` language parameter (via fallback chain), cache response (24h `DefaultCacheTTL`, key `tmdb:tv/{id}/season/{n}`)

- [x] **Task 4: Season/Episode API endpoints** (AC: #1, #3, #4, #5)
  - [x] 4.1 Add `GET /api/v1/series/:id/seasons` — returns season list from `SeasonsJSON` (no TMDB call)
  - [x] 4.2 Add `GET /api/v1/series/:id/seasons/:seasonNumber/episodes` — merges TMDB + local subtitle/file status
  - [x] 4.3 Merge logic: index local episodes by episode_number; attach subtitle_status/file_path when a local file exists
  - [x] 4.4 Response format: `{ "success": true, "data": { "season": {...}, "episodes": [{...}] } }`
  - [x] 4.5 Add to series handler, register routes in `RegisterRoutes`
  - [x] 4.6 Write handler + service tests

- [x] **Task 5: Episode repository — subtitle queries** (AC: #5)
  - [x] 5.1 ~~Add `FindBySeriesAndSeason`~~ → **reused existing `FindBySeasonNumber(ctx, seriesID, seasonNumber)`** (identical semantics; reuse-over-reinvent, retro-9c)
  - [x] 5.2 Update existing episode SELECT queries to include new subtitle columns (shared `episodeSelectColumns` + `scanEpisode` helper)
  - [x] 5.3 Add `UpdateEpisodeSubtitleStatus(ctx, episodeID string, status SubtitleStatus, path, language string) error`

### Frontend

- [x] **Task 6: Season accordion component** (AC: #1, #2, #8)
  - [x] 6.1 Create `apps/web/src/components/media/SeasonAccordion.tsx`
  - [x] 6.2 Props: `seasons: SeasonSummary[]`, `seriesId: string`, `tmdbId: number` (tmdbId gates rendering per AC #1)
  - [x] 6.3 Collapsed state: poster thumbnail (56×84), season name, episode count, air date
  - [x] 6.4 Click toggles expansion — default multi-open (each season owns its own expand state)
  - [x] 6.5 Expansion triggers episode list fetch (lazy load via `enabled: isExpanded`)
  - [x] 6.6 Tailwind responsive: header row + stacked episode rows on mobile
  - [x] 6.7 Write `SeasonAccordion.spec.tsx`

- [x] **Task 7: Episode list component** (AC: #4, #5, #6, #8)
  - [x] 7.1 Create `apps/web/src/components/media/EpisodeList.tsx`
  - [x] 7.2 Props: `episodes: MergedEpisode[]` (+ `seasonNumber` for SxxExx code; loading/error/retry)
  - [x] 7.3 Each row: episode code (S01E05), title, air date, runtime, subtitle status icon
  - [x] 7.4 Subtitle status icons (lucide + color tokens + aria-label): found (green), not_found (red), searching (amber spinner), not_searched (gray), hidden if no local file
  - [x] 7.5 Loading skeleton while fetching episodes
  - [x] 7.6 Error state with retry button (AC: #7)
  - [x] 7.7 Write `EpisodeList.spec.tsx`

- [x] **Task 8: Integrate into TV detail page** (AC: #1, #3)
  - [x] 8.1 In `apps/web/src/routes/media/$type.$id.tsx`, add `<SeasonAccordion />` below credits when `type === 'tv'`
  - [x] 8.2 ~~Pass `localData.seasons`~~ → `SeasonsJSON` is `json:"-"` (not in detail response); added `useSeriesSeasons(id)` hook calling `GET /series/:id/seasons` + pass `tmdbId`
  - [x] 8.3 Add `useSeasonEpisodes(seriesId, seasonNumber, enabled)` TanStack Query hook calling `GET /api/v1/series/:id/seasons/:seasonNumber/episodes`
  - [x] 8.4 `staleTime: 60 * 60 * 1000` (1h — episodes change less frequently)

- [x] **Task 9: TypeScript types & service methods** (AC: #3, #4)
  - [x] 9.1 Add `SeasonSummary`, `MergedEpisode`, `SeasonEpisodesResponse` types to `types/library.ts`
  - [x] 9.2 Add `getSeriesSeasons(seriesId)` and `getSeasonEpisodes(seriesId, seasonNumber)` to `libraryService.ts`
  - [x] 9.3 `MergedEpisode` type: TMDB fields + `subtitleStatus?: string`, `filePath?: string`, `hasLocalFile: boolean`

## Dev Notes

### Architecture Compliance

- **Rule 4 (Layered Architecture):** `SeriesHandler.GetSeasonEpisodes` → `SeriesService.GetSeasonEpisodes` → `EpisodeRepository.FindBySeasonNumber` (reused per 5.1) + `TMDbClient.GetSeasonDetails`
- **Rule 5 (TanStack Query):** Episode data fetched via `useQuery` with lazy loading (only on accordion expand)
- **Rule 6 (Naming):** Endpoint: `/api/v1/series/:id/seasons/:seasonNumber/episodes`; Go: `FindBySeasonNumber` (reused); TS: `getSeasonEpisodes`
- **Rule 10 (API Versioning):** All new endpoints under `/api/v1/`
- **Rule 13 (Error Handling):** TMDB failures return partial data (season list from DB, episodes unavailable); never swallow errors
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` for DOM, `toBeAttached()` for accordion transitions
- **Rule 18 (Case Transform):** Auto via `fetchApi`

### Cross-Stack Split Check

Backend: 5 tasks, Frontend: 4 tasks → Both > 3.

⚠️ **This story has 5 backend tasks and 4 frontend tasks.** Per Agreement 5 (Epic 8 Retro), cross-stack stories with >3 tasks on each side should be split. However, the backend and frontend are tightly coupled (merged episode data format), and splitting would create awkward API-only vs UI-only stories with a hard dependency. **Recommendation: keep as single story** with advisory note that implementation may take longer.

### Project Structure Notes

**Files to CREATE:** _(corrected during impl: migration is 025 not 023; self-registers via `init()`, no registry.go edit)_
- `apps/api/internal/database/migrations/025_add_episode_subtitle_fields.go`
- `apps/web/src/components/media/SeasonAccordion.tsx`
- `apps/web/src/components/media/SeasonAccordion.spec.tsx`
- `apps/web/src/components/media/EpisodeList.tsx`
- `apps/web/src/components/media/EpisodeList.spec.tsx`

**Files to MODIFY:**
- `apps/api/internal/models/episode.go` — add subtitle fields
- `apps/api/internal/tmdb/tv.go` — add `GetSeasonDetails`
- `apps/api/internal/tmdb/types.go` — add `SeasonDetails`, `EpisodeInfo` types
- `apps/api/internal/repository/episode_repository.go` — reused `FindBySeasonNumber`, subtitle queries
- `apps/api/internal/services/series_service.go` — `GetSeasonEpisodes` merge logic
- `apps/api/internal/handlers/series_handler.go` — register new routes
- ~~`apps/api/internal/database/migrations/registry.go` — register 023~~ (N/A — migrations self-register via `init()`)
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

claude-opus-4-8[1m] (Amelia / dev-story workflow)

### Debug Log References

🔗 AC Drift: NONE (checked: `subtitle_status`+episode across _bmad-output/implementation-artifacts/*.md — hits in retro-8-TD4 + 8-9, all REUSE not DRIFT: existing season-batch `subtitle/batch.go:489` filters episodes by FilePath only; `episodes.subtitle_status` did not exist before this story).
📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story or epic-12; defines new wire contracts but no prior story stamps them — normal for a new-subsystem story).

**Story-staleness corrections (verified from code):**
- Task 1 migration number `023` → `025` (`023_create_filter_presets.go` and `024_add_douban_rating_fields.go` already exist).
- Task 1.4 "register in registry.go" → migrations self-register via `init()` + `Register(&m{migrationBase: NewMigrationBase(N, ...)})`; `registry.go` is the registry engine, not hand-edited.

### Completion Notes List

**Story 12-2 implemented across 9 tasks (5 backend + 4 frontend).** Season accordion on the TV detail page: collapsed season summaries (poster/name/count/air-date) read from the series' cached `SeasonsJSON`; on expand, each season lazily fetches its episode list (TMDB metadata merged with local file/subtitle status) and renders SxxExx code, title, air date, runtime, and a per-episode subtitle status indicator (shown only when a local file exists).

🔗 AC Drift: NONE (see Debug Log).
📎 Contract Stamps: NONE (see Debug Log).
🎭 A11y Pre-Flight: PASS (2 components checked — SeasonAccordion, EpisodeList; `eslint` jsx-a11y + Rule-21 clean, 0 warnings introduced). Manual 4-class check: ① responsive image — season poster is a fixed 56×84 thumbnail at `w92` (sub-original, `loading="lazy"`, `alt`); a fixed-size thumbnail does not need srcSet (Story 10-2 H1 applied to large hero imagery). ② modal focus — N/A (native `<button>` accordion, no modal/dialog). ③ aria-live async content — subtitle status indicators carry `role="status"` (implicit `aria-live=polite`) + `aria-label`; error state uses `role="alert"`. ④ custom-widget keyboard/ARIA — accordion headers are native `<button>` with `aria-expanded` + `aria-controls` (keyboard-accessible by default). Lazy-load contract: episode fetch gated by `enabled: isExpanded`, described in AC #3 and the hook comment.

🎨 UX Verification: N/A — Epic 12 has no `.pen` design screen for the season/episode accordion (confirmed in Dev Notes → UX Design Note). Followed the specified pattern guidance: accordion modeled on `CollapsibleSection`, Plex/Jellyfin-style episode rows, subtitle-status colors from design tokens (`--success`/`--error`/`--warning`/`--text-muted`), poster via `getImageUrl(posterPath, 'w92')`. Components carry the Rule-21 `// Implements: <screen-section — pending epic-19-8 mapping>` placeholder for future `.pen` mapping.

🛠️ Pre-existing fix: `gofmt -w internal/models/episode.go` (my new `NullString` subtitle fields widened the struct column, triggering a gofmt realignment of adjacent fields). PRE-EXISTING (not fixed, noted): `internal/tmdb/client.go` (ClientConfig) and `cmd/api/main.go` (line ~545 RegisterRoutes comment block) were already gofmt-unformatted on `HEAD` in regions this story did not touch; left as-is to avoid unrelated diff noise (CI does not gate on gofmt — `main` is green while unformatted).

✅ Full regression gate: `nx test api` PASS (all Go packages); `nx test web` 2055/2055 PASS (171 files — note: the `preexisting-fail-eslint-config-wiring-vitest` batch did NOT reproduce here, all eslint-rule specs passed); `go build ./...` clean; web `tsc --noEmit` 0 errors; eslint touched files 0 errors (1 pre-existing `react-hooks/exhaustive-deps` warning in `$type.$id.tsx` on an untouched `useCallback`); prettier + gofmt clean on touched files; test cleanup verified (no orphaned workers).

📐 Story-staleness corrections (verified from code, applied): migration `023`→`025`; `init()` self-registration not registry.go edit; `FindBySeriesAndSeason` (5.1) reused existing `FindBySeasonNumber`; `localData.seasons` (8.2) unavailable (`SeasonsJSON` is `json:"-"`) → added `useSeriesSeasons` hook over the `GET /series/:id/seasons` endpoint.

### Review Follow-ups (AI) — applied 2026-06-11

Adversarial code review (CR workflow) found 0 High, 2 Medium, 3 Low. Fixed [1] automatically:

- **[M1][Wire-format]** `series_handler.go:385` — renamed error code `SERIES_NOT_LINKED_TMDB` → `TMDB_SERIES_NOT_LINKED` (`SERIES_` was not a registered Rule 7 source prefix; the natural source is the TMDb subsystem, matching the sibling `TMDB_SEASON_UNAVAILABLE`). Single call site, no test/Swagger references.
- **[M2][UX/AC #7]** Season-list fetch (`useSeriesSeasons`) loading/error were unhandled — on failure the accordion silently rendered nothing. `SeasonAccordion` now takes `isLoading`/`isError`/`onRetry`; shows a skeleton while loading and a retry-able `role="alert"` on error (mirrors AC #7's per-season contract). Wired from `$type.$id.tsx` via `seasonsQuery.{isLoading,isError,refetch}`. +3 specs (SeasonAccordion 6→9).
- **[L1][Docs]** Synced story-internal drift: Architecture Compliance (Rule 4/6) and Project Structure Notes now reflect reused `FindBySeasonNumber` (not `FindBySeriesAndSeason`), migration `025` (not `023`), and `init()` self-registration (no `registry.go` edit).
- **[L2][Scope]** _Noted, not changed:_ accordion renders only in the local library detail view (`MediaDetailView`), not the `TMDbDetailView` (tmdb-numeric route) — by design, since `/series/:id/seasons` requires a local series UUID. AC #1's "tmdb_id > 0" is satisfied within the library-detail scope.
- **[L3][Hygiene]** _Noted, not changed:_ `GetSeasons` handler maps all errors (incl. SeasonsJSON parse) to 404; acceptable since GetSeasons returns `[]` (not error) for the no-seasons case. Incidental `.claude/github-star-reminder.txt` working-tree change is a hook artifact, not part of this story.

Post-fix gate: `go build ./...` clean; `go test ./internal/handlers ./internal/services` PASS; SeasonAccordion + EpisodeList specs 14/14 PASS; prettier + gofmt clean on touched files. (Pre-existing, unrelated `tsc` errors remain in dashboard/homepage/library/scanner files + stale `routeTree.gen.ts` — not touched by this story.)

### File List

- `apps/api/internal/database/migrations/025_add_episode_subtitle_fields.go` (NEW)
- `apps/api/internal/database/migrations/025_add_episode_subtitle_fields_test.go` (NEW)
- `apps/api/internal/models/episode.go` (MODIFIED — subtitle fields)
- `apps/api/internal/models/episode_test.go` (MODIFIED — subtitle field test)
- `apps/api/internal/tmdb/types.go` (MODIFIED — `SeasonDetails`, `EpisodeInfo`)
- `apps/api/internal/tmdb/tv.go` (MODIFIED — `GetSeasonDetails`[`WithLanguage`])
- `apps/api/internal/tmdb/client.go` (MODIFIED — ClientInterface season methods)
- `apps/api/internal/tmdb/fallback.go` (MODIFIED — `GetSeasonDetailsWithFallback`)
- `apps/api/internal/tmdb/cache.go` (MODIFIED — cached `GetSeasonDetails`, 24h)
- `apps/api/internal/tmdb/fallback_test.go`, `cache_test.go` (MODIFIED — mocks + cache test)
- `apps/api/internal/services/tmdb_service.go` (MODIFIED — `GetSeasonDetails`)
- `apps/api/internal/services/series_service.go` (MODIFIED — `SetEpisodeDeps` + deps)
- `apps/api/internal/services/series_season.go` (NEW — merge logic, DTOs, `SeasonDetailsProvider`)
- `apps/api/internal/services/series_season_test.go` (NEW — merge tests)
- `apps/api/internal/services/{tmdb_service,enrichment_nfo,explore_block_service}_test.go` (MODIFIED — mocks)
- `apps/api/internal/services/parse_queue_service_test.go` (MODIFIED — episode mock)
- `apps/api/internal/repository/episode_repository.go` (MODIFIED — subtitle SELECTs + `UpdateEpisodeSubtitleStatus`)
- `apps/api/internal/repository/episode_repository_test.go` (MODIFIED — subtitle column + test)
- `apps/api/internal/repository/interfaces.go` (MODIFIED — `UpdateEpisodeSubtitleStatus`)
- `apps/api/internal/handlers/series_handler.go` (MODIFIED — `GetSeasons`, `GetSeasonEpisodes` + routes)
- `apps/api/internal/handlers/series_handler_test.go` (MODIFIED — handler tests)
- `apps/api/internal/handlers/tmdb_handler_test.go` (MODIFIED — mock)
- `apps/api/cmd/api/main.go` (MODIFIED — wire `SetEpisodeDeps`)
- `apps/web/src/types/library.ts` (MODIFIED — `SeasonSummary`, `MergedEpisode`, `SeasonEpisodesResponse`)
- `apps/web/src/services/libraryService.ts` (MODIFIED — `getSeriesSeasons`, `getSeasonEpisodes`)
- `apps/web/src/hooks/useMediaDetails.ts` (MODIFIED — `useSeriesSeasons`, `useSeasonEpisodes` + query keys)
- `apps/web/src/components/media/SeasonAccordion.tsx` (NEW)
- `apps/web/src/components/media/SeasonAccordion.spec.tsx` (NEW)
- `apps/web/src/components/media/EpisodeList.tsx` (NEW)
- `apps/web/src/components/media/EpisodeList.spec.tsx` (NEW)
- `apps/web/src/routes/media/$type.$id.tsx` (MODIFIED — render `<SeasonAccordion />` for TV)

## Change Log

| Date | Change |
|------|--------|
| 2026-06-11 | Task 1: migration 025 adds `subtitle_status`/`subtitle_path`/`subtitle_language` + `idx_episodes_subtitle_status` to episodes table (idempotent, tested). |
| 2026-06-11 | Task 2: extend Episode model with subtitle fields + db/json tags; JSON marshal test. |
| 2026-06-11 | Task 3: `GetSeasonDetails` threaded through TMDB Client→fallback→cache (24h)→TMDbService chain; `SeasonDetails`/`EpisodeInfo` types; cache test. |
| 2026-06-11 | Task 4: `GET /series/:id/seasons` + `GET /series/:id/seasons/:n/episodes` (TMDB+local merge); `SeasonEpisodesResponse`/`MergedEpisode` DTOs; retry-able upstream error mapping (AC #7); handler + service tests. |
| 2026-06-11 | Task 5: episode SELECTs include subtitle columns (shared `episodeSelectColumns`/`scanEpisode`); `UpdateEpisodeSubtitleStatus`; reused `FindBySeasonNumber` for 5.1. |
| 2026-06-11 | Task 9: `SeasonSummary`/`MergedEpisode`/`SeasonEpisodesResponse` types + `getSeriesSeasons`/`getSeasonEpisodes` service methods. |
| 2026-06-11 | Task 7: `EpisodeList` presentational component (SxxExx rows, subtitle status icons, skeleton, retry error state) + spec. |
| 2026-06-11 | Task 6: `SeasonAccordion` component (collapsible per-season, lazy episode fetch on expand, multi-open, tmdbId-gated) + spec. |
| 2026-06-11 | Task 8: integrate `<SeasonAccordion />` into TV detail page; `useSeriesSeasons` + `useSeasonEpisodes` hooks (1h staleTime, lazy enable). |
| 2026-06-11 | Status → review: all 9 tasks complete; full regression gate green (api + web 2055/2055). |
| 2026-06-11 | CR fixes: [M1] error code `SERIES_NOT_LINKED_TMDB`→`TMDB_SERIES_NOT_LINKED` (Rule 7 prefix); [M2] season-list loading/error/retry surfaced in `SeasonAccordion` (+3 specs); [L1] story-internal doc drift synced. Status → done. |
