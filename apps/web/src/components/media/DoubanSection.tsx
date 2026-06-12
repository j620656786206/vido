// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page Douban review section postdates the .pen design
import type { ReactNode } from 'react';
import type { DoubanReviewSummary } from '../../types/library';

export interface DoubanSectionProps {
  doubanId?: string;
  summary?: DoubanReviewSummary | null;
  isLoading?: boolean;
  isError?: boolean;
}

// Single source of truth for the heading id so the <section> landmark is labelled
// by its visible <h2> (aria-labelledby) — mirrors TrailerSection / StreamingAvailability.
const HEADING_ID = 'douban-section-heading';

const MAX_COMMENTS = 5;
// Douban rates on a 1–5 star scale. Kept distinct from MAX_COMMENTS so the comment
// cap and the star-scale maximum stay independent (CR M2).
const MAX_STARS = 5;

function SectionShell({ children }: { children: ReactNode }) {
  return (
    <section
      aria-labelledby={HEADING_ID}
      className="flex flex-col gap-3"
      data-testid="douban-section"
    >
      <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
        豆瓣評論
      </h2>
      {children}
    </section>
  );
}

// Render a 0–5 star score as filled/empty glyphs with an accessible label.
function StarRating({ rating }: { rating: number }) {
  const filled = Math.max(0, Math.min(MAX_STARS, Math.round(rating)));
  return (
    <span
      role="img"
      aria-label={`${filled} 星`}
      className="text-sm text-[var(--accent-primary)]"
      data-testid="douban-comment-rating"
    >
      {'★'.repeat(filled)}
      <span className="text-[var(--text-muted)]">{'☆'.repeat(MAX_STARS - filled)}</span>
    </span>
  );
}

/**
 * DoubanSection renders the "豆瓣評論" section on a media detail page (Story 12-6):
 * a direct link to the title's Douban subject page (AC #1) plus a short summary of
 * Douban user short comments (短評, AC #2). Fail-soft (Rule 27 Pillar 3): it NEVER
 * throws — when the review scrape fails / is empty the review block is omitted while
 * the direct link still renders (AC #5); when no doubanId is known the whole section
 * is omitted (AC #4).
 *
 * The direct link is built client-side from the doubanId the rating query already
 * exposes (Story 12-1's useDoubanRating) — no new backend field is needed.
 */
export function DoubanSection({ doubanId, summary, isLoading, isError }: DoubanSectionProps) {
  // AC #4 — no resolved douban_id: omit the entire section (no link, no reviews).
  if (!doubanId) return null;

  const doubanUrl = `https://movie.douban.com/subject/${doubanId}/`;
  const comments = summary?.topComments?.slice(0, MAX_COMMENTS) ?? [];
  const hasComments = comments.length > 0;

  return (
    <SectionShell>
      {/* AC #1 — direct link; renders even when the review scrape fails (AC #5). */}
      <a
        href={doubanUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="text-sm font-medium text-[var(--accent-primary)] hover:underline"
        data-testid="douban-page-link"
      >
        查看豆瓣頁面
      </a>

      {/* Review summary block. Loading → quiet skeleton; error / empty → omitted
          (the direct link above stays). Never throws (AC #5). */}
      {isLoading && !hasComments ? (
        <div
          className="h-16 animate-pulse rounded-md bg-[var(--bg-tertiary)]"
          data-testid="douban-reviews-skeleton"
          aria-hidden="true"
        />
      ) : !isError && hasComments ? (
        <div className="flex flex-col gap-3" data-testid="douban-reviews">
          {summary && summary.totalComments > 0 ? (
            <p className="text-xs text-[var(--text-muted)]" data-testid="douban-reviews-count">
              共 {summary.totalComments.toLocaleString('zh-TW')} 則短評
            </p>
          ) : null}
          {/* One comment per row (AC #8): author + stars on one line, text below. */}
          <ul className="flex flex-col gap-3">
            {comments.map((comment, index) => (
              <li
                key={`${comment.author}-${index}`}
                className="flex flex-col gap-1 border-b border-[var(--border-subtle)] pb-3 last:border-b-0 last:pb-0"
                data-testid="douban-comment"
              >
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-[var(--text-primary)]">
                    {comment.author}
                  </span>
                  {comment.rating > 0 ? <StarRating rating={comment.rating} /> : null}
                </div>
                <p className="text-sm text-[var(--text-secondary)]">{comment.text}</p>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </SectionShell>
  );
}
