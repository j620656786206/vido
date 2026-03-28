# Story: Media Detail Page Route Refactor (方案 A)

Status: ready-for-dev

## Story

As a user browsing the media library,
I want to click on any media card and see its detail page,
so that I can view information about the media regardless of whether TMDB metadata has been fetched.

## Acceptance Criteria

1. Clicking a media card in the library grid navigates to a detail page using the internal UUID (not tmdbId)
2. The route `/media/$type/$id` accepts UUID string IDs (not just numeric)
3. The detail page fetches data from the local API (`GET /api/v1/movies/:id`) instead of directly calling TMDB
4. When TMDB metadata exists: display full detail page (poster, overview, ratings, credits)
5. When TMDB metadata is missing: display fallback UI with file info (filename, path, size, parse status) and a "search metadata" prompt
6. The detail page uses a full-page layout, not just a side panel overlay
7. No 404 errors for media that exists in the local database
8. PosterCard component accepts string ID instead of numeric ID

## Tasks / Subtasks

- [ ] Task 1: Update PosterCard and LibraryGrid to use internal UUID (AC: #1, #8)
  - [ ] 1.1 `apps/web/src/components/media/PosterCard.tsx`: Change `id: number` → `id: string` in PosterCardProps
  - [ ] 1.2 `apps/web/src/components/library/LibraryGrid.tsx:30`: Change `id: m.tmdbId ?? 0` → `id: m.id`
  - [ ] 1.3 Update `PosterCard` Link params: `id: String(id)` already works for string

- [ ] Task 2: Update route loader to accept UUID strings (AC: #2, #7)
  - [ ] 2.1 `apps/web/src/routes/media/$type.$id.tsx`: Remove `parseInt` + `numericId <= 0` validation
  - [ ] 2.2 Replace with UUID/string validation: non-empty string check
  - [ ] 2.3 Update loader return type: `id: string` instead of `id: number`

- [ ] Task 3: Create local API data fetching hooks (AC: #3)
  - [ ] 3.1 Create `useLocalMovieDetails(id: string)` hook that calls `GET /api/v1/movies/:id`
  - [ ] 3.2 Create `useLocalSeriesDetails(id: string)` hook that calls `GET /api/v1/series/:id`
  - [ ] 3.3 Apply `snakeToCamel` transformation on API response
  - [ ] 3.4 Return data in format compatible with existing MediaDetailPanel

- [ ] Task 4: Refactor MediaDetailRoute to use local API (AC: #3, #4, #5)
  - [ ] 4.1 Replace `useMovieDetails(numericId)` / `useTVShowDetails(numericId)` with local hooks from Task 3
  - [ ] 4.2 When `tmdbId` exists: optionally fetch TMDB supplemental data (credits, videos) as progressive enhancement
  - [ ] 4.3 When `tmdbId` is missing: skip TMDB calls, show fallback UI

- [ ] Task 5: Implement fallback UI for missing metadata (AC: #5)
  - [ ] 5.1 Create fallback detail view showing: filename, file path, file size, created date, parse status
  - [ ] 5.2 Add "Search Metadata" button that links to manual search
  - [ ] 5.3 Show "Enrichment in progress" indicator if parse_status is "pending"

- [ ] Task 6: Fix detail page layout (AC: #6)
  - [ ] 6.1 Ensure detail page renders as full page, not just side panel
  - [ ] 6.2 Side panel can remain as an overlay option from search results
  - [ ] 6.3 From library grid click → full page detail

- [ ] Task 7: Update tests (AC: all)
  - [ ] 7.1 Update PosterCard tests for string ID
  - [ ] 7.2 Update LibraryGrid tests for UUID-based ID
  - [ ] 7.3 Add tests for local API hooks
  - [ ] 7.4 Update route tests for UUID validation

## Dev Notes

### Root Cause Analysis

The 404 bug occurs because:
1. `LibraryGrid.tsx:30` uses `m.tmdbId ?? 0` — when tmdbId is null (no TMDB metadata), ID becomes 0
2. `PosterCard` generates link `/media/movie/0`
3. Route loader in `$type.$id.tsx` rejects ID ≤ 0 with `notFound()`

Additionally, the detail page calls TMDB API directly (`tmdbService.getMovieDetails(tmdbId)`), which is wrong — it should use local DB as the source of truth.

### Architecture Decision (Party Mode Consensus — 方案 A)

**Single route, single data source:**
- All detail pages use internal UUID as the identifier
- Local DB (`/api/v1/movies/:id`) is the primary data source
- TMDB is used for supplemental data only when tmdbId is available (progressive enhancement)
- No dual-route system — keeps things simple

### Key Files to Modify

| File | Change |
|------|--------|
| `apps/web/src/components/media/PosterCard.tsx` | `id: number` → `id: string` |
| `apps/web/src/components/library/LibraryGrid.tsx` | `m.tmdbId ?? 0` → `m.id` |
| `apps/web/src/routes/media/$type.$id.tsx` | UUID validation + local API + fallback UI + layout |
| `apps/web/src/hooks/useMediaDetails.ts` | Add local API hooks |
| `apps/web/src/services/libraryService.ts` | Add `getMovieById(id)` / `getSeriesById(id)` |

### API Endpoints Available

- `GET /api/v1/movies/:id` — returns full Movie object from local DB
- `GET /api/v1/series/:id` — returns full Series object from local DB
- `GET /api/v1/library/movies/:id/videos` — proxies TMDB videos (needs tmdbId internally)

### Data Shape (from local API, after snakeToCamel)

```typescript
interface LocalMovie {
  id: string;           // UUID — always present
  title: string;        // filename or TMDB title
  originalTitle?: string;
  releaseDate: string;
  genres: string[];
  posterPath?: string;  // null if no metadata
  tmdbId?: number;      // null if no metadata
  overview?: string;
  voteAverage?: number;
  filePath?: string;
  fileSize?: number;
  parseStatus: string;  // "pending" | "success" | "failed" | ""
  metadataSource?: string;
  createdAt: string;
  updatedAt: string;
}
```

### Project Structure Notes

- All code in `/apps/web/src/` (NOT root `/internal/`)
- React + TanStack Router file-based routing
- `snakeToCamel` transform at API boundary (`apps/web/src/utils/caseTransform.ts`)
- Tests co-located: `*.spec.tsx` / `*.spec.ts`

### References

- [Source: apps/web/src/components/library/LibraryGrid.tsx#getItemProps] — tmdbId → 0 fallback
- [Source: apps/web/src/routes/media/$type.$id.tsx] — parseInt validation + TMDB API calls
- [Source: apps/web/src/hooks/useMediaDetails.ts] — direct TMDB API hooks
- [Source: apps/api/internal/handlers/movie_handler.go#GetByID] — local movie API
- [Source: apps/web/src/services/libraryService.ts] — snakeToCamel transform pattern
- [Source: Party Mode discussion 2026-03-28] — 方案 A consensus (Winston, Amelia, Sally)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
