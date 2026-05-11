// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)
// Design ref: ux-design.pen Screen PC-1 (XlFIq) — bugfix-10-7 info-density & polish
// Source: ux-design.pen (Pencil app)

import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { MoreHorizontal, Check, Play, Star } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl, getImageSrcSet, getImageSizes } from '../../lib/image';
import { useMovieDetails, useTVShowDetails } from '../../hooks/useMediaDetails';
import { formatPosterMeta, formatRuntime, formatSeriesCount } from '../../lib/formatMedia';
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
  // bugfix-10-7 AC1 — hover-intent debounce: only fetch the TMDb detail (runtime / season
  // count) after the pointer dwells on the card for ~200 ms, so a mouse sweeping across a
  // grid doesn't fire a burst of detail requests. The VISUAL hover effects (image scale-105,
  // play overlay, kebab, badge-cluster fade) stay instant — they are CSS :hover, not gated.
  const [hoverIntent, setHoverIntent] = useState(false);
  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Numeric id ⇒ TMDb item (gets the runtime/episode-count line on hover); UUID ⇒ owned-library
  // item ⇒ 0 ⇒ useMovieDetails/useTVShowDetails stay disabled via their built-in `enabled: id > 0`.
  const tmdbId = /^\d+$/.test(id) ? Number(id) : 0;
  const fetchId = hoverIntent ? tmdbId : 0;
  const movieQuery = useMovieDetails(type === 'movie' ? fetchId : 0);
  const tvQuery = useTVShowDetails(type === 'tv' ? fetchId : 0);

  useEffect(() => {
    return () => {
      if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current);
    };
  }, []);

  const year = releaseDate ? new Date(releaseDate).getFullYear() : null;
  const extraMeta =
    type === 'movie'
      ? formatRuntime(movieQuery.data?.runtime)
      : formatSeriesCount(tvQuery.data?.numberOfSeasons, tvQuery.data?.numberOfEpisodes);
  const metaLine = formatPosterMeta(year, extraMeta);
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

  // Start the hover-intent timer on enter; cancel it on leave (but never reset hoverIntent
  // once true — the data is already loaded, keep showing it and avoid a re-fetch flicker).
  const handleMouseEnter = () => {
    if (hoverTimerRef.current) return;
    hoverTimerRef.current = setTimeout(() => {
      hoverTimerRef.current = null;
      setHoverIntent(true);
    }, 200);
  };

  const handleMouseLeave = () => {
    if (hoverTimerRef.current) {
      clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
  };

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id }}
      data-testid="poster-card"
      onClick={handleCardClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      className={cn(
        'group relative block rounded-lg',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-primary)]',
        // Minimum touch target size (44px) ensured by aspect-ratio and grid min-width
        'min-h-[44px]',
        selectable && 'cursor-pointer'
      )}
    >
      <div
        // WORKAROUND (bugfix-10-4): Chromium drops border-radius clip when transform:scale
        // and overflow-hidden combine on a GPU layer. Use clip-path so corners stay rounded
        // throughout the hover scale-105 transition.
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

        {/* Top-right badge cluster — visible by default; on hover it RECEDES (opacity + scale-95,
            anchored at its top-right corner) so the kebab takes over (MQbvp collision strategy per
            bugfix-10-4 AC #1 / bugfix-10-7 AC #2). transition-all + duration-300 stays in sync with
            the image-wrapper's lg:group-hover:scale-105 transition for a unified kinetic feel. */}
        <div className="absolute right-2 top-2 flex origin-top-right items-center gap-1 transition-all duration-300 lg:group-hover:scale-95 lg:group-hover:opacity-0">
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
            <span className="rounded bg-[var(--accent-primary)]/80 px-1.5 py-0.5 text-[10px] font-medium text-white">
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
            the in-card info-density was instead delivered below the image (year +
            runtime/episode-count, lazy-fetched on hover) by bugfix-10-7 — see the mt-2 block. */}

        {/* Rating badge — MQbvp: bottom-RIGHT slot (was bottom-LEFT in pre-bugfix-10-4), always visible when voteAverage > 0.
            bugfix-10-7 AC #3: lucide <Star> SVG (not the ⭐ emoji) for cross-OS rendering consistency. */}
        {voteAverage !== undefined && voteAverage > 0 && (
          <div className="absolute bottom-2 right-2 z-20">
            <span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-[var(--warning)]">
              <Star
                className="h-3 w-3 fill-[var(--warning)] text-[var(--warning)]"
                aria-hidden="true"
              />
              {voteAverage.toFixed(1)}
            </span>
          </div>
        )}
      </div>

      {/* Title + metadata line — below-image affordance. bugfix-10-7 AC #1: the metadata line
          is `{year} · {extra}` where `extra` is the runtime (movies) or `{seasons} 季 {episodes} 集`
          (series), lazy-fetched on hover. Stays year-only until the fetch resolves (and for
          owned-library UUID cards / touch devices, which never fetch). `truncate` keeps it on
          one line; only the title gets <HighlightText>. */}
      <div className="mt-2">
        <h3 className="truncate text-sm font-medium text-white">
          <HighlightText text={title} query={highlightQuery} />
        </h3>
        {metaLine && (
          <p className="truncate text-xs text-[var(--text-secondary)] transition-opacity duration-200">
            {metaLine}
          </p>
        )}
      </div>
    </Link>
  );
}
