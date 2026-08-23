# Bugfix: deflake `TestGenerationBatch_CancelMidItem` (and the 8 sibling tests that share its sleep)

Status: done

**Origin:** `preexisting-fail-generation-batch-cancel-mid-item-flake` (Epic 9c AI-2 FILE rule, detected 2026-08-21 during 9R-10b CR-249 補審). `TestGenerationBatch_CancelMidItem` (`apps/api/internal/services/generation_batch_test.go:332`) fails under full-suite load with `Should NOT be empty, but was []`. Proven not caused by that change (isolated 5/5 green, clean tree green). Same class as the solved `bugfix-scanner-sse-cancel-flake`: sleep-based synchronisation against a goroutine. **Test-only** — `GenerationBatchProcessor` and `sse.Hub` production code untouched.

---

## Story

As the team,
I want the generation-batch SSE tests to synchronise on the events they assert rather than on a 50 ms sleep,
so that `pnpm nx test api` is green at any load and a red CI run again means a real regression.

## Root cause (two races, both test-side — verified against code)

1. **Registration race.** `newTestGenerationProcessorWithEpisodes` (`:209-220`) calls `hub.Register()`, which only *enqueues* the client on the `register` channel and returns (`sse/hub.go:145-152`); the `Hub.Run` goroutine registers it later. The test then starts the batch immediately. `Hub.Run` selects among `register` and `broadcast` **randomly** when both are ready (`hub.go:97-115`), so under load the first `Broadcast`s can be fanned out to **zero clients** and dropped before the client exists. Every event of a short batch can be lost this way → `drainEvents` waits 2 s for a first event that never comes → `require.NotEmpty` fails. This is the `[]` in the failure message.
2. **Terminal-event race.** `finish` clears `activeBatch` (so `IsRunning()` flips false and `waitUntilIdle` returns) **before** it calls `broadcast` (`generation_batch.go:521-535`). The tests then `time.Sleep(50ms)` and `drainEvents`, which returns as soon as the channel is momentarily empty after the first event (`:169-201`, non-blocking `default`). Under load the terminal `cancelled` event may not have been fanned out yet → `last["status"]` is `running`. Nine tests carry this same `time.Sleep(50 * time.Millisecond)` (`:259, 278, 322, 360, 399, 433, 721, 751, 804`).

Baseline: 30 isolated runs green on this machine — the flake needs the full-suite CPU contention, exactly as the entry says.

## Fix (test-only)

| # | Change | Where |
|---|--------|-------|
| F1 | **Register synchronously**: after `hub.Register()`, wait (bounded, polling `hub.ClientCount() == 1`, or better: `require.Eventually`) before returning the processor. | `newTestGenerationProcessorWithEpisodes` |
| F2 | **Replace every post-batch `time.Sleep(50ms)` + `drainEvents` with `eventsUntilStatus(t, client, terminal...)`**: a helper that reads `client.Events` (bounded, 2 s) and returns all `generation_batch_progress` payloads **up to and including the first one whose `status` is terminal** (`complete`, `cancelled`, `failed`, `budget_ceiling`, `paused`…). Tests keep asserting on `last` exactly as today. | the 9 call sites |
| F3 | Keep `drainEvents` only if a test needs "everything currently queued" without a terminal (check each site; if none, delete it). | helper |
| F4 | `waitUntilIdle` stays (it is already bounded-polling, not a fixed sleep). | — |

Why not fix the Hub instead: the registration race is real in production too, but harmless — an SSE client that connects mid-broadcast misses at most the events in that microsecond window, and every batch/progress stream re-broadcasts. Making `Register` synchronous is a production change outside a FILE-rule test fix; **note it in the PR, do not change it here.**

## Acceptance Criteria

1. **AC #1** — No `time.Sleep` remains in `generation_batch_test.go` except inside the bounded-polling `waitUntilIdle` (which may itself be replaced by `require.Eventually`).
2. **AC #2** — Every test that previously asserted on `last["status"]` still asserts the same terminal status, obtained via F2, not via a sleep.
3. **AC #3** — The processor helper returns only after the hub has registered the client (F1).
4. **AC #4** — Stress: `go test -race -count=200 ./internal/services/ -run 'TestGenerationBatch'` green, AND the same under artificial load (e.g. `GOMAXPROCS=1`, or 8 concurrent `yes > /dev/null` busy loops) — the condition the flake needs.
5. **AC #5** — `GenerationBatchProcessor` / `sse.Hub` production files: git diff 0 lines.

## Tasks / Subtasks

- [x] **Task 1 — Helpers** (AC #1, #3) — `generation_batch_test.go`
  - [x] 1.1 F1 in `newTestGenerationProcessorWithEpisodes` (`require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, 2*time.Second, time.Millisecond)`).
  - [x] 1.2 Add the F2 helper — implemented as `eventsUntilTerminal(t, client)` (status-agnostic: the first non-`running` event IS the terminal one; prod emits only `complete`/`cancelled`/`budget_ceiling`, no mid-batch `paused` status); `t.Fatal` on the 2 s deadline with the statuses seen so far (so a future failure reads as "never saw X, saw [running running]").
- [x] **Task 2 — Replace the 9 sleep sites** (AC #2) — each: delete the sleep, call the helper with that test's expected terminal status, keep the assertions. Sites: `:259, :278, :322, :360, :399, :433, :721, :751, :804` (line numbers pre-edit).
- [x] **Task 3 — Verify** (AC #4, #5)
  - [x] 3.1 `-race -count=200` on `TestGenerationBatch`; then the same with `GOMAXPROCS=1` and with CPU pinning.
  - [x] 3.2 `pnpm nx test api` zero FAIL; `nx lint api`; `format:check`; git diff of `generation_batch.go` and `sse/hub.go` = 0.

### Cross-stack split check — Backend 3 / Frontend 0 ⇒ no split.

## Dev Notes

- Precedent with the same shape: `bugfix-scanner-sse-cancel-flake.md` — drive the synchronisation from the event you are asserting, never from wall-clock.
- `sse.Hub` drops on a full client buffer (100) with a Warn — batches in these tests are 3 items, far below that; not a factor.
- `drainEvents`'s own comment (`:164-168`) records the previous in-place deflake attempt (sub-5-5): it made the *first* event reliable, not the *terminal* one. This story finishes that thought.
- Do not touch `preexisting-fail-*` entries for other files; Rule 24 applies if a new one is found.

### Time-dependent visual coverage — N/A (backend test-only).

### References

- [Source: `apps/api/internal/services/generation_batch_test.go:152-220, 332-365`]
- [Source: `apps/api/internal/services/generation_batch.go:519-560` — `finish` clears state before `broadcast`]
- [Source: `apps/api/internal/sse/hub.go:81-168` — async `Register`, random `select`, non-blocking fan-out]
- [Source: `_bmad-output/implementation-artifacts/bugfix-scanner-sse-cancel-flake.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-23.

### Debug Log References

- Reproduction BEFORE the fix (git stash, 8× `yes > /dev/null`, `GOMAXPROCS=2`, `-count=100` on `CancelMidItem`): **2/100 FAIL** — the flake needs load, as the entry said.
- AFTER: `-race -count=200` (all `TestGenerationBatch`) green; `GOMAXPROCS=1 -count=300` green; 8-busy-loop CPU pinning `-race -count=100` green.

### Completion Notes List

- `eventsUntilTerminal(t, client)` replaces `time.Sleep(50ms)` + `drainEvents` at all 9 sites (reads until the first non-`running` status, 2 s bound, fails with the statuses seen). `drainEvents` deleted (no remaining caller).
- `newTestGenerationProcessorWithEpisodes` now `require.Eventually`-waits for `hub.ClientCount() == 1` after `Register()` (the async-register / random-select race that produced the `[]`).
- `waitUntilIdle` rewritten with `require.Eventually` — zero `time.Sleep` left in the file (AC #1).
- All existing assertions on `last[...]` unchanged (AC #2).
- Production diff: `generation_batch.go` and `sse/hub.go` 0 lines (AC #5). The Hub's async `Register` remains a (harmless) production quirk — noted in the PR, not changed.
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` 237 files green · lint api · format:check · 0 orphaned workers.
- Code review (fresh-context agent, same model): no blocking findings; 2 LOW fixed (dead `return` after `t.Fatalf`; story Task 1.2 name corrected) + INFO (strict string status compare) applied.
- 🔗 AC Drift: N/A (test-only; no shipped AC's observable behaviour changes). 📎 Contract Stamps: NONE (the `[@contract-v2]` on the SSE payload is referenced by the payload-shape test, which is unchanged). 🎭 A11y: N/A. 🎨 UX: SKIPPED.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - `N/A — no out-of-scope work discovered` (the Hub async-Register observation is a documented non-issue, not work).

### File List

- `apps/api/internal/services/generation_batch_test.go`
- `_bmad-output/implementation-artifacts/preexisting-fail-generation-batch-cancel-mid-item-flake.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-23 | code-review: 0 blocking, 2 LOW + 1 INFO applied. Status → done. |
| 2026-08-23 | dev-story (Amelia): helpers + 9 sites + registration wait; old test reproduced 2/100 under load, new 0/600. Status → review. |
| 2026-08-23 | Story created (create-story, Bob). Two test-side races identified: async `Hub.Register` vs random `select` fan-out (the `[]` failure), and `finish` clearing `IsRunning` before its terminal broadcast (the sleep the 9 tests lean on). |
