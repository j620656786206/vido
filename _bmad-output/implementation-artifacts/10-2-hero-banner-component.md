# Story 10.2: Hero Banner Carousel Component

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user visiting the homepage,
I want a visually striking hero banner showcasing trending content,
so that I can discover popular movies and TV shows at a glance.

## Acceptance Criteria

1. Given the homepage loads, when trending data is available, then a full-width hero banner displays with backdrop image, title, year, rating, and overview
2. Given multiple trending items, when the banner is active, then it auto-rotates every 8 seconds with smooth crossfade transition
3. Given the hero banner, when a user clicks on an item, then they are navigated to the media detail page
4. Given the hero banner on mobile, when displayed, then it adapts to a compact layout with smaller backdrop and truncated overview
5. Given trending API returns empty or fails, when the homepage loads, then the hero banner section is gracefully hidden (no error shown)
6. Given the banner has a trailer available (from TMDB videos), when the user clicks a play button, then an embedded YouTube trailer opens in a modal

## Tasks / Subtasks

- [ ] Task 1: HeroBanner component (AC: #1, #2, #4)
  - [ ] 1.1 Create `apps/web/src/components/homepage/HeroBanner.tsx`
  - [ ] 1.2 Full-width backdrop image from TMDB `backdrop_path`
  - [ ] 1.3 Gradient overlay (bottom) for text readability
  - [ ] 1.4 Content: title (zh-TW), year, rating badge, truncated overview
  - [ ] 1.5 Auto-rotation with `useInterval` (8s), pause on hover
  - [ ] 1.6 Dot indicators for manual navigation
  - [ ] 1.7 Mobile responsive: reduce height, truncate overview to 2 lines

- [ ] Task 2: Trending data hook (AC: #1, #5)
  - [ ] 2.1 Create `apps/web/src/hooks/useTrending.ts`
  - [ ] 2.2 TanStack Query: `GET /api/v1/tmdb/trending/movies?time_window=week`
  - [ ] 2.3 Merge movies + TV trending, take top 5 for banner
  - [ ] 2.4 Handle loading/error states gracefully (AC: #5)

- [ ] Task 3: Trailer modal (AC: #6)
  - [ ] 3.1 Create `apps/web/src/components/homepage/TrailerModal.tsx`
  - [ ] 3.2 YouTube embed via iframe with `autoplay=1`
  - [ ] 3.3 Fetch trailer from `GET /api/v1/tmdb/movies/:id/videos` (already exists in client)
  - [ ] 3.4 Close on backdrop click or Escape key

- [ ] Task 4: Integration into homepage (AC: #1)
  - [ ] 4.1 Add `HeroBanner` to `apps/web/src/routes/index.tsx` as first section
  - [ ] 4.2 Position above existing `RecentMediaPanel`

- [ ] Task 5: Tests (AC: #1-6)
  - [ ] 5.1 HeroBanner: render with mock data, auto-rotation logic
  - [ ] 5.2 useTrending: mock API, error/empty states
  - [ ] 5.3 TrailerModal: open/close behavior

## Dev Notes

### Architecture Compliance

- **Component location:** `apps/web/src/components/homepage/` — new directory for homepage-specific components
- **Hook pattern:** Follow existing `useSearch.ts`, `useDownloads.ts` patterns in `apps/web/src/hooks/`
- **Styling:** Tailwind CSS (architecture decision #1)
- **Image loading:** Use TMDB image URL pattern: `https://image.tmdb.org/t/p/original{backdrop_path}`
- **Route:** No new route — extend existing `/` (index) route

### References

- [Source: apps/web/src/components/media/PosterCard.tsx] — Existing card component pattern
- [Source: apps/web/src/routes/index.tsx] — Current homepage layout
- [Source: apps/api/internal/tmdb/movies.go] — GetMovieVideos method (trailers)
- [Source: _bmad-output/planning-artifacts/prd/project-scoping-phased-development.md#2.1] — P2-001 spec

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
