// Implements: Component/DetailHero-v2 (uRGu2)
/**
 * v2 detail backdrop hero (UX Redesign Phase 2 — UX2-3, AC #1, `uRGu2`).
 * Replaces the cramped narrow-panel IA (brief hotspot #2) with a full-page
 * backdrop hero: backdrop image + bottom-gradient scrim (AA over the backdrop),
 * a back affordance, a poster thumbnail, and an info block — status badge → title
 * (H1) → original/EN title → meta row → action row. Shorter backdrop on mobile.
 */
import { ArrowLeft } from 'lucide-react';
import { getImageUrl } from '../../lib/image';
import { filenameToGradient } from './ColorPlaceholder';
import type { StatusDescriptor } from '../../utils/libraryStatus';

interface DetailHeroV2Props {
  backdropPath?: string | null;
  posterPath?: string | null;
  title: string;
  originalTitle?: string;
  /** N1 status badges (lifecycle + subtitle); nulls are skipped (F3). */
  badges?: (StatusDescriptor | null | undefined)[];
  /** Meta row — year · runtime/seasons · genre · ★rating. */
  meta?: React.ReactNode;
  /** Action row — primary/secondary CTAs (resolved per Rule 24). */
  actions?: React.ReactNode;
  onBack: () => void;
}

export function DetailHeroV2({
  backdropPath,
  posterPath,
  title,
  originalTitle,
  badges,
  meta,
  actions,
  onBack,
}: DetailHeroV2Props) {
  const backdrop = getImageUrl(backdropPath ?? null, 'w780');
  const poster = getImageUrl(posterPath ?? null, 'w342');
  const [from, to] = filenameToGradient(title);
  const shownBadges = (badges ?? []).filter(Boolean) as StatusDescriptor[];

  return (
    <section className="relative" data-testid="detail-hero-v2">
      {/* Backdrop + scrim */}
      <div className="absolute inset-x-0 top-0 h-[300px] overflow-hidden sm:h-[420px]">
        {backdrop ? (
          <img src={backdrop} alt="" className="h-full w-full object-cover" />
        ) : (
          <div
            className="h-full w-full"
            style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
          />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-[var(--bg-primary)] via-[var(--bg-primary)]/70 to-transparent" />
      </div>

      {/* Back affordance */}
      <button
        type="button"
        onClick={onBack}
        aria-label="返回媒體庫"
        data-testid="detail-back"
        className="absolute left-4 top-4 z-10 flex h-11 w-11 items-center justify-center rounded-full bg-[var(--overlay-scrim)] text-[var(--text-on-accent)] backdrop-blur-sm transition-colors hover:bg-[var(--bg-tertiary)]"
      >
        <ArrowLeft className="h-5 w-5" aria-hidden="true" />
      </button>

      {/* Info block, overlapping the bottom of the backdrop */}
      <div className="relative px-4 pt-[180px] sm:px-8 sm:pt-[260px]">
        <div className="flex gap-4 sm:gap-6">
          <div className="aspect-[2/3] w-24 shrink-0 overflow-hidden rounded-[var(--radius-lg)] shadow-[var(--shadow-xl)] sm:w-40">
            {poster ? (
              <img src={poster} alt={title} className="h-full w-full object-cover" />
            ) : (
              <div
                className="flex h-full w-full items-center justify-center text-3xl font-bold text-[var(--text-on-accent)]"
                style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
                aria-hidden="true"
              >
                {title.slice(0, 1)}
              </div>
            )}
          </div>

          <div className="min-w-0 flex-1 pb-1">
            {shownBadges.length > 0 && (
              <div className="mb-2 flex flex-wrap gap-1.5">
                {shownBadges.map((b) => (
                  <span
                    key={b.label}
                    data-testid="detail-status-badge"
                    className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${b.className}`}
                  >
                    {b.label}
                  </span>
                ))}
              </div>
            )}
            <h1 className="text-2xl font-bold leading-tight text-[var(--text-primary)] sm:text-3xl">
              {title}
            </h1>
            {originalTitle && originalTitle !== title && (
              <p className="mt-1 text-sm text-[var(--text-secondary)] sm:text-base">
                {originalTitle}
              </p>
            )}
            {meta && (
              <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-[var(--text-secondary)]">
                {meta}
              </div>
            )}
            {actions && <div className="mt-4 flex flex-wrap items-center gap-2">{actions}</div>}
          </div>
        </div>
      </div>
    </section>
  );
}
