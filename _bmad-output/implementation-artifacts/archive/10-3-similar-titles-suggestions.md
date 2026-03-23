# Story 10.3: Similar Titles Suggestions

Status: ready-for-dev

## Story

As a **media collector**,
I want to **see similar titles for media I'm viewing**,
So that **I can find related content**.

## Acceptance Criteria

1. **Given** the user views a media detail page, **When** scrolling to the bottom, **Then** "Similar Titles" section shows 6-10 related items **And** items are sourced from TMDb similar/recommendations API.
2. **Given** similar titles are displayed, **When** one is already in the user's library, **Then** it shows "In Your Library" badge **And** clicking goes to the library entry, not external.
3. **Given** similar titles are displayed, **When** one is not in the library, **Then** it shows basic info (title, year, poster) **And** clicking shows a mini-detail modal.
4. **Given** the user is browsing similar titles, **When** they want to add one, **Then** "Add to Wishlist" button is available **And** the wishlist can be exported for future reference.

## Tasks / Subtasks

- [ ] Task 1: Create wishlist database table (AC: #4)
  - [ ] 1.1 Create migration `015_create_wishlist_table.go`
  - [ ] 1.2 Schema: `id TEXT PK`, `user_id TEXT DEFAULT 'default'`, `tmdb_id INTEGER NOT NULL`, `media_type TEXT NOT NULL CHECK('movie'|'tv')`, `title TEXT NOT NULL`, `poster_path TEXT`, `release_date TEXT`, `added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`, `UNIQUE(user_id, tmdb_id, media_type)`
  - [ ] 1.3 Add indexes: `idx_wishlist_user`, `idx_wishlist_tmdb`
  - [ ] 1.4 Register migration in `registry.go`
  - [ ] 1.5 Write migration test
- [ ] Task 2: Add TMDb similar endpoints to TMDb client (AC: #1)
  - [ ] 2.1 Add `GetSimilarMovies(ctx, movieID, page)` to `movies.go`
  - [ ] 2.2 Add `GetSimilarTVShows(ctx, tvID, page)` to `tv.go`
  - [ ] 2.3 Add response types to `types.go` if not already added in 10-1 (`SimilarMoviesResult`, `SimilarTVShowsResult`)
  - [ ] 2.4 Update `ClientInterface`, cache layer, fallback layer
  - [ ] 2.5 Write unit tests
- [ ] Task 3: Create wishlist model (AC: #4)
  - [ ] 3.1 Add `WishlistItem` struct to `/apps/api/internal/models/recommendation.go`
- [ ] Task 4: Create wishlist repository (AC: #4)
  - [ ] 4.1 Extend `/apps/api/internal/repository/recommendation_repository.go` or create separate `wishlist_repository.go`
  - [ ] 4.2 Interface methods: `AddToWishlist`, `RemoveFromWishlist`, `GetWishlist`, `IsInWishlist`, `ExportWishlist`
  - [ ] 4.3 Write repository tests
- [ ] Task 5: Extend recommendation service for similar titles (AC: #1, #2)
  - [ ] 5.1 Add to `/apps/api/internal/services/recommendation_service.go`:
  - [ ] 5.2 `GetSimilarTitles(ctx, mediaType, tmdbID)`:
    - Call TMDb `/movie/{id}/similar` or `/tv/{id}/similar`
    - Cross-reference results with local library (movies/series repos by tmdb_id)
    - Mark each item with `in_library: true/false` and `library_id` if present
    - Cache results with 7-day TTL (similar titles are stable)
    - Return 6-10 items
  - [ ] 5.3 Write service tests (coverage >= 80%)
- [ ] Task 6: Create wishlist service (AC: #4)
  - [ ] 6.1 Create `/apps/api/internal/services/wishlist_service.go`
  - [ ] 6.2 Add `WishlistServiceInterface` to services package
  - [ ] 6.3 `AddToWishlist(ctx, userID, item)`, `RemoveFromWishlist(ctx, userID, tmdbID, mediaType)`
  - [ ] 6.4 `GetWishlist(ctx, userID, params)` — paginated list
  - [ ] 6.5 `ExportWishlist(ctx, userID, format)` — export as JSON or CSV
  - [ ] 6.6 Write service tests (coverage >= 80%)
- [ ] Task 7: Extend handlers (AC: #1, #2, #3, #4)
  - [ ] 7.1 Add to `/apps/api/internal/handlers/recommendation_handler.go`:
  - [ ] 7.2 `GET /api/v1/recommendations/similar/:type/:id` → similar titles with library cross-reference
  - [ ] 7.3 Create `/apps/api/internal/handlers/wishlist_handler.go`:
  - [ ] 7.4 `GET /api/v1/wishlist` → list wishlist items
  - [ ] 7.5 `POST /api/v1/wishlist` → add to wishlist
  - [ ] 7.6 `DELETE /api/v1/wishlist/:type/:tmdb_id` → remove from wishlist
  - [ ] 7.7 `GET /api/v1/wishlist/export?format=json|csv` → export wishlist
  - [ ] 7.8 Register routes, wire in `main.go`
  - [ ] 7.9 Write handler tests (coverage >= 70%)
- [ ] Task 8: Create frontend types and services (AC: #1, #2, #3, #4)
  - [ ] 8.1 Extend `/apps/web/src/types/recommendation.ts`: add `SimilarTitle`, `SimilarTitlesResponse`, `WishlistItem`
  - [ ] 8.2 Extend `/apps/web/src/services/recommendation.ts`: add `getSimilarTitles(type, id)`
  - [ ] 8.3 Create `/apps/web/src/services/wishlist.ts`: `getWishlist()`, `addToWishlist()`, `removeFromWishlist()`, `exportWishlist(format)`
- [ ] Task 9: Create frontend hooks (AC: #1, #2, #3, #4)
  - [ ] 9.1 Extend `/apps/web/src/hooks/useRecommendations.ts`:
  - [ ] 9.2 `useSimilarTitles(type, id)` — key `['recommendations', 'similar', type, id]`, staleTime 1h
  - [ ] 9.3 Create `/apps/web/src/hooks/useWishlist.ts`:
  - [ ] 9.4 `useWishlist()` — key `['wishlist']`
  - [ ] 9.5 `useAddToWishlist()` — mutation with invalidation
  - [ ] 9.6 `useRemoveFromWishlist()` — mutation with invalidation
- [ ] Task 10: Create UI components (AC: #1, #2, #3, #4)
  - [ ] 10.1 Create `/apps/web/src/components/recommendations/SimilarTitlesSection.tsx`:
    - Grid of 6-10 similar title cards
    - Each card: poster, title, year
    - "In Your Library" badge overlay if `in_library === true`
    - Click: library items navigate to `/media/{type}/{id}`, non-library items open mini-detail modal
  - [ ] 10.2 Create `/apps/web/src/components/recommendations/MiniDetailModal.tsx`:
    - Modal dialog with poster, title, year, overview, rating
    - "Add to Wishlist" button
    - Close button
  - [ ] 10.3 Create `/apps/web/src/components/recommendations/WishlistButton.tsx`:
    - Toggle add/remove from wishlist
    - Heart icon with filled/outlined states
  - [ ] 10.4 Write component tests (spec.tsx co-located)
- [ ] Task 11: Integrate into Media Detail Page (AC: #1, #2, #3)
  - [ ] 11.1 Add `SimilarTitlesSection` to bottom of media detail page (`/apps/web/src/routes/media/$type.$id.tsx`)
  - [ ] 11.2 Position: below `RelatedByPersonSection` (from Story 10-2) if present
  - [ ] 11.3 Show loading skeleton while fetching
  - [ ] 11.4 Hide section if no similar titles found

## Dev Notes

### Architecture Compliance

- **Layered pattern:** Handler → Service → Repository → Database (NO shortcuts)
- **Logging:** `slog.Info("Fetching similar titles", "media_type", mediaType, "tmdb_id", tmdbID)`
- **Error codes:** `TMDB_*`, `DB_*`, `VALIDATION_*`
- **API response format:** Standard `{"success": true/false, "data": ..., "error": ...}`

### Dependencies on Story 10-1 and 10-2

This story extends code from both previous stories:
- **10-1:** `recommendation_repository.go`, `recommendation_service.go`, `recommendation_handler.go`, frontend recommendation types/services/hooks
- **10-2:** `RelatedByPersonSection` (for layout ordering on detail page), TMDb person endpoints
- **Shared migration 013:** `recommendation_cache` table reused for similar titles cache

If implementing independently, ensure the foundation from 10-1 exists.

### TMDb API Endpoints to Integrate

- `GET /movie/{movie_id}/similar?language=zh-TW&page=1` — similar movies
- `GET /tv/{tv_id}/similar?language=zh-TW&page=1` — similar TV shows
- Response: standard TMDb paginated list with `results[]` containing `id`, `title`/`name`, `poster_path`, `release_date`/`first_air_date`, `vote_average`, `genre_ids`, `overview`
- Rate limit and language fallback handled by existing TMDb client layers

### Library Cross-Reference Logic

The key feature of this story is marking similar titles that exist in the user's library:

```go
// In recommendation_service.go
func (s *RecommendationService) GetSimilarTitles(ctx context.Context, mediaType string, tmdbID int) (*SimilarTitlesResponse, error) {
    // 1. Fetch from TMDb
    // 2. For each result, check local DB:
    //    - movieRepo.FindByTMDbID(ctx, result.ID) for movies
    //    - seriesRepo.FindByTMDbID(ctx, result.ID) for TV
    // 3. Mark in_library=true and set library_id if found
    // 4. Return enriched results
}
```

Use batch query for efficiency: collect all TMDb IDs, query DB once with `WHERE tmdb_id IN (...)` rather than N+1 queries. You may need to add a `FindByTMDbIDs(ctx, ids []int64)` method to movie/series repos.

### Wishlist Export Formats

- **JSON:** Standard JSON array of wishlist items with all metadata
- **CSV:** Headers: `title, media_type, tmdb_id, release_date, added_at`
- Set `Content-Disposition: attachment; filename="vido-wishlist.{ext}"` header for download

### Existing Code to Reuse/Extend

- **Story 10-1/10-2 outputs:** recommendation service/handler/repository, frontend recommendation infrastructure
- **TMDb client:** Add similar endpoints to existing `movies.go`/`tv.go`
- **Movie/Series repos:** `FindByTMDbID()` already exists — add batch `FindByTMDbIDs()` for cross-referencing
- **Detail page:** `/apps/web/src/routes/media/$type.$id.tsx` — add SimilarTitlesSection below existing content
- **Modal pattern:** Check existing components for modal patterns to follow (e.g., MetadataEditorDialog)

### Frontend Patterns

- **TanStack Query keys:** `['recommendations', 'similar', type, id]` for similar titles, `['wishlist']` for wishlist
- **staleTime:** 1 hour for similar titles (stable per media item)
- **Modal:** Use Tailwind `fixed inset-0 bg-black/50 flex items-center justify-center` for overlay, `bg-white rounded-lg p-6 max-w-md` for content
- **Badge:** `absolute top-2 left-2 bg-green-600 text-white text-xs px-2 py-1 rounded` for "In Your Library"
- **WishlistButton:** Heart icon toggle with `text-red-500` filled / `text-gray-400` outlined states
- **Navigation:** Library items use `<Link to={/media/${type}/${libraryId}}>`, non-library items use `onClick` to open modal

### Project Structure Notes

New files:
- `/apps/api/internal/services/wishlist_service.go` (new service — separate from recommendation)
- `/apps/api/internal/handlers/wishlist_handler.go` (new handler)
- `/apps/api/internal/repository/wishlist_repository.go` (or extend recommendation_repository)
- `/apps/web/src/components/recommendations/SimilarTitlesSection.tsx`
- `/apps/web/src/components/recommendations/MiniDetailModal.tsx`
- `/apps/web/src/components/recommendations/WishlistButton.tsx`
- `/apps/web/src/services/wishlist.ts`
- `/apps/web/src/hooks/useWishlist.ts`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 10 Story 10.3]
- [Source: project-context.md#Mandatory Rules]
- [Source: apps/api/internal/repository/movie_repository.go — FindByTMDbID pattern]
- [Source: apps/api/internal/tmdb/movies.go — TMDb movie endpoint patterns]
- [Source: apps/api/internal/tmdb/tv.go — TMDb TV endpoint patterns]
- [Source: apps/api/internal/handlers/response.go — API response format]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
