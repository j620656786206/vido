# Story ux3-ai-2: AI generation workspace — frontend (F11 spec → build, PH3-G1 close-out)

Status: review

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

- [x] Task 0 (STOP GATE): ux3-ai-1 = done (merged #152) ✓; node ids + IA/timestamp rulings inherited from its Dev Agent Record; conditionals ALL backlog → baseline (attach-degraded, SSE-opportunistic single-job, cost SSE-only; 9R-10a no effect).
- [x] Task 1 (AC 5): `hooks/useGenerationJobsFeed.ts` — unfiltered `transcription_*` + `generation_batch_progress`, per-media `singleJobs` (retire-on-terminal) + `feed` (cap 200, monotonic seq, NO timestamps); +8 spec cases (MockEventSource).
- [x] Task 2 (AC 2, 3): `components/subtitle/generationWorkspace.ts` — pure `deriveWorkspaceMode` (loading/idle/single/running/attach/terminals) + `modeShowsFeed`; +11 spec cases.
- [x] Task 3 (AC 2, 3, 4, 6, 7): `GenerationWorkspaceV2.tsx` — prop-driven presentational (full state matrix, F9-verbatim banner, event-log pane, idle+preview+launcher, a11y) + `GenerationWorkspace` container (probe→attach effect, hooks compose); +7 spec cases.
- [x] Task 4 (AC 1): `routes/activity.tsx` `validateSearch` `view?:'generation'` (Rule 26 literal) + `ActivityHub` conditional workspace render; SSE gated on active+visible (container). +1 view-gating spec.
- [x] Task 5 (AC 6): `generation_batch` active row → `?view=generation` Link (watcher); header CTA stays launcher; dialog caches start `items[]` (`generationBatchItemsKey`) so the workspace shows the full queue. Detail cross-link DEFERRED (ux3-ai-1 ruling). +1 link spec.
- [x] Task 6 (AC 8): Rule 21 `// Design ref: … F11-D-v2 (l8FsB)` header; `generationWorkspace.ts` utility-exempt header; 3 gallery fixtures (running/budget_ceiling/idle) + 3 NEW `-darwin` baselines (selective-staged, 0 regen noise). `-linux` via CI.
- [x] Task 7 (AC 9, 10): gates green — `nx test web` 229/2501, `nx test api` full, `lint:all` 0-err, build, prettier. UX verify: running baseline ≡ F11-D (l8FsB), budget_ceiling ≡ F12 F9-verbatim. Rule 20 acks + Discovery Triage below.

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

claude-opus-4-8 (Amelia — BMM Dev Agent, dev-story workflow, 2026-07-07)

### Debug Log References

- `pnpm nx test web` — 229 files / 2501 tests green (+26 net new across the 4 new spec files + hub/dialog spec extensions). Cleanup: "No test processes found".
- `pnpm nx test api` — full Go suite green (Epic 9 Retro AI-1 regression gate; FE-only story, gate runs both).
- `pnpm lint:all` 0 errors (124 pre-existing warnings, none in touched files); `pnpm nx build web` green; `prettier --check` clean on all touched files.
- Visual: `playwright --project=visual --update-snapshots` → 3 NEW `generation-workspace-v2/{running,budget_ceiling,idle}` `-darwin` baselines staged; 26 re-render-noise diffs `git checkout`-reverted (non-deterministic regen). `-linux` via CI bootstrap.

### A11y Pre-Flight

🎭 A11y Pre-Flight: PASS (2 new components — GenerationWorkspaceV2, ActivityHub edit; 0 jsx-a11y warnings introduced). Recurring classes: overall progress = `role="progressbar"` + `aria-valuenow/min/max` + `aria-label` (DownloadCardV2 precedent); status transitions = `aria-live="polite"` on the event-log `<ol>` + `role="status"` on the budget banner; skeletons `motion-reduce:animate-none`; 44px touch floor on all buttons. No modal/img/combobox in scope.

### Completion Notes List

- **🎨 Reshape inherited from ux3-ai-1 (capability-honest):** the right pane is a session-scoped EVENT LOG (`useGenerationJobsFeed.feed`), not a transcript — `transcription_*` SSE carries only `{phase, message, percentage}`. NO timestamps rendered (SSE has none; feed keys use a monotonic `seq`) → the workspace reads no wall-clock (Rule 23-clean). Footer states 「僅狀態事件，不含逐字內容」.
- **Capability walls honored:** no pause/resume/per-item-retry/per-item-cancel anywhere; `全部取消` is batch-wide only; movies-only; cost from SSE `spentUsd`/`budgetUsd` only (no HTTP cost surface — 9R-17 backlog); budget non-editable; no history. `deriveRowStates` (reused from the dialog) is the batch-status-authoritative row join.
- **Baseline (no conditional upgrades — all disc entries backlog):** attach-degraded drawn honestly (status probe has no `items[]` → in-flight card + skeleton + honest note, `disc-2026-07-generation-batch-status-items`); single-job rows are SSE-opportunistic only (no load-time active_jobs join — `disc-2026-07-transcription-active-jobs`); cost SSE-only (9R-17); series unaffected (batch movies-only, 9R-10a).
- **Architecture — launcher/watcher (ux3-ai-1 IA ruling):** the F8 dialog stays the LAUNCHER; the workspace is the immersive WATCHER hosted at `/activity?view=generation`. The dialog now caches its start-202 `items[]` under `generationBatchItemsKey` so the workspace renders the full queue for a batch started this session (a cold attach falls back to degraded — the honest baseline). The hub's `generation_batch` active row LINKS to the workspace; the header CTA still opens the dialog. D4-1: one row, one link, no second competing surface.
- **Deviation — detail cross-link DEFERRED (AC 6c):** per ux3-ai-1's ruling, the `前往生成工作區` link inside the F3 dialog was NOT added (would edit a shipped Epic-6 frame's behavior; deferred to a future increment). Recorded, not silently dropped.
- **🔗 AC Drift: N/A** — pure consumer of shipped 9R-16 v2 + 9R-18 contracts; no prior AC's observable behavior changed. (Dialog `items[]` cache is additive — the launcher persisting its own result, no contract change.)
- **📎 Contract Stamps: FOUND** — see Rule 20 acks below (consumer; stamps nothing).
- **Pre-existing failures:** none; full web+api suites green before and after (Epic 9c Retro AI-2 gate satisfied).
- **🔎 Code Review (adversarial, `/code-review high`):** 2 CONFIRMED fixed, rest accepted.
  - CR-1 (FIXED): the container read cached `items[]` via `queryClient.getQueryData` — a NON-reactive read. If the dialog caches `items[]` while the workspace is already mounted (idle → launch-CTA opens the dialog over the workspace → start), the workspace would stay in attach-degraded and never show the full queue. Switched to a `useQuery({ enabled:false })` subscription so a `setQueryData` write re-renders the workspace. `GenerationWorkspaceV2.tsx`.
  - CR-2 (FIXED): the dialog's 409-attach path cleared local `items` but not the query cache — the workspace could join a PRIOR batch's stale `items[]` against the newly-attached batch's progress (wrong row states). Added `removeQueries(generationBatchItemsKey)` on 409-attach → attach-degraded (honest). `GenerationBatchDialogV2.tsx`.
  - Accepted (non-issues): 3 concurrent `EventSource` to `/events` (established per-hook clone convention, §8 no shared utility — each filters its events; under the browser cap); `BudgetBanner` `pausedCount || …` fallback (in `budget_ceiling`, `pausedCount > 0` always — defensive only). Full web suite 2501 green post-fix.

### Rule 20 contract acks (consumer — stamps nothing)

- confirmed against `[@contract-v2]` (Story 9R-16 AC #1/#2/#3/#7/#9) — generation-batch endpoints/status/cancel/preview + 11-key `generation_batch_progress` SSE + `budget_ceiling` semantics; consumed via `subtitleService` + `useGenerationBatchProgress` as shipped (re-verified against the merged shapes this session, unchanged since ux3-subtitle-v2-batch).
- confirmed against `[@contract-v1]` (Story 9R-18) — media ids are UUID strings end-to-end incl. `transcription_*` `media_id`; the feed hook + workspace join on string ids, zero Number()/String() coercion.
- confirmed against `[@contract-v1]` (Story 13-3a §8 shape) — the visibility+view SSE double-gate pattern reused from 13-3b (DiscoverBrowseV2) for lazy §8 compliance.

### Discovery Triage

- **Did this story discover any work outside its current scope?** `N/A — no NEW out-of-scope work discovered.` The epic's gaps were filed by ux3-ai-1 (`disc-2026-07-transcription-active-jobs`) and earlier stories (`disc-2026-07-generation-batch-status-items` — priority raised by ux3-ai-1's addendum; `9R-17`, `9R-10a`, `disc-2026-07-v2-library-multiselect`). This story consumed them as conditionals (all still `backlog` at Task 0 → baseline built); each is bidirectionally linked in sprint-status. No new entry needed.

### File List

New:
- `apps/web/src/hooks/useGenerationJobsFeed.ts` (+ `.spec.ts`) — unfiltered SSE feed + single-jobs hook (AC 4/5).
- `apps/web/src/components/subtitle/generationWorkspace.ts` (+ `.spec.ts`) — pure `deriveWorkspaceMode`/`modeShowsFeed` (AC 2/3).
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx` (+ `.spec.tsx`) — presentational workspace + container (AC 2/3/4/6/7).
- `tests/visual/…/components/generation-workspace-v2/{running,budget_ceiling,idle}/default-visual-darwin.png` — 3 baselines (AC 8; `-linux` via CI).

Modified:
- `apps/web/src/routes/activity.tsx` — `validateSearch` `view?:'generation'` (Rule 26) (AC 1).
- `apps/web/src/components/activity/ActivityHub.tsx` (+ `.spec.tsx`) — `?view=generation` hosts the workspace; `generation_batch` row → workspace Link (AC 1/6).
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` — `generationBatchItemsKey` export + start-`items[]` cache writeback (AC 5).
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 3 workspace fixtures (AC 8).
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — ux3-ai-2 → in-progress → review.
- `_bmad-output/implementation-artifacts/ux3-ai-2-workspace-frontend.md` — this record.

Untouched by design (scope walls): the F8/F9 dialog launch flow (only additive items[] cache), `navModel.ts` (no nav slot), `useGenerationBatchProgress`/`useGenerationProgress`/`deriveRowStates` (reused as-is), all backend.
