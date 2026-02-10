# Story 10.2: Cast and Director Based Recommendations

Status: ready-for-dev

## Story

As a **media collector**,
I want **recommendations based on actors and directors I follow**,
So that **I discover their other works**.

## Acceptance Criteria

1. **Given** the user's library has multiple works by the same director, **When** generating recommendations, **Then** other works by that director are suggested **And** reason shows: "From director [Name]".
2. **Given** the user's library has multiple works with the same actor, **When** generating recommendations, **Then** other works featuring that actor are suggested **And** reason shows: "[Actor Name] is in this".
3. **Given** recommendations are personalized, **When** viewing a specific media detail, **Then** "More from this director" section appears **And** "More with [Lead Actor]" section appears.
4. **Given** the user explicitly "follows" an actor/director, **When** new content becomes available, **Then** it's highlighted in recommendations **And** optional notification is sent.

## Tasks / Subtasks

- [ ] Task 1: Create followed_people database table (AC: #4)
  - [ ] 1.1 Create migration `014_create_followed_people_table.go`
  - [ ] 1.2 Schema: `id TEXT PK`, `user_id TEXT DEFAULT 'default'`, `person_id INTEGER NOT NULL` (TMDb person ID), `person_name TEXT NOT NULL`, `person_type TEXT` ('actor'|'director'|'both'), `followed_at TIMESTAMP`, `notification_enabled BOOLEAN DEFAULT 0`
  - [ ] 1.3 Add `UNIQUE(user_id, person_id)` constraint and `idx_followed_user`, `idx_followed_person` indexes
  - [ ] 1.4 Register migration in `registry.go`
  - [ ] 1.5 Write migration test
- [ ] Task 2: Add TMDb person credits endpoints to TMDb client (AC: #1, #2, #3)
  - [ ] 2.1 Create `/apps/api/internal/tmdb/people.go` with `GetPersonMovieCredits(ctx, personID)` and `GetPersonTVCredits(ctx, personID)`
  - [ ] 2.2 Add response types to `types.go`: `PersonMovieCredits`, `PersonTVCredits`, `PersonCreditItem`
  - [ ] 2.3 Update `ClientInterface` in `client.go`
  - [ ] 2.4 Update cache layer in `cache.go` with cache key pattern `tmdb:person:{id}:movie_credits`
  - [ ] 2.5 Update fallback layer in `fallback.go`
  - [ ] 2.6 Write unit tests
- [ ] Task 3: Create followed people model (AC: #4)
  - [ ] 3.1 Add `FollowedPerson` struct to `/apps/api/internal/models/recommendation.go` (created in 10-1)
- [ ] Task 4: Create followed people repository (AC: #4)
  - [ ] 4.1 Add to `/apps/api/internal/repository/recommendation_repository.go` (extend from 10-1)
  - [ ] 4.2 Add to `RecommendationRepositoryInterface`: `FollowPerson`, `UnfollowPerson`, `GetFollowedPeople`, `IsFollowing`
  - [ ] 4.3 Write repository tests
- [ ] Task 5: Extend recommendation service for person-based recommendations (AC: #1, #2, #3, #4)
  - [ ] 5.1 Add to `/apps/api/internal/services/recommendation_service.go`:
  - [ ] 5.2 `GetPersonBasedRecommendations(ctx, userID)`:
    - Scan library movies/series `credits_json` to extract top directors and actors by appearance count
    - For top 3-5 people, query TMDb `/person/{id}/movie_credits` and `/person/{id}/tv_credits`
    - Filter out items already in library
    - Filter out dismissed items
    - Attach reason: "From director [Name]" or "[Actor] is in this"
    - Cache results with 24h TTL
  - [ ] 5.3 `GetRelatedByPerson(ctx, mediaType, tmdbID)` — for detail page sections:
    - Extract director and lead actors from the media's `credits_json`
    - Query TMDb person credits for each
    - Return grouped: `{ director: [...], lead_actors: [...] }`
  - [ ] 5.4 `FollowPerson(ctx, userID, personID, personName, personType)` and `UnfollowPerson`
  - [ ] 5.5 `GetFollowedPeopleRecommendations(ctx, userID)`:
    - For each followed person, get their latest credits
    - Highlight new releases (release_date within last 6 months)
  - [ ] 5.6 Write service tests (coverage >= 80%)
- [ ] Task 6: Extend recommendation handler (AC: #1, #2, #3, #4)
  - [ ] 6.1 Add to `/apps/api/internal/handlers/recommendation_handler.go`:
  - [ ] 6.2 `GET /api/v1/recommendations/by-person` → person-based recommendations from library analysis
  - [ ] 6.3 `GET /api/v1/recommendations/related/:type/:id` → "More from director" / "More with actor" for detail page
  - [ ] 6.4 `POST /api/v1/people/:id/follow` → follow a person
  - [ ] 6.5 `DELETE /api/v1/people/:id/follow` → unfollow a person
  - [ ] 6.6 `GET /api/v1/people/followed` → list followed people
  - [ ] 6.7 Register routes in handler, wire in `main.go`
  - [ ] 6.8 Write handler tests (coverage >= 70%)
- [ ] Task 7: Create frontend types and service (AC: #1, #2, #3, #4)
  - [ ] 7.1 Extend `/apps/web/src/types/recommendation.ts`: add `PersonRecommendation`, `FollowedPerson`, `RelatedByPersonResponse`
  - [ ] 7.2 Extend `/apps/web/src/services/recommendation.ts`: add `getPersonRecommendations()`, `getRelatedByPerson(type, id)`, `followPerson()`, `unfollowPerson()`, `getFollowedPeople()`
- [ ] Task 8: Create frontend hooks (AC: #1, #2, #3, #4)
  - [ ] 8.1 Extend `/apps/web/src/hooks/useRecommendations.ts`:
  - [ ] 8.2 `usePersonRecommendations()` — TanStack Query with key `['recommendations', 'person']`
  - [ ] 8.3 `useRelatedByPerson(type, id)` — key `['recommendations', 'related', type, id]`
  - [ ] 8.4 `useFollowPerson()` / `useUnfollowPerson()` — mutations with invalidation
  - [ ] 8.5 `useFollowedPeople()` — key `['people', 'followed']`
- [ ] Task 9: Create UI components (AC: #1, #2, #3, #4)
  - [ ] 9.1 Create `/apps/web/src/components/recommendations/PersonRecommendationSection.tsx` — "Recommended by Cast & Directors" section for dashboard
  - [ ] 9.2 Create `/apps/web/src/components/media/RelatedByPersonSection.tsx` — "More from this director" and "More with [Actor]" sections for detail page
  - [ ] 9.3 Create `/apps/web/src/components/media/FollowButton.tsx` — toggle follow/unfollow for person on detail page credits
  - [ ] 9.4 Add FollowButton to existing `CreditsSection` component or person display
  - [ ] 9.5 Write component tests (spec.tsx co-located)
- [ ] Task 10: Integrate into Dashboard and Detail Page (AC: #2, #3)
  - [ ] 10.1 Add `PersonRecommendationSection` to dashboard (below genre recommendations from 10-1)
  - [ ] 10.2 Add `RelatedByPersonSection` to media detail page (`/apps/web/src/routes/media/$type.$id.tsx`)
  - [ ] 10.3 Show loading skeletons while fetching
  - [ ] 10.4 Hide sections if no data available

## Dev Notes

### Architecture Compliance

- **Layered pattern:** Handler → Service → Repository → Database (NO shortcuts)
- **Logging:** `slog.Info("Analyzing library credits", "user_id", userID, "top_directors", directors)`
- **Error codes:** `TMDB_*` for API failures, `DB_*` for database failures
- **API response format:** Standard `{"success": true/false, "data": ..., "error": ...}`

### Dependency on Story 10-1

This story **extends** code created in Story 10-1:
- Database migration 013 creates `recommendation_cache` and `dismissed_recommendations` tables (reused here)
- `recommendation_repository.go`, `recommendation_service.go`, `recommendation_handler.go` are extended
- Frontend `recommendation.ts` types, services, hooks are extended
- If 10-1 is not yet implemented, this story must create those foundational files

### TMDb API Endpoints to Integrate

- `GET /person/{person_id}/movie_credits?language=zh-TW` — person's movie filmography
- `GET /person/{person_id}/tv_credits?language=zh-TW` — person's TV filmography
- Response includes `cast[]` and `crew[]` arrays with `id`, `title`, `poster_path`, `release_date`, `vote_average`
- Rate limit: 40 req/10s (existing rate limiter)
- Language fallback: zh-TW → zh-CN → en (existing fallback layer)

### Credits JSON Parsing

The existing `movies` and `series` tables store credits as JSON in `credits_json` column. Use the existing model helper methods (`GetCredits()`) to parse:

```go
// Existing pattern in models/movie.go
type Credits struct {
    Cast []CastMember `json:"cast"`
    Crew []CrewMember `json:"crew"`
}
type CastMember struct {
    ID        int    `json:"id"`
    Name      string `json:"name"`
    Character string `json:"character"`
    Order     int    `json:"order"`
}
type CrewMember struct {
    ID         int    `json:"id"`
    Name       string `json:"name"`
    Job        string `json:"job"`
    Department string `json:"department"`
}
```

To find top directors: filter `Crew` where `Job == "Director"`, count by `ID` across all library items.
To find top actors: use `Cast` sorted by `Order`, count by `ID` across all library items.

### Existing Code to Reuse/Extend

- **Story 10-1 outputs:** `recommendation_repository.go`, `recommendation_service.go`, `recommendation_handler.go`, frontend hooks/services/components
- **TMDb client:** `/apps/api/internal/tmdb/client.go` — add person methods to new `people.go`
- **Credits data:** `/apps/api/internal/models/movie.go` and `series.go` — `GetCredits()` helper already parses `credits_json`
- **CreditsSection component:** `/apps/web/src/components/media/CreditsSection.tsx` — add FollowButton integration
- **Detail page:** `/apps/web/src/routes/media/$type.$id.tsx` — add RelatedByPersonSection

### Frontend Patterns

- **TanStack Query keys:** `['recommendations', 'person']` for dashboard, `['recommendations', 'related', type, id]` for detail page
- **Mutations:** `useMutation` for follow/unfollow with `invalidateQueries(['people', 'followed'])`
- **Styling:** Tailwind CSS. Section layout: `space-y-6`, card grid: `grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4`
- **FollowButton:** Toggle style with `bg-blue-600`/`bg-gray-200` states

### Project Structure Notes

All new files follow the existing project structure:
- New TMDb file: `/apps/api/internal/tmdb/people.go` (new — follows movies.go/tv.go pattern)
- Extended files: recommendation_repository, recommendation_service, recommendation_handler
- New frontend components in `/apps/web/src/components/recommendations/` and `/apps/web/src/components/media/`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 10 Story 10.2]
- [Source: project-context.md#Mandatory Rules]
- [Source: apps/api/internal/models/movie.go — Credits/CastMember/CrewMember structs]
- [Source: apps/api/internal/tmdb/types.go — TMDb types]
- [Source: apps/api/internal/tmdb/client.go — ClientInterface to extend]
- [Source: apps/api/internal/handlers/response.go — API response format]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
