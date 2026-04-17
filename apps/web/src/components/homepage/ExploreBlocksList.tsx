import { useCallback, useMemo, useState } from 'react';
import { useQueries } from '@tanstack/react-query';
import { useExploreBlocks, exploreBlockKeys } from '../../hooks/useExploreBlocks';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';
import { exploreBlockService } from '../../services/exploreBlockService';
import { ExploreBlock } from './ExploreBlock';
import type { ExploreBlock as ExploreBlockType } from '../../services/exploreBlockService';
import type { Movie, TVShow } from '../../types/tmdb';

const FIVE_MINUTES = 5 * 60 * 1000;
// Story 10-5 Task 2.3 — above-the-fold blocks that MUST fetch immediately to
// keep LCP under 2s. The rest wait until their IntersectionObserver fires.
const EAGER_BLOCK_COUNT = 2;

/**
 * ExploreBlocksList — renders every configured explore block in sort_order.
 * Hides itself while loading to avoid shifting layout beneath HeroBanner,
 * and on error (same AC #5 spirit as HeroBanner).
 *
 * Story 10.3 AC #1, #5.
 *
 * Story 10-4 AC #4 — availability lookup is hoisted here so a multi-block
 * homepage still issues a single `POST /media/check-owned`. The parent owns
 * the ownership query; children receive `isOwned` / `isRequested` predicates
 * as props instead of each running their own hook. Block content is observed
 * via `useQueries` with the same keys the children use, so TanStack Query
 * dedupes the fetches — no extra network traffic.
 *
 * Story 10-5 Task 2.3 — below-the-fold blocks are lazy-loaded. The parent
 * tracks which indices have scrolled into view and enables each useQueries
 * slot only once its block is eager (above-the-fold) or visible. The shared
 * cache keys mean the child's useExploreBlockContent hook and the parent's
 * useQueries fetch once, never twice.
 */
export function ExploreBlocksList() {
  const { data, isLoading, isError } = useExploreBlocks();

  if (isError) return null;

  // L2 fix: reserve vertical space during loading to prevent layout shift
  if (isLoading) {
    return (
      <div
        data-testid="explore-blocks-loading"
        className="min-h-[200px] bg-[var(--bg-primary)]"
        aria-busy="true"
      />
    );
  }

  const blocks = data?.blocks ?? [];
  if (blocks.length === 0) return null;

  return <ExploreBlocksListInner blocks={blocks} />;
}

function ExploreBlocksListInner({ blocks }: { blocks: ExploreBlockType[] }) {
  const [visibleIndices, setVisibleIndices] = useState<Set<number>>(() => new Set());

  const markVisible = useCallback((index: number) => {
    setVisibleIndices((prev) => {
      if (prev.has(index)) return prev;
      const next = new Set(prev);
      next.add(index);
      return next;
    });
  }, []);

  const isEager = useCallback(
    (index: number) => index < EAGER_BLOCK_COUNT || visibleIndices.has(index),
    [visibleIndices]
  );

  // useQueries handles a dynamic number of queries safely (no rules-of-hooks
  // violation when block count changes). Shares keys with the child's
  // useExploreBlockContent so children hit the cache.
  const contentQueries = useQueries({
    queries: blocks.map((block, index) => ({
      queryKey: exploreBlockKeys.content(block.id),
      queryFn: () => exploreBlockService.getContent(block.id),
      staleTime: FIVE_MINUTES,
      retry: 1,
      enabled: isEager(index),
    })),
  });

  const tmdbIds = useMemo(() => {
    const ids: number[] = [];
    for (const q of contentQueries) {
      ids.push(...collectIds(q.data));
    }
    return ids;
  }, [contentQueries]);

  const ownership = useOwnedMedia(tmdbIds);

  return (
    <div data-testid="explore-blocks-list" className="flex flex-col gap-6 md:gap-8">
      {blocks.map((block, index) => (
        <ExploreBlock
          key={block.id}
          block={block}
          ownership={ownership}
          eager={index < EAGER_BLOCK_COUNT}
          onVisible={() => markVisible(index)}
        />
      ))}
    </div>
  );
}

function collectIds(data: { movies?: Movie[]; tvShows?: TVShow[] } | undefined): number[] {
  if (!data) return [];
  if (data.movies?.length) return data.movies.map((m) => m.id);
  if (data.tvShows?.length) return data.tvShows.map((t) => t.id);
  return [];
}
