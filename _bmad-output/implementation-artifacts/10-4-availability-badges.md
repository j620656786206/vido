# Story 10.4: Availability Badges (已有/已請求)

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user browsing trending content,
I want to see badges on media cards indicating whether I already own the content or have requested it,
so that I don't accidentally request content I already have.

## Acceptance Criteria

1. Given a media card on the homepage (trending or explore block), when the item exists in my library (matched by TMDB ID), then an "已有" badge is displayed on the poster card
2. Given a media card, when the item has a pending request (future: request system), then a "已請求" badge is displayed
3. Given the badge display, when rendered, then badges are small pill-shaped overlays at the top-right corner of the poster card, visually consistent with existing badge patterns
4. Given a large number of trending results, when checking ownership, then the lookup is batched (single DB query for all TMDB IDs on the page) to avoid N+1 queries
5. Given Phase 3 (request system) is not yet built, when the "已請求" badge logic is implemented, then it is stubbed to always return false (ready for future integration)

## Tasks / Subtasks

- [ ] Task 1: Backend — ownership lookup endpoint (AC: #1, #4)
  - [ ] 1.1 `POST /api/v1/movies/check-owned` — body: `{ tmdb_ids: [123, 456, ...] }` → returns `{ owned_ids: [123] }`
  - [ ] 1.2 Single SQL query: `SELECT tmdb_id FROM movies WHERE tmdb_id IN (?) AND is_removed = 0`
  - [ ] 1.3 Also check series table: `SELECT tmdb_id FROM series WHERE tmdb_id IN (?)`
  - [ ] 1.4 Merge results, deduplicate

- [ ] Task 2: Frontend — badge component (AC: #3)
  - [ ] 2.1 Create `apps/web/src/components/media/AvailabilityBadge.tsx`
  - [ ] 2.2 Two variants: "已有" (green) and "已請求" (amber)
  - [ ] 2.3 Position: absolute top-right of poster card, small pill shape
  - [ ] 2.4 Consistent with existing `PosterCard` styling

- [ ] Task 3: Frontend — ownership hook + PosterCard integration (AC: #1, #5)
  - [ ] 3.1 Create `apps/web/src/hooks/useOwnedMedia.ts` — batch check owned TMDB IDs
  - [ ] 3.2 Call `POST /api/v1/movies/check-owned` with all visible TMDB IDs
  - [ ] 3.3 Extend `PosterCard` props: add optional `isOwned`, `isRequested` booleans
  - [ ] 3.4 Render `AvailabilityBadge` when either is true
  - [ ] 3.5 Stub `isRequested` to false (AC: #5) — placeholder for Phase 3

- [ ] Task 4: Tests (AC: #1-5)
  - [ ] 4.1 Backend: check-owned endpoint with mock DB data
  - [ ] 4.2 Frontend: badge render variants, PosterCard with badges
  - [ ] 4.3 Hook: batch query behavior, empty results

## Dev Notes

### Architecture Compliance

- **Batch pattern:** Single POST with array of IDs — avoids N+1 queries. Follow existing batch patterns
- **PosterCard modification:** Minimal change — add 2 optional boolean props and conditional badge render
- **Request system stub:** `isRequested` is always false until Phase 3 Epic (P3-001). The badge component and prop are ready, just not populated

### References

- [Source: apps/web/src/components/media/PosterCard.tsx] — Card to extend
- [Source: apps/api/internal/handlers/movie_handler.go] — Handler pattern
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.4] — P2-006 spec

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
