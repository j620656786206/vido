# Story 9R-11 — AI Cost/Quota Controls

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 4 — Robustness prerequisite) · **Owner:** dev (Amelia)
**Date:** 2026-07-05 · **Priority:** P0 (prereq for any batch run) · **Effort:** M · **Feasibility:** PROVEN

## Why

The `ai/` layer had 429 detection + (post-9R-4) retry/backoff, but no throttle, no token/cost
metering, and no budget ceiling. A batch translate/transcribe over a whole library would fan out
unbounded requests (rate-limit failure) and run away on cost. This is the gate before 9R-10
wires the library-wide generation pipeline.

## What shipped

Two composable pieces in `internal/ai/`, both consumed by the Claude (LLM) and Whisper (ASR)
clients so ASR and LLM draw from ONE budget:

### 1. `Governor` — shared throttle (`governor.go`)
- Concurrency semaphore + token-bucket rate limiter (`golang.org/x/time/rate`), built once at
  startup and injected into both clients (`WithClaudeGovernor` / `WithWhisperGovernor`).
- `Acquire(ctx)` waits for a rate token THEN a concurrency slot (Rule 27 ① — don't hold a slot
  while rate-waiting); returns a `release` func. Nil-safe (nil Governor = no-op).
- `governed[T]` helper wraps a call: **budget pre-check → acquire → run**, short-circuiting with
  `ErrBudgetExceeded` before spending when the run's ceiling is hit.

### 2. `Budget` — per-run metering + ceiling (`budget.go`)
- Created per run (per transcription/translation job), carried via `context` (`WithBudget` /
  `BudgetFromContext`) so a batch over many files shares one ceiling without changing every
  signature.
- `RecordLLM(model, in, out)` — token usage + USD cost from a per-model pricing table
  (`claude-haiku-4-5` $1/$5, `sonnet-5` $3/$15, `opus-4-8` $5/$25; unknown → Haiku fallback so
  metering never under-counts). `RecordASR(seconds)` — Whisper $0.006/audio-minute.
- `Exceeded()` gates further calls; `Snapshot()` returns totals for the end-of-run log.
- `maxUSD <= 0` = unlimited (metering still accrues + logs).

### Client integration
- **Claude** (`claude.go`): `claudeResponse` now parses `usage{input_tokens, output_tokens}`;
  `doRequest` runs the retrying request inside `governed(...)` and records usage to the ctx
  budget on success. Reuses the 9R-4 `retryTransient` seam unchanged.
- **Whisper** (`whisper.go`): transcribe runs inside `governed(...)`; on success meters audio
  minutes via the existing `parseWAVInfo` duration (9R-3).
- **Wiring** (`main.go`): one shared `Governor` from config; `TranscriptionService.
  SetRunBudgetUSD` sets the per-run ceiling; `runPipeline` opens a per-run `Budget` on ctx
  spanning BOTH transcription and translation, logging the usage snapshot at the end.

### Config (`config.go`)
`AI_MAX_CONCURRENT` (default 3), `AI_RATE_PER_SEC` (2.0), `AI_RUN_BUDGET_USD` (5.0) + a new
`loadFloat` helper.

## Acceptance Criteria

1. ✅ Concurrency cap + token-bucket throttle across ASR & LLM — one shared `Governor` injected
   into both clients; `Acquire` enforces both.
2. ✅ Token-usage + cost metering logged per job + configurable per-run budget ceiling — `Budget`
   meters LLM tokens/cost + ASR minutes, logs per-call and an end-of-run snapshot; ceiling via
   `AI_RUN_BUDGET_USD` / `SetRunBudgetUSD`.
3. ✅ Backoff/retry shared with 9R-4 — `governed` wraps the existing `retryTransient`; no
   duplication.
4. ✅ Tests for throttle + budget cutoff — governor concurrency-cap / rate-limit / ctx-cancel;
   budget cost/tokens/ASR/exceeded/nil-safe/ctx; **client-level** metering + budget cutoff for
   both Claude and Whisper (no HTTP call once the budget is blown).

## Rule 7

`ErrBudgetExceeded` uses the EXISTING `AI_` prefix (`AI_BUDGET_EXCEEDED`) — code-list addition
only, prefix count unchanged, no CR-workflow sync.

## Dev Notes

- The Gemini provider is not governed here (subtitle generation uses Claude+Whisper; Gemini is
  the filename-parse path, low-volume). If a future batch uses Gemini, inject the same Governor.
- Full library batch orchestration (many media → shared run budget) is **9R-10**; this story
  makes each run's transcription+translation share a budget and provides the shared throttle
  9R-10 relies on.

### Discovery Triage

- **N/A — no out-of-scope work discovered.** The governor/budget seam is exactly what 9R-10
  (pipeline) and any future batch consumer need; no new entries.

### References

- [Source: subtitle-route-c-stories-2026-06.md#9R-11] — ACs.
- [Source: internal/ai/retry.go] — the 9R-4 seam reused (Governor wraps retryTransient).
- [Source: internal/douban/client.go] — `rate.Limiter` precedent (Rule 27 ①).

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- Governor + Budget implemented and wired into Claude + Whisper clients + main.go + transcription
  run. Full api suite + staticcheck green. Reuses the 9R-4 retry seam; ASR + LLM share one
  per-run budget via context.

### File List

- `apps/api/internal/ai/governor.go` (+ test)
- `apps/api/internal/ai/budget.go` (+ test)
- `apps/api/internal/ai/types.go` (ErrBudgetExceeded)
- `apps/api/internal/ai/claude.go` (+ test) — usage parsing, governed doRequest
- `apps/api/internal/ai/whisper.go` (+ test) — governed transcribe, ASR metering
- `apps/api/internal/config/config.go` — AI throttle/budget knobs + loadFloat
- `apps/api/internal/services/transcription_service.go` — per-run budget on ctx
- `apps/api/cmd/api/main.go` — shared Governor + SetRunBudgetUSD wiring
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-11 implemented (dev): shared Governor (concurrency + token-bucket) + per-run Budget (token/cost metering + USD ceiling, ctx-plumbed) wired into Claude + Whisper (reusing the 9R-4 retry seam); config knobs; transcription run shares one budget across ASR+LLM. Tests: throttle + budget cutoff at unit and client level. Full suite + staticcheck green. Status → review. |
