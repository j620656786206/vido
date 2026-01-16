# Story 2.3: Search Results Grid View

Status: ready-for-dev

## Story

As a **media collector**,
I want to **browse search results in a responsive grid view**,
So that **I can quickly scan through multiple results visually**.

## Acceptance Criteria

1. **Given** search results are displayed
   **When** viewed on desktop (>1024px)
   **Then** results display in a 4-6 column grid
   **And** each card shows poster, title, year, rating

2. **Given** search results are displayed
   **When** viewed on tablet (768-1023px)
   **Then** results display in a 3-4 column grid

3. **Given** search results are displayed
   **When** viewed on mobile (<768px)
   **Then** results display in a 2 column grid
   **And** touch targets are at least 44px

4. **Given** the user hovers over a result card (desktop)
   **When** the mouse is over the card
   **Then** additional information appears (genre, description preview)
   **And** the card has a subtle highlight effect

5. **Given** a poster image is loading
   **When** the image has not yet loaded
   **Then** a placeholder skeleton is displayed
   **And** the image lazy loads when entering viewport

## Tasks / Subtasks

### Task 1: Create PosterCard Component (AC: #1, #4, #5)
- [ ] 1.1 Create `apps/web/src/components/media/PosterCard.tsx`
- [ ] 1.2 Implement poster image with lazy loading (`loading="lazy"`)
- [ ] 1.3 Display title (zh-TW), year, and TMDb rating
- [ ] 1.4 Add media type badge (Movie/TV)
- [ ] 1.5 Implement loading skeleton placeholder
- [ ] 1.6 Handle missing poster image with fallback placeholder

### Task 2: Implement Hover Preview (AC: #4)
- [ ] 2.1 Create `apps/web/src/components/media/HoverPreviewCard.tsx`
- [ ] 2.2 Show on hover: genres, overview (truncated), original title
- [ ] 2.3 Add scale transform animation on hover (`scale-105`)
- [ ] 2.4 Add shadow elevation on hover (`shadow-xl → shadow-2xl`)
- [ ] 2.5 Ensure smooth transitions (150ms duration)

### Task 3: Create MediaGrid Component (AC: #1, #2, #3)
- [ ] 3.1 Create `apps/web/src/components/media/MediaGrid.tsx`
- [ ] 3.2 Implement responsive grid with CSS Grid
- [ ] 3.3 Desktop: `repeat(auto-fill, minmax(200px, 1fr))` → 5-6 columns
- [ ] 3.4 Tablet: `repeat(auto-fill, minmax(160px, 1fr))` → 3-4 columns
- [ ] 3.5 Mobile: `repeat(2, 1fr)` → 2 columns fixed
- [ ] 3.6 Set grid gap to 16px (desktop/tablet) and 12px (mobile)

### Task 4: Implement Image Optimization (AC: #5)
- [ ] 4.1 Create `apps/web/src/lib/image.ts` utility for TMDb image URLs
- [ ] 4.2 Implement responsive image sizes (w185 for grid, w342 for detail)
- [ ] 4.3 Add `srcset` for different DPI displays
- [ ] 4.4 Create placeholder component for loading state
- [ ] 4.5 Handle image load errors gracefully

### Task 5: Mobile Touch Optimization (AC: #3)
- [ ] 5.1 Ensure touch targets are minimum 44px × 44px
- [ ] 5.2 Add tap feedback (active state styling)
- [ ] 5.3 Disable hover effects on touch devices
- [ ] 5.4 Ensure scrolling is smooth on mobile

### Task 6: Integrate with Search Page (AC: #1, #2, #3)
- [ ] 6.1 Replace list view in `SearchResults.tsx` with `MediaGrid`
- [ ] 6.2 Pass search results to grid component
- [ ] 6.3 Implement grid skeleton for loading state
- [ ] 6.4 Handle empty state within grid context

### Task 7: Add Navigation to Detail Page (AC: #4)
- [ ] 7.1 Add click handler to PosterCard
- [ ] 7.2 Navigate to `/media/{type}/{id}` on click (Story 2.4 will implement the route)
- [ ] 7.3 Add keyboard navigation (Enter to select)
- [ ] 7.4 Consider opening in new tab for desktop (per UX spec)

### Task 8: Write Tests (AC: #1, #2, #3, #4, #5)
- [ ] 8.1 Write unit tests for PosterCard component
- [ ] 8.2 Write unit tests for MediaGrid responsive behavior
- [ ] 8.3 Write unit tests for HoverPreviewCard
- [ ] 8.4 Test image loading and error states
- [ ] 8.5 Test touch vs mouse interaction modes

## Dev Notes

### CRITICAL: Dependency on Story 2.2

This story **DEPENDS ON** Story 2.2 (Media Search Interface). The search page and search hooks must exist. This story enhances the search results display from list to grid.

### UX Design Requirements

From UX design specification:

**Poster Grid Layout (Core Pattern):**
```css
/* Desktop (1024px+): 5-6 columns */
grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
gap: 16px;

/* Tablet (768-1023px): 3-4 columns */
grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
gap: 12px;

/* Mobile (<768px): 2 columns fixed */
grid-template-columns: repeat(2, 1fr);
gap: 12px;
```

**Hover Interactions (Desktop-first UX-1, UX-8):**
- Hover reveals additional info (genres, description preview)
- Scale transform: `hover:scale-105`
- Shadow elevation: `hover:shadow-2xl`
- Transition duration: 150ms

### File Locations

| Component | Path |
|-----------|------|
| PosterCard | `apps/web/src/components/media/PosterCard.tsx` |
| HoverPreviewCard | `apps/web/src/components/media/HoverPreviewCard.tsx` |
| MediaGrid | `apps/web/src/components/media/MediaGrid.tsx` |
| Image Utils | `apps/web/src/lib/image.ts` |
| Tests | Co-located (`*.spec.tsx`) |

### PosterCard Component Implementation

```tsx
// components/media/PosterCard.tsx

import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { cn } from '../../lib/utils';
import { getImageUrl } from '../../lib/image';
import { HoverPreviewCard } from './HoverPreviewCard';

interface PosterCardProps {
  id: number;
  type: 'movie' | 'tv';
  title: string;
  originalTitle?: string;
  posterPath: string | null;
  releaseDate?: string;
  voteAverage?: number;
  overview?: string;
  genreIds?: number[];
}

export function PosterCard({
  id,
  type,
  title,
  originalTitle,
  posterPath,
  releaseDate,
  voteAverage,
  overview,
  genreIds,
}: PosterCardProps) {
  const [isHovered, setIsHovered] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);

  const year = releaseDate ? new Date(releaseDate).getFullYear() : null;
  const posterUrl = getImageUrl(posterPath, 'w342');

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id: String(id) }}
      className="group relative block"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div
        className={cn(
          'relative aspect-[2/3] overflow-hidden rounded-lg bg-gray-800',
          'transition-all duration-150 ease-out',
          'group-hover:scale-105 group-hover:shadow-2xl'
        )}
      >
        {/* Loading skeleton */}
        {!imageLoaded && !imageError && (
          <div className="absolute inset-0 animate-pulse bg-gray-700" />
        )}

        {/* Poster image */}
        {posterUrl && !imageError ? (
          <img
            src={posterUrl}
            alt={title}
            loading="lazy"
            onLoad={() => setImageLoaded(true)}
            onError={() => setImageError(true)}
            className={cn(
              'h-full w-full object-cover',
              imageLoaded ? 'opacity-100' : 'opacity-0'
            )}
          />
        ) : (
          /* Fallback placeholder */
          <div className="flex h-full w-full items-center justify-center bg-gray-700">
            <span className="text-4xl text-gray-500">🎬</span>
          </div>
        )}

        {/* Media type badge */}
        <div className="absolute right-2 top-2">
          <span className="rounded bg-black/70 px-2 py-0.5 text-xs font-medium text-white">
            {type === 'movie' ? '電影' : '影集'}
          </span>
        </div>

        {/* Rating badge */}
        {voteAverage !== undefined && voteAverage > 0 && (
          <div className="absolute bottom-2 left-2">
            <span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-yellow-400">
              ⭐ {voteAverage.toFixed(1)}
            </span>
          </div>
        )}
      </div>

      {/* Title and year */}
      <div className="mt-2">
        <h3 className="truncate text-sm font-medium text-white">{title}</h3>
        {year && <p className="text-xs text-gray-400">{year}</p>}
      </div>

      {/* Hover preview (desktop only) */}
      {isHovered && (
        <HoverPreviewCard
          title={title}
          originalTitle={originalTitle}
          overview={overview}
          genreIds={genreIds}
        />
      )}
    </Link>
  );
}
```

### MediaGrid Component Implementation

```tsx
// components/media/MediaGrid.tsx

import { PosterCard } from './PosterCard';
import { PosterCardSkeleton } from './PosterCardSkeleton';
import type { Movie, TVShow } from '../../types/tmdb';

interface MediaGridProps {
  movies?: Movie[];
  tvShows?: TVShow[];
  isLoading?: boolean;
  emptyMessage?: string;
}

export function MediaGrid({
  movies = [],
  tvShows = [],
  isLoading,
  emptyMessage = '沒有找到結果',
}: MediaGridProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]">
        {Array.from({ length: 12 }).map((_, i) => (
          <PosterCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  const hasResults = movies.length > 0 || tvShows.length > 0;

  if (!hasResults) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-gray-400">
        <span className="mb-2 text-4xl">🔍</span>
        <p>{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]">
      {movies.map((movie) => (
        <PosterCard
          key={`movie-${movie.id}`}
          id={movie.id}
          type="movie"
          title={movie.title}
          originalTitle={movie.original_title}
          posterPath={movie.poster_path}
          releaseDate={movie.release_date}
          voteAverage={movie.vote_average}
          overview={movie.overview}
          genreIds={movie.genre_ids}
        />
      ))}
      {tvShows.map((show) => (
        <PosterCard
          key={`tv-${show.id}`}
          id={show.id}
          type="tv"
          title={show.name}
          originalTitle={show.original_name}
          posterPath={show.poster_path}
          releaseDate={show.first_air_date}
          voteAverage={show.vote_average}
          overview={show.overview}
          genreIds={show.genre_ids}
        />
      ))}
    </div>
  );
}
```

### HoverPreviewCard Component

```tsx
// components/media/HoverPreviewCard.tsx

import { GENRE_MAP } from '../../lib/genres';

interface HoverPreviewCardProps {
  title: string;
  originalTitle?: string;
  overview?: string;
  genreIds?: number[];
}

export function HoverPreviewCard({
  title,
  originalTitle,
  overview,
  genreIds = [],
}: HoverPreviewCardProps) {
  const genres = genreIds
    .slice(0, 3)
    .map((id) => GENRE_MAP[id])
    .filter(Boolean);

  return (
    <div className="absolute left-0 right-0 top-full z-10 mt-2 hidden rounded-lg bg-gray-800 p-3 shadow-xl lg:block">
      {/* Original title if different */}
      {originalTitle && originalTitle !== title && (
        <p className="mb-1 text-xs text-gray-400">{originalTitle}</p>
      )}

      {/* Genres */}
      {genres.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {genres.map((genre) => (
            <span
              key={genre}
              className="rounded bg-gray-700 px-2 py-0.5 text-xs text-gray-300"
            >
              {genre}
            </span>
          ))}
        </div>
      )}

      {/* Overview */}
      {overview && (
        <p className="line-clamp-3 text-xs text-gray-300">{overview}</p>
      )}
    </div>
  );
}
```

### TMDb Image Utility

```typescript
// lib/image.ts

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

export type ImageSize = 'w92' | 'w154' | 'w185' | 'w342' | 'w500' | 'w780' | 'original';

export function getImageUrl(path: string | null, size: ImageSize = 'w342'): string | null {
  if (!path) return null;
  return `${TMDB_IMAGE_BASE}/${size}${path}`;
}

// For responsive images with srcset
export function getImageSrcSet(path: string | null): string | null {
  if (!path) return null;
  return [
    `${TMDB_IMAGE_BASE}/w185${path} 185w`,
    `${TMDB_IMAGE_BASE}/w342${path} 342w`,
    `${TMDB_IMAGE_BASE}/w500${path} 500w`,
  ].join(', ');
}
```

### TMDb Genre Map (Traditional Chinese)

```typescript
// lib/genres.ts

export const GENRE_MAP: Record<number, string> = {
  // Movie genres
  28: '動作',
  12: '冒險',
  16: '動畫',
  35: '喜劇',
  80: '犯罪',
  99: '紀錄',
  18: '劇情',
  10751: '家庭',
  14: '奇幻',
  36: '歷史',
  27: '恐怖',
  10402: '音樂',
  9648: '懸疑',
  10749: '愛情',
  878: '科幻',
  10770: '電視電影',
  53: '驚悚',
  10752: '戰爭',
  37: '西部',
  // TV genres
  10759: '動作冒險',
  10762: '兒童',
  10763: '新聞',
  10764: '真人秀',
  10765: '科幻奇幻',
  10766: '肥皂劇',
  10767: '脫口秀',
  10768: '戰爭政治',
};
```

### Skeleton Component

```tsx
// components/media/PosterCardSkeleton.tsx

export function PosterCardSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="aspect-[2/3] rounded-lg bg-gray-700" />
      <div className="mt-2 space-y-1">
        <div className="h-4 w-3/4 rounded bg-gray-700" />
        <div className="h-3 w-1/4 rounded bg-gray-700" />
      </div>
    </div>
  );
}
```

### Tailwind Breakpoints Reference

From Tailwind default configuration:
- `sm`: 640px (not used, skip to tablet)
- `md`: 768px (tablet)
- `lg`: 1024px (desktop)
- `xl`: 1280px (wide desktop)
- `2xl`: 1536px (ultra-wide)

### Touch Device Detection

```typescript
// lib/device.ts

export function isTouchDevice(): boolean {
  if (typeof window === 'undefined') return false;
  return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
}
```

Use this to conditionally disable hover effects:

```tsx
const isTouch = isTouchDevice();

<div className={cn(
  'transition-transform',
  !isTouch && 'hover:scale-105'
)}>
```

### Responsive Grid CSS Classes

```css
/* Tailwind classes for the grid */
.media-grid {
  @apply grid grid-cols-2 gap-3;        /* Mobile: 2 columns, 12px gap */
  @apply sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] sm:gap-4;  /* Tablet */
  @apply lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))];           /* Desktop */
}
```

Or use the extended Tailwind config:

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      gridTemplateColumns: {
        'media-grid': 'repeat(auto-fill, minmax(200px, 1fr))',
        'media-grid-sm': 'repeat(auto-fill, minmax(160px, 1fr))',
      },
    },
  },
}
```

### Testing Strategy

1. **PosterCard Tests:**
   - Renders title, year, rating correctly
   - Shows skeleton while image loads
   - Shows fallback when image fails
   - Hover state triggers preview (desktop)

2. **MediaGrid Tests:**
   - Renders correct number of items
   - Shows loading skeletons
   - Shows empty state
   - Responsive column count (mock viewport)

3. **HoverPreviewCard Tests:**
   - Shows original title when different
   - Displays genres correctly
   - Truncates long overview

```typescript
// PosterCard.spec.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { PosterCard } from './PosterCard';

describe('PosterCard', () => {
  const defaultProps = {
    id: 123,
    type: 'movie' as const,
    title: '鬼滅之刃',
    posterPath: '/test.jpg',
    releaseDate: '2020-10-16',
    voteAverage: 8.5,
  };

  it('renders title and year', () => {
    render(<PosterCard {...defaultProps} />);
    expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
    expect(screen.getByText('2020')).toBeInTheDocument();
  });

  it('shows rating badge', () => {
    render(<PosterCard {...defaultProps} />);
    expect(screen.getByText('8.5')).toBeInTheDocument();
  });

  it('shows media type badge', () => {
    render(<PosterCard {...defaultProps} />);
    expect(screen.getByText('電影')).toBeInTheDocument();
  });
});
```

### Project Structure After This Story

```
apps/web/src/
├── components/
│   ├── media/                    # NEW: Media display components
│   │   ├── PosterCard.tsx
│   │   ├── PosterCard.spec.tsx
│   │   ├── PosterCardSkeleton.tsx
│   │   ├── HoverPreviewCard.tsx
│   │   ├── HoverPreviewCard.spec.tsx
│   │   ├── MediaGrid.tsx
│   │   └── MediaGrid.spec.tsx
│   ├── search/                   # From Story 2.2
│   │   ├── SearchBar.tsx
│   │   ├── SearchResults.tsx     # UPDATED: Uses MediaGrid
│   │   └── ...
│   └── ui/
│       └── Pagination.tsx
├── lib/
│   ├── utils.ts                  # cn() helper
│   ├── image.ts                  # NEW: TMDb image utilities
│   ├── genres.ts                 # NEW: Genre ID to name mapping
│   └── device.ts                 # NEW: Device detection
└── ...
```

### References

- [Source: project-context.md#Naming Conventions]
- [Source: ux-design-specification.md#Poster Grid Layout]
- [Source: ux-design-specification.md#Hover Interactions]
- [Source: ux-design-specification.md#Responsive Breakpoints]
- [Source: epics.md#Story 2.3: Search Results Grid View]
- [Source: 2-2-media-search-interface.md - Direct dependency]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

