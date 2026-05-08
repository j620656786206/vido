// Implements: Component/PosterCardHover (MQbvp)
// Source: ux-design.pen (Pencil app)

import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { MoreHorizontal, Check, Play } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl, getImageSrcSet, getImageSizes } from '../../lib/image';
import { HighlightText } from '../ui/HighlightText';
import { AvailabilityBadge } from './AvailabilityBadge';

export interface PosterCardProps {
  id: string;
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
  /** Story 10-4 — the user already owns this title. */
  isOwned?: boolean;
  /** Story 10-4 — the user has an open request for this title. Stubbed to false until Phase 3. */
  isRequested?: boolean;
  highlightQuery?: string;
  onMenuClick?: (e: React.MouseEvent) => void;
  selectable?: boolean;
  selected?: boolean;
  onSelect?: (e: React.MouseEvent) => void;
}

export function PosterCard({
  id,
  type,
  title,
  posterPath,
  releaseDate,
  voteAverage,
  metadataSource,
  isNew,
  isOwned,
  isRequested,
  highlightQuery,
  onMenuClick,
  selectable,
  selected,
  onSelect,
}: PosterCardProps) {
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);

  const year = releaseDate ? new Date(releaseDate).getFullYear() : null;
  const posterUrl = getImageUrl(posterPath, 'w342');
  const posterSrcSet = getImageSrcSet(posterPath);
  const posterSizes = getImageSizes();

  const showFallback = !posterUrl || imageError;
  const showSkeleton = !imageLoaded && !imageError && posterUrl;

  const handleCardClick = (e: React.MouseEvent) => {
    if (selectable && onSelect) {
      e.preventDefault();
      e.stopPropagation();
      onSelect(e);
    }
  };

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id }}
      data-testid="poster-card"
      onClick={handleCardClick}
      className={cn(
        'group relative block rounded-lg',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-primary)]',
        // Minimum touch target size (44px) ensured by aspect-ratio and grid min-width
        'min-h-[44px]',
        selectable && 'cursor-pointer'
      )}
    >
      <div
        // bugfix-10-4 DIAGNOSTIC: inline style for clip-path to bypass any Tailwind
        // arbitrary-value compilation issues. clipPath replaces overflow-hidden +
        // rounded-lg to dodge the Chromium scale + overflow-hidden + border-radius bug.
        style={{ clipPath: 'inset(0 round 0.5rem)' }}
        className={cn(
          'relative aspect-[2/3] bg-[var(--bg-secondary)]',
          'transition-all duration-300 ease-out',
          'transform-gpu',
          // Hover effects only on desktop (lg breakpoint) — disabled in selection mode
          !selectable && 'lg:group-hover:scale-105 lg:group-hover:shadow-2xl',
          // Active state for touch feedback on mobile
          'active:scale-[0.98] active:opacity-90',
          // Selection mode styling
          selectable && selected && 'ring-2 ring-[var(--accent-primary)]',
          selectable && !selected && 'opacity-70'
        )}
      >
        {/* Loading skeleton */}
        {showSkeleton && (
          <div
            data-testid="poster-skeleton"
            className="absolute inset-0 animate-pulse bg-[var(--bg-tertiary)]"
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
            className="flex h-full w-full items-center justify-center bg-[var(--bg-tertiary)]"
          >
            <span role="img" aria-label="無海報圖片" className="text-4xl text-[var(--text-muted)]">
              🎬
            </span>
          </div>
        )}

        {/* Selection checkbox overlay (top-left) — MQbvp: top-left circle slot, mode-gated */}
        {selectable && (
          <div
            data-testid="selection-checkbox"
            className={cn(
              'absolute left-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded border-2 transition-colors',
              selected
                ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)] text-white'
                : 'border-white/60 bg-black/40'
            )}
          >
            {selected && <Check className="h-4 w-4" />}
          </div>
        )}

        {/* Top-right badge cluster — visible by default, fade out on hover so kebab takes over (MQbvp collision strategy per AC #1, AC #10).
            duration-300 syncs with image-wrapper scale-105 transition for unified feel. */}
        <div className="absolute right-2 top-2 flex items-center gap-1 transition-opacity duration-300 lg:group-hover:opacity-0">
          {/* Story 10-4 — availability badges win position over 新增 so owners
              see ownership first. Only one of owned/requested renders. */}
          {isOwned ? (
            <AvailabilityBadge variant="owned" />
          ) : isRequested ? (
            <AvailabilityBadge variant="requested" />
          ) : null}
          {isNew && (
            <span
              data-testid="new-badge"
              className="rounded bg-emerald-500 px-1.5 py-0.5 text-[10px] font-bold text-white"
            >
              新增
            </span>
          )}
          {metadataSource && (
            <span className="rounded bg-[var(--accent-primary)]/80 px-1.5 py-0.5 text-[10px] font-medium text-white opacity-0 transition-opacity lg:group-hover:opacity-100">
              {metadataSource}
            </span>
          )}
          <span className="rounded bg-black/70 px-2 py-0.5 text-xs font-medium text-white">
            {type === 'movie' ? '電影' : '影集'}
          </span>
        </div>

        {/* Kebab menu — MQbvp: top-RIGHT slot (was top-LEFT in pre-bugfix-10-4), hover-only via group-hover */}
        {onMenuClick && (
          <button
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onMenuClick(e);
            }}
            className="absolute right-2 top-2 z-20 rounded-full bg-black/70 p-1.5 text-white opacity-0 transition-opacity duration-300 hover:bg-black/90 lg:group-hover:opacity-100"
            aria-label="更多選項"
            data-testid="poster-menu-button"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
        )}

        {/* Center play overlay — MQbvp: large circular ▶ play affordance, hover-only, decorative (no onClick — propagates to <Link>) */}
        {!selectable && (
          <div
            data-testid="hover-play-overlay"
            aria-hidden="true"
            className="absolute inset-0 z-10 hidden items-center justify-center opacity-0 transition-opacity duration-300 lg:flex lg:group-hover:opacity-100"
          >
            <div className="rounded-full bg-black/60 p-4 backdrop-blur-sm">
              <Play className="h-8 w-8 fill-white text-white" />
            </div>
          </div>
        )}

        {/* Note: MQbvp design originally specified a bottom-left title/year overlay,
            but Party Mode 2026-05-08 (Sally + Alexyu) determined this duplicates the
            below-image title (RusTY) and has legibility issues against varying poster
            backgrounds. Hover state is now action-trigger only (play + kebab + rating);
            in-card info-density redesign deferred to feature-X-postercard-info-density. */}

        {/* Rating badge — MQbvp: bottom-RIGHT slot (was bottom-LEFT in pre-bugfix-10-4), always visible when voteAverage > 0 */}
        {voteAverage !== undefined && voteAverage > 0 && (
          <div className="absolute bottom-2 right-2 z-20">
            <span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-[var(--warning)]">
              ⭐ {voteAverage.toFixed(1)}
            </span>
          </div>
        )}
      </div>

      {/* Title and year — default-state below-image affordance (kept for non-hover continuity per AC #1) */}
      <div className="mt-2">
        <h3 className="truncate text-sm font-medium text-white">
          <HighlightText text={title} query={highlightQuery} />
        </h3>
        {year && <p className="text-xs text-[var(--text-secondary)]">{year}</p>}
      </div>
    </Link>
  );
}
