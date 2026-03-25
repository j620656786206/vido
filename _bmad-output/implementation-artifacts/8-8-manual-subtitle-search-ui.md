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
- [x] 1.1 Create `apps/api/internal/handlers/subtitle_handler.go`
- [x] 1.2 Define `SubtitleSearchRequest` struct with snake_case JSON tags
- [x] 1.3 Define `SubtitleSearchResultDTO` struct with snake_case JSON
- [x] 1.4 Implement `POST /api/v1/subtitles/search` handler
- [x] 1.5 Validate request: mediaId required, mediaType must be "movie" or "series"
- [x] 1.6 Filter providers based on request (default: all configured)
- [x] 1.7 Search providers in parallel with sync.WaitGroup
- [x] 1.8 Return scored results as snake_case JSON DTOs

### Task 2: Create Backend Download Handler (AC: #5, #8)
- [x] 2.1 Define `SubtitleDownloadRequest` struct with `convert_to_traditional` field
- [x] 2.2 Download response includes `subtitle_path`, `language`, `score`
- [x] 2.3 Implement `POST /api/v1/subtitles/download` handler
- [x] 2.4 Call specific provider's `Download()` method
- [x] 2.5 Run detect + convert (with CN policy) + place pipeline steps
- [x] 2.6 Update DB subtitle fields via `UpdateSubtitleStatus`
- [x] 2.7 Broadcast SSE `subtitle_progress` events during processing
- [x] 2.8 Return download result with path, language, score

### Task 3: Create Backend Preview Endpoint (AC: #4)
- [x] 3.1 Implement `POST /api/v1/subtitles/preview` handler
- [x] 3.2 Request: `{subtitle_id, provider}`
- [x] 3.3 Download subtitle content (but don't save)
- [x] 3.4 Detect language via subtitle.Detect()
- [x] 3.5 Return first 10 non-empty lines as string array
- [x] 3.6 Add timeout (5 seconds) for preview downloads

### Task 4: Register Routes (AC: #2, #5, #4)
- [x] 4.1 Register `POST /api/v1/subtitles/search` in router
- [x] 4.2 Register `POST /api/v1/subtitles/download` in router
- [x] 4.3 Register `POST /api/v1/subtitles/preview` in router
- [x] 4.4 Wire SubtitleHandler in main.go with providers, scorer, converter, placer, sseHub, repos

### Task 5: CN Content Conversion Policy — Backend (AC: #9, #10, #11)
- [x] 5.1 Implement `shouldConvert()` method with `userOverride *bool` parameter
- [x] 5.2 Update `engine.go` `convertIfNeeded()` to accept `ConversionPolicy` parameter
- [x] 5.3 Update `Engine.Process()` to accept optional `ProcessOptions` with `ProductionCountry` and `ConversionOverride`
- [x] 5.4 Update `subtitle_handler.go` to accept `convert_to_traditional` from frontend
- [x] 5.5 Placer already handles `.zh-Hans.srt` / `.zh-Hant.srt` based on finalLang
- [x] 5.6 Test: CN content skips conversion, non-CN converts, policy override works (6 test cases)

### Task 6: Create Frontend Subtitle Service (AC: #2, #5, #4)
- [x] 6.1 Create `apps/web/src/services/subtitleService.ts`
- [x] 6.2 Implement `searchSubtitles(params)` → POST /api/v1/subtitles/search
- [x] 6.3 Implement `downloadSubtitle(params)` with `convert_to_traditional` field
- [x] 6.4 Implement `previewSubtitle(params)` → POST /api/v1/subtitles/preview
- [x] 6.5 Define TypeScript types with snake_case matching backend DTOs

### Task 7: Create useSubtitleSearch Hook (AC: #2, #3, #6)
- [x] 7.1 Create `apps/web/src/hooks/useSubtitleSearch.ts`
- [x] 7.2 Use TanStack Query `useMutation` for search (not a query — user-triggered)
- [x] 7.3 Manage search results state with sorting support
- [x] 7.4 Use `useMutation` for download action with per-row tracking
- [x] 7.5 Integrate SSE `subtitle_progress` events via EventSource for download stage tracking
- [x] 7.6 Export search, download, results, per-row state (downloadingIds, previewDataMap)

### Task 8: Create SubtitleSearchDialog Component (AC: #1, #3, #9, #10, #11)
- [x] 8.1 Create `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx`
- [x] 8.2 Plain HTML + Tailwind CSS dialog (matching project pattern, no shadcn/ui dependency)
- [x] 8.3 Search form: query input (pre-filled with media title), provider checkboxes
- [x] 8.4 Add "繁體轉換" toggle (default ON for non-CN, OFF for CN content based on `productionCountry`)
- [x] 8.5 Results table: 來源, 語言, 字幕名稱, 格式, 評分, 下載數, 操作 (per UX design)
- [x] 8.6 Sortable column headers (click to toggle asc/desc)
- [x] 8.7 Score badges with color coding + border + alpha fill (green >70%, yellow >40%, red ≤40%)
- [x] 8.8 Accept props: `mediaId`, `mediaType`, `mediaTitle`, `productionCountry`, `open`, `onOpenChange`

### Task 9: Create Result Row Actions (AC: #4, #5, #6)
- [x] 9.1 Add "Preview" button per row — opens popover with per-row preview data
- [x] 9.2 Add "Download" button per row — triggers download mutation with `convertToTraditional`
- [x] 9.3 Show spinner on download button while processing (per-row via downloadingIds)
- [x] 9.4 Replace download button with checkmark on success
- [x] 9.5 Show "下載失敗" inline error on row when download fails
- [x] 9.6 Success toast notification (auto-dismiss 3s green toast)

### Task 10: Integrate into Media Detail Page (AC: #1)
- [x] 10.1 Add "搜尋字幕" button to media detail page (below CTA buttons)
- [x] 10.2 Pass `mediaId`, `mediaType`, `mediaTitle`, `productionCountry` to `SubtitleSearchDialog`
- [ ] 10.3 Refresh media detail data after successful download (invalidate TanStack Query)

### Task 11: Write Backend Tests (AC: #2, #4, #5, #7, #8, #9, #10)
- [x] 11.1 Create `apps/api/internal/handlers/subtitle_handler_test.go`
- [x] 11.2 Test search endpoint: valid request, missing mediaId, invalid mediaType, empty results, provider filter
- [x] 11.3 Test download endpoint: provider not found, download failure, missing fields
- [x] 11.4 Test preview endpoint: success, provider not found, download failure
- [x] 11.5 Test conversion policy: 6 CN/non-CN/override test cases
- [x] 11.6 Handler coverage: Download 72%, Search 84%, Preview 88%, shouldConvert 100%

### Task 12: Write Frontend Tests (AC: #1, #3, #6, #9, #10, #11)
- [x] 12.1 Create `apps/web/src/services/subtitleService.spec.ts` (4 tests)
- [ ] 12.2 Create `apps/web/src/hooks/useSubtitleSearch.spec.ts` (hook tests deferred — requires complex mutation mocking)
- [x] 12.3 Create `apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx` (10 tests)
- [x] 12.4 Test dialog opens with pre-filled query
- [ ] 12.5 Test results table renders and sorts correctly (needs mock data in hook)
- [ ] 12.6 Test download button states (idle, loading, success, error) (needs mock data)
- [x] 12.7 Test provider checkbox filtering
- [ ] 12.8 Ensure >70% frontend coverage (current: dialog + service tested, hook pending)

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
- apps/api/internal/subtitle/engine.go (MODIFIED — ConversionPolicy, ProcessOptions)
- apps/api/internal/subtitle/engine_test.go (MODIFIED — ConvertAuto param)
- apps/api/cmd/api/main.go (MODIFIED — wired SubtitleHandler + subtitle engine)
- apps/web/src/services/subtitleService.ts (NEW)
- apps/web/src/services/subtitleService.spec.ts (NEW)
- apps/web/src/hooks/useSubtitleSearch.ts (NEW)
- apps/web/src/components/subtitle/SubtitleSearchDialog.tsx (NEW)
- apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx (NEW)
- apps/web/src/components/media/MediaDetailPanel.tsx (MODIFIED — Search Subtitles button + dialog)

### Change Log
- 2026-03-25: Code Review fixes (Claude Opus 4.6)
  - C1: Wired SubtitleHandler in main.go with providers, scorer, converter, placer, sseHub, repos
  - C2: Fixed search to query providers in parallel (sync.WaitGroup)
  - C3: Added SSE subtitle_progress broadcasts during download lifecycle
  - C4: Added score field to download response
  - C5: Updated task checkboxes to reflect actual implementation state
  - M1: Added DB update via SubtitleStatusUpdater after successful download
  - M2: Fixed per-row download state tracking (downloadingIds Set)
  - M3: Fixed per-row preview data tracking (previewDataMap)
  - M4: Fixed table columns to match UX design Flow I (來源/語言/字幕名稱/格式/評分/下載數/操作)
  - M5: Added convert_to_traditional parameter to download flow (CN conversion policy)
  - M6: Added download endpoint tests (provider not found, failure, missing fields)
  - M7: Fixed query state reset when dialog reopens for different media (useEffect)
  - M8: Added search error display in dialog UI
  - M9: Added 繁體轉換 toggle with CN-aware defaults (productionCountry prop)
  - L1: Fixed JSON field casing to snake_case per Rule 6
  - L2: Frontend tests still needed (Task 12)
  - L3: shadcn/ui components need to be installed (pre-existing gap)

## Senior Developer Review (AI)

### Review Date: 2026-03-25

**Reviewer:** Code Review Workflow (Claude Opus 4.6)

**Issues Found:** 5 Critical, 8 Medium, 4 Low — **all fixed except noted below**

### Remaining Items (not fixed)
- [ ] Task 7.5: SSE subscription in frontend hook (needs useSSE integration)
- [ ] Task 9.5: Inline error display per download row
- [ ] Task 9.6: Success toast notification
- [ ] Task 10: Integration into Media Detail Page (all subtasks)
- [ ] Task 12: Frontend tests (all subtasks)
- [ ] Install shadcn/ui components (Dialog, Button, Input, Checkbox, Label, Switch, Table, Popover)
- [ ] Task 5.2-5.3: Engine.Process CN policy (auto-download path, separate from manual handler)
