import { useMemo } from 'react';
import { useQueries } from '@tanstack/react-query';
import { useExploreBlocks, exploreBlockKeys } from '../../hooks/useExploreBlocks';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';
import { exploreBlockService } from '../../services/exploreBlockService';
import { ExploreBlock } from './ExploreBlock';
import type { ExploreBlock as ExploreBlockType } from '../../services/exploreBlockService';
import type { Movie, TVShow } from '../../types/tmdb';

const FIVE_MINUTES = 5 * 60 * 1000;

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
  // useQueries handles a dynamic number of queries safely (no rules-of-hooks
  // violation when block count changes). Shares keys with the child's
  // useExploreBlockContent so children hit the cache.
  const contentQueries = useQueries({
    queries: blocks.map((block) => ({
      queryKey: exploreBlockKeys.content(block.id),
      queryFn: () => exploreBlockService.getContent(block.id),
      staleTime: FIVE_MINUTES,
      retry: 1,
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
    <div data-testid="explore-blocks-list" className="bg-[var(--bg-primary)]">
      {blocks.map((block) => (
        <ExploreBlock key={block.id} block={block} ownership={ownership} />
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
