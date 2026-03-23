# Story 8.7: Auto-Download Service

Status: ready-for-dev

## Story

As a **media collector**,
I want **the system to automatically download the best-scoring subtitle for my media**,
so that **I get Traditional Chinese subtitles without manual intervention**.

## Acceptance Criteria

1. **Given** a media item needing subtitles,
   **When** the subtitle engine processes it,
   **Then** it executes the full pipeline: Search → Score → Download → Convert → Place;
   **And** the media's subtitle_status is updated at each stage via repository.

2. **Given** the engine searches across all configured providers,
   **When** results are returned,
   **Then** providers are queried in parallel with `context.Context` for cancellation;
   **And** individual provider errors do not abort the entire search;
   **And** partial results are merged before scoring.

3. **Given** scored results are available,
   **When** the top-scored subtitle is downloaded,
   **Then** it is passed to the language detector (Story 8-4);
   **And** if zh-Hans or zh, it is converted via OpenCC (Story 8-5);
   **And** the final file is placed via the placer (Story 8-10).

4. **Given** the top-scored subtitle download fails,
   **When** the engine retries,
   **Then** it tries the next-best scored result;
   **And** continues until one succeeds or all are exhausted.

5. **Given** all subtitle download attempts are exhausted,
   **When** no subtitle could be successfully downloaded,
   **Then** the media's subtitle_status is set to `not_found`;
   **And** an SSE event `subtitle_status` is broadcast with status `not_found`;
   **And** the `subtitle_last_searched` timestamp is updated.

6. **Given** a subtitle is successfully placed,
   **When** the pipeline completes,
   **Then** subtitle_status is set to `found`;
   **And** subtitle_path, subtitle_language, subtitle_search_score are updated in DB;
   **And** an SSE event `subtitle_status` is broadcast with status `found`.

7. **Given** the engine is processing,
   **When** each pipeline stage transitions,
   **Then** an SSE `subtitle_status` event is broadcast with the current stage;
   **And** the event payload includes `{mediaId, mediaType, status, stage, message}`.

8. **Given** the engine is invoked as part of batch processing,
   **When** called from the batch service (Story 8-9),
   **Then** it accepts a `context.Context` and respects cancellation;
   **And** it returns a structured result (success/fail + metadata).

## Tasks / Subtasks

### Task 1: Define Engine Types (AC: #1, #7, #8)
- [ ] 1.1 Create `apps/api/internal/subtitle/engine.go`
- [ ] 1.2 Define `Engine` struct with dependencies: `providers []SubtitleProvider`, `scorer *Scorer`, `detector *Detector`, `converter *Converter`, `placer *Placer`, `sseHub *sse.Hub`
- [ ] 1.3 Define `EngineResult` struct: `Success bool`, `SubtitlePath string`, `Language string`, `Score float64`, `Error error`, `ProviderUsed string`
- [ ] 1.4 Define `PipelineStage` type with constants: `StageSearching`, `StageScoring`, `StageDownloading`, `StageConverting`, `StagePlacing`, `StageComplete`, `StageFailed`
- [ ] 1.5 Define `NewEngine(deps) *Engine` constructor

### Task 2: Implement Parallel Provider Search (AC: #2)
- [ ] 2.1 Create `search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)` method
- [ ] 2.2 Use `errgroup.Group` with context for parallel provider calls
- [ ] 2.3 Collect results into thread-safe slice using `sync.Mutex`
- [ ] 2.4 Log individual provider errors at `slog.Warn` level, do not fail pipeline
- [ ] 2.5 Return merged results; return error only if all providers fail

### Task 3: Implement Download with Fallback (AC: #3, #4)
- [ ] 3.1 Create `downloadBestMatch(ctx context.Context, scored []ScoredResult) ([]byte, *ScoredResult, error)` method
- [ ] 3.2 Iterate scored results from best to worst
- [ ] 3.3 Call `provider.Download(ctx, id)` for each attempt
- [ ] 3.4 On success, return downloaded bytes and the matched result
- [ ] 3.5 On failure, log warning and continue to next result
- [ ] 3.6 If all exhausted, return `ErrAllDownloadsFailed` error

### Task 4: Implement Convert Step (AC: #3)
- [ ] 4.1 Create `convertIfNeeded(data []byte) ([]byte, string, error)` method
- [ ] 4.2 Call `detector.DetectLanguage(data)` to get language
- [ ] 4.3 If zh-Hans or zh, call `converter.Convert(data)` and return converted data with "zh-Hant"
- [ ] 4.4 If already zh-Hant, return as-is
- [ ] 4.5 If non-Chinese, return error (should have been filtered by scorer)

### Task 5: Implement Full Pipeline Orchestration (AC: #1, #6, #7)
- [ ] 5.1 Create `Process(ctx context.Context, mediaID int64, mediaType string, query SubtitleQuery) EngineResult` method
- [ ] 5.2 Broadcast SSE at each stage transition
- [ ] 5.3 Stage 1: Set status `searching`, search providers
- [ ] 5.4 Stage 2: Score results
- [ ] 5.5 Stage 3: Set status `searching`, download with fallback
- [ ] 5.6 Stage 4: Convert if needed
- [ ] 5.7 Stage 5: Place file via placer (Story 8-10)
- [ ] 5.8 Stage 6: Update DB via `UpdateSubtitleStatus` with final state
- [ ] 5.9 Return `EngineResult` with outcome details

### Task 6: Implement SSE Broadcasting (AC: #7)
- [ ] 6.1 Create `broadcastStatus(mediaID int64, mediaType string, stage PipelineStage, message string)` helper
- [ ] 6.2 Use `sse.EventSubtitleProgress` event type
- [ ] 6.3 Payload: `{mediaId, mediaType, status, stage, message}`
- [ ] 6.4 Nil-check sseHub before broadcasting (support headless/test mode)

### Task 7: Implement Failure Handling (AC: #5)
- [ ] 7.1 On all downloads exhausted, call `UpdateSubtitleStatus(mediaID, "not_found")`
- [ ] 7.2 Update `subtitle_last_searched` timestamp
- [ ] 7.3 Broadcast SSE `subtitle_status` with `not_found`
- [ ] 7.4 Return `EngineResult{Success: false, Error: ErrAllDownloadsFailed}`

### Task 8: Create Subtitle Service Layer (AC: #1, #8)
- [ ] 8.1 Create `apps/api/internal/services/subtitle_service.go`
- [ ] 8.2 Define `SubtitleServiceInterface` with `ProcessMedia`, `SearchSubtitles`, `DownloadSubtitle` methods
- [ ] 8.3 Implement `SubtitleService` wrapping the engine
- [ ] 8.4 Accept repository dependencies for DB operations
- [ ] 8.5 Handle movie vs. series media type routing

### Task 9: Write Tests (AC: #1–#8)
- [ ] 9.1 Create `apps/api/internal/subtitle/engine_test.go`
- [ ] 9.2 Create mock implementations: `MockSubtitleProvider`, `MockDetector`, `MockConverter`, `MockPlacer`
- [ ] 9.3 Test full pipeline happy path: search → score → download → convert → place
- [ ] 9.4 Test fallback: first download fails, second succeeds
- [ ] 9.5 Test all downloads exhausted → not_found
- [ ] 9.6 Test partial provider failure (one provider errors, others succeed)
- [ ] 9.7 Test all providers fail → error
- [ ] 9.8 Test context cancellation aborts pipeline
- [ ] 9.9 Test SSE events are broadcast at each stage
- [ ] 9.10 Test zh-Hans content triggers conversion
- [ ] 9.11 Test zh-Hant content skips conversion
- [ ] 9.12 Create `apps/api/internal/services/subtitle_service_test.go` with service-level tests
- [ ] 9.13 Ensure >80% coverage on engine.go

## Dev Notes

### Architecture & Patterns
- The engine is the **pipeline orchestrator** — it coordinates providers, scorer, detector, converter, and placer
- Each dependency is injected via constructor, enabling full mockability
- Pattern follows existing `scanner_service.go` which also orchestrates multi-step background work with SSE progress
- `errgroup.Group` pattern for parallel provider calls matches Go best practices
- `EngineResult` struct enables the batch service (Story 8-9) to aggregate outcomes

### Project Structure Notes
- Engine: `apps/api/internal/subtitle/engine.go` (pipeline logic)
- Service: `apps/api/internal/services/subtitle_service.go` (business layer)
- Dependencies: scorer.go (8-6), detector.go (8-4), converter.go (8-5), placer.go (8-10)
- Repository: `UpdateSubtitleStatus` from Story 0-2
- SSE: `apps/api/internal/sse/hub.go` — uses `EventSubtitleProgress` event type

### References
- PRD: P1-017 (Auto-download best subtitle)
- Story 8-6: Scorer provides sorted `[]ScoredResult`
- Story 8-4: `detector.DetectLanguage()` returns language tag
- Story 8-5: `converter.Convert()` with OpenCC s2twp profile
- Story 8-10: `placer.Place()` handles file output
- Story 0-2: `UpdateSubtitleStatus` repository method
- Story 0-3: SSE hub with `EventSubtitleProgress`
- Existing pattern: `apps/api/internal/services/scanner_service.go` for background processing with SSE

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
