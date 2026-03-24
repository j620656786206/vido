# Story 8.9: Batch Subtitle Processing

Status: ready-for-dev

## Story

As a **media collector**,
I want **to search and download subtitles for an entire season or my whole library in one operation**,
so that **I don't have to manually trigger subtitle search for each media item individually**.

## Acceptance Criteria

1. **Given** a user triggers batch subtitle processing,
   **When** the scope is "season",
   **Then** all episodes in the specified season are queued for subtitle processing;
   **And** items with subtitle_status `found` are skipped.

2. **Given** a user triggers batch subtitle processing,
   **When** the scope is "library",
   **Then** all media items (movies + series episodes) with subtitle_status `not_searched` or `not_found` are queued;
   **And** items with subtitle_status `found` are skipped.

3. **Given** batch processing is in progress,
   **When** progress is reported via SSE,
   **Then** the event payload includes `{totalItems, currentItem, currentIndex, successCount, failCount, status}`;
   **And** events are broadcast for each item completion.

4. **Given** the backend `POST /api/v1/subtitles/batch` endpoint,
   **When** called with a valid scope,
   **Then** it returns 202 Accepted with `{batchId, totalItems}`;
   **And** processing continues in the background;
   **And** the request body accepts `{scope: "season"|"library", seasonId?: number}`.

5. **Given** batch processing is running,
   **When** providers have rate limits,
   **Then** the system respects per-provider rate limits;
   **And** introduces delays between items to avoid throttling;
   **And** targets 100 items processed in under 10 minutes.

6. **Given** batch processing encounters failures on individual items,
   **When** a single item fails,
   **Then** it is marked `not_found` and processing continues with the next item;
   **And** the failure is logged and counted in progress;
   **And** the batch does not abort.

7. **Given** a batch is in progress,
   **When** the user requests another batch,
   **Then** the system returns 409 Conflict with a message indicating a batch is already running;
   **And** includes the current batch progress.

8. **Given** batch processing completes,
   **When** all items are processed,
   **Then** a final SSE event is broadcast with `status: "complete"` and summary counts;
   **And** the batch is removed from active state.

9. **Given** batch processing encounters a media item with `production_countries` containing "CN",
   **When** the engine processes that item's subtitle,
   **Then** OpenCC s2twp conversion is skipped and Simplified Chinese is preserved;
   **And** the subtitle file uses `.zh-Hans.srt` extension.

## Tasks / Subtasks

### Task 1: Define Batch Types (AC: #3, #4)
- [ ] 1.1 Create `apps/api/internal/subtitle/batch.go`
- [ ] 1.2 Define `BatchScope` type with constants: `ScopeSeason`, `ScopeLibrary`
- [ ] 1.3 Define `BatchRequest` struct: `Scope BatchScope`, `SeasonID *int64`
- [ ] 1.4 Define `BatchProgress` struct: `BatchID string`, `TotalItems int`, `CurrentIndex int`, `CurrentItem string`, `SuccessCount int`, `FailCount int`, `Status string`
- [ ] 1.5 Define `BatchResult` struct: `BatchID string`, `TotalItems int`, `SuccessCount int`, `FailCount int`, `Duration time.Duration`, `FailedItems []FailedItem`
- [ ] 1.6 Define `FailedItem` struct: `MediaID int64`, `MediaType string`, `Title string`, `Error string`

### Task 2: Implement Batch Processor (AC: #1, #2, #5, #6)
- [ ] 2.1 Define `BatchProcessor` struct with dependencies: `engine *Engine`, `movieRepo`, `seriesRepo`, `sseHub *sse.Hub`
- [ ] 2.2 Implement `Process(ctx context.Context, req BatchRequest) (*BatchResult, error)`
- [ ] 2.3 Implement `collectItems(ctx, scope, seasonID)` to gather media items needing subtitles (include production_countries)
- [ ] 2.4 For season scope: query episodes by seasonID where subtitle_status != 'found'
- [ ] 2.5 For library scope: use `FindNeedingSubtitleSearch` from Story 0-2 repositories
- [ ] 2.6 Process items sequentially (to respect rate limits)
- [ ] 2.7 Add configurable delay between items (default 3 seconds)
- [ ] 2.8 Call `engine.Process()` for each item
- [ ] 2.9 Aggregate success/fail counts
- [ ] 2.10 Continue on individual item failure, log and track

### Task 3: Implement Rate Limiting (AC: #5)
- [ ] 3.1 Create per-provider rate limiter using `golang.org/x/time/rate`
- [ ] 3.2 Configure limits: Assrt 1 req/2s, OpenSubtitles 1 req/s, Zimuku 1 req/3s
- [ ] 3.3 Apply rate limiting in the batch delay between items
- [ ] 3.4 Expose rate limit config via `BatchConfig` struct

### Task 4: Implement SSE Progress Broadcasting (AC: #3, #8)
- [ ] 4.1 Broadcast progress event after each item completes
- [ ] 4.2 Use `sse.EventSubtitleProgress` event type with batch-specific payload
- [ ] 4.3 Include: totalItems, currentIndex, currentItem title, successCount, failCount
- [ ] 4.4 Broadcast final "complete" event with summary when batch finishes

### Task 5: Implement Batch Concurrency Guard (AC: #7)
- [ ] 5.1 Add `activeBatch` field to `BatchProcessor` with `sync.Mutex` protection
- [ ] 5.2 Check if batch is already running before starting new one
- [ ] 5.3 Return current progress if batch is active
- [ ] 5.4 Clear active batch on completion/error

### Task 6: Implement Context Cancellation (AC: #6)
- [ ] 6.1 Accept `context.Context` in `Process()` method
- [ ] 6.2 Check `ctx.Done()` between items
- [ ] 6.3 On cancellation, return partial results with status "cancelled"
- [ ] 6.4 Broadcast cancellation SSE event

### Task 7: Create Batch Handler (AC: #4, #7)
- [ ] 7.1 Add batch methods to `apps/api/internal/handlers/subtitle_handler.go`
- [ ] 7.2 Implement `POST /api/v1/subtitles/batch` handler
- [ ] 7.3 Validate request: scope required, seasonId required for season scope
- [ ] 7.4 Return 202 Accepted with batchId and totalItems
- [ ] 7.5 Return 409 Conflict if batch already running (include progress)
- [ ] 7.6 Launch batch processing in background goroutine
- [ ] 7.7 Implement `GET /api/v1/subtitles/batch/status` to check current batch progress
- [ ] 7.8 Register routes in router

### Task 8: Add Batch to Subtitle Service (AC: #1, #2)
- [ ] 8.1 Add `StartBatch(ctx, req)` method to `SubtitleServiceInterface`
- [ ] 8.2 Add `GetBatchStatus()` method to `SubtitleServiceInterface`
- [ ] 8.3 Implement in `SubtitleService` delegating to `BatchProcessor`

### Task 9: Write Tests (AC: #1–#8)
- [ ] 9.1 Create `apps/api/internal/subtitle/batch_test.go`
- [ ] 9.2 Test season scope: collects correct episodes, skips `found` items
- [ ] 9.3 Test library scope: collects movies + series, skips `found` items
- [ ] 9.4 Test sequential processing with delays
- [ ] 9.5 Test individual item failure does not abort batch
- [ ] 9.6 Test concurrency guard: second batch returns 409
- [ ] 9.7 Test context cancellation stops processing
- [ ] 9.8 Test SSE progress events are broadcast per item
- [ ] 9.9 Test final complete event includes summary
- [ ] 9.10 Test empty item list (all already found) returns immediately
- [ ] 9.11 Test handler returns 202 with batchId
- [ ] 9.12 Ensure >80% coverage on batch.go

## Dev Notes

### Architecture & Patterns
- Batch processor is deliberately sequential per item to simplify rate limiting; parallelism happens inside the engine (parallel provider search)
- Concurrency guard uses a simple mutex + active batch pointer — no need for a full job queue since only one batch at a time is allowed
- The 3-second default delay between items provides natural rate limiting; per-provider rate limiters add additional safety
- Background goroutine for processing follows existing pattern in `scan_scheduler.go`
- 100 items in 10 minutes = 6 seconds per item budget, well within the 3-second delay + pipeline time

### Project Structure Notes
- Batch logic: `apps/api/internal/subtitle/batch.go`
- Handler: extends `apps/api/internal/handlers/subtitle_handler.go` (from Story 8-8)
- Service: extends `SubtitleService` from Story 8-7
- Dependencies: Engine (8-7), Movie/Series repos (0-2), SSE Hub (0-3)
- Rate limiter: may need `golang.org/x/time/rate` dependency

### References
- PRD: P1-019 (Batch subtitle processing)
- Story 8-7: `Engine.Process()` handles individual media item pipeline
- Story 0-2: `FindNeedingSubtitleSearch` repository method for library scope
- NFR: 100 items < 10 minutes performance target
- Existing pattern: `apps/api/internal/services/scan_scheduler.go` for background processing

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
