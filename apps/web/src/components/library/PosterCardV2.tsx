// Implements: Component/PosterCard-v2 (hD7Tw)
/**
 * v2 poster card (UX Redesign Phase 2 — UX2-2, §3.3 / §5.1 / N1).
 *
 * Tighter than the legacy card: a 2-line CJK title grid (reserves two full lines,
 * truncates with ellipsis, never clips mid-glyph — R2 fix), JetBrains Mono
 * year·meta in `text-secondary`, and ONE lifecycle/subtitle status badge on the
 * poster (§2.5). The badge is an EXCEPTION signal (ux3-0-2): a lifecycle exception
 * (整理中/失敗) wins, else a subtitle exception (缺字幕/簡中/有字幕); the happy steady
 * state (已入庫 + 繁中) and unknown states show NO badge (never errors — F3). The
 * subtitle source prefers the authoritative engine result (subtitleStatus/
 * subtitleLanguage, ux3-0-1) over embedded tracks. Links to the detail route.
 */
import { Link } from '@tanstack/react-router';
import { Star } from 'lucide-react';
import { getImageUrl } from '../../lib/image';
import { filenameToGradient } from '../media/ColorPlaceholder';
import { pickPosterBadge } from '../../utils/libraryStatus';
import type { LibraryMovie, LibrarySeries } from '../../types/library';

interface PosterCardV2Props {
  id: string;
  /** Detail-route media type (series is mapped to `tv` by the caller). */
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  /** Release year (already formatted). */
  year?: string;
  /** Right-hand meta — runtime ("142 分") or season count ("3 季"). */
  meta?: string;
  voteAverage?: number;
  media: Pick<
    LibraryMovie | LibrarySeries,
    'parseStatus' | 'subtitleTracks' | 'subtitleStatus' | 'subtitleLanguage'
  >;
}

export function PosterCardV2({
  id,
  type,
  title,
  posterPath,
  year,
  meta,
  voteAverage,
  media,
}: PosterCardV2Props) {
  const badge = pickPosterBadge(media);
  const img = getImageUrl(posterPath ?? null, 'w342');
  const [from, to] = filenameToGradient(title);
  const metaLine = [year, meta].filter(Boolean).join(' · ');

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id }}
      data-testid={`poster-v2-${id}`}
      className="group/card flex flex-col gap-2"
    >
      <div className="relative aspect-[2/3] overflow-hidden rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] shadow-[var(--shadow-md)] transition-transform duration-200 group-hover/card:scale-[1.02] group-focus-visible/card:scale-[1.02]">
        {img ? (
          <img src={img} alt={title} loading="lazy" className="h-full w-full object-cover" />
        ) : (
          <div
            className="flex h-full w-full items-center justify-center text-3xl font-bold text-[var(--text-on-accent)]"
            style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
            aria-hidden="true"
          >
            {title.slice(0, 1)}
          </div>
        )}

        {badge && (
          <span
            data-testid="poster-status-badge"
            className={`absolute right-1.5 top-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium ${badge.className}`}
          >
            {badge.label}
          </span>
        )}

        {typeof voteAverage === 'number' && voteAverage > 0 && (
          <span className="absolute bottom-1.5 left-1.5 flex items-center gap-0.5 rounded-full bg-[var(--overlay-scrim)] px-1.5 py-0.5 font-mono text-[11px] text-[var(--text-on-accent)]">
            <Star className="h-3 w-3 fill-[var(--warning)] text-[var(--warning)]" />
            {voteAverage.toFixed(1)}
          </span>
        )}
      </div>

      {/* 2-line CJK title grid (§3.3) — reserves two lines, ellipsis on overflow */}
      <div>
        <h3
          className="line-clamp-2 min-h-[2.75em] text-sm font-medium leading-snug text-[var(--text-primary)] transition-colors group-hover/card:text-[var(--accent-text)]"
          title={title}
        >
          {title}
        </h3>
        {metaLine && (
          <p className="mt-0.5 truncate font-mono text-[11px] text-[var(--text-secondary)]">
            {metaLine}
          </p>
        )}
      </div>
    </Link>
  );
}
