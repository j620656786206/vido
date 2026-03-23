# Story 10.1: Genre-Based Recommendations

Status: ready-for-dev

## Story

As a **media collector**,
I want to **receive recommendations based on my genres**,
So that **I can discover similar content I might enjoy**.

## Acceptance Criteria

1. **Given** the user has media in their library, **When** the system analyzes their collection, **Then** it identifies the top genres by count **And** generates recommendations based on genre overlap.
2. **Given** recommendations are generated, **When** viewing the Dashboard, **Then** a "Recommended for You" section appears **And** shows 6-12 recommendations with posters.
3. **Given** a recommendation is displayed, **When** hovering over it, **Then** the reason is shown: "Because you like [Genre]" **And** clicking opens the detail page.
4. **Given** the user dismisses a recommendation, **When** clicking "Not Interested", **Then** that title is hidden **And** similar content is de-prioritized.

## Tasks / Subtasks

- [ ] Task 1: Create recommendation database tables (AC: #1, #4)
  - [ ] 1.1 Create migration `013_create_recommendation_tables.go` with `user_preferences`, `dismissed_recommendations`, and `recommendation_cache` tables
  - [ ] 1.2 Register migration in `registry.go`
  - [ ] 1.3 Write migration test
- [ ] Task 2: Add TMDb recommendation/discover endpoints to TMDb client (AC: #1)
  - [ ] 2.1 Add `GetMovieRecommendations(ctx, movieID)` to `movies.go`
  - [ ] 2.2 Add `GetTVRecommendations(ctx, tvID)` to `tv.go`
  - [ ] 2.3 Add `DiscoverMovies(ctx, params)` to `movies.go` for genre-based discovery
  - [ ] 2.4 Add `DiscoverTVShows(ctx, params)` to `tv.go` for genre-based discovery
  - [ ] 2.5 Add response types to `types.go` (`DiscoverParams`, `RecommendationResult`)
  - [ ] 2.6 Update `ClientInterface` in `client.go`
  - [ ] 2.7 Update cache layer in `cache.go` for new endpoints
  - [ ] 2.8 Update fallback layer in `fallback.go` for new endpoints
  - [ ] 2.9 Write unit tests for new endpoints
- [ ] Task 3: Create recommendation models (AC: #1, #4)
  - [ ] 3.1 Create `/apps/api/internal/models/recommendation.go` with `RecommendationItem`, `DismissedRecommendation`, `RecommendationCache` structs
- [ ] Task 4: Create recommendation repository (AC: #1, #4)
  - [ ] 4.1 Create `/apps/api/internal/repository/recommendation_repository.go`
  - [ ] 4.2 Add `RecommendationRepositoryInterface` to `interfaces.go`
  - [ ] 4.3 Implement: `GetDismissedByUser`, `DismissRecommendation`, `GetCachedRecommendations`, `SaveCachedRecommendations`, `ClearExpiredCache`
  - [ ] 4.4 Write repository tests
- [ ] Task 5: Create recommendation service (AC: #1, #2, #4)
  - [ ] 5.1 Create `/apps/api/internal/services/recommendation_service.go`
  - [ ] 5.2 Add `RecommendationServiceInterface` to services package
  - [ ] 5.3 Implement `GetGenreBasedRecommendations(ctx, userID)`:
    - Query movie + series repos to analyze user's library genres
    - Rank genres by frequency → pick top 3
    - Use TMDb `/discover/movie` and `/discover/tv` with genre IDs
    - Filter out items already in library (by tmdb_id)
    - Filter out dismissed items
    - Attach reason string: "Because you like [Genre]"
    - Cache results with 24h TTL
  - [ ] 5.4 Implement `DismissRecommendation(ctx, userID, tmdbID, mediaType)`
  - [ ] 5.5 Write service tests (mock TMDb client and repos, coverage >= 80%)
- [ ] Task 6: Create recommendation handler (AC: #2, #3, #4)
  - [ ] 6.1 Create `/apps/api/internal/handlers/recommendation_handler.go`
  - [ ] 6.2 `GET /api/v1/recommendations` → returns genre-based recommendations
  - [ ] 6.3 `POST /api/v1/recommendations/:id/dismiss` → dismiss a recommendation
  - [ ] 6.4 Register routes in handler + wire in `main.go`
  - [ ] 6.5 Write handler tests (coverage >= 70%)
- [ ] Task 7: Create frontend recommendation types and service (AC: #2, #3)
  - [ ] 7.1 Add types to `/apps/web/src/types/recommendation.ts`: `RecommendationItem`, `RecommendationResponse`
  - [ ] 7.2 Create `/apps/web/src/services/recommendation.ts` with `getGenreRecommendations()`, `dismissRecommendation()`
- [ ] Task 8: Create frontend recommendation hooks (AC: #2, #3, #4)
  - [ ] 8.1 Create `/apps/web/src/hooks/useRecommendations.ts`
  - [ ] 8.2 Implement `useGenreRecommendations()` with TanStack Query (staleTime: 30min)
  - [ ] 8.3 Implement `useDismissRecommendation()` mutation with cache invalidation
- [ ] Task 9: Create recommendation UI components (AC: #2, #3, #4)
  - [ ] 9.1 Create `/apps/web/src/components/recommendations/RecommendationCard.tsx` — poster, title, reason tooltip on hover
  - [ ] 9.2 Create `/apps/web/src/components/recommendations/RecommendationSection.tsx` — horizontal scrollable grid of 6-12 cards with "Recommended for You" heading
  - [ ] 9.3 Add "Not Interested" dismiss button to RecommendationCard
  - [ ] 9.4 Clicking card navigates to `/media/{type}/{tmdbId}` detail page
  - [ ] 9.5 Write component tests (spec.tsx co-located)
- [ ] Task 10: Integrate into Dashboard (AC: #2)
  - [ ] 10.1 Add `RecommendationSection` to dashboard/home route
  - [ ] 10.2 Show loading skeleton while fetching
  - [ ] 10.3 Hide section if no recommendations available

## Dev Notes

### Architecture Compliance

- **Layered pattern:** Handler → Service → Repository → Database (NO shortcuts)
- **Logging:** Use `log/slog` exclusively — `slog.Info("Generating recommendations", "user_id", userID, "top_genres", genres)`
- **Error codes:** Use `TMDB_*` for API failures, `DB_*` for database failures, `VALIDATION_*` for input errors
- **API response format:** Wrap all responses in `{"success": true/false, "data": ..., "error": ...}`

### TMDb API Endpoints to Integrate

- `GET /discover/movie?with_genres={ids}&language=zh-TW&sort_by=popularity.desc&page=1` — genre-based movie discovery
- `GET /discover/tv?with_genres={ids}&language=zh-TW&sort_by=popularity.desc&page=1` — genre-based TV discovery
- `GET /movie/{id}/recommendations?language=zh-TW` — movie-specific recommendations (used in Story 10-3)
- Rate limit: 40 req/10s (already enforced by existing client rate limiter)
- Language fallback: zh-TW → zh-CN → en (existing fallback layer handles this)

### Genre ID Mapping

TMDb uses numeric genre IDs. The existing `Genre` struct in `types.go` already has `ID int` and `Name string`. When querying `/discover`, pass genre IDs directly. The user's library stores genres as JSON string arrays of genre names — you need to map names back to TMDb IDs via the existing genre data from movie/series details, or call TMDb `/genre/movie/list` and `/genre/tv/list` once and cache.

### Database Schema (Migration 013)

```sql
CREATE TABLE IF NOT EXISTS user_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT 'default',
    preference_type TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    preference_value TEXT,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, preference_type, preference_key)
);

CREATE TABLE IF NOT EXISTS dismissed_recommendations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT 'default',
    tmdb_id INTEGER NOT NULL,
    media_type TEXT NOT NULL CHECK(media_type IN ('movie', 'tv')),
    dismissed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, tmdb_id, media_type)
);

CREATE TABLE IF NOT EXISTS recommendation_cache (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT 'default',
    cache_type TEXT NOT NULL,
    cache_key TEXT NOT NULL,
    recommendations TEXT NOT NULL,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(user_id, cache_type, cache_key)
);

CREATE INDEX idx_dismissed_user ON dismissed_recommendations(user_id);
CREATE INDEX idx_rec_cache_expires ON recommendation_cache(expires_at);
```

Note: `user_id` defaults to `'default'` since multi-user (Epic 13) is not yet implemented. This prepares for future user support.

### Existing Code to Reuse

- **TMDb client:** `/apps/api/internal/tmdb/client.go` — extend `ClientInterface`, add methods to `movies.go`/`tv.go`
- **TMDb cache layer:** `/apps/api/internal/tmdb/cache.go` — add cache keys for discover/recommendations
- **TMDb language fallback:** `/apps/api/internal/tmdb/fallback.go` — add fallback for new methods
- **Movie/Series repos:** `/apps/api/internal/repository/movie_repository.go`, `series_repository.go` — use `List()` to analyze library genres
- **Response helpers:** `/apps/api/internal/handlers/response.go` — use `SuccessResponse()`, `ErrorResponse()`
- **Frontend fetch pattern:** `/apps/web/src/services/tmdb.ts` — follow same `fetchApi<T>` pattern
- **Frontend hook pattern:** `/apps/web/src/hooks/useSearchMedia.ts` — follow same TanStack Query pattern
- **Component patterns:** `/apps/web/src/components/search/` and `/apps/web/src/components/media/` — follow existing card/grid patterns

### Frontend Patterns

- **TanStack Query keys:** `['recommendations', 'genre']` for genre-based
- **staleTime:** 30 minutes for recommendations (less volatile than search)
- **Mutation:** Use `useMutation` for dismiss with `queryClient.invalidateQueries({ queryKey: ['recommendations'] })`
- **Styling:** Tailwind CSS utility classes. Horizontal scroll: `overflow-x-auto flex gap-4`
- **Hover tooltip:** Use Tailwind `group`/`group-hover` for reason display
- **Navigation:** Use TanStack Router `useNavigate()` or `<Link>` for card clicks

### Project Structure Notes

All new files follow the existing project structure:
- Backend: `/apps/api/internal/{layer}/{feature}_*.go`
- Frontend: `/apps/web/src/{category}/{feature}/Component.tsx`
- Tests co-located: `*_test.go` and `*.spec.tsx` in same directory

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 10 Story 10.1]
- [Source: project-context.md#Mandatory Rules]
- [Source: _bmad-output/planning-artifacts/architecture.md#TMDb Integration]
- [Source: _bmad-output/planning-artifacts/architecture.md#Caching Strategy]
- [Source: apps/api/internal/tmdb/client.go — TMDb client interface]
- [Source: apps/api/internal/tmdb/types.go — TMDb types]
- [Source: apps/api/internal/handlers/response.go — API response format]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
