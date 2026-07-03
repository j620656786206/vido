# Story ux3-4-3 — Downloads v2 frontend (deep page: card actions + live SSE)

Status: review

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Type:** frontend · **FRs:** PH3-M3 (Epic 14 v2)
**Design:** ux3-4-1 (`.pen` `flow-d-downloads-v2`) · **Owner:** dev (`dev-story`) → tea (visual + E2E)
**Consumes:** `ux3-4-2-downloads-actions-be` `[@contract-v1]` (actions) + `ux3-4-2b-downloads-sse-be` `[@contract-v1]` (SSE payload)

## ⚠️ Two execution gates (read FIRST)

This story is fully prepped but **gated** — like `ux3-discover-facet-aggregation-fe`, the data-layer work splits cleanly from the gated work:

- **GATE A — design drawn.** The visual tasks (AC2 restyle, AC6 states, the `DownloadCard-v2` look) need `ux3-4-1`'s `.pen` `flow-d-downloads-v2` frames to exist (ux-designer Sally must execute ux3-4-1 first). The route/shell-gate scaffolding (AC1) is NOT blocked by this.
- **GATE B — backend merged.** Card actions (AC3) need `ux3-4-2-downloads-actions-be` merged; live SSE (AC4) needs `ux3-4-2b-downloads-sse-be` merged. The v2 restyle of the **existing read-only** list (AC1/2/6 on the current `GET /downloads`) is NOT blocked by GATE B — it can ship the look first and light up actions+SSE when the BE lands.

> **Possible internal split (dev's call):** if one pass is too large, split along the gates — `ux3-4-3a` (shell-gate + v2 restyle + states + pagination, on existing GET, GATE A only) and `ux3-4-3b` (card actions + live SSE + batch, GATE B). Authored here as ONE story (matches the ux3-3-2 single-FE-story precedent); the split is pre-partitioned by AC if needed.

## Story

As a NAS owner managing my download queue,
I want `/downloads` migrated to the v2 shell — with **per-download card actions**, **live SSE progress** replacing the 5-second poll, and **batch ops + pagination** on the retained deep page,
so that Downloads matches the v2 design language and I can finally **control** downloads (pause/resume/remove), see progress update **live without polling storms**, and act on many at once — the gaps the redesign brief (Part 4 D1) called out.

## Context — restyle the EXISTING deep page + light up the new BE (not a rebuild)

The shipped `/downloads` (`apps/web/src/routes/downloads.tsx`, 256 lines) already renders a paginated, status-filtered download list via `useDownloads` / `useDownloadCounts` — but it **polls every 5s** (`useDownloads.ts:69` `refetchInterval: 5000`), has **no card actions** (the service is read-only), and is **not under the v2 shell** (no `staticData: { shell: 'v2' }`). So this story is: **(a)** v2 restyle + shell-gate, **(b)** add card actions (consume ux3-4-2), **(c)** replace the 5s poll with lazy SSE (consume ux3-4-2b), **(d)** add batch ops. The legacy render stays byte-unchanged under the flag OFF.

## Acceptance Criteria

1. **Shell-version gating.** `/downloads` gets `staticData: { shell: 'v2' }` and renders `DownloadsBrowseV2` under the v2 shell (`HomeSidebar-v2`, **下載 active**) when `new_shell_enabled`; the legacy render is **byte-unchanged** under the flag OFF (same gate as `ux3-2-3` activity / `ux3-3-2` discover via `useShellVersion()`). **Downloads is a mobile bottom-4 tab (下載 active)** — NOT More (contrast Discover). [GATE A-light: scaffolding unblocked; visual under GATE A]
2. **v2 deep-page restyle.** The list renders `DownloadCard-v2` (from ux3-4-1: title 2-line CJK clamp, source indicator, progress bar + `xx.x%`, speed ↓↑, ETA, size — **all numerics JetBrains Mono**, status token via DL-v2 §2.5 reusing the N1 lifecycle mapping) + a **status-filter toolbar** (the 6 live values `all/downloading/paused/completed/seeding/error`, counts Mono) + **pagination** restyled to v2. Token-only, Noto Sans TC, 44px. [GATE A]
3. **Card actions (consume `ux3-4-2` `[@contract-v1]`).** Each card exposes `暫停` / `繼續` (state-dependent) / `移除（保留檔案）` / `移除（連同檔案刪除）` (D3-D-v2). Destructive remove gets a confirm dialog. Add `pauseDownload(hash)` / `resumeDownload(hash)` / `removeDownload(hash, deleteFiles)` to `downloadService.ts` (the current `fetchApi` is GET-only — add a POST/DELETE-capable path) + a `useDownloadActions` mutation hook with optimistic update + `invalidateQueries(downloadKeys)`. **`confirmed against [@contract-v1]`** (ux3-4-2: `POST /downloads/:hash/pause|resume`, `DELETE /downloads/:hash?deleteFiles=`). [GATE B]
4. **Live SSE progress, poll retired (consume `ux3-4-2b` `[@contract-v1]`).** A new **lazy** `useDownloadProgress` hook — modeled on `useScanProgress.ts` (`startTracking()`, `new EventSource('/api/v1/events')`, `addEventListener('download_progress', …)`, cleanup on unmount, `mountedRef` guard) — receives `download_progress` snapshots and updates the downloads query cache (`queryClient.setQueryData`). On the v2 page the `useDownloads` `refetchInterval` is turned **OFF** (SSE is the freshness source); **shell-gated so legacy keeps `refetchInterval: 5000`**. **NEVER** `new EventSource()` in a mount-time `useEffect([])` (project-context §8 lazy-SSE rule — eager connect breaks Playwright `networkidle`); connect when the page is active. **`confirmed against [@contract-v1]`** (ux3-4-2b `download_progress` payload shape + key casing). [GATE B]
5. **Batch ops.** A select mode (per-card checkboxes) + a batch action bar (`全選` / `批次暫停` / `批次移除` / `已選 N 項`), reusing the **slice-accepting** action methods (one request for many hashes, per ux3-4-2 AC5). Matches ux3-4-1 `D2-D-v2`. [GATE B]
6. **Four states (N4) to v2** (match ux3-4-1 `D4/D5/D6`): loading skeleton (card-shaped + toolbar), **empty-no-downloads distinct** (`目前沒有下載任務` + a quiet `前往探索` affordance, never a bare blank), and **qBittorrent-unreachable per-section fail-soft** (`無法連線到 qBittorrent` + `重試` + `前往設定`; shell + nav still render; page never hard-fails — reuse the existing `*ConnectionError` → `SETUP_REQUIRED`/`qbtErrorToHTTPStatus` surface the GET path already handles). [GATE A for look; the fail-soft branch already exists in data]
7. **v2 enforcement.** Token-only color (no hex), all CJK Noto Sans TC (TY-1), all numerics (%, speed, ETA, size, counts) JetBrains Mono, colored body text uses `*-text` AA variants (TC-2), `text-disabled` carries no load-bearing text (TC-1).
8. **Reuse, no fork.** Reuse `downloadService` / `useDownloads` / `useDownloadCounts` / the pagination + filter state already in `downloads.tsx`; converge `DownloadCard-v2` tokens with `PosterCardV2` / DL-v2 atoms; **reuse the existing SSE `/api/v1/events` connection** if a shared SSE manager exists rather than opening a second `EventSource` (verify — `useScanProgress` opens its own; if there's no shared manager, lazy-open one for downloads and clean it up).
9. **Tests.** Vitest covers: shell-gating render (v2 vs legacy byte-unchanged); `DownloadCard-v2` (status token, Mono numerics, action affordances); `useDownloadActions` optimistic + invalidate; `useDownloadProgress` updates cache from a `download_progress` event + does NOT connect on mount; batch select + batch call; the three non-default states. **E2E** adds a downloads-v2 block reusing `tests/support/helpers/seed-helpers.ts` real seeding — **NO data-dependent `test.skip` self-skips** (Epic 20 lesson). `nx build web` + `nx lint web` green.

## Tasks / Subtasks

**⏸ Delivered in two PRs (dev's call — the story pre-partitioned this along the gates):**
**4-3a (GATE A, PR #1 — this delivery): AC1/AC2/AC6 + the AC7/8/9 slice for the restyle.**
**4-3b (GATE B, next PR): AC3 actions / AC4 SSE / AC5 batch + the AC7/8/9 slice for those.**

- [x] (AC #1) Add `staticData: { shell: 'v2' }` to `/downloads`; split into `DownloadsPage` (v2 → `DownloadsBrowseV2`, 下載 active in bottom-4 — already wired in `navModel.ts`) / `LegacyDownloads` (byte-unchanged) via `useShellVersion()`. [GATE A-light] ✅ 4-3a
- [x] (AC #2, #6) Build `DownloadsBrowseV2` + `DownloadCardV2` + `DownloadsStatesV2` to the ux3-4-1 frames (toolbar / cards / pagination / skeleton / empty / qBT-fail-soft). [GATE A] ✅ 4-3a
- [x] (AC #3) `downloadService`: added a POST/DELETE `mutateApi` helper + `pauseDownload`/`resumeDownload`/`removeDownload`; `useDownloadActions` mutation hook (optimistic patch + rollback + `invalidateQueries(downloadKeys)`); card action cluster (暫停/繼續) + destructive-remove Radix confirm dialog (保留檔案 / 連同檔案刪除). `confirmed against [@contract-v1]` (ux3-4-2). ✅ 4-3b
- [x] (AC #4) `useDownloadProgress` lazy-SSE hook (mirrors `useScanProgress`); merges `download_progress` snapshots into the query cache (preserve `parse_status`, map bare array → envelope, drop removed); `refetchInterval` turned OFF on v2 via `useShellVersion()` gate in `useDownloads` (legacy keeps 5s). NEVER connects on mount — visibility-gated `startTracking()`. `confirmed against [@contract-v1]` (ux3-4-2b). ✅ 4-3b
- [x] (AC #5) Batch select mode (per-card checkboxes) + batch action bar (全選 / 批次暫停 / 批次繼續 / 批次移除 / 已選 N 項). Batch = **N parallel single-hash requests** (`Promise.allSettled`) — the ux3-4-2 HTTP API is single-hash only (see Completion Notes deviation). ✅ 4-3b + sort control + select-mode.
- [x] (AC #7, #8) Token-lint pass; reuse/converge — Pagination/Button/Dialog atoms reused; status→token via libraryStatus TINT; no forked card chrome; the SSE hook lazy-opens ONE `EventSource` (no shared manager exists — same as `useScanProgress`) and cleans it up. ✅ 4-3a + 4-3b.
- [x] (AC #9) Vitest (28 total: 4-3a shell-gate/card/3-states/status-util + 4-3b useDownloadActions optimistic/rollback/batch, useDownloadProgress no-mount-connect/merge, card actions/confirm/select) + E2E downloads-v2 block (6: restyle / empty / qBT-fail-soft / card-pause / remove-confirm-DELETE / batch-pause — route-interception, no self-skips); `nx build`/`nx lint` web green. ✅ 4-3a + 4-3b.

## Dev Notes

### Cross-stack split check (MANDATORY) — NO split (FE-only)

Backend tasks: **0** — the two BE halves are separate, merged stories (`ux3-4-2` actions + `ux3-4-2b` SSE); this story only **consumes** their `[@contract-v1]`. Frontend tasks: ~7. Backend 0 → **single frontend story, no cross-stack split.** (Internal FE split along GATE A/B is optional — see top.)

### Source tree (real symbols — do not invent)

- Route: `apps/web/src/routes/downloads.tsx` (256 lines; add `staticData` + shell-gate split).
- Service: `apps/web/src/services/downloadService.ts` — `downloadService.{getDownloads,getDownloadDetails,getDownloadCounts}` + types `Download` / `FilterStatus` (`all|downloading|paused|completed|seeding|error`) / `PaginatedDownloads`; `fetchApi<T>` (`:99`) is **GET-only → add a POST/DELETE helper** for actions.
- Hooks: `apps/web/src/hooks/useDownloads.ts` — `useDownloads` (`:64`, `refetchInterval: 5000` `:69` → turn off on v2), `useDownloadCounts` (`:84`), `downloadKeys` (query-key factory for invalidation).
- SSE precedent: `apps/web/src/hooks/useScanProgress.ts` — lazy `startTracking()`, `new EventSource('/api/v1/events')` (`:162`), `addEventListener('scan_progress', …)` (`:170`), `mountedRef` guard, "No polling — SSE only" (`:226`). Mirror for `download_progress`.
- Shell-gate pattern: `routes/discover.tsx:47-73` (`staticData: { shell: 'v2' }` + `useShellVersion()` → `DiscoverBrowseV2` / `LegacyDiscover`); `routes/activity.tsx` (bottom-4 tab precedent — Downloads is a tab like Activity, unlike Discover).
- v2 atoms / shell: `HomeSidebar-v2`, `MobileTabItem` (下載 active), `PosterCardV2` tokens; DL-v2 §2.5 status→token (reuse the N1 lifecycle mapping from `ux3-0-2` `pickPosterBadge` / `libraryStatus`).

### Design reference (ux3-4-1, flow-d-downloads-v2 — must be drawn first, GATE A)

- `D1-D-v2` default · `D2-D-v2` batch · `D3-D-v2` card actions · `D1-M-v2` mobile (下載 in bottom-4) · `D4-D-v2` skeleton · `D5-D-v2` empty · `D6-D-v2` qBT-unreachable. `Component/DownloadCard-v2`. Screenshots `_bmad-output/screenshots/flow-d-downloads-v2/` (created when ux3-4-1 ships).

### Contract acks (Rule 20)

- **`confirmed against [@contract-v1]`** — `ux3-4-2-downloads-actions-be` (method+path+`deleteFiles` semantics+envelope) AND `ux3-4-2b-downloads-sse-be` (`download_progress` payload shape + key casing). Record both ack lines; if either BE bumps to v2 before this ships, re-ack (Rule 20 stale-mark).

### Project Structure Notes

- Touches `apps/web/src/{routes/downloads.tsx, services/downloadService.ts, hooks/useDownloads.ts, hooks/useDownloadProgress.ts (new), components/downloads/* (new: DownloadsBrowseV2, DownloadCard-v2, DownloadsStatesV2)}` + tests. No new route. Shell-gated — legacy preserved under flag OFF.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?** **Likely YES** — download cards may render a relative "added 3h ago" / ETA-as-clock. **ETA is server-supplied** (seconds remaining, not a client clock read) so it is clock-independent; but if `DownloadCard-v2` renders `added_on` as a relative time it reads the wall clock → **capture ≥2 fixture baselines** (e.g. `recent` / `stale`) per Rule 23, pair with `withFixedClock(page, iso)` + a `Clock-mocked` marker. If no component reads the wall clock, state `N/A`. **Decide during dev.**
- Reference: `project-context.md` Rule 23; helper `tests/visual/clock-mock.ts` `withFixedClock`; precedent story 19-9 (`recent`/`stale`).

### Discovery Triage

- **Carried in from ux3-4-1 / ux3-4-2 / ux3-4-2b:**
  - **② spawn-blocking (already filed)** — card actions BE (`ux3-4-2`) + download SSE BE (`ux3-4-2b`) are GATE B; this story is `blocked-by` their merge for AC3/4/5 (AC1/2/6 on the existing GET are not).
  - **② design prereq (already filed)** — `ux3-4-1` `.pen` frames are GATE A for the visual ACs.
  - **③** — shared-SSE-connection question (AC8): if no shared `/api/v1/events` manager exists, this opens a 2nd `EventSource` (like `useScanProgress` does). Acceptable, but file a `backlog` consolidation note if connection count becomes a concern.
- Reference: `project-context.md` Rule 24.

### References

- [Source: `ux3-4-1-downloads-design.md` (flow-d-downloads-v2 frames + DownloadCard-v2)]
- [Source: `ux3-4-2-downloads-actions-be.md` `[@contract-v1]`] · [Source: `ux3-4-2b-downloads-sse-be.md` `[@contract-v1]`]
- [Source: apps/web/src/routes/downloads.tsx] · [Source: apps/web/src/services/downloadService.ts] · [Source: apps/web/src/hooks/useDownloads.ts]
- [Source: apps/web/src/hooks/useScanProgress.ts (lazy-SSE consumer pattern)]
- [Source: apps/web/src/routes/discover.tsx#47-73 (shell-gate)] · [Source: routes/activity.tsx (bottom-4 tab)]
- [Source: project-context.md §8 SSE Hub + frontend lazy-SSE pattern; Rule 5 (TanStack Query), Rule 18 (case transform), Rule 23]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia, dev-story) — 2026-07-03

### Debug Log References

- `npx vitest run` (4 new specs) → 13 pass; full `pnpm nx test web` → 206 files / 2264 tests pass.
- `pnpm nx build web` → EXIT 0; `pnpm nx lint web` → EXIT 0 (Rule 21 headers + jsx-a11y clean); `pnpm lint:all` + prettier clean.
- `npx playwright test tests/e2e/downloads-v2.spec.ts --project=chromium` → 3 pass (15.1s).

### Completion Notes List

**PR #1 — ux3-4-3a (GATE A: shell-gate + v2 restyle + states + pagination, on the existing read-only GET). GATE-B (AC3 actions / AC4 SSE / AC5 batch) deferred to PR #2 (ux3-4-3b).**

- **AC1 (shell-gate)** — `/downloads` gets `staticData: { shell: 'v2' }`; the current `DownloadsPage` body is preserved byte-unchanged as `LegacyDownloads`, and a new `DownloadsPage` branches on `useShellVersion()` (`shell === 'v2' ? <DownloadsBrowseV2/> : <LegacyDownloads/>`) — mirrors discover/activity. 下載 is already a bottom-4 tab in `navModel.ts` (auto-active, no config change).
- **AC2 (restyle)** — `DownloadCardV2` (2-line CJK title clamp, qBittorrent source pill, progress bar + `xx.x%`, ↓↑ speed / ETA / size — all Mono + `tabular-nums`, status token) + `DownloadsBrowseV2` (v2 container + status-filter toolbar: 6 values with Mono counts + v2 `Pagination` atom reused). Status→token via a new `downloadStatus.ts` that mirrors `libraryStatus.ts` TINT (ux3-4-1 decision #5 — one badge system). **Deferred to 4-3b (paired with batch/actions): the D7 List|Table view toggle, the 選取 select-mode, and a sort control** (4-3a uses the newest-first default).
- **AC6 (4 states)** — `DownloadsStatesV2`: card-shaped skeleton (`aria-busy`), empty (all-filter → distinct `目前沒有下載任務` + quiet 前往探索; other filters → switch hint), and qBT-unreachable per-section fail-soft (`role=alert` + 重試 + 前往設定). Fail-soft covers BOTH the poll error AND resolved-but-not-configured (`useDownloads` fails closed on the qBT config gate with no error, so `DownloadsBrowseV2` also reads `useQBittorrentConfig`).
- **AC7/8 (v2 enforcement + reuse)** — token-only color (no hex), CJK Noto Sans TC default, numerics `font-mono tabular-nums`, accent/error body text via `-text` AA variants (TC-2). Reused `ui/Pagination` + `ui/Button` atoms + `useDownloads`/`useDownloadCounts` (poll stays for 4-3a). `TechBadge` intentionally NOT used for the source indicator (its categories are video/audio/hdr and it uses non-token literals) — a token-styled source pill instead.
- **🎭 A11y Pre-Flight: PASS** (5 components checked, 0 jsx-a11y warnings on touched files, 0 introduced). The 4 recurring classes: responsive `<img>` — **N/A** (no images); modal focus — **N/A** (the destructive-remove dialog is 4-3b); async-revealed status — the progress bar carries `role=progressbar` + `aria-valuenow` (SR-queryable) and the status pill is a static span **by deliberate choice** — `aria-live` on a value that updates every poll would spam the SR (anti-pattern); custom widgets — the filter toolbar uses `role=tablist`/`role=tab`/`aria-selected`/`aria-controls` on real `<button>`s (keyboard-native).
- **🔗 AC Drift: NONE** (4-3a consumes NO wire contract — it restyles the EXISTING `GET /downloads` read path; the `[@contract-v1]` acks for ux3-4-2 actions + ux3-4-2b SSE happen in 4-3b when those endpoints are consumed).
- **📎 Contract Stamps: NONE** (4-3a defines/consumes no `[@contract-v*]`; the two `confirmed against [@contract-v1]` acks are 4-3b work — both upstream BE halves are merged and ready).
- **Rule 23 (time-dependent visual): N/A** — `DownloadCardV2` reads no wall clock (`Date.now()`/`new Date()`): ETA is server-supplied seconds (clock-independent) and the card renders no relative "added Nh ago".

**PR #2 — ux3-4-3b (GATE B: card actions + live SSE + batch). Consumes the two merged BE `[@contract-v1]` halves.**

- **AC3 (card actions)** — `downloadService` gained a POST/DELETE `mutateApi` helper + `pauseDownload`/`resumeDownload`/`removeDownload`. `useDownloadActions` runs each as a `useMutation` with an **optimistic** cache patch (pause→paused, resume→downloading, remove→dropped + totalItems−) and **rollback on error** + `invalidateQueries(downloadKeys)` onSettled. `DownloadCardV2` renders a state-dependent 暫停/繼續 button + a 移除 button that opens a **Radix Dialog** (focus-trap + Escape + aria-modal for free) offering 保留檔案 / 連同檔案刪除. `confirmed against [@contract-v1]` (ux3-4-2: `POST /downloads/:hash/pause|resume`, `DELETE /downloads/:hash?deleteFiles=`).
- **AC4 (live SSE, poll retired)** — `useDownloadProgress` mirrors `useScanProgress`: lazy `startTracking()`, one `EventSource(/api/v1/events)`, `addEventListener('download_progress')`, `mountedRef` guard, reconnect, unmount cleanup. `applyDownloadSnapshot` reconciles the ux3-4-2b `[@contract-v1]` three deltas into the cache — **preserve `parse_status`** (merge, never replace), map the bare `Torrent[]` into each page's `.items`, drop removed torrents. `useDownloads` gates `refetchInterval` OFF on v2 via `useShellVersion()` (legacy keeps 5000). **NEVER connects on mount** — `DownloadsBrowseV2` calls `startTracking()` from a **visibility-gated** effect (deps `[isVisible, …]`, not bare `[]`), which also disconnects on tab-hide → the BE broadcaster's `ClientCount()` gate drops the server poll to zero. `confirmed against [@contract-v1]` (ux3-4-2b).
- **AC5 (batch)** — select mode (per-card checkboxes) + a batch action bar. **⚠️ Deviation (documented):** AC5 says "one request for many hashes", but ux3-4-2's slice-accepting methods live at the qBittorrent Go-client layer — the **HTTP API is single-hash only** (no batch route). So a batch op is **N parallel single-hash requests** via `Promise.allSettled` (correct for a single-user NAS). Filed as Discovery Triage below; a batch endpoint is a future BE story if request volume ever matters.
- **AC7/8** — reused `Button`/`Dialog`/`Pagination` atoms; the SSE hook opens ONE `EventSource` (no shared manager exists — same as `useScanProgress`). Added a sort control (欄位 + 方向) + select-mode toggle to the toolbar. **D7 Table view deferred** as a follow-up enhancement (not an AC).
- **🎭 A11y Pre-Flight: PASS** (jsx-a11y clean on all touched files). Modal focus — the remove/batch-remove confirms use Radix Dialog (focus-trapped, Escape-closes, aria-modal). Custom widgets — actions are real `<button>`s with descriptive aria-labels; the select checkbox is a native `<input type=checkbox>` with an aria-label; select-toggle carries `aria-pressed`. Progress stays `role=progressbar` + `aria-valuenow` (no noisy per-tick aria-live). The batch bar is keyboard-native.
- **🔗 AC Drift: NONE** — this half CONSUMES ux3-4-2 + ux3-4-2b `[@contract-v1]` as-shipped (pause/resume/remove wire shape + `download_progress` payload); it changes no upstream contract. The AC5 single-vs-batch gap is an FE-side implementation choice, not a contract change.
- **📎 Contract Stamps: FOUND** — `confirmed against [@contract-v1]` recorded for BOTH ux3-4-2 (actions) and ux3-4-2b (SSE `download_progress`), the two upstream stamps this story acks. Neither upstream bumped before this shipped (no Rule-20 stale-mark needed).
- **Rule 23 (time-dependent visual): N/A** — no touched component reads `Date.now()`/`new Date()` (ETA is server-supplied seconds; the card shows no relative "added Nh ago").

### Discovery Triage (4-3b)

- **④ — batch download HTTP endpoint.** AC5's "one request for many hashes" isn't backed by the BE: ux3-4-2 exposes single-hash HTTP routes only (its `[]string` methods are qBT-Go-client-internal). Implemented client-side as N `Promise.allSettled` requests. **Deferred** — a `POST /downloads/batch/{pause,resume}` + `DELETE /downloads/batch` endpoint is only worth it if a user routinely batches dozens of torrents (unlikely on a single-user NAS). File a `backlog` entry if it surfaces. Reference: `project-context.md` Rule 24; origin: this story's AC5 vs the ux3-4-2 HTTP surface.

### File List

- `apps/web/src/routes/downloads.tsx` (MODIFIED — 4-3a: staticData shell:v2 + useShellVersion gate; current page → LegacyDownloads byte-unchanged)
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx` (NEW 4-3a; MODIFIED 4-3b — actions + lazy-SSE wiring + select mode + batch bar + sort)
- `apps/web/src/components/downloads/DownloadCardV2.tsx` (NEW 4-3a; MODIFIED 4-3b — action cluster + selection checkbox + remove confirm dialog)
- `apps/web/src/components/downloads/DownloadsStatesV2.tsx` (NEW 4-3a — skeleton / empty / qBT-fail-soft)
- `apps/web/src/components/downloads/downloadStatus.ts` (NEW 4-3a — status→token util, mirrors libraryStatus TINT)
- `apps/web/src/components/downloads/index.ts` (MODIFIED 4-3a — barrel exports for the v2 components)
- `apps/web/src/services/downloadService.ts` (MODIFIED 4-3b — mutateApi POST/DELETE helper + pause/resume/remove + getSSEUrl)
- `apps/web/src/hooks/useDownloads.ts` (MODIFIED 4-3b — refetchInterval gated OFF on v2 via useShellVersion; export usePageVisibility)
- `apps/web/src/hooks/useDownloadActions.ts` (NEW 4-3b — optimistic pause/resume/remove mutations, single + batch)
- `apps/web/src/hooks/useDownloadProgress.ts` (NEW 4-3b — lazy SSE + applyDownloadSnapshot cache merge)
- `apps/web/src/components/downloads/DownloadCardV2.spec.tsx` (NEW 4-3a; MODIFIED 4-3b — action/selection/dialog tests)
- `apps/web/src/components/downloads/DownloadsStatesV2.spec.tsx` (NEW 4-3a)
- `apps/web/src/components/downloads/downloadStatus.spec.ts` (NEW 4-3a)
- `apps/web/src/routes/downloads.spec.tsx` (NEW 4-3a — shell-gate v2-vs-legacy)
- `apps/web/src/hooks/useDownloadActions.spec.ts` (NEW 4-3b)
- `apps/web/src/hooks/useDownloadProgress.spec.ts` (NEW 4-3b)
- `tests/e2e/downloads-v2.spec.ts` (NEW 4-3a; MODIFIED 4-3b — card-pause / remove-confirm-DELETE / batch-pause)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED — 4-3a/4-3b split entries + statuses)

## Change Log

| Date       | Change                                                                                                                                       |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-06-30 | Story created (SM create-story). FE v2 downloads page consuming ux3-4-1 design + ux3-4-2 (actions) + ux3-4-2b (SSE) `[@contract-v1]` ×2. Two execution gates (design drawn / BE merged) partition the work; replaces the 5s poll with lazy SSE (shell-gated). FE-only, no split. Status → ready-for-dev (gated). |
| 2026-07-03 | **PR #2 — ux3-4-3b (GATE B) implemented** (dev-story, Amelia). AC3 card actions (downloadService POST/DELETE + useDownloadActions optimistic+rollback+invalidate + Radix confirm dialog; acks ux3-4-2 [@contract-v1]). AC4 lazy useDownloadProgress SSE (applyDownloadSnapshot preserves parse_status / maps bare array→envelope / drops removed; v2 5s poll retired via useShellVersion gate; visibility-gated startTracking, never mount-connect; acks ux3-4-2b [@contract-v1]). AC5 batch select + bar (N Promise.allSettled requests — ux3-4-2 HTTP is single-hash only; deviation + Triage ④ filed) + sort control. +15 Vitest (28 total) + 3 E2E (6 total); build/lint/test/lint:all/prettier green; jsx-a11y clean. A11y PASS, AC-Drift NONE, Contract-Stamps FOUND (2 acks), Rule-23 N/A. D7 Table view deferred. Umbrella + both split halves → done on merge. |
| 2026-07-03 | **PR #1 — ux3-4-3a (GATE A) implemented** (dev-story, Amelia). Both gates now open (ux3-4-1 #107, ux3-4-2 + ux3-4-2b merged). Delivered AC1 (shell-gate, legacy byte-unchanged) + AC2 (DownloadCardV2 + DownloadsBrowseV2 + status toolbar + reused Pagination) + AC6 (skeleton/empty/qBT-fail-soft) + the AC7/8/9 slice for the restyle (13 Vitest + 3 E2E, jsx-a11y clean, build/lint/test green). status→token via downloadStatus.ts mirroring libraryStatus TINT. A11y PASS; AC Drift NONE; Contract Stamps NONE (acks are 4-3b). Deferred to PR #2 (ux3-4-3b, GATE B): AC3 card actions / AC4 lazy-SSE (retire the 5s poll) / AC5 batch + D7 Table view + sort control + select-mode. Story stays in-progress. |
