/**
 * JetBrains Mono really paints the readouts (disc-2026-10-font-mono-not-jetbrains AC #3, #5).
 *
 * @tags @e2e @fonts
 *
 * jsdom resolves no CSS, so whether `font-mono` lands on the bundled face — and
 * whether the woff2 comes from Vido itself, not a CDN — is only provable in a real
 * browser. The old `.font-mono` rule in `@layer base` was dead for months because
 * nothing measured the computed style.
 */
import { test, expect } from '../support/fixtures';
import { movieItem, stubLibraryBaseline, stubLibraryList } from '../support/helpers/library-stubs';

const FILE = '[Group] Some.Movie--2024...Remux.mkv';
const ITEMS = [
  movieItem('m1', '執法記錄', 'none', {
    video_resolution: '3840x2160',
    video_codec: 'H.265',
    audio_codec: 'DTS',
  }),
  movieItem('m2', 'Chainsaw Man', 'unknown', {
    parse_status: 'failed',
    file_path: `/media/movies/${FILE}`,
  }),
];

const isLocal = (url: string) => ['localhost', '127.0.0.1'].includes(new URL(url).hostname);

test.describe('font-mono is the bundled JetBrains Mono @e2e @fonts', () => {
  test.beforeEach(async ({ page }, testInfo) => {
    // Font loading and resourceType('font') are what this measures; one engine is
    // enough to prove the bundle and the cascade (both are engine-independent).
    test.skip(testInfo.project.name !== 'chromium', 'font bundle check — chromium only');
    await page.addInitScript(() => localStorage.setItem('vido:library:view', 'list'));
    await stubLibraryBaseline(page);
    await stubLibraryList(page, () => ({ items: ITEMS }));
  });

  test('[P1] a font-mono readout computes to JetBrains Mono, loads it, ligatures off, no CDN', async ({
    page,
  }) => {
    const fontUrls: string[] = [];
    page.on('request', (req) => {
      if (req.resourceType() === 'font') fontUrls.push(req.url());
    });

    await page.goto('/library');
    await expect(page.getByTestId('list-row-v2-m2')).toBeVisible({ timeout: 15000 });
    const mono = page.getByTestId('library-list-v2').locator('.font-mono').first();
    await expect(mono).toBeVisible();

    const facts = await mono.evaluate(async (el) => {
      await document.fonts.ready;
      const cs = window.getComputedStyle(el);
      return {
        family: cs.fontFamily,
        features: cs.fontFeatureSettings,
        loaded: [...document.fonts]
          .filter((f) => f.status === 'loaded')
          .map((f) => f.family.replace(/["']/g, '')),
      };
    });

    expect(facts.family).toMatch(/^"?JetBrains Mono Variable"?,/);
    expect(facts.features).toContain('"calt" 0');
    expect(facts.features).toContain('"liga" 0');
    // A real FontFace that finished loading — not document.fonts.check(), which
    // answers true for a family that does not exist at all.
    expect(facts.loaded).toContain('JetBrains Mono Variable');

    expect(fontUrls.some((u) => u.includes('jetbrains-mono-'))).toBe(true);
    expect(fontUrls.filter((u) => !u.startsWith('data:') && !isLocal(u))).toEqual([]);
  });
});
