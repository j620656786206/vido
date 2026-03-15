import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { MoreHorizontal } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl, getImageSrcSet, getImageSizes } from '../../lib/image';
import { HighlightText } from '../ui/HighlightText';
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
  metadataSource?: string;
  isNew?: boolean;
  highlightQuery?: string;
  onMenuClick?: (e: React.MouseEvent) => void;
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
  metadataSource,
  isNew,
  highlightQuery,
  onMenuClick,
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
          'transition-all duration-300 ease-out',
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

        {/* Badges (top-right) */}
        <div className="absolute right-2 top-2 flex items-center gap-1">
          {isNew && (
            <span
              data-testid="new-badge"
              className="rounded bg-emerald-500 px-1.5 py-0.5 text-[10px] font-bold text-white"
            >
              新增
            </span>
          )}
          {metadataSource && (
            <span className="rounded bg-blue-600/80 px-1.5 py-0.5 text-[10px] font-medium text-white opacity-0 transition-opacity lg:group-hover:opacity-100">
              {metadataSource}
            </span>
          )}
          <span className="rounded bg-black/70 px-2 py-0.5 text-xs font-medium text-white">
            {type === 'movie' ? '電影' : '影集'}
          </span>
        </div>

        {/* More menu button (visible on hover) */}
        {onMenuClick && (
          <button
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onMenuClick(e);
            }}
            className="absolute left-2 top-2 rounded-full bg-black/70 p-1.5 text-white opacity-0 transition-opacity hover:bg-black/90 lg:group-hover:opacity-100"
            aria-label="更多選項"
            data-testid="poster-menu-button"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
        )}

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
        <h3 className="truncate text-sm font-medium text-white">
          <HighlightText text={title} query={highlightQuery} />
        </h3>
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
