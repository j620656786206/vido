import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';

/**
 * J7-D structural guard.
 *
 * The ruling caps settings FORM CARDS at 768px (`max-w-3xl`) for field
 * scannability. It was applied to the whole page instead, which narrowed the
 * `<h1>` too — so the page heading jumped 160px as you moved between settings
 * tabs (form pages 768px, data pages 928px) and a deliberate rule read as a bug.
 *
 * Invariant: on every settings page the heading is OUTSIDE the width cap. This
 * is a source guard rather than a DOM test because the regression is a wrapper
 * that renders identically in isolation — you only see it against a sibling tab.
 */
const ROUTES_DIR = join(__dirname);

const PAGES_WITH_A_CAPPED_FORM = ['connection.tsx', 'keys.tsx'];

describe('J7-D — settings page headers span the layout column', () => {
  it.each(PAGES_WITH_A_CAPPED_FORM)('%s does not cap the page above its <h1>', (file) => {
    const source = readFileSync(join(ROUTES_DIR, file), 'utf8');

    const headingAt = source.indexOf('<h1');
    expect(headingAt, `${file} should render a page heading`).toBeGreaterThan(-1);

    const capAt = source.indexOf('max-w-3xl');
    if (capAt !== -1) {
      expect(
        capAt,
        `${file}: max-w-3xl appears before the <h1>, so the heading is inside the form cap`
      ).toBeGreaterThan(headingAt);
    }
  });
});
