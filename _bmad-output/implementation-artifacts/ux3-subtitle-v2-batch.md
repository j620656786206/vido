# Story ux3-subtitle-v2-batch: Subtitle UI v2 — batch generation surfaces (PH3-M5 slice 2)

Status: ready-for-dev

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
10. **Contract acks.** Dev Notes record: confirmed against [@contract-v1] (Story 9R-16 AC #1/#2/#3/#7/#9 — endpoints, preview, budget_ceiling status, SSE payload). Slice-1 (`ux3-subtitle-v2`) component reuse is a same-epic file dependency, not a stamped contract — re-verify `GenerationProgressV2`/`useGenerationProgress` props as shipped.

## Tasks / Subtasks

- [ ] Task 1: Service methods (AC: 3)
  - [ ] start/status/cancel/preview + Rule 18 + spec
- [ ] Task 2: `useGenerationBatchProgress` (AC: 3)
  - [ ] Sibling reducer hook, double-nest unwrap, terminal incl. budget_ceiling + spec
- [ ] Task 3: `GenerationBatchDialogV2` (AC: 1, 2, 7, 8)
  - [ ] Scope segments + queue rows + cost line + cancel + F9 banner/actions + states + spec + gallery fixtures
- [ ] Task 4: Activity hub (AC: 4)
  - [ ] Launch CTA + `generation_batch` ACTIVE_META/kind union + specs
- [ ] Task 5: Library re-point (AC: 5)
  - [ ] SelectionToolbar relabel + `selectedIds` flow into dialog + series-excluded note + spec
- [ ] Task 6: Series CTA flip (AC: 6, conditional)
  - [ ] If 9R-10a merged: wire + spec; else document carry-forward
- [ ] Task 7: Verification (AC: 9)
  - [ ] lint:all / affected tests / build / browser-verify vs f8-d/f8-m/f9-d PNGs

**Cross-stack split check:** backend tasks = 0 (9R-16 owns them), frontend tasks = 7 → single story. ✓

## Dev Notes

### 9R-16 contract (authored shapes — re-verify at dev time)

- `POST /api/v1/subtitles/generation-batch` `{scope:"missing"|"selected", media_ids?:[int64]}` → 202 `{batch_id, total_items, items:[{media_id,title}]}`; 409 `TRANSCRIPTION_BATCH_RUNNING` (+progress in error body); 503 `TRANSCRIPTION_DISABLED`; 400 `VALIDATION_*`; empty missing-scope → 200 `{total_items:0, items:[]}`.
- ⚠️ Scope=selected capability honor (9R-16 AC 8 ruling, confirmed at 9R-16 CR 2026-07-06): the BE **REJECTS the whole request with 400** (`VALIDATION_INVALID_FORMAT`) if ANY `media_ids` entry is not a movie with a media file — it does NOT filter server-side. FE MUST exclude series ids client-side (AC 5's visible note) BEFORE sending; the 400 is defense-in-depth, not the filtering mechanism.
- `GET .../status` → `{running, progress|null}`; `POST .../cancel` → `{cancelled, running}`; `GET .../preview?scope=missing` → `{total_items}`. Note (fetch-batch parity): after ANY terminal state the status probe returns `{running:false, progress:null}` — terminal snapshots (incl. `budget_ceiling` counts) arrive only via the SSE event, so don't rely on a post-terminal GET to rebuild F9.
- SSE `generation_batch_progress` (double-nested envelope → `parsed.data`): `{batch_id, total_items, current_index, current_media_id, current_item, success_count, fail_count, paused_count, status, spent_usd, budget_usd}`, `status` ∈ running|complete|cancelled|error|budget_ceiling.
- Per-item stage detail: join existing `transcription_extracting/progress/translation_progress/transcription_complete/transcription_failed` on `current_media_id` (slice-1's `useGenerationProgress` already handles these — frozen stages 提取音訊→轉錄中→翻譯中→簡轉繁→AI校正→完成). ⚠️ Join caveat (9R-16 CR 2026-07-06): on `cancelled`/`budget_ceiling` the interrupted in-flight item ALSO emits `transcription_failed` (the per-item pipeline reports its own abort; per-item vocabulary is frozen, there is no per-item "paused" stage) — the batch event's `status`/`paused_count` is authoritative for row rendering; treat a `transcription_failed` for `current_media_id` that coincides with a non-`error` terminal batch status as 已暫停/已取消, not 失敗.

### FE anchors (verified 2026-07-05)

- **Dialog template:** `components/subtitle/BatchSubtitleDialog.tsx` — container/panel split, on-open recovery probe (:287-302), 409-recover (:322-331), inline cancel-confirm (:341-348), Escape gating (:305-312). Mirror the ORCHESTRATION; the visuals are new v2 (tokens, per `DownloadCardV2`).
- **Hook template:** `hooks/useSubtitleBatchProgress.ts` — reducer/terminal-close shape. Envelope note (corrected 2026-07-05): its `snakeToCamel(event.data || event)` and slice-1's `parsed.data` are EQUIVALENT unwraps of the same full-`Event` envelope (`sse/handler.go:44` sends the whole struct) — both correct; follow slice-1's explicit `parsed.data` form for consistency, and do NOT "fix" the fetch hook.
- **Multi-select chain (LEGACY SHELL ONLY):** `routes/library.tsx:195-198` selection state, `:592-604` SelectionToolbar render, `:795` dialog render (currently discards selection); `components/library/SelectionToolbar.tsx:54-63` the button. v2 `LibraryBrowseV2` has none → `disc-2026-07-v2-library-multiselect`.
- **Activity:** `components/activity/ActivityHub.tsx:36-39` ACTIVE_META, `:92-95` current/total render, `:236-241` header (CTA slot); `services/activityService.ts:17-29` ActiveJob union; hub polls 15s (`useActivity.ts:31-37`) — the batch dialog's live data comes from SSE, the hub row from the poll (eventual-consistent, fine).
- **Slice-1 reuse (NOT YET BUILT — dependency):** `ManageSubtitleDialogV2.tsx` (series CTA to flip), `GenerationProgressV2.tsx` (stage stepper for the active row; optional cost props stay dormant per-item), `useGenerationProgress.ts` (per-item SSE), `glossaryService.ts`. If slice-1 is still undeveloped when this story starts, STOP — sequence slice 1 first.
- **缺字幕 count:** comes ONLY from 9R-16's preview endpoint — there is NO list-filter/stat API (the dormant `subtitleStatus` search param is unwired server-side; tracked: `disc-2026-06-library-subtitle-status-filter` — do not absorb).
- **ID conversion:** the 9R-16 contract's `media_ids`/`media_id` are int64, but library selection is `Set<string>` (`library.tsx:195`, string `Movie.ID`) — `Number()` each id when building `scope=selected` payloads and `String()` when joining SSE `current_media_id` back to UI rows.

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

- [Source: _bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md AC #1/#2/#3/#7/#9 [@contract-v1]]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2.md — slice-1 components/hooks + Discovery Triage batch split]
- [Source: _bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md AC #5 (amended scope segments) + budget-ceiling semantics]
- [Source: ux-design.pen F8 i9Nun1/H717g, F9 JMqPg; _bmad-output/screenshots/flow-f-subtitle-v2/]
- [Source: project-context.md Rules 5/18/21/23/24/26; §8]

## Dev Agent Record

### Agent Model Used

(fill at dev time)

### Debug Log References

### Completion Notes List

### Discovery Triage

Authoring-time discoveries (SM Bob, 2026-07-05, filed in sprint-status.yaml):

- **③ `disc-2026-07-v2-library-multiselect`** — the v2 shell's `LibraryBrowseV2` has NO selection mode / SelectionToolbar / batch-ops at all (the whole multi-select chain lives in legacy `LibraryPage` only). The F8 `已選項目` path is therefore legacy-only until v2 grows multi-select. Bidirectional: entry names this story.
- `subtitleStatus` list-filter BE gap — already tracked (`disc-2026-06-library-subtitle-status-filter`); referenced, not re-filed.
- (Dev: add in-flight discoveries per Rule 24 before marking done.)

### File List
