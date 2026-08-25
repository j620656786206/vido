// Design ref: ux-design.pen Screen HP-1 Homepage Desktop (sAaCR)
import { useCallback, useMemo, useState } from 'react';
import { Link } from '@tanstack/react-router';
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
 * Story 10-4 AC #4 — availability lookup is hoisted here so each visibility
 * batch issues ONE `POST /media/check-owned` covering every enabled block,
 * not one POST per block (the pre-10-4 N+1 pattern). Children receive
 * `isOwned` / `isRequested` predicates as props; block content is observed
 * via `useQueries` with the same keys the children use, so TanStack Query
 * dedupes the fetches.
 *
 * Story 10-5 Task 2.3 — below-the-fold blocks are lazy-loaded. As lazy
 * blocks reveal, each newly-settled batch expands the id list and triggers
 * a fresh ownership POST (cumulative — includes every id seen so far). So a
 * 4-block homepage (2 eager + 2 lazy, user scrolls to the end) issues up to
 * 3 POSTs: one when eagers settle, one per lazy-block settle. Cannot be
 * further reduced without giving up the lazy-load bandwidth win.
 *
 * The `tmdbIds` memo intentionally does NOT update while a currently-enabled
 * content query is inflight — prevents firing a mid-fetch POST with only a
 * partial batch, which would double-up when the remaining query settles.
 */
export function ExploreBlocksList() {
  const { data, isLoading, isError } = useExploreBlocks();

  // 固定詞彙 (critique R1 P1): the user CONFIGURED these blocks — when they
  // cannot load, silence renders「不存在＝你沒要求」, which is a lie. The amber
  // line is the honest state: you asked for this, and it is not happening.
  if (isError) return <ExploreDegradedNotice reason="blocks" />;

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

  // Stability gate: if ANY currently-enabled content query is still inflight,
  // skip the ownership recompute this render. Otherwise a lazy-block reveal
  // would trigger one POST for the partial batch (just the eagers) and then a
  // second POST once the lazy block resolves. Waiting for the full enabled
  // batch to settle collapses that to one POST per batch.
  const anyEnabledInflight = blocks.some(
    (_, index) => isEager(index) && contentQueries[index]?.isLoading
  );

  const tmdbIds = useMemo(() => {
    if (anyEnabledInflight) return [];
    const ids: number[] = [];
    for (const q of contentQueries) {
      ids.push(...collectIds(q.data));
    }
    return ids;
  }, [contentQueries, anyEnabledInflight]);

  const ownership = useOwnedMedia(tmdbIds);

  // Same honesty rule per block: a block whose CONTENT query failed hides
  // itself (fail-soft, F3) but must not vanish silently — one quiet page-level
  // line accounts for every hidden block instead of per-block noise.
  const failedCount = blocks.filter(
    (_, index) => isEager(index) && contentQueries[index]?.isError
  ).length;

  return (
    <div data-testid="explore-blocks-list" className="flex flex-col gap-6 md:gap-8">
      {failedCount > 0 && <ExploreDegradedNotice reason="content" count={failedCount} />}
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

/**
 * The degraded-state readout (critique R1 P1 — skeleton-then-vanish was the
 * page claiming the user never configured explore blocks). Quiet inline line,
 * warning vocabulary (asked-but-not-happening), with the one door that can
 * actually fix it. role="status": informative, not interruptive.
 */
function ExploreDegradedNotice({
  reason,
  count,
}: {
  reason: 'blocks' | 'content';
  count?: number;
}) {
  const text =
    reason === 'blocks' ? '探索區塊目前無法載入' : `${count} 個探索區塊的內容目前無法載入`;
  return (
    <div
      role="status"
      data-testid="explore-degraded-notice"
      className="mx-auto w-full max-w-7xl px-4 sm:px-6"
    >
      <p className="flex flex-wrap items-center gap-x-2 rounded-[var(--radius-md)] bg-[var(--warning-tint)] px-3 py-2 text-sm text-[var(--warning-text)]">
        {text}（TMDb 未設定或無法連線）
        <Link
          to="/settings/connection"
          className="font-medium underline underline-offset-2 hover:text-[var(--text-primary)]"
        >
          前往連線設定
        </Link>
      </p>
    </div>
  );
}

function collectIds(data: { movies?: Movie[]; tvShows?: TVShow[] } | undefined): number[] {
  if (!data) return [];
  // Merge both arrays — a block can in principle return both types (mixed
  // discover results). `useOwnedMedia` dedupes anyway, so belt-and-braces.
  const ids: number[] = [];
  if (data.movies?.length) ids.push(...data.movies.map((m) => m.id));
  if (data.tvShows?.length) ids.push(...data.tvShows.map((t) => t.id));
  return ids;
}
