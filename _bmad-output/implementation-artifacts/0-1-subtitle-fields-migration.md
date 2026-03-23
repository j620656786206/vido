# Story 0-1: Subtitle Fields Migration

## Status: done

## Story
As a developer, I need subtitle tracking fields in the movies and series tables so that Epic 7 (Scanner) and Epic 8 (Subtitle Engine) can track subtitle search status per media item.

## Acceptance Criteria
- [x] Migration 018 adds 5 columns to movies table: subtitle_status, subtitle_path, subtitle_language, subtitle_last_searched, subtitle_search_score
- [x] Migration 018 adds 5 columns to series table (same fields)
- [x] Indexes created on subtitle_status for both tables
- [x] SubtitleStatus type defined with 4 states: not_searched, searching, found, not_found
- [x] Movie struct updated with subtitle fields using proper db/json tags
- [x] Series struct updated with subtitle fields using proper db/json tags
- [x] `go build` passes

## Tasks
- [x] Task 1: Create migration file `018_add_subtitle_fields.go`
- [x] Task 2: Add `SubtitleStatus` type to `models/movie.go`
- [x] Task 3: Add subtitle fields to Movie struct
- [x] Task 4: Add subtitle fields to Series struct

## Dev Agent Record

### Completion Notes
- Migration follows existing pattern from `017_create_backups_table.go`
- SubtitleStatus type placed in `movie.go` alongside ParseStatus (shared type for both Movie and Series)
- Fields use `sql.NullString`, `sql.NullTime`, `sql.NullFloat64` for nullable columns
- Default value `not_searched` set in migration SQL

### File List
| Action | File |
|--------|------|
| CREATE | `apps/api/internal/database/migrations/018_add_subtitle_fields.go` |
| MODIFY | `apps/api/internal/models/movie.go` |
| MODIFY | `apps/api/internal/models/series.go` |

### Change Log
- 2026-03-23: Implemented migration + model updates. Build passes.
