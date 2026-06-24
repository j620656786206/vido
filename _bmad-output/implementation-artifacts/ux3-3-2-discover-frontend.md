# Story ux3-3-2 — Discover v2 frontend (persistent instant filter rail)

Status: done

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

3. **All 5 dimensions live + single live total.** Genre / year / rating / region / streaming-platform stay
   live (confirm the `/discover` params: `vote_gte` rating-min, `region`, `with_watch_providers` — all
   backend-backed per the ux3-3-1 capability audit; **if any is NOT actually queryable, demote only that
   one** to a disabled「即將推出」, do not demote the rest). The rail shows ONE **live total result count**
   (`符合 N 部`, JetBrains Mono) reusing the **enabled-gated draft-count infrastructure** already built for
   the mobile sheet (`FilterBottomSheet` `套用篩選（${totalResults} 部結果）` / `isCounting`) — zero extra
   uncoalesced count queries.
   > **[amended — Party Mode 2026-06-23, Alexyu, decision 1A]** Originally specced as *per-facet* counts
   > (a count next to every chip). TMDb `/discover` returns only `total_results` (no facet aggregation —
   > verified vs `apps/api/internal/tmdb/types.go`), so true per-facet counts would cost ~N×2 uncoalesced
   > queries (banned by Rule 27). Shipped a single live total instead; per-facet counts deferred to a future
   > backend aggregation endpoint (`sprint-status.yaml` → `ux3-discover-facet-aggregation-be`). The `.pen`
   > I1-D-v2 design was realigned to the single-total to keep design↔build in sync.

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

- [x] (AC #1) Add `staticData: { shell: 'v2' }` to the `/discover` route + render under `HomeSidebar-v2`
      (探索 active); shell-gate so flag-OFF legacy is byte-unchanged. Confirm 探索-via-More on mobile.
- [x] (AC #2, #7) Restyle the desktop `FilterPanel` sidebar to the v2 rail (264px, header+badge+collapse →
      `篩選(n)`, grid reflow); demote `FilterChipBar` to a read/remove summary. Converge with `LibraryFilterRail`.
- [x] (AC #3) Wire a single live total (`符合 N 部`) via the enabled-gated draft-count infra; confirm
      `vote_gte`/`region`/`with_watch_providers` params (demote only an unbacked one). Mono count.
      *(Per-facet counts renegotiated → single total, decision 1A; per-facet deferred to backlog.)*
- [x] (AC #4) Debounce numeric year/score inputs in `FilterPanel`; keep categorical chips instant.
- [x] (AC #5) `useFilterState.setFilters` → `navigate({ ..., replace: true })` for intermediate toggles.
- [x] (AC #6) Coalesce the `type='all'` movies+tv pair into one logical loading state in `useDiscoverResults`.
- [x] (AC #8) Build the three v2 states (skeleton / no-result-distinct / per-section fail-soft).
- [x] (AC #9) Restyle `FilterBottomSheet` to v2 (keep batch + draft count).
- [x] (AC #10, #11) Confirm `vote_average` reaches the cards; verify no new BE needed (reuse Epic 11 engine).
- [x] (AC #12) Vitest + extend `discover-filters` E2E (seed-helpers, no self-skips); `nx build`/`nx lint` web green.

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

Claude Opus 4.8 (1M context) — BMAD dev agent (Amelia), `dev-story` workflow. Party Mode (architect/ux/tea/sm/pm) facilitated the AC #3 decision.

### Completion Notes List

**Decisions (Party Mode 2026-06-23, Alexyu):** `1A` single live total · `2改` update `.pen` · `3照辦` blueprint.

- **🔗 AC Drift: FOUND** — Story 11-2 AC #4 (back/forward steps through each filter toggle — *push* history) → ux3-3-2 AC #5 (intermediate toggles use `replace:true`). **Contained to the v2 shell:** `useFilterState` gates `replace` on `useShellVersion()`, so the legacy shell keeps 11-2 push semantics **byte-unchanged** (legacy E2E `[P0] browser back restores previous filter state` still valid). Epic 11 is pre-Rule-20 (implicit v0), no contract ack required. (Reference: `_bmad-output/implementation-artifacts/11-2-persistent-filter-chip-ui.md` — AC drift reference.)
- **📎 Contract Stamps: NONE** (no `[@contract-v*]` in this story or upstream Epic 11 refs — pre-Rule-20; this story defines/consumes no wire contracts).
- **🎭 A11y Pre-Flight: PASS** (new components: rail collapse/expand buttons carry `aria-label` 收合篩選/開啟篩選; single-total has `aria-live=polite`; per-section error is `role=alert`; inert Requests entry is `disabled`+`aria-disabled`. jsx-a11y lint clean on touched files, 0 introduced; `nx lint web` green).
- **🎨 UX Verification: PASS (aligned)** — see table below. The single intentional deviation (per-chip counts → single total, decision 1A) was resolved by updating `.pen` I1-D-v2 (decision 2改).
- **AC #3 resolution:** TMDb `/discover` returns only `total_results` (NO facet aggregation — verified official docs + `apps/api/internal/tmdb/types.go` `SearchResultMovies`). True per-facet counts would cost ~N×2 uncoalesced queries (banned by AC #3 + Rule 27 rate limit). Shipped a SINGLE live total `符合 N 部` reusing `useDiscoverResults().totalResults` — **zero extra queries**.
- **AC #3/#10 backend verification:** `vote_gte` / `region` / `watch_providers` / `watch_region` are all backend-backed (`apps/api/internal/handlers/tmdb_handler.go` + `internal/tmdb/movies.go`) → **NO dimension demoted to 即將推出**. `vote_average` reaches cards via `MediaGrid`→`PosterCard` (already shipped).
- **Legacy byte-unchanged (AC #1):** all v2 refinements are opt-in so the legacy render is unchanged — `replace` gated on shell; `debounceMs` / `summary` / `variant='v2'` / `keepPrevious` are props the legacy path never passes.
- **Discovery Triage (Rule 24):** ③ `ux3-discover-facet-aggregation-be` filed in `sprint-status.yaml` (true per-facet counts via a future BE aggregation endpoint). ① inert `想要清單 · 即將推出` Requests entry added to the v2 toolbar (PH3-R2 reserve; `disabled`).
- **🕐 Time-dependent visual coverage:** N/A — no touched `apps/web/src/components/**` file reads wall-clock time (Discover is a filter/grid surface).
- **.pen (decision 2改):** I1-D-v2 (`fxCVk`) — 25 per-facet `cnt` nodes removed + `符合 412 部` single-total added via Pencil MCP; `i1-d.png` regenerated (only the genuinely-changed PNG staged). ⚠️ **Pencil has not flushed `ux-design.pen` to disk** (no MCP save tool) — requires **Cmd+S in Pencil**, then commit `ux-design.pen` + `_bmad-output/screenshots/flow-i-discover-v2/i1-d.png` together.
- **Tests:** web vitest **2235 pass**; `nx test api` green; `nx build web` green; `nx lint web` green; prettier clean. E2E: legacy `discover-filters.spec` preserved; new **v2-rail block** added (flag-ON via `localStorage` seed + flag-endpoint stub) — runs in CI / **P10 browser-verify** (no v2-shell-ON E2E harness yet, per the ux3-0-7 precedent).

**🎨 UX Verification — implementation vs design (I1-D-v2 / I4-D-v2 / I6/I7/I8):**

| Area | Design | Implementation | Match? |
|------|--------|----------------|--------|
| Rail chrome | 264px `$bg-primary`, right hairline, 篩選 + Mono badge + collapse chevron | `DiscoverFilterRail` (mirrors `LibraryFilterRail`) | ✅ |
| 5 dimensions | 類型/年份/評分/地區/串流平台 chips | `search/FilterPanel` (same 5, all live) | ✅ |
| Facet counts | per-chip (was `動作 340`) | single total `符合 N 部` (1A) → `.pen` updated to match | ✅ aligned |
| Active chip | `$accent-subtle` + ✓ | `FilterPanel` active chipClass | ✅ |
| Chip-bar summary | muted read/remove + 清除全部 | `FilterChipBar summary` variant | ✅ |
| Rail footer | 清除全部篩選 + rotate-ccw | same | ✅ |
| Collapsed (I4-D-v2) | 篩選(n) button, grid wider | `discover-rail-expand` + MediaGrid auto-fill reflow | ✅ |
| States (I6/I7/I8) | skeleton / no-result-distinct / per-section fail-soft | `DiscoverStatesV2` | ✅ |
| Cards | ★ + rating | `PosterCard` voteAverage | ✅ |
| Mobile (I4-M-v2) | batch sheet, radius-xl, scrim | `FilterBottomSheet variant='v2'` | ✅ |

### File List

- `apps/web/src/routes/discover.tsx` (M — `staticData: { shell: 'v2' }` + shell-gate split into `DiscoverPage`/`LegacyDiscover`)
- `apps/web/src/routes/discover.spec.tsx` (A — shell-gating tests)
- `apps/web/src/components/search/DiscoverBrowseV2.tsx` (A — CR M2: rail count reflects `isFetching`; CR L1: options-object `useDiscoverResults` call)
- `apps/web/src/components/search/DiscoverBrowseV2.spec.tsx` (A — CR M2: +1 test for 計算中… on background refetch)
- `apps/web/src/components/search/DiscoverFilterRail.tsx` (A — CR M3: now composes the shared `FilterRailShell`)
- `apps/web/src/components/search/DiscoverFilterRail.spec.tsx` (A)
- `apps/web/src/components/ui/FilterRailShell.tsx` (A — CR M3: shared persistent-rail chrome — converges 探索 + 媒體庫 rails)
- `apps/web/src/components/library/LibraryFilterRail.tsx` (M — CR M3: migrated to the shared `FilterRailShell`; pixel-identical DOM, ux3-0-7 chrome convergence per AC #11)
- `apps/web/src/components/search/DiscoverStatesV2.tsx` (A)
- `apps/web/src/components/search/DiscoverStatesV2.spec.tsx` (A)
- `apps/web/src/components/search/FilterPanel.tsx` (M — `debounceMs` prop + year local-state debounce)
- `apps/web/src/components/search/FilterPanel.spec.tsx` (M — debounce tests)
- `apps/web/src/components/search/FilterChipBar.tsx` (M — `summary` variant)
- `apps/web/src/components/search/FilterBottomSheet.tsx` (M — `variant='v2'`; CR L1: options-object `useDiscoverResults` call)
- `apps/web/src/hooks/useFilterState.ts` (M — shell-gated `replace`)
- `apps/web/src/hooks/useFilterState.spec.tsx` (M)
- `apps/web/src/hooks/useDiscoverResults.ts` (M — `keepPrevious` + coalesced `isLoading`/`isFetching`; CR L1: options-object signature `{ enabled, keepPrevious }`)
- `apps/web/src/hooks/useDiscoverResults.spec.tsx` (M — CR L1: options-object call)
- `tests/e2e/discover-filters.spec.ts` (M — v2 rail block)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (M — status→review + Rule-24 ③ backlog entry)
- `ux-design.pen` (M — I1-D-v2 single-total; ⚠️ PENDING Pencil Cmd+S to flush to disk)
- `_bmad-output/screenshots/flow-i-discover-v2/i1-d.png` (M — regenerated)

## Senior Developer Review (AI)

**Reviewer:** Alexyu (BMAD `code-review` workflow, adversarial) · **Date:** 2026-06-24 · **Outcome:** Approve (fixes applied)

**Scope gates:** Rule 7 Wire Format N/A (no Go error-code files) · Rule 20 Contract Bump N/A (no stamp bumps) ·
Rule 25 Mega-line N/A (project-context.md untouched). Git File List vs git reality: **0 discrepancies**.
`.pen` flush verified (committed diff `40+/250−` removes the per-facet nodes → Cmd+S did happen).

Findings: **0 High · 3 Medium · 3 Low**. No "marked [x] but not done" criticals — `vote_average`→cards
verified end-to-end (`MediaGrid`→`PosterCard voteAverage`), all refinements shell-gated.

**Fixed (review choice [1] — auto-fix):**

- **[M1] AC #3 wording vs reality** — AC #3 said *per-facet* counts; shipped a single live total. Amended the
  AC text + Task to record the single-total decision (Party Mode 1A) so the story file is truthful; per-facet
  stays deferred to backlog `ux3-discover-facet-aggregation-be`. (doc-only)
- **[M2] Rail count wired to `isLoading`** — under `keepPreviousData`, `isLoading` only flips on a cold load,
  so a re-filter showed the **stale** total and never `計算中…`; the hook's `isFetching` was dead code. Now
  `DiscoverBrowseV2` passes `isCounting={isFetching}` (grid skeleton still on `isLoading`). +1 regression test.
- **[M3] `DiscoverFilterRail` forked `LibraryFilterRail` chrome** (violates AC #11 "converge, don't fork") —
  extracted `components/ui/FilterRailShell.tsx` (pixel-identical DOM/classes/testids) and migrated **both**
  rails onto it. LibraryFilterRail (ux3-0-7) DOM is byte-identical → no visual-baseline churn; its 3 specs
  still green.
- **[L1] Positional boolean params** — `useDiscoverResults(…, true, true)` → options object
  `{ enabled, keepPrevious }`; all 3 call sites + spec updated.

**Noted, NOT fixed (out of scope / informational):**

- **[L2]** `tsc --noEmit` repo-wide is not clean: 5 pre-existing `TS2352` errors in
  `apps/web/src/routes/test/-gallery.fixtures.tsx` (a dev-only component-gallery fixtures file, **not** in this
  story's diff). `nx build web` is green because Vite/esbuild transpiles without a full typecheck. Touched files
  are type-clean. Recommend a separate tracking ticket for the gallery fixtures type debt.
- **[L3]** type='all' grid interleaves by `voteCount` (≠ the `評分`/`vote_average` sort key) and only sorts the
  current page — pre-existing behavior shared with legacy `SearchResults`, not introduced here.

**Verification:** 57 web specs green (incl. LibraryFilterRail regression) · ESLint clean (Rule 21 header added
to `FilterRailShell`) · touched files type-clean · prettier clean.

## Change Log

| Date | Change |
|------|--------|
| 2026-06-24 | **Adversarial CR (auto-fix):** M1 AC #3 wording realigned to the shipped single-total (decision 1A); M2 rail count now reflects `isFetching` (was `isLoading` — stale under `keepPreviousData`) +regression test; M3 extracted shared `FilterRailShell` and converged 探索 + 媒體庫 rails (AC #11, pixel-identical); L1 `useDiscoverResults` → options object. 57 specs / lint / typecheck(touched) / prettier green. |
| 2026-06-23 | ux3-3-2 implemented: `/discover` v2 shell-gated persistent filter rail (`DiscoverBrowseV2` + `DiscoverFilterRail` + `DiscoverStatesV2`). AC #3 = single live total `符合 N 部` (per-facet counts deferred → backlog `ux3-discover-facet-aggregation-be`, TMDb has no facet aggregation). Debounce/replace/coalesce/summary/v2-sheet refinements shell-gated so legacy is byte-unchanged. `vote_average` confirmed on cards. `.pen` I1-D-v2 aligned to single-total. Tests green (web 2235 / api / build / lint / prettier). Status → review. |
| 2026-06-23 | 🔗 AC Drift: Story 11-2 AC #4 (push history) → ux3-3-2 AC #5 (`replace:true`) — v2-only via `useShellVersion()` gate; legacy push semantics preserved. |
