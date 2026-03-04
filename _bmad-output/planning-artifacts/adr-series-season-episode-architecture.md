# ADR: Series/Season/Episode 3-Tier Architecture

**Status:** Approved
**Date:** 2026-03-04
**Author:** Winston (Architect) + Alexyu (Project Lead)
**Origin:** Epic 4 Retrospective Action Item #3 / TD-H2
**Impact:** Epic 5 (Media Library Management) prerequisite

---

## Context

Vido's parse pipeline (`ParseQueueService.ProcessNextJob()`) always creates a `Movie` record regardless of media type. When the filename parser correctly identifies a TV show (e.g., `S01E05` pattern), the metadata search routes to TMDb TV endpoints and returns TV show data — but the result is forced into a `Movie` struct, losing all series/season/episode structure.

The `Series` and `Episode` models and repositories already exist (created in migration 002 and 006), and the TMDb client already supports TV show search and details. However, the `seasons` table does not exist — season data is stored as a JSON blob (`SeasonsJSON` column) inside the `series` table.

This ADR defines the architecture changes required to support proper 3-tier media hierarchy: **Series → Season → Episode**.

---

## Decisions

### Decision 1: Independent `seasons` DB Table

**Choice:** Create a new `seasons` table as a first-class entity.

**Rejected:** Keeping seasons as JSON blob inside `series.seasons` column.

**Rationale:**
- Industry standard: Jellyfin, Plex, Emby all model Season as an independent entity
- Enables Season-level queries, sorting, pagination (needed for Epic 5 Media Library)
- Proper relational integrity with FK constraints and CASCADE deletes
- JSON blob cannot be indexed, JOINed, or filtered at DB level

### Decision 2: Lazy Season/Episode Creation

**Choice:** When processing a TV download, only create the Season and Episode records for the specific file being parsed.

**Rejected:** Eagerly fetching all seasons/episodes from TMDb when a Series is first created.

**Rationale:**
- Minimizes TMDb API calls (rate limit: 40 req/10s)
- Avoids creating hundreds of empty Episode records for long-running shows
- Seasons/episodes populate naturally as user downloads more content
- Series-level `SeasonsJSON` (TMDb summary) still provides overview data for UI display without needing all Season records

### Decision 3: Add `season_id` FK to Episodes

**Choice:** Add a nullable `season_id` column to the `episodes` table as a direct FK to `seasons`.

**Rejected:** Relying solely on `series_id + season_number` composite key for Season association.

**Rationale:**
- Single-column FK enables simpler JOINs: `JOIN seasons s ON e.season_id = s.id`
- DB-level referential integrity with CASCADE delete
- Better query performance (single index vs composite condition)
- Nullable so existing Episode records are unaffected (backward compatible)

---

## Architecture Design

### Database Schema

#### New Table: `seasons`

```sql
CREATE TABLE seasons (
    id TEXT PRIMARY KEY,
    series_id TEXT NOT NULL,
    tmdb_id INTEGER,
    season_number INTEGER NOT NULL,
    name TEXT,
    overview TEXT,
    poster_path TEXT,
    air_date TEXT,
    episode_count INTEGER,
    vote_average REAL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
    UNIQUE(series_id, season_number)
);

CREATE INDEX idx_seasons_series_id ON seasons(series_id);
CREATE INDEX idx_seasons_tmdb_id ON seasons(tmdb_id);
```

**Design Notes:**
- `UNIQUE(series_id, season_number)` — one Season record per series per season number
- `ON DELETE CASCADE` — deleting a Series removes all its Seasons
- `season_number = 0` represents TMDb's "Specials" convention
- Fields mirror `SeasonSummary` struct already defined in `models/series.go`

#### Migration: Add `season_id` to `episodes`

```sql
ALTER TABLE episodes ADD COLUMN season_id TEXT REFERENCES seasons(id) ON DELETE SET NULL;
CREATE INDEX idx_episodes_season_id ON episodes(season_id);
```

**Design Notes:**
- Nullable — existing episodes without a Season record remain valid
- `ON DELETE SET NULL` (not CASCADE) — if a Season is somehow deleted, Episodes survive but lose the link
- Does NOT change the existing `UNIQUE(series_id, season_number, episode_number)` constraint

#### Entity Relationship

```
series (existing)
  │
  ├── 1:N → seasons (NEW)
  │           │
  │           └── 1:N → episodes (existing, + season_id FK)
  │
  └── 1:N → episodes (existing series_id FK, kept for backward compat)
```

### Go Model: `Season`

**File:** `apps/api/internal/models/season.go`

```go
type Season struct {
    ID           string         `db:"id" json:"id"`
    SeriesID     string         `db:"series_id" json:"seriesId"`
    TMDbID       sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`
    SeasonNumber int            `db:"season_number" json:"seasonNumber"`
    Name         sql.NullString `db:"name" json:"name,omitempty"`
    Overview     sql.NullString `db:"overview" json:"overview,omitempty"`
    PosterPath   sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
    AirDate      sql.NullString `db:"air_date" json:"airDate,omitempty"`
    EpisodeCount sql.NullInt64  `db:"episode_count" json:"episodeCount,omitempty"`
    VoteAverage  sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`
    CreatedAt    time.Time      `db:"created_at" json:"createdAt"`
    UpdatedAt    time.Time      `db:"updated_at" json:"updatedAt"`
}
```

**Validation:**
- `ID` required
- `SeriesID` required
- `SeasonNumber >= 0` (0 = Specials)

### Go Model Update: `Episode`

**File:** `apps/api/internal/models/episode.go` — add one field:

```go
SeasonID sql.NullString `db:"season_id" json:"seasonId,omitempty"`
```

### Repository Interface: `SeasonRepository`

**File:** `apps/api/internal/repository/interfaces.go`

```go
type SeasonRepositoryInterface interface {
    Create(ctx context.Context, season *models.Season) error
    FindByID(ctx context.Context, id string) (*models.Season, error)
    FindBySeriesID(ctx context.Context, seriesID string) ([]models.Season, error)
    FindBySeriesAndNumber(ctx context.Context, seriesID string, seasonNumber int) (*models.Season, error)
    Update(ctx context.Context, season *models.Season) error
    Delete(ctx context.Context, id string) error
    Upsert(ctx context.Context, season *models.Season) error
}
```

**Key method:** `FindBySeriesAndNumber` — used by ParseQueueService to check if Season already exists before creating, and to retrieve `season.ID` for Episode creation.

**Key method:** `Upsert` — creates or updates based on `UNIQUE(series_id, season_number)`. This is the primary method used during parse pipeline to avoid duplicates when multiple episodes from the same season are downloaded.

### ParseQueueService Changes

**File:** `apps/api/internal/services/parse_queue_service.go`

#### New Dependencies

```go
type ParseQueueService struct {
    parseJobRepo    repository.ParseJobRepositoryInterface
    parserService   ParserServiceInterface
    metadataService MetadataServiceInterface
    movieRepo       repository.MovieRepositoryInterface
    seriesRepo      repository.SeriesRepositoryInterface   // NEW
    seasonRepo      repository.SeasonRepositoryInterface   // NEW
    episodeRepo     repository.EpisodeRepositoryInterface   // NEW
    logger          *slog.Logger
}
```

#### ProcessNextJob — MediaType Branching

Current Step 3 (`// Step 3: Create media entry from best match`) must be replaced with branching logic:

```
if mediaType == "movie":
    → createMovieFromMatch(ctx, bestMatch, searchResult, job)  // existing logic, extracted

if mediaType == "tv":
    → createTVEntryFromMatch(ctx, bestMatch, searchResult, job, parseResult)
```

#### createTVEntryFromMatch — New Method

```
func (s *ParseQueueService) createTVEntryFromMatch(
    ctx, bestMatch, searchResult, job, parseResult
) error:

    1. Upsert Series
       - Check if Series exists by TMDb ID: seriesRepo.FindByTMDbID()
       - If not found: create new Series from bestMatch metadata
       - If found: use existing Series (update metadata if needed)

    2. Upsert Season
       - seasonNumber = parseResult.Season (from filename parser)
       - Check: seasonRepo.FindBySeriesAndNumber(seriesID, seasonNumber)
       - If not found: create Season with data from Series.SeasonsJSON
         (TMDb TV details include season summaries)
       - If found: use existing Season

    3. Create Episode
       - episodeNumber = parseResult.Episode (from filename parser)
       - Map: title from bestMatch or filename, file_path from job
       - Set: series_id, season_id, season_number, episode_number
       - Use episodeRepo.Upsert() to handle re-downloads gracefully

    4. Return series.ID as the mediaID for the ParseJob
```

#### Season Data Source

When creating a Season record, the data comes from the **Series' TMDb details** which include a `seasons[]` array (already captured in `Series.SeasonsJSON`). The flow:

1. `metadataService.SearchMetadata(mediaType="tv")` returns TV show match
2. Series is created/found with TMDb ID
3. `Series.GetSeasons()` parses `SeasonsJSON` → `[]SeasonSummary`
4. Find the matching `SeasonSummary` by `season_number`
5. Map `SeasonSummary` fields to `Season` model

If no season summary is available (e.g., TMDb data incomplete), create a minimal Season with just `series_id` and `season_number`.

### Frontend TypeScript Type

**File:** `apps/web/src/types/tmdb.ts` — add:

```typescript
export interface SeasonDetail {
  id: string
  seriesId: string
  tmdbId?: number
  seasonNumber: number
  name?: string
  overview?: string
  posterPath?: string
  airDate?: string
  episodeCount?: number
  voteAverage?: number
  createdAt: string
  updatedAt: string
}
```

Update `Episode` type (if exists) to include optional `seasonId` field.

---

## Migration Strategy

### Migration 015: Create Seasons Table and Add Episode FK

**File:** `apps/api/internal/database/migrations/015_create_seasons_table.go`

This is a **single migration** that:

1. Creates the `seasons` table with indexes
2. Adds `season_id` column to `episodes` table
3. Creates index on `episodes.season_id`

**No data backfill required** — existing Episode records will have `season_id = NULL`, which is valid. Future parse jobs will populate `season_id` correctly.

### Rollback

```sql
DROP INDEX IF EXISTS idx_episodes_season_id;
ALTER TABLE episodes DROP COLUMN season_id;  -- SQLite: requires table rebuild
DROP TABLE IF EXISTS seasons;
```

Note: SQLite doesn't support `DROP COLUMN` natively before 3.35.0. The migration system should use table rebuild pattern if needed.

---

## Impact Analysis

### Files to Create

| File | Purpose |
|------|---------|
| `models/season.go` | Season domain model + validation |
| `repository/season_repository.go` | Season CRUD + Upsert |
| `database/migrations/015_create_seasons_table.go` | DB schema changes |

### Files to Modify

| File | Change |
|------|--------|
| `models/episode.go` | Add `SeasonID` field |
| `repository/interfaces.go` | Add `SeasonRepositoryInterface` |
| `repository/episode_repository.go` | Include `season_id` in INSERT/UPDATE/scan |
| `services/parse_queue_service.go` | Add TV branching, inject new repos |
| `cmd/api/main.go` | Wire SeasonRepository, update ParseQueueService constructor |
| `apps/web/src/types/tmdb.ts` | Add `SeasonDetail` type, update Episode |

### Files NOT Changed

- `models/series.go` — `SeasonsJSON` kept as-is (useful for TMDb summary display)
- `repository/series_repository.go` — no schema change
- `tmdb/` — already supports TV endpoints
- `metadata/` — already routes TV correctly

### Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Existing Episode data has no `season_id` | Column is nullable, no backfill needed |
| Multiple episodes from same season downloaded concurrently | `Upsert` with `UNIQUE(series_id, season_number)` prevents duplicates |
| TMDb rate limit during Series detail fetch | Already rate-limited at 40 req/10s in TMDb client |
| ParseQueueService constructor signature change | Update all callers (main.go, tests) |

---

## Implementation Order

Recommended sequence for the dev story:

1. **Migration** — Create `seasons` table + add `season_id` to episodes
2. **Season Model** — `models/season.go` with validation
3. **Season Repository** — `repository/season_repository.go` + interface
4. **Episode Model Update** — Add `SeasonID` field + update repository SQL
5. **ParseQueueService Refactor** — Extract `createMovieFromMatch`, add `createTVEntryFromMatch`
6. **Wiring** — Update `main.go` constructor
7. **Tests** — Unit tests for Season repo, ParseQueueService TV branch
8. **Frontend Type** — Add `SeasonDetail` TypeScript type

---

## Appendix: TMDb Data Mapping

### Series ← TMDb TVShowDetails

| Series Field | TMDb Source | Notes |
|-------------|-------------|-------|
| `title` | `Name` | zh-TW name |
| `original_title` | `OriginalName` | |
| `first_air_date` | `FirstAirDate` | |
| `last_air_date` | `LastAirDate` | |
| `overview` | `Overview` | |
| `poster_path` | `PosterPath` | |
| `backdrop_path` | `BackdropPath` | |
| `number_of_seasons` | `NumberOfSeasons` | |
| `number_of_episodes` | `NumberOfEpisodes` | |
| `status` | `Status` | "Returning Series", "Ended", etc. |
| `in_production` | `InProduction` | |
| `tmdb_id` | `ID` | |
| `vote_average` | `VoteAverage` | |
| `seasons` (JSON) | `Seasons[]` | Array of season summaries |
| `networks` (JSON) | `Networks[]` | |

### Season ← TMDb Season (from TVShowDetails.Seasons[])

| Season Field | TMDb Source | Notes |
|-------------|-------------|-------|
| `tmdb_id` | `Season.ID` | |
| `season_number` | `Season.SeasonNumber` | 0 = Specials |
| `name` | `Season.Name` | e.g., "Season 1", "Specials" |
| `overview` | `Season.Overview` | |
| `poster_path` | `Season.PosterPath` | |
| `air_date` | `Season.AirDate` | |
| `episode_count` | `Season.EpisodeCount` | |

### Episode ← Filename Parser + TMDb

| Episode Field | Source | Notes |
|--------------|--------|-------|
| `series_id` | Upserted Series ID | |
| `season_id` | Upserted Season ID | |
| `season_number` | `parseResult.Season` | From filename parser |
| `episode_number` | `parseResult.Episode` | From filename parser |
| `title` | Filename or TMDb | Filename as fallback |
| `file_path` | `job.FilePath` | Download path |

---

**Document Generated:** 2026-03-04 by Winston (Architect Agent)
**Approved by:** Alexyu (Project Lead)
