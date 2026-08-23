# Bugfix: `AutoGenerator` single-flight guard drops an overlapping trigger instead of deferring it

Status: done

**Origin:** Rule 24 ③ from 9R-10b 補審 **M6** (2026-08-21). The single-flight guard (CR M1) makes a second scan-complete trigger that lands mid-round a **no-op**. But `scanner_service.go:314` only fires the scan-complete callback when `FilesCreated > 0 || FilesUpdated > 0`, so the dropped trigger is not "picked up by the next scan" — it is picked up by the next scan *that happens to add files*, which on a quiet library can be days away. To the user: "I added a file, nothing happened."

**Ruled together with** `bugfix-autogenerator-no-timeout-or-shutdown` (done, PR #263): that story landed the lifecycle first and extracted `spawnRound()` precisely so this one builds on it. Backend only.

---

## Story

As the Vido NAS operator,
I want a scan that completes while a free-lane round is already running to be **queued as one follow-up round** rather than dropped,
so that files added during a long round are processed as soon as the round ends — not after the next lucky scan.

## Root cause (verified against code, post-#263)

- `Run` (`auto_generation.go:295-298`): `if g.running { … "skipping"; return }`. Dropped, not deferred.
- `ScannerService.StartScan` → `onScanComplete()` only when files were created/updated (`scanner_service.go:314`). Nothing else re-triggers the free lane. `WorkerPool` is unaffected (it queues per item; this is the round-level trigger).
- Scheduled scans + a manual scan are the common overlap: the scheduled one finishes while the manual one's round is still extracting. The manual scan's new files are the ones dropped.

## Design

One boolean, merged triggers, re-run through the existing lifecycle entry:

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Where the deferral lives | `pending bool` next to `running`, guarded by the same `mu` | One lock, one invariant: *after a round ends, if `pending` then exactly one more round starts.* |
| D2 | N overlapping triggers → how many re-runs | **One.** `pending` is a flag, not a counter | Each round re-enumerates the whole eligible set (up to `maxPerRun`); a second follow-up would find nothing the first did not. |
| D3 | How the follow-up starts | Through **`spawnRound()`** (goroutine under `lifetime`, `wg`-tracked, refused when `stopped`) | It is the one entry #263 built for this. A follow-up must not escape `Stop()`'s drain. |
| D4 | Stop while a follow-up is pending | **Dropped** — `Stop()` clears `pending`; `spawnRound` refuses anyway | Shutdown is not the time to start work; the next boot's first scan covers it. |
| D5 | Follow-up when the first round was cancelled (not via Stop) or aborted | Still runs (the flag is independent of the outcome) | A round that died on a locked DB is exactly the one whose trigger should not be lost. `spawnRound`'s `stopped` check is the only veto. |

`wg` safety (corrected at review, M2): the Add **can** be from zero (a follow-up spawned at the end of a round entered by a direct `Run`). What makes it safe is the `mu` ordering — `spawnRound` does the `stopped` check and the `wg.Add` in one critical section, and `Stop` sets `stopped` under the same lock before `wg.Wait` — so an Add either happens-before the Wait or is refused. Never hoist the `stopped` check out of the lock.

## Acceptance Criteria

1. **AC #1 — Overlap defers.** Given a round is in flight, when a second trigger arrives (`ScanCallback()()` or a direct `Run`), then the second call returns immediately (as today) AND exactly one follow-up round starts after the first completes, processing the full eligible set again.
2. **AC #2 — Merge.** Given N ≥ 2 triggers arrive during one round, then exactly **one** follow-up round runs (total rounds = 2).
3. **AC #3 — No re-trigger loop.** Given the follow-up round runs with no trigger during it, then no third round starts (`pending` is cleared when the follow-up is claimed).
4. **AC #4 — Stop wins.** Given a trigger was deferred during a round and `Stop()` is called before the round ends, then no follow-up runs and `Stop()` still drains cleanly.
5. **AC #5 — Logging.** The "already in flight — skipping" Debug line becomes "already in flight — follow-up round queued"; the follow-up's start is logged at Info with `"reason", "deferred_trigger"`.
6. **AC #6 — Existing guarantees hold.** `TestAutoGenerator_SecondRoundIsSkippedWhileOneIsInFlight` is rewritten to the new contract (no two rounds *concurrently*; the second is queued, not dropped); every other test in the package stays green; `cost_consent_test.go` diff 0; `-race` green.

## Tasks / Subtasks

- [x] **Task 1 — `pending` flag** (AC #1–#5) — `apps/api/internal/subtitle/auto_generation.go`
  - [x] 1.1 Add `pending bool` beside `running` with a comment naming this story and the `scanner_service.go:314` trigger condition.
  - [x] 1.2 In `Run`'s guard: `if g.running { g.pending = true; unlock; Debug "follow-up round queued"; return }`.
  - [x] 1.3 In `Run`'s deferred unlock: `g.running = false; rerun := g.pending; g.pending = false; unlock; if rerun { Info "…starting deferred follow-up round" reason=deferred_trigger; g.spawnRound() }`. `spawnRound` is called **outside** `mu` (it takes `mu` itself).
  - [x] 1.4 `Stop()`: clear `pending` in the same critical section that sets `stopped` (D4).
- [x] **Task 2 — Tests** (all ACs) — `apps/api/internal/subtitle/auto_generation_test.go`
  - [x] 2.1 Rewrite `TestAutoGenerator_SecondRoundIsSkippedWhileOneIsInFlight` → `TestAutoGenerator_OverlappingTriggerIsDeferredNotDropped`: same `entered`/`release` choreography, then `h.gen.Stop()` to drain, assert `refIDs() == [m1 m2 m1 m2]` and that the second `Run` call returned before `release` (no concurrency).
  - [x] 2.2 `TestAutoGenerator_ManyOverlappingTriggersMergeIntoOneFollowUp` — three triggers mid-round → `[m1 m2 m1 m2]`.
  - [x] 2.3 `TestAutoGenerator_FollowUpDoesNotReTrigger` — covered by 2.1/2.2's exact-length assertion after drain; add a round counter on the fake policy (`calls == 2`) to make it explicit.
  - [x] 2.4 `TestAutoGenerator_StopDropsThePendingFollowUp` — trigger mid-round, `Stop()` while the item is blocked, release; assert `policy.calls == 1` and `refIDs()` has no second pass.
  - [x] 2.5 `-race -count=5` on the package.
- [x] **Task 3 — Gates** — `pnpm nx test api`, `pnpm nx test web`, `nx lint api`, `format:check`; Dev Agent Record.

### Cross-stack split check

Backend 3 / Frontend 0 ⇒ no split.

## Dev Notes

- **Do not** make `pending` a counter (D2) and **do not** re-run via `go g.Run(...)` directly — only `spawnRound()` (D3).
- **Do not** touch `scanner_service.go:314`'s trigger condition; widening it to every scan is a different (product) decision.
- `spawnRound()` must be called after `mu` is released in the defer — it locks `mu` itself; calling it under the lock deadlocks (the 補審 M7 class: shows up as `panic: test timed out`).
- Rule 14 / 13 as in #263. No new ports, no schema, no contract stamps touched.

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched (backend only).

### References

- [Source: `apps/api/internal/subtitle/auto_generation.go:234-260` — `spawnRound`; `:289-306` — single-flight guard + deferred unlock]
- [Source: `apps/api/internal/services/scanner_service.go:314` — trigger only on created/updated files]
- [Source: `_bmad-output/implementation-artifacts/bugfix-autogenerator-no-timeout-or-shutdown.md` — D1–D5, "Interaction with sibling backlog items"]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-23.

### Debug Log References

- RED: the three deferral tests failed before the `pending` flag (`[m1 m2]` vs expected `[m1 m2 m1 m2]`).
- Two test-choreography fixes found by stress runs, not by the implementation: (1) `go`-spawned `ScanCallback` triggers scheduled AFTER the round ended are legitimately new triggers → 3 rounds; mid-round triggers now use synchronous `Run`. (2) `StopDropsThePendingFollowUp` raced `Stop` against the round's end (`GOMAXPROCS=1` → 1998/2000 fail); the 2nd item now blocks on `ctx.Done()`, which fires only after `stopped/pending` were written under `mu`.

### Completion Notes List

- `pending bool` under `mu`; `Run`'s guard queues instead of dropping; deferred unlock claims the flag and calls `spawnRound(spawnReasonDeferred)` outside the lock; `Stop` clears `pending`. `spawnRound` now returns bool and takes a reason; the follow-up Info line is logged inside it on the success path only (review L4).
- `Run` checks `ctx.Err()` **before** the single-flight guard (review L5) so a dead-ctx trigger neither starts nor queues.
- Tests: `OverlappingTriggerIsDeferredNotDropped` (replaces `SecondRoundIsSkippedWhileOneIsInFlight`; still pins "no concurrent pass"), `ManyOverlappingTriggersMergeIntoOneFollowUp`, `StopDropsThePendingFollowUp`, `DeferredFollowUpIsLogged` (AC #5). Stress: `-race -count=300` and `GOMAXPROCS=1 -count=1000` green.
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` 2763 green · lint api · format:check · 0 orphaned workers · `cost_consent_test.go` diff 0.
- 🔗 AC Drift: FOUND — 9R-10b CR M1 single-flight contract "second scan completing mid-round is a no-op" → "queued as one follow-up round; still never concurrent". Recorded here; `9R-10b-on-add-autotrigger.md` is the reference.
- 📎 Contract Stamps: NONE (no `[@contract-v*]` in scope).
- 🎭 A11y Pre-Flight: N/A (100% backend). 🎨 UX Verification: SKIPPED.
- Code review (fresh-context agent, same model): 1 H / 2 M / 3 L — all fixed (H1 flaky test, M2 wrong wg-safety argument, M3 stale comments, L4 premature Info log, L5 guard order, L6 AC #5 untested).

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - `N/A — no out-of-scope work discovered`.

### File List

- `apps/api/internal/subtitle/auto_generation.go`
- `apps/api/internal/subtitle/auto_generation_test.go`
- `_bmad-output/implementation-artifacts/bugfix-autogenerator-dropped-round-not-deferred.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md` (AC drift reference — see Completion Notes; not modified)

## Change Log

| Date | Change |
|------|--------|
| 2026-08-23 | dev + code-review (1H/2M/3L all fixed) + gates green. Status → done. |
| 2026-08-23 | Story created (create-story, yolo) on top of #263's `spawnRound()`. |
