# Story 2.4: Media Detail Page

Status: ready-for-dev

## Story

As a **media collector**,
I want to **view detailed information about a movie or TV show**,
So that **I can learn more before adding it to my library**.

## Acceptance Criteria

1. **Given** the user clicks on a search result
   **When** the detail page loads
   **Then** it displays:
   - Full Traditional Chinese title and original title
   - High-resolution poster
   - Release year and runtime
   - Genre tags
   - Director and main cast
   - Plot summary in Traditional Chinese
   - TMDb rating

2. **Given** the media is a TV show
   **When** viewing the detail page
   **Then** it also displays:
   - Number of seasons and episodes
   - Air date information
   - Network/streaming platform
   - Created by information

3. **Given** the detail page is loading
   **When** data is being fetched
   **Then** a loading skeleton is displayed
   **And** the page transition completes within 200ms (NFR-P11)

4. **Given** the user is on desktop
   **When** viewing media details
   **Then** details appear in a side panel (Spotify-style)
   **And** the main content remains visible

5. **Given** the user is on mobile
   **When** viewing media details
   **Then** details appear in a full-screen modal or new page

## Tasks / Subtasks

### Task 1: Create Media Detail Route (AC: #1, #2, #3)
- [ ] 1.1 Create `apps/web/src/routes/media/$type.$id.tsx` with TanStack Router
- [ ] 1.2 Validate `$type` parameter (movie | tv)
- [ ] 1.3 Implement route loader for prefetching data
- [ ] 1.4 Handle invalid routes with 404 page

### Task 2: Create TMDb Detail API Hooks (AC: #1, #2)
- [ ] 2.1 Add `getMovieDetails(id: number)` to TMDb service
- [ ] 2.2 Add `getTVShowDetails(id: number)` to TMDb service
- [ ] 2.3 Add `getMovieCredits(id: number)` for cast/director
- [ ] 2.4 Add `getTVShowCredits(id: number)` for cast/creator
- [ ] 2.5 Create `useMovieDetails` TanStack Query hook
- [ ] 2.6 Create `useTVShowDetails` TanStack Query hook
- [ ] 2.7 Create `useMediaCredits` hook for cast information

### Task 3: Create SidePanel Component (AC: #4)
- [ ] 3.1 Create `apps/web/src/components/ui/SidePanel.tsx`
- [ ] 3.2 Implement slide-in animation from right (300ms)
- [ ] 3.3 Set width to 400-500px on desktop
- [ ] 3.4 Add close button and click-outside-to-close
- [ ] 3.5 Add keyboard support (Escape to close)
- [ ] 3.6 Implement backdrop overlay with blur

### Task 4: Create MediaDetailPanel Component (AC: #1, #2, #3)
- [ ] 4.1 Create `apps/web/src/components/media/MediaDetailPanel.tsx`
- [ ] 4.2 Display high-resolution poster (w500 size)
- [ ] 4.3 Display backdrop image as header (if available)
- [ ] 4.4 Show title (zh-TW) and original title
- [ ] 4.5 Display year, runtime, and rating
- [ ] 4.6 Show genre tags as chips
- [ ] 4.7 Display plot overview (zh-TW)
- [ ] 4.8 Add loading skeleton state

### Task 5: Create Credits Section (AC: #1, #2)
- [ ] 5.1 Create `apps/web/src/components/media/CreditsSection.tsx`
- [ ] 5.2 Display director (for movies) or created by (for TV)
- [ ] 5.3 Show top 6 cast members with profile pictures
- [ ] 5.4 Display character names under actor names
- [ ] 5.5 Handle missing profile images gracefully

### Task 6: Create TV Show Specific Section (AC: #2)
- [ ] 6.1 Create `apps/web/src/components/media/TVShowInfo.tsx`
- [ ] 6.2 Display number of seasons and episodes
- [ ] 6.3 Show first air date and last air date
- [ ] 6.4 Display status (Ended, Returning, etc.)
- [ ] 6.5 Show networks/streaming platforms

### Task 7: Implement Mobile View (AC: #5)
- [ ] 7.1 Create full-screen modal variant for mobile
- [ ] 7.2 Adjust layout for narrow screens
- [ ] 7.3 Use single-column layout
- [ ] 7.4 Ensure touch-friendly interactions

### Task 8: Add TypeScript Types (AC: #1, #2)
- [ ] 8.1 Add `MovieDetails` type to `types/tmdb.ts`
- [ ] 8.2 Add `TVShowDetails` type to `types/tmdb.ts`
- [ ] 8.3 Add `Credits` type with cast and crew
- [ ] 8.4 Add `CastMember` and `CrewMember` types

### Task 9: Write Tests (AC: #1, #2, #3, #4, #5)
- [ ] 9.1 Write unit tests for MediaDetailPanel
- [ ] 9.2 Write unit tests for SidePanel animations
- [ ] 9.3 Write unit tests for CreditsSection
- [ ] 9.4 Write hook tests with mock API responses
- [ ] 9.5 Test responsive behavior (desktop vs mobile)

## Dev Notes

### CRITICAL: Dependencies

This story **DEPENDS ON:**
- Story 2.1: TMDb API endpoints for details (`/api/v1/tmdb/movies/{id}`, `/api/v1/tmdb/tv/{id}`)
- Story 2.3: PosterCard click navigation to this page

### UX Design Pattern: Side Panel Detail

From UX specification, vido uses **Sidebar Detail Panel** (Spotify-style):

```
┌────────────────────────────────────────────────────────────┐
│  Main Content Area                    │  Side Panel        │
│  (Media Grid stays visible)           │  (400-500px)       │
│                                       │                    │
│  [Grid of posters...]                 │  [Poster]          │
│                                       │  Title             │
│                                       │  ★ 8.5 | 2h 19m    │
│                                       │  Genres            │
│                                       │  Overview...       │
│                                       │  Cast...           │
└────────────────────────────────────────────────────────────┘
```

**Benefits:**
- Grid stays visible, no context loss
- Smooth browsing experience
- Leverage large desktop screens

### File Locations

| Component | Path |
|-----------|------|
| Media Detail Route | `apps/web/src/routes/media/$type.$id.tsx` |
| SidePanel | `apps/web/src/components/ui/SidePanel.tsx` |
| MediaDetailPanel | `apps/web/src/components/media/MediaDetailPanel.tsx` |
| CreditsSection | `apps/web/src/components/media/CreditsSection.tsx` |
| TVShowInfo | `apps/web/src/components/media/TVShowInfo.tsx` |
| TMDb Detail Hooks | `apps/web/src/hooks/useMediaDetails.ts` |
| Types | `apps/web/src/types/tmdb.ts` |

### TMDb API Endpoints Needed

**Backend must implement (Story 2.1):**

```
GET /api/v1/tmdb/movies/{id}        → MovieDetails
GET /api/v1/tmdb/movies/{id}/credits → Credits
GET /api/v1/tmdb/tv/{id}            → TVShowDetails
GET /api/v1/tmdb/tv/{id}/credits    → Credits
```

### TypeScript Types for Details

```typescript
// types/tmdb.ts

export interface MovieDetails {
  id: number;
  title: string;
  original_title: string;
  overview: string;
  release_date: string;
  poster_path: string | null;
  backdrop_path: string | null;
  vote_average: number;
  vote_count: number;
  runtime: number;              // Minutes
  budget: number;
  revenue: number;
  status: string;               // "Released", "Post Production", etc.
  tagline: string;
  genres: Genre[];
  production_countries: Country[];
  spoken_languages: Language[];
  imdb_id: string;
  homepage: string | null;
}

export interface TVShowDetails {
  id: number;
  name: string;
  original_name: string;
  overview: string;
  first_air_date: string;
  last_air_date: string;
  poster_path: string | null;
  backdrop_path: string | null;
  vote_average: number;
  vote_count: number;
  episode_run_time: number[];
  number_of_seasons: number;
  number_of_episodes: number;
  status: string;               // "Ended", "Returning Series", etc.
  type: string;                 // "Scripted", "Documentary", etc.
  tagline: string;
  genres: Genre[];
  created_by: Creator[];
  networks: Network[];
  in_production: boolean;
  seasons: Season[];
}

export interface Credits {
  id: number;
  cast: CastMember[];
  crew: CrewMember[];
}

export interface CastMember {
  id: number;
  name: string;
  character: string;
  profile_path: string | null;
  order: number;
}

export interface CrewMember {
  id: number;
  name: string;
  job: string;
  department: string;
  profile_path: string | null;
}

export interface Genre {
  id: number;
  name: string;
}

export interface Network {
  id: number;
  name: string;
  logo_path: string | null;
}

export interface Creator {
  id: number;
  name: string;
  profile_path: string | null;
}

export interface Season {
  id: number;
  name: string;
  overview: string;
  poster_path: string | null;
  season_number: number;
  episode_count: number;
  air_date: string | null;
}
```

### TanStack Query Hooks

```typescript
// hooks/useMediaDetails.ts

import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';

export const detailKeys = {
  all: ['details'] as const,
  movie: (id: number) => [...detailKeys.all, 'movie', id] as const,
  movieCredits: (id: number) => [...detailKeys.movie(id), 'credits'] as const,
  tv: (id: number) => [...detailKeys.all, 'tv', id] as const,
  tvCredits: (id: number) => [...detailKeys.tv(id), 'credits'] as const,
};

export function useMovieDetails(id: number) {
  return useQuery({
    queryKey: detailKeys.movie(id),
    queryFn: () => tmdbService.getMovieDetails(id),
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
}

export function useMovieCredits(id: number) {
  return useQuery({
    queryKey: detailKeys.movieCredits(id),
    queryFn: () => tmdbService.getMovieCredits(id),
    staleTime: 10 * 60 * 1000,
  });
}

export function useTVShowDetails(id: number) {
  return useQuery({
    queryKey: detailKeys.tv(id),
    queryFn: () => tmdbService.getTVShowDetails(id),
    staleTime: 10 * 60 * 1000,
  });
}

export function useTVShowCredits(id: number) {
  return useQuery({
    queryKey: detailKeys.tvCredits(id),
    queryFn: () => tmdbService.getTVShowCredits(id),
    staleTime: 10 * 60 * 1000,
  });
}
```

### SidePanel Component

```tsx
// components/ui/SidePanel.tsx

import { useEffect, useCallback } from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SidePanelProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  title?: string;
}

export function SidePanel({ isOpen, onClose, children, title }: SidePanelProps) {
  // Close on Escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Panel */}
      <div
        className={cn(
          'fixed right-0 top-0 z-50 h-full w-full sm:w-[450px]',
          'bg-gray-900 shadow-2xl',
          'transform transition-transform duration-300 ease-out',
          isOpen ? 'translate-x-0' : 'translate-x-full'
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-800 p-4">
          {title && <h2 className="text-lg font-semibold text-white">{title}</h2>}
          <button
            onClick={onClose}
            className="rounded-lg p-2 text-gray-400 hover:bg-gray-800 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="h-[calc(100%-60px)] overflow-y-auto">
          {children}
        </div>
      </div>
    </>
  );
}
```

### MediaDetailPanel Component

```tsx
// components/media/MediaDetailPanel.tsx

import { getImageUrl } from '../../lib/image';
import { CreditsSection } from './CreditsSection';
import { TVShowInfo } from './TVShowInfo';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';

interface MediaDetailPanelProps {
  type: 'movie' | 'tv';
  details: MovieDetails | TVShowDetails;
  credits?: Credits;
  isLoading?: boolean;
}

export function MediaDetailPanel({
  type,
  details,
  credits,
  isLoading,
}: MediaDetailPanelProps) {
  if (isLoading) {
    return <MediaDetailSkeleton />;
  }

  const isMovie = type === 'movie';
  const movie = isMovie ? (details as MovieDetails) : null;
  const tvShow = !isMovie ? (details as TVShowDetails) : null;

  const title = isMovie ? movie!.title : tvShow!.name;
  const originalTitle = isMovie ? movie!.original_title : tvShow!.original_name;
  const year = isMovie
    ? movie!.release_date?.slice(0, 4)
    : tvShow!.first_air_date?.slice(0, 4);
  const runtime = isMovie
    ? movie!.runtime
    : tvShow!.episode_run_time?.[0];

  const posterUrl = getImageUrl(details.poster_path, 'w500');
  const backdropUrl = getImageUrl(details.backdrop_path, 'w780');

  // Find director from crew
  const director = credits?.crew?.find((c) => c.job === 'Director');

  return (
    <div className="flex flex-col">
      {/* Backdrop header */}
      {backdropUrl && (
        <div className="relative h-48 w-full">
          <img
            src={backdropUrl}
            alt=""
            className="h-full w-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-gray-900 to-transparent" />
        </div>
      )}

      <div className="p-4">
        {/* Poster and basic info */}
        <div className="flex gap-4">
          {posterUrl && (
            <img
              src={posterUrl}
              alt={title}
              className="h-48 w-32 rounded-lg object-cover shadow-lg"
            />
          )}
          <div className="flex-1">
            <h1 className="text-xl font-bold text-white">{title}</h1>
            {originalTitle !== title && (
              <p className="text-sm text-gray-400">{originalTitle}</p>
            )}

            <div className="mt-2 flex items-center gap-3 text-sm text-gray-300">
              {year && <span>{year}</span>}
              {runtime && <span>{runtime} 分鐘</span>}
              <span className="flex items-center gap-1 text-yellow-400">
                ⭐ {details.vote_average.toFixed(1)}
              </span>
            </div>

            {/* Genres */}
            <div className="mt-3 flex flex-wrap gap-2">
              {details.genres?.map((genre) => (
                <span
                  key={genre.id}
                  className="rounded-full bg-gray-800 px-3 py-1 text-xs text-gray-300"
                >
                  {genre.name}
                </span>
              ))}
            </div>
          </div>
        </div>

        {/* Overview */}
        <div className="mt-6">
          <h3 className="mb-2 text-sm font-semibold text-gray-400">劇情簡介</h3>
          <p className="text-sm leading-relaxed text-gray-300">
            {details.overview || '暫無簡介'}
          </p>
        </div>

        {/* TV Show specific info */}
        {tvShow && <TVShowInfo show={tvShow} />}

        {/* Credits */}
        {credits && (
          <CreditsSection
            director={director}
            cast={credits.cast?.slice(0, 6)}
            createdBy={tvShow?.created_by}
          />
        )}
      </div>
    </div>
  );
}

function MediaDetailSkeleton() {
  return (
    <div className="animate-pulse p-4">
      <div className="h-48 w-full rounded-lg bg-gray-700" />
      <div className="mt-4 flex gap-4">
        <div className="h-48 w-32 rounded-lg bg-gray-700" />
        <div className="flex-1 space-y-2">
          <div className="h-6 w-3/4 rounded bg-gray-700" />
          <div className="h-4 w-1/2 rounded bg-gray-700" />
          <div className="h-4 w-1/4 rounded bg-gray-700" />
        </div>
      </div>
    </div>
  );
}
```

### Route Implementation

```tsx
// routes/media/$type.$id.tsx

import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { SidePanel } from '../../components/ui/SidePanel';
import { MediaDetailPanel } from '../../components/media/MediaDetailPanel';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
} from '../../hooks/useMediaDetails';

export const Route = createFileRoute('/media/$type/$id')({
  parseParams: (params) => ({
    type: params.type as 'movie' | 'tv',
    id: parseInt(params.id, 10),
  }),
  component: MediaDetailRoute,
});

function MediaDetailRoute() {
  const { type, id } = Route.useParams();
  const navigate = useNavigate();

  const isMovie = type === 'movie';

  const movieDetails = useMovieDetails(id);
  const tvDetails = useTVShowDetails(id);
  const movieCredits = useMovieCredits(id);
  const tvCredits = useTVShowCredits(id);

  const details = isMovie ? movieDetails : tvDetails;
  const credits = isMovie ? movieCredits : tvCredits;

  const handleClose = () => {
    navigate({ to: '/search' });
  };

  return (
    <SidePanel isOpen={true} onClose={handleClose}>
      <MediaDetailPanel
        type={type}
        details={details.data!}
        credits={credits.data}
        isLoading={details.isLoading}
      />
    </SidePanel>
  );
}
```

### Status Labels (Traditional Chinese)

```typescript
// lib/status.ts

export const MOVIE_STATUS: Record<string, string> = {
  'Released': '已上映',
  'Post Production': '後期製作',
  'In Production': '製作中',
  'Planned': '計畫中',
  'Canceled': '已取消',
  'Rumored': '傳聞',
};

export const TV_STATUS: Record<string, string> = {
  'Returning Series': '回歸中',
  'Ended': '已完結',
  'Canceled': '已取消',
  'In Production': '製作中',
  'Planned': '計畫中',
};
```

### Responsive Breakpoints

- **Desktop (≥1024px):** Side panel (450px width)
- **Tablet (768-1023px):** Side panel or bottom sheet
- **Mobile (<768px):** Full-screen modal

### Animation Timing

From UX spec:
- Panel slide-in: `300ms` ease-out
- Backdrop fade: `200ms`
- Skeleton pulse: CSS `animate-pulse`

### Testing Strategy

1. **MediaDetailPanel Tests:**
   - Renders movie details correctly
   - Renders TV show details correctly
   - Shows skeleton when loading
   - Displays all required fields

2. **SidePanel Tests:**
   - Opens and closes correctly
   - Closes on Escape key
   - Closes on backdrop click
   - Animation classes applied

3. **Hook Tests:**
   - Fetches correct endpoint based on type
   - Uses proper query keys
   - Handles loading and error states

### Project Structure After This Story

```
apps/web/src/
├── routes/
│   ├── media/
│   │   └── $type.$id.tsx       # NEW: Detail route
│   ├── search.tsx
│   └── index.tsx
├── components/
│   ├── media/
│   │   ├── MediaDetailPanel.tsx     # NEW
│   │   ├── MediaDetailPanel.spec.tsx
│   │   ├── CreditsSection.tsx       # NEW
│   │   ├── TVShowInfo.tsx           # NEW
│   │   ├── PosterCard.tsx
│   │   └── MediaGrid.tsx
│   └── ui/
│       ├── SidePanel.tsx            # NEW
│       ├── SidePanel.spec.tsx
│       └── Pagination.tsx
├── hooks/
│   ├── useMediaDetails.ts           # NEW
│   ├── useMediaDetails.spec.ts
│   └── useSearchMedia.ts
├── services/
│   └── tmdb.ts                      # UPDATED: Add detail methods
├── types/
│   └── tmdb.ts                      # UPDATED: Add detail types
└── lib/
    ├── status.ts                    # NEW: Status translations
    └── image.ts
```

### References

- [Source: project-context.md#Rule 5: TanStack Query for Server State]
- [Source: ux-design-specification.md#Sidebar Detail Panel]
- [Source: architecture.md#Frontend Architecture]
- [Source: epics.md#Story 2.4: Media Detail Page]
- [Source: /internal/tmdb/types.go - TMDb type definitions]
- [Source: 2-1-tmdb-api-integration-with-zh-tw-priority.md - Backend dependency]
- [Source: 2-3-search-results-grid-view.md - PosterCard navigation]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

