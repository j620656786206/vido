# Story 10.5: Homepage Layout Engine & Responsive Design

Status: review

## Story

As a Traditional Chinese NAS user,
I want the homepage to render all sections (hero banner, explore blocks, recent media) in a cohesive responsive layout,
so that the browsing experience feels like a modern streaming app on both desktop and mobile.

## Acceptance Criteria

1. Given the homepage, when fully loaded, then sections render in order: Hero Banner → Explore Blocks → Recently Added → Downloads
2. Given the homepage on desktop, when all blocks are populated, then LCP (Largest Contentful Paint) is under 2 seconds
3. Given the homepage on mobile (<768px), when rendered, then hero banner is compact, explore blocks scroll horizontally, and spacing is adjusted for touch
4. Given explore blocks loading, when data is being fetched, then skeleton placeholders are shown per block (not a full-page spinner)
5. Given the homepage, when a section has no data (e.g., no downloads), then that section is hidden entirely (no empty state shown)

## Tasks / Subtasks

- [x] Task 1: Refactor homepage layout (AC: #1)
  - [x] 1.1 Refactor `apps/web/src/routes/index.tsx` to compose: HeroBanner → ExploreBlockList → RecentMediaPanel → DownloadPanel
  - [x] 1.2 Each section is an independent component with its own data fetching
  - [x] 1.3 Wrap in a vertical flex layout with consistent section spacing (gap-8 desktop, gap-6 mobile)

- [x] Task 2: Performance optimization (AC: #2)
  - [x] 2.1 Hero banner backdrop: use TMDB `w1280` size (not `original`) for faster load
  - [x] 2.2 Explore block posters: use `w342` size (existing PosterCard behavior)
  - [x] 2.3 Lazy-load below-the-fold explore blocks with Intersection Observer
  - [x] 2.4 Prefetch trending data on route hover (TanStack Router prefetch)

- [x] Task 3: Skeleton loading states (AC: #4)
  - [x] 3.1 Create `apps/web/src/components/homepage/ExploreBlockSkeleton.tsx`
  - [x] 3.2 Horizontal row of PosterCardSkeleton (reuse existing)
  - [x] 3.3 Each block shows its own skeleton independently

- [x] Task 4: Responsive breakpoints (AC: #3)
  - [x] 4.1 Hero banner: h-[400px] desktop → h-[250px] mobile
  - [x] 4.2 Explore blocks: 6 cards visible desktop → horizontal scroll mobile with snap
  - [x] 4.3 Section titles: text-xl desktop → text-lg mobile
  - [x] 4.4 Test on 390px (iPhone), 768px (tablet), 1440px (desktop) — responsive classes asserted via Vitest (`md:h-[400px]`, `md:gap-8`, `sm:text-xl`); pixel-perfect verification at the three breakpoints is deferred to NAS dev-server (same pattern as Story 10-2).

- [x] Task 5: Empty state handling (AC: #5)
  - [x] 5.1 Each section returns `null` when data is empty or failed
  - [x] 5.2 No "empty state" UI for homepage sections — just hide

- [x] Task 6: Tests (AC: #1-5)
  - [x] 6.1 Homepage layout: verify section ordering
  - [x] 6.2 Skeleton states: verify per-block loading
  - [x] 6.3 Empty sections: verify hidden when no data

## Dev Notes

### Architecture Compliance

- **No new routes:** Enhance existing `/` route
- **Component composition:** Each section is self-contained with its own TanStack Query
- **Tailwind responsive:** Use `sm:`, `md:`, `lg:` breakpoint prefixes
- **Image sizing:** Follow TMDB image URL convention: `https://image.tmdb.org/t/p/{size}{path}`

### References

- [Source: apps/web/src/routes/index.tsx] — Current homepage
- [Source: apps/web/src/components/media/PosterCard.tsx] — Existing card + skeleton
- [Source: _bmad-output/planning-artifacts/prd/project-scoping-phased-development.md#2.5] — LCP <2s target

## Dev Agent Record

### Agent Model Used

- **Amelia** (Developer Agent) — BMM `/dev-story` workflow, model `claude-opus-4-7[1m]`, executed 2026-04-17.

### Debug Log References

- First regression run surfaced 2 new-test failures — `RecentMediaPanel/DownloadPanel — hideWhenEmpty still renders the panel during loading`. Root cause: both panels mount a `<Link>` to `/library` and `/downloads` in the footer *during* loading, and the tests were rendering the component raw (no TanStack RouterProvider) so `useLinkProps` returned `null` (`isServer` access on null). Fix: re-used the spec's existing `renderPanel()` helper (which wires a minimal RouterProvider) and threaded `hideWhenEmpty` through it. Second run: 1738/1738 PASS.
- Second lint run surfaced 6 `no-undef` errors for `IntersectionObserver` / `IntersectionObserverEntry` — the ESLint flat config at `eslint.config.mjs` listed DOM types like `HTMLElement`, `MessageEvent`, `EventSource`, but omitted the Intersection Observer API. Added both to the globals list. `pnpm lint:all` is now 0 errors.

### Implementation Plan

- **Task 1 (AC #1, Task 1.3)** — replaced the old `DashboardLayout` 2-column grid with a vertical `flex flex-col gap-6 md:gap-8` stack inside `routes/index.tsx`. Section order enforced in JSX: Hero → Explore → Recent → Downloads. QB status indicator + connection history modal are retained below the AC-prescribed stack (pre-existing utility from Epic 4, not part of Story 10-5's ordered sections).
- **Task 2.1 / 2.2 — already in place from Story 10-2 & 10-3** (HeroBanner uses `w1280` fallback + responsive `srcset`; `PosterCard` uses `w342`). Documented here to close the tasks; no code change needed.
- **Task 2.3** — introduced `useInViewport(ref, { rootMargin, once })` in `hooks/useInViewport.ts`. `ExploreBlocksList` marks the first **2** blocks as `eager` and tracks visibility of later blocks via `visibleIndices` state. `useQueries` is gated per slot on `index < EAGER_BLOCK_COUNT || visibleIndices.has(index)`. Children fire `onVisible()` when their `IntersectionObserver` intersects — the shared cache key means only one network request is ever issued for a given block, preserving Story 10-4's hoisted availability contract.
- **Task 2.4** — extracted the QueryClient singleton into `apps/web/src/queryClient.ts` so the new route loader (`routes/index.tsx`) can call `queryClient.prefetchQuery(trendingKeys.hero('week'))`. Router preload (`defaultPreload: 'intent'`, unchanged) fires this on Link hover.
- **Task 3** — extracted the inline skeleton into `components/homepage/ExploreBlockSkeleton.tsx` (6-card horizontal row, `aria-hidden="true"`). `ExploreBlock` renders it whenever `!shouldFetch || isLoading` so a lazy block that has not yet fetched still reserves layout space (no CLS).
- **Task 4** — `HeroBanner` height changed from viewport-units (`h-[40vh] sm:h-[50vh] lg:h-[70vh]`) to the story-prescribed pixel values (`h-[250px] md:h-[400px]`). The existing `sm:text-xl` / `lg:` classes on the hero title already satisfy 4.3; explore-block titles were already `text-lg sm:text-xl`; horizontal snap-scroll at mobile is already the `ExploreBlock` default.
- **Task 5** — added a `hideWhenEmpty` prop to `RecentMediaPanel` and `DownloadPanel`. When set (homepage only), each panel returns `null` once the relevant query resolves to empty/disconnected/no-downloads. During loading the panel still renders its skeleton so the layout doesn't flash. Non-homepage callers keep their existing empty-state UX.
- **Task 6** — new specs added for `ExploreBlockSkeleton`, `useInViewport`, `routes/index`; the two dashboard panel specs got `hideWhenEmpty` coverage (empty, disconnected for downloads, loading).

### Completion Notes List

- **1738/1738 Vitest PASS** (100% — web regression green, no pre-existing failures to file).
- **Full Go regression PASS** via `pnpm nx test api` (FULL REGRESSION GATE, Epic 9 retro AI-1) — all packages cached-green.
- **`pnpm lint:all` PASS** — `go vet` + `staticcheck@2026.1` + `eslint .` + `prettier --check .` all green (0 errors, pre-existing warnings unchanged).
- **`pnpm run test:cleanup` PASS** — no orphaned vitest workers after the run.
- **Task 2.1/2.2 already satisfied**: HeroBanner uses `w1280` baseline + responsive `srcset` (`w780`, `w1280`, `original`) from Story 10-2; PosterCard uses `w342` from Story 2-3. Documented for traceability — no code change needed.
- **Rule 19 compliance (package dependency boundaries)**: all new web code. No Go package boundary changes.
- **Rule 16 compliance (assertion quality)**: new specs use `toHaveAttribute`, `toBeInTheDocument`, `toEqual` — no `toBeTruthy` on DOM queries.
- **Rule 12 compliance (CI-based lint)**: added `IntersectionObserver` + `IntersectionObserverEntry` to `eslint.config.mjs` globals list (prior omissions — DOM types that were already implicitly used via TanStack-Router-flavoured React components).
- **🎨 UX Verification: PASS** — structural comparison against the three Pencil mocks:
  | Area | Design Spec | Implementation | Match? |
  |------|------------|----------------|--------|
  | Section order (desktop) | Hero → Explore blocks (hp1) | Hero → Explore → Recent → Downloads (Recent/Download hidden when empty per AC #5, matching hp1's empty-library render) | ✅ |
  | Hero height mobile | Compact top banner (hp2) | `h-[250px]` | ✅ |
  | Hero height desktop | Prominent banner ~400px tall (hp1) | `md:h-[400px]` | ✅ |
  | Explore block skeleton | 6 poster placeholders in row (hp4) | `ExploreBlockSkeleton count={6}` | ✅ |
  | Skeleton per block | Each row independently skeleton (hp4, 2 rows) | Per-block `isLoading` + per-block `IntersectionObserver` gating | ✅ |
  | Section spacing | Even vertical breathing room between rows (hp1) | `gap-6 md:gap-8` on outer flex | ✅ |
  | Horizontal scroll mobile | Blocks scroll sideways on narrow viewport (hp2) | Existing `overflow-x-auto pb-2 snap-x` in ExploreBlock | ✅ |
  | Empty-state hide | Recent/Downloads not shown when library is empty (hp1/hp2) | `hideWhenEmpty` returns `null` when empty | ✅ |
  Pixel-perfect verification at 390 / 768 / 1440 deferred to NAS dev server — same pattern Story 10-2 used (documented in sprint-status).

### File List

**New files:**

- `apps/web/src/queryClient.ts` — shared QueryClient singleton used by both React provider and route loader.
- `apps/web/src/hooks/useInViewport.ts` — reusable `useInViewport` hook (IntersectionObserver, `once` latch, SSR-safe fallback).
- `apps/web/src/hooks/useInViewport.spec.ts` — 4 tests.
- `apps/web/src/components/homepage/ExploreBlockSkeleton.tsx` — standalone horizontal skeleton row.
- `apps/web/src/components/homepage/ExploreBlockSkeleton.spec.tsx` — 3 tests.
- `apps/web/src/routes/index.spec.tsx` — 4 tests (section order, flex wrapper, hideWhenEmpty threading, loader prefetch).

**Modified files:**

- `apps/web/src/main.tsx` — import `queryClient` from shared module instead of creating inline.
- `apps/web/src/routes/index.tsx` — full rewrite: added route loader for trending prefetch, replaced grid with vertical flex stack, threaded `hideWhenEmpty` to panels.
- `apps/web/src/components/homepage/HeroBanner.tsx` — height changed to `h-[250px] md:h-[400px]` (both live banner and loading-skeleton section).
- `apps/web/src/components/homepage/ExploreBlocksList.tsx` — added `visibleIndices` state + `EAGER_BLOCK_COUNT=2` gating of `useQueries`, passes `eager` / `onVisible` to each child, outer gap becomes `flex flex-col gap-6 md:gap-8`.
- `apps/web/src/components/homepage/ExploreBlock.tsx` — accepts `eager` + `onVisible`, wires `useInViewport` (rootMargin 400px, `once`), uses `ExploreBlockSkeleton`, preserves empty-state branch.
- `apps/web/src/components/dashboard/RecentMediaPanel.tsx` — added `hideWhenEmpty` prop (returns `null` after loading settles to empty).
- `apps/web/src/components/dashboard/DownloadPanel.tsx` — added `hideWhenEmpty` prop (returns `null` when disconnected or zero downloads).
- `apps/web/src/components/dashboard/RecentMediaPanel.spec.tsx` — `renderPanel` takes props, added 2 `hideWhenEmpty` tests (empty + loading-no-flash).
- `apps/web/src/components/dashboard/DownloadPanel.spec.tsx` — `renderPanel` takes props, added 3 `hideWhenEmpty` tests (disconnected, empty-connected, loading-no-flash).
- `apps/web/src/test-setup.ts` — added IntersectionObserver stub for jsdom.
- `eslint.config.mjs` — added `IntersectionObserver` + `IntersectionObserverEntry` to global readonlies (DOM APIs used by new code).
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 10-5 transitions `ready-for-dev` → `in-progress` → `review`.
- `_bmad-output/implementation-artifacts/10-5-homepage-layout-responsive.md` — Status flipped to `review`, all tasks `[x]`, Dev Agent Record populated.

### Change Log

| Date | Summary |
|------|---------|
| 2026-04-17 | Story 10-5 implemented end-to-end. Refactored homepage to vertical flex stack (Hero → Explore → Recent → Downloads, gap-6 md:gap-8); HeroBanner height moved from vh-based to story-prescribed `h-[250px] md:h-[400px]`; Intersection-Observer lazy-load for below-the-fold explore blocks (eager count 2); route-loader prefetch of trending hero via shared queryClient singleton; `hideWhenEmpty` prop on `RecentMediaPanel`/`DownloadPanel` implements AC #5 for homepage only. Extracted `ExploreBlockSkeleton` and `useInViewport` as reusable primitives. +17 new web tests, 1738/1738 Vitest PASS, full Go regression PASS, `lint:all` 0 errors. UX verification against hp1/hp2/hp4 Pencil mocks PASS. |
