# Story UX2-2: A′ Browse v2 — Library in v2 behind `new_shell_enabled`

Status: ready-for-dev

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
_(to be filled by dev-story)_
