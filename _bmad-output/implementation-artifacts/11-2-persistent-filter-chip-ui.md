# Story 11.2: Persistent Filter Chip UI

Status: review

## Story

As a Traditional Chinese NAS user browsing content,
I want active filters displayed as persistent pill/chip elements at the top of the page,
so that I can see what's currently filtered and easily add, remove, or clear filters.

## Acceptance Criteria

1. Given active filters, when displayed, then each filter is a removable chip/pill at the top of the content area (not hidden in dropdowns)
2. Given filter chips, when a user clicks the X on a chip, then that filter is removed and results update immediately
3. Given multiple active filters, when displayed, then a "清除全部" button appears to remove all filters at once
4. Given filter state, when the user navigates away and returns (back button), then the filter state is preserved via URL query params
5. Given a filter panel, when the user selects genre/year/region/rating/platform options, then chips are added immediately and results update
6. Given mobile viewport, when filters are active, then chips wrap to multiple lines and the filter panel is accessible via a bottom sheet

## Tasks / Subtasks

- [x] Task 1: Filter state management via URL (AC: #4)
  - [x] 1.1 Create `apps/web/src/hooks/useFilterState.ts` — sync filter state with URL search params
  - [x] 1.2 Params: `genre`, `year_gte`, `year_lte`, `region`, `rating_gte`, `platform`, `sort_by`
  - [x] 1.3 Use TanStack Router's `useSearch` for type-safe URL state
  - [x] 1.4 Back/forward navigation preserves filter state

- [x] Task 2: FilterChipBar component (AC: #1, #2, #3)
  - [x] 2.1 Create `apps/web/src/components/search/FilterChipBar.tsx`
  - [x] 2.2 Render active filters as removable pill chips (reuse existing `Component/FilterChip` design pattern)
  - [x] 2.3 Each chip shows filter label + value (e.g., "類型: 動畫")
  - [x] 2.4 X button per chip to remove individual filter
  - [x] 2.5 "清除全部" button when ≥2 filters active

- [x] Task 3: Filter panel (AC: #5)
  - [x] 3.1 Create `apps/web/src/components/search/FilterPanel.tsx`
  - [x] 3.2 Genre multi-select (from TMDB genre list, zh-TW labels)
  - [x] 3.3 Year range: min/max year inputs
  - [x] 3.4 Region: TW, JP, KR, US, CN quick-select chips
  - [x] 3.5 Rating: slider or min rating input
  - [x] 3.6 Platform: Netflix, Disney+, KKTV chip toggles
  - [x] 3.7 Sort dropdown: popularity, date, rating

- [x] Task 4: Mobile filter bottom sheet (AC: #6)
  - [x] 4.1 On mobile: filter panel renders as a bottom sheet (triggered by filter icon button)
  - [x] 4.2 Chips still visible above content (condensed, horizontal scroll)
  - [x] 4.3 Follow existing bottom sheet pattern from `apps/web/src/components/`

- [x] Task 5: Integration with discover results (AC: #1, #5)
  - [x] 5.1 Create `apps/web/src/hooks/useDiscoverResults.ts` — TanStack Query consuming filter state
  - [x] 5.2 Call `GET /api/v1/tmdb/discover/movies` or `/tv` with current filter params
  - [x] 5.3 Create browse/discover page or integrate into existing search route
  - [x] 5.4 Results grid reuses `PosterCard` component

- [x] Task 6: Tests (AC: #1-6)
  - [x] 6.1 FilterChipBar: render, remove, clear all
  - [x] 6.2 useFilterState: URL sync, back button preservation
  - [x] 6.3 FilterPanel: selection interactions
  - [x] 6.4 Mobile: bottom sheet open/close

## Dev Notes

### Architecture Compliance

- **URL state:** Filter state in URL search params — no Zustand/Redux needed. TanStack Router handles it
- **Chip design:** Follow existing `Component/FilterChip` from ux-design.pen (cornerRadius:100, bg-tertiary, text-secondary + X icon)
- **Genre list:** Fetch from `GET /genre/movie/list` and `/genre/tv/list` TMDB endpoints (may need to add these to client)
- **Bottom sheet:** Follow existing pattern from Flow D mobile filter (Screen 10)

### References

- [Source: apps/web/src/components/] — Existing component patterns
- [Source: ux-design.pen Component/FilterChip (jD7gF)] — Filter chip design
- [Source: ux-design.pen Screen 7 (rsAxf)] — Desktop search + filter layout
- [Source: ux-design.pen Screen 10 (oypj1)] — Mobile filter bottom sheet
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-011

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / BMM dev-story workflow)

### Debug Log References

- `pnpm nx build web` — clean; TanStack Router plugin regenerated `routeTree.gen.ts` with `/discover`
- `pnpm nx test web --skip-nx-cache` — 1915/1915 pass (156 files); test-cleanup reported no orphaned processes
- `pnpm nx test api` — all Go packages pass (full-regression gate, Epic 9 Retro AI-1)
- `pnpm exec eslint <new files>` — clean (Rule 21 `local/implements-pen-node-id` satisfied on all 3 new `components/search/` files)
- `pnpm exec prettier --check <new files>` — clean

### Completion Notes List

**AC Drift / Contract checks (Step 2):**

- 🔗 AC Drift: NONE (checked: `year_gte`/`vote_gte`/`watch_providers`/`/tmdb/discover`/`sort=` across `_bmad-output/implementation-artifacts/*.md` — hits in Story 10-1/10-1a/11-1 are the discover endpoint this story CONSUMES read-only. Zero backend files touched; no prior AC's observable behavior changed.)
- 📎 Contract Stamps: NONE (Story 11-1 recorded no `[@contract-v*]` stamps — pre-Rule-20 implicit v0; downstream→upstream ack requirement skipped per the v0 forward-only retrofit.)

**Architecture / key decisions:**

- **URL is the single source of truth (Rule 5):** all filter state lives in `/discover` route search params via TanStack Router; `useFilterState` reads `useSearch`/`useNavigate`. Back/forward navigation preserves filter state with zero extra code (AC #4). No Zustand/Redux.
- **URL param → backend param mapping:** the route exposes the Task 1.2 param names (`genre`, `year_gte`, `year_lte`, `region`, `rating_gte`, `platform`, `sort_by`); `buildDiscoverParams` translates them to the Story 11-1 wire contract (`rating_gte`→`vote_gte`, `platform`→`watch_providers`+`watch_region`, `sort_by`→TMDb-native `sort`). The abstract `sort_by` (`popularity`/`date`/`rating`) is mapped per media type (`date`→`primary_release_date.desc` for movies / `first_air_date.desc` for TV).
- **Year as min/max inputs + rating as min-rating chips:** follows the authoritative 11-UX flow-g screens (AS-1 desktop `年份範圍` two-input field; AS-4 mobile `最低評分` ★6+/7+/8+/9+ chips) and Task 3.3/3.5. (The generic flow-c/flow-d screens showed decade chips; the dedicated 11-UX screens supersede them and match the story tasks.)
- **Platform provider IDs live on the frontend:** Story 11-1 CR M1 removed the backend `TWWatchProviderIDs` map as dead code, so `PLATFORM_OPTIONS` (Netflix=8, Disney+=337, KKTV=425) now carries the display→ID mapping (`lib/discoverFilters.ts`). KKTV=425 per TMDb's TW provider catalog.
- **Reuse:** `FilterChipBar` reuses the `Component/FilterChip (jD7gF)` pill design (same `.pen` node as `library/FilterChips`); the results grid reuses `SearchResults` (→`MediaGrid`/`PosterCard`/`Pagination`); genre labels reuse `lib/genres.ts` `GENRE_MAP`. Desktop applies filters instantly (AC #5); the mobile bottom sheet drafts locally and commits on `套用篩選 (N 部結果)`.

**🔍 Discovery Triage (Rule 24):**

- ③ backlog-with-carry-forward-link: `/discover` has no top-nav entry yet (AppShell nav is logo + search + settings only). Out of scope for this story (no AC requires nav wiring) and non-blocking (route is reachable by URL and is the target of filter navigation). Tracked: `disc-nav-entry-discover-route` (see sprint-status.yaml) — discovered by Story 11-2.

**🎨 UX Verification (Step 9) — verified against 11-UX flow-g screens:**

| Area | Design Spec (flow-g) | Implementation | Match? |
|------|----------------------|----------------|--------|
| Active chips | Blue pills `類型: 動畫` / `年份: 2022-2024` + `清除全部` (AS-1) | `FilterChipBar` — `bg-accent/20 text-blue-300` pills, per-chip X, `清除全部` at ≥2 | ✅ |
| Desktop sidebar | `進階篩選` panel: 類型 / 年份範圍 (min-max) / 地區 / 排序方式 (AS-1) | `aside` `進階篩選` + `FilterPanel` (instant-apply); superset adds 最低評分 + 平台 per Task 3.5/3.6 | ✅ (superset) |
| Genre chips | Multi-select chips, blue when active w/ check (AS-1/AS-4) | `chipClass` accent border + `Check` icon when active | ✅ |
| Year range | Two numeric inputs `2023 — 2024` (AS-4) | `filter-year-gte` / `filter-year-lte` inputs with `—` separator | ✅ |
| Region | Flag chips 台灣/日本/韓國 (AS-4) | `REGION_OPTIONS` flag+label chips (TW/JP/KR/US/CN per Task 3.4) | ✅ |
| Min rating | `★6+ ★7+ ★8+ ★9+`, blue when active (AS-4) | `filter-rating-*` chips, `★{n}+` | ✅ |
| Mobile sheet | Bottom sheet `篩選條件` + `清除全部` + `套用篩選 (N 部結果)` (AS-4) | `FilterBottomSheet` — slide-up, drag handle, draft+apply with live count | ✅ |

No discrepancies requiring backend changes. `ux-design.pen` was NOT modified (read-only via the design contract), so the CLAUDE.md screenshot-regeneration workflow does not apply.

**🧪 TEA `*automate` (2026-06-03, Murat):**

- Added 6 E2E integration tests (`tests/e2e/discover-filters.spec.ts`) covering the URL↔filter↔API↔chips↔results round trip + browser-back persistence (AC #4) + mobile bottom sheet (AC #6). Hermetic (discover API mocked — no `TMDB_API_KEY`). Coverage report: `_bmad-output/automation-summary-11-2.md`. All 6 pass (chromium); discover unit suite still green (20/20); ESLint + Prettier clean.
- 🔍 Discovery Triage (Rule 24) — lane ① expand-scope-in-place: E2E surfaced a HIGH-severity bug in this story's own code — a single-value deep link (`?genre=16`/`?platform=8`) silently dropped the filter because TanStack Router JSON-parses a lone numeric query value into a `number`, which failed the route's `typeof === 'string'` guard. Fixed in place (`validateSearch` `toCsvString()` + `parseCsvInts` `String()` coercion) and guarded by the `[P2] deep link` E2E. Strengthens AC #4 (URL filter persistence). No separate backlog entry — the fix lives in the code this story shipped.

### File List

- `apps/web/src/lib/discoverFilters.ts` (new — filter model, URL↔backend mapping, chip descriptors, sort mapping, `buildDiscoverParams`)
- `apps/web/src/lib/discoverFilters.spec.ts` (new — parse/serialize/chips/sort/params unit tests)
- `apps/web/src/hooks/useFilterState.ts` (new — Task 1: URL-synced filter state)
- `apps/web/src/hooks/useFilterState.spec.tsx` (new — Task 6.2: URL sync + page reset + clearAll)
- `apps/web/src/hooks/useDiscoverResults.ts` (new — Task 5.1: TanStack Query discover hooks)
- `apps/web/src/hooks/useDiscoverResults.spec.tsx` (new — media-type gating + param mapping + query keys)
- `apps/web/src/services/tmdb.ts` (modified — added `discoverMovies`/`discoverTVShows`)
- `apps/web/src/components/search/FilterChipBar.tsx` (new — Task 2: persistent removable chips + clear-all)
- `apps/web/src/components/search/FilterChipBar.spec.tsx` (new — Task 6.1: render/remove/clear-all)
- `apps/web/src/components/search/FilterPanel.tsx` (new — Task 3: genre/region/year/rating/platform/sort controls)
- `apps/web/src/components/search/FilterPanel.spec.tsx` (new — Task 6.3: selection interactions)
- `apps/web/src/components/search/FilterBottomSheet.tsx` (new — Task 4: mobile bottom sheet w/ draft+apply)
- `apps/web/src/components/search/FilterBottomSheet.spec.tsx` (new — Task 6.4: open/close/apply/clear)
- `apps/web/src/routes/discover.tsx` (new — Task 5.3: discover page integrating chips + panel + sheet + results)
- `apps/web/src/routeTree.gen.ts` (regenerated — `/discover` route registered by TanStack Router Vite plugin)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified — 11-2 ready-for-dev → in-progress → review; + `disc-nav-entry-discover-route` backlog entry)
- `tests/e2e/discover-filters.spec.ts` (new — TEA `*automate`: 6 E2E integration tests, AC #1-6)
- `apps/web/src/routes/discover.tsx` (modified — TEA bug fix: `toCsvString()` coercion in `validateSearch` for single-value numeric `genre`/`platform` deep links)
- `apps/web/src/lib/discoverFilters.ts` (modified — TEA bug fix: `parseCsvInts` defensively `String()`-coerces input)
- `_bmad-output/automation-summary-11-2.md` (new — TEA `*automate` coverage report)

### Change Log

| Date       | Change                                                                                                              |
| ---------- | ------------------------------------------------------------------------------------------------------------------- |
| 2026-06-03 | Task 1: `useFilterState` + `discoverFilters` lib — filter state synced to `/discover` URL search params (AC #4)      |
| 2026-06-03 | Task 2: `FilterChipBar` — persistent removable chips, per-chip remove, `清除全部` at ≥2 active (AC #1, #2, #3)         |
| 2026-06-03 | Task 3: `FilterPanel` — genre/region/year-range/min-rating/platform/sort controls, instant-apply (AC #5)            |
| 2026-06-03 | Task 4: `FilterBottomSheet` — mobile slide-up sheet, local draft + `套用篩選 (N 部結果)` commit (AC #6)               |
| 2026-06-03 | Task 5: `useDiscoverResults` + `tmdbService.discover*` + `/discover` route integrating filters with the grid (AC #1, #5) |
| 2026-06-03 | Task 6: unit tests for chip bar, filter state, filter panel, bottom sheet, discover results (AC #1–6)                |
| 2026-06-03 | TEA `*automate`: added 6 E2E integration tests (`tests/e2e/discover-filters.spec.ts`) — filter→chip→URL→re-query, browser-back persistence, remove/clear, deep-link, mobile sheet (AC #1–6) |
| 2026-06-03 | TEA bug fix (E2E-surfaced, HIGH): single-value deep link `?genre=16`/`?platform=8` silently dropped the filter (numeric search param failed the `typeof === 'string'` guard). Fixed via `toCsvString()` in `validateSearch` + defensive `String()` in `parseCsvInts` (AC #4) |
