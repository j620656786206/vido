# Story ux3-2-3 — Activity hub frontend (`/activity` route)

**Epic:** ux3-activity-hub (UX Redesign Phase 3) · **Status:** done (FE) · **Type:** frontend
**Consumes:** ux3-2-2 `GET /api/v1/activity` · **Design:** ux3-2-1 `flow-k-activity-v2`

## What

The FE half of the Activity hub (ADR **D4-1**) — the net-new `/activity` route built from
the `flow-k-activity-v2` design, consuming the fail-soft aggregate. Completes the
cross-stack build: design (#85) → BE (#86) → FE (this).

### Route + data
- `routes/activity.tsx` — `createFileRoute('/activity')` + `staticData:{shell:'v2'}`
  (full-bleed under AppShellV2; flag stays read-once in `__root`, F4). Net-new → **no
  shell-version branch** (Activity has no legacy version; the hub renders directly).
- `services/activityService.ts` + `hooks/useActivity.ts` — mirror `statusSummaryService` /
  `useStatusSummary`: fetch + `snakeToCamel` (Rule 18), poll while the tab is visible.

### Components (the design's four states, N4)
- `ActivityRow` (Component/ActivityRow-v2) — explain-why row: icon-chip + title + why-detail
  + right slot (Mono percent/count, status word, or accent CTA) over an optional progress bar.
- `ActivityHub` — sections 進行中 (live scan / batch-subtitle, progress + count chip) →
  待處理 (→ `前往處理`, deep-links the library unmatched filter) → 下載 (summary row →
  `開啟下載頁`, links the deep page — D4-1 HYBRID) → 活動記錄 (完成/失敗 tinted log). Maps the
  backend's copy-free `kind`/`result` enums to icon + title + tone here (copy lives on the
  client, i18n).
- `ActivityStates` — skeleton (A4), calm empty + next-step CTA (A5), per-section fail-soft
  banner + 重試 (A6 — a degraded section degrades alone; the page only shows a single error
  when the request itself fails, F3).

### Nav — 活動 goes live
- `navModel.ts`: new `ACTIVITY` destination; added to the desktop sidebar + rail; the mobile
  bottom-4 becomes **首頁 · 媒體庫 · 活動 · 下載** (per the A2-M-v2 design — `探索` moves into
  the More sheet). `系統` stays deferred (route not built yet).
- `AppSidebar.tsx`: 活動 rendered at the top of the 任務 group.

## Gates

- `nx build web` (regenerates `routeTree.gen.ts` with `/activity`) · `nx test web`
  (2184 passed — incl. ActivityHub 4-state + fail-soft + nav specs, relativeTime util) ·
  `nx lint web` (0 errors) · prettier clean. `nx typecheck web` is pre-existing-broken on
  main (not the gate).

## Notes / deferred (Rule 24)

- The recent feed shows **parse** events only (BE v1); scan/subtitle/AI completion isn't
  persisted yet (activity-log table = follow-up). AI active-jobs are absent until job
  tracking lands (→ ux3-ai-subtitle epic).
- ⚠️ A browser-verify of `/activity` @ 390/768/1440 under real content is recommended
  (not a CI gate).
