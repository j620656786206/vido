// Design ref: ux-design.pen Screen 4 Detail Panel Desktop (RgSxQ)
// Story 12-1 — dual rating section; Douban rating postdates the .pen design.
import { Star } from 'lucide-react';
import { formatVoteCount } from '../../utils/formatVoteCount';
import { cn } from '../../lib/utils';

export interface DualRatingDisplayProps {
  /** TMDb rating on a 0-10 scale. */
  tmdbRating?: number;
  /** TMDb vote count (abbreviated for display). */
  tmdbVoteCount?: number;
  /** Douban rating on a 0-10 scale. */
  doubanRating?: number;
  /** Douban vote count (abbreviated for display). */
  doubanVoteCount?: number;
  /** When true, the Douban slot shows a loading skeleton (enrichment in flight). */
  doubanLoading?: boolean;
  className?: string;
}

function RatingBadge({
  label,
  rating,
  voteCount,
}: {
  label: string;
  rating: number;
  voteCount?: number;
}) {
  return (
    <div className="flex items-center gap-1.5" data-testid={`rating-${label}`}>
      <span className="text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
        {label}
      </span>
      <Star className="h-4 w-4 fill-[var(--warning)] text-[var(--warning)]" aria-hidden="true" />
      <span className="font-semibold text-[var(--warning)]">{rating.toFixed(1)}</span>
      {voteCount != null && voteCount > 0 && (
        <span className="text-xs text-[var(--text-secondary)]">({formatVoteCount(voteCount)})</span>
      )}
    </div>
  );
}

function RatingSkeleton({ label }: { label: string }) {
  return (
    // role=status + aria-live=polite so the rating is announced when it resolves
    // (project-context a11y pre-flight: aria-live on async-revealed content).
    <div
      className="flex items-center gap-1.5"
      data-testid={`rating-skeleton-${label}`}
      role="status"
      aria-live="polite"
      aria-label={`正在載入 ${label} 評分`}
    >
      <span className="text-xs font-medium uppercase tracking-wide text-[var(--text-secondary)]">
        {label}
      </span>
      <div className="h-4 w-16 animate-pulse rounded bg-[var(--bg-tertiary)]" />
    </div>
  );
}

/**
 * DualRatingDisplay renders TMDb and Douban ratings side-by-side (Story 12-1).
 * Horizontal on desktop, stacked on mobile. The Douban slot shows a skeleton
 * while enrichment is in flight and disappears entirely when no Douban rating
 * exists (graceful degradation — AC #4/#7).
 */
export function DualRatingDisplay({
  tmdbRating,
  tmdbVoteCount,
  doubanRating,
  doubanVoteCount,
  doubanLoading,
  className,
}: DualRatingDisplayProps) {
  const hasTmdb = tmdbRating != null && tmdbRating > 0;
  const hasDouban = doubanRating != null && doubanRating > 0;

  // Nothing to render: no TMDb rating, no Douban rating, and not loading.
  if (!hasTmdb && !hasDouban && !doubanLoading) {
    return null;
  }

  return (
    <div
      className={cn('flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-4', className)}
      data-testid="dual-rating-display"
    >
      {hasTmdb && <RatingBadge label="TMDb" rating={tmdbRating} voteCount={tmdbVoteCount} />}
      {hasDouban ? (
        <RatingBadge label="豆瓣" rating={doubanRating} voteCount={doubanVoteCount} />
      ) : (
        doubanLoading && <RatingSkeleton label="豆瓣" />
      )}
    </div>
  );
}
