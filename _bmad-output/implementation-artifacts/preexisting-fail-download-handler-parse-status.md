# Story: Fix download_handler_test.go PaginatedResponse Mismatch

Status: ready-for-dev

## Story

As a developer,
I want the 4 WithParseStatus download handler tests to correctly unwrap the PaginatedResponse envelope,
so that parse_status enrichment logic is properly tested after the pagination API change.

## Acceptance Criteria

1. Given `download_handler_test.go`, when all 4 WithParseStatus tests run, then zero type assertion failures on `response.Data.([]interface{})` occur
2. Given the ListDownloads handler returns a PaginatedResponse, when tests parse the response, then they correctly access items via `response.Data["items"]` instead of treating `response.Data` as a direct array
3. Given all fixes are applied, when `go test ./internal/handlers/ -run TestDownloadHandler_ListDownloads -v` runs, then all 4 tests pass
4. Given the pagination metadata, when tests validate the response, then they also verify `page`, `total_items`, `total_pages` fields are present and correct
5. Given the full handler test suite, when run, then no regressions are introduced

## Tasks / Subtasks

- [ ] Task 1: Fix response parsing in all 4 WithParseStatus tests (AC: #1, #2, #3)
  - [ ] 1.1 In `apps/api/internal/handlers/download_handler_test.go`, update `TestDownloadHandler_ListDownloads_WithParseStatus` (line ~688): replace `response.Data.([]interface{})` with `response.Data.(map[string]interface{})["items"].([]interface{})`
  - [ ] 1.2 Update `TestDownloadHandler_ListDownloads_WithParseStatus_NoJob` (line ~757): same response parsing fix
  - [ ] 1.3 Update `TestDownloadHandler_ListDownloads_WithParseStatus_Failed` (line ~799): same response parsing fix
  - [ ] 1.4 Update `TestDownloadHandler_ListDownloads_WithoutParseQueueService` (line ~849): same response parsing fix

- [ ] Task 2: Add pagination metadata assertions (AC: #4)
  - [ ] 2.1 In each of the 4 tests, after extracting the `map[string]interface{}`, assert `page` == 1, `total_items` matches expected count, `total_pages` == 1

- [ ] Task 3: Verify all tests pass (AC: #3, #5)
  - [ ] 3.1 Run: `cd apps/api && go test ./internal/handlers/ -run TestDownloadHandler_ListDownloads -v`
  - [ ] 3.2 Verify all 4 WithParseStatus tests pass
  - [ ] 3.3 Run full handler test suite: `cd apps/api && go test ./internal/handlers/ -v`
  - [ ] 3.4 Run full test suite: `cd apps/api && go test ./...`

## Dev Notes

### Root Cause

Commit `3e133da` (feat: add download page design and backend pagination API) changed `ListDownloads` handler to wrap results in `PaginatedResponse`:

**Before:**
```go
SuccessResponse(c, items)  // response.Data = []DownloadItem
```

**After:**
```go
SuccessResponse(c, PaginatedResponse{
    Items:      allItems[start:end],
    Page:       page,
    PageSize:   pageSize,
    TotalItems: total,
    TotalPages: totalPages,
})
// response.Data = { "items": [...], "page": 1, "page_size": 100, ... }
```

The 4 WithParseStatus tests were written against the pre-pagination format and never updated.

### PaginatedResponse Struct

Defined at `apps/api/internal/handlers/response.go:25-32`:
```go
type PaginatedResponse struct {
    Items      interface{} `json:"items"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalItems int         `json:"total_items"`
    TotalPages int         `json:"total_pages"`
}
```

### Response Parsing Pattern

All 4 tests have this identical failing pattern at their assertion blocks:

```go
// CURRENT (fails — response.Data is map, not slice):
dataSlice, ok := response.Data.([]interface{})
require.True(t, ok)
```

Replace with:

```go
// FIXED — unwrap PaginatedResponse envelope:
dataMap, ok := response.Data.(map[string]interface{})
require.True(t, ok, "response.Data should be a PaginatedResponse map")
dataSlice, ok := dataMap["items"].([]interface{})
require.True(t, ok, "items should be a slice")
```

### Existing Pattern — Check Other ListDownloads Tests

Other tests in the same file (e.g., `TestDownloadHandler_ListDownloads`, `TestDownloadHandler_ListDownloads_Pagination`) were already updated to use the paginated response format. Use them as reference for the correct parsing pattern.

### What the 4 Tests Validate (After Fix)

1. **WithParseStatus** — Completed download gets `parse_status` enrichment (status + media_id), in-progress download does not
2. **WithParseStatus_NoJob** — Completed download with no parse job gets `parse_status: nil`
3. **WithParseStatus_Failed** — Failed parse job includes `error_message` field
4. **WithoutParseQueueService** — Handler works without ParseQueueService injected (graceful degradation)

### What NOT to Do

- DO NOT modify `download_handler.go` — the pagination response is correct and used by the frontend
- DO NOT revert to non-paginated response format
- DO NOT change `PaginatedResponse` struct
- DO NOT modify the parse status enrichment logic — it's correct, only the test assertions need updating

### References

- [Source: apps/api/internal/handlers/download_handler_test.go:688-884] — 4 failing test functions
- [Source: apps/api/internal/handlers/download_handler.go:118-136] — PaginatedResponse wrapping in ListDownloads
- [Source: apps/api/internal/handlers/response.go:25-32] — PaginatedResponse struct definition
- [Source: git commit 3e133da] — Pagination change that broke the tests

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context) — SM agent (Bob) create-story workflow, YOLO mode

### Debug Log References

### Completion Notes List

- All 4 tests fail at the same line pattern: `response.Data.([]interface{})` type assertion
- Fix is mechanical: unwrap PaginatedResponse map then access `["items"]`
- Other ListDownloads tests in the same file already use the correct pattern — copy it
- 1 file, 4 identical fixes + optional pagination metadata assertions

### File List

- `apps/api/internal/handlers/download_handler_test.go` — Fix response.Data parsing in 4 WithParseStatus tests
