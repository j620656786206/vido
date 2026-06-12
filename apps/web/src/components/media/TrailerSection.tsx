// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page trailer section postdates the .pen design
import type { ReactNode } from 'react';
import { useMediaVideos } from '../../hooks/useMediaDetails';
import { pickBestTrailer, pickTmdbVideoFallbackUrl } from '../../lib/trailers';
import { TrailerEmbed } from './TrailerEmbed';

export interface TrailerSectionProps {
  tmdbId: number;
  type: 'movie' | 'tv';
  title: string;
}

// Single source of truth for the heading id so the <section> landmark is labelled
// by its visible <h2> (aria-labelledby) — mirrors StreamingAvailability (12-4).
const HEADING_ID = 'trailer-section-heading';

// Shared landmark shell so the embed and fallback branches can't drift on
// heading / aria / spacing (DRY — single section wrapper).
function SectionShell({ children }: { children: ReactNode }) {
  return (
    <section
      aria-labelledby={HEADING_ID}
      className="flex flex-col gap-3"
      data-testid="trailer-section"
    >
      <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
        預告片
      </h2>
      {children}
    </section>
  );
}

/**
 * TrailerSection renders the "預告片" section on a media detail page (Story 12-5).
 * It self-fetches the title's TMDB videos and applies the ADR Decision 4 fallback
 * chain: an embeddable YouTube trailer → the inline youtube-nocookie embed
 * (reuses TrailerEmbed as-is); videos exist but none is a YouTube Trailer →
 * outbound link to the TMDB videos page; nothing / loading / error → render
 * nothing. Fail-soft (Rule 27 Pillar 3): it NEVER throws or breaks the page.
 *
 * Keyed by the TMDB numeric id, so it renders in BOTH the local-library and
 * TMDB-numeric detail views (AC #7) — each passes the id present in its view.
 */
export function TrailerSection({ tmdbId, type, title }: TrailerSectionProps) {
  const { data, isError } = useMediaVideos(tmdbId, type, true);

  // Fail-soft: loading (data undefined) / error / no data all render nothing
  // (AC #4). The section simply appears once embeddable data arrives — no
  // skeleton flash (Task 3.3).
  if (isError || !data) return null;

  const best = pickBestTrailer(data.results);

  // AC #1/#5 — an embeddable YouTube trailer exists: reuse the inline
  // button→iframe embed (youtube-nocookie, key-regex guarded inside TrailerEmbed);
  // autoPlay so activation plays immediately (AC #5).
  if (best) {
    return (
      <SectionShell>
        <TrailerEmbed videoKey={best.key} title={title} autoPlay />
      </SectionShell>
    );
  }

  // AC #3 — videos exist but none is an embeddable YouTube trailer: link out to
  // the TMDB videos page instead of an embed (ADR Decision 4 fallback chain).
  // `results?.length` keeps the section fail-soft (AC #4) even on a malformed body.
  if (data.results?.length) {
    return (
      <SectionShell>
        <a
          href={pickTmdbVideoFallbackUrl(tmdbId, type)}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm font-medium text-[var(--accent-primary)] hover:underline"
          data-testid="trailer-section-fallback-link"
        >
          在 TMDB 觀看預告片
        </a>
      </SectionShell>
    );
  }

  // AC #4 — no videos at all: quiet omit (render nothing). Never an error.
  return null;
}
