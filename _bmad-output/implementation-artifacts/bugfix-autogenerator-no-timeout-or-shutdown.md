# Bugfix: `AutoGenerator` round has no shutdown hook and no per-item deadline

Status: done

**Origin:** Rule 24 ③ from 9R-10b CR-249 **L1** (2026-08-20), mirrored in the 補審 Medium table as **M8**. `subtitle.AutoGenerator.ScanCallback` spawns `go g.Run(context.Background())` — no cancellation, no deadline, no place in `main.go`'s graceful-shutdown block. The CR left it unfixed because it needs **three lifecycle decisions** (join graceful shutdown or not; how long a round may run; what happens to the item that was mid-flight when the plug is pulled). This story makes those decisions and lands them. Backend only.

**Sibling entries this story deliberately does NOT absorb** (see "Interaction with sibling backlog items" below): `bugfix-autogenerator-dropped-round-not-deferred` (補審 M6), `bugfix-auto-exclusion-never-expires` (補審 M2).

---

## Story

As the Vido NAS operator,
I want the free-lane auto-generation round to be stopped cleanly when the API shuts down and to never be pinned by one stuck item,
so that a restart (upgrade, Unraid reboot, `docker restart`) does not strand media rows in an in-flight subtitle status, does not write to a closed database, and does not quietly park an innocent item out of the free lane.

## Root cause — what actually goes wrong today (verified against code)

Read these before touching anything; two of the three "obvious" harms are NOT the real ones.

1. **"A stuck ffmpeg pins the goroutine forever" is only half true.** The two subprocesses already carry their own deadlines: `Extractor` wraps ffmpeg in `context.WithTimeout(ctx, e.timeout)` with `defaultExtractTimeout = 10 * time.Minute` (`apps/api/internal/subtitle/extractor.go:19-22, 189-195`) and `FFprobeService` wraps ffprobe in a 10 s timeout (`apps/api/internal/services/ffprobe_service.go:65-66, 108-113`). So an ffmpeg that genuinely hangs is killed after 10 min and the item fails via `failItem`. What is **unbounded** is everything in `ProcessItem` that is *not* a subprocess — `media.Load`, `feedGlossary`, the run-store writes, OpenCC conversion, `Placer` — none of which check `ctx` on the `context.Background()` the round currently runs under. A locked SQLite file on the NAS (the 2026-08-01 FUSE incident class) is the realistic way one of those blocks.

2. **The real shutdown defect is a write-after-close, not a leak.** `main.go`'s shutdown block (`apps/api/cmd/api/main.go:1060-1119`) stops every other background component, then calls `db.Close()` at `:1113`. The auto-generation goroutine is not in that list, so it keeps running straight through `db.Close()`. Two outcomes, both bad:
   - `ProcessItem` is mid-item: the next repo write fails with `sql: database is closed`; `failItem` then tries its cleanup writes (`run.Update`, `setMediaStatus(... not_searched)`) on the same closed DB and they fail too (`process_item.go:661-690` — it logs and continues). The media row is left at `extracting` (`probing` is SSE-only — `process_item.go:118-121` writes the media row straight to `extracting`), which `autoEligibleStatuses` (`auto_generation.go:54-58`) **excludes** — so after the restart that item is invisible to the free lane until someone processes it manually. That is a stranded item, the exact thing `failItem`'s comment says it exists to prevent.
   - `Run` is between items: `collect`/`ListByStatus` errors → "round aborted" logged against a closing process. Harmless but noisy.
   Note the process only lives ~5 s after `db.Close()` (`shutdownCtx` at `:1109`), so "goroutine leak" is not the harm — the **five seconds of writes against a closed handle** are.

3. **A cancelled item counts as a failed attempt.** `failItem` writes a `failed` run row regardless of cause (`process_item.go:661-675`; cancellation is explicitly kept — `TestProcessItem_CancellationStillRecordsTheFailure`, `process_item_test.go:786`). `excludedMediaIDs` then **counts** `failed` rows per media id and parks the item at `autoFailureAttemptLimit = 3` (`auto_generation.go:25-45, 392-406`). Once this story adds cancellation, three restarts that each land on the same long, alphabetically-early item would park it permanently — compounded by `bugfix-auto-exclusion-never-expires`. The fix must therefore make shutdown-cancellation **not** count, while a genuine per-item timeout **does** (a file that takes >15 min is exactly what the limit is for).

## Lifecycle decisions (the part the CR could not make)

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Join graceful shutdown? | **Yes** — `AutoGenerator.Stop()` called from `main.go` right after `subtitlePipelinePool.Stop()`, **before** `db.Close()` | Mirror of the `WorkerPool` contract (`worker_pool.go:197-219`): cancel the in-flight work, then block until the goroutine has returned, so `failItem`'s cleanup writes land while the DB is still open. Rule 14 (Graceful Shutdown) says background goroutines MUST honour cancellation. |
| D2 | Round timeout? | **No round-level timeout. Per-ITEM deadline instead:** `AutoGenerationItemTimeout = 15 * time.Minute`, a package constant like `AutoGenerationMaxPerRun` | A round is already bounded by `maxPerRun × item deadline`. A round-level deadline would kill item 18 because items 1–17 were slow — punishing the wrong file, and writing a `failed` row on an item that did nothing wrong. 15 min = `defaultExtractTimeout` (10 min) + ffprobe (10 s) + generous slack for DB/OpenCC/placement, so the subprocess deadlines still fire first on the common path and this one only catches the non-subprocess hang in §1. |
| D3 | The mid-flight item on shutdown | **Fail it, but mark the failure as cancelled so it is NOT counted toward `autoFailureAttemptLimit`.** Marker: new `CancelledRunPrefix = "cancelled: "` written by `failItem` when `errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)` — the `ctx.Err()` half is REQUIRED: ffprobe killed by cancellation surfaces as an `*exec.ExitError` (`signal: killed`), not `context.Canceled` (`ffprobe_service.go:123-127` only special-cases `DeadlineExceeded`), so the cause alone misses the probe step; `excludedMediaIDs` skips rows with that prefix when counting failures. A `DeadlineExceeded` (per-item timeout) is NOT marked and keeps counting. | `failItem`'s existing contract (cleanup survives cancellation via `context.WithoutCancel`) is right and stays. What is wrong is only the bookkeeping downstream. The prefix pattern is the one `DeferredPaidRunPrefix` already established (`process_item.go:549`) — a reason carried in `error_message`, not a new enum member (CR-249 "看過、決定保留" #3). |
| D4 | Rounds after `Stop()` | **`ScanCallback` becomes a no-op after `Stop()`** (logged at Debug) | A scan completing during the ~5 s shutdown window must not spawn a fresh goroutine behind `Stop()`'s back. |
| D5 | Where the cancellable context comes from | **`AutoGenerator` owns it**: `context.WithCancel(context.Background())` created in `NewAutoGenerator`, cancelled by `Stop()` | `autoGenerator` is constructed at `main.go:678` inside the pipeline-mode block, while `subtitlePipelineCtx` is created at `:1035` — reusing it would force a main.go reorder for no gain. Owning the ctx keeps the wiring a single `Stop()` call, like `scanScheduler`, `backupScheduler` and the others in the shutdown block. |

## Acceptance Criteria

1. **AC #1 — Stop cancels and drains.** Given a round is in flight (a `ProcessItem` call is blocking), when `AutoGenerator.Stop()` is called, then the blocking `ProcessItem` observes `ctx.Done()` with `context.Canceled`, and `Stop()` returns only **after** `Run` has returned (WaitGroup drain, the `WorkerPool.Stop` shape). `Stop()` is idempotent (second call returns immediately, no panic, no double-close).

2. **AC #2 — A cancelled round stops iterating.** Given a round with N eligible items and cancellation landing during item k, then items k+1…N are **not** passed to `ProcessItem` (check `ctx.Err()` at the top of the loop in `Run`), and the finish log line reports the round as cancelled (`"subtitle auto-generation cancelled"`, with the same counters as the finished line plus `"remaining"`).

3. **AC #3 — No new rounds after Stop.** Given `Stop()` has been called, when the scan-complete callback fires, then no goroutine is spawned and `ProcessItem` is never called; a Debug line is logged.

4. **AC #4 — Per-item deadline.** Given `AutoGenerationItemTimeout` (default 15 min; `WithAutoItemTimeout(d)` option for tests, mirroring `WithAutoMaxPerRun`), when one item's `ProcessItem` runs longer than that, then its ctx reaches `DeadlineExceeded`, the item fails via the pipeline's normal `failItem` path (a `failed` run row WITHOUT the cancelled marker), and the round **continues** to the next item with a fresh deadline. Each item gets its own `context.WithTimeout(roundCtx, itemTimeout)` and its cancel func is released when the item returns (no leaked timers across 20 items).

5. **AC #5 — Cancelled failures do not count toward parking.** Given `failItem` is reached while `errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)` is true (the item ctx is `Canceled` on shutdown and `DeadlineExceeded` on the per-item timeout, so `ctx.Err()` separates the two even when the cause is an opaque `*exec.ExitError` from a killed ffprobe), then `run.ErrorMessage` starts with `CancelledRunPrefix` (`"cancelled: "`). Given `excludedMediaIDs` sees `failed` rows, then rows whose `ErrorMessage` has that prefix are **not** counted toward `autoFailureAttemptLimit`; a `DeadlineExceeded` failure is unmarked and still counts. The existing `TestProcessItem_CancellationStillRecordsTheFailure` invariant (run row written, media reverted to `not_searched`) is unchanged.

6. **AC #6 — main.go wiring.** `autoGenerator` is hoisted to a `var autoGenerator *subtitle.AutoGenerator` beside `subtitlePipelinePool` (nil in legacy mode), and the shutdown block calls `autoGenerator.Stop()` **immediately after** the `subtitlePipelinePool.Stop()` block and **before** `db.Close()`, guarded by the nil check and with a `slog.Info("Stopping subtitle auto-generation...")` line matching its neighbours. The existing `ComposeScanCallback(postScanEnrichment, autoGenerator.ScanCallback())` registration is byte-unchanged apart from the variable now being the hoisted one.

7. **AC #7 — Zero-paid guard untouched.** `internal/cost_consent_test.go` git diff is **0 lines**; all four tests in `cost_consent_free_lane_test.go` (`TestFreeLane_NeverReachesPaidPorts`, `TestFreeLane_FreeRoutesStillComplete`, `TestFreeLane_DeferredItemStaysOnTheConsentList`, `TestDeferredMarker_WriterAndReaderAgree`) stay green. Every `ProcessItem` call still carries `ProcessItemOptions{FreeOnly: true}` (`TestAutoGenerator_AlwaysProcessesFreeOnly` stays green).

## Tasks / Subtasks

- [x] **Task 1 — `AutoGenerator` lifecycle** (AC #1, #2, #3, #4) — `apps/api/internal/subtitle/auto_generation.go`
  - [x] 1.1 Add fields: `lifetime context.Context`, `cancel context.CancelFunc`, `wg sync.WaitGroup`, `stopped bool` (guarded by the existing `mu`), `itemTimeout time.Duration`. Create `lifetime, cancel` in `NewAutoGenerator` (D5). Default `itemTimeout = AutoGenerationItemTimeout`.
  - [x] 1.2 Add `const AutoGenerationItemTimeout = 15 * time.Minute` (the file does not import `time` yet — add it) with a doc comment stating the derivation in D2 (extract 10 min + probe 10 s + slack; only the non-subprocess hang is new coverage). Add `WithAutoItemTimeout(d time.Duration)` option (`d > 0` only, like `WithAutoMaxPerRun`).
  - [x] 1.3 Rewrite `ScanCallback()`: under `mu`, if `stopped` → Debug log and return; else `wg.Add(1)` **before** the `go`, and the goroutine does `defer g.wg.Done(); g.Run(g.lifetime)`. (The `wg.Add` must be under the same lock as the `stopped` check, or `Stop()` can race between the check and the Add.)
  - [x] 1.4 Add `Stop()`: under `mu` set `stopped = true` (idempotent — return early if already set **after** still calling `wg.Wait()`; see `WorkerPool.Stop`'s CR M2 note on why the wait sits outside the flag check), `cancel()`, unlock, `wg.Wait()`, Info log `"subtitle auto-generation stopped"` once.
  - [x] 1.5 In `Run`, right after the single-flight guard and BEFORE the policy read, `if ctx.Err() != nil { Debug log; return }` — otherwise a callback `wg.Add`-ed just before `Stop()` runs the policy query on a cancelled ctx and emits the Error-level "cannot read library policy — round aborted" line on every shutdown. Then in the item loop: at the top of each iteration `if ctx.Err() != nil { break }` and track `remaining`; wrap each `ProcessItem` call in `itemCtx, cancelItem := context.WithTimeout(ctx, g.itemTimeout)` and call `cancelItem()` right after it returns (not deferred inside the loop). After the loop, log `"subtitle auto-generation cancelled"` (Warn) with `remaining` when `ctx.Err() != nil`, else the existing `"finished"` Info line. Keep `Run` synchronous and keep accepting an external `ctx` — tests call it directly.
  - [x] 1.6 Keep the single-flight guard exactly as is; `Stop()` must not touch `running` (the deferred unlock in `Run` owns it).

- [x] **Task 2 — Cancelled-failure marker** (AC #5) — `apps/api/internal/subtitle/process_item.go`, `auto_generation.go`
  - [x] 2.1 Add `const CancelledRunPrefix = "cancelled: "` next to `DeferredPaidRunPrefix` (`process_item.go:549`) with a comment naming this story and the `autoFailureAttemptLimit` interaction.
  - [x] 2.2 In `failItem`, when `errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)` (NOT `DeadlineExceeded` — check `ctx.Err()` on the ctx passed in, which is the item ctx), prefix `run.ErrorMessage` with `CancelledRunPrefix` before `truncateErrorMessage`. Everything else in `failItem` stays byte-identical (the `context.WithoutCancel` cleanup is the part that makes D1 work).
  - [x] 2.3 In `excludedMediaIDs`'s `failed` loop, `continue` on `strings.HasPrefix(r.ErrorMessage, CancelledRunPrefix)` before incrementing `attempts`. Update the function comment's "Failures are COUNTED" paragraph to state the exemption and why.

- [x] **Task 3 — main.go wiring** (AC #6) — `apps/api/cmd/api/main.go`
  - [x] 3.1 Hoist: add `autoGenerator *subtitle.AutoGenerator` to the `var (...)` block at `:591-594`; change `autoGenerator := subtitle.NewAutoGenerator(` at `:678` to an assignment.
  - [x] 3.2 Shutdown block: after the `subtitlePipelinePool.Stop()` `if` at `:1089-1092` add the nil-guarded `autoGenerator.Stop()` with its Info line. Order matters: must precede `db.Close()` at `:1113`. Add a one-line comment citing this story and AC #6.
  - [x] 3.3 Extend the `"Subtitle generation pipeline enabled"` log at `:692` with `"scan_auto_item_timeout", subtitle.AutoGenerationItemTimeout` (same style as the existing `"scan_auto_free_generation", true` key).

- [x] **Task 4 — Tests** (all ACs) — `apps/api/internal/subtitle/auto_generation_test.go`, `process_item_test.go`
  - [x] 4.1 `TestAutoGenerator_StopCancelsTheInFlightRoundAndDrains` — fake processor blocks on `<-ctx.Done()` then returns `ctx.Err()`; spawn via `ScanCallback()()`; call `Stop()`; assert `Stop` returned only after the processor recorded the call AND the observed error `errors.Is(..., context.Canceled)`. Use channels, not `time.Sleep` (the `preexisting-fail-*-flake` lesson: sleep-based races are banned).
  - [x] 4.2 `TestAutoGenerator_StopIsIdempotent` — two `Stop()` calls, no panic, and `Stop()` on a generator that never ran returns immediately.
  - [x] 4.3 `TestAutoGenerator_CancelledRoundStopsIterating` (AC #2) — three items; cancel the ctx from inside item 1's fake; assert `refIDs() == ["m1"]`. Call `Run(ctx)` directly with a cancellable ctx (no goroutine needed).
  - [x] 4.4 `TestAutoGenerator_ScanCallbackAfterStopIsANoOp` (AC #3) — `Stop()` first, then `ScanCallback()()`, then assert zero `ProcessItem` calls. Because the no-op path spawns nothing there is no goroutine to synchronise on; assert synchronously.
  - [x] 4.5 `TestAutoGenerator_EachItemGetsItsOwnDeadline` (AC #4) — `WithAutoItemTimeout(50 * time.Millisecond)`; fake processor for `m1` blocks on `<-ctx.Done()` and returns `ctx.Err()`; `m2` returns immediately and records whether its ctx had a deadline in the future (`ctx.Deadline()` ok && after the m1 failure). Assert `refIDs() == ["m1","m2"]`, m1's error is `DeadlineExceeded`, m2 ran with a fresh deadline. This is the only test that may use a real timer, and it waits on the deadline itself, never on a sleep.
  - [x] 4.6 `TestAutoGenerator_CancelledFailuresDoNotCountTowardParking` (AC #5) — three `failed` rows for `m1`, two with `CancelledRunPrefix`; assert `m1` is still enumerated. Add a `cancelledRun(mediaID)` builder beside `failedRun`. Also extend `TestAutoGenerator_ExcludesItemsThatKeepFailing` context: three unmarked rows still park (already covered — just confirm it stays green).
  - [x] 4.7 `TestProcessItem_CancellationMarksTheRunAsCancelled` in `process_item_test.go` — clone of `TestProcessItem_CancellationStillRecordsTheFailure` (`:786`) adding `assert.True(strings.HasPrefix(h.runs.lastUpdate(t).ErrorMessage, CancelledRunPrefix))`; and `TestProcessItem_DeadlineExceededIsNotMarkedCancelled` using `context.WithTimeout` + a fake that returns `context.DeadlineExceeded`, asserting the prefix is absent. Plus `TestProcessItem_KilledProbeUnderCancellationIsMarkedCancelled`: cancel the ctx and have the probe fake return a plain non-ctx error (the `*exec.ExitError` shape) — the prefix must still be present via the `ctx.Err()` half.
  - [x] 4.8 `go test -race ./internal/subtitle/...` must pass — the `wg.Add`-under-lock detail in 1.3 is exactly what `-race` exists to catch.

- [x] **Task 5 — Gates & record** (Rule 15)
  - [x] 5.1 `pnpm nx test api` zero FAIL (never `run_in_background`); `pnpm nx lint api` green; `pnpm format:check` green.
  - [x] 5.2 Fill Dev Agent Record: File List, Completion Notes with the guard-test names as evidence for AC #7 (9R-10a L1 lesson — claims need named tests), Discovery Triage.

### Cross-stack split check

Backend tasks: 5. Frontend tasks: 0. ⇒ **no split** (threshold is >3 on BOTH sides).

## Dev Notes

### Known limits (state them, do not "fix" them here)

- **`Stop()` is unbounded**, exactly like `WorkerPool.Stop` — `wg.Wait()` holds the shutdown block until the in-flight item returns. `ProcessItem` itself never checks `ctx.Err()` on the free lane; cancellation is only observed at the next `database/sql` call or subprocess. A hang inside OpenCC / `Placer` / `feedGlossary` is therefore not interruptible, and the 15-min deadline only takes effect at the next ctx-aware call. Docker's `stop_grace_period` (default 10 s) will SIGKILL first in that case, reproducing the original write-after-close harm — accepted: this story makes the *normal* restart clean, not the pathological one. Do not add a timeout to `Stop()` (it would reintroduce the closed-DB write it exists to prevent).
- **`failItem`'s cleanup writes are outside the per-item deadline** (review L4): they run on `context.WithoutCancel`, so a database that locks up *inside* that cleanup (the 2026-08-01 FUSE class) is bounded by nothing here and `Stop()` waits on it until the stop grace period. The constant's doc comment says so.
- **Head-of-line re-selection after restarts** (review M3): enumeration order is fixed and a cancelled item reverts to `not_searched` with an exempt row, so on a NAS restarted more often than its longest item takes, the same item is selected first every round and never counts toward parking. Accepted as the flip side of AC #5; recorded as input to `bugfix-auto-exclusion-never-expires` (M2) — a bounded exemption (only the N most recent cancelled rows exempt) would close it.
- **The marker fires on any caller cancellation, not only shutdown** (review M2): `WorkerPool` stop and a user cancelling a consent batch mid-item (`batch.go:184`) also write `cancelled:` rows, exempt from parking. Intended — none of those say anything about the file. Nothing outside `excludedMediaIDs` / `isDeferredOutcome` parses `error_message`.
- When the per-item deadline fires mid-ffmpeg, `extractor.go:199-203` reports "ffmpeg timed out after 10m" (it reads `extractCtx.Err()`, which inherits the parent deadline). Cosmetic; do not set `WithAutoItemTimeout` below 10 min in any test that exercises a real extractor.

### What NOT to do

- **Do not add a round-level timeout.** D2 explains why; if you believe one is needed, stop and raise it rather than adding a second deadline layer.
- **Do not change `failItem`'s cleanup semantics.** `context.WithoutCancel(ctx)` (`process_item.go:667`) is what lets a cancelled item still write its `failed` row and revert the media row while the DB is open. D1's ordering (Stop before `db.Close`) is what gives those writes a live handle. Both halves are required.
- **Do not reuse `subtitlePipelineCtx`.** See D5. The pool and the generator stop independently; that is intentional.
- **Do not touch `running` from `Stop()`.** The single-flight guard is owned by `Run`'s deferred unlock; a `Stop` that flips it would race the still-draining round and re-open the 補審 M7 deadlock class.
- **Do not widen `ScanCallback`'s signature.** `ComposeScanCallback(prev, next func())` (`scan_callback.go:25`) and `ScannerService.SetOnScanComplete(func())` are both `func()`; the lifetime ctx is a field, not a parameter.

### Interaction with sibling backlog items (read before designing `Stop`)

- **`bugfix-autogenerator-dropped-round-not-deferred` (M6)** — its planned fix is a `pending` flag that re-runs one round after the current one finishes. That future re-run must **also** be gated on `stopped` and must run under `lifetime` so `Stop()` drains it. Design `ScanCallback`'s "check `stopped` → `wg.Add(1)` → `go`" sequence as a small private method (e.g. `spawnRound()`) so M6 can call the same entry point later. Do **not** implement the `pending` flag here — its entry says the two are to be ruled together, and this story's ruling is: *the lifecycle lands first; M6 builds on `spawnRound()`.*
- **`bugfix-auto-exclusion-never-expires` (M2)** — AC #5 removes the one way THIS story would have made M2 worse (shutdowns silently parking items). It does not address M2's own cases (file replaced / format newly supported); those still need the version-bound marker M2 describes.
- **`preexisting-fail-generation-batch-cancel-mid-item-flake`** — unrelated code path (`GenerationBatchProcessor`), but the same lesson applies to Task 4: no `time.Sleep` synchronisation.

### Architecture compliance

- **Rule 14 (Resource Lifecycle / Graceful Shutdown)** — the whole point: goroutine accepts ctx, honours cancellation, drained by WaitGroup.
- **Rule 13 (Error Handling Completeness)** — cancellation is case 2 (logged, halted for this round); per-item deadline is case 2 per item (logged, round continues). Neither is swallowed.
- **Rule 11 (Interface Location)** — no new ports. `AutoGenerator` already holds every dependency it needs; `Stop()` is a method on the concrete type like `WorkerPool.Stop`.
- **Rule 20 (AC Contract Versioning)** — `ProcessItemOptions` is `[@contract-v1]` (`pipeline.go:96`); this story adds nothing to it. No bump owed. `failItem`'s `ErrorMessage` gains a prefix for one cause — additive, same precedent as `DeferredPaidRunPrefix` (9R-10b, no bump).
- **Rule 2 (slog)** — new log lines use `slog` with structured keys matching the existing `"subtitle auto-generation ..."` family.
- **Rule 9 (Test Co-location)** — tests go in the existing `auto_generation_test.go` / `process_item_test.go`.

### Library / framework

- Go stdlib only: `context.WithCancel`, `context.WithTimeout`, `sync.WaitGroup`, `errors.Is`. No new dependencies. `context.WithoutCancel` (Go 1.21+) is already in use at `process_item.go:667`.

### File structure

| File | Change |
|------|--------|
| `apps/api/internal/subtitle/auto_generation.go` | lifecycle fields, `AutoGenerationItemTimeout`, `WithAutoItemTimeout`, `ScanCallback` rewrite, `Stop()`, loop changes in `Run`, `excludedMediaIDs` exemption |
| `apps/api/internal/subtitle/process_item.go` | `CancelledRunPrefix`, `failItem` marker |
| `apps/api/cmd/api/main.go` | hoist `autoGenerator`, shutdown-block `Stop()`, log key |
| `apps/api/internal/subtitle/auto_generation_test.go` | Task 4.1–4.6 |
| `apps/api/internal/subtitle/process_item_test.go` | Task 4.7 |

### Testing standards

- Table/harness style already in `auto_generation_test.go` (`newAutoHarness`, `autoFakeItemProcessor.outcome` hook). The fake's `ProcessItem` currently ignores its ctx (`_ context.Context`) — Task 4 needs it to **receive** the ctx; change the fake's signature to pass ctx into the `outcome` func (only 2 existing `outcome` assignments, `:432` and `:674` — either change the signature to `func(ctx, ref)` or add a second hook `outcomeCtx`; changing the signature is fine at that churn).
- Synchronise with channels (`entered`/`release` pattern from `TestAutoGenerator_SecondRoundIsSkippedWhileOneIsInFlight`, `:656`). **No `time.Sleep`.**
- Run the package with `-race` at least once (Task 4.8).

### Previous-story intelligence (9R-10b + its CR/補審)

- Fakes must match the **real** contract (CR H2: the fake series resolver once returned `(nil, nil)` where production returns a wrapped `sql.ErrNoRows`, and the test asserted behaviour production didn't have). For this story that means: the fake processor on cancellation must return an error (`ctx.Err()`), exactly as the real `failItem` does — it must not return a nil-error outcome.
- Deleting the single-flight guard once **deadlocked** the package rather than failing a test (補審 M7). `Stop()` + `wg.Wait()` introduce the same class of risk: if `wg.Add` is ever skipped on a path that still calls `Done`, or `Stop` waits on a round that is waiting on `Stop`'s lock, the symptom will be `panic: test timed out`. Keep the lock scope in `Stop()` to the flag+cancel only; `wg.Wait()` happens **outside** the lock.
- Claims need evidence: Completion Notes must name the tests that prove AC #7 (zero paid calls) still hold.

### Git intelligence

Last 5 commits (`aa7671bb`…`97a4c45c`) are 9R-13a/b (.nfo localisation) and 9R-5 (Whisper filter) — no overlap with `auto_generation.go`, `process_item.go` or the `main.go` shutdown block. The most recent touch of these files is 9R-10b's PR #247 and its 補審 follow-ups; branch from current `main`.

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched (backend only; no `apps/web` changes).

### References

- [Source: `apps/api/internal/subtitle/auto_generation.go:177-181` — current `ScanCallback` with `context.Background()`]
- [Source: `apps/api/internal/subtitle/auto_generation.go:25-45, 392-406` — `autoFailureAttemptLimit` and the `failed`-counting loop AC #5 amends]
- [Source: `apps/api/internal/services/ffprobe_service.go:123-127` — killed ffprobe surfaces as `*exec.ExitError`, not `context.Canceled`]
- [Source: `apps/api/internal/subtitle/process_item.go:549` — `DeferredPaidRunPrefix` precedent; `:661-690` — `failItem` with `context.WithoutCancel`]
- [Source: `apps/api/internal/subtitle/worker_pool.go:197-219, 432-454` — `WorkerPool.Stop` contract and ctx-priority loop this story mirrors]
- [Source: `apps/api/internal/subtitle/extractor.go:19-22, 189-211` — 10 min ffmpeg deadline + cancelled-vs-timed-out distinction]
- [Source: `apps/api/internal/services/ffprobe_service.go:65-66, 108-113` — 10 s ffprobe deadline]
- [Source: `apps/api/cmd/api/main.go:591-594, 678-691, 1060-1119` — var block, construction, shutdown sequence]
- [Source: `apps/api/internal/subtitle/process_item_test.go:786` — `TestProcessItem_CancellationStillRecordsTheFailure`]
- [Source: `_bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md:577-581, 673` — CR L1 / 補審 M8 origin]
- [Source: `project-context.md` Rule 13, 14, 20, 24]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — dev-story, 2026-08-23.

### Debug Log References

- RED: package failed to compile on `CancelledRunPrefix` / `WithAutoItemTimeout` before implementation (Task 4 tests written first).
- Fault injection: with the `failItem` marker condition forced to `false`, `TestProcessItem_CancellationMarksTheRunAsCancelled` and `TestProcessItem_KilledProbeUnderCancellationIsMarkedCancelled` go red — the AC #5 tests bite.
- First draft of `TestAutoGenerator_StopCancelsTheInFlightRoundAndDrains` had a test-side race (a `go`-spawned `WaitGroup.Wait` checked via `select default`); replaced with an `atomic.Bool` set inside the item before it returns — `Stop`'s `wg.Wait` is then the only thing that orders the flag before the assertion.

### Completion Notes List

- **D1–D5 implemented as ruled.** `AutoGenerator` owns `lifetime`/`cancel`; `ScanCallback` → `spawnRound()` (stopped-check + `wg.Add` under `mu`, then `go Run(lifetime)`); `Stop()` = flag + `cancel()` under `mu`, `wg.Wait()` outside, Info log once. `Run` returns before the policy read on a dead ctx (Task 1.5 early return), breaks the item loop on `ctx.Err()` and logs `cancelled` (Warn, with `remaining`) instead of `finished`. Each item runs under `context.WithTimeout(ctx, itemTimeout)` with the cancel released right after the call.
- **AC #5 marker**: `CancelledRunPrefix = "cancelled: "` in `failItem`, keyed on `errors.Is(cause, Canceled) || errors.Is(ctx.Err(), Canceled)`; `excludedMediaIDs` skips prefixed rows when counting toward `autoFailureAttemptLimit`. `DeadlineExceeded` is unmarked and counts (test-pinned).
- **main.go**: `autoGenerator` hoisted; `Stop()` after the pool's `Stop()` block and before `db.Close()`; boot log gains `scan_auto_item_timeout`.
- **Tests added (10)**: `auto_generation_test.go` — `StopCancelsTheInFlightRoundAndDrains`, `StopIsIdempotent`, `CancelledRoundStopsIterating`, `AlreadyCancelledRoundDoesNotReadThePolicy`, `ScanCallbackAfterStopIsANoOp`, `EachItemGetsItsOwnDeadline`, `ItemTimeoutDefaultsToTheConstant`, `CancelledFailuresDoNotCountTowardParking` (+ `cancelledRun` builder; fake `outcome` now receives ctx — 2 call sites updated). `process_item_test.go` — `CancellationMarksTheRunAsCancelled`, `KilledProbeUnderCancellationIsMarkedCancelled`, `DeadlineExceededIsNotMarkedCancelled`. Package run 3× under `-race`: green.
- **AC #7 evidence**: `git diff --stat apps/api/internal/cost_consent_test.go` = 0 lines; `TestFreeLane_NeverReachesPaidPorts`, `TestFreeLane_FreeRoutesStillComplete`, `TestFreeLane_DeferredItemStaysOnTheConsentList`, `TestDeferredMarker_WriterAndReaderAgree`, `TestAutoGenerator_AlwaysProcessesFreeOnly` all green in the full run.
- **Gates**: `pnpm nx test api` zero FAIL (all packages ok) · `pnpm nx test web` 237 files passed · `pnpm nx lint api` green · `pnpm format:check` green · `pnpm test:cleanup` → 0 orphaned workers.
- 🔗 **AC Drift: NONE** (checked: `ScanCallback|failItem|error_message` across `_bmad-output/implementation-artifacts/*.md` — 6 hits: 9R-10b, sub-1-5b, sub-1-6, sub-4-1, sub-4-2, this story. sub-1-5b AC #5 "failed → `error_message` diagnostic English + sentinel, bounded 1000 B" still holds — the prefix is English, applied before `truncateErrorMessage`. 9R-10b AC #4 (composed scan callback, free lane only) unchanged. All REUSE.)
- 📎 **Contract Stamps: NONE** (this story defines no `[@contract-v*]` AC; it references `ProcessItemOptions [@contract-v1]` (`pipeline.go:96`) read-only — `FreeOnly` semantics untouched, no bump owed).
- 🎭 **A11y Pre-Flight: N/A** (100% backend — no apps/web/ files touched).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story.
- Pre-existing failures: none observed.

**Code review fixes (2026-08-23, fresh-context adversarial agent — same model, ⚠️ not a model switch):** 0 H / 3 M / 5 L, all 8 fixed.
- ✅ M1 — the mid-flight cancelled item is no longer counted as `failed` nor logged as "item failed"; it folds into `remaining` (Info line "item cancelled mid-flight"), loop breaks there. Test: `TestAutoGenerator_CancelledItemIsNotCountedAsFailed` (captures slog, asserts `failed=0 remaining=2`). Found and fixed a second bug while writing it: `remaining` was being overwritten by the next iteration's loop-head check.
- ✅ M2 — `CancelledRunPrefix` and `excludedMediaIDs` comments now say "caller cancellation (shutdown / pool stop / batch cancel)"; Known limits updated.
- ✅ M3 — head-of-line re-selection recorded in Known limits and in the `excludedMediaIDs` comment as input to M2 `bugfix-auto-exclusion-never-expires`.
- ✅ L4 — `AutoGenerationItemTimeout` doc corrected: `WithoutCancel` cleanup is outside the deadline.
- ✅ L5 — `Run` refuses to start once `stopped` (same `mu` critical section as the single-flight check). Test: `TestAutoGenerator_RunAfterStopIsRefused`.
- ✅ L6 — deadline test asserts `m2Deadline.After(m1Deadline)` instead of `time.Until(dl) > 0` (load-independent).
- ✅ L7 — parking boundary pinned both ways: `limit-1` genuine + 2 cancelled enumerates; `limit` genuine + 2 cancelled stays parked (`TestAutoGenerator_CancelledRowsDoNotRescueAParkedItem`).
- ✅ L8 — `newAutoHarness` registers `t.Cleanup(h.gen.Stop)`.
- Post-fix gates: `go vet` clean · package `-race -count=3` green · `pnpm nx test api` zero FAIL · `pnpm nx test web` 237 files / 2763 tests green · lint api green · format:check green · 0 orphaned workers.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - `N/A — no out-of-scope work discovered` (M6 `bugfix-autogenerator-dropped-round-not-deferred` and M2 `bugfix-auto-exclusion-never-expires` are pre-existing tracked entries; `spawnRound()` was extracted as the story required so M6 can reuse it).

### File List

- `apps/api/internal/subtitle/auto_generation.go`
- `apps/api/internal/subtitle/auto_generation_test.go`
- `apps/api/internal/subtitle/process_item.go`
- `apps/api/internal/subtitle/process_item_test.go`
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/bugfix-autogenerator-no-timeout-or-shutdown.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-23 | code-review (fresh-context agent, same model): 3 M / 5 L found, all fixed — cancelled item not counted as failed (+ `remaining` overwrite bug), `Run` refuses after Stop, boundary tests, harness cleanup, doc corrections, Known limits ×3. Status → done. |
| 2026-08-23 | dev-story (Amelia): Tasks 1–5 implemented — `AutoGenerator` lifecycle (`Stop`, `spawnRound`, per-item 15-min deadline, cancelled-round logging), `CancelledRunPrefix` marker in `failItem` + exemption in `excludedMediaIDs`, main.go shutdown wiring; 11 new tests; all gates green. Status → review. |
| 2026-08-23 | Fresh-context validation pass: 4 line refs corrected; D3/AC #5 marker trigger widened to `ctx.Err()` (killed ffprobe is an `*exec.ExitError`); media-row strand state is `extracting` only; early `ctx.Err()` return before policy read added (Task 1.5); "Known limits" section added (unbounded `Stop`, `stop_grace_period`). |
| 2026-08-23 | Story created (create-story, Bob). Lifecycle decisions D1–D5 made at authoring; three code-verified corrections to the backlog entry's framing: (1) ffmpeg/ffprobe already carry deadlines — the unbounded part is the non-subprocess steps; (2) the shutdown harm is write-after-`db.Close()` stranding items at in-flight status, not a leak; (3) cancellation would feed `autoFailureAttemptLimit` without AC #5. |
