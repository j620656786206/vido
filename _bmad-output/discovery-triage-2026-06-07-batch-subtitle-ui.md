# Discovery Triage — Batch Subtitle Processing has no frontend UI trigger

**Date:** 2026-06-07 · **Raised by:** Murat (TEA) during TestSprite coverage-gap analysis
**Rule:** Discovery Triage (Rule 24) · **Lane:** ③ backlog-with-carry-forward-link
**sprint-status entry:** `disc-2026-06-batch-subtitle-frontend-ui` (backlog)
**Carry-forward link:** Story 8-9 / retro-8-TD4 / PRD `P1-019`

---

## Classification: 未開發 (not-yet-developed) — NOT 開發遺失 (lost mid-development)

The backend for "Batch Subtitle Processing" is **fully built, wired, and tested**. The frontend trigger to invoke it was **never scoped into any story**. Nothing was built and then removed — so this is a *missing-frontend-story* gap, not a regression.

---

## Evidence

### Backend — DONE, production-ready (and backend-only by AC design)

| Artifact | Evidence |
|---|---|
| HTTP routes registered | `subtitle_handler.go:67-68` → `POST /api/v1/subtitles/batch` (StartBatch) + `GET /api/v1/subtitles/batch/status` (GetBatchStatus) |
| DI wiring | `main.go:497-500` — `NewBatchProcessor(...)` + `SetBatchProcessor(...)` (Story 8-9) |
| Engine | `internal/subtitle/batch.go` — BatchScope (season\|library), BatchRequest, BatchProgress, FailedItem, BatchConfig |
| SSE | broadcasts `subtitle_batch_progress` (per project-context §8 event list) |
| Tests | `batch_test.go` — **28 Go test functions** |
| Stories | `8-9-batch-subtitle-processing` = **done**; `retro-8-TD4-season-scope-batch` = **done** |
| AC scope | Story 8-9 ACs 1-6 and TD4 ACs/tasks are **100% backend** (scope handling, SSE payload, `POST /batch` returns 202, rate limits, per-item failure). **Neither story has a frontend AC.** |

### Frontend — the trigger does not exist

| Check | Result |
|---|---|
| Web consumer of `POST /subtitles/batch` or `/batch/status` | **none** (`grep apps/web/src` → 0 hits) |
| Web consumer of `subtitle_batch_progress` SSE | **none** |
| Library selection-mode batch actions | delete / reparse / export only (`useBatchDelete/Reparse/Export`, Story 5-7) |
| `BatchProgress.tsx` | git history → created by **Story 5-7** (delete/reparse/export progress); **never tied to subtitle** |
| "搜尋字幕" entry points | single-item only — `SubtitleSearchDialog` + `MediaDetailPanel` (Story 8-8) |
| Any frontend batch-subtitle story in sprint-status | **none** — `8-8` is "manual-subtitle-search-ui" (single item); no `8-X` for batch UI |

### Priority context

PRD `P1-019 批次字幕處理` is **P2** ("whole season or whole library, queue-based"). `P1-018` (single-item manual search) is the **P0** one and is done (TC071-078). Consistent with: *backend landed in Epic 8, frontend deferred as P2 and never re-scoped.*

> This is the **inverse of Rule 15's precedent** (Epic 10: client method existed ≠ HTTP route registered). Here: **HTTP route registered + tested ≠ UI built.** The API contract is ready and waiting for a consumer that was never scoped.

---

## ✅ PM Decision (2026-06-08): build in v4

Alexyu (acting PM) chose **v4 — build now**. Rationale: backend is done + tested (sunk cost paid) and the design already exists (`subtitle-engine-design-brief.md` G4 desktop + G6 mobile; screenshots in `flow-f-subtitle/`), so the marginal cost is **frontend implementation only** (~1 story), and the design brief frames this as the v4 core differentiator. Deferring would ship a feature whose engine + design exist but users can't reach. **Next action: SM `/create-story`** (see routing below).

## Routing — what each agent should do

- **🧭 PM (John):** ✅ DONE — decided v4 (2026-06-08). No further PM action.
- **🏃 SM (Bob):** On PM go, `/create-story` a frontend story — "Batch Subtitle UI trigger + progress" — with a **carry-forward link to 8-9 / P1-019**. Likely surface: library selection-mode batch action ("Search Subtitles") + a season-level action on the series/season detail, consuming `POST /subtitles/batch` and the `subtitle_batch_progress` SSE (reuse the scan-progress card pattern). Apply Rule 20 ack: confirm against 8-9's API contract.
- **🎨 UX (Sally):** Confirm whether the `.pen` design already has a batch-subtitle screen (the subtitle-engine design brief mentions "batch processing for library-wide subtitle acquisition"). If a screen exists, the gap is purely implementation; if not, a design screen is needed first (per the standalone-spec-screen rule).
- **💻 DEV:** No action until the story exists. When it does: SSE consumer must follow the **lazy-connection pattern** (project-context §8 — never connect on mount; gate on an active batchId) to avoid breaking Playwright `networkidle`.
- **🧪 TEA (me):** Once the UI ships, this unlocks a **TestSprite journey case** (batch search → progress → completion) that is currently impossible (no UI). Until then, batch subtitle stays correctly at the **Go integration level** (28 tests). Tracked.

---

## Secondary finding (logged, low-severity)

`retro-8-TS1` recorded *"28 deprecated tests removed"* (2026-03-27) — but **30 v3-orphan `.py` files were still on disk** as of 2026-06-07. The deletion was marked done without being executed. **Resolved in this session** (Tier-0 `git rm`). Process note for the next retro: a "removed N files" claim should be verifiable by a post-state `ls | wc -l`, not just asserted.
