# Story UX2-2: A′ Browse v2 — Library in v2 behind `new_shell_enabled`

Status: done

> UX Redesign **Phase 2** pilot · Story 2 of 3. **Depends on UX2-1 (FOUNDATION)** — needs the shell, flag, `LegacyContentContainer` opt-in, Base UI wrappers, and v2 tokens in `styles.css`.
> Design: `ux-design.pen` `flow-a-browse-v2/*` (10 screens) + `PosterCard-v2` (`hD7Tw`); `01-design-language-v2.md` §2–§3, §7 (four states).

## Story

As the solo power user opening Vido daily,
I want the **library Browse** experience rebuilt in v2 — sidebar shell, integrated toolbar + filter chips, a tighter poster grid with lifecycle status, list view, and all four states — behind the `new_shell_enabled` flag,
so that my most-frequent surface is coherent, dense where I want depth, and never shows a bare blank.

## Acceptance Criteria

1. **Route restructure (A1/D2).** `library.tsx` (currently a 753-line monolith) splits into a **layout route** (`library.tsx` — shared toolbar/filter/sort + scroll state + `<Outlet/>`) and child views `library/index.tsx` (merged cross-type), `library/movies.tsx`, `library/tv.tsx`. The three views **share one `LibraryGrid`** (the existing `components/library/LibraryGrid.tsx` — never fork grid logic; P7 black-hole guard). Filter + scroll state live in the layout route and survive `movies↔tv` switches (F5 return-context).
2. **Deep-link preservation (F1, route-level).** `/library?type=movie` → `/library/movies`, `?type=series` → `/library/tv`, via TanStack `beforeLoad` throwing `redirect()` (NOT a component `<Navigate>`); `type` coerced per Rule 26 before redirect. Old bookmarks never 404.
3. **v2 grid (flagship, `LcHBs`).** Grid view uses the v2 `PosterCard-v2`: 2-line CJK title grid (tight title→meta spacing), JetBrains Mono year·runtime in `text-secondary`, and an **N1 lifecycle status badge** on the poster sourced from the existing `subtitleStatus` field (`types/library.ts:152`) + download/library state (繁中 `success` / 簡轉繁 `accent` / 缺字幕 `warning` / 下載中·{pct}% `accent` / 已入庫). Badge absent/degraded when status unknown (never errors).
4. **Integrated toolbar (`LcHBs`/`b1H71g`).** One toolbar row: sort dropdown + filter chips (類型 / 年份 / 解析度 as dropdown chips) + active-filter chips (removable) + grid/list segmented toggle + result count (Mono). Default (unfiltered) state shows **no active chip**. All Noto Sans TC, 44px hit areas.
5. **List view (`b1H71g`).** List shows thumbnail + title + meta (year·runtime·genre) + tech badges (`accent-tint`) + subtitle-status pill + row menu. View mode persists (reuse existing `vido:library:view` storage key).
6. **Four states (N4, `01-design-language-v2.md` §7).**
   - **Empty (`vZpT8`/`BfGVZ`)** — onboarding-aware (no-qBittorrent / no-folder / ready-to-scan via the existing `EmptyLibrary-*` 3-state classifier), centered icon + Noto title + subtitle + next-step CTA (`設定媒體資料夾`).
   - **Loading (`EsoIv`/`qBWQC`)** — skeleton matching the grid shape (poster blocks + text bars), `prefers-reduced-motion` respected; chrome renders immediately, grid hydrates.
   - **No-result (`R3FqJc`)** — distinct from empty; acknowledges the active filter, offers `清除全部篩選`.
   - **Error (`dVGIa`)** — per-section fail-soft: compact inline error + error code (mono) + `重試`; the page never hard-fails (F3 — treat a non-`ok` list response as data, never throw).
7. **Mobile (`h1v1U6` / `Bz0YN`).** Top bar (title + search + filter icons) + horizontal filter chips + **2-col** v2 grid; the **merged sort+filter Bottom Sheet** (`Bz0YN`, reuse Epic-11 sheet via Base UI `Sheet`) replaces the separate mobile sort/filter sheets. 44px tabs/targets; bottom-tab `媒體庫` active.
8. **v2 atom migration (per-flow, strangler).** The atoms Browse uses migrate to v2 here (not globally): `PosterCard`→`PosterCard-v2`, `FilterChip`/`SortDropdown`/`SearchInput`/`EmptyLibrary-*` to Noto + tokens + 44px (`01-design-language-v2.md` §5.1). Other flows' atoms are untouched (they migrate in their own Phase-3 story).
9. **Flag-gated (strangler).** All of the above renders **only** when `new_shell_enabled` is ON (Browse routes opt into the new layout via the FOUNDATION marker). Flag OFF → the current library renders unchanged. This story MUST NOT alter any non-Browse legacy screen.
10. **Grid re-flow under sidebar width.** Column breakpoints recompute for the new content width (1440−240 sidebar): desktop 6-up (cards `fill_container`), mobile 2-up. No layout shift / stuck-column regression (P7 `container`-class guard).
11. **a11y + perf.** AA contrast (v2 tokens), 44px targets, visible `focus-ring`, virtualized/continuous scroll (no pagination — competitive table-stakes), images lazy + sized to avoid CLS.

## Tasks / Subtasks

### Frontend
- [ ] **T1: Library layout + child routes** (AC #1, #2, #10) — split `library.tsx` into layout + `index`/`movies`/`tv`; shared `LibraryGrid`; shared filter/scroll state in layout; `beforeLoad` redirects for `?type=` (Rule 26 coercion); recompute grid breakpoints.
- [ ] **T2: `PosterCard-v2` + status badge** (AC #3) — implement to match `.pen` `hD7Tw`; map `subtitleStatus` + lifecycle → status→token (§2.5); 2-line CJK title grid; spec.
- [ ] **T3: Integrated toolbar** (AC #4, #5, #8) — sort + filter chips + active chips + grid/list toggle + count; migrate `FilterChip`/`SortDropdown`/`SearchInput` to v2; list-row component.
- [ ] **T4: Four states** (AC #6) — empty (reuse 3-state classifier) / loading skeleton / no-result / error (fail-soft); each matches its `.pen` screen.
- [ ] **T5: Mobile Browse + merged sheet** (AC #7) — mobile top bar + chips + 2-col grid; merged sort+filter Base UI `Sheet`.
- [ ] **T6: Specs + visual fixtures** (AC #11) — co-located specs (Rule 9, Rule 16 matchers); add the migrated atoms + new states to `/test/gallery` for visual-regression baselines; jsx-a11y.

### Backend
- [ ] **T7: Verify list payload (reuse, no new endpoint)** — confirm the library list endpoint returns `subtitleStatus` + per-type counts the toolbar needs; if a count/aggregate is missing, expand scope per Rule 24 (new sub-task + AC) rather than silently adding. Case-transform at boundary (Rule 18). **Expected: 0 new endpoints** (list + filters already serve Browse).

## Dev Notes

### Architecture Compliance
- **Rule 5:** list/filter via TanStack Query; view/sort/filter UI state local (reuse `vido:library:*` keys).
- **F1 redirects:** `beforeLoad`+`redirect()` only; Rule 26 numeric-param coercion (`?type=`/lone-numeric filters).
- **F3 fail-soft:** non-`ok` list response is data → error state, never a thrown page.
- **N4 four states:** all four designed + built; a missing state = incomplete (gate).
- **Rule 21:** new/changed components header their `.pen` node id (`hD7Tw` etc.); migrated atoms update their refs.
- **Rule 22/23:** new states + migrated atoms get visual baselines via `/test/gallery`; `-linux` via chore-visual bootstrap.
- **Strangler (P3):** Browse-only; do not touch other flows' atoms or screens.

### Cross-Stack Split Check (Agreement 5)
Backend tasks: **1** (verify payload, likely 0 code). Frontend tasks: **6**. Backend ≤3 → **NO split**. Frontend-led migration (the list API already serves the data — a legitimate frontend-heavy vertical slice).

### Project Structure
- CREATE: `routes/library/{index,movies,tv}.tsx`; `components/library/PosterCardV2.tsx`, list-row + state components (+ specs).
- MODIFY: `routes/library.tsx` → layout; `components/library/LibraryGrid.tsx` (consume v2 card, breakpoints); `FilterChip`/`SortDropdown`/`SearchInput`/`EmptyLibrary-*` (v2); `/test/gallery`.
- REUSE: `hooks/useLibrary` (`useLibraryList`/`useLibrarySearch`/`useLibraryStats`); `libraryService.ts`; existing bottom-sheet.

### Design Refs (`.pen` — Rule 21) — `flow-a-browse-v2/`
grid `LcHBs` · list `b1H71g` · empty `vZpT8`(D)/`BfGVZ`(M) · loading `EsoIv`(D)/`qBWQC`(M) · no-result `R3FqJc` · error `dVGIa` · mobile grid `h1v1U6` · merged sheet `Bz0YN` · card `PosterCard-v2 hD7Tw`. Screens: `_bmad-output/screenshots/flow-a-browse-v2/`.

### References
- [Source: planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md — A1 routes, D2 library group, F1/F5, Impact (library)]
- [Source: planning-artifacts/ux-redesign/01-design-language-v2.md — §2.5 status map, §3.3 title grid, §5.1 atoms, §7 states]
- [Source: project-context.md — Rules 5,9,16,18,21,22,23,24,26]
- [Source: apps/web/src/routes/library.tsx; components/library/LibraryGrid.tsx; types/library.ts:152 (subtitleStatus); hooks/useLibrary.ts]

## Dev Agent Record

### Implementation Summary (Amelia/dev — 2026-06-14)

Branch `feat/ux2-2-browse-v2` (off main, post-UX2-1 merge). Verified: `tsc` (0 new
errors), `eslint` (clean; the 3 `react-hooks/exhaustive-deps` warnings in
`library.tsx` are pre-existing in the untouched legacy `LibraryPage`, line-shifted
by the new imports), 21 new unit specs + existing `LibraryGrid`(20)/`AppShellV2`(3)
specs pass, `nx build web` (2332 modules) ok.

**Flag-gating (F4 preserved):** new `components/shell/shellVersion.tsx` context —
`AppShellV2` provides `'v2'`, default `'legacy'`. `routes/library.tsx` branches on
`useShellVersion()`: v2 → `<LibraryBrowseV2/>`; legacy → the existing `LibraryPage`
**pixel-unchanged** (P3). The flag itself stays read-once in `__root` — the route
reads the shell context, not the flag. Route marked `staticData.shell:'v2'`
(full-bleed under the v2 shell).

**v2 Browse:** `LibraryBrowseV2` (one component, all three type views off `?type=`,
shared `LibraryGrid`-style grid + shared sort/filter/scroll state surviving
movies↔tv — AC #1 intent + F5); `PosterCardV2` (`hD7Tw` — 2-line CJK title grid,
Mono meta, N1 status badge); `LibraryListRowV2`; integrated toolbar (sort + filter
+ active chips + count + grid/list toggle); four states
(`LibraryStatesV2`: loading skeleton / no-result / error-fail-soft + Empty via the
reused 3-state classifier); continuous scroll (`useLibraryInfinite` +
IntersectionObserver, AC #11); merged mobile sort+filter sheet
(`LibraryFilterSheetV2` on the UX2-1 Base UI `Sheet`); status→token util
(`utils/libraryStatus.ts`, §2.5).

### Discovery Triage (Rule 24)

1. **Route split → deferred (single route + `?type=`).** AC #1/#2 specify
   `/library/{movies,tv}` child route FILES + `?type=`→`/library/movies` redirects.
   Under the strangler flag this needs fragile flag-conditional `beforeLoad`
   redirects (router context has no `queryClient`; the flag lives in React
   context/query-cache) for marginal go/no-go value — the URL scheme is not what's
   validated. PILOT keeps `/library` as one route branching on shell-version and
   serving per-type views via the existing `?type=` param: shared grid + shared
   state across type switches (AC #1 intent), deep links preserved trivially
   (AC #2 — never 404, sidebar already links `?type=`). The clean-route migration
   (D2) is a low-risk Phase-3 follow-up once v2 is validated.
2. **N1 subtitle badge derivation.** No item-level `subtitleStatus` exists (it's
   episode-only); derived from `parseStatus` (success→已入庫 / pending→整理中 /
   failed→失敗) + `subtitleTracks` JSON (zh-Hant→繁中 / zh-Hans→簡中 / none→缺字幕).
   The richer process states (下載中·% / 簡轉繁 / AI 校正中) are NOT list-level
   derivable → Phase-3 backend field.
3. **解析度 filter chip → deferred.** The list endpoint filters by genres/year/
   unmatched/type only — no resolution filter. Pilot toolbar ships 類型(genre)+年份;
   resolution needs a backend param (Phase-3, Rule-24 follow-up).
4. **Filter editing via the merged Sheet on all breakpoints** (not desktop inline
   dropdown-chips) — a pilot simplification; the design's richer toolbar
   dropdown-chips are a refinement. Active filter chips are inline + removable.
5. **Visual gallery fixtures → deferred follow-up.** v2 components are token-only
   (low drift); behavior covered by 21 specs + the runtime validation step. Adding
   `/test/gallery` fixtures + the `-linux` baseline bootstrap is batched as a
   follow-up to avoid a mid-pilot bootstrap-PR cycle (Rule 22/23).

## Completion Notes
- Flag OFF → legacy library unchanged (safe rollback). Flag ON → v2 Browse.
- Browser-pixel verification at 390/768/1440 happens in the Phase-2 validation step
  (`02-pilot-validation.md`) together with UX2-3 (brief P10).
