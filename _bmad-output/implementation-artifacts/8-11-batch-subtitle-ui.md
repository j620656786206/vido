# Story 8.11: Batch Subtitle Search UI (frontend trigger + progress)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

> **Origin:** Discovery Triage `disc-2026-06-batch-subtitle-frontend-ui` (Rule 24, 2026-06-07) — the batch-subtitle backend (Story 8-9 + retro-8-TD4) shipped complete and tested, but **no frontend story ever scoped a UI trigger**. PM decision (2026-06-08): build in **v4**. This is the frontend half. **Post-epic addendum** under Epic 8 (which remains `done`); backend is already done, so this is a **frontend-only** story (no cross-stack split — backend task count is 0).

## Story

As a **media collector**,
I want **to trigger a batch subtitle search for my whole library (or a TV season) from the UI and watch live progress**,
so that **I don't have to open the single-item subtitle dialog for every movie and episode one at a time**.

## Acceptance Criteria

> Scoped strictly to what the **Story 8-9 backend contract** actually supports (Rule 20 ack — confirmed against 8-9 AC #1–#8). Design G4/G6 elements NOT backed by 8-9 (Pause, a separate "converted" counter, a distinct "missing-only" scope) are triaged out — see Discovery Triage.

1. **Given** the user is on `/library`,
   **When** they enter selection mode and open the batch actions,
   **Then** a **"批次字幕搜尋"** action is visible alongside the existing batch delete / reparse / export actions;
   **And** it is reachable via `data-testid="batch-subtitle-btn"`.

2. **Given** the user opens the batch subtitle panel,
   **When** the panel renders,
   **Then** it shows a **scope selector** with the two backend-supported scopes: **「整個媒體庫」(library)** pre-selected, and **「整季」(season)** shown only when a TV-season context is available (`seasonId` present);
   **And** a primary **「開始批次搜尋」** button (`data-testid="batch-subtitle-start-btn"`).

3. **Given** the user clicks 「開始批次搜尋」 with scope `library`,
   **When** the request is sent,
   **Then** the frontend calls `POST /api/v1/subtitles/batch` with body `{ scope: "library" }` (camelCase→snake_case per Rule 18);
   **And** on **202 Accepted** it reads `{ batchId, totalItems }` and transitions the panel to the processing state.

4. **Given** a batch is processing,
   **When** `subtitle_batch_progress` SSE events arrive,
   **Then** the panel shows a **progress bar** + **「{currentIndex} / {totalItems}」** counter (monospace),
   **And** the **current item title** (`currentItem`),
   **And** **found / not-found** counters mapped from `successCount` / `failCount`;
   **And** all dispatches are guarded by a mounted ref (no state update after unmount).

5. **Given** a batch is processing,
   **When** the user clicks **「取消」** (`data-testid="batch-subtitle-cancel-btn"`),
   **Then** a confirmation appears (「確定要取消嗎？已處理的結果會保留。」);
   **And** confirming cancels the batch (backend context-cancellation path); the panel reflects the `cancelled` status.

6. **Given** the batch reaches `status: "complete"`,
   **When** the completion SSE event arrives,
   **Then** the panel shows a summary (找到 {successCount} · 未找到 {failCount} · 共 {totalItems})
   **And** a **「關閉」** button, and a **「查看未找到項目」** link that navigates to the library filtered to `subtitle_status=not_found`.

7. **Given** a batch is already running,
   **When** the user attempts to start another,
   **Then** the **409 Conflict** response is handled gracefully — the panel shows the in-progress batch (from the 409 body / `GET /api/v1/subtitles/batch/status`) instead of erroring.

8. **Given** the SSE connection,
   **When** the batch panel mounts,
   **Then** it **MUST NOT** open `EventSource` on mount — it connects only after a batch is started (lazy pattern, project-context §8);
   **And** cleans up `EventSource.close()` on unmount and on completion.

9. **Given** the mobile viewport,
   **When** a batch is processing,
   **Then** the progress surfaces as a bottom-sheet peek (per design G6 — simplified: progress + single-row stats + current item, no recent-results list).

## Tasks / Subtasks

> All FRONTEND. Backend (Story 8-9) is `done` — do NOT modify Go code; consume the existing contract.

- [ ] **Task 1: Subtitle service — batch methods (AC: #3, #7)**
  - [ ] 1.1 Add `startBatch(params: { scope: 'library' | 'season'; seasonId?: number })` to `apps/web/src/services/subtitleService.ts` → `POST /subtitles/batch`, body wrapped in `camelToSnake(...)`, response through `snakeToCamel` (Rule 18). Return type `{ batchId: string; totalItems: number }`.
  - [ ] 1.2 Add `getBatchStatus()` → `GET /subtitles/batch/status`, returns the progress shape.
  - [ ] 1.3 Handle 409: surface the response body's in-progress progress to the caller (don't throw a generic error).
  - [ ] 1.4 Define TS types: `BatchScope`, `BatchStartResult`, `BatchProgress` (`batchId`, `totalItems`, `currentIndex`, `currentItem`, `successCount`, `failCount`, `status`) in `libs/shared-types` or co-located, matching the snake_case wire payload.

- [ ] **Task 2: Lazy SSE hook `useSubtitleBatchProgress` (AC: #4, #8)**
  - [ ] 2.1 Create `apps/web/src/hooks/useSubtitleBatchProgress.ts` modeled EXACTLY on `useScanProgress.ts` (eventSourceRef + mountedRef + `startTracking()` gate + backoff reconnect, NO connect-on-mount).
  - [ ] 2.2 `addEventListener('subtitle_batch_progress', ...)` → reduce into progress state; guard with `mountedRef.current`.
  - [ ] 2.3 Treat the `status: "complete"` and `status: "cancelled"` payloads as terminal → close `EventSource`.
  - [ ] 2.4 Expose `{ progress, status, startTracking, reset }`. Reconnect with backoff on error; do NOT fall back to polling.

- [ ] **Task 3: Batch subtitle panel component (AC: #2, #4, #5, #6, #9)**
  - [ ] 3.1 Create `apps/web/src/components/subtitle/BatchSubtitleDialog.tsx` — desktop side-panel/modal per design G4; reuse the SidePanel/Dialog UI primitive used by `SubtitleSearchDialog`.
  - [ ] 3.2 Idle state: scope radio selector (library default; season only when `seasonId` provided) + `batch-subtitle-start-btn`.
  - [ ] 3.3 Processing state: progress bar (`data-testid="batch-subtitle-progress-bar"`), monospace `{idx}/{total}` counter, current-item line, found/not-found stat counters, `batch-subtitle-cancel-btn`.
  - [ ] 3.4 Cancel confirmation (reuse the confirm pattern from library batch delete) → on confirm, call cancel + reflect `cancelled`.
  - [ ] 3.5 Completed state: summary counts + `batch-subtitle-close-btn` + 「查看未找到項目」 link → `/library?subtitle_status=not_found` (verify the library route's filter param name before wiring).
  - [ ] 3.6 Mobile (G6): render as bottom-sheet peek (72px) with simplified single-row stats; reuse the mobile sheet pattern from `SubtitleSearchDialog` mobile / Detail Panel mobile.
  - [ ] 3.7 `data-testid="batch-subtitle-dialog"` on the root; Escape closes (idle/completed only — guard against closing mid-run without confirm).

- [ ] **Task 4: Wire the trigger into the library selection toolbar (AC: #1)**
  - [ ] 4.1 In `apps/web/src/routes/library.tsx` selection-mode toolbar (near `select-all-btn` / the existing batch delete/reparse/export actions), add a 「批次字幕搜尋」 button `data-testid="batch-subtitle-btn"` that opens `BatchSubtitleDialog` with scope=library.
  - [ ] 4.2 (If low-effort) also expose the season-scope entry from a TV-season context where `seasonId` is available; otherwise note it as a follow-up. Do NOT silently expand scope — if season wiring is non-trivial, record it in Discovery Triage.

- [ ] **Task 5: Tests (AC: all)**
  - [ ] 5.1 `subtitleService.spec.ts`: `startBatch` sends snake_case body + parses camelCase; 409 path returns in-progress progress.
  - [ ] 5.2 `useSubtitleBatchProgress.spec.ts`: no EventSource on mount; opens only after `startTracking()`; reduces a `subtitle_batch_progress` event; closes on `complete`; no dispatch after unmount.
  - [ ] 5.3 `BatchSubtitleDialog.spec.tsx`: idle→processing→complete state machine; cancel confirmation; 「查看未找到項目」 navigation; scope selector visibility (season hidden without `seasonId`). Use Rule 16 assertions (`toBeInTheDocument`, `toBeAttached` for any hover/transition).
  - [ ] 5.4 Add the component to the visual gallery fixtures (`apps/web/src/routes/test/-gallery.fixtures.tsx`) for idle + processing + complete states so visual baselines exist (Rule 21/22). Capture `-darwin` + `-linux` baselines per the 19-4b/19-5 workflow.

## Dev Notes

### API contract to consume (Rule 20 ack — confirmed against Story 8-9 [done])

- `POST /api/v1/subtitles/batch` — body `{ scope: "season" | "library", seasonId?: number }` → **202** `{ batchId, totalItems }`; **409** `{ ...progress }` if a batch is already running. [Source: 8-9-batch-subtitle-processing.md AC #4, #7; subtitle_handler.go:67 `StartBatch`]
- `GET /api/v1/subtitles/batch/status` → current `BatchProgress`. [Source: subtitle_handler.go:68 `GetBatchStatus`]
- SSE event `subtitle_batch_progress` (NOT `subtitle_progress`) — payload `{ batch_id, total_items, current_index, current_item, success_count, fail_count, status }`; terminal `status: "complete"` event carries summary counts; `status: "cancelled"` on cancel. [Source: 8-9 AC #3/#8, Task 4; `sse/hub.go EventSubtitleBatchProgress`; project-context.md §8 broadcast event list]
- **Backend scope semantics:** `library` scope already filters to items with `subtitle_status` ∈ {`not_searched`, `not_found`} and skips `found` (8-9 AC #2). So "missing-only" is INHERENT to library scope — do not build a separate "missing" scope. `season` scope needs `seasonId` (retro-8-TD4). [Source: 8-9 AC #1/#2; retro-8-TD4-season-scope-batch.md]

### Architecture patterns & constraints (project-context.md — the bible)

- **Lazy SSE (§8, Epic 7 retro lesson — CRITICAL):** NEVER `new EventSource()` in a mount-time `useEffect([])`. Gate on a user action / active batchId. Eager SSE breaks Playwright `networkidle`. Mirror `useScanProgress.ts` (`startTracking()` + `mountedRef` + backoff, no polling fallback). [Source: project-context.md §8 "Frontend Lazy SSE Connection Pattern"; apps/web/src/hooks/useScanProgress.ts]
- **Rule 5 — server state via TanStack Query** for the `getBatchStatus` fetch; the live progress stream is SSE-driven local reducer state (same split as scan progress).
- **Rule 18 — API boundary case transform:** every POST body `camelToSnake(...)`, every response `snakeToCamel(...)`. `subtitleService.ts` already uses the shared `fetchApi`. [Source: project-context.md Rule 18; subtitleService.ts:6]
- **Rule 16 — specific assertions** in tests (`toBeInTheDocument`, `toBeAttached` for hover/transition).
- **Rule 9 — co-located tests** (`*.spec.tsx` beside source).

### Source tree components to touch / reuse

- `apps/web/src/services/subtitleService.ts` — extend (has `search`/`download`/`preview` via `fetchApi`).
- `apps/web/src/hooks/useScanProgress.ts` — COPY the lazy-SSE shape (do not reinvent).
- `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx` — reuse dialog/side-panel + mobile-sheet patterns + testid conventions.
- `apps/web/src/components/library/BatchProgress.tsx` — existing Story 5-7 batch progress (delete/reparse/export); reference for the progress-bar/cancel/close pattern + testids (`batch-progress`, `progress-bar`, `progress-cancel-btn`, `progress-close-btn`, `progress-text`). The subtitle batch panel is RICHER (G4) so it gets its own component, but match the visual language.
- `apps/web/src/routes/library.tsx` — selection-mode toolbar (`enterSelectionMode`, `selectedIds`, `select-all-btn`, existing batch mutations) is where the trigger goes.

### Design reference

- **G4 (desktop)** + **G6 (mobile)** in `_bmad-output/planning-artifacts/subtitle-engine-design-brief.md` (lines ~178–315); screenshots in `_bmad-output/screenshots/flow-f-subtitle/`. Colors/spacing per the design system (--accent-primary start button, --success/--error stat counters, 6px progress bar).
- ⚠️ **Design exceeds backend (scoped OUT — see Discovery Triage):** G4 shows a **Pause/繼續** control, a separate **「轉換」(converted)** stat, and a third **「缺少字幕的項目」** scope radio. The 8-9 backend supports none of these as distinct features (no pause endpoint; SSE has only success/fail counts; library scope already = missing-only). Build the MVP without them; they are triaged to a backend backlog item.

### Project Structure Notes

- New files: `components/subtitle/BatchSubtitleDialog.tsx` (+ `.spec.tsx`), `hooks/useSubtitleBatchProgress.ts` (+ `.spec.ts`). Edits: `services/subtitleService.ts`, `routes/library.tsx`, `test/-gallery.fixtures.tsx`.
- Naming per Rule 6 (PascalCase.tsx components, camelCase hooks/services). New component file needs the Rule 21 `// Implements:` header — this is a screen-section component, so use the `// Design ref: ux-design.pen — ...` form (no exact `.pen` Reusable Component node; G4/G6 are screen frames). Verify against the ESLint `local/implements-pen-node-id` rule before commit.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.** The batch panel renders server-driven progress (counts, current item, status); no relative-time / wall-clock rendering. If the dev introduces an elapsed-time or ETA display, this MUST be revisited (clock-mock per Rule 23 + ≥2 fixture state baselines).
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### Testing standards summary

- Vitest + RTL, ≥70% coverage (project-context Testing Infrastructure). Co-located specs. Specific matchers (Rule 16). Foreground test runs only (memory: No Background Tests — orphaned vitest workers). Add gallery fixtures for visual baselines (Rule 21/22), capturing `-darwin` + `-linux`.
- A TestSprite journey case for batch subtitle becomes possible ONCE this UI ships (currently blocked by "no UI" — see `disc-2026-06-batch-subtitle-frontend-ui` / test-design-testsprite-coverage-2026-06.md). Not part of this story's DoD, but flag for TEA `*automate` follow-up.

### References

- [Source: _bmad-output/discovery-triage-2026-06-07-batch-subtitle-ui.md] — origin + PM v4 decision + routing
- [Source: _bmad-output/implementation-artifacts/8-9-batch-subtitle-processing.md] — backend contract (AC #1–#8)
- [Source: _bmad-output/implementation-artifacts/retro-8-TD4-season-scope-batch.md] — season scope (`seasonId`)
- [Source: _bmad-output/planning-artifacts/subtitle-engine-design-brief.md#G4] + #G6 — UI design
- [Source: apps/api/internal/handlers/subtitle_handler.go:60-68] — registered routes
- [Source: apps/web/src/hooks/useScanProgress.ts] — lazy-SSE reference implementation
- [Source: project-context.md §8 Real-Time Events / Lazy SSE; Rule 5; Rule 16; Rule 18; Rule 21]
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md] — P1-019 (批次字幕處理, P2)

## Dev Agent Record

### Agent Model Used

(to be filled by dev)

### Debug Log References

### Completion Notes List

### Discovery Triage

<!-- Rule 24 — out-of-scope work surfaced DURING story authoring is triaged here at discovery time. -->

- **Did this story discover any work outside its current scope?** **YES** — design (G4) specifies UI features the Story 8-9 backend does not support. Triaged below:
  - **③ backlog-with-carry-forward-link** — **Pause/Resume for batch subtitle** (G4 「暫停」/「繼續」). 8-9 backend supports cancel (context cancellation) but has NO pause/resume endpoint. → file backlog `disc-2026-06-batch-subtitle-pause` (backend story; this story 8-11 ships cancel-only). Bidirectional: 8-11 omits Pause; the backlog entry names 8-11 as the consumer that will add the control once the backend lands.
  - **③ backlog-with-carry-forward-link** — **「轉換 (converted)」 stat counter** (G4 4th counter). The `subtitle_batch_progress` SSE payload has only `success_count` / `fail_count` — no converted count. → same backlog `disc-2026-06-batch-subtitle-pause` (or a sibling `...-converted-stat`): backend must add a converted counter to the SSE payload before the frontend can show it. 8-11 shows found/not-found only.
  - **① expand-scope-in-place (clarification, not new work)** — the design's 「缺少字幕的項目」 (missing-only) scope is NOT a third backend scope; it is what `library` scope already does (skips `found`). 8-11 AC #2 absorbs this by labeling the library scope accurately — no separate scope built. No new tracked entry needed.
- Reference: `project-context.md` Rule 24; origin `disc-2026-06-batch-subtitle-frontend-ui`.

### File List

(to be filled by dev)

## Change Log

- 2026-06-08: Story created (SM Bob, *create-story *yolo). Frontend-only; backend 8-9 done. Scoped to 8-9 contract; Pause + converted-stat triaged to backend backlog (Rule 24). Status → ready-for-dev.
