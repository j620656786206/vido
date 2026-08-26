// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv)
// Section: own-content zone, 最近新增 row.
// Re-anchored from the v2 frames in two steps: ux3-1-8 moved the containing
// frame to H1-D-v3, and the v3 state frames (H4/H5/H6-D-v3) replaced the v2
// state refs below once they were drawn. The five v2 homepage frames are now
// deleted from the .pen.
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
 *  - Loading  → poster-shaped skeleton row (reduced-motion aware), matches H4-D-v3 (B7UO8).
 *  - Empty    → quiet 「尚無最近新增」 hint, graceful collapse with no top gap (H5-D-v3, RvS6c).
 *  - Error    → compact inline error-tint banner + 重試 (H6-D-v3, zRyNS); the section degrades
 *               alone, Hero/Explore below still render.
 *  - Data     → horizontal scroller of PosterCardV2.
 *
 * Header carries the 進行中 · N lifecycle chip (H1-D-v3): N = items currently being
 * processed (parseStatus 'pending' = 整理中), derived from the same data the row already
 * fetches — no extra request. Hidden when N = 0 (exception-signal only, like the badge).
 *
 * Token-only colors; Noto Sans TC (CJK) + JetBrains Mono (the numeric chip). 44px touch
 * floor on the 重試 control (N5).
 */
import { AlertTriangle, ChevronLeft, ChevronRight } from 'lucide-react';
import { useRef } from 'react';
import { Link } from '@tanstack/react-router';
import { useRecentlyAdded, RECENT_LIMIT } from '../../hooks/useLibrary';
import { PosterCardV2 } from '../library/PosterCardV2';
import type { LibraryItem, LibraryMovie, LibrarySeries } from '../../types/library';

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
  const scrollerRef = useRef<HTMLDivElement | null>(null);

  const scroll = (direction: 'left' | 'right') => {
    const el = scrollerRef.current;
    if (!el) return;
    const delta = direction === 'right' ? el.clientWidth * 0.8 : -el.clientWidth * 0.8;
    el.scrollBy({ left: delta, behavior: 'smooth' });
  };
  const items = data ?? [];
  const cards = items.map(toCard).filter(Boolean) as CardFields[];
  const inProgress = countInProgress(items);

  return (
    <section data-testid="home-recently-added" aria-labelledby="home-ra-title">
      <div className="mb-3 flex items-center justify-between">
        <h2 id="home-ra-title" className="text-lg font-semibold text-[var(--text-primary)]">
          最近新增
        </h2>
        {/* 整理中 · N — exception-signal chip (hidden at 0), a door to the
            Activity hub. ⚖️ R2 ruling (2026-08-26): parseStatus=pending is
            QUEUED, not running — the same items' poster badges wear amber
            整理中, and one screen may not dress one truth in two colours
            (固定詞彙). The chip now matches the badge exactly: same word, same
            amber. (R1 had briefly ruled it green; R2's contradiction finding
            superseded that.) Count stays scoped to THIS row — /activity
            pending.parse_count measures the capped parse-job QUEUE, which
            live-diverges from item parseStatus (0 vs 3 on the seeded env). */}
        {inProgress > 0 && (
          <Link
            to="/activity"
            data-testid="home-recent-progress"
            // after:-inset-y grows the TOUCH target to ≥44px without fattening
            // the visual pill (critique R3 P3 — measured 24px, N5 floor is 44).
            className="relative flex items-center gap-1 rounded-[var(--radius-md)] bg-[var(--warning-tint)] px-2.5 py-1 text-xs font-medium text-[var(--warning-text)] transition-colors after:absolute after:-inset-x-1 after:-inset-y-2.5 after:content-[''] hover:bg-[var(--bg-tertiary)]"
          >
            整理中
            <span className="font-mono tabular-nums">· {inProgress}</span>
          </Link>
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
        // Sparse/empty (H5-D-v3): graceful collapse, quiet hint, no top gap.
        <p data-testid="home-recent-empty" className="text-sm text-[var(--text-muted)]">
          尚無最近新增
        </p>
      ) : (
        // Same scroll affordance as the explore rows (critique R2 P2: two
        // horizontal shelves on one page spoke two grammars — this one had NO
        // chevrons/scrims while a clipped first card begged for them).
        <div className="group/scroller relative">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 left-0 z-0 hidden w-14 bg-gradient-to-r from-[var(--bg-primary)] to-transparent opacity-0 transition-opacity duration-300 group-hover/scroller:opacity-100 group-focus-within/scroller:opacity-100 lg:block"
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 right-0 z-0 hidden w-14 bg-gradient-to-l from-[var(--bg-primary)] to-transparent opacity-0 transition-opacity duration-300 group-hover/scroller:opacity-100 group-focus-within/scroller:opacity-100 lg:block"
          />
          <button
            type="button"
            onClick={() => scroll('left')}
            aria-label="向左捲動"
            data-testid="home-recent-scroll-left"
            className="absolute left-0 top-1/2 z-10 hidden -translate-x-1/2 -translate-y-1/2 rounded-full bg-[var(--bg-secondary)]/95 p-2 text-[var(--text-primary)] opacity-0 shadow-lg ring-1 ring-[var(--border-subtle)]/70 backdrop-blur-sm transition-opacity duration-300 hover:bg-[var(--bg-tertiary)] focus-visible:opacity-100 group-hover/scroller:opacity-100 group-focus-within/scroller:opacity-100 lg:block"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <button
            type="button"
            onClick={() => scroll('right')}
            aria-label="向右捲動"
            data-testid="home-recent-scroll-right"
            className="absolute right-0 top-1/2 z-10 hidden translate-x-1/2 -translate-y-1/2 rounded-full bg-[var(--bg-secondary)]/95 p-2 text-[var(--text-primary)] opacity-0 shadow-lg ring-1 ring-[var(--border-subtle)]/70 backdrop-blur-sm transition-opacity duration-300 hover:bg-[var(--bg-tertiary)] focus-visible:opacity-100 group-hover/scroller:opacity-100 group-focus-within/scroller:opacity-100 lg:block"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
          <div
            ref={scrollerRef}
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
        </div>
      )}
    </section>
  );
}
