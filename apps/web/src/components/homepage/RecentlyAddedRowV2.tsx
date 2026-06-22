// Design ref: ux-design.pen Screen H1-D-v2 (yixu1) — own-content zone, 最近新增 row (gda31)
/**
 * Recently-added own-content row (UX Redesign Phase 3 — ux3-1-2).
 *
 * The v2 replacement for the legacy dashboard `RecentMediaPanel` (party-mode finding:
 * do NOT raw-reuse the legacy panel or home looks half-legacy). A single horizontal
 * PosterCardV2 row whose badge reflects ux3-0-1's lifecycle field via ux3-0-2's
 * `pickPosterBadge`. It is the second element of the D3 own-above-external zone (below
 * 繼續觀看, above Hero/Explore).
 *
 * Four states (§7 / N4), each fail-soft (F3) so the home page never hard-fails:
 *  - Loading  → poster-shaped skeleton row (reduced-motion aware), matches H4-D-v2.
 *  - Empty    → quiet 「尚無最近新增」 hint, graceful collapse with no top gap (H5-D-v2).
 *  - Error    → compact inline error-tint banner + 重試 (H6-D-v2); the section degrades
 *               alone, Hero/Explore below still render.
 *  - Data     → horizontal scroller of PosterCardV2.
 *
 * Header carries the 進行中 · N lifecycle chip (H1-D-v2): N = items currently being
 * processed (parseStatus 'pending' = 整理中), derived from the same data the row already
 * fetches — no extra request. Hidden when N = 0 (exception-signal only, like the badge).
 *
 * Token-only colors; Noto Sans TC (CJK) + JetBrains Mono (the numeric chip). 44px touch
 * floor on the 重試 control (N5).
 */
import { AlertTriangle } from 'lucide-react';
import { useRecentlyAdded } from '../../hooks/useLibrary';
import { PosterCardV2 } from '../library/PosterCardV2';
import type { LibraryItem, LibraryMovie, LibrarySeries } from '../../types/library';

const RECENT_LIMIT = 20;
const SKELETON_COUNT = 8;

interface CardFields {
  id: string;
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  year?: string;
  meta: string;
  voteAverage?: number;
  media: LibraryMovie | LibrarySeries;
}

function toCard(item: LibraryItem): CardFields | null {
  const isMovie = item.type === 'movie';
  const media = isMovie ? item.movie : item.series;
  if (!media) return null;
  const date = isMovie ? item.movie?.releaseDate : item.series?.firstAirDate;
  const year = date ? date.slice(0, 4) : undefined;
  const meta = isMovie
    ? item.movie?.runtime
      ? `${item.movie.runtime} 分`
      : ''
    : item.series?.numberOfSeasons
      ? `${item.series.numberOfSeasons} 季`
      : '';
  return {
    id: media.id,
    type: isMovie ? 'movie' : 'tv',
    title: media.title,
    posterPath: media.posterPath,
    year,
    meta,
    voteAverage: media.voteAverage,
    media,
  };
}

/** Items still being processed (parseStatus 'pending' = 整理中) — the 進行中 · N chip. */
function countInProgress(items: LibraryItem[]): number {
  return items.filter((it) => (it.movie ?? it.series)?.parseStatus === 'pending').length;
}

export function RecentlyAddedRowV2() {
  const { data, isLoading, isError, refetch } = useRecentlyAdded(RECENT_LIMIT);
  const items = data ?? [];
  const cards = items.map(toCard).filter(Boolean) as CardFields[];
  const inProgress = countInProgress(items);

  return (
    <section data-testid="home-recently-added" aria-labelledby="home-ra-title">
      <div className="mb-3 flex items-center justify-between">
        <h2 id="home-ra-title" className="text-xl font-semibold text-[var(--text-primary)]">
          最近新增
        </h2>
        {/* 進行中 · N — lifecycle chip, exception-signal only (hidden at 0). */}
        {inProgress > 0 && (
          <span
            data-testid="home-recent-progress"
            className="flex items-center gap-1 rounded-full bg-[var(--accent-tint)] px-2.5 py-1 text-xs font-medium text-[var(--accent-text)]"
          >
            進行中
            <span className="font-mono tabular-nums">· {inProgress}</span>
          </span>
        )}
      </div>

      {isLoading ? (
        <div
          data-testid="home-recent-skeleton"
          aria-busy="true"
          aria-label="載入中"
          className="flex gap-3 overflow-hidden md:gap-4"
        >
          {Array.from({ length: SKELETON_COUNT }).map((_, i) => (
            <div key={i} className="w-[140px] shrink-0 sm:w-[160px]">
              <div className="aspect-[2/3] animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] motion-reduce:animate-none" />
              <div className="mt-2 h-3.5 w-4/5 animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
              <div className="mt-1 h-2.5 w-2/5 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
            </div>
          ))}
        </div>
      ) : isError ? (
        // Fail-soft (F3): the section degrades alone; Hero/Explore below still render.
        <div
          data-testid="home-recent-error"
          role="alert"
          className="flex items-center justify-between gap-3 rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-4 py-3"
        >
          <p className="flex items-center gap-2 text-sm font-medium text-[var(--error-text)]">
            <AlertTriangle className="h-4 w-4 shrink-0 text-[var(--error)]" aria-hidden="true" />
            無法載入，請稍後再試
          </p>
          <button
            type="button"
            onClick={() => refetch()}
            data-testid="home-recent-retry"
            className="min-h-[44px] shrink-0 rounded-[var(--radius-md)] px-4 text-sm font-medium text-[var(--error-text)] transition-colors hover:bg-[var(--error)]/10"
          >
            重試
          </button>
        </div>
      ) : cards.length === 0 ? (
        // Sparse/empty (H5-D-v2): graceful collapse, quiet hint, no top gap.
        <p data-testid="home-recent-empty" className="text-sm text-[var(--text-muted)]">
          尚無最近新增
        </p>
      ) : (
        <div
          data-testid="home-recent-row"
          className="flex gap-3 overflow-x-auto pb-2 [scrollbar-width:thin] md:gap-4"
        >
          {cards.map((c) => (
            <div key={`${c.type}-${c.id}`} className="w-[140px] shrink-0 sm:w-[160px]">
              <PosterCardV2
                id={c.id}
                type={c.type}
                title={c.title}
                posterPath={c.posterPath}
                year={c.year}
                meta={c.meta}
                voteAverage={c.voteAverage}
                media={c.media}
              />
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
