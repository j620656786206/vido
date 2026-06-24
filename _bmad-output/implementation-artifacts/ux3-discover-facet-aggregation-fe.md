# Story (UX3): Discover Contextual Facet Counts — Frontend

Status: ready-for-dev

<!-- Story key: ux3-discover-facet-aggregation-fe (FE consumer of the facet-counts BE contract).
     Source: tech-spec-ux3-discover-facet-aggregation.md (Tasks 6–11, AC8–AC12).
     Consumes ux3-discover-facet-aggregation-be [@contract-v1]. Two EXECUTION GATES — see "Blocking Prerequisites". -->

## Story

As a **Discover-v2 user composing filters on the desktop rail**,
I want **each filter chip to show how many results it would yield (in JetBrains Mono), with dead-end (0-result) chips dimmed but still clickable**,
so that **I can see at a glance which facets are worth picking and avoid dead-ends — without clicking each one and waiting.**

(Frontend half only. Consumes the `GET /api/v1/tmdb/discover/facet-counts` endpoint from `ux3-discover-facet-aggregation-be` `[@contract-v1]`.)

## 🚧 Blocking Prerequisites (execution gates — read FIRST)

This story is **prepped and dev-ready in content**, but two gates govern EXECUTION order:

1. **`.pen` design prerequisite — `ux3-discover-facet-aggregation-design` (ux-designer, NOT this dev flow).** ux3-3-2 decision 2改 REMOVED the per-facet count nodes from `.pen` I1-D-v2 (`fxCVk`); they must be re-added with the NEW states before **Task 5 (per-chip count UI)** is built. The dev MUST NOT invent the chip visual (violates the design-system-conformance rule — `project-context.md` / memory `design-must-conform-to-current-design-system`). **The data layer (Tasks 1–4) can proceed against `[@contract-v1]` now; the visual Task 5/6 wait on the design.**
2. **BE endpoint merged — `ux3-discover-facet-aggregation-be` (`ready-for-dev`).** Integration/E2E against the live endpoint needs it merged; unit work mocks the service and can proceed. Also the infra prereq `infra-cache-entries-expiry-sweep` must ship before this goes live (it bounds the cache growth the BE fan-out amplifies).

## Acceptance Criteria

> Consumes `ux3-discover-facet-aggregation-be` `[@contract-v1]` — request `{base discover params}` + `{genre_values, region_values, rating_values, platform_values}` CSVs; response `{ counts: { genre:{id:int}, region:{code:int}, rating:{value:int}, platform:{id:int} }, partial: bool }`. **This story's Dev Notes record `confirmed against [@contract-v1]` (Rule 20).**

1. **AC1 (per-chip count display — tech-spec AC8):** Given the v2 shell and counts available, when the desktop rail renders, then each genre/region/rating/platform chip shows its **contextual** count in **JetBrains Mono** (`font-mono tabular-nums`), keyed by the same id/code the chip uses (`genre.id` / `region.code` / `rating value` / `platform.id`).
2. **AC2 (dead-end dimmed-but-selectable — tech-spec AC9):** Given a facet's count is `0`, when rendered, then the chip is **DIMMED** (reduced opacity / muted text) but **still clickable** (NOT `disabled` / `aria-disabled`) — the user can switch to it.
3. **AC3 (contextual counts):** Given a base selection (e.g. `genre=28`), when counts render, then each chip's count reflects (base ∩ that facet), served by the BE — the FE does not compute counts, it only displays the keyed response.
4. **AC4 (instant-feel — tech-spec AC10):** Given a chip is toggled, when counts recompute, then **only non-toggled dimensions update**, already-known counts stay stable (no full-rail "everything loading"), there are **no per-chip spinners** (subtle fill only), and the recompute is **debounced ~350ms** after the committed filter settles (matching the year-input debounce).
5. **AC5 (progressive partial fill):** Given the endpoint returns `partial: true`, when rendered, then the resolved counts show and the unresolved chips keep their prior/empty state and fill on a **re-poll**; the re-poll uses **backoff + conservative concurrency (AR-F7)** so it does not re-drain the shared limiter.
6. **AC6 (fallback to single total — tech-spec AC11):** Given facet-counts is unavailable, errors, or returns all-`partial` with nothing resolvable, when the rail renders, then it **falls back to today's single total** `符合 N 部` (the page never hard-fails; the existing `discover-rail-count` footer stays).
7. **AC7 (shell + scope gating):** Given the shell is NOT `v2`, when the rail/sheet renders, then `useDiscoverFacetCounts` is **gated off** (no network). Facet-counts are **DESKTOP-rail only** — the mobile `FilterBottomSheet` keeps its single draft total and does NOT request per-facet counts (tech-spec AC12).
8. **AC8 (no regression to the instant rail):** Given the v2 rail shipped in ux3-3-2, when facet-counts are added, then the categorical chips stay **instant** (counts are additive decoration; toggling a chip still applies instantly via `setFilters`), and the legacy shell path is byte-unchanged.

## Tasks / Subtasks

_Tasks 1–4 (data layer) can start now against `[@contract-v1]`; Tasks 5–6 (visual) wait on the `.pen` design (Gate 1)._

- [ ] **Task 1 — Response type + service call (AC: #1, #3) — consumes `[@contract-v1]`**
  - [ ] File: `apps/web/src/services/tmdb.ts`. Add `FacetCounts` type:
    ```typescript
    export interface FacetCounts {
      counts: {
        genre?: Record<string, number>;
        region?: Record<string, number>;
        rating?: Record<string, number>;
        platform?: Record<string, number>;
      };
      partial: boolean;
    }
    ```
  - [ ] Add `async discoverFacetCounts(params: URLSearchParams): Promise<FacetCounts>` → `fetchApi<FacetCounts>(\`/tmdb/discover/facet-counts?${params.toString()}\`)` (mirror `discoverMovies` `tmdb.ts:100`; `fetchApi` already `snakeToCamel`-transforms — `counts`/`partial` survive as-is).
  - [ ] The `params` carry the base filter **plus** the `*_values` candidate CSVs built by the Task 3 helper.
- [ ] **Task 2 — `useDiscoverFacetCounts` hook (NEW) (AC: #3, #4, #5, #7)**
  - [ ] File: `apps/web/src/hooks/useDiscoverFacetCounts.ts`. Mirror `useDiscoverResults.ts:92` structure (TanStack Query).
  - [ ] `enabled` gate = `useShellVersion() === 'v2'` **AND** caller opts in (desktop only — the mobile sheet never calls it). Off → no network.
  - [ ] queryKey from the **base** `buildDiscoverParams(filters, 'all')` + the candidate set (so a base-filter change re-queries; a sort/page change does NOT — the BE normalizes those away, AR-F2).
  - [ ] **Debounce ~350ms** on committed-filter change (match FilterPanel's `debounceMs={350}`); keep already-known counts stable across the debounce (no flash). Recommended: debounce the `filters`→queryKey input, or `placeholderData: keepPreviousData` so prior counts persist while refetching.
  - [ ] **Partial re-poll (AR-F7):** when `data.partial`, schedule a refetch with **backoff** (e.g. via `refetchInterval` returning increasing delays, stopping when `partial` clears) and conservative concurrency so re-polls don't re-drain the shared limiter.
  - [ ] Return shape (mirror `UseDiscoverResultsResult`): `{ counts: FacetCounts['counts'] | undefined, partial: boolean, isLoading: boolean, isFetching: boolean }`.
- [ ] **Task 3 — Candidate enumeration helper (AC: #1, #3) — Q1=A**
  - [ ] File: `apps/web/src/lib/discoverFilters.ts`. Add a helper that produces the `*_values` candidate CSVs FROM the existing inventory consts (the FE is the single inventory source, Q1=A):
    - `genre_values` = `GENRE_FILTER_OPTIONS.map(g => g.id)` (`:54`)
    - `region_values` = `REGION_OPTIONS.map(r => r.code)` (`:59`)
    - `rating_values` = `RATING_OPTIONS` (`:78`)
    - `platform_values` = `PLATFORM_OPTIONS.map(p => p.id)` (`:71`)
  - [ ] Provide a function (e.g. `buildFacetCountParams(filters): URLSearchParams`) that = `buildDiscoverParams(filters, 'all')` (base) **with** the four `*_values` CSVs appended. The count inner-keys returned by the BE then align 1:1 with the chip keys (`String(genre.id)` / `region.code` / `String(value)` / `String(platform.id)`).
  - [ ] **No "add facet to params" mapping needed FE-side** — the BE computes the contextual "base + facet" (Q1=A); the FE only enumerates candidates + displays the keyed response.
- [ ] **Task 4 — Wire the rail (AC: #4, #6, #7, #8)**
  - [ ] File: `apps/web/src/components/search/DiscoverFilterRail.tsx`. Call `useDiscoverFacetCounts(filters, { enabled: true })` (desktop rail is already v2-only via `DiscoverBrowseV2`); thread `counts` + `partial` as a new prop to `FilterPanel`.
  - [ ] **Fallback (AC6):** when `counts` is undefined/all-unavailable, render exactly today's footer (`符合 ${totalResults} 部` / `計算中…`, `:49-71`) — no change. Counts are additive; their absence degrades to the current rail.
  - [ ] Keep `totalResults` / `isCounting` footer as the summary/fallback (unchanged).
- [ ] **Task 5 — Per-chip count UI (AC: #1, #2) — ⛔ GATED on `.pen` design (Gate 1)**
  - [ ] File: `apps/web/src/components/search/FilterPanel.tsx`. Add an optional `facetCounts?: FacetCounts['counts']` prop (desktop-rail passes it; the mobile sheet does NOT → mobile stays count-less, AC7).
  - [ ] In each chip `.map` (genre `:147`, region `:169`, rating `:219`, platform `:240`), render the count after the label, before `</button>`: a `font-mono tabular-nums` span keyed by the chip's id/code (`String(genre.id)` / `region.code` / `String(value)` / `String(platform.id)`).
  - [ ] `count === 0` → apply a **dimmed** modifier to `chipClass` (`:30`) but keep the button enabled + clickable (AC2 — do NOT set `disabled`).
  - [ ] Build the chip per the `.pen` `FacetCountChip` (filter-controls-v2) design states (computing/progressive-fill + dead-end-dimmed) — **match the design, do not invent** (Gate 1).
  - [ ] Subtle fill (no per-chip spinner). Add `data-testid` for the count (e.g. `facet-count-{dim}-{key}`) for E2E/unit assertions.
- [ ] **Task 6 — FE tests (AC: all)**
  - [ ] Files: `FilterPanel.spec.tsx`, `useDiscoverFacetCounts.spec.tsx` (NEW), extend `tests/e2e/discover-filters.spec.ts`.
  - [ ] `FilterPanel.spec.tsx` (mirror `:1-24` setup): per-chip count renders in Mono; `0` → dimmed but still fires `onChange` when clicked (AC2 — use `toBeAttached`/`toHaveClass`, NOT `toBeDisabled`); mobile/no-`facetCounts` prop → no counts rendered (AC7).
  - [ ] `useDiscoverFacetCounts.spec.tsx` (mirror `useDiscoverResults.spec.tsx:1-22`, mock `tmdbService.discoverFacetCounts`, `QueryClientProvider` wrapper): debounce (~350ms via fake timers); partial re-poll with backoff; disabled when `useShellVersion()!=='v2'` (mock the hook); fallback (undefined counts).
  - [ ] E2E (`discover-filters.spec.ts`, reuse `enableV2Shell` `:216` + `stubDiscover` `:100` + add a `**/tmdb/discover/facet-counts*` route stub): rail shows per-chip counts; a 0-count chip is dimmed yet still navigates on click; endpoint-unavailable → falls back to `符合 N 部`. Follow the seed-helper / no-self-skip convention.
  - [ ] Run `npx playwright test tests/e2e/discover-filters.spec.ts --project=chromium`; `npx vitest run "FilterPanel|useDiscoverFacetCounts"` (from `apps/web`); `pnpm lint:all`.

## Dev Notes

### Contract consumption (Rule 20)

- **confirmed against `[@contract-v1]`** (`ux3-discover-facet-aggregation-be`): request = base discover params (the existing `buildDiscoverParams` query keys) + `genre_values` / `region_values` / `rating_values` / `platform_values` CSVs; response = `{ counts: { <dim>: { <id|code|value>: int } }, partial: bool }`. Inner keys are the facet value **as the FE supplied it** → they align with the chip keys directly. If the BE bumps the contract, this story stale-marks per Rule 20.

### Ratified design decisions (Party Mode 2026-06-24) that shape THIS story

- **Q1 = A (FE owns inventory; sends candidates).** The FE `discoverFilters.ts` consts ARE the candidate source (Task 3). No BE inventory; the BE counts only what the FE sends. This is why Task 3 is a thin enumeration over existing consts and there is **no FE-side "add facet to params" mapping** (the BE does the contextual add).
- **Q2 = A (reuse / shared cache).** Transparent to the FE — counts are TMDb `total_results` (approximate, page-capped ~10k, per-locale). Present as exact, tolerate small drift vs the grid (tech-spec Decision #7).

### Codebase anchors (current line numbers — verified 2026-06-24, post-ux3-3-2)

- **`DiscoverFilterRail.tsx`**: props `:19-31` (`filters`, `activeCount`, `totalResults`, `isCounting`, `onChange`, `onClearAll`, `onCollapse`); `<FilterPanel filters onChange debounceMs={350}/>` `:74`; single-total footer `:49-71` (`data-testid="discover-rail-count"`, `符合 N 部` / `計算中…`). Wire `useDiscoverFacetCounts` here; thread `counts` to `FilterPanel`.
- **`FilterPanel.tsx`**: `chipClass(active)` `:30-36`; genre map `:144-163` (`GENRE_FILTER_OPTIONS`, `data-testid="filter-genre-{id}"`, key `genre.id`); region map `:166-185` (`REGION_OPTIONS`, key `region.code`); rating map `:216-234` (`RATING_OPTIONS`, key `value`); platform map `:237-256` (`PLATFORM_OPTIONS`, key `platform.id`). Counts insert after the label, before `</button>`.
- **`discoverFilters.ts`**: `GENRE_FILTER_OPTIONS` `:54` (18 ids), `REGION_OPTIONS` `:59` (TW/JP/KR/US/CN), `PLATFORM_OPTIONS` `:71` (8/337/425), `RATING_OPTIONS` `:78` (6/7/8/9); `buildDiscoverParams` `:213-239` (base → query params). **NOTE:** region inventory is TW/JP/KR/US/**CN** and platforms are 8/337/**425 (KKTV)** — use the live consts, do not assume the tech-spec's illustrative ids.
- **`useDiscoverResults.ts`**: `discoverKeys` `:13-23`, `useDiscoverMovies` `:28-45` (queryKey + `enabled` + `placeholderData: keepPreviousData`), `useDiscoverResults` `:92-120` (coalesced `isLoading`/`isFetching`/`totalResults`). Mirror for the new hook (but SINGLE query, not movie+tv pair — the BE sums).
- **`tmdb.ts`**: `fetchApi` `:24-39` (auto `snakeToCamel`); `discoverMovies` `:100`; `API_BASE_URL` `:13` (`/api/v1`). Add `discoverFacetCounts`.
- **`shellVersion.tsx`**: `useShellVersion(): 'legacy'|'v2'` `:14-22`; `DiscoverPage` gate `discover.tsx:69-74` (`shell==='v2' ? <DiscoverBrowseV2/> : <LegacyDiscover/>`). The desktop rail only renders in v2, but gate the hook explicitly too (defense-in-depth).
- **`useFilterState.ts`**: `{ filters, setFilters (navigate replace on v2), clearAll }` `:13-58`. Chip toggles call `setFilters` (instant). Debounce the COUNTS recompute, never the chip apply (AC8).
- **`FilterBottomSheet.tsx`**: separate mobile component `:9-24` (batch draft, `variant:'v2'`); chosen `<lg` in `DiscoverBrowseV2.tsx:138-154` (desktop rail `hidden lg:block`, sheet below). **Do NOT pass `facetCounts` to it** (AC7 — mobile stays count-less).
- **Tests**: `FilterPanel.spec.tsx:1-24` (RTL + `baseFilters`), debounce test w/ fake timers `:81-100`; `useDiscoverResults.spec.tsx:1-22` (mock `tmdbService`, `QueryClientProvider` wrapper); E2E `tests/e2e/discover-filters.spec.ts` — `enableV2Shell` `:216-234` (localStorage `vido:flag:new_shell_enabled` + settings route stub), `stubDiscover` `:100-111`, v2 rail test `:237-256`.
- **`.pen` FacetCountChip**: Component Library `filter-controls-v2`; single-total after ux3-3-2 decision 2改. Per-chip count states must be re-added by the design prereq (Gate 1) — `.pen` read via Pencil MCP only, never `Read`/`Grep`.

### Rule compliance

- Rule 5 (TanStack Query for server state), Rule 16 (test matchers — `toBeAttached`/`toHaveClass` for the dimmed-but-present chip, NOT `toBeVisible`; per memory, don't assert CSS hover/opacity with `toBeVisible`), Rule 18 (the response is snake→camel transformed by `fetchApi`; `counts`/`partial` keys are simple), Rule 20 (`confirmed against [@contract-v1]`), Rule 27 ③ (graceful fallback to the single total).
- **Cross-stack split check:** 6 tasks, all frontend; 0 backend → single story, **no split**.

### Project Structure Notes

- All edits under `apps/web/src/{services,hooks,lib,components/search}` + co-located specs + the E2E spec. No backend changes.
- **Visual baselines:** Task 5 changes `FilterPanel`'s rendered appearance (per-chip counts). The `Visual Regression` CI gate will flag the diff; the changed `-darwin` baselines regen locally and the `-linux` set is bootstrapped by the CI workflow (per CLAUDE.md — never hand-commit `-linux`). Treat as an intentional, reviewed visual change.

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `FilterPanel` / `DiscoverFilterRail` / the new `FacetCountChip` render counts from server data, not `Date.now()` / `new Date()`. (Rule 23 not triggered.)

### References

- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md`] — FE Tasks 6–11, AC8–AC12, Technical Decisions #2/#6/#8, Risks F1/F5/F6, Transport ADR-2 (synchronous-with-time-budget + FE re-poll), `.pen` design prereq.
- [Source: `_bmad-output/implementation-artifacts/ux3-discover-facet-aggregation-be.md`] — `[@contract-v1]` request/response shape this story acks.
- [Source: `apps/web/src/{components/search/DiscoverFilterRail.tsx,components/search/FilterPanel.tsx,components/search/FilterBottomSheet.tsx,lib/discoverFilters.ts,hooks/useDiscoverResults.ts,hooks/useFilterState.ts,services/tmdb.ts,components/shell/shellVersion.tsx},routes/discover.tsx,tests/e2e/discover-filters.spec.ts] — anchors above (verified current, 2026-06-24).
- [Source: `project-context.md` Rule 5/16/18/20/27; memory `design-must-conform-to-current-design-system`] — governing rules.

## Dev Agent Record

### Agent Model Used

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

### Discovery Triage

<!-- Rule 24 (project-context.md). -->

- **Did this story discover any work outside its current scope?** **YES — one item, pre-triaged at story-creation:**
  - **③ — `.pen` FacetCountChip per-chip-count design does not exist (design prereq).** ux3-3-2 decision 2改 removed the per-facet count nodes; reviving per-chip Mono counts + the computing/dead-end-dimmed states needs a ux-designer/Pencil pass on I1-D-v2 (`fxCVk`) + `i1-d.png` regen. Out-of-scope for this DEV story (it is a design discipline, not React) but **blocks Task 5** (the visual). Filed as **`ux3-discover-facet-aggregation-design`** (backlog, Owner ux-designer), bidirectional link. Lane ③. (Data-layer Tasks 1–4 are NOT blocked by it.)
- Reference: `project-context.md` Rule 24; memory `design-must-conform-to-current-design-system`; origin: FE story prep (2026-06-24).

### File List

_(to be filled by dev agent)_

- `apps/web/src/services/tmdb.ts` (MODIFIED — `FacetCounts` type + `discoverFacetCounts`)
- `apps/web/src/hooks/useDiscoverFacetCounts.ts` (NEW)
- `apps/web/src/lib/discoverFilters.ts` (MODIFIED — candidate enumeration helper)
- `apps/web/src/components/search/DiscoverFilterRail.tsx` (MODIFIED — wire hook)
- `apps/web/src/components/search/FilterPanel.tsx` (MODIFIED — per-chip count UI)
- `apps/web/src/hooks/useDiscoverFacetCounts.spec.tsx` (NEW)
- `apps/web/src/components/search/FilterPanel.spec.tsx` (MODIFIED)
- `tests/e2e/discover-filters.spec.ts` (MODIFIED)
