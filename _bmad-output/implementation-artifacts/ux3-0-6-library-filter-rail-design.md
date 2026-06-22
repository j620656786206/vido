# Story ux3-0-6 — Desktop library filter rail design (`.pen` flow-i)

**Epic:** ux3-foundation (UX Redesign Phase 3, reopened) · **Status:** done (design landed + MCP-reviewed, PR #89)
**Owner:** ux-designer (Claude, Pencil MCP) · **Type:** design · **FRs:** PH3-F3 (D2 — library filter)

## Story

As the design system,
I want the desktop library filter redrawn as a left rail in v2,
So that dev builds the rail against a spec (per-flow recipe step 1) instead of the
current mobile bottom-sheet misused on desktop.

## Problem

`LibraryBrowseV2` renders `LibraryFilterSheetV2` (a Base UI Dialog hardcoded
`fixed inset-x-0 bottom-0 max-h-[85vh]` + drag handle) on **all** viewports. On a 1440
desktop the sheet is a full-bleed empty panel with a meaningless drag handle. Mobile is
correct; desktop is wrong. The stale ref `flow-i/i1-d.png` (2026-06-05, PR #31) predates
the v2 Design System (2026-06-16) — concept only (desktop has a left rail), visuals
re-derived from scratch.

## What landed (in `ux-design.pen`, flow-i-advanced-search)

Built via Pencil MCP from the **current v2 Design System** (`design-language-v2` `V2Kez`,
`navigation-shell-v2` `CLo58`), token-only, CJK Noto Sans TC + numeric JetBrains Mono.

- **`I5-D` (`YEqii`)** — rail persistent (hero). v2 nav shell + 264px filter rail
  (`類型 / 類別 / 年份 / 未匹配` — exactly `FilterPanel`'s fields) + toolbar (result count ·
  `SortSelector` · grid/list toggle) + active-filter chip row above the grid + 4-col grid.
- **`I6-D` (`SPMwD`)** — rail collapsed: toolbar shows a `篩選 (n)` button, grid reflows to
  6 columns.
- **`I7-D` (`m3yZy`)** — rail states spec: genre **loading** skeleton (reduced-motion safe,
  static), genre **load-failed** fail-soft (inline 紅框 + 重試, rest of rail usable),
  filtered **no-results** empty state (`清除篩選` CTA).

## Design decisions (finalized — these drive ux3-0-7)

1. **Apply semantics:** instant apply + URL sync, **no apply/reset buttons** on desktop;
   only `清除全部`. (Mobile bottom sheet keeps the batch `套用/重置`.)
2. **Sort location:** content-area top toolbar (NOT the rail) — sort ≠ filter; matches the
   shipped A3p-D toolbar.
3. **Persist/collapse:** persistent at `lg`+; collapsible. Width **264px**. Collapsed = rail
   hidden, toolbar gains `篩選 (n)`; grid reflows wider.
4. **Active chips:** a row above the grid (FilterChip, single ✕ to remove); trailing
   `清除全部`. Visible whether the rail is open or collapsed.
5. **States:** every section fail-soft — genre loading skeleton / genre load-failed inline
   retry / filtered no-results empty state.
6. **Grid columns:** rail open `lg:3 / xl:4 / 2xl:5`; rail collapsed `lg:4 / xl:6`. gap-16.

**Visual hierarchy:** nav shell = `$bg-secondary` (elevated, primary IA); filter rail =
`$bg-primary` flush with content + right hairline `$border-subtle` (a subordinate
second-level sidebar) — clearly distinct from the nav per `navigation-shell-v2`.

## Review (MCP) — PASS

- Reuses `FilterPanel`'s field set only (`類型/類別/年份/未匹配`) + `SortSelector` — **dropped
  i1-d's invented `地區` / `最低評分`** (not in `FilterPanel`).
- Token-only colors (`$accent-subtle` + `$accent-text` + `$accent-primary` border for
  selected; `$text-secondary` for clear — not i1-d's alarming red). DM Sans → Noto Sans TC
  for all CJK; counts/badges JetBrains Mono.
- Rail visually distinct from the v2 nav shell; coexists as a second-level sidebar.

## Reconciliation note (for ux3-0-7)

The shipped v2 library page **A3p-D filters via toolbar dropdown-chips** (`類型/年份/解析度`),
not a rail. The full `FilterPanel` (multi-select `類別`, `未匹配`) doesn't fit that toolbar
model → the rail carries it; the **collapsed rail = toolbar `篩選` button** reconciles with
A3p-D's chip model. ux3-0-7 should decide whether A3p-D's toolbar chips become quick-filter
mirrors of rail state.

## Close-out

- Delivered in **PR #89** (`feat(ux-i-rail)`, squash `68eb939`). `ux-design.pen` +3558/−0
  (purely additive — no existing v2 work clobbered).
- Screenshots regenerated via `scripts/export-pen-screenshots.py` (`SCREENS` extended with
  `i5-d`/`i6-d`/`i7-d`); only the genuinely-new PNGs committed (re-render noise discarded).
- **Next:** ux3-0-7 — dev builds the rail in code (`lg:` rail, instant-apply + URL sync,
  collapse, chips, three states), keeping the `<lg` bottom sheet unchanged.
