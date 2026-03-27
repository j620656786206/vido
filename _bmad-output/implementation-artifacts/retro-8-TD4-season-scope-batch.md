# Story retro-8-TD4: Season Scope Batch Subtitle Processing

Status: done

## Story

As a user,
I want to trigger batch subtitle search for all episodes in a specific season,
so that I can efficiently find subtitles for a whole season at once.

## Acceptance Criteria

1. `POST /api/v1/subtitles/batch` with `{"scope": "season", "season_id": 123}` starts a batch job for that season's episodes
2. Only episodes needing subtitles (`subtitle_status != "found"`) are included in the batch
3. Batch progress SSE events report per-episode progress (same format as existing batch scopes)
4. Season with no eligible episodes returns an appropriate response (empty batch / informative message)
5. All existing batch tests continue to pass without modification

## Tasks / Subtasks

- [x] Task 1: Add `CollectEpisodesBySeasonID` to `BatchCollector` interface (AC: 1)
  - [x] 1.1 Add `CollectEpisodesBySeasonID(ctx context.Context, seasonID int64) ([]BatchItem, error)` to the `BatchCollector` interface in `batch.go`
- [x] Task 2: Implement `CollectEpisodesBySeasonID` in `RepoCollector` (AC: 1, 2)
  - [x] 2.1 Query episodes by `season_id` where `subtitle_status` indicates subtitles are needed
  - [x] 2.2 Convert each `Episode` to a `BatchItem` with proper `MediaID`, `MediaType`, `Title`, `MediaFilePath`
  - [x] 2.3 Return empty slice (not error) when no eligible episodes exist
- [x] Task 3: Implement `collectSeasonItems()` in batch processor (AC: 1, 2, 3, 4)
  - [x] 3.1 Add `collectSeasonItems(ctx context.Context, seasonID int64) ([]BatchItem, error)` method
  - [x] 3.2 Call `collector.CollectEpisodesBySeasonID(ctx, seasonID)`
  - [x] 3.3 Handle empty result — return appropriate response for "no eligible episodes"
- [x] Task 4: Wire into `collectItems()` switch case (AC: 1, 3)
  - [x] 4.1 Replace `ScopeSeason` case (currently returns `"season scope not yet implemented"`) with call to `collectSeasonItems()`
  - [x] 4.2 Pass `req.SeasonID` (dereferenced) to the method
- [x] Task 5: Add unit tests for season scope (AC: 1, 2, 4, 5)
  - [x] 5.1 Test `CollectEpisodesBySeasonID` — returns only episodes needing subtitles for the given season
  - [x] 5.2 Test `CollectEpisodesBySeasonID` — returns empty slice for season with no eligible episodes
  - [x] 5.3 Test `collectItems()` with `ScopeSeason` — full integration through the switch case
  - [x] 5.4 Test batch processing with season scope — verify SSE progress events fire per episode
- [x] Task 6: Run full test suite for regressions (AC: 5)
  - [x] 6.1 Run `nx test api` — all existing batch tests pass
  - [x] 6.2 Verify `ScopeLibrary` batch tests unchanged and passing

## Dev Notes

### Key Files

| File | What to Change |
|------|---------------|
| `apps/api/internal/subtitle/batch.go` | Add interface method, implement `collectSeasonItems()`, wire switch case (~line 306-308) |
| `apps/api/internal/subtitle/batch.go` | Implement `CollectEpisodesBySeasonID` on `RepoCollector` |
| `apps/api/internal/subtitle/batch_test.go` | Add season scope tests (mock collector + integration) |

### Episode Query Pattern

The `Episode` model has a `SeasonID` foreign key. The query should look like:

```go
func (c *RepoCollector) CollectEpisodesBySeasonID(ctx context.Context, seasonID int64) ([]BatchItem, error) {
    // Query episodes by season_id where subtitle_status needs subtitles
    // Convert to []BatchItem
}
```

Filter episodes where `subtitle_status` is NOT `"found"` — use the same status check pattern as `CollectSeriesNeedingSubtitles`.

### Episode to BatchItem Conversion

Each episode should map to a `BatchItem` with:
- `MediaID` = episode ID
- `MediaType` = appropriate episode type constant
- `Title` = episode title (or series + season + episode number for display)
- `MediaFilePath` = episode's file path

### Existing Infrastructure — Do NOT Change

- **Handler** (`subtitle_handler.go`) — already validates `season_id` is present for season scope and passes it via `BatchRequest.SeasonID`
- **Batch processor** — already handles sequential processing, SSE progress events, cancellation
- **`BatchScope` enum** — `ScopeSeason` already defined
- **`BatchRequest` struct** — `SeasonID *int64` field already exists

### Repository Note

`EpisodeRepository` has `FindBySeasonNumber(ctx, seriesID, seasonNumber)` but NOT `FindBySeasonID`. The `RepoCollector` may need to query the database directly or use a combination of existing repository methods. Check what query methods are available on the repository and add a new one if needed.

### References

- [Source: apps/api/internal/subtitle/batch.go] — BatchCollector interface, RepoCollector, collectItems() switch
- [Source: apps/api/internal/handlers/subtitle_handler.go] — StartBatch handler (do not modify)
- [Source: apps/api/internal/models/episode.go] — Episode model with SeasonID field
- [Source: apps/api/internal/models/season.go] — Season model
- [Source: epic-8-retro-2026-03-25.md#TD4] — Retro action item origin

## Change Log

- 2026-03-26: Story created — ready-for-dev
- 2026-03-27: Implementation complete — season scope with EpisodeSeasonFinder, RepoCollector, 7 tests
- 2026-03-27: QA pass — all code paths verified, SQL injection-safe, interface consistent
- 2026-03-27: CR fixes — SeasonID changed from *int64 to *string (UUID), nil episodeRepo guard added
- 2026-03-27: Status → done
