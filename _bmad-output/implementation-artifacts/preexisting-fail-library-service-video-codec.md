# Story: Fix library_service_test.go Migration Drift

Status: review

## Story

As a developer,
I want the library_service_test.go test database schema to stay in sync with production migrations,
so that all 14 tests pass without "no such column" errors from migration drift.

## Acceptance Criteria

1. Given `library_service_test.go`, when all 14 tests run, then zero "no such column: video_codec" or "no such column: file_size" errors occur
2. Given `setupTestDB()` is called, when it creates the test database, then the schema matches the production schema (all migrations applied)
3. Given future migrations are added, when library service tests run, then they automatically include new columns without manual schema updates
4. Given the migration runner is used in tests, when the test database is created, then FTS5 virtual tables and triggers still function correctly
5. Given all library service tests pass, when `go test ./internal/services/ -run TestLibraryService -v` is run, then all 14 tests pass

## Tasks / Subtasks

- [x] Task 1: Replace hardcoded schema with migration runner (AC: #1, #2, #3)
  - [x] 1.1 Refactor `setupTestDB()` in `apps/api/internal/services/library_service_test.go` to use `migrations.NewRunner(db)` + `runner.Up(ctx)` instead of hardcoded CREATE TABLE statements
  - [x] 1.2 Add import: `"github.com/vido/api/internal/database/migrations"` and blank import `_ "github.com/vido/api/internal/database/migrations"` (to trigger init() registration of all migrations including 021)
  - [x] 1.3 Remove the entire hardcoded schema block (lines ~36-217): CREATE TABLE movies, CREATE TABLE series, CREATE VIRTUAL TABLE movies_fts, series_fts, all triggers, all indexes
  - [x] 1.4 Replace with: `runner, err := migrations.NewRunner(db)` → `require.NoError(t, err)` → `err = runner.Up(context.Background())` → `require.NoError(t, err)`

- [x] Task 2: Verify all 14 tests pass (AC: #1, #4, #5)
  - [x] 2.1 Run: `cd apps/api && go test ./internal/services/ -run TestLibraryService -v`
  - [x] 2.2 Verify all 14 pass: SaveMovieFromTMDb, SearchLibrary, GetMovieByID, ListLibrary, DeleteMovie, GetRecentlyAdded, FilterByGenre, FilterByYearRange, GetDistinctGenres, GetLibraryStats, CombinedFilters, SaveSeriesFromTMDb, DeleteSeries, GetSeriesByID
  - [x] 2.3 Verify FTS5 search still works (SearchLibrary test exercises FTS)

- [x] Task 3: Verify no regressions (AC: #5)
  - [x] 3.1 Run full services test suite: `cd apps/api && go test ./internal/services/ -v`
  - [x] 3.2 Run full test suite: `cd apps/api && go test ./...`

## Dev Notes

### Root Cause

`setupTestDB()` at `apps/api/internal/services/library_service_test.go:21-217` manually creates `movies` and `series` tables with hardcoded DDL. Migration 021 (`apps/api/internal/database/migrations/021_media_tech_info.go`) added 6 columns to `movies` and 7 to `series` (including `file_size`), but `setupTestDB()` was never updated.

The repository layer's `movieSelectColumns` and `seriesSelectColumns` constants (`movie_repository.go:612`, `series_repository.go:589`) reference these columns in every SELECT, causing:
- **Movie tests:** `SQL logic error: no such column: video_codec (1)`
- **Series tests:** `SQL logic error: no such column: file_size (1)`

### Missing Columns

**movies table missing:** `video_codec TEXT`, `video_resolution TEXT`, `audio_codec TEXT`, `audio_channels INTEGER`, `subtitle_tracks TEXT`, `hdr_format TEXT`

**series table missing:** `file_size INTEGER`, `video_codec TEXT`, `video_resolution TEXT`, `audio_codec TEXT`, `audio_channels INTEGER`, `subtitle_tracks TEXT`, `hdr_format TEXT`

### Recommended Fix: Use Migration Runner

Replace hardcoded DDL with the migration runner. This is the **drift-proof** approach — future migrations automatically apply to test schema.

```go
func setupTestDB(t *testing.T) *sql.DB {
    tmpFile, err := os.CreateTemp("", "test_library_*.db")
    require.NoError(t, err)
    tmpFile.Close()
    t.Cleanup(func() { os.Remove(tmpFile.Name()) })

    db, err := sql.Open("sqlite", tmpFile.Name()+"?_pragma=foreign_keys(1)")
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })

    runner, err := migrations.NewRunner(db)
    require.NoError(t, err)
    err = runner.Up(context.Background())
    require.NoError(t, err)

    return db
}
```

### Migration Runner API

- **Package:** `github.com/vido/api/internal/database/migrations`
- **`NewRunner(db *sql.DB)`** — creates runner, auto-creates `schema_migrations` table
- **`runner.Up(ctx)`** — applies all pending migrations in version order
- **Registration:** Each migration file has `func init() { Register(...) }` — importing the package triggers registration
- **All 21 migrations** will run: creates tables, FTS, triggers, indexes — replaces entire hardcoded schema

### Import Pattern

Must blank-import the migrations package to trigger `init()` registration:
```go
import (
    "github.com/vido/api/internal/database/migrations"
    _ "github.com/vido/api/internal/database/migrations" // register all migrations
)
```

Note: If the package is already used directly (e.g., `migrations.NewRunner`), the blank import is redundant — but add a comment either way for clarity.

### What NOT to Do

- DO NOT manually add the 6+7 missing columns to hardcoded DDL — this will drift again with the next migration
- DO NOT skip FTS verification — the migration runner creates FTS tables and triggers, verify SearchLibrary test still passes
- DO NOT modify any migration files — the migrations are correct, only the test setup is wrong

### References

- [Source: apps/api/internal/services/library_service_test.go:21-217] — Broken setupTestDB() with hardcoded schema
- [Source: apps/api/internal/database/migrations/021_media_tech_info.go] — Migration that added the missing columns
- [Source: apps/api/internal/repository/movie_repository.go:612-621] — movieSelectColumns referencing video_codec
- [Source: apps/api/internal/repository/series_repository.go:589-599] — seriesSelectColumns referencing file_size + tech columns
- [Source: apps/api/internal/database/migrations/runner.go:32] — NewRunner() API
- [Source: apps/api/internal/database/migrations/registry.go:27] — Register() for init() pattern

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context) — SM agent (Bob) create-story workflow, YOLO mode

### Debug Log References

### Completion Notes List

- Preexisting test failure — migration 021 added columns but test hardcoded schema was not updated
- Recommended fix: use migration runner instead of hardcoded DDL (drift-proof)
- 14 tests affected (11 movie, 3 series)
- Simple 1-file fix, minimal risk
- ✅ Replaced 200-line hardcoded DDL with 4-line migration runner call
- ✅ Key discovery: `NewRunner()` creates empty runner — must call `runner.RegisterAll(migrations.GetAll())` to load global registry migrations
- ✅ All 14 LibraryService tests pass, FTS5 search verified
- ✅ Full `go test ./...` regression: only pre-existing failures (download_handler 4 tests, setup_service 1 test), zero new regressions
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Change Log

- 2026-04-09: Replaced hardcoded DDL in `setupTestDB()` with migration runner (`migrations.NewRunner` + `RegisterAll` + `Up`). Removed ~200 lines of manual CREATE TABLE/FTS/trigger SQL, replaced with drift-proof migration runner. All 14 tests pass.

### File List

- `apps/api/internal/services/library_service_test.go` — Replace setupTestDB() hardcoded schema with migration runner
