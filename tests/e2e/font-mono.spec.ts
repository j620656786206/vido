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
    const fonts: { url: string; status: number; type: string; head: string }[] = [];
    const pending: Promise<void>[] = [];
    page.on('response', (res) => {
      if (res.request().resourceType() !== 'font' || res.url().startsWith('data:')) return;
      pending.push(
        res
          .body()
          .then((b) => b.subarray(0, 4).toString('latin1'))
          .catch(() => '')
          .then((head) => {
            fonts.push({
              url: res.url(),
              status: res.status(),
              type: res.headers()['content-type'] ?? '',
              head,
            });
          })
      );
    });

    await page.goto('/library');
    await expect(page.getByTestId('list-row-v2-m2')).toBeVisible({ timeout: 15000 });
    // The failed row's file name: the text where `--` / `...` must not ligate.
    const fileName = page.getByTestId('list-row-filename');
    await expect(fileName).toContainText('--2024...');

    const facts = await fileName.evaluate(async (el) => {
      await document.fonts.ready;
      const cs = window.getComputedStyle(el);
      const faces = [...document.fonts];
      return {
        family: cs.fontFamily,
        features: cs.fontFeatureSettings,
        loaded: faces
          .filter((f) => f.status === 'loaded')
          .map((f) => f.family.replace(/["']/g, '')),
        errored: faces.filter((f) => f.status === 'error').map((f) => f.family),
      };
    });
    await Promise.all(pending);

    expect(facts.family).toMatch(/^"?JetBrains Mono Variable"?,/);
    expect(facts.features).toContain('"calt" 0');
    expect(facts.features).toContain('"liga" 0');
    // A real FontFace that finished loading — not document.fonts.check(), which
    // answers true for a family that does not exist at all.
    expect(facts.loaded).toContain('JetBrains Mono Variable');
    expect(facts.errored).toEqual([]);

    expect(fonts.filter((f) => !isLocal(f.url))).toEqual([]);
    const jb = fonts.filter((f) => f.url.includes('jetbrains-mono-'));
    expect(jb.length).toBeGreaterThanOrEqual(1);
    // A static server's SPA fallback answers a missing woff2 with 200 + index.html;
    // only a real woff2 starts with the "wOF2" signature.
    for (const f of jb) {
      expect(f, f.url).toMatchObject({ status: 200, type: 'font/woff2', head: 'wOF2' });
    }
  });
});
