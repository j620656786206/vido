import type { Video } from '../types/tmdb';

// Shared across the homepage TrailerModal (Story 10-2) and the detail-page
// TrailerSection (Story 12-5). Single source of truth for trailer selection so
// the two surfaces never drift on the "best trailer" rule (Rule 21 / AC-drift).

const VALID_VIDEO_KEY = /^[a-zA-Z0-9_-]+$/;

// Story 10-2 AC #6 — picks the best YouTube trailer (official → newest).
// Returns null when no embeddable trailer exists (drives the AC #3 fallback on
// the detail page, and the empty-state on the homepage modal).
export function pickBestTrailer(results: Video[] | undefined): Video | null {
  if (!results || results.length === 0) return null;

  const youtubeTrailers = results.filter(
    (v) => v.site === 'YouTube' && v.type === 'Trailer' && VALID_VIDEO_KEY.test(v.key)
  );
  if (youtubeTrailers.length === 0) return null;

  // Prefer official; among the same officiality, prefer the most recent.
  return [...youtubeTrailers].sort((a, b) => {
    if (a.official !== b.official) return a.official ? -1 : 1;
    return (b.publishedAt || '').localeCompare(a.publishedAt || '');
  })[0];
}

// Story 12-5 AC #3 — when TMDB has videos but no embeddable YouTube trailer,
// link out to the title's TMDB videos tab (not just the main page) so the link
// text "在 TMDB 觀看預告片" lands the user where the videos actually are.
export function pickTmdbVideoFallbackUrl(tmdbId: number, type: 'movie' | 'tv'): string {
  return `https://www.themoviedb.org/${type === 'tv' ? 'tv' : 'movie'}/${tmdbId}/videos`;
}
