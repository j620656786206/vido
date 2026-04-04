# Story 10.5: Homepage Layout Engine & Responsive Design

Status: ready-for-dev

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

- [ ] Task 1: Refactor homepage layout (AC: #1)
  - [ ] 1.1 Refactor `apps/web/src/routes/index.tsx` to compose: HeroBanner → ExploreBlockList → RecentMediaPanel → DownloadPanel
  - [ ] 1.2 Each section is an independent component with its own data fetching
  - [ ] 1.3 Wrap in a vertical flex layout with consistent section spacing (gap-8 desktop, gap-6 mobile)

- [ ] Task 2: Performance optimization (AC: #2)
  - [ ] 2.1 Hero banner backdrop: use TMDB `w1280` size (not `original`) for faster load
  - [ ] 2.2 Explore block posters: use `w342` size (existing PosterCard behavior)
  - [ ] 2.3 Lazy-load below-the-fold explore blocks with Intersection Observer
  - [ ] 2.4 Prefetch trending data on route hover (TanStack Router prefetch)

- [ ] Task 3: Skeleton loading states (AC: #4)
  - [ ] 3.1 Create `apps/web/src/components/homepage/ExploreBlockSkeleton.tsx`
  - [ ] 3.2 Horizontal row of PosterCardSkeleton (reuse existing)
  - [ ] 3.3 Each block shows its own skeleton independently

- [ ] Task 4: Responsive breakpoints (AC: #3)
  - [ ] 4.1 Hero banner: h-[400px] desktop → h-[250px] mobile
  - [ ] 4.2 Explore blocks: 6 cards visible desktop → horizontal scroll mobile with snap
  - [ ] 4.3 Section titles: text-xl desktop → text-lg mobile
  - [ ] 4.4 Test on 390px (iPhone), 768px (tablet), 1440px (desktop)

- [ ] Task 5: Empty state handling (AC: #5)
  - [ ] 5.1 Each section returns `null` when data is empty or failed
  - [ ] 5.2 No "empty state" UI for homepage sections — just hide

- [ ] Task 6: Tests (AC: #1-5)
  - [ ] 6.1 Homepage layout: verify section ordering
  - [ ] 6.2 Skeleton states: verify per-block loading
  - [ ] 6.3 Empty sections: verify hidden when no data

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

### Debug Log References

### Completion Notes List

### File List
