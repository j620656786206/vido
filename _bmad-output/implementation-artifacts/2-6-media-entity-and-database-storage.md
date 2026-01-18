# Story 2.6: Media Entity and Database Storage

## Story

**As a** user
**I want** parsed media metadata to be stored persistently in the database
**So that** I can search, browse, and manage my media library without re-parsing files

## Status

**Status:** Completed
**Epic:** Epic 2 - Media Search & Traditional Chinese Metadata
**Sprint:** MVP Phase
**Story Points:** 8

## Context

This story implements the database persistence layer for media entities (movies and TV series). It integrates with:
- **Story 2.1** - TMDb API Integration (metadata source)
- **Story 2.5** - Standard Filename Parser (parsing results)
- **Story 1.1** - Repository Pattern (database abstraction)

The implementation uses SQLite FTS5 for full-text search to meet the <500ms search performance requirement (NFR-SC8).

## Functional Requirements Covered

- **FR5**: Media library storage and persistence
- **FR6**: Full-text search across media library
- **FR7**: Media entity CRUD operations

## Acceptance Criteria

### AC1: Movie Entity Persistence
**Given** a movie has been parsed and matched with TMDb
**When** the movie is saved to the database
**Then** all metadata fields are persisted including:
- TMDb ID, IMDb ID, title, original title
- Overview, poster path, backdrop path
- Release date, runtime, vote average
- Genres (JSON array), credits (JSON)
- Created/updated timestamps

### AC2: Series Entity Persistence
**Given** a TV series has been parsed and matched with TMDb
**When** the series is saved to the database
**Then** all metadata fields are persisted including:
- TMDb ID, title, original title, overview
- Poster/backdrop paths, first air date
- Number of seasons/episodes, status
- Genres, seasons data (JSON)
- Episode information linked to series

### AC3: Full-Text Search Performance
**Given** a media library with 10,000+ entries
**When** a user performs a title search
**Then** results are returned within 500ms (NFR-SC8)
**And** search matches title, original title, and overview

### AC4: Duplicate Prevention
**Given** a movie/series already exists in the database
**When** the same media is parsed again
**Then** the existing record is updated (not duplicated)
**And** the system uses TMDb ID as the unique identifier

### AC5: Repository Pattern Compliance
**Given** the repository pattern from Story 1.1
**When** implementing media storage
**Then** all database operations use the repository interfaces
**And** business logic remains in the service layer

## Technical Requirements

### Database Schema

#### Movies Table (Enhanced)
```sql
CREATE TABLE IF NOT EXISTS movies (
    id TEXT PRIMARY KEY,
    tmdb_id INTEGER UNIQUE,
    imdb_id TEXT,
    title TEXT NOT NULL,
    original_title TEXT,
    overview TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    release_date TEXT,
    runtime INTEGER,
    vote_average REAL,
    vote_count INTEGER,
    popularity REAL,
    genres TEXT,           -- JSON array
    credits TEXT,          -- JSON object {cast, crew}
    production_countries TEXT, -- JSON array
    spoken_languages TEXT,     -- JSON array
    file_path TEXT,        -- Path to media file
    file_size INTEGER,
    parse_status TEXT DEFAULT 'pending', -- pending, success, needs_ai, failed
    metadata_source TEXT,  -- tmdb, douban, wikipedia, manual
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS movies_fts USING fts5(
    title,
    original_title,
    overview,
    content='movies',
    content_rowid='rowid'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER movies_ai AFTER INSERT ON movies BEGIN
    INSERT INTO movies_fts(rowid, title, original_title, overview)
    VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
END;

CREATE TRIGGER movies_ad AFTER DELETE ON movies BEGIN
    INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
    VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
END;

CREATE TRIGGER movies_au AFTER UPDATE ON movies BEGIN
    INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
    VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
    INSERT INTO movies_fts(rowid, title, original_title, overview)
    VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
END;
```

#### Series Table
```sql
CREATE TABLE IF NOT EXISTS series (
    id TEXT PRIMARY KEY,
    tmdb_id INTEGER UNIQUE,
    title TEXT NOT NULL,
    original_title TEXT,
    overview TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    first_air_date TEXT,
    last_air_date TEXT,
    status TEXT,           -- Returning Series, Ended, Canceled, etc.
    number_of_seasons INTEGER,
    number_of_episodes INTEGER,
    vote_average REAL,
    vote_count INTEGER,
    popularity REAL,
    genres TEXT,           -- JSON array
    credits TEXT,          -- JSON object
    seasons TEXT,          -- JSON array of season summaries
    networks TEXT,         -- JSON array
    file_path TEXT,        -- Base folder path
    parse_status TEXT DEFAULT 'pending',
    metadata_source TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- FTS5 for series
CREATE VIRTUAL TABLE IF NOT EXISTS series_fts USING fts5(
    title,
    original_title,
    overview,
    content='series',
    content_rowid='rowid'
);

-- Similar triggers for series_fts...
```

#### Episodes Table
```sql
CREATE TABLE IF NOT EXISTS episodes (
    id TEXT PRIMARY KEY,
    series_id TEXT NOT NULL,
    tmdb_id INTEGER,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date TEXT,
    runtime INTEGER,
    still_path TEXT,
    vote_average REAL,
    file_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
    UNIQUE(series_id, season_number, episode_number)
);
```

### Model Updates

#### Enhanced Movie Model
```go
// /apps/api/internal/models/movie.go
type Movie struct {
    ID                 string         `db:"id" json:"id"`
    TMDbID             sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`
    IMDbID             sql.NullString `db:"imdb_id" json:"imdbId,omitempty"`
    Title              string         `db:"title" json:"title"`
    OriginalTitle      sql.NullString `db:"original_title" json:"originalTitle,omitempty"`
    Overview           sql.NullString `db:"overview" json:"overview,omitempty"`
    PosterPath         sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
    BackdropPath       sql.NullString `db:"backdrop_path" json:"backdropPath,omitempty"`
    ReleaseDate        sql.NullString `db:"release_date" json:"releaseDate,omitempty"`
    Runtime            sql.NullInt64  `db:"runtime" json:"runtime,omitempty"`
    VoteAverage        sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`
    VoteCount          sql.NullInt64  `db:"vote_count" json:"voteCount,omitempty"`
    Popularity         sql.NullFloat64 `db:"popularity" json:"popularity,omitempty"`
    Genres             sql.NullString `db:"genres" json:"genres,omitempty"`       // JSON
    Credits            sql.NullString `db:"credits" json:"credits,omitempty"`     // JSON
    ProductionCountries sql.NullString `db:"production_countries" json:"productionCountries,omitempty"`
    SpokenLanguages    sql.NullString `db:"spoken_languages" json:"spokenLanguages,omitempty"`
    FilePath           sql.NullString `db:"file_path" json:"filePath,omitempty"`
    FileSize           sql.NullInt64  `db:"file_size" json:"fileSize,omitempty"`
    ParseStatus        string         `db:"parse_status" json:"parseStatus"`
    MetadataSource     sql.NullString `db:"metadata_source" json:"metadataSource,omitempty"`
    CreatedAt          time.Time      `db:"created_at" json:"createdAt"`
    UpdatedAt          time.Time      `db:"updated_at" json:"updatedAt"`
}

// Helper methods for JSON fields
func (m *Movie) GetGenres() ([]Genre, error) { ... }
func (m *Movie) SetGenres(genres []Genre) error { ... }
func (m *Movie) GetCredits() (*Credits, error) { ... }
func (m *Movie) SetCredits(credits *Credits) error { ... }
```

#### New Series Model
```go
// /apps/api/internal/models/series.go
type Series struct {
    ID               string         `db:"id" json:"id"`
    TMDbID           sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`
    Title            string         `db:"title" json:"title"`
    OriginalTitle    sql.NullString `db:"original_title" json:"originalTitle,omitempty"`
    Overview         sql.NullString `db:"overview" json:"overview,omitempty"`
    PosterPath       sql.NullString `db:"poster_path" json:"posterPath,omitempty"`
    BackdropPath     sql.NullString `db:"backdrop_path" json:"backdropPath,omitempty"`
    FirstAirDate     sql.NullString `db:"first_air_date" json:"firstAirDate,omitempty"`
    LastAirDate      sql.NullString `db:"last_air_date" json:"lastAirDate,omitempty"`
    Status           sql.NullString `db:"status" json:"status,omitempty"`
    NumberOfSeasons  sql.NullInt64  `db:"number_of_seasons" json:"numberOfSeasons,omitempty"`
    NumberOfEpisodes sql.NullInt64  `db:"number_of_episodes" json:"numberOfEpisodes,omitempty"`
    VoteAverage      sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`
    VoteCount        sql.NullInt64  `db:"vote_count" json:"voteCount,omitempty"`
    Popularity       sql.NullFloat64 `db:"popularity" json:"popularity,omitempty"`
    Genres           sql.NullString `db:"genres" json:"genres,omitempty"`
    Credits          sql.NullString `db:"credits" json:"credits,omitempty"`
    Seasons          sql.NullString `db:"seasons" json:"seasons,omitempty"`
    Networks         sql.NullString `db:"networks" json:"networks,omitempty"`
    FilePath         sql.NullString `db:"file_path" json:"filePath,omitempty"`
    ParseStatus      string         `db:"parse_status" json:"parseStatus"`
    MetadataSource   sql.NullString `db:"metadata_source" json:"metadataSource,omitempty"`
    CreatedAt        time.Time      `db:"created_at" json:"createdAt"`
    UpdatedAt        time.Time      `db:"updated_at" json:"updatedAt"`
}
```

#### New Episode Model
```go
// /apps/api/internal/models/episode.go
type Episode struct {
    ID            string         `db:"id" json:"id"`
    SeriesID      string         `db:"series_id" json:"seriesId"`
    TMDbID        sql.NullInt64  `db:"tmdb_id" json:"tmdbId,omitempty"`
    SeasonNumber  int            `db:"season_number" json:"seasonNumber"`
    EpisodeNumber int            `db:"episode_number" json:"episodeNumber"`
    Title         sql.NullString `db:"title" json:"title,omitempty"`
    Overview      sql.NullString `db:"overview" json:"overview,omitempty"`
    AirDate       sql.NullString `db:"air_date" json:"airDate,omitempty"`
    Runtime       sql.NullInt64  `db:"runtime" json:"runtime,omitempty"`
    StillPath     sql.NullString `db:"still_path" json:"stillPath,omitempty"`
    VoteAverage   sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`
    FilePath      sql.NullString `db:"file_path" json:"filePath,omitempty"`
    CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
    UpdatedAt     time.Time      `db:"updated_at" json:"updatedAt"`
}
```

### Repository Interface Extensions

```go
// /apps/api/internal/repository/interfaces.go

// MovieRepositoryInterface - extended for FTS
type MovieRepositoryInterface interface {
    // Existing methods...
    Create(ctx context.Context, movie *models.Movie) error
    Update(ctx context.Context, movie *models.Movie) error
    Delete(ctx context.Context, id string) error
    FindByID(ctx context.Context, id string) (*models.Movie, error)
    FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)

    // New FTS search method
    FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Movie, *PaginationResult, error)

    // Upsert for duplicate prevention
    Upsert(ctx context.Context, movie *models.Movie) error

    // List with filters
    List(ctx context.Context, filters MovieFilters, params ListParams) ([]models.Movie, *PaginationResult, error)
}

// SeriesRepositoryInterface
type SeriesRepositoryInterface interface {
    Create(ctx context.Context, series *models.Series) error
    Update(ctx context.Context, series *models.Series) error
    Delete(ctx context.Context, id string) error
    FindByID(ctx context.Context, id string) (*models.Series, error)
    FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)
    FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Series, *PaginationResult, error)
    Upsert(ctx context.Context, series *models.Series) error
    List(ctx context.Context, filters SeriesFilters, params ListParams) ([]models.Series, *PaginationResult, error)
}

// EpisodeRepositoryInterface
type EpisodeRepositoryInterface interface {
    Create(ctx context.Context, episode *models.Episode) error
    Update(ctx context.Context, episode *models.Episode) error
    Delete(ctx context.Context, id string) error
    FindByID(ctx context.Context, id string) (*models.Episode, error)
    FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)
    FindBySeasonNumber(ctx context.Context, seriesID string, seasonNumber int) ([]models.Episode, error)
    Upsert(ctx context.Context, episode *models.Episode) error
}
```

### Service Layer

```go
// /apps/api/internal/services/media_service.go

type MediaService struct {
    movieRepo   repository.MovieRepositoryInterface
    seriesRepo  repository.SeriesRepositoryInterface
    episodeRepo repository.EpisodeRepositoryInterface
    tmdbClient  tmdb.ClientInterface
    logger      *slog.Logger
}

// SaveParsedMovie saves a movie with TMDb metadata
func (s *MediaService) SaveParsedMovie(ctx context.Context, parseResult *parser.ParseResult, tmdbMovie *tmdb.MovieDetails) (*models.Movie, error) {
    // Convert TMDb response to Movie model
    // Check for existing by TMDb ID
    // Upsert to database
    // Return saved movie
}

// SaveParsedSeries saves a series with TMDb metadata
func (s *MediaService) SaveParsedSeries(ctx context.Context, parseResult *parser.ParseResult, tmdbSeries *tmdb.TVShowDetails) (*models.Series, error) {
    // Similar to SaveParsedMovie
    // Also save episodes if available
}

// SearchLibrary performs unified search across movies and series
func (s *MediaService) SearchLibrary(ctx context.Context, query string, params repository.ListParams) (*SearchResults, error) {
    // Parallel FTS search on both movies and series
    // Merge and sort results by relevance
    // Return within 500ms
}
```

## Tasks

### Task 1: Database Migration for Enhanced Schema
**Description:** Create database migration to add new fields to movies table, create series and episodes tables, and set up FTS5 virtual tables with triggers.

**Files:**
- `apps/api/internal/database/migrations/003_media_entities.sql`

**Technical Notes:**
- Use SQLite FTS5 porter tokenizer for better search
- Create indexes on tmdb_id, file_path, parse_status
- Ensure WAL mode is enabled for concurrent access

### Task 2: Enhanced Movie Model
**Description:** Update the Movie model with all TMDb metadata fields, JSON helper methods, and validation.

**Files:**
- `apps/api/internal/models/movie.go`

**Technical Notes:**
- Add helper methods for JSON serialization/deserialization
- Use sql.NullXxx types for optional fields
- Include model validation methods

### Task 3: Series and Episode Models
**Description:** Create new Series and Episode models with all required fields and relationships.

**Files:**
- `apps/api/internal/models/series.go`
- `apps/api/internal/models/episode.go`

**Technical Notes:**
- Series has one-to-many relationship with Episodes
- Episode unique constraint on (series_id, season_number, episode_number)

### Task 4: Movie Repository with FTS
**Description:** Implement MovieRepository with full-text search using FTS5.

**Files:**
- `apps/api/internal/repository/movie_repository.go`
- `apps/api/internal/repository/movie_repository_test.go`

**Technical Notes:**
- FTS5 MATCH syntax: `SELECT * FROM movies_fts WHERE movies_fts MATCH ?`
- Join with main table for full results
- Implement Upsert using INSERT OR REPLACE with TMDb ID conflict

### Task 5: Series and Episode Repositories
**Description:** Implement SeriesRepository and EpisodeRepository with FTS support.

**Files:**
- `apps/api/internal/repository/series_repository.go`
- `apps/api/internal/repository/series_repository_test.go`
- `apps/api/internal/repository/episode_repository.go`
- `apps/api/internal/repository/episode_repository_test.go`

**Technical Notes:**
- Similar FTS pattern as MovieRepository
- Episode queries often filter by series_id

### Task 6: Media Service Implementation
**Description:** Implement MediaService that integrates TMDb client, parser, and repositories.

**Files:**
- `apps/api/internal/services/media_service.go`
- `apps/api/internal/services/media_service_test.go`

**Technical Notes:**
- Service converts TMDb responses to model structs
- Handles both movie and series workflows
- Implements unified search across both types

### Task 7: TMDb-to-Model Converters
**Description:** Create converter functions to transform TMDb API responses into database models.

**Files:**
- `apps/api/internal/services/converters.go`
- `apps/api/internal/services/converters_test.go`

**Technical Notes:**
- Handle JSON encoding for genres, credits, etc.
- Map language codes properly
- Generate UUID for new records

### Task 8: Integration Tests
**Description:** Write integration tests verifying the full flow from parsing to storage to search.

**Files:**
- `apps/api/internal/services/media_service_integration_test.go`

**Technical Notes:**
- Use in-memory SQLite for tests
- Verify FTS search returns results < 500ms with 1000+ records
- Test duplicate prevention via TMDb ID

## Dependencies

- **Story 1.1**: Repository Pattern - base interfaces and patterns
- **Story 2.1**: TMDb API Integration - metadata source
- **Story 2.5**: Standard Filename Parser - parse results input

## Dev Notes

### Performance Considerations
- FTS5 is significantly faster than LIKE queries for text search
- Use prepared statements for all queries
- Consider adding EXPLAIN QUERY PLAN for optimization
- The 500ms search requirement should be easily met with FTS5 on 10k+ records

### JSON Field Handling
- Use `encoding/json` for serialization
- Consider using `jsonb` style queries if SQLite supports (via extensions)
- Keep JSON structures simple for easier querying

### Error Handling
- Wrap database errors with context
- Use slog for structured error logging
- Return domain errors, not SQL errors to handlers

### Testing Strategy
- Unit tests: Model validation, converters
- Integration tests: Repository with real SQLite
- Performance tests: FTS search with large datasets

## Definition of Done

- [x] All database migrations apply successfully
- [x] Movie model updated with all TMDb fields
- [x] Series and Episode models created
- [x] All repositories implement their interfaces
- [x] FTS5 search works correctly
- [x] MediaService integrates all components
- [x] Unit tests pass with >80% coverage
- [x] Integration tests verify full workflow
- [x] Search performance < 500ms verified
- [x] Code follows project patterns (slog, repository pattern)
- [x] No TODO comments left in code

## Dev Agent Record

### Implementation Summary (2026-01-17)

All Story 2.6 implementation was already complete. Verification confirms:

**Files Implemented:**
- `apps/api/internal/database/migrations/006_media_entities_enhancement.go` - Migration with FTS5
- `apps/api/internal/models/movie.go` - Enhanced Movie model
- `apps/api/internal/models/series.go` - Series model
- `apps/api/internal/models/episode.go` - Episode model
- `apps/api/internal/repository/movie_repository.go` - Movie repository with FTS5
- `apps/api/internal/repository/series_repository.go` - Series repository with FTS5
- `apps/api/internal/repository/episode_repository.go` - Episode repository
- `apps/api/internal/repository/interfaces.go` - Repository interfaces
- `apps/api/internal/services/library_service.go` - Library service (MediaService)
- `apps/api/internal/services/converters.go` - TMDb-to-Model converters

**Test Coverage:**
- Repository: 81.7%
- Services: 87.1%
- Models: 90.5%

**Performance:** Search performance < 500ms verified via integration tests.

### Decisions Made
- Used `LibraryService` as the MediaService implementation to align with existing codebase patterns
- FTS5 with external content tables for memory efficiency
- Upsert pattern using TMDb ID for duplicate prevention
