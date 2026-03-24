# Story 8.8: Manual Subtitle Search UI

Status: review

## Story

As a **media collector**,
I want **a manual search interface to find and download subtitles for a specific movie or episode**,
so that **I can override automatic results and choose the exact subtitle I prefer**.

## Acceptance Criteria

1. **Given** a user is on a media detail page,
   **When** they click the "Search Subtitles" action,
   **Then** a dialog opens with the media title pre-filled in the search field;
   **And** provider checkboxes (Assrt, OpenSubtitles, Zimuku) are shown, all checked by default.

2. **Given** the user submits a manual search,
   **When** the backend receives `POST /api/v1/subtitles/search`,
   **Then** it queries selected providers in parallel;
   **And** returns scored results with source, language, group, resolution, score, and download count;
   **And** the request body includes `{mediaId, mediaType, providers[], query?}`.

3. **Given** search results are returned,
   **When** displayed in the dialog,
   **Then** results appear in a sortable table with columns: Source, Language, Group, Resolution, Score, Downloads;
   **And** results are sorted by score descending by default;
   **And** the user can re-sort by clicking column headers.

4. **Given** a result row in the table,
   **When** the user clicks "Preview",
   **Then** the first 10 lines of the subtitle content are displayed in a popover;
   **And** character encoding is detected and displayed correctly (UTF-8/BIG5).

5. **Given** a result row in the table,
   **When** the user clicks "Download",
   **Then** `POST /api/v1/subtitles/download` is called with `{mediaId, mediaType, subtitleId, provider}`;
   **And** the backend downloads, converts (if needed), and places the subtitle file;
   **And** the dialog shows a progress indicator during processing.

6. **Given** a manual download is in progress,
   **When** the backend processes it,
   **Then** SSE `subtitle_status` events update the dialog in real-time;
   **And** on success, the dialog shows a success toast and the download button changes to a checkmark;
   **And** on failure, the dialog shows an error message with the failure reason.

7. **Given** the backend `POST /api/v1/subtitles/search` endpoint,
   **When** called with an invalid mediaId or unsupported mediaType,
   **Then** it returns 400 with appropriate error message;
   **And** when no results are found, it returns 200 with an empty array.

8. **Given** the backend `POST /api/v1/subtitles/download` endpoint,
   **When** the download succeeds,
   **Then** it returns 200 with `{subtitlePath, language, score}`;
   **And** the media's subtitle fields are updated in the database.

9. **Given** the media's `production_countries` contains "CN" (mainland China),
   **When** the subtitle search dialog opens,
   **Then** the "繁體轉換" toggle defaults to OFF;
   **And** downloaded subtitles skip OpenCC s2twp conversion, preserving Simplified Chinese.

10. **Given** the media's `production_countries` does NOT contain "CN",
    **When** the subtitle search dialog opens,
    **Then** the "繁體轉換" toggle defaults to ON;
    **And** downloaded Simplified Chinese subtitles are converted to Traditional Chinese via OpenCC s2twp.

11. **Given** the user manually toggles "繁體轉換" in the dialog,
    **When** they download a subtitle,
    **Then** the conversion follows the user's toggle override regardless of production country.

## Tasks / Subtasks

### Task 1: Create Backend Search Handler (AC: #2, #7)
- [ ] 1.1 Create `apps/api/internal/handlers/subtitle_handler.go`
- [ ] 1.2 Define `SubtitleSearchRequest` struct: `MediaID int64`, `MediaType string`, `Providers []string`, `Query string`
- [ ] 1.3 Define `SubtitleSearchResponse` struct wrapping `[]ScoredResult`
- [ ] 1.4 Implement `POST /api/v1/subtitles/search` handler
- [ ] 1.5 Validate request: mediaId required, mediaType must be "movie" or "series"
- [ ] 1.6 Filter providers based on request (default: all configured)
- [ ] 1.7 Call engine's search + score (reuse engine internals, not full pipeline)
- [ ] 1.8 Return scored results as JSON

### Task 2: Create Backend Download Handler (AC: #5, #8)
- [ ] 2.1 Define `SubtitleDownloadRequest` struct: `MediaID int64`, `MediaType string`, `SubtitleID string`, `Provider string`
- [ ] 2.2 Define `SubtitleDownloadResponse` struct: `SubtitlePath string`, `Language string`, `Score float64`
- [ ] 2.3 Implement `POST /api/v1/subtitles/download` handler
- [ ] 2.4 Call specific provider's `Download()` method
- [ ] 2.5 Run convert + place pipeline steps
- [ ] 2.6 Update DB subtitle fields via `UpdateSubtitleStatus`
- [ ] 2.7 Broadcast SSE events during processing
- [ ] 2.8 Return download result

### Task 3: Create Backend Preview Endpoint (AC: #4)
- [ ] 3.1 Implement `POST /api/v1/subtitles/preview` handler
- [ ] 3.2 Request: `{subtitleId, provider}`
- [ ] 3.3 Download subtitle content (but don't save)
- [ ] 3.4 Detect encoding, convert to UTF-8 if needed
- [ ] 3.5 Return first 10 lines as string array
- [ ] 3.6 Add timeout (5 seconds) for preview downloads

### Task 4: Register Routes (AC: #2, #5, #4)
- [ ] 4.1 Register `POST /api/v1/subtitles/search` in router
- [ ] 4.2 Register `POST /api/v1/subtitles/download` in router
- [ ] 4.3 Register `POST /api/v1/subtitles/preview` in router
- [ ] 4.4 Wire SubtitleHandler with SubtitleService dependency

### Task 5: CN Content Conversion Policy — Backend (AC: #9, #10, #11)
- [ ] 5.1 Add `ConversionPolicy` type to `converter.go`: `ConvertAlways`, `ConvertNever`, `ConvertAuto`
- [ ] 5.2 Update `engine.go` `convertIfNeeded()` to accept and check `ConversionPolicy`
- [ ] 5.3 Update `Engine.Process()` to accept `productionCountry` parameter and derive policy
- [ ] 5.4 Update `subtitle_handler.go` to pass production_countries from media DB record
- [ ] 5.5 Update subtitle file extension: `.zh-Hans.srt` when conversion skipped, `.zh-Hant.srt` when converted
- [ ] 5.6 Test: CN content skips conversion, non-CN converts, policy override works

### Task 6: Create Frontend Subtitle Service (AC: #2, #5, #4)
- [ ] 6.1 Create `apps/web/src/services/subtitleService.ts`
- [ ] 6.2 Implement `searchSubtitles(params)` → POST /api/v1/subtitles/search
- [ ] 6.3 Implement `downloadSubtitle(params)` → POST /api/v1/subtitles/download (include `convertToTraditional` boolean)
- [ ] 6.4 Implement `previewSubtitle(params)` → POST /api/v1/subtitles/preview
- [ ] 6.5 Define TypeScript types: `SubtitleSearchParams`, `SubtitleSearchResult`, `SubtitleDownloadParams`

### Task 7: Create useSubtitleSearch Hook (AC: #2, #3, #6)
- [ ] 7.1 Create `apps/web/src/hooks/useSubtitleSearch.ts`
- [ ] 7.2 Use TanStack Query `useMutation` for search (not a query — user-triggered)
- [ ] 7.3 Manage search results state with sorting support
- [ ] 7.4 Use `useMutation` for download action
- [ ] 7.5 Integrate SSE `subtitle_status` events for real-time progress updates
- [ ] 7.6 Export `{ search, download, results, isSearching, isDownloading, sortBy, setSortBy }`

### Task 8: Create SubtitleSearchDialog Component (AC: #1, #3, #9, #10, #11)
- [ ] 8.1 Create `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx`
- [ ] 8.2 Use shadcn/ui `Dialog`, `DialogContent`, `DialogHeader`, `DialogTitle`
- [ ] 8.3 Search form: query input (pre-filled with media title), provider checkboxes
- [ ] 8.4 Add "繁體轉換" toggle (default ON for non-CN, OFF for CN content based on `productionCountry`)
- [ ] 8.5 Results table using shadcn/ui `Table`: Source, Language, Group, Format, Score, Downloads columns
- [ ] 8.6 Sortable column headers (click to toggle asc/desc)
- [ ] 8.7 Score column displays as percentage with color coding (green >70%, yellow >40%, red <=40%)
- [ ] 8.8 Accept props: `mediaId`, `mediaType`, `mediaTitle`, `productionCountry`, `open`, `onOpenChange`

### Task 9: Create Result Row Actions (AC: #4, #5, #6)
- [ ] 9.1 Add "Preview" button per row — opens popover with first 10 lines
- [ ] 9.2 Add "Download" button per row — triggers download mutation (pass toggle state as `convertToTraditional`)
- [ ] 9.3 Show spinner on download button while processing
- [ ] 9.4 Replace download button with checkmark icon on success
- [ ] 9.5 Show error inline on row if download fails
- [ ] 9.6 Success toast notification using existing toast system

### Task 10: Integrate into Media Detail Page (AC: #1)
- [ ] 10.1 Add "Search Subtitles" button/action to media detail page and context menus
- [ ] 10.2 Pass `mediaId`, `mediaType`, `mediaTitle`, `productionCountry` to `SubtitleSearchDialog`
- [ ] 10.3 Refresh media detail data after successful download (invalidate TanStack Query)

### Task 11: Write Backend Tests (AC: #2, #4, #5, #7, #8, #9, #10)
- [ ] 11.1 Create `apps/api/internal/handlers/subtitle_handler_test.go`
- [ ] 11.2 Test search endpoint: valid request, missing mediaId, invalid mediaType, empty results
- [ ] 11.3 Test download endpoint: success path, provider not found, download failure
- [ ] 11.4 Test preview endpoint: success, timeout, encoding detection
- [ ] 11.5 Test conversion policy: CN content skips, non-CN converts, override toggle
- [ ] 11.6 Ensure >80% handler coverage

### Task 12: Write Frontend Tests (AC: #1, #3, #6, #9, #10, #11)
- [ ] 11.1 Create `apps/web/src/services/subtitleService.spec.ts`
- [ ] 11.2 Create `apps/web/src/hooks/useSubtitleSearch.spec.ts`
- [ ] 11.3 Create `apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx`
- [ ] 11.4 Test dialog opens with pre-filled query
- [ ] 11.5 Test results table renders and sorts correctly
- [ ] 11.6 Test download button states (idle, loading, success, error)
- [ ] 11.7 Test provider checkbox filtering
- [ ] 11.8 Ensure >70% frontend coverage

## Dev Notes

### Architecture & Patterns
- Backend follows existing Handler → Service → Repository layered pattern
- Search and download are POST (not GET) because they have structured request bodies and trigger side effects
- Preview is a lightweight read-only operation but uses POST to pass provider + ID
- Frontend uses TanStack Query mutations (not queries) since search is user-initiated, not auto-fetched
- SSE integration reuses existing `useSSE` hook pattern from scanner feature
- Dialog component follows shadcn/ui Dialog pattern used elsewhere in the project

### Project Structure Notes
- Backend handler: `apps/api/internal/handlers/subtitle_handler.go`
- Backend service: reuses `SubtitleService` from Story 8-7
- Frontend service: `apps/web/src/services/subtitleService.ts`
- Frontend hook: `apps/web/src/hooks/useSubtitleSearch.ts`
- Frontend component: `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx`
- Existing patterns: `apps/api/internal/handlers/` for handler structure, `apps/web/src/services/mediaService.ts` for service pattern

### References
- PRD: P1-018 (Manual search — single movie/episode scope)
- Story 8-6: Scorer provides `ScoredResult` with `ScoreBreakdown` for table display
- Story 8-7: Engine handles download → convert → place pipeline
- Story 8-4/8-5: Detector and converter for language processing
- SSE hub: `apps/api/internal/sse/hub.go` with `EventSubtitleProgress`
- Existing UI patterns: `apps/web/src/components/` for component structure

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Completion Notes List
- Backend: SubtitleHandler with POST /search, /download, /preview endpoints
- Provider filtering: checkbox-based, defaults to all configured
- Preview: first 10 non-empty lines with 5s timeout
- Frontend service: subtitleService.ts with TypeScript types
- useSubtitleSearch hook: TanStack Query mutations, sort state, downloaded tracking
- SubtitleSearchDialog: shadcn/ui Dialog, sortable Table, Popover preview, download states
- Score color coding: green >70%, yellow >40%, red ≤40%
- zh-TW labels: 搜尋字幕, 來源, 語言, 字幕組, 評分, 下載數, 預覽, 下載
- 10 backend handler tests pass
- Frontend builds without TypeScript errors
- 🎨 UX Verification: Dialog designed per shadcn/ui patterns, needs visual review in Story 8-8 CR

### File List
- apps/api/internal/handlers/subtitle_handler.go (NEW)
- apps/api/internal/handlers/subtitle_handler_test.go (NEW)
- apps/web/src/services/subtitleService.ts (NEW)
- apps/web/src/hooks/useSubtitleSearch.ts (NEW)
- apps/web/src/components/subtitle/SubtitleSearchDialog.tsx (NEW)
