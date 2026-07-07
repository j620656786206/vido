# Story ux3-ai-1: AI generation workspace design — F11 exploration → validated spec (PH3-G1, Epic 7 step 1)

Status: review

**Epic:** `ux3-ai-subtitle` (Epic 7 — PH3-G1, Epic 9 P1-020/021) · **per-flow-recipe step 1 (design)**
**Owner:** ux-designer (Sally / Pencil MCP) · **Type:** design (delivers `.pen` + screenshots, not code) · **Gates:** `ux3-ai-2-workspace-frontend` (filed backlog, blocked on this)
**Design seed:** `F11-D-v2` (node `l8FsB`, `f11-d-v2.png`) — inline-agent's unprompted full-page generation-workspace exploration, **ruled KEPT as Epic 7 reference by Alexyu (party-mode P4, 2026-07-05)**. Canvas annotation `rhhQ0` marks it 「非 story 規格」 — this story's job is to turn it INTO the spec.

## Story

As the design system,
I want the F11 generation-workspace exploration validated and expanded into a buildable v2 spec — a full-page immersive AI-jobs surface hosted inside the 活動 destination, showing the generation queue, live pipeline activity, and budget state, drawn strictly to what the backend can actually do,
so that dev builds Epic 7's remaining surface (the workspace) against a capability-honest spec instead of an unvalidated exploration frame, completing PH3-G1 (Epic 9 AI subtitles finally have a full UI).

## Context — what Epic 7 still owes after PH3-M5

PH3-M5 (slices `ux3-subtitle-v2` + `ux3-subtitle-v2-batch`, both DONE 2026-07-06) already delivered most of the epic skeleton's "AI-jobs in Activity + a detail trigger": F1 管理字幕 dialog, F3 GenerationProgress (frozen 6 stages), F6/F7 glossary, F8/F9 batch dialog, Activity CTA + `generation_batch` row. **What remains — this story's scope:**

1. **The immersive workspace (F11).** Today all generation visibility is trapped in modals — closing the dialog loses everything (`GenerationBatchDialogV2.tsx:28-29` comment). F11 gives the 38-item batch a full-page home: queue + live activity + budget, survivable across navigation.
2. **Single-job (per-item 生成字幕) visibility.** `active_jobs` covers `scan`/`subtitle_batch`/`generation_batch` only (`activity_service.go:129-162`) — a single detail-triggered transcription is INVISIBLE in Activity. BE gap → filed ③ `disc-2026-07-transcription-active-jobs`; the workspace designs the slot capability-annotated.

**F11 is a starting point, not a spec:** one desktop frame, no mobile variant, no N4 states, drawn before the 9R-16/9R-18 contracts existed. Sally may keep, redraw, or restructure it — the ACs below bind the OUTCOME, not the exploration's layout.

## Acceptance Criteria

1. **Workspace frames land (revise `F11-D-v2` l8FsB in place + NEW `F11-M-v2`).** Full-page v2 surface, **hosted inside the 活動 destination** — sidebar 活動 active, breadcrumb 活動 → 生成字幕. NOT a new nav slot (`navModel.ts:7` "earn a slot only when the route exists" + destination-map crosswalk: "F 字幕 / G AI 字幕 — NOT top-level destinations… surface inside Detail + Activity", `03-phase3-destination-epic-map.md:67,88-89`; hosting precedent = 想要清單 under `/discover?view=requests`, nav-ADR:630). Merged-block convention (desktop row above, mobile below, captions above frames). DL-v2 throughout: token-only color, Noto CJK, **Mono ALL numerics** (counts, %, `$X.XX`, timestamps — number+CJK-unit split per Rule TY-3), 44px touch floor.

2. **Capability-honor pass (Rule 24) — the drawn surface must match the audited backend exactly (table in Dev Notes).** Binding rules: (a) **NO Pause/Resume control anywhere** — no pause endpoint exists for anything (audit §3); (b) cancel = **全部取消 batch-wide only** (`POST /subtitles/generation-batch/cancel`) — no per-item cancel, no partial cancel, no single-transcription cancel; (c) **no per-item retry** in batch (failures are counted, loop continues); (d) **movies-only** rows (9R-16 AC 8; series = 9R-10a not merged); (e) 下次繼續 = new `scope=missing` batch (resume-for-free ruling — completed items self-exclude via 9R-16 AC 12 writeback); (f) cost figures come ONLY from batch SSE `spent_usd`/`budget_usd` — no standalone cost/usage surface exists (9R-17 backlog; per-run token/call counters are logged-only, never on the wire); budget ceiling is env-only (`AI_RUN_BUDGET_USD`, default $5) — do NOT draw a budget-edit affordance; (g) no batch history — terminal snapshots arrive via SSE only, post-terminal `GET status` = `{running:false, progress:null}`.

3. **N4 four states + the two workspace-specific states, all drawn (per DL-v2 §7 — design ALL of them or it doesn't ship):**
   - **Idle/empty** (no generation running): calm 目前沒有進行中的生成 + launch affordance + Mono 缺字幕 preview count (`GET …/preview?scope=missing`);
   - **Running**: queue rows (done/active/queued — active row carries the FROZEN 6-stage stepper 提取音訊→轉錄中→翻譯中→簡轉繁→AI校正→完成, reuse `Component/GenerationProgress-v2` XkGvG), overall Mono `N / M` + progress, cost line, 即時更新（SSE）indicator, 全部取消;
   - **Terminals**: `budget_ceiling` (F9 JMqPg semantics verbatim — warning-tint banner, 已暫停 — 下次繼續 rows, 關閉 + 下次繼續 actions, success-not-error), `complete`, `cancelled`, `error`;
   - **Attach-degraded** (page opened MID-batch): the status probe has NO `items[]` (`disc-2026-07-generation-batch-status-items`) — the workspace can render only overall counters + cost + the in-flight item; DRAW this state honestly (e.g. 佇列明細自本頁開啟起顯示) rather than pretending the full queue exists;
   - **Fail-soft error** for the surface's own data fetch (activity/preview down → inline 重試, page never hard-fails).

4. **Live-activity column (the F11 exploration's 即時日誌) — session-scoped semantics, Rule 23-safe.** The log accumulates from SSE (`transcription_*` + `generation_batch_progress` events) from the moment the page opens — **no history/log endpoint exists**; annotate 「日誌自開啟本頁起累積」 on the frame. ⚠️ **Rule 23 handoff warning (mandatory annotation):** SSE payloads carry NO timestamps — any drawn per-row time would be client receive-time (`new Date()` at ingest → Rule 23 `Clock-mocked` marker + ≥2 clock-pinned fixture states in the FE story). Prefer designing WITHOUT per-row wall-clock timestamps (sequence/stage labels suffice); if timestamps stay, the annotation must say they are client-received times and cite Rule 23.

5. **Single-job visibility slot.** Design how a detail-triggered single 生成字幕 job appears in the workspace: opportunistically joinable from live SSE (`transcription_*` events carry `media_id` — a workspace listening unfiltered sees any in-flight job's NEXT event), but **on page load a single job started elsewhere is invisible until its next event** and `active_jobs` has no transcription kind (BE gap, filed ③ `disc-2026-07-transcription-active-jobs`). Draw the single-job row as the same row idiom as batch items; annotate the discovery gate on the frame (「單部任務的載入時可見性 gated on disc-2026-07-transcription-active-jobs」). Also annotate the intended Activity-hub `ACTIVE_META` vocabulary for the future kind (label + icon choice) so the FE story doesn't invent it.

6. **IA/entry decisions resolved and annotated on canvas:** (a) Activity hub `generation_batch` row + 批次生成字幕 CTA → do they LINK to the workspace, and does `GenerationBatchDialogV2` remain the launch/watch modal or does the workspace absorb watching? Resolve the dialog↔workspace relationship explicitly (D4-1 boundary: no two competing surfaces for the same job — recommend: dialog stays the LAUNCHER, workspace is the immersive WATCHER, hub row links to workspace); (b) exact hosting mechanic note for FE (`/activity?view=generation` search-param per the requests precedent, or a child route — either is ADR-compliant; if search-param, Rule 26 string-coercion note); (c) detail-dialog cross-link (F3 progress → 前往生成工作區?) decided yes/no.

7. **Component discipline (Rule 21/22).** REUSE registered components — `Component/GenerationProgress-v2` (XkGvG) for the active row stepper, `Component/ActivityRow-v2` (fF8nX) idiom for any hub-side rows, DL-v2 §2.5 status→token map (no bespoke palette). Any NET-NEW component (e.g. workspace queue row, log-line row) gets its own `Component/*-v2` node **registered in the Component Library frame `sJzat`** (ref instance + Noto 12 `$text-muted` caption). All frame/component node ids recorded in this story file at completion for Rule 21 headers.

8. **Export + handoff.** `SCREENS` dict in `scripts/export-pen-screenshots.py` updated (revised `l8FsB` → `f11-d-v2`, new mobile node → `f11-m-v2`, flow-f-subtitle-v2 folder); `python3 scripts/export-pen-screenshots.py` run; ONLY genuinely-changed PNGs staged (regen is non-deterministic — CLAUDE.md workflow); `.pen` + screenshots committed together. Design-review pass vs these ACs recorded in Dev Agent Record (Sally MCP self-review; party-mode spec review optional at Alexyu's call — P4 already ruled adoption, so a second full panel is NOT required unless the design deviates structurally from F11).

## Tasks / Subtasks

- [x] Task 1 (AC 1, 2): Read the live canvas — `get_editor_state` + `batch_get` on `l8FsB` + annotation `rhhQ0` + F8/F9 (i9Nun1/JMqPg) for vocabulary/token continuity. Audit every drawn control in F11 against the Dev Notes capability table; list keep/cut/change decisions. **DONE** (Sally MCP review — key catch: F11 seed's right pane was a live 逐字稿 stream, which has NO SSE backing; reshaped to event log).
- [x] Task 2 (AC 1, 3): Revise `F11-D-v2` in place (hosted-in-Activity chrome, capability-honored controls) + draw `F11-M-v2`; draw the state set (idle / running / budget_ceiling / attach-degraded). **DONE** — `l8FsB` (running), `PXB0z` (F11-M), `iH98f` (F12 budget_ceiling), `F7ohe` (F13 idle+attach+failsoft); terminal complete/cancelled/error annotated as token-swap variants on F12 (note `NtMLG`).
- [x] Task 3 (AC 4, 5): Live-activity column semantics + Rule 23 timestamp ruling; single-job row + BE-gap annotations. **DONE** — event log (`DUvwI`, renamed `event-log-pane`) is session-scoped, NO timestamps (Rule 23-clean), footer note `僅狀態事件，不含逐字內容`; single-task amber note `DP53I` cites `disc-2026-07-transcription-active-jobs`.
- [x] Task 4 (AC 6): IA annotations. **DONE** — breadcrumb `活動 ＞ 生成字幕` (`O6OQNE`); dialog=launcher ruling annotated on F13 idle panel (`U72BV`); capability boundary note `c4FIoB`. Detail cross-link (AC 6c) DEFERRED — see Completion Notes.
- [x] Task 5 (AC 7): Component registration pass. **DONE** — net-new `Component/GenQueueRow-v2` (`aw4Qr`, no built-in per-row action) registered in `sJzat` progress-v2 row (`g6eVs` cell `BTZCm`); reuses `XkGvG` (active-row stepper) + `YDPhc`/`otvKh` buttons.
- [x] Task 6 (AC 8): SCREENS dict updated; Cmd+S saved; `export-pen-screenshots.py` run → exactly 5 genuinely-changed PNGs (`f11-d`/`f11-m`/`f12-d`/`f13-d-v2` + `component-library`), zero re-render noise; committed on `feat/ux3-ai-1-workspace-design`; status → review. Visual spot-check of all 4 workspace frames PASS (no overflow/collapse; alignment/tokens/type confirmed on the rendered PNGs).

**Cross-stack split check:** backend tasks = 0, frontend tasks = 0 (design-only) → single story. ✓

## Dev Notes

### Backend capability table (code-verified 2026-07-07 — the design MUST NOT contradict this)

**Endpoints (all live, `main.go:680-726`):**

| Surface | Route | Notes |
|---|---|---|
| Batch start | `POST /api/v1/subtitles/generation-batch` `{scope: missing\|selected, media_ids?: [uuid-string]}` | 202 `{batch_id, total_items, items:[{media_id,title}]}`; empty-missing → 200 `{total_items:0, items:[]}`; 409 `TRANSCRIPTION_BATCH_RUNNING` (progress rides error body); 503 `TRANSCRIPTION_DISABLED` (`generation_batch_handler.go:69-131`) |
| Batch status | `GET …/status` | `{running, progress\|null}`; **post-terminal = `{running:false, progress:null}`** — no history (`:148-153`) |
| Batch cancel | `POST …/cancel` | `{cancelled, running}`, idempotent, all-or-nothing (`:163-176`) |
| Preview | `GET …/preview?scope=missing` | `{total_items}` — missing only (`:189-204`) |
| Single generation | `POST /api/v1/movies/:id/transcribe?translate=true` | 202 `{job_id, message}`; movies-only; 409 `TRANSCRIPTION_IN_PROGRESS`; **no cancel route** (`transcription_handler.go:44-117`) |
| Activity | `GET /api/v1/activity` | `active_jobs.kind` ∈ `scan\|subtitle_batch\|generation_batch` — **no single-transcription kind** (`activity_service.go:129-162`); `recent` = parse events only (`:107-113`) |

**SSE (single stream `GET /api/v1/events`, double-nested envelope → `parsed.data`):** `generation_batch_progress` = 11 keys `{batch_id, total_items, current_index, current_media_id (uuid string), current_item, success_count, fail_count, paused_count, status, spent_usd, budget_usd}`, `status` ∈ `running|complete|cancelled|error|budget_ceiling` (`generation_batch.go:395-416`); per-item `transcription_extracting|transcription_progress|translation_progress|transcription_complete|transcription_failed` with `phase`/`percentage`/`media_id` — **no timestamps in any payload** (`transcription_service.go:315-728`). Join caveat (9R-16 CR): on `cancelled`/`budget_ceiling` the interrupted item ALSO emits `transcription_failed` — batch status is authoritative (`deriveRowStates` precedent, `GenerationBatchDialogV2.tsx:64-78`).

**ABSENT capabilities (never draw as live):** pause/resume (nothing, anywhere); per-item retry/cancel/reorder; series generation (9R-10a ready-for-dev, not merged); batch history / terminal-snapshot re-probe (`disc-2026-07-generation-batch-status-items` + its CR addendum); `items[]` on the status probe (same entry); cost/usage HTTP endpoint (9R-17 backlog — SSE `spent_usd`/`budget_usd` is the ONLY cost wire); budget editing (env-only `AI_RUN_BUDGET_USD`, `config.go:67,127`); standalone AI-correction endpoint (correction = in-pipeline stage, AI校正 stage in the frozen stepper / `correcting` stage in Epic 8 fetch events); transcription kind in `active_jobs` (filed ③, AC 5).

### FE anchors (what the workspace will be built on — design accordingly)

- Shipped v2 pieces to visually extend, not fork: `GenerationBatchDialogV2.tsx` (F8/F9 tokens/strings — the workspace speaks the SAME vocabulary: 已完成 N/M, 本次用量 $X.XX / 上限 $5.00, 已暫停 — 下次繼續, 全部取消), `GenerationProgressV2.tsx` (frozen 6-stage stepper, `GENERATION_STAGES` exported), `ActivityHub.tsx:38-46` (`ACTIVE_META`/`COUNTED_KINDS`), `ActivityRow.tsx` (fF8nX).
- Route pattern for the FE story: `staticData: { shell: 'v2' }` net-new v2 surface (`routes/activity.tsx:8-11` precedent); hosted-view precedent `/discover?view=requests` (`DiscoverBrowseV2.tsx:106-110`).
- SSE hooks all exist (`useGenerationBatchProgress`, `useGenerationProgress` — lazy §8); the workspace FE story composes them, likely with an unfiltered transcription listener variant for AC 5.
- A11y notes for the design: stepper = `<ol aria-label>` + `data-state` idiom (not role=progressbar); linear bars = `role="progressbar"` + `aria-valuenow` (DownloadCardV2/RequestRow precedent); status transitions `aria-live="polite"`; skeletons `motion-reduce:animate-none`.

### Design sources

- Seed frame: `F11-D-v2` (`l8FsB`) + exploration annotation `rhhQ0`, flow-f-subtitle-v2 block (x≈31590 y≈-5921); screenshot `_bmad-output/screenshots/flow-f-subtitle-v2/f11-d-v2.png`.
- Party-mode P4 ruling (2026-07-05, `9R-UX-subtitle-v2-design.md` Completion Notes): F11 KEPT + archived as Epic 7 reference — adoption is settled; THIS story validates and specs it.
- Vocabulary/token continuity: F8-D-v2 `i9Nun1`, F9-D-v2 `JMqPg` (drawn strings are canonical — reuse verbatim); DL-v2 `ux-redesign/01-design-language-v2.md` (§2.5 status tokens, §3.1 TY-3, §7 four states).
- IA law: `ux-redesign/01-nav-ia-decision-adr.md` (D4-1) + `03-phase3-destination-epic-map.md:58,67,88-89` (G = 活動 (AI jobs) + Detail; not a destination).
- Old `flow-g-ai-subtitle` (g1–g3) = superseded/dead (fetch-era, pre-DL-v2) — zero reuse (annotation jRbiH).
- ⚠️ `ux-design.pen` WILL change → the CLAUDE.md screenshot workflow applies in full (export script, selective staging, same-commit rule).

### Scope walls

- NO code (design story — `.pen` + screenshots + this file + sprint-status only). NO new nav destination. NO pause/retry/history/budget-edit affordances (capability). NO series rows (9R-10a). NO redrawing F1–F10 (shipped; `chore-pen-subtitle-v2-design-sync` owns pending sync items). NO second Activity entry competing with the hub row + CTA (D4-1).

### Time-dependent visual coverage

- **N/A — design story, no component code.** But AC 4 REQUIRES the Rule 23 timestamp ruling to be annotated on the canvas so the FE story inherits it (the log column is the epic's one wall-clock trap: SSE payloads carry no timestamps; client receive-time display ⇒ `Clock-mocked` marker + ≥2 clock-pinned fixture states downstream).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 7 skeleton + 2026-06-16 Route C revision + PH3-G1 (lines 80-84, 246-252)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md §1/§3 — G not a destination]
- [Source: _bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md — F11 deviation + party-mode P4 ruling]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2.md + ux3-subtitle-v2-batch.md — shipped slices, frozen vocabularies, deriveRowStates authority]
- [Source: apps/api/internal/handlers/{generation_batch_handler,transcription_handler,activity_handler}.go + services/{generation_batch,transcription_service,activity_service}.go + sse/hub.go — capability audit]
- [Source: apps/web/src/components/{subtitle,activity}/* + hooks/useGeneration*.ts + components/shell/navModel.ts — FE anchors]
- [Source: project-context.md Rules 21/22/23/24/26 + §8]

## Change Log

| Date       | Change |
| ---------- | ------ |
| 2026-07-07 | Story authored (SM Bob, create-story yolo — ultimate-context: 2 explorer scouts BE+FE, capability table code-verified). Epic 7 step 1 per per-flow recipe: F11 exploration (party-mode-P4-kept) → validated workspace spec, hosted-in-Activity per destination-map. Capability walls baked (no pause / no per-item retry / no history / SSE-only cost / movies-only). Filed: ③ disc-2026-07-transcription-active-jobs (BE), ux3-ai-2-workspace-frontend (backlog, blocked-by this); addendum on disc-2026-07-generation-batch-status-items (workspace raises attach-case priority). Epic ux3-ai-subtitle → in-progress. Status → ready-for-dev. |

## Dev Agent Record

### Agent Model Used

Draw: Pencil In-App AI agent (yolo, from `ux3-ai-1-workspace-design-prompt.md`). Review + fixes: Sally (ux-designer) via Pencil MCP — Claude Opus 4.8.

### Debug Log References

- Sally MCP structural review 2026-07-07: read `l8FsB`/`DUvwI`/`Y53L5`/`whlu3`/`aw4Qr`/`O6OQNE` (F11-D), `uqZYr`/`zn1DI` (F12), `o9EHBu`/`MpQyc`/`D3YIE0`/`j69jh` (F13), `g6eVs`/`ISilG` (sJzat), `c4FIoB` (capability note). Verdict: near-perfect — 1 LOW hygiene fix applied, 1 IA item deferred.
- ⚠️ `get_screenshot` unavailable this session (Pencil.app live renderer dropped mid-review; `get_editor_state` failed but `filePath`-scoped `batch_get`/`batch_design` worked). Visual-fidelity pass (alignment/overflow/contrast) done structurally via node-tree reads (tokens/fonts/strings/sizes all verified); a screenshot spot-check rides the export step once saved.

### Node ID manifest (for Rule 21 headers in `ux3-ai-2` FE story)

- **Frames:** `l8FsB` F11-D-v2 生成工作區—執行中 (revised in place) · `PXB0z` F11-M-v2 (手機) · `iH98f` F12-D-v2 預算上限終態 · `F7ohe` F13-D-v2 閒置／中途接入／失敗.
- **Net-new component:** `Component/GenQueueRow-v2` = `aw4Qr` (registered `sJzat` progress-v2 row `g6eVs`, cell `BTZCm`).
- **Reused components:** `XkGvG` GenerationProgress-v2 (active-row stepper) · `fF8nX` ActivityRow-v2 (row idiom) · `BDeUS` HomeSidebar-v2 (活動 active) · `otvKh` ButtonPrimary (下次繼續 / 批次生成字幕) · `YDPhc` ButtonSecondary (全部取消 / 關閉 / 重試) · `S86VM` MobileTabItem (F11-M tab bar).
- **Key sub-nodes:** event log pane `DUvwI` (F11-D) / `LTW74` (F12) — both renamed `transcript-pane` → `event-log-pane`; capability note `c4FIoB`; single-task note `DP53I`; F12 terminal-variant note `NtMLG`; F11 exploration note `rhhQ0` (preserved).

### Completion Notes List

- **🎨 Reshape (the load-bearing capability catch):** F11 seed's right column drew a live 逐字稿 (transcript) stream — but `transcription_*` SSE carries only `{phase, message, percentage}`, NEVER transcript text. Drawing it would have shipped a non-existent capability. Reshaped to a **session-scoped event log** (stage transitions / item completions / cost ticks / queued), NO timestamps (SSE has none → keeps the FE story Rule 23-clean), with the honest footer note `僅狀態事件，不含逐字內容`.
- **Capability-honor: PASS.** No pause/resume, no per-item cancel/retry, cancel = batch-wide (`全部取消`) only, movies-only rows, cost from SSE `spent_usd`/`budget_usd` only, budget non-editable (env), no history/re-probe, attach-degraded drawn honestly (no fake queue). All eight walls listed in amber note `c4FIoB`; per-row action absent at the component level (`aw4Qr`).
- **IA rulings expressed:** hosted-in-Activity (breadcrumb 活動＞生成字幕); dialog = launcher / workspace = watcher (idle CTA opens F8 dialog — annotated `U72BV`, workspace never rebuilds scope selection); hosting mechanic = `/activity?view=generation` (handed to FE story). budget_ceiling = F9-verbatim banner, success-styled (`$warning-tint`, not error), footer 關閉 + 下次繼續.
- **Deviation — detail cross-link DEFERRED (AC 6c):** the prompt asked for a `前往生成工作區` link inside the F3 生成進度 dialog, but F3-D-v2 (`JbXai`) is a shipped Epic-6 frame the same prompt forbids touching (prompt self-contradiction). Ruling: do NOT edit the shipped F3 frame in this design story; the FE story (`ux3-ai-2`) may add the pop-out link in code (design-approved) if desired. No canvas change.
- **Fix applied (LOW, Sally):** event-log pane frames `DUvwI` (F11-D) + `LTW74` (F12) were still named `transcript-pane` (seed residue; content correct). Renamed to `event-log-pane` via `batch_design` (name-only, zero visual/baseline impact). Mobile pane was already correctly named `eventlog-section`.
- **N4 + workspace states drawn:** running (F11-D), budget_ceiling + complete/cancelled/error variants (F12), idle-empty + attach-degraded + fail-soft (F13), mobile (F11-M). Preview count `缺繁中字幕：[38 Mono] 部` (TY-3), overall `已完成 [12] / [38] 部`, cost `[$0.42] ／上限 [$5.00]` — all Mono numerics, number+CJK-unit split.
- **Untouched (verified):** F1–F10, F8-D `i9Nun1`, F9-D `JMqPg`, old F*/G*, `rhhQ0` — byte-intact; agent only added new frames + revised `l8FsB`.
- **⏳ Delivery gate:** SCREENS dict updated in `scripts/export-pen-screenshots.py`. Awaiting manual **Cmd+S** in Pencil.app (MCP/inline edits are in-memory until saved) → export → stage only genuinely-changed PNGs → commit `feat(ux3-ai-1): AI generation workspace design — F11 validated spec (.pen flow-f-subtitle-v2)` → status → review.

### Discovery Triage

Story-authoring-time discoveries (SM Bob, 2026-07-07 — filed in sprint-status.yaml same day):

- **③ backlog-with-carry-forward-link — `disc-2026-07-transcription-active-jobs`** (BE): `GET /api/v1/activity` `active_jobs` has no kind for single-movie transcription jobs (`activity_service.go:129-162` — scan/subtitle_batch/generation_batch only; the in-progress map `transcription_service.go:94` is not exposed), and `recent` carries no AI-generation terminal events (`:107-113`). A detail-triggered 生成字幕 is invisible to Activity AND to a freshly-opened workspace (SSE-opportunistic only). Needed for complete AI-jobs visibility (this design's AC 5 slot + the FE story's load-time join). Non-blocking: the workspace ships SSE-opportunistic without it.
- **Addendum recorded on `disc-2026-07-generation-batch-status-items`** (existing entry, not re-filed): the workspace makes the attach-degraded case FIRST-CLASS (a deep-linkable page attaches mid-batch far more often than a modal) — `items[]` + terminal-snapshot on the status probe upgrade the workspace from in-flight-card-only to full queue on load. Priority raised accordingly.
- **FE follow-up filed — `ux3-ai-2-workspace-frontend`** (backlog, blocked-by: this story): builds the workspace per the validated spec (route/hosted view, SSE composition, ACTIVE_META wiring within capability, gallery fixtures + baselines).

- **Did this story discover any work outside its current scope?** YES — triaged above (1 new ③ BE entry, 1 addendum on an existing entry, 1 FE follow-up story entry).

### File List

- `ux-design.pen` — revised `l8FsB` (F11-D-v2 workspace); new `PXB0z` (F11-M), `iH98f` (F12), `F7ohe` (F13); new component `aw4Qr` (GenQueueRow-v2) + `sJzat` registration cell; capability note `c4FIoB`; pane renames (`DUvwI`/`LTW74`). ⏳ **needs Cmd+S to persist.**
- `scripts/export-pen-screenshots.py` — SCREENS += `PXB0z`/`iH98f`/`F7ohe` (f11-m/f12-d/f13-d-v2). ✅ done.
- `_bmad-output/implementation-artifacts/ux3-ai-1-workspace-design-prompt.md` — the inline-agent prompt (Sally-authored). ✅ done.
- `_bmad-output/implementation-artifacts/ux3-ai-1-workspace-design.md` — this record. ✅ done.
- ⏳ **pending export+commit** (after Cmd+S): `_bmad-output/screenshots/flow-f-subtitle-v2/{f11-d,f11-m,f12-d,f13-d}-v2.png` (only genuinely-changed) + `sprint-status.yaml` (→ review).
