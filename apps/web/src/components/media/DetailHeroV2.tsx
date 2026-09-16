// Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen B4p-D (N2fmG6) + Screen B3p-M (SzNRb) + Screen B9-D (Tn4Gz)
/**
 * v2 detail backdrop hero (UX Redesign Phase 2 — UX2-3, AC #1, `uRGu2`).
 * Replaces the cramped narrow-panel IA (brief hotspot #2) with a full-page
 * backdrop hero: backdrop image + bottom-gradient scrim (AA over the backdrop),
 * a back affordance, a poster thumbnail, and an info block — status badge → title
 * (H1) → original/EN title → meta row → action row. Shorter backdrop on mobile.
 */
import { useState } from 'react';
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
  // dsr-2 AC #7: a TMDb image that fails to load (404, NAS offline from the CDN)
  // falls back to the same gradient as a null path — never a broken <img>. The
  // flag stores the URL that FAILED, not a boolean: when the next title is already
  // cached (e.g. back navigation, or a recommendation opened before), React reuses this
  // instance without a skeleton in between, and a sticky boolean would hide its art.
  const [failedBackdrop, setFailedBackdrop] = useState<string | null>(null);
  const [failedPoster, setFailedPoster] = useState<string | null>(null);
  const shownBadges = (badges ?? []).filter(Boolean) as StatusDescriptor[];

  return (
    <section className="relative" data-testid="detail-hero-v2">
      {/* Backdrop + scrim */}
      <div className="absolute inset-x-0 top-0 h-[300px] overflow-hidden sm:h-[420px]">
        {backdrop && backdrop !== failedBackdrop ? (
          <img
            src={backdrop}
            alt=""
            onError={() => setFailedBackdrop(backdrop)}
            className="h-full w-full object-cover"
          />
        ) : (
          <div
            data-testid="detail-backdrop-fallback"
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
        className="absolute left-4 top-4 z-10 flex h-11 w-11 items-center justify-center rounded-full bg-[var(--overlay-scrim)] text-[var(--text-on-scrim)] backdrop-blur-sm transition-colors hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]"
      >
        <ArrowLeft className="h-5 w-5" aria-hidden="true" />
      </button>

      {/* Info block, overlapping the bottom of the backdrop */}
      <div className="relative px-4 pt-[180px] sm:px-8 sm:pt-[260px]">
        <div className="flex gap-4 sm:gap-6">
          {/* No shadow (dsr-2 AC #6): shadows belong to floating layers only (dsr-9),
              and this tile sits in the page flow. */}
          <div
            data-testid="detail-poster-tile"
            className="aspect-[2/3] w-24 shrink-0 overflow-hidden rounded-[var(--radius-lg)] sm:w-40"
          >
            {poster && poster !== failedPoster ? (
              <img
                src={poster}
                alt={title}
                onError={() => setFailedPoster(poster)}
                className="h-full w-full object-cover"
              />
            ) : (
              <div
                data-testid="detail-poster-fallback"
                // --text-on-scrim, not --text-on-accent: this tile is the same
                // hash gradient the poster cards use — clamped DARK — so the
                // near-black --text-on-accent (#14161a) was low-contrast on it
                // in BOTH themes, not just 日巡. Same fix, same reason.
                className="flex h-full w-full items-center justify-center text-3xl font-bold text-[var(--text-on-scrim)]"
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
