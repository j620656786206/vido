import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';

/**
 * TMDB logo asset gate (story sub-6-9).
 *
 * `TmdbAttribution` points at `/images/tmdb-logo.svg`, and TMDB's terms are
 * satisfied by THEIR mark — not by a lookalike, not by a third-party recolour.
 * The file therefore cannot be generated from inside this repo; someone has to
 * download it from TMDB's brand page and commit it.
 *
 * This spec is that hand-off, written as a gate rather than a TODO comment:
 * while the file is missing the attribution renders its text-wordmark fallback
 * (honest, but not the mark the terms ask for), and a TODO in a component
 * header is exactly the kind of note that survives to production. It goes
 * green the moment the real asset lands, and nothing else needs to change.
 *
 * TO SATISFY IT:
 *   1. https://www.themoviedb.org/about/logos-attribution → primary short logo (SVG)
 *   2. save as apps/web/public/images/tmdb-logo.svg
 *   3. add a comment at the top of the file recording the source URL and the
 *      download date, e.g.
 *      <!-- TMDB primary short logo — https://www.themoviedb.org/about/logos-attribution — downloaded YYYY-MM-DD -->
 */
const LOGO_PATH = join(__dirname, '..', '..', '..', 'public', 'images', 'tmdb-logo.svg');

const HOW_TO_FIX =
  'Download the primary short logo (SVG) from https://www.themoviedb.org/about/logos-attribution, ' +
  'save it as apps/web/public/images/tmdb-logo.svg, and record the source URL + download date in a ' +
  'comment at the top of the file. See the docstring in this spec.';

describe('TMDB logo asset', () => {
  it('is committed at apps/web/public/images/tmdb-logo.svg', () => {
    expect(existsSync(LOGO_PATH), `TMDB logo missing. ${HOW_TO_FIX}`).toBe(true);
  });

  it('is an SVG that records where it came from and when', () => {
    if (!existsSync(LOGO_PATH)) {
      // The first test already reports the missing file; failing twice for one
      // cause just makes the run harder to read.
      return;
    }
    const svg = readFileSync(LOGO_PATH, 'utf8');

    expect(svg).toContain('<svg');
    expect(
      svg.includes('themoviedb.org'),
      `TMDB logo has no source URL in its header. ${HOW_TO_FIX}`
    ).toBe(true);
    // An ISO-ish date anywhere in the file header — provenance is a date, not "recently".
    expect(
      /\d{4}-\d{2}-\d{2}/.test(svg.slice(0, 500)),
      `TMDB logo has no download date in its header. ${HOW_TO_FIX}`
    ).toBe(true);
  });
});
