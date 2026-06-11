// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page recommendations section postdates the .pen design
// Tiles reuse PosterCard (Component/PosterCard RusTY), which carries its own .pen link.
import { PosterCard } from './PosterCard';
import { PosterCardSkeleton } from './PosterCardSkeleton';
import type { RecommendationItem } from '../../types/library';

export interface RelatedContentProps {
  items: RecommendationItem[];
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
}

// Single source of truth for the section heading id so the <section> landmark is
// labelled by the visible <h2> (aria-labelledby) instead of duplicating the text
// in an aria-label (avoids double screen-reader announcement). Only one section
// branch renders at a time, so the id is unique in the DOM.
const HEADING_ID = 'related-content-heading';

// Shared responsive grid class — reuses MediaGrid's layout (reuse-over-reinvent,
// no new responsive CSS; Story 12-3 AC #8).
const GRID_CLASS =
  'grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] md:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]';

/**
 * RelatedContent renders the "相關推薦" section on a media detail page (Story 12-3).
 * It is fail-soft (Rule 27 Pillar 3): a load error or empty result NEVER throws or
 * breaks the rest of the page — it shows a quiet retry affordance or renders nothing.
 * Each tile reuses PosterCard with the "已有" owned badge driven by item.isOwned.
 */
export function RelatedContent({ items, isLoading, isError, onRetry }: RelatedContentProps) {
  if (isLoading) {
    return (
      <section
        aria-labelledby={HEADING_ID}
        className="flex flex-col gap-3"
        data-testid="related-content"
      >
        <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
          相關推薦
        </h2>
        <div className={GRID_CLASS} data-testid="related-content-skeleton">
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <PosterCardSkeleton key={i} />
          ))}
        </div>
      </section>
    );
  }

  if (isError) {
    return (
      <section
        aria-labelledby={HEADING_ID}
        className="flex flex-col gap-3"
        data-testid="related-content"
      >
        <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
          相關推薦
        </h2>
        <div
          role="alert"
          className="flex flex-col items-center gap-3 rounded-lg border border-[var(--border-subtle)] px-4 py-6 text-center"
          data-testid="related-content-error"
        >
          <p className="text-sm text-[var(--text-secondary)]">無法載入相關推薦,請稍後再試。</p>
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="rounded-md border border-[var(--border-subtle)] px-3 py-1.5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
            >
              重試
            </button>
          )}
        </div>
      </section>
    );
  }

  // Empty results — render nothing (quiet, never an error; AC #5).
  if (!items || items.length === 0) return null;

  // Note: items carry a `source` ("recommendations" | "similar") on the response,
  // but the design intentionally does not surface a 推薦/類似 label yet (Story 12-3
  // Critical Detail #1). Wire it to a sub-heading here when that label is designed.
  return (
    <section
      aria-labelledby={HEADING_ID}
      className="flex flex-col gap-3"
      data-testid="related-content"
    >
      <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
        相關推薦
      </h2>
      <div className={GRID_CLASS}>
        {items.map((item) => (
          <PosterCard
            key={`${item.mediaType}-${item.id}`}
            id={String(item.id)}
            type={item.mediaType}
            title={item.title}
            posterPath={item.posterPath}
            releaseDate={item.releaseDate}
            voteAverage={item.voteAverage}
            isOwned={item.isOwned}
          />
        ))}
      </div>
    </section>
  );
}
