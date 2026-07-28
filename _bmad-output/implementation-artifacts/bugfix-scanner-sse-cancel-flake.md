# Bugfix: deflake TestScannerService_SSEBroadcast_ScanCancelled

Status: done

**Origin:** `preexisting-fail-scanner-sse-scan-cancelled-flake` (backlog since 2026-05-04, bugfix-10-1 era). **Escalated 2026-07-28** during sub-1-5a's adversarial CR: the test began failing even on **isolated** single-test runs (2 of 3 under normal session load) — no longer only under full-suite CPU contention. Spawned by Alexyu ruling at the sub-1-5a CR follow-up ("現在就開一個 bugfix story 把它修掉") so the 1.5b/1.6 gates stay clean.

---

## Root cause

`scanner_service_test.go` raced a goroutine — `time.Sleep(1 * time.Millisecond)` then `svc.CancelScan()` — against a **real filesystem walk** over 20 temp files. Whenever the machine was loaded, the walk finished inside the 1 ms and the scan broadcast `scan_complete` before the cancel landed: `expected "scan_cancelled", actual "scan_complete"`. The race got easier to lose over time as the suite around it grew heavier.

## Fix (test-only — ScannerService untouched)

Drive the cancel **from inside the walk** instead of racing it: the `FindByFilePath` mock gains a `.Run(...)` callback that calls `svc.CancelScan()` on its first invocation (`sync.Once`). At that point the scan is guaranteed active and mid-walk, so:

1. `CancelScan()` cannot return `ErrScanNotActive` (the reason pre-cancelling was never viable);
2. the next `WalkDir` callback sees the closed `cancelChan` → `filepath.SkipAll`;
3. `StartScan`'s post-walk check broadcasts `scan_cancelled` — deterministically, at any load.

**Deadlock check (the backlog entry's caveat, verified):** the `FindByFilePath` call site (`processVideoFile`, scanner_service.go:470) holds no `s.mu` — the mutex is only taken for short progress-counter updates around it — so `CancelScan()`'s `s.mu.Lock()` inside the callback is safe.

## Accepted trade-off

The old 1 ms sleep *probabilistically* exercised the Story 7b-5 **pre-walk** early-cancel check. The deterministic hook fires mid-walk, so that branch loses its incidental coverage: it cannot be hooked deterministically without a `libraryRepo` in the fixture (the only pre-walk mock touchpoint) or a sync hook in `ScannerService` — both out of proportion for a test-only deflake. The mid-walk path exercises the same `broadcastScanCancelled` surface the assertion cares about.

## Verification

- Isolated: `go test ./internal/services/ -run TestScannerService_SSEBroadcast_ScanCancelled -count=30` green (previously 2/3 failed on a single run).
- Package: `go test ./internal/services/ -count=1` ×3 green.
- Full: `go test ./...` green (34 pkgs).

## File List

| File | Change |
| --- | --- |
| `apps/api/internal/services/scanner_service_test.go` | **modified** — `TestScannerService_SSEBroadcast_ScanCancelled`: sleep-goroutine race replaced with a `sync.Once` `CancelScan()` inside the `FindByFilePath` mock callback; stale 1 ms/7b-5 comment rewritten |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `preexisting-fail-scanner-sse-scan-cancelled-flake` → done (resolution appended); this story's entry added |

## Change Log

| Date | Change |
| --- | --- |
| 2026-07-28 | Deflaked via mock-callback-driven cancellation; burn-in 30× isolated + 3× package + full suite green. ScannerService source untouched. |
