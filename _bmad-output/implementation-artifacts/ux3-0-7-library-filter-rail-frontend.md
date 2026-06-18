# Story ux3-0-7 — Desktop library filter rail (frontend)

**Epic:** ux3-foundation (UX Redesign Phase 3, reopened) · **Status:** done (CR passed — M1/M2/M3 fixed; 53 tests green, build+lint green)
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

1. [x] `FilterPanel.tsx` — added an **instant** mode (`instant` prop: apply `onApply` per
   change, hides `套用/重置` + the panel title) vs the existing batch mode for the sheet. All
   field logic reused; no new filter logic.
2. [x] New `LibraryFilterRail.tsx` wrapping `FilterPanel instant` + rail header (`篩選` +
   active-count badge + collapse toggle) + pinned `清除全部`; `$bg-primary` + right hairline;
   genre list scrolls inside the rail (UX watch-out #1).
3. [x] `LibraryBrowseV2.tsx` — renders the rail at `lg`+ (`hidden lg:block`) and keeps the
   bottom sheet `<lg` (`lg:hidden` trigger); `SortSelector` already in the toolbar; collapse →
   toolbar `篩選 (n)` button (`library-rail-expand`); grid reflows `lg:3/xl:4/2xl:5` ↔
   `lg:4/xl:6` by rail state; collapse persisted to `localStorage`.
4. [x] Active-filter chip row above the grid — pre-existing `FilterChips` retained (single
   remove + `清除全部`).
5. [x] URL sync (instant) — `applyFilters`/`patchSearch` already write `genres/yearMin/
   yearMax/unmatched` to the URL on every change; the route `validateSearch` coerces params
   (genres-by-name = string, year = number) so Rule 26 single-value drops don't apply here.
6. [x] States — genre loading skeleton (`filter-genre-loading`, reduced-motion safe) +
   load-failed inline retry (`filter-genre-error`/`-retry`, fail-soft) in `FilterPanel`;
   filtered no-results (`LibraryNoResultV2`) pre-existing.
7. [~] Tests — **Vitest 41/41 green** (FilterPanel 28: +instant apply/de-select + hidden
   actions + genre loading/error/retry; LibraryBrowseV2 10: +rail renders, collapse→expand +
   grid reflow, re-expand, active-count; LibraryFilterRail 3). `nx build web` green;
   `nx lint web` 0 errors (my files). **E2E: deferred** — no `new_shell_enabled`-ON v2-library
   Playwright harness exists yet (`discover-filters.spec` is `/discover`, a different route);
   the v2 rail's felt behaviour is the **P10 browser-verify** gate. CI's existing E2E suite is
   unaffected (zero overlap; legacy `/library` `FilterPanel` batch path preserved).

## Dev Agent Record

**Agent:** Amelia (BMM dev-story). **Branch:** `feat/ux3-0-7-library-filter-rail` (off main
incl. #87/#89/#90).

**Implemented:**
- `FilterPanel.tsx` — `instant` prop (controlled off `filters`; per-change `onApply`; hides
  `套用/重置` + title); genre section now branches loading-skeleton / error+retry / chips
  (reused `useLibraryGenres` `isLoading/isError/refetch`). Min-44px touch targets on chips.
- `LibraryFilterRail.tsx` — new desktop rail (264px, `$bg-primary` + right hairline; sticky;
  scrollable body with pinned clear-all footer; collapse toggle; active-count badge).
- `LibraryBrowseV2.tsx` — `lg:flex` `[rail | main]`; rail `hidden lg:block` (collapsible,
  persisted); `<lg` keeps the sheet; collapsed → toolbar `篩選 (n)`; grid `gridColsClass`
  reflow; shared `handleTypeChange`; `activeFilterCount` (genres + decade-as-one; excludes
  type=全部).

**Decisions:**
- Decade range (`yearMin`+`yearMax`) counts as ONE active facet (matches design I5-D badge
  "3" = 2 genres + 2020s); type=全部 is not a constraint (UX watch-out #3).
- Reduced-motion: skeleton uses `motion-reduce:animate-none` (UX watch-out #2; debounce of
  the re-query left to a follow-up — instant nav is already cheap client-side).
- Mobile bottom sheet untouched (correct as-is).

**Verification:** `npx vitest run` (3 files) 41/41 · `nx build web` ✓ · `nx lint web` 0
errors on changed files.

**File List:** `apps/web/src/components/library/FilterPanel.tsx` (M),
`apps/web/src/components/library/FilterPanel.spec.tsx` (M),
`apps/web/src/components/library/LibraryFilterRail.tsx` (A),
`apps/web/src/components/library/LibraryFilterRail.spec.tsx` (A),
`apps/web/src/components/library/LibraryBrowseV2.tsx` (M),
`apps/web/src/components/library/LibraryBrowseV2.spec.tsx` (M),
`apps/web/src/components/library/FilterChips.tsx` (M — CR M2),
`apps/web/src/components/library/FilterChips.spec.tsx` (M — CR M2).

## Change Log

| Date       | Change                                                                                   |
| ---------- | ---------------------------------------------------------------------------------------- |
| 2026-06-18 | Story implemented (Tasks 1–6 [x], Task 7 [~] E2E deferred). 41/41 vitest, build+lint green. |
| 2026-06-18 | Adversarial CR (Amelia): 0 High / 3 Medium / 4 Low. Fixed M1/M2/M3 + L1/L3 (option [1]).  |

### CR fixes applied (2026-06-18)

- **M1** — `類型` buttons (`FilterPanel.tsx`) + genre-error `重試` button gained `min-h-[44px]`;
  the final AC (all rail controls ≥44px) was PARTIAL, now fully met. Regression test added
  (`FilterPanel.spec` — type controls ≥44px).
- **M2** — `FilterChips` now renders a full decade range (both bounds set) as ONE combined
  chip (`YYYY–YYYY 年`, single ✕) so the chip row matches the rail's decade-as-one badge (AC
  #4/#7). New optional `onRemoveYears` prop does an **atomic** clear (single navigate, no
  two-step race); `LibraryBrowseV2` wires it; half-open ranges keep their single-bound chip.
  `FilterChips.spec` updated + 1 new test.
- **M3** — added this Change Log (was missing — story-hygiene gap).
- **L1** — rail sticky height `calc(100vh-5rem)` → `calc(100vh-4rem)` to match `top-16` (no
  16px bottom gap).
- **L3** — `FilterPanel` instant mode now guards the local-state sync `useEffect` (instant
  reads `filters` directly; the dead updates are skipped).
- **Still open (accepted):** L2 — instant re-query debounce (UX watch-out #2, follow-up); L4
  — `railCollapsed` persistence is desktop-only semantics but visually inert `<lg` (shared
  `grid-cols-2 sm:grid-cols-3`). Post-fix: **vitest 53/53** (FilterChips 9→11, FilterPanel
  28→29), `nx build web` ✓, `nx lint web` 0 errors.

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
