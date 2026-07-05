# Story 13.3b: Request Status Tracking — Frontend Live 想要清單 (SSE upgrade)

Status: ready-for-dev

**Epic:** Epic 13 — Request System · **FR:** P3-003 (G-3) · **Artery #3 (FE half)**
**Depends on: 13-3a merged** (request_progress SSE) **+ 13-1b merged** (the static 想要清單 view this story upgrades). GATE-B (13-0's capability-honor note: FE consumption gated on the 13-3/13-4 BE) is SATISFIED once those land.
**Split:** FE half of 13-3. Per the 13-1b SCOPE WALL ruling: 13-1b shipped the static list; **this story owns SSE/live status/progress % — and nothing else is missing from the view.**

## Story

As a user watching my 想要清單,
I want rows to update live — 搜尋中 flips to 下載中 with a moving %, then 已入庫 — without refreshing,
so that the request pipeline feels alive and trustworthy.

## Acceptance Criteria

1. **`useRequestProgress` hook — clone `useDownloadProgress.ts` end-to-end (§8 lazy, LINE-FOR-LINE template).** New `hooks/useRequestProgress.ts` mirrors `useDownloadProgress.ts` (122 lines, scouted verbatim): NO connect on mount; `startTracking()`/`stopTracking()` with the `readyState === 2` idempotence guard; `es.addEventListener('request_progress', …)`; parse `JSON.parse(e.data)` then `snakeToCamel<MediaRequest[]>(event.data ?? event)` (wire wraps the whole Event `{id,type,data}`); `Array.isArray` guard; fixed `SSE_RECONNECT_MS = 10000` reconnect via the **latest-ref-in-effect** pattern (`connectRef` — the retro-ux3-4 lint fix, lines 59/83/90-92 of the template); `mountedRef` guards on every handler; full cleanup on unmount. `requestService.getSSEUrl()` added returning `${API_BASE_URL}/events` (mirror `downloadService.ts:177-179`).

2. **[@contract-v1-consumer] Cache patching — `applyRequestSnapshot`.** Exported helper mirroring `applyDownloadSnapshot` (`useDownloadProgress.ts:30-50`): target `queryClient.getQueryCache().findAll({ queryKey: [...requestKeys.all, 'list'] })`; identity key = request `id`. Divergence from the downloads template, recorded here: the requests list cache is a BARE array (13-1b's `listRequests()` — no pagination envelope), and the SSE payload is a FULL snapshot of live+just-transitioned rows, so the merge is: map snapshot by `id`; for cached rows present in the snapshot → replace with snapshot row; cached rows ABSENT from the snapshot are STALE-TERMINAL rows → keep as-is (they're `completed`/`failed` history the snapshot no longer carries); snapshot rows not yet in cache (created in another tab) → append. `progress` (camelCase, 0–1) rides into the row for rendering. Never `invalidateQueries` on an SSE frame (no refetch storm — setQueryData only, downloads convention).

3. **Wiring — visibility-gated, view-scoped.** In the 13-1b requests view (Discover-hosted, `?view=requests`): `const { startTracking, stopTracking } = useRequestProgress()` + `usePageVisibility()` (the `useDownloads.ts:31-40` singleton), with the effect gating on BOTH `isVisible` AND the requests view being the active view — exactly the `DownloadsBrowseV2.tsx:109-114` shape plus the view condition. Leaving the view or hiding the tab closes the EventSource (no idle connection — §8; Playwright `networkidle` stays safe).

4. **Live rendering per design (13-0 L1).** `RequestRow` (13-1b component) rows now: status token flips live through the DL-v2 §2.5 map (already wired for all 5 in 13-1b — capability-honor pays off here, zero new token work); `downloading` rows render Mono progress `%` (`font-mono text-xs tabular-nums`, `role="progressbar"` + `aria-valuenow` — `DownloadCardV2.tsx:99-109` pattern) with Rule TY-3 number/CJK-unit splits; `failed` rows surface `error_message`; transitions announce politely (`role="status" aria-live="polite"` on the row-status region — AvailabilityBadge precedent). No layout changes vs the 13-1b static view — verify against `flow-l-requests-v2/` L1 screenshots.

5. **N4 integrity preserved + the motion-reduce fix.** L5 skeleton / L6 empty / L7 fail-soft from 13-1b remain untouched; SSE errors NEVER hard-fail the view (the static fetch stays the fallback; reconnect is silent). **Scouted convention gap:** `DownloadsStatesV2.tsx` omits per-element `motion-reduce:animate-none` (relies only on global CSS) while newer v2 states files (`LibraryStatesV2`, `DiscoverStatesV2`, `ActivityStates`, `DetailStatesV2`) include it — if 13-1b's `RequestsStates` followed the newer convention, keep it; if any `animate-pulse` lacks the guard, add it here.

6. **Tests + gates.** `useRequestProgress.spec.ts` cloned from `useDownloadProgress.spec.ts` (the `MockEventSource` class + `vi.stubGlobal('EventSource', …)` + static `instances[]` + `emit(type, payload)` harness): asserts NO EventSource on mount (§8), opens on `startTracking()` against `/events`, `request_progress` emit patches the `requestKeys` cache (envelope + snake→camel incl. `progress`), stale-terminal rows preserved, reconnect timer on error. View spec: visibility+view gating (start/stop calls). `pnpm nx test web`, `pnpm nx lint web` (Rule 21 headers already on 13-1b files; any NEW component file needs its own), `pnpm lint:all` green. Visual baselines: only if a NEW visual state is introduced (live % is data, not new chrome) — expected NONE; if fixtures are added, `-linux` via CI bootstrap only.

## Tasks / Subtasks

- [ ] Task 1 (AC #1): `hooks/useRequestProgress.ts` — clone template; `requestService.getSSEUrl()`; spec with MockEventSource harness.
- [ ] Task 2 (AC #2): `applyRequestSnapshot` (exported from the hook file, downloads convention) — merge rules incl. stale-terminal preservation + append-new; spec cases.
- [ ] Task 3 (AC #3): wire into the requests view — visibility + active-view gating effect; view spec for start/stop.
- [ ] Task 4 (AC #4): `RequestRow` live bits — Mono `%` progressbar (TY-3 splits), `error_message` on failed, `aria-live` status region; spec updates.
- [ ] Task 5 (AC #5): N4 regression pass + `motion-reduce:animate-none` audit on the requests states file.
- [ ] Task 6 (AC #6): full gates — `pnpm nx test web`, `pnpm lint:all`; screenshot-compare against `flow-l-requests-v2/` L1 (mandatory UX verification).

## Dev Notes

> ⚠️ **STALE-MARK (filed by 13-7a create-story, SM Bob, 2026-07-05 — re-confirm before dev):** AC #2's merge rule "cached rows ABSENT from the snapshot are STALE-TERMINAL → keep as-is" predates 13-7a's hard-DELETE cancel (`DELETE /api/v1/requests/{id}` removes pending rows outright). An absent row is therefore NOT necessarily terminal history — it may have been deleted-while-active in another tab, and keep-as-is would preserve a phantom pending row until refetch. Required adjustment: preserve absent rows ONLY when the CACHED row's status is terminal (`completed`/`failed`); DROP absent rows whose cached status is active (`pending`/`searching`/`downloading`). Also note: the AC-#3 prose "error_message on failed rows" is partially stale — the caption RENDERING already ships on main (13-1b, `RequestRow.tsx:72-74`) and 13-7b relocates it into the action cluster; this story owns its LIVE refresh only. See 13-7a Discovery Triage.

### Developer context — copy-map (scouted 2026-07-04)

- **THE template:** `hooks/useDownloadProgress.ts` — connect/reconnect/latest-ref excerpt scouted verbatim (lines 61-105); `applyDownloadSnapshot` :30-50; `SSE_RECONNECT_MS=10000` :27. Spec harness: `useDownloadProgress.spec.ts` (MockEventSource :10-31, stubGlobal :84).
- **Gating:** `DownloadsBrowseV2.tsx:109-114` (`usePageVisibility` from `useDownloads.ts:31-40`); add the `view === 'requests'` condition — the downloads page gates on visibility alone because the whole PAGE is the feature; here the view is one Discover mode.
- **Poll retirement note:** downloads v2 retired its 5s poll in favor of SSE (`useDownloads.ts:68`). The requests list has NO poll to retire — 13-1b's static fetch (staleTime default) + SSE patching is the complete freshness story; do NOT add `refetchInterval`.
- **Event-name convention:** named events via `addEventListener('request_progress', …)` — never an `onmessage` switch (house inventory: download_progress/scan_*/subtitle_*). No shared SSE utility exists (each hook self-contains) — cloning is the convention, don't invent an abstraction (YAGNI per 11-1 culture; a third clone can motivate extraction later).
- **Progress rendering:** `DownloadCardV2.tsx:46,99-109` (pct rounding, progressbar a11y, `font-mono tabular-nums`); fill tokens `progressFillClass` :21-27 if a bar is in the L1 design (verify screenshots — if L1 shows only a % figure, no bar, render only the figure).
- **13-1b files this story edits** (authored, will exist post-merge): `components/requests/{RequestRow,RequestsView}.tsx`, `services/requestService.ts`, `hooks/useRequestedMedia.ts` (untouched), `requestKeys` factory.

### Contract acks (Rule 20)

- confirmed against `[@contract-v1]` (Story 13-3a AC #4) — `request_progress` bare-array payload = request resource + ephemeral `progress`; confirmed against `[@contract-v1]` (Story 13-3a AC #2) — derivation semantics (import-window rows arrive as held `downloading`; the FE renders them as 下載中, no special casing); confirmed against `[@contract-v1]` (Story 13-1a AC #3) — list shape (bare array under `data.requests`, camelCase post-transform).
- This story stamps NO new contracts (pure consumer).

### Scope walls

- NO new views/routes (upgrade-in-place). NO polling. NO season/episode UI (13-2b). NO settings UI (13-6). NO backend changes. NO SSE utility extraction.

### Latest-tech note

No new dependency. EventSource/vitest/stubGlobal all in-repo patterns.

### Project Structure Notes

- New: `hooks/useRequestProgress.ts(+spec)`; edits: `services/requestService.ts`, `components/requests/{RequestRow,RequestsView}.tsx(+specs)`, possibly the requests states file (motion-reduce audit).
- Commit scope `feat(13-3b): …`; branch off `main`; gh `j620656786206`; Prettier before commit.

### Time-dependent visual coverage

- **Expected N/A** — live progress is server-pushed data, not a wall-clock read; 13-1b already ruled no relative-time rendering. IF a dev adds any `Date.now()`/`new Date()` (e.g., "updated X ago"), Rule 23 fires: `Clock-mocked` header + ≥2 `clockTime` fixture states (`recent`/`stale`) per the `-gallery.fixtures.tsx:2278-2345` precedent. Prefer not to.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#13-3]
- [Source: apps/web/src/hooks/useDownloadProgress.ts + useDownloadProgress.spec.ts (template)]
- [Source: _bmad-output/implementation-artifacts/13-3a-request-status-tracking.md#AC-2/#AC-4 ([@contract-v1])]
- [Source: _bmad-output/implementation-artifacts/13-1b-one-click-request.md (static view + SCOPE WALL)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2.5 + §3.1-TY-3]
- [Source: project-context.md#§8-Frontend-Lazy-SSE + Rule-5/16/18/20/21/23/26]

## Change Log

| Date       | Change |
| ---------- | ------ |
| 2026-07-04 | Story created (SM create-story, yolo). FE half of 13-3; upgrades the 13-1b static view per the recorded SCOPE WALL. Pure contract consumer (acks 13-3a AC #2/#4, 13-1a AC #3). Merge rule: stale-terminal rows preserved, snapshot rows replace by id. Visibility+view double gate. Status → ready-for-dev. |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - If **YES**: classify each per Rule 24 (①/②/③) with tracked entry IDs; prose-only mentions are banned.

### File List
