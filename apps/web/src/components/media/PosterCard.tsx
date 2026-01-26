import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { cn } from '../../lib/utils';
import { getImageUrl, getImageSrcSet, getImageSizes } from '../../lib/image';
import { HoverPreviewCard } from './HoverPreviewCard';

export interface PosterCardProps {
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
  const posterSrcSet = getImageSrcSet(posterPath);
  const posterSizes = getImageSizes();

  const showFallback = !posterUrl || imageError;
  const showSkeleton = !imageLoaded && !imageError && posterUrl;

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id: String(id) }}
      data-testid="poster-card"
      className={cn(
        'group relative block rounded-lg',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
        // Minimum touch target size (44px) ensured by aspect-ratio and grid min-width
        'min-h-[44px]'
      )}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div
        className={cn(
          'relative aspect-[2/3] overflow-hidden rounded-lg bg-gray-800',
          'transition-all duration-150 ease-out',
          // Hover effects only on desktop (lg breakpoint)
          'lg:group-hover:scale-105 lg:group-hover:shadow-2xl',
          // Active state for touch feedback on mobile
          'active:scale-[0.98] active:opacity-90'
        )}
      >
        {/* Loading skeleton */}
        {showSkeleton && (
          <div
            data-testid="poster-skeleton"
            className="absolute inset-0 animate-pulse bg-gray-700"
          />
        )}

        {/* Poster image */}
        {posterUrl && !imageError && (
          <img
            src={posterUrl}
            srcSet={posterSrcSet || undefined}
            sizes={posterSizes}
            alt={title}
            loading="lazy"
            onLoad={() => setImageLoaded(true)}
            onError={() => setImageError(true)}
            className={cn('h-full w-full object-cover', imageLoaded ? 'opacity-100' : 'opacity-0')}
          />
        )}

        {/* Fallback placeholder */}
        {showFallback && (
          <div
            data-testid="poster-fallback"
            className="flex h-full w-full items-center justify-center bg-gray-700"
          >
            <span role="img" aria-label="無海報圖片" className="text-4xl text-gray-500">
              🎬
            </span>
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
