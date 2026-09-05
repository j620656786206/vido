import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';

/**
 * TMDB logo asset gate (story sub-6-9).
 *
 * `TmdbAttribution` points at `/images/tmdb-logo.svg`, and TMDB's terms are
 * satisfied by THEIR mark — not by a lookalike, not by a third-party recolour.
 * The file therefore cannot be generated from inside this repo; it has to be
 * downloaded from TMDB's brand page and committed by hand.
 *
 * ⚠️ THE FILE IS NOT HERE YET. Ruled 2026-09-05 (Alexyu): sub-6-9 ships the
 * §3 notice now and the mark lands later — tracked as `backlog-tmdb-logo-asset`
 * in sprint-status.yaml, which also carries the constraint that sub-7-7
 * (bundled TMDb key) must not ship before it. Until then the attribution
 * renders its text-wordmark fallback.
 *
 * So this spec asserts what is TRUE TODAY rather than pretending otherwise:
 * IF the asset is present, it must be a real SVG carrying its provenance
 * (AC #1 requires the source URL and download date in the file header). That
 * assertion is what actually protects the compliance property — a file dropped
 * in later without provenance is indistinguishable from a lookalike someone
 * exported from a design tool. The "does it exist at all" half is the backlog
 * entry's job, because a red suite that everyone learns to ignore protects
 * nothing.
 *
 * TO CLOSE IT OUT:
 *   1. https://www.themoviedb.org/about/logos-attribution → primary short logo (SVG)
 *   2. save as apps/web/public/images/tmdb-logo.svg
 *   3. add a comment at the top of the file recording the source URL and the
 *      download date, e.g.
 *      <!-- TMDB primary short logo — https://www.themoviedb.org/about/logos-attribution — downloaded YYYY-MM-DD -->
 *   4. delete the backlog entry, and add the gallery fixture sub-6-9 deferred.
 */
const LOGO_PATH = join(__dirname, '..', '..', '..', 'public', 'images', 'tmdb-logo.svg');

const HOW_TO_FIX =
  'Download the primary short logo (SVG) from https://www.themoviedb.org/about/logos-attribution, ' +
  'save it as apps/web/public/images/tmdb-logo.svg, and record the source URL + download date in a ' +
  'comment at the top of the file. See the docstring in this spec.';

describe('TMDB logo asset', () => {
  it('records where it came from and when, once it is committed', () => {
    if (!existsSync(LOGO_PATH)) {
      // Not yet dropped in — see the docstring and `backlog-tmdb-logo-asset`.
      // The component renders its text-wordmark fallback in this state, which
      // TmdbAttribution.spec.tsx covers directly.
      return;
    }
    const svg = readFileSync(LOGO_PATH, 'utf8');

    expect(svg).toContain('<svg');
    expect(
      svg.includes('themoviedb.org'),
      `TMDB logo has no source URL in its header. ${HOW_TO_FIX}`
    ).toBe(true);
    // An ISO-ish date in the file header — provenance is a date, not "recently".
    expect(
      /\d{4}-\d{2}-\d{2}/.test(svg.slice(0, 500)),
      `TMDB logo has no download date in its header. ${HOW_TO_FIX}`
    ).toBe(true);
  });
});
