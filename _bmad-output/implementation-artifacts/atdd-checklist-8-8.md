# ATDD Checklist - Epic 8, Story 8-8: Manual Subtitle Search UI

**Date:** 2026-03-25
**Author:** TEA Agent (Claude Opus 4.6)
**Primary Test Level:** API (Go handler tests)

---

## Story Summary

**As a** media library user
**I want** to manually search, preview, and download subtitles from multiple providers
**So that** I can find the best subtitle match for my media with CN-aware conversion

---

## Acceptance Criteria

1. Dialog opens with media title pre-filled, provider checkboxes all checked by default
2. Backend POST /search with parallel providers, scored results
3. Sortable table with columns: 來源, 語言, 字幕名稱, 格式, 評分, 下載數, 操作
4. Preview: first 10 lines in popover with encoding detection
5. Download: POST /download, progress indicator
6. SSE subtitle_status events update dialog in real-time
7. Validation: 400 for invalid mediaId/mediaType, 200 empty for no results
8. Download returns {subtitlePath, language, score}, DB updated
9. CN production_countries → 繁體轉換 OFF
10. Non-CN → 繁體轉換 ON, convert via OpenCC
11. User toggle override

---

## Test Files Created (GREEN Phase)

### API Tests (20 tests)

**File:** `apps/api/internal/handlers/subtitle_handler_test.go`

| Test | AC | Status |
|------|-----|--------|
| TestSubtitleHandler_Search_Success | #2 | GREEN |
| TestSubtitleHandler_Search_MissingMediaID | #7 | GREEN |
| TestSubtitleHandler_Search_InvalidMediaType | #7 | GREEN |
| TestSubtitleHandler_Search_ProviderFilter | #2 | GREEN |
| TestSubtitleHandler_Search_EmptyResults | #7 | GREEN |
| TestSubtitleHandler_Search_ResponseFields | #2 | GREEN |
| TestSubtitleHandler_Search_MultipleProviders | #2 | GREEN |
| TestSubtitleHandler_Download_ProviderNotFound | #5 | GREEN |
| TestSubtitleHandler_Download_DownloadFailure | #5 | GREEN |
| TestSubtitleHandler_Download_MissingFields | #7 | GREEN |
| TestSubtitleHandler_Download_Success | #5,#8 | GREEN |
| TestSubtitleHandler_Download_WithConvertToTraditionalFalse | #9,#11 | GREEN |
| TestSubtitleHandler_Download_ResponseFields | #8 | GREEN |
| TestSubtitleHandler_ShouldConvert (6 subtests) | #9,#10,#11 | GREEN |
| TestSubtitleHandler_Preview_Success | #4 | GREEN |
| TestSubtitleHandler_Preview_ProviderNotFound | #4 | GREEN |
| TestSubtitleHandler_Preview_DownloadFailure | #4 | GREEN |
| TestSubtitleHandler_Preview_ReturnsFirst10Lines | #4 | GREEN |
| TestExtractFirstLines | #4 | GREEN |
| TestExtractFirstLines_Empty | #4 | GREEN |

### Component Tests (20 tests)

**File:** `apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx`

| Test | AC | Status |
|------|-----|--------|
| renders when open | #1 | GREEN |
| does not render when closed | #1 | GREEN |
| pre-fills search input with media title | #1 | GREEN |
| shows all provider checkboxes checked by default | #1 | GREEN |
| toggles provider checkbox on click | #1 | GREEN |
| shows empty state when no results | #3 | GREEN |
| shows 繁體轉換 toggle ON for non-CN | #10 | GREEN |
| shows 繁體轉換 toggle OFF for CN | #9 | GREEN |
| calls onOpenChange when close button clicked | #1 | GREEN |
| calls onOpenChange on backdrop click | #1 | GREEN |
| toggles 繁體轉換 switch on click | #11 | GREEN |
| renders results table with all 7 columns | #3 | GREEN |
| renders result count | #3 | GREEN |
| renders result rows with correct data | #3 | GREEN |
| displays score as percentage with color coding | #3 | GREEN |
| displays format column values | #3 | GREEN |
| shows download buttons per row | #5 | GREEN |
| shows preview buttons per row | #4 | GREEN |
| shows checkmark for downloaded items | #5 | GREEN |
| shows loading spinner for downloading items | #5 | GREEN |

### Service Tests (4 tests)

**File:** `apps/web/src/services/subtitleService.spec.ts`

| Test | AC | Status |
|------|-----|--------|
| searchSubtitles sends POST with correct params | #2 | GREEN |
| searchSubtitles throws on API error | #7 | GREEN |
| downloadSubtitle sends convert_to_traditional | #9,#11 | GREEN |
| previewSubtitle returns lines | #4 | GREEN |

---

## AC ↔ Test Coverage Matrix

| AC | API | Component | Service | Coverage |
|----|-----|-----------|---------|----------|
| #1 | - | 5 tests | - | ✅ COVERED |
| #2 | 4 tests | - | 1 test | ✅ COVERED |
| #3 | - | 6 tests | - | ✅ COVERED |
| #4 | 4 tests | 1 test | 1 test | ✅ COVERED |
| #5 | 3 tests | 3 tests | - | ✅ COVERED |
| #6 | (SSE broadcasts in handler code) | - | - | ⚠️ PARTIAL (no integration test) |
| #7 | 4 tests | - | 1 test | ✅ COVERED |
| #8 | 2 tests | - | - | ✅ COVERED |
| #9 | 1 test | 1 test | 1 test | ✅ COVERED |
| #10 | 1 test | 1 test | - | ✅ COVERED |
| #11 | 6 tests | 1 test | 1 test | ✅ COVERED |

---

## Required data-testid Attributes

### SubtitleSearchDialog
- `subtitle-search-dialog` — Dialog container/backdrop
- `subtitle-search-input` — Search query input
- `subtitle-search-btn` — Search button
- `subtitle-results-table` — Results table
- `subtitle-empty-state` — Empty state message
- `subtitle-row-{id}` — Result row by subtitle ID
- `preview-btn-{id}` — Preview button per row
- `download-btn-{id}` — Download button per row
- `provider-{id}` — Provider checkbox (assrt, opensubtitles, zimuku)
- `convert-toggle` — 繁體轉換 toggle label
- `search-subtitles-button` — Button in MediaDetailPanel

---

## Running Tests

```bash
# Backend: Run all subtitle handler tests
cd apps/api && go test ./internal/handlers/ -run TestSubtitleHandler -v -count=1

# Backend: Run with coverage
cd apps/api && go test ./internal/handlers/ -run TestSubtitle -coverprofile=cover.out

# Frontend: Run all tests
npx nx run web:test -- --run

# Frontend: Run subtitle tests only
npx vitest run apps/web/src/components/subtitle/ apps/web/src/services/subtitleService.spec.ts
```

---

## Notes

- AC #6 (SSE integration) has no end-to-end test — SSE broadcasting is tested at the hub level in `sse/hub_test.go`, and the handler calls `broadcastStatus()` at correct points, but there's no test that verifies the full flow (handler → SSE → frontend EventSource).
- The project does not use Playwright/Cypress for E2E tests. All tests are at API (Go httptest) and Component (vitest + @testing-library/react) levels.
- CN conversion policy (AC #9-11) is thoroughly tested at both backend (shouldConvert 6 scenarios) and frontend (toggle default state + override).

---

**Generated by BMad TEA Agent** - 2026-03-25
