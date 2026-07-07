# Story ux3-ai-2: AI generation workspace — frontend (F11 spec → build, PH3-G1 close-out)

Status: ready-for-dev

> ## 🛑 STOP GATE (read before ANY implementation)
>
> **Depends on: `ux3-ai-1-workspace-design` DONE + Sally-reviewed.** This story was pair-authored 2026-07-07 (13-7a/b / ux3-subtitle-v2-batch precedent) while the design story was still `ready-for-dev`. If the design story is not `done` when you start: **STOP — sequence the design first.**
> On start, read the design story's **Dev Agent Record** and inherit (do NOT invent):
> 1. **Node ids** — F11-D-v2 (revised `l8FsB`) / F11-M-v2 (new) / any net-new `Component/*-v2` ids → Rule 21 headers + gallery `penNode` fields.
> 2. **IA rulings (design AC 6):** (a) dialog↔workspace relationship (authored assumption: dialog = LAUNCHER, workspace = immersive WATCHER, hub row links to workspace); (b) hosting mechanic (`/activity?view=generation` search-param vs child route — both pre-scoped below); (c) detail-dialog cross-link yes/no.
> 3. **Rule 23 timestamp ruling (design AC 4):** whether the live-activity feed renders per-row times. If yes → `Clock-mocked` marker + ≥2 clock-pinned fixture states are MANDATORY (Rule 23); if no → zero wall-clock reads.
> 4. **Drawn zh-TW strings** — reuse verbatim from the completed frames (F8/F9 vocabulary continuity is already binding, see AC 2).
> If the completed design restructured beyond the design-story ACs (e.g. not hosted in Activity, or new capability-violating controls), **bounce to SM for story revision** — do not improvise.

**Epic:** `ux3-ai-subtitle` (Epic 7 — PH3-G1, Epic 9 P1-020/021) · **per-flow-recipe step 2 (FE)** · closes the epic's remaining surface
**Blocked-by:** `ux3-ai-1-workspace-design` · **FE-only** (all BE gaps are ③-filed, none absorbed)

## Story

As a Vido user who just launched batch generation for the 38 titles missing 繁中 subtitles,
I want a full-page generation workspace under 活動 — queue, live pipeline activity, cost, and budget state — that survives navigation and can be reopened mid-run,
so that a long-running AI batch has a home I can leave and return to, instead of a modal that loses everything on close.

## Acceptance Criteria

1. **Hosted workspace route (v2 shell only).** New `GenerationWorkspaceV2` full-page surface hosted inside the 活動 destination per the design ruling — sidebar 活動 stays active, breadcrumb 活動 → 生成字幕. **NO new nav slot** (`navModel.ts` untouched; destination-map: G ≠ destination). Pre-scoped mechanics (implement whichever the design record ruled):
   - **(preferred, requests precedent)** search-param: `routes/activity.tsx` gains `validateSearch` with `view?: 'generation'` (literal-checked, Rule 26 — mirror `discover.tsx:18-30` `view?: 'requests'` + coercion-helper style) and `ActivityHub` conditionally renders the workspace in place of the hub body (chrome persists — `DiscoverBrowseV2.tsx:238-270` shape);
   - **(alternative)** child route `routes/activity.generation.tsx` with `staticData: { shell: 'v2' }` (`activity.tsx:8-11` pattern).
   Deep-linkable either way; leaving the view/page closes all EventSources (§8 — lazy, visibility+view gated, `DiscoverBrowseV2.tsx:113-121` double-gate precedent from 13-3b).

2. **Running state — one vocabulary with F8/F9, one row-state authority.** Queue rows (title + state 完成/失敗/轉錄中/排隊中; active row renders the FROZEN 6-stage stepper via `GenerationProgressV2` XkGvG), overall Mono `N / M` + progress, cost line 本次用量 Mono `$X.XX` / 上限 Mono `$5.00` (SSE `spent_usd`/`budget_usd` ONLY — no other cost source exists, 9R-17 backlog), 即時更新（SSE）indicator, **全部取消** (inline confirm → `cancelGenerationBatch()`). Row states MUST go through the exported `deriveRowStates` (`GenerationBatchDialogV2.tsx:64`) — the batch-status-authoritative join (a `transcription_failed` for `current_media_id` coinciding with a non-`error` terminal renders 已暫停/已取消, never 失敗). Reuse `useGenerationBatchProgress` + `subtitleService` batch methods as-is; all numerics `font-mono tabular-nums`, number+CJK-unit split (TY-3).

3. **Terminals + attach-degraded.** `budget_ceiling` renders F9-verbatim (warning-tint banner 已達本次預算上限（$X）— 已完成N部，剩餘M部下次繼續; rows 已暫停 — 下次繼續; actions 關閉/離開 + 下次繼續 = NEW `scope=missing` batch); `complete`/`cancelled`/`error` per design frames. **Attach case (workspace opened MID-batch):** on-mount `getGenerationBatchStatus()` probe → seed `startTracking(seed)`; the probe has NO `items[]`, so render the degraded layout the design drew (overall counters + cost + in-flight card + 「佇列明細自本頁開啟起顯示」) — do NOT fake a queue. **Conditional upgrade:** IF `disc-2026-07-generation-batch-status-items` has landed by dev time (check sprint-status + the shipped Go), consume `items[]`/terminal-snapshot for full-queue attach and record the ack; ELSE ship degraded and note it.

4. **Live-activity feed (session-scoped).** New client-side accumulator: every `generation_batch_progress` transition + every `transcription_*` event appends a feed row (stage label + item title + message) from the moment the page opens; annotate 自開啟本頁起累積 per design. Feed capped (e.g. last 200 rows, drop-oldest) — unbounded arrays leak on long batches. Timestamp rendering strictly per the design's Rule 23 ruling (STOP-gate item 3): if times render, `// Clock-mocked:` marker + ≥2 clock-pinned fixture states (`clockTime` field, `withFixedClock` — Rule 23 AC 1d/4); otherwise ZERO `Date.now()`/`new Date()` in new components (ESLint `local/time-dependent-fixture-stability` gates).

5. **Single-job opportunistic rows.** New lazy SSE hook `useGenerationJobsFeed` (clone convention — sibling of `useGenerationProgress`, §8 lazy, double-nest `parsed.data` unwrap, `snakeToCamel`, 10s reconnect, `mountedRef`): listens to the five `transcription_*` events UNFILTERED, maintains a per-`media_id` map of in-flight single jobs (a job appears on its next event; terminal events retire it after render). Detail-triggered 生成字幕 jobs thus appear in the workspace opportunistically. HONEST limitation per design annotation: a single job started elsewhere is invisible until its next SSE event (`active_jobs` has no transcription kind — `disc-2026-07-transcription-active-jobs`). **Conditional:** IF that disc entry has landed by dev time, add the load-time join (probe active_jobs for the new kind, seed the map, wire `ACTIVE_META` per the design's vocabulary annotation) and record the ack; ELSE ship SSE-opportunistic only.

6. **Idle/empty + entries.** No batch running + no single jobs → calm 目前沒有進行中的生成 + Mono 缺字幕 preview count (`previewGenerationBatch()`) + launch affordance per design ruling (authored assumption: CTA opens the existing `GenerationBatchDialogV2` — the dialog stays the launcher; do NOT rebuild scope selection in the page). Entry links per design: Activity hub `generation_batch` row → workspace; hub 批次生成字幕 CTA behavior per ruling; detail cross-link only if ruled yes. D4-1: the workspace is the hub-linked immersive surface — no second competing Activity entry, no duplicate hub summary inside the workspace.

7. **Four-state + a11y.** Surface states: idle-empty / running / terminals / attach-degraded / fail-soft error (activity or preview fetch down → inline 重試, page never hard-fails) / loading skeleton (`motion-reduce:animate-none`). A11y: overall progress = `role="progressbar"` + `aria-valuenow/min/max` (`DownloadCardV2.tsx:99-109` precedent); status transitions `aria-live="polite"`; stepper keeps the `<ol aria-label>` + `data-state` idiom (XkGvG as-is); feed region labelled; 44px touch floor.

8. **Rule 21 / 23 / 26 / fixtures.** New component files carry headers with the DESIGN RECORD's node ids (`// Design ref: ux-design.pen Screen F11-D-v2 (l8FsB)` + net-new component ids — verify live via Pencil MCP; ESLint gates). Gallery fixtures for each net-new Library-registered component + a seeded workspace running-state fixture (`seedQueries` precedent) with states named after the frozen batch-status vocabulary; NEW `-darwin` baselines only (`-linux` via CI bootstrap; full-regen noise reverted). Search params (if mechanic a): literal string check per Rule 26.

9. **Tests + gates.** Colocated specs: feed hook (unfiltered accumulation, per-media map, retire-on-terminal, cap, double-nest unwrap asserted), workspace container (state matrix incl. attach-degraded + F9 terminal + race: per-item failed vs terminal batch event — container-level, M3 precedent), route/view gating (start/stop SSE on view+visibility, 13-3b spec shape), hub-link wiring. Full gates: `pnpm nx test web` + `pnpm nx test api` (Epic 9 Retro AI-1 full regression), `pnpm lint:all`, `pnpm nx build web`, prettier. UX screenshot verification vs the design's exported `flow-f-subtitle-v2/f11-*.png` @390/768/1440 (mandatory — Sally gate). E2E: extend `tests/e2e/batch-subtitle.spec.ts` with a workspace journey ONLY if the suite's env allows (pre-existing local detach-loop — CI is the venue; note either way).

10. **Contract acks (Rule 20).** Record in Dev Agent Record: confirmed against `[@contract-v2]` (Story 9R-16 AC #1/#2/#3/#7/#9 — batch endpoints/preview/status/cancel/budget_ceiling/SSE 11-key payload, as re-acked by ux3-subtitle-v2-batch post-9R-18); confirmed against `[@contract-v1]` (Story 9R-18 — media ids UUID strings end-to-end incl. `transcription_*` `media_id`). This story stamps NO new contracts. The design record + slice-1/2 components are same-epic file dependencies, not stamped contracts — re-verify props as shipped (`GenerationProgressV2` props, `useGenerationBatchProgress` state at `useGenerationBatchProgress.ts:27-56`, `deriveRowStates` signature).

## Tasks / Subtasks

- [ ] Task 0 (STOP GATE): verify `ux3-ai-1-workspace-design` = done; extract node ids + IA/timestamp rulings from its Dev Agent Record; re-check the two conditional disc entries + 9R-17/9R-10a status in sprint-status.
- [ ] Task 1 (AC 5): `hooks/useGenerationJobsFeed.ts` — unfiltered `transcription_*` listener, per-media map + feed accumulator + cap; spec (MockEventSource harness, slice-1 pattern).
- [ ] Task 2 (AC 2, 3): workspace state composition — batch probe/seed/attach (`useGenerationBatchProgress` + `getGenerationBatchStatus`), `deriveRowStates` join, conditional items[] upgrade; spec.
- [ ] Task 3 (AC 2, 3, 4, 6, 7): `components/subtitle/GenerationWorkspaceV2.tsx` (+ any design-record net-new row/feed components) — full state matrix, F9-verbatim terminal, feed pane, idle+preview+launcher, a11y; specs.
- [ ] Task 4 (AC 1): hosting mechanic per ruling (activity route `validateSearch` + hub conditional render, or child route); SSE view+visibility double gate; route/view specs.
- [ ] Task 5 (AC 6): Activity hub links — `generation_batch` row → workspace, CTA behavior per ruling, (conditional) detail cross-link; `ActivityHub` spec updates.
- [ ] Task 6 (AC 8): Rule 21 headers (Pencil MCP verify), gallery fixtures + `-darwin` baselines (selective staging).
- [ ] Task 7 (AC 9, 10): full gates + UX screenshot verify @390/768/1440 vs `f11-*.png`; Rule 20 ack lines; Discovery Triage updates.

**Cross-stack split check:** backend tasks = 0 (③-filed, none absorbed), frontend tasks = 8 → single story. ✓

## Dev Notes

### Inherit-from-design manifest (fill at Task 0 — the four STOP-gate items)

| # | Item | Where it lands in this story |
|---|---|---|
| 1 | Node ids (F11-D/F11-M revised/new + net-new components) | Rule 21 headers, gallery `penNode`, AC 8 |
| 2 | IA rulings a/b/c | AC 1 mechanic, AC 6 entries, Task 4/5 |
| 3 | Rule 23 timestamp ruling | AC 4, fixture plan, Time-dependent section |
| 4 | Drawn zh-TW strings + state-frame set | AC 2/3/6 copy, fixture state names |

### Backend surface (code-verified 2026-07-07 — full table in `ux3-ai-1-workspace-design.md` Dev Notes; condensed here)

- Batch: `POST/GET/POST/GET /api/v1/subtitles/generation-batch{,/status,/cancel,/preview?scope=missing}` (`generation_batch_handler.go:37-204`). Status probe post-terminal = `{running:false, progress:null}`; NO `items[]` on probe. 409 `TRANSCRIPTION_BATCH_RUNNING` carries progress in error body.
- SSE (single `GET /api/v1/events`, double-nested envelope → `parsed.data`): `generation_batch_progress` 11 keys, `status ∈ running|complete|cancelled|error|budget_ceiling` (`generation_batch.go:395-416`); `transcription_{extracting,progress,complete,failed}` + `translation_progress` with `phase`/`percentage`/`media_id` (uuid string) — **NO timestamps in any payload** (`transcription_service.go:315-728`).
- ABSENT (never draw/call): pause/resume; per-item retry/cancel; transcription cancel; series (9R-10a not merged — batch movies-only regardless, 9R-16 AC 8); batch history/terminal re-probe; cost HTTP endpoint (SSE-only); budget editing (env `AI_RUN_BUDGET_USD`); transcription kind in `active_jobs` (`activity_service.go:129-162`).

### FE anchors (verified 2026-07-07)

- **Reuse as-is:** `useGenerationBatchProgress` (`hooks/useGenerationBatchProgress.ts:27-56` state incl. `pausedCount/spentUsd/budgetUsd/currentMediaId`, `startTracking(seed?)` :173, terminal-close :94), `subtitleService` batch methods (:296-352), `deriveRowStates` + F8/F9 strings (`GenerationBatchDialogV2.tsx:64-125,258,399-407`), `GenerationProgressV2` (frozen `GENERATION_STAGES` export), `useGenerationProgress` per-media join if the active row needs it (`hooks/useGenerationProgress.ts:52,144-149` — filters by uuid-string mediaId).
- **Clone-convention template for the new hook:** slice-1 `useGenerationProgress.ts` (envelope unwrap :179, five events :199-219, reconnect, mountedRef). No shared SSE utility — cloning IS the convention (a third clone may motivate extraction; not here).
- **Route/view gating:** `discover.tsx:18-30` (`view?: 'requests'` literal + Rule 26 helpers), `DiscoverBrowseV2.tsx:106-121` (showRequests + isVisible double gate, 13-3b), `activity.tsx:8-11` (net-new v2 route, no legacy branch — the workspace likewise has NO legacy-shell variant).
- **Hub anchors:** `ActivityHub.tsx:38-46` (`ACTIVE_META`/`COUNTED_KINDS`), :253-280 (CTA + dialog mount), `activityService.ts:19` (open kind union).
- **Dialog relationship:** `GenerationBatchDialogV2` stays intact as the launcher (authored assumption) — this story adds LINKS, it does not fork or absorb the dialog's launch flow. If the design ruled workspace-absorbs-watching, the dialog STILL isn't deleted (library legacy entry uses it, `routes/library.tsx:836`).
- **Visual baselines:** `--update-snapshots=missing` aborts at the pre-existing `ui-dialog/default` darwin mismatch — use full-update + selective-revert (slice-1/2 proven flow); stage ONLY new PNGs.

### Carry-forward conditionals (check at Task 0, wire only if landed)

| Entry | If landed by dev time |
|---|---|
| `disc-2026-07-generation-batch-status-items` | consume `items[]` + terminal snapshot → full-queue attach + reconnect re-probe (kills the L1 stuck-running case); ack the shipped shape |
| `disc-2026-07-transcription-active-jobs` | load-time single-job join via active_jobs + `ACTIVE_META` wiring per design vocabulary |
| `9R-17-ai-usage-endpoint` | cost slot beyond SSE (per design's dormant-slot annotation) — otherwise SSE-only |
| `9R-10a-series-episode-trigger` | NO effect here (batch stays movies-only per 9R-16 AC 8; series CTA flip rides 9R-10a itself) |

### Scope walls

- NO backend changes (all gaps ③-filed). NO new nav slot / `navModel.ts` edit. NO pause/per-item-retry/history affordances. NO v2 library multi-select (`disc-2026-07-v2-library-multiselect`). NO legacy-shell workspace variant. NO rebuild of launch/scope-selection inside the page (dialog is the launcher, per ruling). NO `.pen` edits (design story owns the canvas; if a genuine design defect is found, bounce to Sally — do not redraw). NO SSE-utility extraction.

### Time-dependent visual coverage

- **Conditional on the design's Rule 23 ruling (STOP-gate item 3):**
  - Ruling = no timestamps → `N/A — no wall-clock-reading components touched`; assert zero `Date.now()`/`new Date()` in new files (grep + ESLint).
  - Ruling = client receive-time rendered → the feed component gets `// Clock-mocked:` marker + ≥2 clock-pinned fixture states (`clockTime` field, e.g. `feed-fresh` / `feed-aged`), baselines captured per Rule 23 AC 1d/4.
- Fixture vocabulary: batch statuses (`running`/`budget_ceiling`/…) + frozen stage names remain the state names — do not rename (baseline↔fixture mapping).

### References

- [Source: _bmad-output/implementation-artifacts/ux3-ai-1-workspace-design.md — capability table, design ACs, discovery entries (the gating spec)]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2-batch.md — 9R-16 [@contract-v2] acks, deriveRowStates authority, envelope caveats, baseline flow]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2.md — slice-1 hooks/components, frozen vocabulary, Rule 23 handoff]
- [Source: apps/api/internal/… — generation_batch_handler.go / generation_batch.go / transcription_service.go / activity_service.go / sse/hub.go (audit line refs above)]
- [Source: apps/web/src/… — hooks/useGenerationBatchProgress.ts, hooks/useGenerationProgress.ts, components/subtitle/GenerationBatchDialogV2.tsx, components/activity/ActivityHub.tsx, routes/{activity,discover}.tsx]
- [Source: project-context.md Rules 5/18/20/21/23/24/26 + §8; CLAUDE.md UX-verification workflow]

## Change Log

| Date       | Change |
| ---------- | ------ |
| 2026-07-07 | Story authored (SM Bob, create-story yolo — pair-authored with ux3-ai-1 per 13-7a/b precedent, STOP gate on design completion). Composes shipped slice-1/2 hooks/components into the F11 workspace: hosted-in-Activity route (two mechanics pre-scoped, ruling inherited), deriveRowStates-authoritative queue, F9-verbatim terminals, attach-degraded honesty, session-scoped feed (Rule 23 conditional), opportunistic single-job rows via new unfiltered `useGenerationJobsFeed`. 4 carry-forward conditionals tabled. FE-only, 8 tasks, no split. Status → ready-for-dev (gated). |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - If **YES**: classify each per Rule 24 (①/②/③) with tracked entry IDs; prose-only mentions are banned.
- Authoring-time note: no NEW discoveries this run — the epic's gaps were filed by ux3-ai-1 (`disc-2026-07-transcription-active-jobs`) and earlier stories (`disc-2026-07-generation-batch-status-items`, `9R-17`, `9R-10a`, `disc-2026-07-v2-library-multiselect`); this story consumes them as conditionals, re-verify at Task 0.

### File List

(Expected: `hooks/useGenerationJobsFeed.ts` (+spec), `components/subtitle/GenerationWorkspaceV2.tsx` (+spec, + any design-record net-new components), `routes/activity.tsx` OR `routes/activity.generation.tsx`, `components/activity/ActivityHub.tsx` (+spec), gallery fixtures + `-darwin` baselines, sprint-status.yaml, this file.)
