# Story 10.3: Custom Explore Blocks CRUD

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user,
I want to create and customize themed content blocks on the homepage (e.g., "近期台灣院線", "熱門韓劇"),
so that the homepage shows discovery content relevant to my interests.

## Acceptance Criteria

1. Given the homepage, when explore blocks are configured, then each block shows a horizontal scrollable row of poster cards with a section title
2. Given the settings/homepage section, when a user creates a new block, then they can specify: name, content type (movie/tv), genre filter, language/region filter, sort order, and max items
3. Given multiple explore blocks, when the user reorders them, then the homepage reflects the new order immediately
4. Given an explore block, when the user edits or deletes it, then the homepage updates without page reload
5. Given a fresh install, when no blocks are configured, then 3 default blocks are created: "熱門電影", "熱門影集", "近期新片"
6. Given explore block data, when fetched from TMDB, then results use the discover API with the block's filter parameters

## Tasks / Subtasks

- [ ] Task 1: Backend — explore blocks CRUD (AC: #2, #3, #4)
  - [ ] 1.1 Create `explore_blocks` table: `id TEXT PK, name TEXT, content_type TEXT, genre_ids TEXT, language TEXT, region TEXT, sort_by TEXT, max_items INT, sort_order INT, created_at, updated_at`
  - [ ] 1.2 Create migration #022 (or next available)
  - [ ] 1.3 Create `ExploreBlockRepository` interface + SQLite implementation
  - [ ] 1.4 API endpoints:
    - `GET /api/v1/explore-blocks` — list all blocks (ordered)
    - `POST /api/v1/explore-blocks` — create block
    - `PUT /api/v1/explore-blocks/:id` — update block
    - `DELETE /api/v1/explore-blocks/:id` — delete block
    - `PUT /api/v1/explore-blocks/reorder` — batch reorder
  - [ ] 1.5 Seed default blocks on first run (AC: #5)

- [ ] Task 2: Backend — explore block content endpoint (AC: #6)
  - [ ] 2.1 `GET /api/v1/explore-blocks/:id/content` — fetch TMDB discover results using block's saved filters
  - [ ] 2.2 Use Story 10-1's `DiscoverMovies`/`DiscoverTVShows` with block params
  - [ ] 2.3 Apply content filters (far-future, low-quality) from Story 10-1's `ContentFilterService`
  - [ ] 2.4 Cache results per block (1-hour TTL)

- [ ] Task 3: Frontend — ExploreBlock component (AC: #1)
  - [ ] 3.1 Create `apps/web/src/components/homepage/ExploreBlock.tsx`
  - [ ] 3.2 Section title + horizontal scrollable row of `PosterCard`
  - [ ] 3.3 "查看更多" link at end of row
  - [ ] 3.4 Loading skeleton while fetching

- [ ] Task 4: Frontend — block management UI (AC: #2, #3, #4)
  - [ ] 4.1 Add "自訂首頁" section in Settings page
  - [ ] 4.2 Create/Edit modal: block name, content type selector, genre multi-select, language/region, sort, max items
  - [ ] 4.3 Drag-to-reorder list (or up/down arrows)
  - [ ] 4.4 Delete with confirmation

- [ ] Task 5: Frontend — homepage integration (AC: #1, #5)
  - [ ] 5.1 Create `apps/web/src/hooks/useExploreBlocks.ts` — fetch blocks + their content
  - [ ] 5.2 Render ExploreBlock for each configured block below HeroBanner
  - [ ] 5.3 TanStack Query with staleTime: 5 minutes

- [ ] Task 6: Tests (AC: #1-6)
  - [ ] 6.1 Backend: repository CRUD tests, handler tests, seed logic
  - [ ] 6.2 Frontend: ExploreBlock render, management UI interactions
  - [ ] 6.3 Integration: create block → verify content appears on homepage

## Dev Notes

### Architecture Compliance

- **DB pattern:** Follow existing repository pattern (`MovieRepositoryInterface` etc.)
- **Migration:** Next available number after Epic 9c's #021
- **API format:** `ApiResponse<T>` wrapper, snake_case (Rule 18)
- **Frontend state:** TanStack Query for server state, no local state management needed
- **Drag reorder:** Use `@dnd-kit/sortable` or simple up/down buttons (simpler for single-user)

### Cross-Stack Split Check

⚠️ This story has 3 backend tasks and 3 frontend tasks. At the boundary but acceptable — the backend CRUD is straightforward and the frontend is a natural consumer. No split recommended.

### References

- [Source: apps/api/internal/services/tmdb_service.go] — TMDB service interface
- [Source: apps/web/src/components/media/PosterCard.tsx] — Card component to reuse
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.4] — P2-002 spec

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
