# Story: Fix setup_service_test.go Key Name Mismatches

Status: done

## Story

As a developer,
I want setup_service_test.go to use the correct snake_case key names matching production API format,
so that all ValidateStep tests pass without panics or assertion mismatches.

## Acceptance Criteria

1. Given `setup_service_test.go`, when all ValidateStep tests run, then zero panics from nil error dereference occur
2. Given test data maps, when keys are used to test validation, then they match the snake_case format that the production API receives (after frontend `camelToSnake` conversion)
3. Given all fixes are applied, when `go test ./internal/services/ -run TestSetupService_ValidateStep -v` runs, then all test cases pass
4. Given `TestSetupService_ValidateStep_EdgeCases`, when all edge case tests run, then TMDb key length validation and media-folder file detection both work correctly
5. Given the complete services test suite, when run, then no regressions are introduced

## Tasks / Subtasks

- [x] Task 1: Fix key name mismatches in TestSetupService_ValidateStep (AC: #1, #2, #3)
  - [x] 1.1 In `apps/api/internal/services/setup_service_test.go`, find all test cases in `TestSetupService_ValidateStep` (line ~538) that use camelCase keys
  - [x] 1.2 Replace `"mediaFolderPath"` → `"media_folder_path"` in: "media-folder - valid path" (line ~589), "media-folder - nonexistent path" (line ~604)
  - [x] 1.3 Replace `"tmdbApiKey"` → `"tmdb_api_key"` in: "api-keys - valid TMDb key" (line ~613), "api-keys - invalid TMDb key (too short)" (line ~627)
  - [x] 1.4 Replace `"qbtUrl"` → `"qbt_url"` in: "qbittorrent - valid URL" (line ~563), "qbittorrent - invalid URL" (line ~578)

- [x] Task 2: Fix key name mismatches in TestSetupService_ValidateStep_EdgeCases (AC: #1, #2, #4)
  - [x] 2.1 In the same file, find all test cases in `TestSetupService_ValidateStep_EdgeCases` (line ~420) that use camelCase keys
  - [x] 2.2 Replace `"mediaFolderPath"` → `"media_folder_path"` in: "media-folder - path is a file not directory" (line ~477)
  - [x] 2.3 Replace `"tmdbApiKey"` → `"tmdb_api_key"` in: "api-keys - TMDb key exactly 16 chars (valid)" (line ~487), "api-keys - TMDb key 15 chars (invalid)" (line ~495)
  - [x] 2.4 Replace `"qbtUrl"` → `"qbt_url"` in: "qbittorrent - URL exactly 7 chars (valid)" (line ~503), "qbittorrent - URL 6 chars (invalid)" (line ~511)

- [x] Task 3: Verify all tests pass (AC: #3, #4, #5)
  - [x] 3.1 Run: `cd apps/api && go test ./internal/services/ -run TestSetupService_ValidateStep -v`
  - [x] 3.2 Verify all test cases pass including edge cases (previously: 4 panics + 3 assertion failures)
  - [x] 3.3 Run full services suite: `cd apps/api && go test ./internal/services/ -v`
  - [x] 3.4 Run full test suite: `cd apps/api && go test ./...`

## Dev Notes

### Root Cause

The frontend `setupService.ts` calls `camelToSnake()` (from `apps/web/src/utils/caseTransform.ts`) which **recursively converts all JSON keys** from camelCase to snake_case before sending to the API. So the Go backend receives snake_case keys like `media_folder_path`, `tmdb_api_key`, `qbt_url`.

The Go implementation correctly uses snake_case:
- `setup_service.go:234` → `data["media_folder_path"]`
- `setup_service.go:249` → `data["tmdb_api_key"]`
- `setup_service.go:194` → `data["qbt_url"]`

But the Go tests incorrectly use camelCase:
- `"mediaFolderPath"`, `"tmdbApiKey"`, `"qbtUrl"`

This causes the validation functions to see empty strings (key not found), which either:
1. **Returns nil** (for optional fields like API keys) → test expects error → `err.Error()` on nil → **PANIC**
2. **Returns wrong error** (for required fields like media folder) → test assertion mismatch → **FAIL**

### Complete Failure Inventory (7 test cases)

**Panics (nil pointer dereference on `err.Error()`):**
- `TestSetupService_ValidateStep / api-keys - invalid TMDb key (too short)` — key `"tmdbApiKey"` not found → nil → panic
- `TestSetupService_ValidateStep / qbittorrent - invalid URL` — key `"qbtUrl"` not found → nil → panic
- `TestSetupService_ValidateStep_EdgeCases / api-keys - TMDb key 15 chars (invalid)` — same
- `TestSetupService_ValidateStep_EdgeCases / qbittorrent - URL 6 chars (invalid)` — same

**Assertion mismatches (error returned but wrong message):**
- `TestSetupService_ValidateStep / media-folder - valid path` — expects no error, gets "media folder path is required"
- `TestSetupService_ValidateStep / media-folder - nonexistent path` — expects "does not exist", gets "media folder path is required"
- `TestSetupService_ValidateStep_EdgeCases / media-folder - path is a file not directory` — expects "not a directory", gets "media folder path is required"

**Coincidentally passing (key mismatch hidden by optional field semantics):**
- `api-keys - valid TMDb key` — key not found → nil → test expects nil → PASS
- `api-keys - skip (empty)` — no key → nil → PASS
- `qbittorrent - valid URL` — key not found → nil (skip allowed) → test expects nil → PASS
- `qbittorrent - skip (empty URL)` — no key → nil → PASS
- `api-keys - TMDb key exactly 16 chars (valid)` — key not found → nil → PASS

### Fix: 3 Search-and-Replace Operations

```
"mediaFolderPath" → "media_folder_path"   (5 occurrences in test data maps)
"tmdbApiKey"      → "tmdb_api_key"        (6 occurrences in test data maps)
"qbtUrl"          → "qbt_url"             (6 occurrences in test data maps)
```

### What NOT to Do

- DO NOT modify `setup_service.go` — the implementation is correct (matches production API format)
- DO NOT add camelCase fallback logic to the service — the frontend handles conversion
- DO NOT change the `camelToSnake` utility — it works correctly
- DO NOT change the handler or request struct — they correctly pass through JSON keys

### References

- [Source: apps/api/internal/services/setup_service_test.go:477-516] — Edge case tests with wrong keys
- [Source: apps/api/internal/services/setup_service_test.go:538-660] — Main ValidateStep tests with wrong keys
- [Source: apps/api/internal/services/setup_service.go:206-258] — Validation functions (correct snake_case keys)
- [Source: apps/web/src/services/setupService.ts:71-76] — Frontend validateStep with camelToSnake
- [Source: apps/web/src/utils/caseTransform.ts:24-35] — Recursive camelToSnake converter

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context) — SM agent (Bob) create-story workflow, YOLO mode

### Debug Log References

### Completion Notes List

- Sprint-status described 2 failures but root cause analysis found 7 failing tests (4 panics + 3 assertion mismatches)
- Additionally 5 tests pass coincidentally due to optional field semantics hiding the key mismatch
- Fix is test-only: 3 search-and-replace operations in 1 file
- Go implementation is correct — matches production snake_case format from frontend camelToSnake conversion
- TA: Upgraded assert.Error→require.Error to prevent nil dereference on test failure (4x); added t.Cleanup for temp file in edge case
- CR-1: Added mockSecrets.AssertExpectations, useNilSecrets field, ErrorIs sentinel check for ErrSetupAlreadyCompleted
- CR-2: Upgraded assert.NoError→require.NoError on success paths (4x); added 3 canonical key failure tests (qbittorrent.host/username/password); added errMsg check to IsFirstRun error test

### Change Log

- 2026-04-09 (DS): Replaced camelCase keys with snake_case in 17 test data map entries across TestSetupService_ValidateStep and TestSetupService_ValidateStep_EdgeCases
- 2026-04-09 (TA): Upgraded assert.Error to require.Error on failure paths; added t.Cleanup for temp file in edge case test
- 2026-04-09 (CR-1): Added mockSecrets.AssertExpectations, useNilSecrets field for nil-secrets dispatch, ErrorIs sentinel check
- 2026-04-09 (CR-2): require.NoError on success paths, 3 canonical key error path tests, IsFirstRun error message verification

### File List

- `apps/api/internal/services/setup_service_test.go` — Replace camelCase keys with snake_case in test data maps
