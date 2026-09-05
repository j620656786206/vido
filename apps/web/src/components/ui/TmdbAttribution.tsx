// Design ref: ux-design.pen — no current screen frame; the TMDB attribution is a compliance line that rides the frames it sits in (settings shell C4-D, detail page), ruled in party-mode 2026-09-03 as not taking a design slot
/**
 * TMDB attribution — TMDB API Terms of Use §3 (story sub-6-9).
 *
 * The terms require BOTH the TMDB logo and a verbatim non-endorsement notice
 * wherever the app uses TMDB data. JustWatch's equivalent has been shipping
 * since 12-4 (`StreamingAvailability.tsx`, commented "mandatory licensing
 * requirement"); TMDB's — the source of every poster, title and overview in
 * this app — had nothing until this story.
 *
 * Two rules this component exists to keep:
 *  - THE SENTENCE IS QUOTED, NOT WRITTEN. `TMDB_ATTRIBUTION_EN` is the exact
 *    string §3 prescribes. It is not copy, it is not ours to shorten, and it
 *    must not be "improved" by a future copy pass. The zh-TW line beside it is
 *    a reading aid FOR our user and carries no legal weight — which is why the
 *    English original stays on screen rather than being replaced by it.
 *  - THE MARK IS THE OFFICIAL FILE. `/images/tmdb-logo.svg` is TMDB's own
 *    asset, downloaded from their brand page — never a hand-drawn lookalike
 *    and never a recolour, both of which would breach the same brand terms
 *    this component exists to satisfy.
 */
import { useState } from 'react';
import { cn } from '../../lib/utils';

/**
 * The §3 sentence, verbatim. Any edit to this string is a compliance change,
 * not a copy change — check https://www.themoviedb.org/api-terms-of-use first.
 */
export const TMDB_ATTRIBUTION_EN =
  'This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.';

/** zh-TW reading aid for the sentence above. Explanatory only — never a replacement. */
export const TMDB_ATTRIBUTION_ZH =
  '本應用程式使用 TMDB 與 TMDB API，但未經 TMDB 認可、認證或核准。';

export const TMDB_LOGO_SRC = '/images/tmdb-logo.svg';
export const TMDB_HOME_URL = 'https://www.themoviedb.org/';

/**
 * The mark, with the codebase's never-a-broken-image rule applied (the same
 * idiom `ProviderLogo` uses one file over): if the SVG cannot be loaded, fall
 * back to a plain "TMDB" wordmark so the attribution still reads correctly and
 * the link keeps its accessible name. The fallback is load-bearing today —
 * the official asset is dropped in by hand (see `tmdb-logo-asset.spec.ts`).
 */
function TmdbLogo({ className }: { className?: string }) {
  const [failed, setFailed] = useState(false);

  if (failed) {
    return (
      <span data-testid="tmdb-logo-fallback" className="text-xs font-semibold tracking-wide">
        TMDB
      </span>
    );
  }

  return (
    <img
      src={TMDB_LOGO_SRC}
      alt="TMDB"
      onError={() => setFailed(true)}
      className={cn('w-auto', className)}
    />
  );
}

export interface TmdbAttributionProps {
  /**
   * `full` — logo + the §3 sentence + its zh-TW gloss. For places that account
   * for the data source in prose (the API-keys settings row).
   * `inline` — one muted 資料來源 line, sized to sit next to JustWatch's.
   */
  variant?: 'full' | 'inline';
  className?: string;
}

export function TmdbAttribution({ variant = 'full', className }: TmdbAttributionProps) {
  const logoLink = (
    <a
      href={TMDB_HOME_URL}
      target="_blank"
      rel="noopener noreferrer"
      // Focus is the global :focus-visible ring (styles.css §Focus ring) — a
      // real <a href> is focusable, so no per-link ring is needed here.
      className="inline-flex items-center"
      data-testid="tmdb-attribution-link"
    >
      <TmdbLogo className={variant === 'full' ? 'h-4' : 'h-3'} />
    </a>
  );

  if (variant === 'inline') {
    return (
      <p
        data-testid="tmdb-attribution"
        className={cn('flex items-center gap-1 text-xs text-[var(--text-muted)]', className)}
      >
        資料來源：
        {logoLink}
      </p>
    );
  }

  return (
    <div
      data-testid="tmdb-attribution"
      className={cn('flex flex-col gap-1.5 text-xs text-[var(--text-muted)]', className)}
    >
      {logoLink}
      <p lang="en">{TMDB_ATTRIBUTION_EN}</p>
      <p>{TMDB_ATTRIBUTION_ZH}</p>
    </div>
  );
}
