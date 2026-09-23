import { readFileSync, readdirSync } from 'node:fs';
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
 *
 * dsr-3a: the heading is now `SettingsPageHeader`, so that is what must come
 * before the cap.
 */
const ROUTES_DIR = join(__dirname);
const COMPONENTS_DIR = join(__dirname, '../../components/settings');

const PAGES_WITH_A_CAPPED_FORM = ['connection.tsx', 'keys.tsx', 'subtitle.tsx'];

describe('J7-D — settings page headers span the layout column', () => {
  it.each(PAGES_WITH_A_CAPPED_FORM)('%s does not cap the page above its heading', (file) => {
    const source = readFileSync(join(ROUTES_DIR, file), 'utf8');

    const headingAt = source.indexOf('<SettingsPageHeader');
    expect(headingAt, `${file} should render the shared page heading`).toBeGreaterThan(-1);

    const capAt = source.indexOf('max-w-3xl');
    if (capAt !== -1) {
      expect(
        capAt,
        `${file}: max-w-3xl appears before the heading, so the heading is inside the form cap`
      ).toBeGreaterThan(headingAt);
    }
  });
});

/**
 * dsr-3a — one heading, one component. Ten routes hand-wrote the same h1 and
 * two tabs had drifted to an 18px h2, so the title changed size along the
 * strip. Every page tab now goes through SettingsPageHeader, and no route may
 * hand-write an <h1> again.
 */
describe('dsr-3a — every settings page uses the shared heading', () => {
  const routeFiles = readdirSync(ROUTES_DIR).filter(
    (f) => f.endsWith('.tsx') && !f.endsWith('.spec.tsx')
  );

  // Redirect-only routes render nothing. 效能監控 is the one exception with a
  // page: its frame (C14 GQoB1 / O5N9J) deliberately has no page heading — the
  // placeholder's own title is the only title on a page nobody can navigate to.
  // 外觀's heading lives inside AppearanceSettings because its description is
  // live state (see that file).
  const NO_HEADING = ['index.tsx', 'qbittorrent.tsx', 'performance.tsx', 'appearance.tsx'];

  it('finds the fourteen route files it is guarding', () => {
    expect(routeFiles).toHaveLength(14);
  });

  it.each(routeFiles.filter((f) => !NO_HEADING.includes(f)))(
    '%s renders SettingsPageHeader',
    (file) => {
      const source = readFileSync(join(ROUTES_DIR, file), 'utf8');
      expect(source).toContain('<SettingsPageHeader');
    }
  );

  it.each(routeFiles)('%s never hand-writes an <h1>', (file) => {
    const source = readFileSync(join(ROUTES_DIR, file), 'utf8');
    expect(source).not.toContain('<h1');
  });

  it('外觀 renders SettingsPageHeader from its component', () => {
    const source = readFileSync(join(COMPONENTS_DIR, 'AppearanceSettings.tsx'), 'utf8');
    expect(source).toContain('<SettingsPageHeader');
    expect(source).not.toContain('<h2');
  });

  it.each(routeFiles)('%s carries a Rule 21 Design ref header', (file) => {
    const source = readFileSync(join(ROUTES_DIR, file), 'utf8');
    expect(source.split('\n')[0]).toMatch(/^\/\/ Design ref: ux-design\.pen/);
  });
});
