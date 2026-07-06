# Story ux3-subtitle-v2-batch: Subtitle UI v2 — batch generation surfaces (PH3-M5 slice 2)

Status: review

> **Depends on: `9R-16-batch-generation-endpoint` (backend API must be ready) + `ux3-subtitle-v2` slice 1 (shared components must exist).** Series-CTA flip additionally gated on `9R-10a`. Authored as the FE half of the 9R-16 pair (13-7a/b precedent) — all endpoint/SSE shapes below are 9R-16 [@contract-v1]; **re-verify against 9R-16's shipped code at dev time** if its review changed anything.

## Story

As a Vido user staring at a library where 38 titles lack 繁中 subtitles,
I want to launch batch generation from the Activity hub or my library selection, watch it progress live, and have it stop cleanly at my budget with the remainder queued for next time,
so that bulk healing is one action — and running out of budget reads as "done for today", never as a failure.

## Acceptance Criteria

1. **Batch dialog (F8 — `F8-D-v2` i9Nun1 desktop / `F8-M-v2` H717g mobile sheet).** New `GenerationBatchDialogV2`: title `批次生成字幕`; scope selector `範圍：` with segments **`缺字幕的項目`** (+ Mono count from `GET /subtitles/generation-batch/preview?scope=missing`) and **`已選項目`** (rendered ONLY when the dialog is opened with a non-empty selection — see AC 5; otherwise the segment is absent, scope=missing preselected); idle → start via `POST /subtitles/generation-batch` (202 returns `{batch_id,total_items,items[]}` — render the F8 queue rows from `items`); running state: `已完成` + Mono `12 / 38` + progress bar, per-item rows 完成/轉錄中/排隊中 (the active row shows the frozen stage stepper by joining `transcription_*` events on `current_media_id` — reuse slice-1's `GenerationProgressV2`/`useGenerationProgress`), cost line `本次用量：` Mono `$X.XX` ` / 上限 ` Mono `$5.00` (from the batch SSE `spent_usd`/`budget_usd`), `即時更新（SSE）` indicator, primary `全部取消` (→ `POST .../cancel`; confirm-inline per BatchSubtitlePanel precedent). 409 `TRANSCRIPTION_BATCH_RUNNING` on open/start → recover-and-attach (on-open `GET .../status` probe, fetch-dialog precedent). All numerics `font-mono tabular-nums`, number+unit split Mono/Noto.
2. **Budget-ceiling state (F9 — `JMqPg` 批次生成—預算上限).** When SSE `status:"budget_ceiling"`: banner `已達本次預算上限（` Mono `$5.00` `）— 已完成` Mono `N` `部，剩餘` Mono `M` `部下次繼續`; paused rows read `已暫停 — 下次繼續`; actions become `關閉` + **`下次繼續`** (= start a NEW `scope=missing` batch — 9R-16 resume-for-free ruling; completed items self-exclude). This is a NORMAL terminal state — success styling semantics, NOT error tokens.
3. **Service + hook.** `generationBatchService` (or extend `subtitleService.ts`): `startGenerationBatch({scope, mediaIds?})` (Rule 18 `camelToSnake` → `media_ids`), `getGenerationBatchStatus()`, `cancelGenerationBatch()`, `previewGenerationBatch()`. NEW `useGenerationBatchProgress` hook — sibling of `useSubtitleBatchProgress` (its reducer/terminal-close shape) listening for **`generation_batch_progress`** with the ⚠️ **double-nested envelope** (`parsed.data`, hub convention — NOTE the fetch hook's single-nest unwrap does NOT transfer; slice-1's unwrap does), `snakeToCamel` at ingest, lazy connect (§8), terminal states `complete|cancelled|error|budget_ceiling` close the stream.
4. **Activity hub integration.** (a) Launch CTA: header-area button `批次生成字幕` (`data-testid="activity-generation-batch-cta"`) opening the dialog with scope=missing — the hub's first action-button (existing CTAs are Links; keep it visually consistent with hub tokens). (b) `ACTIVE_META` gains kind **`generation_batch`** → `批次生成` (Captions-family icon) rendering `current / total` like `subtitle_batch`; extend `ActiveJob.kind` union in `activityService.ts`. D4-1 boundary: the hub row + CTA are the ONLY Activity-side entries — no second competing surface.
5. **Library multi-select entry (legacy shell) re-point.** `SelectionToolbar`'s subtitle button re-labels 批次字幕搜尋 → `批次生成字幕` (keep `data-testid="batch-subtitle-btn"` unless specs force rename — document either way) and `routes/library.tsx:795`'s dialog swaps `BatchSubtitleDialog` → `GenerationBatchDialogV2` **with the actual `selectedIds` passed** (today's wiring discards the selection — fix it: selected ids → `scope=selected` + `media_ids`, movies only; selected series → excluded with a visible note, capability honor). Fetch-batch dialog file stays (superseded reference; gallery fixture keeps it compiling) but is no longer reachable from live UI — record this in Completion Notes. ⚠️ v2 shell (`LibraryBrowseV2`) has NO selection mode at all — that gap is `disc-2026-07-v2-library-multiselect` (filed by this story); do NOT build v2 multi-select here. In the v2 shell the feature is reachable via the Activity CTA (scope=missing).
6. **Series 生成字幕 flip (conditional — 9R-10a).** IF `9R-10a` is merged when this story is developed: flip slice-1's disabled series CTA in `ManageSubtitleDialogV2` live (confirm 9R-10a's route shape first — do NOT invent it). IF NOT merged: leave disabled, note it, and this task carries forward on the 9R-10a entry. Batch scope stays movies-only either way (9R-16 AC 8).
7. **Four-state + a11y.** Dialog states: idle (scope + counts) / running / budget_ceiling / cancelled / error / empty-scope (`total_items:0` → `目前沒有缺字幕的項目` friendly state, not an error). `role=progressbar` on bars, `aria-live` on status transitions, Escape gated while running (fetch-dialog precedent), skeleton respects `prefers-reduced-motion`.
8. **Rule 21 / 23 / 26.** New component files: `// Design ref: ux-design.pen Screen F8-D-v2 (i9Nun1)` (+ F9 ref where apt) — verify IDs via Pencil MCP. NO wall-clock reads — all timing/progress/cost from SSE payload (frozen-stage + Rule 23 handoff). New search params: none planned; the dialog opens from CTAs, not deep links (if one is added, string-coerce per `toCsvString`). Token-only styling; gallery fixtures for dialog states incl. `budget_ceiling` (fixture states named after batch statuses; `-linux` via CI bootstrap PR).
9. **Tests + gates.** Specs: service (4 methods, envelope, 409 body), hook (double-nest unwrap asserted, terminal close per status incl. `budget_ceiling`), dialog (state matrix, scope-segment presence logic, queue rows from `items`, cost line, cancel confirm, 409 recover, empty-scope), Activity (CTA opens dialog; `generation_batch` row renders), SelectionToolbar re-point (selection actually flows). `pnpm lint:all` + affected `nx test web` + build green. Screenshot-verify vs `flow-f-subtitle-v2/f8-d-v2.png`, `f8-m-v2.png`, `f9-d-v2.png` @390/768/1440 (Sally gate).
10. **Contract acks.** Dev Notes record: confirmed against [@contract-v2] (Story 9R-16 AC #1/#2/#3/#7/#9 — endpoints, preview, budget_ceiling status, SSE payload; re-acked at the post-9R-18 rebase 2026-07-06 — media ids are UUID strings, authored v1 acks superseded). Slice-1 (`ux3-subtitle-v2`) component reuse is a same-epic file dependency, not a stamped contract — re-verify `GenerationProgressV2`/`useGenerationProgress` props as shipped.

## Tasks / Subtasks

- [x] Task 1: Service methods (AC: 3)
  - [x] start/status/cancel/preview + Rule 18 + spec
- [x] Task 2: `useGenerationBatchProgress` (AC: 3)
  - [x] Sibling reducer hook, double-nest unwrap, terminal incl. budget_ceiling + spec
- [x] Task 3: `GenerationBatchDialogV2` (AC: 1, 2, 7, 8)
  - [x] Scope segments + queue rows + cost line + cancel + F9 banner/actions + states + spec + gallery fixtures
  - [x] (Rule 24 ① absorbed) invalidate `libraryKeys.all` + preview query on batch terminal — 9R-16 AC 12 writeback means badges/counts change; without invalidation the library keeps stale 缺字幕 badges after a batch
- [x] Task 4: Activity hub (AC: 4)
  - [x] Launch CTA + `generation_batch` ACTIVE_META/kind union + specs
- [x] Task 5: Library re-point (AC: 5)
  - [x] SelectionToolbar relabel + `selectedIds` flow into dialog + series-excluded note + spec
- [x] Task 6: Series CTA flip (AC: 6, conditional)
  - [x] 9R-10a NOT merged (ready-for-dev at dev time 2026-07-06) → ELSE branch: series CTA in `ManageSubtitleDialogV2` stays capability-disabled; carry-forward documented on the 9R-10a sprint-status entry
- [x] Task 7: Verification (AC: 9)
  - [x] lint:all / full `nx test web` / build / fixture-driven screenshot-verify vs f8-d/f9-d PNGs

**Cross-stack split check:** backend tasks = 0 (9R-16 owns them), frontend tasks = 7 → single story. ✓

## Dev Notes

> ✅ **STALE [@contract-v1→v2] RESOLVED** (2026-07-06, same day, at the post-9R-18 rebase): re-ack completed — `confirmed against [@contract-v2] (Story 9R-16 AC #1/#2/#3/#7/#9, re-verified post-9R-18)` lines recorded in the Dev Agent Record below, verified against the MERGED Go code (`generation_batch_handler.go` `MediaIDs []string`, `generation_batch.go` `MediaID`/`CurrentMediaID string`, PR #148 9cbf4370). All `Number()`/`String()` media-id conversions dropped (library selection ids pass through unconverted; SSE `current_media_id` joins rows directly); every spec/fixture converted to the 9R-18 AC 7 UUID-string fixture convention. Original stale-mark (by 9R-18, 2026-07-06): upstream 9R-16 ACs #1/#2/#3/#7/#9 bumped — media ids are UUID **STRINGS** end-to-end (start body `media_ids: [<string>...]`, 202 `items[].media_id` string, SSE `current_media_id` string; `transcription_*` payloads' `media_id` also string, stamped [@contract-v1] first formalization). The authored-shapes section below has been updated to the v2 shapes.

### 9R-16 contract (authored shapes — re-verified post-9R-18, [@contract-v2])

- `POST /api/v1/subtitles/generation-batch` `{scope:"missing"|"selected", media_ids?:[string uuid]}` → 202 `{batch_id, total_items, items:[{media_id: string, title}]}`; 409 `TRANSCRIPTION_BATCH_RUNNING` (+progress in error body); 503 `TRANSCRIPTION_DISABLED`; 400 `VALIDATION_*`; empty missing-scope → 200 `{total_items:0, items:[]}`.
- ⚠️ Scope=selected capability honor (9R-16 AC 8 ruling, confirmed at 9R-16 CR 2026-07-06): the BE **REJECTS the whole request with 400** (`VALIDATION_INVALID_FORMAT`) if ANY `media_ids` entry is not a movie with a media file — it does NOT filter server-side. FE MUST exclude series ids client-side (AC 5's visible note) BEFORE sending; the 400 is defense-in-depth, not the filtering mechanism.
- `GET .../status` → `{running, progress|null}`; `POST .../cancel` → `{cancelled, running}`; `GET .../preview?scope=missing` → `{total_items}`. Note (fetch-batch parity): after ANY terminal state the status probe returns `{running:false, progress:null}` — terminal snapshots (incl. `budget_ceiling` counts) arrive only via the SSE event, so don't rely on a post-terminal GET to rebuild F9.
- SSE `generation_batch_progress` (double-nested envelope → `parsed.data`): `{batch_id, total_items, current_index, current_media_id (string uuid), current_item, success_count, fail_count, paused_count, status, spent_usd, budget_usd}`, `status` ∈ running|complete|cancelled|error|budget_ceiling.
- Per-item stage detail: join existing `transcription_extracting/progress/translation_progress/transcription_complete/transcription_failed` on `current_media_id` (slice-1's `useGenerationProgress` already handles these — frozen stages 提取音訊→轉錄中→翻譯中→簡轉繁→AI校正→完成). ⚠️ Join caveat (9R-16 CR 2026-07-06): on `cancelled`/`budget_ceiling` the interrupted in-flight item ALSO emits `transcription_failed` (the per-item pipeline reports its own abort; per-item vocabulary is frozen, there is no per-item "paused" stage) — the batch event's `status`/`paused_count` is authoritative for row rendering; treat a `transcription_failed` for `current_media_id` that coincides with a non-`error` terminal batch status as 已暫停/已取消, not 失敗.

### FE anchors (verified 2026-07-05)

- **Dialog template:** `components/subtitle/BatchSubtitleDialog.tsx` — container/panel split, on-open recovery probe (:287-302), 409-recover (:322-331), inline cancel-confirm (:341-348), Escape gating (:305-312). Mirror the ORCHESTRATION; the visuals are new v2 (tokens, per `DownloadCardV2`).
- **Hook template:** `hooks/useSubtitleBatchProgress.ts` — reducer/terminal-close shape. Envelope note (corrected 2026-07-05): its `snakeToCamel(event.data || event)` and slice-1's `parsed.data` are EQUIVALENT unwraps of the same full-`Event` envelope (`sse/handler.go:44` sends the whole struct) — both correct; follow slice-1's explicit `parsed.data` form for consistency, and do NOT "fix" the fetch hook.
- **Multi-select chain (LEGACY SHELL ONLY):** `routes/library.tsx:195-198` selection state, `:592-604` SelectionToolbar render, `:795` dialog render (currently discards selection); `components/library/SelectionToolbar.tsx:54-63` the button. v2 `LibraryBrowseV2` has none → `disc-2026-07-v2-library-multiselect`.
- **Activity:** `components/activity/ActivityHub.tsx:36-39` ACTIVE_META, `:92-95` current/total render, `:236-241` header (CTA slot); `services/activityService.ts:17-29` ActiveJob union; hub polls 15s (`useActivity.ts:31-37`) — the batch dialog's live data comes from SSE, the hub row from the poll (eventual-consistent, fine).
- **Slice-1 reuse (NOT YET BUILT — dependency):** `ManageSubtitleDialogV2.tsx` (series CTA to flip), `GenerationProgressV2.tsx` (stage stepper for the active row; optional cost props stay dormant per-item), `useGenerationProgress.ts` (per-item SSE), `glossaryService.ts`. If slice-1 is still undeveloped when this story starts, STOP — sequence slice 1 first.
- **缺字幕 count:** comes ONLY from 9R-16's preview endpoint — there is NO list-filter/stat API (the dormant `subtitleStatus` search param is unwired server-side; tracked: `disc-2026-06-library-subtitle-status-filter` — do not absorb).
- **ID conversion — NONE ([@contract-v2], superseded 2026-07-06 by 9R-18):** media ids are UUID STRINGS end-to-end; library selection `Set<string>` ids pass into `media_ids` unconverted, and SSE `current_media_id` joins UI rows directly. (The authored v1 guidance — `Number()` on send / `String()` on join — is obsolete; zero such conversions remain in the batch files.)

### Design sources

- F8 `i9Nun1`/`H717g`, F9 `JMqPg` — canvas names authoritative (prose F-numbers renumbered; F4-D-v2 `U8rRtv` is a PER-ITEM failed state owned by slice 1, NOT batch). Screenshots `flow-f-subtitle-v2/f8-d-v2.png`, `f8-m-v2.png`, `f9-d-v2.png`. Exact zh-TW strings in the ACs are the drawn strings — reuse verbatim.
- **Drawn-vs-buildable deltas (pre-cleared for the Sally gate — do NOT block on these):** (a) F8's active row shows a SERIES (怪奇物語 S4E7) but batch v1 is movies-only (9R-16 AC 8) — the built dialog will show movie rows; (b) the drawn stepper has 5 segments (提取音訊/轉錄中/翻譯中/簡轉繁/完成) while the frozen component vocabulary is 6 (+AI校正) — render the component's frozen list. Screenshot comparison verifies layout/tokens/strings, not these two sample-content deltas.
- D4-1 boundary (ux3-2-1): Activity hub = explain-why rows; batch dialog opens OVER the hub (F8/F9 backdrops literally render A1-D-v2). No new destination, no second entry.
- No `.pen` modification → no screenshot regen.

### Project Structure Notes

- New: `components/subtitle/GenerationBatchDialogV2.tsx` (+ spec + fixtures), `hooks/useGenerationBatchProgress.ts` (+ spec), service methods (+ spec). Edits: `ActivityHub.tsx`, `activityService.ts`, `SelectionToolbar.tsx`, `routes/library.tsx`, (conditional) `ManageSubtitleDialogV2.tsx`. No new routes.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - Expected **N/A — no wall-clock-reading components touched**: progress, counts, and cost are all SSE-supplied; no relative-time display is drawn. If one sneaks in: Rule 23 marker + ≥2 clock-pinned fixture states mandatory.
  - Batch-status fixture vocabulary (`running`/`budget_ceiling`/`cancelled`…) mirrors the 9R-16 status enum — treat as frozen once baselined.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: _bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md AC #1/#2/#3/#7/#9 [@contract-v2] — v1→v2 bump by 9R-18 (string media ids), re-acked 2026-07-06]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2.md — slice-1 components/hooks + Discovery Triage batch split]
- [Source: _bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md AC #5 (amended scope segments) + budget-ceiling semantics]
- [Source: ux-design.pen F8 i9Nun1/H717g, F9 JMqPg; _bmad-output/screenshots/flow-f-subtitle-v2/]
- [Source: project-context.md Rules 5/18/21/23/24/26; §8]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — Amelia, 2026-07-06

### Rule 20 contract re-verification (against MERGED 9R-16 code, PR #147 ce15f39c)

- confirmed against [@contract-v2] (Story 9R-16 AC #1, re-verified post-9R-18) — `generation_batch_handler.go`: `POST /api/v1/subtitles/generation-batch` 202 `{batch_id, total_items, items:[{media_id,title}]}`; empty-missing → 200 `{total_items:0, items:[]}`; 409 `TRANSCRIPTION_BATCH_RUNNING` with the progress snapshot riding the error-body `data`; 503 `TRANSCRIPTION_DISABLED`; 400 `VALIDATION_REQUIRED_FIELD` (selected w/o media_ids) / `VALIDATION_INVALID_FORMAT` (missing WITH media_ids — shipped refinement: the story annotation didn't list this reject; FE never sends media_ids for scope=missing, so no impact | non-movie/no-file id via `ErrGenerationSelectionInvalid`).
- confirmed against [@contract-v2] (Story 9R-16 AC #2, re-verified post-9R-18) — `GET .../preview?scope=missing` → `{total_items}`; any other scope → 400 (preview supports missing only).
- confirmed against [@contract-v2] (Story 9R-16 AC #3, re-verified post-9R-18) — `GET .../status` → `{running, progress|null}`; `POST .../cancel` → `{cancelled, running}` (idempotent, `cancelled:false` when nothing runs). Post-terminal probe returns `{running:false, progress:null}` — `finish()` clears `activeBatch`; terminal snapshots reach clients via broadcast ONLY (L3 dead-store removal confirmed in code).
- confirmed against [@contract-v2] (Story 9R-16 AC #7, re-verified post-9R-18) — `budget_ceiling` status shipped as `GenerationBatchStatusBudgetCeiling = "budget_ceiling"`; pre-check ceiling emits `current_index=i`, mid-item ceiling `current_index=i+1`, both with `paused_count = len(items)-i` (paused, NOT failed). ⚠️ Shipped refinement recorded: `current_index` semantics differ between the pre-start-cancel (`i`) and mid-item-cancel (`i+1`) finishes — FE therefore derives the interrupted row from `current_media_id` (and `paused_count` for F9), NEVER from `current_index` arithmetic (see `deriveRowStates`).
- confirmed against [@contract-v2] (Story 9R-16 AC #9, re-verified post-9R-18) — SSE `generation_batch_progress` (hub const `EventGenerationBatchProgress`, `sse/hub.go:41`); payload = exactly the 11 contract keys (`generation_batch.go` `broadcast()` hand-built map): batch_id, total_items, current_index, current_media_id, current_item, success_count, fail_count, paused_count, status, spent_usd, budget_usd; status ∈ running|complete|cancelled|error|budget_ceiling; envelope = FULL `Event` struct on the `data:` line (`sse/handler.go:45` `sendSSEEvent(w, type, event)`) → `parsed.data` unwrap confirmed.
- Slice-1 reuse re-verified as shipped: `GenerationProgressV2` props (`phase/failedPhase/percentage/message/error/costUsedText/costLimitText/onRetry`), `useGenerationProgress` → `{progress, startTracking(mediaId:number), reset}` (per-media filter + terminal self-close), frozen 6-stage vocabulary incl. AI校正.
- Activity side re-verified: `activity_service.go:154` ships `kind:"generation_batch"` with `percent_done/detail/current/total` — matches the AC 4b row render.

### Debug Log References

- `--update-snapshots=missing` visual run aborts at the PRE-EXISTING `components/ui-dialog/default` darwin mismatch before reaching the new fixtures (single mega-test) → used the proven full-update + selective-revert flow; only the 3 new PNGs staged.

### Completion Notes List

- **Envelope**: `useGenerationBatchProgress` uses slice-1's explicit `parsed.data ?? parsed` unwrap; the fetch hook's `event.data || event` left untouched (equivalent — per the corrected Dev Note).
- **Row-state authority (9R-16 CR caveat)**: implemented in exported `deriveRowStates()` — for `budget_ceiling` the last `paused_count` rows render 已暫停 — 下次繼續 regardless of a recorded per-item `transcription_failed`; for `cancelled`/`error` rows from the `current_media_id` row onward render 已取消; per-item failures are only RECORDED while the batch status is `running`. Unit-tested including the exact race (failed event for the interrupted item + terminal batch event).
- **409/recover-attach limitation (documented)**: the status probe carries NO `items[]`, so a recovered dialog renders the overall counter + cost + the in-flight item card only (queue rows need the start-202 response). Filed as ③ `disc-2026-07-generation-batch-status-items`.
- **Legacy fetch dialog**: `BatchSubtitleDialog.tsx` file stays (gallery fixtures `subtitle-batch-subtitle-panel-*` keep it compiling) but is NO LONGER reachable from live UI — `routes/library.tsx` now renders `GenerationBatchDialogV2` with the selection actually flowing (previous wiring discarded it). `SelectionToolbar` relabelled 批次字幕搜尋 → 批次生成字幕; `data-testid="batch-subtitle-btn"` KEPT (specs/E2E key on it; documented per AC 5).
- **Selection classification**: selections can span pages, so `library.tsx` accumulates an id→type map (`selectionTypesRef`) in `handleSelect`/`handleSelectAll`; at CTA click movie ids are `Number()`-converted (non-finite → excluded, defensive) and series/unknown ids become `excludedSeriesCount` for the dialog's visible note (已排除 N 部影集（影集字幕生成即將推出）).
- **Undrawn-state strings (token-only, no drawn source)**: 開始生成 (idle CTA), 已取消 row label + 批次發生錯誤 banner (error terminal), 確定要取消整個批次嗎？已完成的字幕會保留。 (inline cancel confirm), 已排除…部影集 note. All AC-listed strings are the drawn strings verbatim; F9 banner uses the DRAWN warning-tint/warning tokens (verified via Pencil MCP — "success-styled" in AC 2 means non-error semantics; the .pen draws warning-tint).
- **Escape/scrim gating** while running via Radix `onEscapeKeyDown`/`onPointerDownOutside`/`onInteractOutside` preventDefault; the ✕ close remains available (closing stops WATCHING only; recover-on-open re-attaches — ManageSubtitleDialogV2 precedent).
- **Task 6 ELSE branch**: 9R-10a still `ready-for-dev` → series CTA stays capability-disabled; carry-forward noted on the 9R-10a sprint-status entry (flip + route-shape confirmation rides that story).
- **Rule 21**: header `// Design ref: ux-design.pen Screen F8-D-v2 (i9Nun1)`; node ids i9Nun1/H717g/JMqPg verified live via Pencil MCP `batch_get`.
- **Rule 23**: no wall-clock reads anywhere in the new files; all progress/cost/counts SSE-supplied. Fixture states named after the frozen batch-status vocabulary (idle/running/budget_ceiling).
- Gates: full `pnpm nx test web` 225 files / 2450 tests green (+30 new); `pnpm lint:all` 0 errors; `pnpm nx build web` green; prettier clean on all touched files; 3 new `-darwin` baselines (0 `-linux` — CI bootstrap PR handles those); pre-existing `ui-dialog/default` darwin visual failure ignored per dispatch.

### Discovery Triage

Authoring-time discoveries (SM Bob, 2026-07-05, filed in sprint-status.yaml):

- **③ `disc-2026-07-v2-library-multiselect`** — the v2 shell's `LibraryBrowseV2` has NO selection mode / SelectionToolbar / batch-ops at all (the whole multi-select chain lives in legacy `LibraryPage` only). The F8 `已選項目` path is therefore legacy-only until v2 grows multi-select. Bidirectional: entry names this story.
- `subtitleStatus` list-filter BE gap — already tracked (`disc-2026-06-library-subtitle-status-filter`); referenced, not re-filed.

Dev-time discoveries (Amelia, 2026-07-06):

- **① absorbed** — library/preview query invalidation on batch terminal (9R-16 AC 12 writeback makes 缺字幕 badges stale without it). Added as a Task 3 subtask; implemented in the container terminal effect.
- **③ `disc-2026-07-generation-batch-status-items`** — `GET /subtitles/generation-batch/status` returns only the progress snapshot (no `items[]`), so a dialog that ATTACHES to a running batch (409/recover) cannot render the full F8 queue — it falls back to the in-flight item card. BE enhancement: include the enumerated queue in the status/progress payload. Filed in sprint-status.yaml; bidirectional.

### File List

- `apps/web/src/services/subtitleService.ts` — generation-batch types + 4 service methods (M)
- `apps/web/src/services/subtitleService.spec.ts` — +9 specs (M)
- `apps/web/src/hooks/useGenerationBatchProgress.ts` — NEW lazy-SSE batch hook
- `apps/web/src/hooks/useGenerationBatchProgress.spec.ts` — NEW, 11 specs (double-nest unwrap, terminal-close matrix incl. budget_ceiling)
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` — NEW panel + container + `deriveRowStates`
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.spec.tsx` — NEW, 26 specs
- `apps/web/src/components/activity/ActivityHub.tsx` — launch CTA + `generation_batch` ACTIVE_META + counted-kinds right slot + dialog mount (M)
- `apps/web/src/components/activity/ActivityHub.spec.tsx` — +2 specs, dialog stub (M)
- `apps/web/src/services/activityService.ts` — `ActiveJob.kind` union + `generation_batch` (M)
- `apps/web/src/components/library/SelectionToolbar.tsx` — relabel 批次生成字幕 (M)
- `apps/web/src/components/library/SelectionToolbar.spec.tsx` — +1 spec (M)
- `apps/web/src/routes/library.tsx` — selection type map + `handleOpenGenerationBatch` + dialog swap w/ selection flowing (M)
- `apps/web/src/routes/library.spec.tsx` — +1 spec (selection actually flows) + dialog stub (M)
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 3 new fixtures (idle / running / budget_ceiling) (M)
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/*-visual-darwin.png` — 3 NEW baselines
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — status transitions + Task 6 carry-forward on 9R-10a + new ③ entry (M)
- `_bmad-output/implementation-artifacts/ux3-subtitle-v2-batch.md` — this record (M)
- `tests/e2e/batch-subtitle.spec.ts` — CR H1 fix: suite re-pointed from the unmounted fetch dialog to GenerationBatchDialogV2 (M, review commit)

## Senior Developer Review (AI) — 2026-07-06

Reviewer: adversarial CR vs e99c2761 (`git diff main...HEAD`). **Verdict: APPROVED-WITH-FIXES-APPLIED** (fixes uncommitted in working tree; H2/L1 remain open as filed discoveries).

### Findings

| # | Sev | Where | Issue | Status |
|---|-----|-------|-------|--------|
| H1 | HIGH | `tests/e2e/batch-subtitle.spec.ts` | CI-gated e2e suite (test.yml `test-e2e-sharded`, chromium) still drove `BatchSubtitleDialog` via `/library` — this branch unmounted it, so 5/6 tests fail in the PR gate. Dev gates (unit/lint/build) never ran Playwright. | FIXED — suite rewritten against `GenerationBatchDialogV2` (7 wire-level journeys incl. a NEW `media_ids`-on-the-wire selection test). Local run blocked by a PRE-EXISTING env issue (the OLD suite fails identically at the first `enter-selection-btn` click — local Go backend has real data; CI runs clean). CI must confirm. |
| H2 | HIGH | cross-story (BE+FE) | Movie PKs are UUIDs (`uuid.New().String()`), but the whole Route C chain requires numeric ids: BE `toItem()` ParseInt-skips UUID movies while `PreviewMissing` SQL-counts them (preview N>0 → start 0 items → empty-scope), and `library.tsx` `Number(uuid)`=NaN excludes every movie. Feature inoperative on real data. Faithful to 9R-16 [@contract-v1] + slice-1 precedent — not this story's regression. | OPEN — filed `disc-2026-07-movie-id-int64-contract-mismatch` (needs contract ruling). |
| M1 | MED | `GenerationBatchDialogV2.tsx` (panel) | AC 5's "visible note" was gated on `scope==='selected'`, which requires ≥1 movie id — so when EVERY selected item was excluded (all-series selection; and the H2 UUID case) the selection silently vanished with no note. | FIXED — note renders whenever `isIdle && excludedSeriesCount > 0`; + spec. |
| M2 | MED | `GenerationBatchDialogV2.tsx` (fallback card) | 409/recover-attach fallback card used `isRunning ? 'active' : 'stopped'` — a budget-paused in-flight item rendered 已取消 (violates AC 2 paused semantics), complete rendered 已取消 too. | FIXED — status-mapped (running→active / budget_ceiling→paused / complete→done|failed / else stopped); + 2 specs. |
| M3 | MED | `GenerationBatchDialogV2.spec.tsx` | The CR-annotated batch-status-authoritative race was unit-tested ONLY on the `deriveRowStates` helper; the container's failedIds recording gate (records per-item failures only while `running`) had no test — no proof the real component never paints a paused row 失敗. | FIXED — 2 container-level race specs (per-item failed BEFORE and AFTER the terminal batch event; real container + effects + panel). |
| L1 | LOW | `useGenerationBatchProgress.ts` + BE | If the terminal SSE event is lost (`sse/hub.go:151-154` drops on full broadcast channel; 10s reconnect gap) the dialog stays running forever — no re-probe on reconnect, and the post-terminal status probe (`{running:false, progress:null}`) cannot rebuild F9. §8 forbids polling fallback; FE-only fix would misreport the terminal kind. | OPEN — addendum recorded on `disc-2026-07-generation-batch-status-items` (BE: expose last terminal snapshot). |
| L2 | LOW | F9 banner copy | `已完成N部` renders `success+fail` — contract-consistent (N+M=total per 9R-16 paused_count arithmetic) but overstates when failCount>0. | NOTE for Sally/product; no change. |

### Rule 20 ack verification (result: PASS)

All 5 `confirmed against [@contract-v1]` lines in Dev Notes re-verified against shipped Go: AC#1 handler responses/codes incl. the two `VALIDATION_*` splits ✓; AC#2 preview missing-only ✓; AC#3 status/cancel + post-terminal `{running:false, progress:null}` (`finish()` clears `activeBatch`) ✓; AC#7 `budget_ceiling` + `paused_count=len(items)-i` in BOTH pre-check and mid-item finishes — `totalItems-pausedCount === idxOfCurrent` holds, so `deriveRowStates` arithmetic is sound and `current_media_id` always points at a real queue item on every terminal broadcast ✓; AC#9 11-key hand-built payload + full-Event envelope (`sendSSEEvent(w, type, event)`) → `parsed.data` unwrap ✓. Slice-1 reuse + activity `generation_batch` kind ✓. Rule 25: `project-context.md` untouched ✓. Baselines: exactly 3 new + 3 relabel recaptures, 0 `-linux`, label-only delta pixel-verified ✓. Sprint-status dev edits surgical ✓.

### Gate re-runs (post-fix)

- `GenerationBatchDialogV2.spec.tsx` 31/31; full `pnpm nx test web` 2455 green (+5 review specs); `pnpm lint:all` 0 errors; prettier clean on touched files.
- e2e: not runnable in this local env (pre-existing detach-loop, affects old suite equally) — rides the CI `test-e2e-sharded` gate.
