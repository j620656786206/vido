# Story ux3-3-2 — Discover v2 frontend (persistent instant filter rail)

Status: ready-for-dev

**Epic:** ux3-discover-v2 (UX Redesign Phase 3) · **Type:** frontend · **FRs:** PH3-M2, PH3-R2
**Design:** ux3-3-1 (`.pen` `flow-i-discover-v2`, PR #94) · **Owner:** dev (`dev-story`) → tea (visual + E2E)

## Story

As a NAS owner discovering titles across my library + TMDB,
I want `/discover` migrated to the v2 shell with the **persistent instant filter rail** from the
ux3-3-1 design,
so that Discover matches the v2 design language and the 媒體庫 Library rail — one consistent,
instant, always-visible power-filter — without regressing the instant behavior it already ships.

## Context — restyle + refine an ALREADY-instant rail (not a rebuild)

The shipped `/discover` (`apps/web/src/routes/discover.tsx:108-138`) is **already** a persistent
264px instant-apply desktop sidebar (`FilterPanel`) with genre/year/region/rating/platform **live**,
a results column (`PresetChips` → `FilterChipBar` → `SearchResults`), a mobile batch `FilterBottomSheet`,
and `SavePresetDialog`. It is **not yet under the v2 shell** (no `staticData: { shell: 'v2' }` — unlike
`activity.tsx:9` / home). So this story is a **v2 restyle + 7 targeted refinements**, NOT a rebuild, and
NOT a behavior change to batch (the ux3-3-1 adversarial UX panel + shipped-code audit settled on the
**persistent instant rail**; a batch popover would be a regression).

## Acceptance Criteria

1. **Shell-version gating.** `/discover` gets `staticData: { shell: 'v2' }` and renders under the v2
   shell (`HomeSidebar-v2`, 探索 active) when `new_shell_enabled`; the legacy shell render is byte-unchanged
   under the flag OFF (same shell-gating pattern as `ux3-1-2` home / `ux3-2-3` activity). 探索 stays in the
   mobile **More** sheet (no bottom-4 tab — ADR D1-b).

2. **Desktop FilterPanel → v2 persistent rail.** Restyle the `lg+` `FilterPanel` sidebar to the
   ux3-3-1 rail: 264px, `$bg-primary`, right `$border-subtle`, `篩選` header + active-count badge +
   **collapse chevron** → `篩選(n)` collapsed state that reflows the grid wider (mirror
   `LibraryFilterRail` chrome — converge, don't fork the look). Token-only, Noto Sans TC, 44px targets.

3. **All 5 dimensions live + per-facet counts.** Genre / year / rating / region / streaming-platform stay
   live (confirm the `/discover` params: `vote_gte` rating-min, `region`, `with_watch_providers` — all
   backend-backed per the ux3-3-1 capability audit; **if any is NOT actually queryable, demote only that
   one** to a disabled「即將推出」, do not demote the rest). Each facet value shows a **result count** in
   JetBrains Mono; reuse the **enabled-gated draft-count infrastructure** already built for the mobile sheet
   (`FilterBottomSheet` `套用篩選（${totalResults} 部結果）` / `isCounting`) rather than firing N extra
   uncoalesced count queries.

4. **Debounce numeric inputs.** Year-range (and any score numeric) inputs debounce (~300–400ms) before
   committing — `FilterPanel` currently calls `onChange` per keystroke (typing `1995` = 4 queries).
   Categorical chips (genre/region/rating/platform/sort) stay **instant**.

5. **History hygiene.** `useFilterState.setFilters` (`apps/web/src/hooks/useFilterState.ts:31-42`) navigates
   with **`replace: true`** for intermediate filter toggles so a multi-dimension compose does not push every
   half-built combo onto the browser history stack (Back leaves Discover, not your own toggles).

6. **Coalesce `type='all'` double-fire.** `useDiscoverResults` (`hooks/useDiscoverResults.ts:30-41`) fires
   two queries (`discoverMovies` + `discoverTVShows`) per change; treat the pair as **one logical refresh**
   for loading/skeleton state so a single toggle does not visibly double-flash.

7. **Demote the chip bar to a read/remove summary.** With the rail as the primary editor, `FilterChipBar`
   becomes a **lighter read/remove summary + 清除全部** above the grid (not a second competing editor), per
   the ux3-3-1 design and the triple-surface resolution.

8. **Four states (N4) to v2.** Loading skeleton (rail-shaped + grid), **no-result distinct from empty**
   (`找不到相符的結果` + active-filter echo + 清除篩選 / 調整搜尋), and **per-section fail-soft** (TMDB section
   degrades inline `無法載入…媒體庫結果不受影響` + 重試; local results still render; page never hard-fails) —
   all match ux3-3-1 frames I6/I7/I8.

9. **Mobile keeps batch.** The `lg-` `FilterBottomSheet` stays a **batch** sheet (`套用篩選（N 部結果）`),
   restyled to v2 (radius-xl, overlay-scrim) — a transient sheet = batch is correct and matches ux3-3-1
   `I4-M-v2`. Reachable via `篩選` button; 探索 via More.

10. **`vote_average` on result cards.** Discover result cards show the rating (the design puts ★ + rating on
    every `PosterCardV2`); confirm the Discover results carry `vote_average` (TMDB discover returns it) —
    v2-followups gap was the LIBRARY list query, not discover, but verify end-to-end.

11. **Reuse, no fork.** Reuse `FilterPanel` / `FilterChipBar` / `PresetChips` / `FilterBottomSheet` /
    `SavePresetDialog` / `useFilterState` / `useDiscoverResults` / `useLibraryGenres` / `discoverFilters.ts`
    (no new filter engine, no new BE expected — confirm). Where the rail chrome overlaps `LibraryFilterRail`,
    converge rather than duplicate.

12. **Tests.** Vitest covers: shell-gating render, the rail (facet counts, collapse, instant categorical +
    debounced numeric), `replace:true` history, coalesced `type='all'` loading, chip-bar-as-summary, and the
    three non-default states. **E2E reuses `tests/support/helpers/seed-helpers.ts` real seeding — NO
    data-dependent `test.skip` self-skips** (Epic 20 lesson); `discover-filters.spec` extended for the v2 rail.
    `nx build web` + `nx lint web` green.

## Tasks / Subtasks

- [ ] (AC #1) Add `staticData: { shell: 'v2' }` to the `/discover` route + render under `HomeSidebar-v2`
      (探索 active); shell-gate so flag-OFF legacy is byte-unchanged. Confirm 探索-via-More on mobile.
- [ ] (AC #2, #7) Restyle the desktop `FilterPanel` sidebar to the v2 rail (264px, header+badge+collapse →
      `篩選(n)`, grid reflow); demote `FilterChipBar` to a read/remove summary. Converge with `LibraryFilterRail`.
- [ ] (AC #3) Wire per-facet counts via the enabled-gated draft-count infra; confirm `vote_gte`/`region`/
      `with_watch_providers` params (demote only an unbacked one). Mono counts.
- [ ] (AC #4) Debounce numeric year/score inputs in `FilterPanel`; keep categorical chips instant.
- [ ] (AC #5) `useFilterState.setFilters` → `navigate({ ..., replace: true })` for intermediate toggles.
- [ ] (AC #6) Coalesce the `type='all'` movies+tv pair into one logical loading state in `useDiscoverResults`.
- [ ] (AC #8) Build the three v2 states (skeleton / no-result-distinct / per-section fail-soft).
- [ ] (AC #9) Restyle `FilterBottomSheet` to v2 (keep batch + draft count).
- [ ] (AC #10, #11) Confirm `vote_average` reaches the cards; verify no new BE needed (reuse Epic 11 engine).
- [ ] (AC #12) Vitest + extend `discover-filters` E2E (seed-helpers, no self-skips); `nx build`/`nx lint` web green.

## Dev Notes

### Cross-stack split check (MANDATORY) — NO split

Backend tasks: **0–1** (confirm-only — `vote_gte`/`region`/`with_watch_providers` are already backend-backed
per the ux3-3-1 audit; `vote_average` comes from TMDB discover; add a column to a response ONLY if verification
shows it missing). Frontend tasks: **~9** (shell-gate, rail restyle, facet counts, debounce, replace:true,
coalesce, chip-bar summary, four states, mobile sheet, tests). Backend ≤ 3 → **single frontend story, no split.**

### Source tree (real symbols — do not invent)

- Route: `apps/web/src/routes/discover.tsx` (`:108-138` desktop rail + results; `:140-152` mobile sheet + save dialog).
- Filter: `apps/web/src/components/search/FilterPanel.tsx` (genre/region/year/rating/platform/sort; `:53-56`
  year `setYear` per-keystroke → debounce target).
- Chip bar / presets / sheet / save: `components/search/{FilterChipBar,PresetChips,FilterBottomSheet,SavePresetDialog}.tsx`
  (`FilterBottomSheet:126` draft-count `套用篩選（${totalResults} 部結果）` + `isCounting` = reuse for facet counts).
- State: `hooks/useFilterState.ts` (`:31-42` `setFilters` → add `replace:true`; `:44` `clearAll`).
- Results: `hooks/useDiscoverResults.ts` (`:30-41` two queryKeys movies+tv → coalesce); `lib/discoverFilters.ts`
  (`GENRE_FILTER_OPTIONS`, `REGION_OPTIONS`, `PLATFORM_OPTIONS`, `RATING_OPTIONS`, `SORT_OPTIONS`,
  `buildDiscoverParams`).
- Rail chrome reference (converge): `apps/web/src/components/library/LibraryFilterRail.tsx` (ux3-0-7 — the
  shipped persistent instant rail this should look like); genres `hooks/useLibrary*` → `useLibraryGenres`.
- Shell-gate pattern: `routes/activity.tsx:9` (`staticData: { shell: 'v2' }`); `ux3-1-2` home shell-version gate.

### Design reference (ux3-3-1, flow-i-discover-v2)

- Rail expanded `I1-D-v2` · collapsed `I4-D-v2` · suggestions `I3-D-v2` · save `I5-D-v2` · skeleton `I6-D-v2`
  · no-result `I7-D-v2` · fail-soft `I8-D-v2` · mobile `I2-M-v2` / `I4-M-v2`. Design Language v2 §7 FilterRail
  catalog; Component Library `filter-controls-v2` (FacetCountChip). Screenshots `_bmad-output/screenshots/flow-i-discover-v2/`.

### Project Structure Notes

- Touches `apps/web/src/{routes/discover.tsx, components/search/*, hooks/useFilterState.ts, hooks/useDiscoverResults.ts}`
  + tests. No new route; `/discover` already exists. Shell-gated — legacy path preserved under flag OFF.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` /
  `new Date()` / `Date.UTC()` / `Date.parse()`?** Expected **NO** — Discover is a filter/grid surface
  (no wall-clock-derived UI). **Verify during dev**; if a touched component reads wall-clock time, capture
  ≥2 fixture baselines per Rule 23 and pair with a `Clock-mocked` / `Time-bomb-exempt` marker. Else state
  `N/A — no wall-clock-reading components touched`.
- Reference: `project-context.md` Rule 23.

### Discovery Triage

- **Carried in from ux3-3-1 (design):**
  - **① expand-scope-in-place** — confirm `vote_gte`(rating) / `region` / `with_watch_providers` queryability
    (AC #3) + `vote_average` on discover cards (AC #10): tracked by ACs in THIS story. If verification finds an
    unbacked param → **② spawn-blocking-story** (file a backend `sprint-status` entry; demote that one dim).
  - No other out-of-scope work anticipated; if found, triage into a lane before marking done (Rule 24).

### References

- [Source: _bmad-output/implementation-artifacts/ux3-3-1-discover-design.md#Close-out (rail decision + ux3-3-2 ACs)]
- [Source: apps/web/src/routes/discover.tsx#108-152]
- [Source: apps/web/src/components/search/FilterPanel.tsx]
- [Source: apps/web/src/hooks/useFilterState.ts#31-44]
- [Source: apps/web/src/hooks/useDiscoverResults.ts#30-41]
- [Source: apps/web/src/components/library/LibraryFilterRail.tsx (ux3-0-7 rail chrome to converge with)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§7 four-state, §6 shell]

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List
