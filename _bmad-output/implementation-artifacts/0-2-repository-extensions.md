# Story 0-2: Repository Extensions

## Status: done

## Story
As a developer, I need extended repository methods for bulk operations and subtitle queries so that Epic 7 (Scanner) can batch-insert scan results and Epic 8 (Subtitle Engine) can query/update subtitle status.

## Acceptance Criteria
- [x] BulkCreate method on MovieRepository and SeriesRepository (transaction-based batch insert)
- [x] FindByParseStatus method on both repositories
- [x] UpdateSubtitleStatus method on both repositories (atomic subtitle field update)
- [x] FindBySubtitleStatus method on both repositories
- [x] FindNeedingSubtitleSearch method on both repositories (not_searched OR stale)
- [x] Interface definitions updated in `interfaces.go`
- [x] Compile-time assertions pass
- [x] `go build` passes

## Tasks
- [x] Task 1: Add 5 method signatures to MovieRepositoryInterface
- [x] Task 2: Add 5 method signatures to SeriesRepositoryInterface
- [x] Task 3: Add scanMovie helper + movieSelectColumns constant
- [x] Task 4: Implement 5 methods on MovieRepository
- [x] Task 5: Add scanSeries helper + seriesSelectColumns constant
- [x] Task 6: Implement 5 methods on SeriesRepository

## Dev Agent Record

### Completion Notes
- BulkCreate uses `BeginTx` + prepared statement loop + `Commit` with `defer tx.Rollback()`
- UpdateSubtitleStatus sets `subtitle_last_searched` to `time.Now()` automatically
- FindNeedingSubtitleSearch uses OR: `subtitle_status = 'not_searched' OR (subtitle_last_searched < ?)`
- New scan helpers (`scanMovie`/`scanSeries`) include all subtitle columns
- Existing methods retain their original column lists to avoid breaking changes

### File List
| Action | File |
|--------|------|
| MODIFY | `apps/api/internal/repository/interfaces.go` |
| MODIFY | `apps/api/internal/repository/movie_repository.go` |
| MODIFY | `apps/api/internal/repository/series_repository.go` |

### Change Log
- 2026-03-23: Implemented 5 methods on both repositories. Build passes.
