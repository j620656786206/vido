import { PosterCardSkeleton } from '../media/PosterCardSkeleton';

interface ExploreBlockSkeletonProps {
  // Matches the live block's card count so the layout reserves the right
  // height and prevents CLS when real data swaps in. Defaults to 6 — the
  // desktop visible row count per Story 10-5 Task 4.2.
  count?: number;
}

/**
 * Horizontal row of PosterCardSkeleton placeholders.
 *
 * Story 10-5 Task 3.1. Extracted from the inline skeleton in ExploreBlock so
 * both the block and the homepage-level loading state can render the same
 * placeholder without duplicating the wrapper classes.
 */
export function ExploreBlockSkeleton({ count = 6 }: ExploreBlockSkeletonProps) {
  return (
    <div
      data-testid="explore-block-skeleton-row"
      className="flex gap-4 overflow-hidden pb-2"
      aria-hidden="true"
    >
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={`skeleton-${i}`}
          data-testid="explore-block-skeleton"
          className="w-[140px] shrink-0 sm:w-[160px]"
        >
          <PosterCardSkeleton />
        </div>
      ))}
    </div>
  );
}
