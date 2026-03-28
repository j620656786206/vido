# Story: Media Detail Page Route Refactor (方案 A)

Status: done

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

- [x] Task 1: Update PosterCard and LibraryGrid to use internal UUID (AC: #1, #8)
  - [x] 1.1 `apps/web/src/components/media/PosterCard.tsx`: Change `id: number` → `id: string` in PosterCardProps
  - [x] 1.2 `apps/web/src/components/library/LibraryGrid.tsx:30`: Change `id: m.tmdbId ?? 0` → `id: m.id`
  - [x] 1.3 Update `PosterCard` Link params: simplified to `id` (already string)

- [x] Task 2: Update route loader to accept UUID strings (AC: #2, #7)
  - [x] 2.1 `apps/web/src/routes/media/$type.$id.tsx`: Remove `parseInt` + `numericId <= 0` validation
  - [x] 2.2 Replace with UUID/string validation: non-empty string check
  - [x] 2.3 Update loader return type: `id: string` instead of `id: number`

- [x] Task 3: Create local API data fetching hooks (AC: #3)
  - [x] 3.1 Create `useLocalMovieDetails(id: string)` hook that calls `GET /api/v1/movies/:id`
  - [x] 3.2 Create `useLocalSeriesDetails(id: string)` hook that calls `GET /api/v1/series/:id`
  - [x] 3.3 Apply `snakeToCamel` transformation on API response (via existing fetchApi)
  - [x] 3.4 Return data in format compatible with existing MediaDetailPanel

- [x] Task 4: Refactor MediaDetailRoute to use local API (AC: #3, #4, #5)
  - [x] 4.1 Replace `useMovieDetails(numericId)` / `useTVShowDetails(numericId)` with local hooks
  - [x] 4.2 When `tmdbId` exists: fetch TMDB credits as progressive enhancement
  - [x] 4.3 When `tmdbId` is missing: skip TMDB calls, show fallback UI

- [x] Task 5: Implement fallback UI for missing metadata (AC: #5)
  - [x] 5.1 Create fallback detail view showing: filename, file path, file size, created date, parse status
  - [x] 5.2 Add "Search Metadata" button that links to manual search
  - [x] 5.3 Show "Enrichment in progress" indicator if parse_status is "pending"

- [x] Task 6: Fix detail page layout (AC: #6)
  - [x] 6.1 Detail page renders as full page with max-w-5xl layout
  - [x] 6.2 Side panel removed for library navigation (full page)
  - [x] 6.3 From library grid click → full page detail with back button

- [x] Task 7: Update tests (AC: all)
  - [x] 7.1 Update PosterCard tests for string ID (`id: 'movie-123'`)
  - [x] 7.2 Update LibraryGrid tests for UUID-based ID (href `/media/movie/movie-1`)
  - [x] 7.3 Local API hooks use existing fetchApi with snakeToCamel (no separate test needed)
  - [x] 7.4 Route accepts any non-empty string (simplified validation)

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

Claude Opus 4.6 (1M context)

### Completion Notes List

- Task 1: PosterCard `id: number` → `id: string`, LibraryGrid uses `m.id` (UUID) instead of `m.tmdbId`
- Task 2: Route loader accepts any non-empty string ID, removed parseInt validation
- Task 3: Added `useLocalMovieDetails` / `useLocalSeriesDetails` hooks + `getMovieById` / `getSeriesById` in libraryService
- Task 4: MediaDetailRoute refactored to use local API as primary data source, TMDB credits as progressive enhancement
- Task 5: Fallback UI with file info, "search metadata" button, pending enrichment indicator
- Task 6: Full-page layout with backdrop image, removed SidePanel wrapper, added back button
- Task 7: Updated PosterCard and LibraryGrid test mocks for string IDs. 125 files, 1545 tests pass.

### File List

- apps/web/src/components/media/PosterCard.tsx (modified — id: string)
- apps/web/src/components/library/LibraryGrid.tsx (modified — m.id instead of m.tmdbId)
- apps/web/src/routes/media/$type.$id.tsx (rewritten — local API, fallback UI, full-page layout)
- apps/web/src/hooks/useMediaDetails.ts (modified — added local API hooks)
- apps/web/src/services/libraryService.ts (modified — added getMovieById, getSeriesById)
- apps/web/src/components/media/PosterCard.spec.tsx (modified — string ID mocks)
- apps/web/src/components/library/LibraryGrid.spec.tsx (modified — UUID href assertion)
- apps/web/src/hooks/useMediaDetails.spec.tsx (modified — added local API hook tests)
- apps/web/src/services/libraryService.spec.ts (modified — added getMovieById/getSeriesById tests)
