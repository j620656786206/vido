# Story ux3-0-7 — Desktop library filter rail (frontend)

**Epic:** ux3-foundation (UX Redesign Phase 3, reopened) · **Status:** ready-for-dev
**Owner:** Dev (Amelia) · **Type:** frontend · **FRs:** PH3-F3 (D2 — library filter)
**Design ref:** ux3-0-6 (`.pen` flow-i `I5-D`/`I6-D`/`I7-D`, PR #89)

## Story

As a desktop user,
I want a persistent left filter rail in the library,
So that I filter without a meaningless full-bleed bottom sheet, while mobile keeps its
correct bottom sheet.

## Acceptance Criteria

**Given** the v2 library (`LibraryBrowseV2`) at `lg`+ (≥1024px),
**When** rendered,
**Then** a 264px left filter rail shows the existing `FilterPanel` fields (`類型 / 類別 /
年份 / 未匹配`) as a second-level sidebar (`$bg-primary` + right hairline, distinct from the
`$bg-secondary` nav shell). The rail is NOT the `Sheet`. (Design `I5-D`.)

**Given** a viewport `<lg` (<1024px, incl. tablet),
**When** the filter is opened,
**Then** the existing `LibraryFilterSheetV2` bottom sheet renders **unchanged** (batch
`套用/重置`). No rail below `lg`. (Out of scope to alter the sheet.)

**Given** the desktop rail,
**When** any filter control changes,
**Then** it applies **instantly** (re-query) and **syncs to the URL** — no `套用/重置`
buttons on desktop; only `清除全部`. Back/forward and shareable links reproduce the filter
state. (Decision 1.)

**Given** filter state in the URL,
**When** parsed,
**Then** numeric-looking params are coerced per **Rule 26** (TanStack search-param
coercion — `toCsvString`/`String()` for genres + decade ranges) so single-value deep links
(`?genre=16`) are not silently dropped.

**Given** sort,
**When** the user changes it,
**Then** `SortSelector` lives in the content-area top toolbar (not the rail). (Decision 2.)

**Given** the rail,
**When** the user collapses it,
**Then** the rail hides, the toolbar shows a `篩選 (n)` button (n = active filter count),
and the grid reflows from `lg:3 / xl:4 / 2xl:5` (rail open) to `lg:4 / xl:6` (collapsed),
gap-16. Re-expanding restores the rail. (Decisions 3 + 6.)

**Given** active filters,
**When** any are set,
**Then** a chip row above the grid shows each as a removable `FilterChip` (single ✕) with a
trailing `清除全部`; visible whether the rail is open or collapsed. (Decision 4.)

**Given** the genre (`類別`) list async-loads via `useLibraryGenres`,
**When** loading → a static skeleton (reduced-motion safe); **on error** → an inline
fail-soft `類別載入失敗 · 重試` block (the rest of the rail stays usable); **when filters
yield zero results** → the grid shows a no-results empty state with a `清除篩選` CTA.
(Decision 5.)

**Given** all rail controls,
**Then** touch targets are ≥44px, colors are token-only (`var(--…)`), and transitions are
`prefers-reduced-motion` friendly.

## Tasks

1. [ ] `FilterPanel.tsx` — support an **instant** mode (apply `onApply` per change, hide the
   `套用/重置` actions) vs the existing batch mode for the sheet. Reuse all field logic
   (type/genre/decade/unmatched) — no new filter logic.
2. [ ] New `LibraryFilterRail` (desktop) wrapping `FilterPanel` (instant) + rail header
   (`篩選` + active-count badge + collapse toggle) + `清除全部`; `$bg-primary` + right
   hairline; sections use the v2 `SidebarGroupLabel` type idiom.
3. [ ] `LibraryBrowseV2.tsx` (:243 filter button, :350 `LibraryFilterSheetV2`) — render the
   rail at `lg`+ and keep the bottom sheet `<lg`; move `SortSelector` to the toolbar; add the
   `篩選 (n)` collapse affordance; reflow grid columns by rail state.
4. [ ] Active-filter chip row above the grid (`FilterChip` per active facet, single-remove +
   `清除全部`).
5. [ ] URL sync (instant) for `type / genres / yearMin-yearMax (decades) / unmatched` with
   Rule-26 coercion; rail state ↔ URL ↔ query single source of truth.
6. [ ] States: genre loading skeleton / genre load-failed inline retry / filtered no-results
   empty state.
7. [ ] Tests — Vitest: rail renders at `lg` & sheet `<lg`; instant-apply + URL sync;
   collapse → `篩選 (n)` + column reflow; chip remove/clear-all; three states. E2E
   (`discover-filters` desktop): assert the **rail** (not the sheet) + chips + URL (the
   current desktop cases only assert chips/URL, missing the rail). web suite green;
   `nx build web` green; `nx lint web` 0 errors.

## Dev notes

- **Discovery Triage (Rule 24 ③):** this story resolves the carried-forward gap
  `v2-followups` #1 (desktop filter = mobile bottom-sheet). Bidirectional link recorded.
- **Shell-gated:** lives in the v2 `LibraryBrowseV2` path (behind `new_shell_enabled`); the
  legacy `LibraryPage` is untouched.
- **Sequence after PR #87** (`ux3-2-3-activity-frontend`) merges — #87 edits
  `shell/AppSidebar.tsx` + `navModel.ts`; the rail is a content-level second sidebar
  (independent of `AppSidebar`, no direct conflict), but branch off the latest main so the
  rail builds against the final shell.
- **Mobile bottom sheet is out of scope** — its design didn't change (correct as-is); only
  add the desktop rail.
- **Reuse, don't reinvent:** `FilterPanel` (fields), `SortSelector` (toolbar), `FilterChip`
  / `GenreTag` / `Checkbox` v2 components, `useLibraryGenres`.
- Design ref: `ux-design.pen` flow-i `I5-D`/`I6-D`/`I7-D` (PR #89) + ux3-0-6.
- **Browser-verify (P10)** is the human gate at 390 / 768 / 1440 — rail at lg, sheet below,
  instant-apply feel, collapse reflow, deep-link reproduction.

### UX review notes (Sally, 2026-06-17) — PASS with 3 watch-outs

1. **Long genre lists must scroll inside the rail** — real `useLibraryGenres` may return
   >12 genres; the rail content area scrolls internally (like the nav shell) with the
   `清除全部` footer pinned, so it never gets pushed off-screen. (Add to Task 2.)
2. **Debounce instant-apply** — rapid genre toggling should debounce the re-query so the grid
   doesn't thrash; honor `prefers-reduced-motion` on the reflow. (Task 1/5.)
3. **Active-count badge excludes `類型=全部`** — the `(n)` counts only constraining facets
   (selected genres + decade + unmatched), NOT the `全部`/all default. (Task 2/4.)

### Architecture review (Winston, 2026-06-17) — PASS, no ADR

- **No new ADR.** Reuses TanStack Router search params + **Rule 26** coercion (existing) and
  the existing library List query (genre/year/unmatched supported since Epic 5 `5-5` +
  `ux3-0-1`) — **FE-only, zero backend/API/repo change**. (`FullTextSearch` `vote_average`,
  v2-followups gap #2, is a SEPARATE story — not here.)
- **Single source of truth:** the desktop rail and the mobile sheet MUST bind to the SAME
  filter state (the route's `validateSearch` schema → query); do NOT fork filter state per
  breakpoint — URL is canonical. Mirrors the P7 "one shared grid, no fork" precedent of
  `ux3-0-5`.
- **One `FilterPanel`:** instant vs batch is a mode prop on the SAME component — never
  duplicate field logic across rail/sheet.
- **Breakpoint render:** prefer a single mounted filter tree; if rail↔sheet is swapped by
  viewport, ensure only one is active so there is a single URL writer.
