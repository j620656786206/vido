# Bugfix 20-1: Season-summary endpoint reads a dead column (accordion empty)

Status: ready-for-dev

> Epic 20 — Test Quality Hardening · Story 1 of 3. Discovered 2026-06-14 during the
> Phase-2 v2 pilot's first real-data test (seeded local library → TV-detail season
> accordion was empty).

## Bug

`GET /api/v1/series/:id/seasons` always returns `[]` for every series, so the
season accordion (UX2-3 TV variant / Epic 12-2) never lists seasons.

**Root cause (verified in code):**
- `apps/api/internal/services/series_season.go:56` `GetSeasons` →
  `s.repo.FindByID(id).GetSeasons()`, and `models.Series.GetSeasons()`
  (`models/series.go:171`) parses `s.SeasonsJSON` — the `series.seasons` JSON
  column (migration 006).
- BUT `series_repository.go` `seriesSelectColumns` (`:589`) does **not** select
  `seasons`, and `scanSeries` (`:603`) never scans into `SeasonsJSON`. So
  `SeasonsJSON` is always empty → `GetSeasons()` → `[]`, regardless of column data.
- The `series.seasons` JSON column is **dead**: the series repo neither reads nor
  writes it. The canonical season data lives in the separate **`seasons` table**
  (migration 015, managed by `season_repository.go`), which `GetSeasonEpisodes`
  already uses for episodes.

## Acceptance Criteria

1. **Single source of truth = the `seasons` table.** `SeriesService.GetSeasons`
   reads from `season_repository` (e.g. `FindBySeriesID`), not the dead
   `series.seasons` JSON column. Returns `[]models.SeasonSummary` mapped from
   `[]models.Season` (id, season_number, name, overview, poster_path, air_date,
   episode_count). Ordered by `season_number`. (Rule 4 layering preserved.)
2. **`SeriesService` is wired with the seasons repo.** `repos.Seasons` injected
   (a setter alongside `SetEpisodeDeps`, wired in `cmd/api/main.go`). Rule 15
   self-verification: handler→service→repo path complete.
3. **Regression test at the real DB read path (the test that would have caught
   this).** A repository/integration test using a real sqlite DB seeds a series +
   N `seasons` rows and asserts `GetSeasons` (and `GET /series/:id/seasons`)
   returns them ordered. NOT a mock of the service (the existing handler/service
   tests mocked exactly the layer that hid the bug — Rule 16 specific assertions).
4. **Dead column flagged.** Add a `// DEPRECATED` note on `series.seasons` /
   `models.Series.SeasonsJSON` + `GetSeasons/SetSeasons`; the drop-column
   migration is a tracked follow-up (Rule 24 — backlog entry, not in this story).
5. **No FE change.** `GET /series/:id/seasons` contract (the `SeasonSummary[]`
   shape) is unchanged; the v2 + legacy accordion both light up once data flows.
6. **Verify on localhost** with seeded data (the 2026-06-14 seed already populated
   the `seasons` table? — NO, only the `series.seasons` column; this story's impl
   reads the table, so seeding must also write the `seasons` table, or rely on a
   real scan). Confirm 進擊的巨人 shows 4 seasons.

## Tasks / Subtasks (Amelia)

- [ ] **T1** Add `GetSeasons` read from `season_repository.FindBySeriesID` +
  map to `SeasonSummary`; remove the dead `FindByID().GetSeasons()` read.
- [ ] **T2** Wire `repos.Seasons` into `SeriesService` (setter + `main.go`).
- [ ] **T3** Integration test (real sqlite): seed series + seasons rows, assert
  `GetSeasons` returns them ordered; handler test hits the wired service.
- [ ] **T4** `// DEPRECATED` markers on the dead column/methods + a backlog entry
  for the drop-column migration (Rule 24).
- [ ] **T5** `pnpm lint:all` (go vet/staticcheck) + `go test ./...`; verify on
  localhost (seed the `seasons` table for the test series).

## Dev Notes
- [Source: apps/api/internal/services/series_season.go:56 GetSeasons]
- [Source: apps/api/internal/repository/series_repository.go:589 seriesSelectColumns / :603 scanSeries]
- [Source: apps/api/internal/repository/season_repository.go FindBySeriesID]
- [Source: apps/api/internal/models/series.go:171 GetSeasons (dead JSON path)]
- [Source: project-context.md — Rule 4 layering, Rule 15 DB sync, Rule 16 assertions, Rule 24 triage]

## Dev Agent Record
_(to be filled by dev-story)_
