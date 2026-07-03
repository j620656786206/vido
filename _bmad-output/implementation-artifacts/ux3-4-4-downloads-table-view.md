# Story ux3-4-4 — Downloads v2 desktop Table view (D7) + List|Table toggle

Status: done

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Type:** frontend · **FRs:** PH3-M3 (Epic 14 v2)
**Design:** ux3-4-1 (`.pen` `flow-d-downloads-v2` frame `D7-D-v2` = node `w3ipb`) · **Owner:** dev (`dev-story`) → tea (visual + E2E)
**Follows:** `ux3-4-3` (List view + shell-gate + card actions + live SSE + batch, merged #111/#112). This story adds the **alternate Table view** that ux3-4-3b deferred.

## Story

As a NAS owner triaging a long download queue,
I want a **dense, sortable Table view** on the desktop Downloads page with a **List | Table toggle**,
so that I can scan and act on many downloads at once in a compact tabular layout (the ux3-4-1 `D7-D-v2` design), while the single-column card List stays the default for casual monitoring.

## Context — an alternate view over the SAME data + plumbing (not a rebuild)

`ux3-4-3` shipped the v2 deep page: `DownloadsBrowseV2` renders a single-column `DownloadCardV2` **List** (default), with a status-filter toolbar, a sort control, select-mode + batch bar, card actions (`useDownloadActions`), live SSE (`useDownloadProgress`), and the four states. This story adds a **second rendering** of the exact same `useDownloads` page data — a dense table (`DownloadsTableV2`) — behind a **List | Table toggle** in the existing toolbar. Everything else is REUSED: the filter toolbar, pagination, `useDownloadActions`, `useDownloadProgress`, `downloadStatus`, `formatters`, and the sort state. The only net-new UI is the table, its column-header sorting, its persistent checkbox column, and a table-shaped skeleton.

## Acceptance Criteria

1. **View toggle.** The `DownloadsBrowseV2` toolbar gains a **List | Table** segmented toggle (desktop). The choice **persists** across navigations/reloads (localStorage, e.g. `vido:downloads:view`, default `list`). Mobile is unaffected — it stays the card List (the table is a desktop-only affordance; below the `sm`/`md` breakpoint the toggle is hidden and List renders). [D7-D-v2]
2. **`DownloadsTableV2` — dense sortable table.** Renders the current page's `Download[]` as a table matching `D7-D-v2`: columns = **[checkbox] · 名稱 · 狀態 · 大小 · 進度 · 速度 · ETA · 操作**. Title cell = 2-line clamp; 狀態 = the `downloadStatus` token pill; 大小/速度/ETA = JetBrains Mono + `tabular-nums`; 進度 = a compact inline progress bar + `xx.x%`. Semantic `<table>` with `<th scope="col">` headers. Token-only, Noto Sans TC. [D7-D-v2]
3. **Sortable column headers.** Clicking a sortable header (名稱 / 大小 / 進度 / 狀態 — the `SortField` values) sets `sortField` and toggles `sortOrder`, reusing the SAME sort state that already drives `useDownloads` (lift it from `DownloadsBrowseV2` if needed). The active-sort header shows a direction indicator (↑/↓) and `aria-sort`. No new sort plumbing — the List sort control and the Table headers are two controls over one state.
4. **Persistent checkbox column.** In Table view the checkbox column is **always present** (no separate select-mode toggle — the design's "persistent checkbox col, no select-mode"). Selecting rows reuses the SAME selection state + batch action bar from `ux3-4-3b` (which morphs in-place above the table). A header checkbox does 全選 / clear for the current page.
5. **Row actions.** Each row exposes pause/resume + remove (reusing `useDownloadActions` + the same Radix confirm dialog for destructive remove) — an actions cell (icon buttons) or a per-row overflow menu, matching `D7-D-v2`.
6. **Live SSE + reuse (no fork).** The Table view consumes the SAME `useDownloads` cache updated live by `useDownloadProgress` (rows update in place; no second EventSource, no second poll). All four states work in Table view: **table-shaped skeleton** (the variant ux3-4-1 deferred — build it now), empty, and qBT-unreachable fail-soft (the empty/error states may reuse the existing `DownloadsStatesV2` full-width, they need not be tabular).
7. **v2 enforcement.** Token-only color (no hex), CJK Noto Sans TC, all numerics JetBrains Mono, AA `-text` variants for accent/error body text (TC-2), `text-disabled` carries no load-bearing text (TC-1). Horizontal overflow on narrow desktop widths scrolls inside the table's own container (never the page body).
8. **Tests.** Vitest: view-toggle (List↔Table render + localStorage persistence); `DownloadsTableV2` (columns, status token, Mono numerics, `aria-sort` on the active header, header-click → sort callback); persistent checkbox + header 全選; row actions call the handlers. E2E: a downloads-v2 Table block (toggle to Table, assert rows + a column sort request + a row action) — route-interception, **no data-dependent self-skips**. `nx build`/`nx lint` web green (incl. Rule 21 header + jsx-a11y).

## Tasks / Subtasks

- [x] (AC #1) `useDownloadsView()` hook backed by localStorage (`vido:downloads:view`, default `list`, invalid→list); List|Table segmented toggle in `DownloadsBrowseV2`'s toolbar (`hidden lg:flex`); a guarded `useIsDesktop()` (matchMedia) forces List below `lg` even if the stored pref is `table`.
- [x] (AC #2, #7) `DownloadsTableV2` — semantic `<table>` in an `overflow-x-auto` container to the `D7-D-v2` columns (☑·名稱·狀態·大小·進度·速度·ETA·操作); reuses `downloadStatus` + `formatters`; token-only, Mono `tabular-nums` numerics.
- [x] (AC #3) Sortable `<th>` headers (名稱/狀態/進度 — the `SortField`s) wired to the SHARED `sortField`/`sortOrder` (lifted in `DownloadsBrowseV2` via `handleSort`); `aria-sort` (ascending/descending/none) + ↑/↓ arrow on the active header.
- [x] (AC #4) Persistent checkbox column + header checkbox (全選/取消全選, indeterminate when partial), reusing the `ux3-4-3b` `selected` Set + batch bar (bar shows on selection in Table; on select-mode in List).
- [x] (AC #5) Row action cell reusing **extracted** `DownloadRowActions` (pause/resume + remove Radix confirm) — shared by BOTH `DownloadCardV2` and the table row (no fork, AC7).
- [x] (AC #6) Wired into `DownloadsBrowseV2` (same `useDownloads` data + `useDownloadProgress` SSE + pagination — no second EventSource); `DownloadsTableSkeletonV2` (the deferred variant); empty + qBT-fail-soft reuse the existing full-width states.
- [x] (AC #8) Vitest (10: `useDownloadsView` persist/default/invalid; `DownloadsTableV2` render/aria-sort/header-click/checkbox/全選/row-actions) + E2E Table block (toggle→table + column-sort request + row action, route-interception, no self-skips); `nx build`/`nx lint` web green (Rule 21 + jsx-a11y).

## Dev Notes

### Cross-stack split check (MANDATORY) — NO split (FE-only)

Backend tasks: **0** — this is a pure alternate rendering over the existing `GET /downloads` + the already-merged `ux3-4-2`/`ux3-4-2b` `[@contract-v1]` (already consumed by `ux3-4-3`; NO new ack, NO contract change). Frontend tasks: ~6. Single frontend story, no split.

### Source tree (real symbols shipped by ux3-4-3 — reuse, do not re-invent)

- Container: `apps/web/src/components/downloads/DownloadsBrowseV2.tsx` — owns the toolbar, `sortField`/`sortOrder` state, select mode + `selected` Set + batch bar, `useDownloads`/`useDownloadCounts`, `useDownloadActions`, `useDownloadProgress` (visibility-gated), pagination. Add the view toggle + branch List vs `DownloadsTableV2`; **lift `sortField`/`sortOrder` + the selection state** so both views share them.
- List card (reference for cells/actions): `apps/web/src/components/downloads/DownloadCardV2.tsx` — props `{download, selectable, selected, onSelectChange, onPause, onResume, onRemove}`; the remove Radix dialog pattern; Mono/token conventions.
- States: `apps/web/src/components/downloads/DownloadsStatesV2.tsx` (`DownloadsSkeletonV2`/`DownloadsEmptyV2`/`DownloadsQbtErrorV2`) — reuse empty + fail-soft; add a table-shaped skeleton (new export or a `variant` prop).
- Status→token: `apps/web/src/components/downloads/downloadStatus.ts` (`getDownloadStatus`). Formatters: `apps/web/src/components/downloads/formatters.ts` (`formatSpeed/Size/ETA/Progress`).
- Hooks (reuse as-is): `useDownloadActions` (`hooks/useDownloadActions.ts`), `useDownloadProgress` (`hooks/useDownloadProgress.ts`), `useDownloads`/`useDownloadCounts`/`usePageVisibility`/`downloadKeys` (`hooks/useDownloads.ts`). `SortField` = `'added_on' | 'name' | 'progress' | 'status'` (`services/downloadService.ts:19`).
- Atoms: `ui/Button`, `ui/Dialog`, `ui/Pagination`. Sortable-table precedent, if any: check `components/library/LibraryListRowV2.tsx` / any existing `*TableV2` for the header-sort + `aria-sort` convention before inventing one.

### Design reference (ux3-4-1, GATE A — already drawn/merged)

- `D7-D-v2` (node `w3ipb`) = the dense sortable Table with the persistent checkbox column + in-place batch-bar morph. Screenshot: `_bmad-output/screenshots/flow-d-downloads-v2/d7-d-v2.png`. The List default is `D1-D-v2` (`cK1KF`). Mobile (`D1-M-v2`) has NO table — card list only.

### Contract acks (Rule 20)

- **NONE new.** This story consumes only what `ux3-4-3` already consumes (`ux3-4-2` actions + `ux3-4-2b` `download_progress`, both `[@contract-v1]`, already `confirmed against` in ux3-4-3). No re-ack needed unless an upstream BE bumps before this ships (Rule 20 stale-mark).

### Time-dependent visual coverage

- **N/A** — like `DownloadCardV2`, the table cells read no wall clock: ETA is server-supplied seconds; no relative "added Nh ago" is rendered. If a dev chooses to add an 加入時間 column rendering `addedOn` as a **relative** time, that reads `Date.now()` → capture ≥2 clock-mocked fixtures per Rule 23 (`withFixedClock`); rendering it as an **absolute** date (`formatDate`) stays clock-independent. Decide during dev.

### Discovery Triage

- **Carried in from ux3-4-3b:** D7 Table view was explicitly deferred as a follow-up enhancement (not an AC of ux3-4-3); this story formalizes it. No new out-of-scope work anticipated — it is a rendering over existing data/plumbing.
- Reference: `project-context.md` Rule 24.

### References

- [Source: `ux3-4-1-downloads-design.md` — `D7-D-v2` (`w3ipb`), v2.1 rework: List default + dense sortable Table]
- [Source: `ux3-4-3-downloads-frontend.md` — the List view + shared plumbing this story reuses; the D7 deferral note]
- [Source: apps/web/src/components/downloads/{DownloadsBrowseV2,DownloadCardV2,DownloadsStatesV2,downloadStatus,formatters}.tsx]
- [Source: apps/web/src/hooks/{useDownloads,useDownloadActions,useDownloadProgress}.ts]
- [Source: project-context.md §8 (SSE reuse — no second EventSource), Rule 5 (TanStack Query), Rule 21 (.pen header), Rule 23]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia, dev-story) — 2026-07-03

### Debug Log References

- `npx vitest run` (2 new specs) → 10 pass; full `pnpm nx test web` → 210 files / 2289 tests pass.
- `pnpm nx build web` EXIT 0; `pnpm nx lint web` EXIT 0 (Rule 21 header + jsx-a11y clean); `pnpm lint:all` + prettier clean.
- `npx playwright test tests/e2e/downloads-v2.spec.ts --project=chromium` → 7 pass (incl. the new Table block), 15.4s.

### Completion Notes List

- **AC1 (toggle + persist)** — `useDownloadsView()` (localStorage `vido:downloads:view`, default `list`, invalid→list). List|Table segmented toggle (`清單`/`表格`) in the toolbar, `hidden lg:flex` (desktop-only). A guarded `useIsDesktop()` (`useSyncExternalStore` + `matchMedia('(min-width:1024px)')`, falls back to desktop when matchMedia is absent) forces the card List below `lg` even if a stale desktop pref says `table` — exercised for real in the E2E's 1280px viewport (unit tests never render the real container: `downloads.spec.tsx` stubs `DownloadsBrowseV2`).
- **AC2 (table)** — `DownloadsTableV2`: semantic `<table>` in `overflow-x-auto`, columns ☑·名稱·狀態·大小·進度·速度·ETA·操作; reuses `downloadStatus` + `formatters`; numerics `font-mono tabular-nums`; status pill token; inline progress bar with `role=progressbar` + `aria-valuenow`.
- **AC3 (sort)** — sortable headers for the three `SortField`s that have columns (名稱/狀態/進度); wired to the SHARED `sortField`/`sortOrder` via `DownloadsBrowseV2.handleSort` (click same field → toggle order, else set field + desc). `aria-sort` = ascending/descending/none; ↑/↓ arrow on the active header. 加入時間 has no column but the toolbar sort control still offers it (one state, two controls).
- **AC4 (selection)** — persistent checkbox column + a header checkbox (全選/取消全選, `indeterminate` when partial) reusing the ux3-4-3b `selected` Set. Batch bar shows on selection in Table (`selected.size>0`) vs on select-mode in List — the select-mode toggle is hidden in Table (checkboxes are always present).
- **AC5/AC7 (no fork)** — extracted the pause/resume + remove-confirm cluster from `DownloadCardV2` into a shared `DownloadRowActions`, now used by BOTH the card and the table row (same Radix dialog, same aria-labels — the existing DownloadCardV2 action tests stayed green through the refactor).
- **AC6 (reuse)** — Table renders inside `DownloadsBrowseV2` over the same `useDownloads` data; `useDownloadProgress` SSE updates both views (ONE EventSource, no second poll); pagination shared. `DownloadsTableSkeletonV2` added (the variant ux3-4-1 deferred); empty + qBT-fail-soft reuse the full-width `DownloadsStatesV2`.
- **🎭 A11y Pre-Flight: PASS** (jsx-a11y clean). Table is semantic (`<table>`/`<thead>`/`<th scope=col>`); sortable headers carry `aria-sort`; checkboxes are native `<input type=checkbox>` with aria-labels; the header checkbox uses `indeterminate`; row actions reuse the Radix-dialog `DownloadRowActions` (focus-trap + Escape + aria-modal); progress cells keep `role=progressbar`. No `<img>` (no responsive-image class).
- **🔗 AC Drift: NONE** — consumes NO new wire contract; an alternate rendering over the same `useDownloads` data + the ux3-4-2/ux3-4-2b `[@contract-v1]` already consumed by ux3-4-3.
- **📎 Contract Stamps: NONE** — this story defines/consumes no `[@contract-v*]` of its own; the upstream acks live in ux3-4-3 (unchanged).
- **Rule 23 (time-dependent visual): N/A** — no added column reads the wall clock (no relative-time 加入時間 column was added; ETA is server-supplied seconds).

### File List

- `apps/web/src/hooks/useDownloadsView.ts` (NEW — List|Table view preference, localStorage-persisted)
- `apps/web/src/components/downloads/DownloadsTableV2.tsx` (NEW — dense sortable Table)
- `apps/web/src/components/downloads/DownloadRowActions.tsx` (NEW — shared pause/resume + remove-confirm cluster)
- `apps/web/src/components/downloads/DownloadCardV2.tsx` (MODIFIED — use the extracted DownloadRowActions)
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx` (MODIFIED — view toggle + useIsDesktop + List/Table branch + shared sort/handleSort + table skeleton)
- `apps/web/src/components/downloads/DownloadsStatesV2.tsx` (MODIFIED — add DownloadsTableSkeletonV2)
- `apps/web/src/components/downloads/index.ts` (MODIFIED — barrel exports)
- `apps/web/src/hooks/useDownloadsView.spec.ts` (NEW)
- `apps/web/src/components/downloads/DownloadsTableV2.spec.tsx` (NEW)
- `tests/e2e/downloads-v2.spec.ts` (MODIFIED — Table view E2E block)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED — story → in-progress)

## Change Log

| Date       | Change                                                                                                                                                                          |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-03 | Story created (dev-authored follow-up, Amelia) to formalize the D7 Table view deferred by ux3-4-3b. FE-only, no split; reuses all ux3-4-3 plumbing (actions/SSE/status/sort/selection); adds `DownloadsTableV2` + List\|Table toggle + table skeleton. No new contract ack. Status → ready-for-dev. |
| 2026-07-03 | Implemented (dev-story, Amelia). useDownloadsView (localStorage) + List\|Table toggle (desktop-only via useIsDesktop) + DownloadsTableV2 (semantic table, aria-sort headers wired to shared sortField/sortOrder, persistent checkbox col + header 全選, Mono numerics) + extracted shared DownloadRowActions (used by card + table, no fork) + DownloadsTableSkeletonV2. Reuses useDownloads/useDownloadProgress SSE (one EventSource) + pagination + batch bar. 10 Vitest + 1 E2E (7 total in downloads-v2 spec); build/lint/test/lint:all/prettier green; jsx-a11y clean. A11y PASS, AC-Drift NONE, Contract-Stamps NONE, Rule-23 N/A. Status → review. On merge → 4-4 done + close the ux3-downloads-v2 epic. |
