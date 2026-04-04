# Story 11.3: Instant Search with zh-TW Priority

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user,
I want instant search with debounced suggestions that prioritizes Traditional Chinese results,
so that I find content quickly using Chinese or original titles.

## Acceptance Criteria

1. Given the search bar, when the user types ≥2 characters, then debounced suggestions appear within 300ms showing movies, TV shows, and people as separate categories
2. Given a search query in Chinese, when results are returned, then items with zh-TW title matches are ranked above original-title-only matches
3. Given a search query, when TMDB returns results, then the system searches both zh-TW and original titles simultaneously and merges/deduplicates
4. Given the search dropdown, when a user clicks a suggestion, then they navigate to the media detail page
5. Given the search bar on mobile, when activated, then it expands to full-width with a dedicated search view
6. Given an existing Epic 2 search implementation, when enhanced, then backward compatibility with existing search routes is maintained

## Tasks / Subtasks

- [ ] Task 1: Backend — dual-language search + ranking (AC: #2, #3)
  - [ ] 1.1 Extend `SearchMovies` and `SearchTVShows` in TMDB service to query both `zh-TW` and `en` languages
  - [ ] 1.2 Merge results by TMDB ID, prefer zh-TW metadata when available
  - [ ] 1.3 Boost score for items where zh-TW title matches the query (move to top)
  - [ ] 1.4 New endpoint: `GET /api/v1/search?q={query}&page=1` — unified search across movies + TV + local library

- [ ] Task 2: Backend — people search (AC: #1)
  - [ ] 2.1 Add `SearchPeople(ctx, query, page)` to TMDB client → `GET /search/person`
  - [ ] 2.2 Include in unified search response as separate category

- [ ] Task 3: Frontend — search suggestions dropdown (AC: #1, #4)
  - [ ] 3.1 Create `apps/web/src/components/search/SearchSuggestions.tsx`
  - [ ] 3.2 Debounce input (300ms) using `useDebouncedValue` or `useDebounce`
  - [ ] 3.3 Three sections in dropdown: 電影, 影集, 人物
  - [ ] 3.4 Each item: poster thumbnail + title + year
  - [ ] 3.5 Click navigates to `/media/movie.{id}` or `/media/tv.{id}`
  - [ ] 3.6 Keyboard navigation: arrow keys + Enter

- [ ] Task 4: Frontend — enhanced search bar (AC: #5)
  - [ ] 4.1 Replace or enhance existing search bar in toolbar
  - [ ] 4.2 Desktop: inline suggestions dropdown below search bar
  - [ ] 4.3 Mobile: full-screen search overlay on focus
  - [ ] 4.4 Clear button, Escape to close

- [ ] Task 5: Tests (AC: #1-6)
  - [ ] 5.1 Backend: dual-language merge, zh-TW boost ranking
  - [ ] 5.2 Frontend: debounce timing, keyboard navigation, click navigation
  - [ ] 5.3 Backward compatibility: existing `/search` route still works

## Dev Notes

### Architecture Compliance

- **Debounce:** 300ms client-side debounce. Do NOT debounce on server
- **Existing search:** Epic 2 already has `SearchMovies`/`SearchTVShows` with language fallback. Extend, don't replace
- **Unified endpoint:** New `/api/v1/search` aggregates movies + TV + people. Existing per-type endpoints remain
- **zh-TW boost:** Application-layer ranking. If query matches zh-TW `title` field (substring), boost that result's position

### References

- [Source: apps/api/internal/tmdb/movies.go] — SearchMovies, SearchMoviesWithLanguage
- [Source: apps/api/internal/services/tmdb_service.go] — TMDbServiceInterface
- [Source: apps/web/src/routes/search.tsx] — Existing search page
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-013, P2-014

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
