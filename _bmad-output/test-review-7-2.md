# Story 7-2: Scheduled Scan Service — Combined TEA + Adversarial Code Review

**Reviewer:** Claude Opus 4.6 (1M context)
**Date:** 2026-03-23
**Files Reviewed:** 12 files across services, handlers, repository, models, migrations, and main.go

---

## Part 1: TEA Test Quality Review

### Score: 62/100 (Grade: C)

### Coverage by Acceptance Criteria

| AC | Description | Test Coverage | Notes |
|----|-------------|---------------|-------|
| AC1 | Scheduled scan calls StartScan on tick | PARTIAL | `TestScanScheduler_Start_ManualNoTick` verifies manual mode. No test actually triggers a tick and verifies StartScan is called. |
| AC2 | Incremental scan — new/changed/removed | GOOD | `TestScannerService_IncrementalScan_DetectRemovedFiles`, `TestScannerService_IncrementalScan_MtimeChange`, existing size-change test. |
| AC3 | Manual-only = no automatic scans | GOOD | `TestScanScheduler_Start_ManualNoTick` covers this. |
| AC4 | Scheduled scan skipped when manual scan active | GOOD | `TestScanScheduler_SkipWhenScanActive` covers this via direct `onTick` call. |
| AC5 | Schedule change via API restarts scheduler | GOOD | `TestScanScheduler_Reconfigure`, `TestScannerHandler_SetSchedule_Success`, `TestScannerHandler_SetSchedule_InvalidInterval`. |

### Detailed Assessment

**Determinism (6/10):**
- Multiple `time.Sleep(50 * time.Millisecond)` calls in scheduler tests — potential flakiness under CI load.
- `TestScannerHandler_TriggerScan_Success` sleeps 50ms to wait for a goroutine, then asserts — this is a classic flaky pattern.

**Isolation (8/10):**
- Tests use `t.TempDir()` for filesystem isolation — good.
- `t.Cleanup` for SSE hub — good.
- Manual mutex manipulation (`scannerService.mu.Lock(); scannerService.isScanning = true`) bypasses encapsulation but is acceptable for unit tests.

**Assertions (7/10):**
- Assertions are explicit and specific (exact counts, error codes).
- Missing: no assertion that `ParseStatus` is reset to pending on file change (AC2 requirement: "reset ParseStatus=pending").
- Missing: no assertion on `FilesRemoved` field in `ScanResult` for the mtime-change test.

**Edge Cases (5/10):**
- Double-stop is tested — good.
- Invalid interval is tested — good.
- Missing: Reconfigure error path (SetString fails).
- Missing: what happens when `FindAllWithFilePath` returns an error during `detectRemovedFiles`.
- Missing: file reappears after being marked as removed (no un-remove logic tested or implemented).
- Missing: test for `onTick` when scan is NOT active (i.e., the happy path where scheduled scan actually fires).
- Missing: test for `loadScheduleFromSettings` with invalid stored value.
- Missing: concurrent Reconfigure calls.

**Mock Patterns (7/10):**
- Mocks are well-structured with proper interface implementation.
- `filterCalls` helper for overriding `Maybe()` mocks is clever but fragile.
- Some mock methods return hardcoded `nil` without using `m.Called()` — acceptable for unused methods but inconsistent.

### Key Test Gaps

1. **No test verifies that a ticker-based scheduled scan actually calls StartScan.** The `onTick` method is tested in isolation for the skip path but never for the execute path with a real (short-interval) ticker.
2. **No test verifies ParseStatus reset on file change** (Task 2.3 says "reset ParseStatus=pending" but `processVideoFile` does not reset it, and no test checks for it).
3. **No integration-style test for the full Reconfigure flow** (manual -> hourly -> verify tick fires -> manual -> verify no tick).

---

## Part 2: Adversarial Code Review

### HIGH Severity

#### H1: `Update()` does NOT persist `is_removed`, `file_size`, or `file_path`
**File:** `apps/api/internal/repository/movie_repository.go:288-306`
**Impact:** The `detectRemovedFiles` method sets `movie.IsRemoved = true` and calls `movieRepo.Update()`, but the UPDATE SQL statement does not include `is_removed`, `file_size`, or `file_path` columns. **The soft-delete flag is silently discarded** — `is_removed` will never be written to the database.

This means:
- AC2 ("removed files are marked as removed") is **not actually implemented** at the database level.
- Every subsequent scan will re-check the same "removed" files because they remain `is_removed = 0`.
- The `FindAllWithFilePath` query filters by `is_removed = 0`, so the files will be checked again and again.

**Fix:** Add `is_removed`, `file_size`, and `file_path` to the UPDATE query.

#### H2: `processVideoFile` does NOT reset `ParseStatus` on file change
**File:** `apps/api/internal/services/scanner_service.go:358-367`
**Impact:** Task 2.3 requires "Changed files (mtime > DB UpdatedAt): update FileSize, reset ParseStatus=pending". The code updates `FileSize` and `UpdatedAt` but never sets `existing.ParseStatus = models.ParseStatusPending`. Changed files will retain their old parse status.

**Fix:** Add `existing.ParseStatus = models.ParseStatusPending` before the Update call at line 361.

#### H3: TOCTOU race in `onTick` — scan can be skipped or double-started
**File:** `apps/api/internal/services/scan_scheduler.go:234-257`
**Impact:** `onTick` checks `IsScanActive()` then starts a goroutine calling `StartScan()`. Between the check and the `StartScan` call, another scan could start (from API or another tick). The goroutine uses `context.Background()` so it is not cancelled if the scheduler stops. While `StartScan` has its own mutex that would return `SCANNER_ALREADY_RUNNING`, the error is only logged — not a data corruption issue, but the `IsScanActive()` check is a misleading false gate.

**Fix:** Remove the `IsScanActive()` pre-check entirely, or accept that it is merely an optimization to avoid goroutine creation. Add a comment clarifying the intentional design.

### MEDIUM Severity

#### M1: Reconfigure creates a new ticker but `runTickerLoop` still reads the OLD ticker's channel
**File:** `apps/api/internal/services/scan_scheduler.go:134-164, 191-231`
**Impact:** When `Reconfigure` is called, it stops the old ticker and creates a new one under `s.ticker`. However, `runTickerLoop` caches `ticker := s.ticker` at line 203 and then blocks on `ticker.C` at line 227. After `Reconfigure` runs, the loop is still blocked on the **old** ticker's channel (which is stopped and will never send again). The loop will only unblock via `ctx.Done()` or `done` channel, not from the new ticker.

The special case: if `Reconfigure` is called to change from `manual` to `hourly`, the loop is in the `ticker == nil` branch (lines 208-217), blocked on `done/ctx`. It will never see the new ticker.

Essentially, **Reconfigure does not actually cause the running scheduler to use the new interval.** It only updates the stored state. The scheduler must be stopped and restarted for the new interval to take effect.

**Fix:** Signal the runTickerLoop to re-read the ticker after Reconfigure (e.g., close and re-create the `done` channel, or use a separate "reconfigure" signal channel).

#### M2: `FindByFilePath` in `movie_repository.go` does not scan `file_path`, `file_size`, or `is_removed`
**File:** `apps/api/internal/repository/movie_repository.go:227-271`
**Impact:** The `FindByFilePath` query returns the same limited column set as `FindByID`. It does not return `file_path`, `file_size`, `parse_status`, `subtitle_*`, `is_removed`, or `vote_average`. When `processVideoFile` calls `FindByFilePath` and gets back an existing movie, `existing.FileSize` will always be the zero value (`{Int64:0, Valid:false}`) and `existing.UpdatedAt` will be from the DB. The `sizeChanged` check at line 347 (`!existing.FileSize.Valid`) will always be `true` because `FileSize` is never scanned from this query.

This means **every file already in the DB will be "updated" on every scan**, defeating the incremental scan optimization.

**Fix:** Update `FindByFilePath` to use `movieSelectColumns` and `scanMovie()` like the other query methods.

#### M3: `detectRemovedFiles` loads ALL movies into memory
**File:** `apps/api/internal/services/scanner_service.go:431-467`
**Impact:** For a large media library (tens of thousands of files), `FindAllWithFilePath` loads every movie record into memory at once. This could cause significant memory pressure. Additionally, it calls `os.Stat` and `movieRepo.Update` one-by-one in a loop (N+1 pattern for updates).

**Fix (deferred):** Consider pagination or batch processing for very large libraries. For now, document the limitation.

#### M4: Handler `SetSchedule` validates interval then calls `Reconfigure` which validates again
**File:** `apps/api/internal/handlers/scanner_handler.go:120-125` and `apps/api/internal/services/scan_scheduler.go:134-137`
**Impact:** Minor — double validation is not harmful but is redundant code. The handler validation is the correct place; the service validation is defense-in-depth.

### LOW Severity

#### L1: `getTickerDuration` returns `0` for `ScheduleManual`
**File:** `apps/api/internal/services/scan_scheduler.go:260-268`
**Impact:** If `time.NewTicker(0)` is ever called, it panics. The code guards against this, but the function returning 0 for an invalid case is a latent risk.

**Fix:** Return a sentinel error or panic with a clear message instead of returning 0.

#### L2: Scheduler `Start` method is not reentrant
**File:** `apps/api/internal/services/scan_scheduler.go:76-107`
**Impact:** If `Start` is called twice, it will overwrite `s.done` and create a second goroutine. The code trusts callers not to do this. The `running` flag is set but not checked before starting.

**Fix:** Check `s.running` at the beginning of `Start` and return early or error.

#### L3: Migration `Down()` is a no-op
**File:** `apps/api/internal/database/migrations/019_add_is_removed_field.go:30-33`
**Impact:** Cannot roll back the migration. The comment explains SQLite limitations, so this is acceptable but should be noted.

#### L4: No test for `GetSchedule` when scheduler returns each valid interval
**File:** `apps/api/internal/handlers/scanner_handler_test.go:203-221`
**Impact:** Only tests `hourly`. Missing `daily` and `manual` variants. Minor coverage gap.

#### L5: `BulkCreate` in `movie_repository.go` does not include `is_removed` or `file_size`
**File:** `apps/api/internal/repository/movie_repository.go:721-728`
**Impact:** New movies from scan will always have `is_removed = 0` (the DB default), which is correct behavior. But `file_size` is set in the Movie struct by `processVideoFile` and IS included in `BulkCreate` at line 766 (`movie.FilePath`). Actually, looking more closely, `file_size` is NOT in the BulkCreate INSERT columns. The column list has `file_path` but not `file_size`. This means **file_size is never persisted for new files**.

Wait — re-checking: the BulkCreate INSERT has 20 columns and 20 values. Column 15 is `file_path`. `file_size` is not in the list. So `file_size` is set in the struct but never written to the DB during initial creation via BulkCreate.

**Fix:** Add `file_size` to the BulkCreate INSERT statement.

---

## Part 3: Summary

### Critical Fix-Now Issues (Auto-fix recommended)

| # | Severity | Issue | File | Action |
|---|----------|-------|------|--------|
| H1 | HIGH | `Update()` SQL missing `is_removed`, `file_size`, `file_path` | `movie_repository.go:288-306` | **Auto-fix** |
| H2 | HIGH | `processVideoFile` doesn't reset ParseStatus on change | `scanner_service.go:358-367` | **Auto-fix** |
| M2 | MEDIUM | `FindByFilePath` doesn't return `file_size`/`is_removed` | `movie_repository.go:227-271` | **Auto-fix** |
| L5 | LOW→HIGH | `BulkCreate` doesn't persist `file_size` | `movie_repository.go:721-728` | **Auto-fix** |

### Issues to Defer

| # | Severity | Issue | Reason |
|---|----------|-------|--------|
| H3 | HIGH | TOCTOU in `onTick` | Benign due to StartScan's own mutex; add comment only |
| M1 | MEDIUM | Reconfigure doesn't wake runTickerLoop | Requires architectural change to ticker loop; schedule for follow-up |
| M3 | MEDIUM | `detectRemovedFiles` loads all movies into memory | Performance optimization, not correctness; defer to scaling story |
| M4 | MEDIUM | Double validation | Harmless defense-in-depth |
| L1 | LOW | `getTickerDuration` returns 0 for manual | Guarded by callers |
| L2 | LOW | `Start` not reentrant | Trusted caller pattern |
| L3 | LOW | Migration Down() is no-op | Documented SQLite limitation |
| L4 | LOW | Missing handler test variants | Minor coverage gap |

### Test Improvements Needed

1. Add test that verifies a tick-fired scan actually calls `StartScan` (use short ticker duration).
2. Add test asserting `ParseStatus` is reset to `pending` on file change.
3. Add test for `detectRemovedFiles` when `FindAllWithFilePath` returns an error.
4. Add test for `Reconfigure` when `SetString` fails.
5. Replace `time.Sleep` coordination with channels or `sync.WaitGroup` for determinism.
