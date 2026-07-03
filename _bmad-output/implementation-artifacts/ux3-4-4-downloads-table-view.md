# Story ux3-4-4 — Downloads v2 desktop Table view (D7) + List|Table toggle

Status: ready-for-dev

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

- [ ] (AC #1) Add the List|Table toggle to `DownloadsBrowseV2`'s toolbar + a `useDownloadsView()` (or local state) backed by localStorage (`vido:downloads:view`, default `list`); hide the toggle + force List below the desktop breakpoint.
- [ ] (AC #2, #7) Build `DownloadsTableV2` (`components/downloads/`) — semantic `<table>` to the `D7-D-v2` columns; reuse `downloadStatus` + `formatters`; token-only, Mono numerics; overflow-x container.
- [ ] (AC #3) Sortable `<th>` headers wired to the shared `sortField`/`sortOrder` (lift the state so both the List sort control and the Table headers drive it); `aria-sort` + ↑/↓ on the active header.
- [ ] (AC #4) Persistent checkbox column + header 全選/clear, reusing the `ux3-4-3b` selection state + batch bar (the bar renders above the table too).
- [ ] (AC #5) Row action cell (pause/resume + remove-with-confirm) reusing `useDownloadActions` + the Radix dialog.
- [ ] (AC #6) Wire Table view to render inside `DownloadsBrowseV2` (same data, SSE, pagination); build the **table-shaped skeleton** variant; verify empty + qBT-fail-soft still render.
- [ ] (AC #8) Vitest (toggle+persist, table render/sort/aria-sort, checkbox/全選, row actions) + E2E Table block (route-interception, no self-skips); `nx build`/`nx lint` web green.

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

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

### File List

_(to be filled by dev agent)_

## Change Log

| Date       | Change                                                                                                                                                                          |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-03 | Story created (dev-authored follow-up, Amelia) to formalize the D7 Table view deferred by ux3-4-3b. FE-only, no split; reuses all ux3-4-3 plumbing (actions/SSE/status/sort/selection); adds `DownloadsTableV2` + List\|Table toggle + table skeleton. No new contract ack. Status → ready-for-dev. |
