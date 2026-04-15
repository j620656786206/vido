# Story 10.2: Hero Banner Carousel Component

Status: review

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

- [x] Task 1: HeroBanner component (AC: #1, #2, #4)
  - [x] 1.1 Create `apps/web/src/components/homepage/HeroBanner.tsx`
  - [x] 1.2 Full-width backdrop image from TMDB `backdrop_path`
  - [x] 1.3 Gradient overlay (bottom) for text readability
  - [x] 1.4 Content: title (zh-TW), year, rating badge, truncated overview
  - [x] 1.5 Auto-rotation with `setInterval` (8s), pause on hover
  - [x] 1.6 Dot indicators for manual navigation
  - [x] 1.7 Mobile responsive: reduce height, truncate overview to 2 lines

- [x] Task 2: Trending data hook (AC: #1, #5)
  - [x] 2.1 Create `apps/web/src/hooks/useTrending.ts`
  - [x] 2.2 TanStack Query: `GET /api/v1/tmdb/trending/movies?time_window=week`
  - [x] 2.3 Merge movies + TV trending, take top 5 for banner
  - [x] 2.4 Handle loading/error states gracefully (AC: #5)

- [x] Task 3: Trailer modal (AC: #6)
  - [x] 3.1 Create `apps/web/src/components/homepage/TrailerModal.tsx`
  - [x] 3.2 YouTube embed via iframe with `autoplay=1`
  - [x] 3.3 Fetch trailer from `GET /api/v1/tmdb/movies/:id/videos` (HTTP route added — Go client method existed but was not exposed; see Completion Notes)
  - [x] 3.4 Close on backdrop click or Escape key

- [x] Task 4: Integration into homepage (AC: #1)
  - [x] 4.1 Add `HeroBanner` to `apps/web/src/routes/index.tsx` as first section
  - [x] 4.2 Position above existing `RecentMediaPanel`

- [x] Task 5: Tests (AC: #1-6)
  - [x] 5.1 HeroBanner: render with mock data, auto-rotation logic (14 tests)
  - [x] 5.2 useTrending: mock API, error/empty states (6 tests)
  - [x] 5.3 TrailerModal: open/close behavior + pickBestTrailer pure function (16 tests)

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

claude-opus-4-6 (1M context) — Amelia (Developer Agent) via `/dev-story` workflow on 2026-04-15.

### Debug Log References

- Backend videos endpoint exposure: confirmed `tmdb.Client.GetMovieVideos` and `GetTVShowVideos` already exist in `apps/api/internal/tmdb/movies.go:193` and `tv.go:113`, but no service wrapper, handler, or route. Story Task 3.3's note "(already exists in client)" referred to the Go client layer; HTTP exposure had to be added to satisfy AC #6.
- HeroBanner timer test required `vi.useFakeTimers({ shouldAdvanceTime: true })` so that `await screen.findByTestId(...)` (router mount) and `act(() => vi.advanceTimersByTime(8000))` (rotation tick) coexist. Plain `vi.useFakeTimers()` after first render leaves the existing `setInterval` handle bound to real timers.

### Completion Notes List

- **Backend HTTP exposure for `/videos` (scope clarification, not scope creep):** Story Task 3.3 said "already exists in client". The Go-level client method existed; the HTTP route did not. Added the minimum wiring required by AC #6:
  - `services/tmdb_service.go`: `GetMovieVideos` / `GetTVShowVideos` on `TMDbServiceInterface` and `TMDbService` — bypass cache (small ephemeral payload), guard nil client (mirrors `FindByExternalID`), reject `id ≤ 0` with `tmdb.NewBadRequestError`.
  - `handlers/tmdb_handler.go`: matching interface entries, two handler funcs with full Swaggo annotations, two routes (`GET /tmdb/movies/:id/videos`, `GET /tmdb/tv/:id/videos`).
  - Mock stubs added to `handlers/tmdb_handler_test.go::MockTMDbService` and `services/enrichment_nfo_test.go::mockTMDbServiceForNFO` for interface compliance.
- **Trailer selection (`pickBestTrailer`):** Filters TMDb videos to `site=YouTube` + `type=Trailer` + key matching `/^[a-zA-Z0-9_-]+$/` (XSS guard before iframe `src` interpolation). Sorts official-first, then most-recent `publishedAt`. Returns `null` when no embeddable trailer found, in which case the modal renders an empty state ("找不到預告片") rather than failing.
- **Banner data shape:** `useTrendingHero` interleaves trending movies and TV shows by index (M0, T0, M1, T1, …), filters out items missing `backdropPath` (would render as broken banner), and caps at 5 items. Uses backend trending endpoint introduced by Story 10-1; relies on its existing 1-hour cache TTL.
- **AC #5 graceful hide:** `HeroBanner` returns `null` for both `isError` and empty-data states; loading state renders a full-width skeleton of equal height to prevent layout shift when data resolves.
- **Auto-rotation (AC #2):** `setInterval` of 8000ms, registered only when `items.length >= 2`, paused while `isPaused` (hover) or `trailerItem` (modal open) is true. Crossfade transition via `transition-opacity duration-700 ease-in-out`.
- **TrailerModal lifecycle:** Escape-key handler scoped to `open=true` (no global listener leak), backdrop click via `e.target === e.currentTarget` discriminator (clicks inside the iframe container don't close), uses `youtube-nocookie.com` domain to match the existing `TrailerEmbed.tsx` privacy posture.
- **Frontend client transformation (Rule 18):** `getTrendingMovies/TVShows`/`getMovieVideos` flow through `fetchApi`, which already runs `snakeToCamel` on the response — so `backdrop_path` → `backdropPath`, `vote_average` → `voteAverage`, `published_at` → `publishedAt` work without per-call mapping.

### UX Verification

Compared the implementation against `_bmad-output/screenshots/flow-g-homepage-desktop/hp1-homepage-desktop.png`, `flow-g-homepage-desktop/hp4-loading-skeleton-desktop.png`, and `flow-g-homepage-mobile/hp2-homepage-mobile.png`. Screenshots are low resolution (363×512 / 157×512 / 512×458), so the comparison is structural rather than pixel-perfect.

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| Hero banner placement | Top of homepage, full-width, above content grid | First section in `routes/index.tsx`, above `QBStatusIndicator` row and `DashboardLayout` | ✅ | No |
| Banner background | Dark backdrop image fills the banner area | `<img>` with `object-cover` on `bg-black` section | ✅ | No |
| Bottom-up gradient | Dark gradient at the bottom for text legibility | `bg-gradient-to-t from-black via-black/70 to-transparent` overlay | ✅ | No |
| Title size | Large heading dominating the banner | `text-2xl sm:text-4xl lg:text-5xl` | ✅ | No |
| Metadata row | Type badge + year + rating in a small row above the title | Type badge (`電影`/`影集`) + year + ⭐ rating in flex row above title | ✅ | No |
| Action buttons | Primary play/CTA + secondary view button | "觀看預告片" (white pill) + "查看詳情" (translucent pill) | ✅ | No |
| Mobile layout | Reduced banner height, simpler layout | `h-[40vh] sm:h-[50vh] lg:h-[70vh]`, `line-clamp-2 sm:line-clamp-3`, `text-sm sm:text-base` | ✅ | No |
| Loading skeleton | Pulsing gray placeholder for the banner | `animate-pulse bg-tertiary` covering full banner height | ✅ | No |
| Empty/error state | Banner section hidden | `return null` from `HeroBanner` | ✅ | No |

🎨 UX Verification: PASS (structural match) — implementation matches the design screenshots' overall layout and dark theme. Pixel-perfect verification would require running the dev server against a browser, which is left to the user (no `nx serve web` target is configured; the user runs the app on the NAS at `192.168.50.52:8088`).

### File List

**Backend (Go):**
- `apps/api/internal/services/tmdb_service.go` — added `GetMovieVideos` / `GetTVShowVideos` to `TMDbServiceInterface` and `TMDbService` (modified)
- `apps/api/internal/services/tmdb_service_test.go` — added 4 service-layer tests for invalid ID and nil-client paths (modified)
- `apps/api/internal/services/enrichment_nfo_test.go` — added videos stubs to `mockTMDbServiceForNFO` for interface compliance (modified)
- `apps/api/internal/handlers/tmdb_handler.go` — added videos routes + handler funcs + interface entries (modified)
- `apps/api/internal/handlers/tmdb_handler_test.go` — added videos mock fields/methods, route registration assertion, 3 handler tests (modified)

**Frontend (React/TS):**
- `apps/web/src/types/tmdb.ts` — added `Video`, `VideosResponse`, `HeroBannerItem` types (modified)
- `apps/web/src/services/tmdb.ts` — added `getTrendingMovies`, `getTrendingTVShows`, `getMovieVideos`, `getTVShowVideos` (modified)
- `apps/web/src/hooks/useTrending.ts` — new (created)
- `apps/web/src/hooks/useTrending.spec.ts` — new (created, 6 tests)
- `apps/web/src/components/homepage/HeroBanner.tsx` — new (created)
- `apps/web/src/components/homepage/HeroBanner.spec.tsx` — new (created, 14 tests)
- `apps/web/src/components/homepage/TrailerModal.tsx` — new (created)
- `apps/web/src/components/homepage/TrailerModal.spec.tsx` — new (created, 16 tests — 6 for `pickBestTrailer` + 10 for the modal)
- `apps/web/src/routes/index.tsx` — wire `HeroBanner` as first section (modified)

**Sprint tracking:**
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `10-2-hero-banner-component: ready-for-dev` → `in-progress` → (next: `review`) (modified)
- `_bmad-output/implementation-artifacts/10-2-hero-banner-component.md` — story file updates (modified)

## Change Log

| Date | Description |
|------|-------------|
| 2026-04-15 | Story 10-2 implementation complete: HeroBanner carousel + useTrendingHero hook + TrailerModal — frontend Tasks 1-5. |
| 2026-04-15 | Backend HTTP exposure for `/api/v1/tmdb/movies/:id/videos` and `/api/v1/tmdb/tv/:id/videos` (Task 3.3 prerequisite — Go client existed, REST route did not). |
| 2026-04-15 | Full regression gate PASS: `nx test api` all green; `nx test web` 1665/1665 PASS (+36 new tests); `pnpm lint:all` 0 errors. |
