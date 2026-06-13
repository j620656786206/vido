// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * v2 Browse list row (UX Redesign Phase 2 — UX2-2, AC #5, `b1H71g`).
 * Thumbnail + title + meta (year · runtime/seasons · genre) + tech badges
 * (accent-tint Mono) + subtitle-status pill + a row affordance. Links to detail.
 */
import { Link } from '@tanstack/react-router';
import { Star, ChevronRight } from 'lucide-react';
import { getImageUrl } from '../../lib/image';
import { deriveSubtitleStatus, deriveLifecycleStatus } from '../../utils/libraryStatus';
import type { LibraryMovie, LibrarySeries } from '../../types/library';

interface LibraryListRowV2Props {
  id: string;
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  meta: string;
  voteAverage?: number;
  media: LibraryMovie | LibrarySeries;
}

function TechPill({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-[var(--radius-sm)] bg-[var(--accent-tint)] px-1.5 py-0.5 font-mono text-[11px] text-[var(--accent-text)]">
      {children}
    </span>
  );
}

export function LibraryListRowV2({
  id,
  type,
  title,
  posterPath,
  meta,
  voteAverage,
  media,
}: LibraryListRowV2Props) {
  const img = getImageUrl(posterPath ?? null, 'w185');
  const subtitle = deriveSubtitleStatus(media);
  const lifecycle = deriveLifecycleStatus(media);
  const techs = [media.videoResolution, media.videoCodec, media.audioCodec].filter(
    Boolean
  ) as string[];

  return (
    <Link
      to="/media/$type/$id"
      params={{ type, id }}
      data-testid={`list-row-v2-${id}`}
      className="flex min-h-[44px] items-center gap-3 rounded-[var(--radius-md)] px-2 py-2 transition-colors hover:bg-[var(--bg-secondary)]"
    >
      <div className="h-[60px] w-10 shrink-0 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]">
        {img && <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <h3 className="truncate text-sm font-medium text-[var(--text-primary)]">{title}</h3>
          {lifecycle && lifecycle.label !== '已入庫' && (
            <span
              className={`shrink-0 rounded-full px-1.5 py-0.5 text-[11px] ${lifecycle.className}`}
            >
              {lifecycle.label}
            </span>
          )}
        </div>
        <p className="truncate font-mono text-[11px] text-[var(--text-secondary)]">{meta}</p>
      </div>

      <div className="hidden shrink-0 items-center gap-1.5 sm:flex">
        {techs.slice(0, 3).map((t) => (
          <TechPill key={t}>{t}</TechPill>
        ))}
        {subtitle && (
          <span
            className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${subtitle.className}`}
          >
            {subtitle.label}
          </span>
        )}
      </div>

      {typeof voteAverage === 'number' && voteAverage > 0 && (
        <span className="hidden shrink-0 items-center gap-0.5 font-mono text-[11px] text-[var(--text-secondary)] sm:flex">
          <Star className="h-3 w-3 fill-[var(--warning)] text-[var(--warning)]" />
          {voteAverage.toFixed(1)}
        </span>
      )}
      <ChevronRight className="h-4 w-4 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
    </Link>
  );
}
