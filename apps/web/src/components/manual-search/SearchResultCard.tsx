/**
 * SearchResultCard Component (Story 3.7 - AC2)
 * Individual search result card with poster, title, year, and source badge
 */

import { useState } from 'react';
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ManualSearchResultItem } from '../../services/metadata';

export interface SearchResultCardProps {
  item: ManualSearchResultItem;
  isSelected: boolean;
  onSelect: () => void;
}

// Source badge colors
const SOURCE_COLORS: Record<string, string> = {
  tmdb: 'bg-green-600',
  douban: 'bg-yellow-600',
  wikipedia: 'bg-blue-600',
};

export function SearchResultCard({
  item,
  isSelected,
  onSelect,
}: SearchResultCardProps) {
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [showOverview, setShowOverview] = useState(false);

  const showFallback = !item.posterUrl || imageError;

  return (
    <button
      onClick={onSelect}
      onMouseEnter={() => setShowOverview(true)}
      onMouseLeave={() => setShowOverview(false)}
      className={cn(
        'group relative text-left w-full rounded-lg overflow-hidden',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
        'transition-all duration-200',
        isSelected && 'ring-2 ring-blue-500'
      )}
      data-testid="search-result-card"
    >
      {/* Poster Container */}
      <div
        className={cn(
          'relative aspect-[2/3] overflow-hidden rounded-lg bg-slate-800',
          'transition-transform duration-150 ease-out',
          'group-hover:scale-[1.02]'
        )}
      >
        {/* Loading skeleton */}
        {!imageLoaded && !imageError && item.posterUrl && (
          <div className="absolute inset-0 animate-pulse bg-slate-700" />
        )}

        {/* Poster image */}
        {item.posterUrl && !imageError && (
          <img
            src={item.posterUrl}
            alt={item.title}
            loading="lazy"
            onLoad={() => setImageLoaded(true)}
            onError={() => setImageError(true)}
            className={cn(
              'h-full w-full object-cover',
              imageLoaded ? 'opacity-100' : 'opacity-0'
            )}
          />
        )}

        {/* Fallback placeholder */}
        {showFallback && (
          <div className="flex h-full w-full items-center justify-center bg-slate-700">
            <span className="text-4xl text-slate-500">🎬</span>
          </div>
        )}

        {/* Source badge (AC4) */}
        <div className="absolute right-2 top-2">
          <span
            className={cn(
              'rounded px-2 py-0.5 text-xs font-medium text-white',
              SOURCE_COLORS[item.source] || 'bg-slate-600'
            )}
          >
            {item.source.toUpperCase()}
          </span>
        </div>

        {/* Rating badge */}
        {item.rating !== undefined && item.rating > 0 && (
          <div className="absolute bottom-2 left-2">
            <span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-yellow-400">
              ⭐ {item.rating.toFixed(1)}
            </span>
          </div>
        )}

        {/* Selected indicator */}
        {isSelected && (
          <div className="absolute inset-0 bg-blue-500/20 flex items-center justify-center">
            <div className="rounded-full bg-blue-500 p-2">
              <Check className="h-6 w-6 text-white" />
            </div>
          </div>
        )}

        {/* Hover overlay with overview */}
        {showOverview && item.overview && (
          <div className="absolute inset-0 bg-black/80 p-3 overflow-y-auto">
            <p className="text-xs text-slate-300 line-clamp-[10]">
              {item.overview}
            </p>
          </div>
        )}
      </div>

      {/* Title and year */}
      <div className="mt-2 px-1">
        <h3 className="truncate text-sm font-medium text-white">
          {item.titleZhTW || item.title}
        </h3>
        {item.titleZhTW && item.titleZhTW !== item.title && (
          <p className="truncate text-xs text-slate-400">{item.title}</p>
        )}
        <div className="flex items-center gap-2 mt-1">
          {item.year && (
            <span className="text-xs text-slate-500">{item.year}</span>
          )}
          <span className="text-xs text-slate-600">
            {item.mediaType === 'movie' ? '電影' : '影集'}
          </span>
        </div>
      </div>
    </button>
  );
}

export default SearchResultCard;
