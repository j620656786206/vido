// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)
// Design ref: ux-design.pen Screen PC-1 (XlFIq) — bugfix-10-7 info-density & polish
// Source: ux-design.pen (Pencil app)

import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { MoreHorizontal, Check, Star, Film } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl, getImageSrcSet, getImageSizes } from '../../lib/image';
import { useMovieDetails, useTVShowDetails } from '../../hooks/useMediaDetails';
import { formatPosterMeta, formatRuntime, formatSeriesCount } from '../../lib/formatMedia';
import { HighlightText } from '../ui/HighlightText';
import { AvailabilityBadge } from './AvailabilityBadge';
import { RequestButton } from '../requests/RequestButton';

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
  /**
   * Critique R1 P2/#8 — a row whose HEADING already names the type (熱門影集)
   * repeats it on every card as pure noise. Mixed grids (search) keep the
   * default; single-type rows pass false.
   */
  showTypeBadge?: boolean;
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
  showTypeBadge = true,
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

  // Story 13-1b — the hover 想要 affordance: TMDb-numeric cards only (owned-library
  // UUID cards can't be requested), not in selection mode, not already owned
  // (the 已有 badge covers owned; requested cards show the pill for honest feedback),
  // AND only on surfaces that wire the requested state (isRequested !== undefined —
  // CR M1: an unwired surface would show a button whose state never reflects the
  // created request, inviting duplicate clicks; legacy Search/Library stay unchanged).
  const showRequestOverlay = !selectable && tmdbId > 0 && !isOwned && isRequested !== undefined;

  const handleCardClick = (e: React.MouseEvent) => {
    if (selectable && onSelect) {
      e.preventDefault();
      e.stopPropagation();
      onSelect(e);
    }
  };

  // Start the hover-intent timer on enter; cancel it on leave (but never reset hoverIntent
  // once true — the data is already loaded, keep showing it and avoid a re-fetch flicker).
  // Once hoverIntent is true there's nothing left to arm, so subsequent re-hovers are a no-op
  // (don't spin up a pointless 200 ms timer on every mouseEnter — bugfix-10-7 CR L2).
  const handleMouseEnter = () => {
    if (hoverIntent || hoverTimerRef.current) return;
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
        'group relative block rounded-[var(--radius-lg)]',
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
        style={{ clipPath: 'inset(0 round 0.75rem)' }}
        className={cn(
          'relative aspect-[2/3] bg-[var(--bg-secondary)]',
          'transition-all duration-[var(--motion-state)] ease-out',
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
            <Film aria-label="無海報圖片" className="h-10 w-10 text-[var(--text-muted)]" />
          </div>
        )}

        {/* Selection checkbox overlay (top-left) — MQbvp: top-left circle slot, mode-gated */}
        {selectable && (
          <div
            data-testid="selection-checkbox"
            className={cn(
              'absolute left-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded border-2 transition-colors',
              // Selected: the ✓ sits on a SOLID accent fill, so it wears the
              // on-accent token. Unselected: the box floats on POSTER ART, and
              // its white rim / black wash are the imagery relationship — no
              // theme token owns those alphas, so they stay literal.
              selected
                ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)] text-[var(--text-on-accent)]'
                : 'border-[var(--text-on-scrim)]/60 bg-[var(--overlay-scrim)]/60'
            )}
          >
            {selected && <Check className="h-4 w-4" />}
          </div>
        )}

        {/* Top-right badge cluster — visible by default; on hover it RECEDES (opacity + scale-95,
            anchored at its top-right corner) so the kebab takes over (MQbvp collision strategy per
            bugfix-10-4 AC #1 / bugfix-10-7 AC #2). Every layer of this hover — wrapper scale, badge
            recede, kebab, gradient — runs on --motion-state, because it is ONE gesture and a card
            whose parts arrive at different times reads as four cards. Change one, change them all. */}
        <div className="absolute right-2 top-2 flex origin-top-right items-center gap-1 transition-all duration-[var(--motion-state)] lg:group-hover:scale-95 lg:group-hover:opacity-0">
          {/* Story 10-4 — availability badges win position over 新增 so owners
              see ownership first. Only one of owned/requested renders. */}
          {isOwned ? (
            <AvailabilityBadge variant="owned" />
          ) : isRequested ? (
            <AvailabilityBadge variant="requested" />
          ) : null}
          {isNew && (
            // 新增 is a classification (recently-added), not a live status —
            // neutral scrim, never green (critique R3, same ruling as 已有).
            <span
              data-testid="new-badge"
              className="rounded-full bg-[var(--overlay-scrim)] px-1.5 py-0.5 text-[11px] font-medium text-[var(--text-on-scrim)]"
            >
              新增
            </span>
          )}
          {metadataSource && (
            <span className="rounded-full bg-[var(--overlay-scrim)] px-1.5 py-0.5 text-[11px] font-medium text-[var(--text-on-scrim)]">
              {metadataSource}
            </span>
          )}
          {showTypeBadge && (
            <span className="rounded bg-[var(--overlay-scrim)] px-2 py-0.5 text-xs font-medium text-[var(--text-on-scrim)]">
              {type === 'movie' ? '電影' : '影集'}
            </span>
          )}
        </div>

        {/* Kebab menu — MQbvp: top-RIGHT slot (was top-LEFT in pre-bugfix-10-4), hover-only via group-hover */}
        {onMenuClick && (
          <button
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onMenuClick(e);
            }}
            // Kebab floats on POSTER ART, so its backing is the scrim token — dark in
            // both themes. The icon stays literal white: it is the only value that
            // clears the scrim in BOTH themes (see the report on --text-on-scrim).
            // The 90% hover keeps its own alpha; no token carries it.
            className="absolute right-2 top-2 z-20 rounded-full bg-[var(--overlay-scrim)] p-1.5 text-[var(--text-on-scrim)] opacity-0 transition-opacity duration-[var(--motion-state)] hover:bg-[var(--overlay-scrim)] lg:group-hover:opacity-100"
            aria-label="更多選項"
            data-testid="poster-menu-button"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
        )}

        {/* The MQbvp hover ▶ overlay was REMOVED (critique R2 P1): a big play
            affordance promised playback the product does not have — the click
            lands on the detail page. 誠實優先於好看: the card is a link, and
            the existing hover scale + title tint already say「可以點」without
            claiming「可以播」. */}
        {/* Note: MQbvp design originally specified a bottom-left title/year overlay,
            but Party Mode 2026-05-08 (Sally + Alexyu) determined this duplicates the
            below-image title (RusTY) and has legibility issues against varying poster
            backgrounds. Hover state is now action-trigger only (play + kebab + rating);
            the in-card info-density was instead delivered below the image (year +
            runtime/episode-count, lazy-fetched on hover) by bugfix-10-7 — see the mt-2 block. */}

        {/* Rating badge — MQbvp: bottom-RIGHT slot (was bottom-LEFT in pre-bugfix-10-4), always visible when voteAverage > 0.
            bugfix-10-7 AC #3: lucide <Star> SVG (not the ⭐ emoji) for cross-OS rendering consistency.
            Story 13-1b: recedes on hover when the 想要 scrim takes the bottom edge —
            same collision strategy as the top-right badge cluster vs the kebab. */}
        {voteAverage !== undefined && voteAverage > 0 && (
          <div
            // Critique R1 P2: aligned with PosterCardV2's recipe — bottom-LEFT,
            // scrim token, mono digits, 宣紙白. Two card systems on one page had
            // opposite rating corners; and --warning is a STATUS colour, not a
            // rating decoration (固定詞彙).
            className={cn(
              'absolute bottom-2 left-2 z-20',
              showRequestOverlay &&
                'transition-opacity duration-[var(--motion-state)] lg:group-hover:opacity-0'
            )}
          >
            <span className="flex items-center gap-1 rounded-full bg-[var(--overlay-scrim)] px-2 py-0.5 font-mono text-xs text-[var(--text-on-scrim)]">
              <Star className="h-3 w-3 fill-current" aria-hidden="true" />
              {voteAverage.toFixed(1)}
            </span>
          </div>
        )}

        {/* 想要 hover scrim — Story 13-1b (design L2 card context): bottom gradient
            scrim with the full-width request affordance. Desktop-hover only (mobile
            requests via the detail page, L4); pointer-events gated so the invisible
            overlay never blocks the card <Link> before hover. */}
        {showRequestOverlay && (
          <div
            data-testid="poster-request-overlay"
            // CR M2: group-focus-within mirrors group-hover so keyboard users who Tab
            // onto the (otherwise invisible) button get the revealed scrim, not a
            // focus ring floating on an invisible control.
            // The gradient's dark stop is a scrim over POSTER ART, so it shares the
            // scrim token with the badges and the kebab above rather than being a
            // fourth private black. Its alpha moves 80% → the token's 70%.
            className="pointer-events-none absolute inset-x-0 bottom-0 z-10 hidden bg-gradient-to-t from-[var(--overlay-scrim)] to-transparent p-3 pt-8 opacity-0 transition-opacity duration-[var(--motion-state)] lg:block lg:group-hover:pointer-events-auto lg:group-hover:opacity-100 lg:group-focus-within:pointer-events-auto lg:group-focus-within:opacity-100"
          >
            <RequestButton
              tmdbId={tmdbId}
              mediaType={type}
              title={title}
              owned={false}
              requested={!!isRequested}
              fullWidth
            />
          </div>
        )}
      </div>

      {/* Title + metadata line — below-image affordance. bugfix-10-7 AC #1: the metadata line
          is `{year} · {extra}` where `extra` is the runtime (movies) or `{seasons} 季 {episodes} 集`
          (series), lazy-fetched on hover. Stays year-only until the fetch resolves (and for
          owned-library UUID cards / touch devices, which never fetch). The <p> is ALWAYS
          rendered — falling back to a non-breaking space when there's nothing to show — so the
          card height never changes: a year-less card mustn't grow a line when the runtime
          resolves on hover (AC #1: "MUST NOT push the card layout" — bugfix-10-7 CR M1).
          `truncate` keeps it on one line; only the title gets <HighlightText>. */}
      {/* Critique R1 P2: same 2-line CJK title grid as PosterCardV2 (§3.3) —
          the two card systems disagreed on title lines AND hardcoded white. */}
      <div className="mt-2">
        <h3
          className="line-clamp-2 min-h-[2.75em] text-sm font-medium leading-snug text-[var(--text-primary)]"
          title={title}
        >
          <HighlightText text={title} query={highlightQuery} />
        </h3>
        <p className="truncate font-mono text-[11px] text-[var(--text-secondary)]">
          {metaLine || '\u00A0'}
        </p>
      </div>
    </Link>
  );
}
