import { useEffect, useMemo, useRef } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronRight, ChevronLeft } from 'lucide-react';
import { useExploreBlockContent } from '../../hooks/useExploreBlocks';
import { useInViewport } from '../../hooks/useInViewport';
import type { OwnedMediaState } from '../../hooks/useOwnedMedia';
import type { ExploreBlock as ExploreBlockType } from '../../services/exploreBlockService';
import type { Movie, TVShow } from '../../types/tmdb';
import { PosterCard } from '../media/PosterCard';
import { ExploreBlockSkeleton } from './ExploreBlockSkeleton';
import { cn } from '../../lib/utils';

interface ExploreBlockProps {
  block: ExploreBlockType;
  // Story 10-4 AC #4 — ownership state is owned by the parent
  // ExploreBlocksList so a multi-block homepage issues one POST, not N.
  ownership: OwnedMediaState;
  // Story 10-5 Task 2.3 — above-the-fold blocks fetch immediately; below-the-fold
  // blocks wait until they approach the viewport before firing their network
  // request. Defaults to eager=true so existing single-block tests stay green.
  eager?: boolean;
  // Parent callback so ExploreBlocksList can enable its own useQueries slot
  // once the block becomes visible (keeps hoisted ownership in sync — see
  // Story 10-4 AC #4).
  onVisible?: () => void;
}

/**
 * ExploreBlock — horizontal scrollable row of TMDb discover results.
 *
 * Story 10.3 AC #1. Hides itself gracefully on empty / error to avoid
 * rendering a broken stub on the homepage (mirrors HeroBanner AC #5 pattern).
 */
export function ExploreBlock({ block, ownership, eager = true, onVisible }: ExploreBlockProps) {
  const sectionRef = useRef<HTMLElement | null>(null);
  // Eager blocks already pass their own visibility assertion to parent; skip
  // mounting a no-op observer for them. rootMargin trades a little bandwidth
  // for a smoother reveal on lazy blocks — posters are in place by the time
  // the user sees the block.
  const isInViewport = useInViewport(sectionRef, {
    rootMargin: '400px',
    once: true,
    disabled: eager,
  });
  const shouldFetch = eager || isInViewport;

  const { data, isLoading, isError } = useExploreBlockContent(shouldFetch ? block.id : undefined);
  const scrollerRef = useRef<HTMLDivElement | null>(null);

  const items = useMemo(() => getBlockItems(data), [data]);
  const { isOwned, isRequested } = ownership;

  // Notify the parent once per block so its useQueries slot enables — shared
  // TanStack Query cache means only one network request is actually issued.
  useEffect(() => {
    if (isInViewport && !eager && onVisible) onVisible();
  }, [isInViewport, eager, onVisible]);

  if (isError) return null;

  // Match the "查看更多" destination to block content type — routes to
  // the matching TMDb discover view pre-scoped to this block's filters.
  const seeMoreTo = buildSeeMoreTarget(block);

  const scroll = (direction: 'left' | 'right') => {
    const el = scrollerRef.current;
    if (!el) return;
    const delta = direction === 'right' ? el.clientWidth * 0.8 : -el.clientWidth * 0.8;
    el.scrollBy({ left: delta, behavior: 'smooth' });
  };

  // While waiting to enter the viewport (lazy) or while the query is inflight,
  // we still want to reserve space so the page doesn't jitter as blocks pop in.
  const showSkeleton = !shouldFetch || isLoading;

  return (
    <section
      ref={sectionRef}
      data-testid={`explore-block-${block.id}`}
      aria-label={block.name}
      className="mx-auto w-full max-w-7xl px-4 sm:px-6"
    >
      <div className="mb-3 flex items-end justify-between">
        <h2
          className="text-lg font-semibold text-[var(--text-primary)] md:text-xl"
          data-testid="explore-block-title"
        >
          {block.name}
        </h2>
        <Link
          to={seeMoreTo.to}
          search={seeMoreTo.search}
          className="flex items-center gap-1 text-sm text-[var(--text-secondary)] hover:text-[var(--accent-primary)]"
          data-testid="explore-block-see-more"
        >
          查看更多
          <ChevronRight className="h-4 w-4" />
        </Link>
      </div>

      <div className="relative">
        {/* Desktop scroll chevrons — hidden on touch */}
        <button
          type="button"
          onClick={() => scroll('left')}
          aria-label="向左捲動"
          data-testid="explore-block-scroll-left"
          className="absolute left-0 top-1/2 z-10 hidden -translate-x-1/2 -translate-y-1/2 rounded-full bg-black/70 p-2 text-white hover:bg-black/90 lg:block"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <button
          type="button"
          onClick={() => scroll('right')}
          aria-label="向右捲動"
          data-testid="explore-block-scroll-right"
          className="absolute right-0 top-1/2 z-10 hidden translate-x-1/2 -translate-y-1/2 rounded-full bg-black/70 p-2 text-white hover:bg-black/90 lg:block"
        >
          <ChevronRight className="h-5 w-5" />
        </button>

        {showSkeleton ? (
          <ExploreBlockSkeleton />
        ) : (
          <div
            ref={scrollerRef}
            data-testid="explore-block-scroller"
            className={cn(
              'flex gap-4 overflow-x-auto pb-2 snap-x',
              'scrollbar-thin scrollbar-thumb-[var(--bg-tertiary)]'
            )}
          >
            {items.map((item) => (
              <div
                key={`${item.type}-${item.id}`}
                className="w-[140px] shrink-0 snap-start sm:w-[160px]"
              >
                <PosterCard
                  id={String(item.id)}
                  type={item.type}
                  title={item.title}
                  originalTitle={item.originalTitle}
                  posterPath={item.posterPath}
                  releaseDate={item.releaseDate}
                  voteAverage={item.voteAverage}
                  overview={item.overview}
                  genreIds={item.genreIds}
                  isOwned={isOwned(item.id)}
                  isRequested={isRequested(item.id)}
                />
              </div>
            ))}

            {items.length === 0 && (
              <div
                className="py-8 text-sm text-[var(--text-muted)]"
                data-testid="explore-block-empty"
              >
                沒有符合條件的內容
              </div>
            )}
          </div>
        )}
      </div>
    </section>
  );
}

interface DisplayItem {
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

function getBlockItems(data: { movies?: Movie[]; tvShows?: TVShow[] } | undefined): DisplayItem[] {
  if (!data) return [];
  // Merge both arrays so mixed discover results render side-by-side. Backend
  // today returns one type per block; this stays correct if that changes.
  const items: DisplayItem[] = [];
  if (data.movies?.length) {
    for (const m of data.movies) {
      items.push({
        id: m.id,
        type: 'movie' as const,
        title: m.title,
        originalTitle: m.originalTitle,
        posterPath: m.posterPath,
        releaseDate: m.releaseDate,
        voteAverage: m.voteAverage,
        overview: m.overview,
        genreIds: m.genreIds,
      });
    }
  }
  if (data.tvShows?.length) {
    for (const t of data.tvShows) {
      items.push({
        id: t.id,
        type: 'tv' as const,
        title: t.name,
        originalTitle: t.originalName,
        posterPath: t.posterPath,
        releaseDate: t.firstAirDate,
        voteAverage: t.voteAverage,
        overview: t.overview,
        genreIds: t.genreIds,
      });
    }
  }
  return items;
}

// "查看更多" routes into the main search view with filter pre-applied when
// filter scaffolding (Epic 11) lands. For now, route to /search — the target
// can be refined later without touching ExploreBlock call sites.
function buildSeeMoreTarget(_block: ExploreBlockType): {
  to: string;
  search: Record<string, unknown>;
} {
  return {
    to: '/search',
    search: {},
  };
}
