/**
 * Noto Sans TC really paints the body text (disc-2026-10-font-sans-not-noto AC #5, #13).
 *
 * @tags @e2e @fonts
 *
 * The old `.font-primary` class was used by nothing, so Noto never drew a glyph
 * and every machine showed its own system face. This proves, in a real browser,
 * that body text computes to the bundled face, that the face actually loads, that
 * every font byte comes from Vido itself (an offline NAS has no CDN), and that a
 * Chinese library page stays inside the per-page download budget.
 */
import { test, expect } from '../support/fixtures';
import { movieItem, stubLibraryBaseline, stubLibraryList } from '../support/helpers/library-stubs';

const TITLES = [
  '執法記錄',
  '鏢人：風起大漠',
  '牧師的最後誘惑',
  '阿凡達：火與燼',
  '你的名字',
  '葬送的芙莉蓮',
  '神隱少女',
  '霍爾的移動城堡',
];
const ITEMS = TITLES.map((t, i) => movieItem(`s${i + 1}`, t, 'zh_hant'));

/** One fresh page load of a zh-TW library must not pull more Noto than this. */
const NOTO_BUDGET_BYTES = 2_000_000;

const isLocal = (url: string) => ['localhost', '127.0.0.1'].includes(new URL(url).hostname);

test.describe('body text is the bundled Noto Sans TC @e2e @fonts', () => {
  test.beforeEach(async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'chromium', 'font bundle check — chromium only');
    await stubLibraryBaseline(page);
    await stubLibraryList(page, () => ({ items: ITEMS }));
  });

  test('[P1] body computes to Noto Sans TC, it loads from Vido, under the page budget', async ({
    page,
  }) => {
    const fonts: { url: string; status: number; type: string; head: string; bytes: number }[] = [];
    const pending: Promise<void>[] = [];
    page.on('response', (res) => {
      if (res.request().resourceType() !== 'font' || res.url().startsWith('data:')) return;
      const meta = {
        url: res.url(),
        status: res.status(),
        type: res.headers()['content-type'] ?? '',
      };
      pending.push(
        res
          .body()
          .then(
            (b) =>
              void fonts.push({
                ...meta,
                head: b.subarray(0, 4).toString('latin1'),
                bytes: b.length,
              })
          )
          .catch(() => void fonts.push({ ...meta, head: '', bytes: -1 }))
      );
    });

    await page.goto('/library');
    await expect(page.getByText('葬送的芙莉蓮').first()).toBeVisible({ timeout: 15000 });

    const facts = await page.evaluate(async () => {
      await document.fonts.ready;
      return {
        lang: document.documentElement.lang,
        body: window.getComputedStyle(document.body).fontFamily,
        loadedNoto: [...document.fonts].filter(
          (f) => f.family.replace(/["']/g, '') === 'Noto Sans TC Variable' && f.status === 'loaded'
        ).length,
        errored: [...document.fonts].filter((f) => f.status === 'error').map((f) => f.family),
      };
    });
    await Promise.all(pending);

    expect(facts.lang).toBe('zh-Hant-TW');
    expect(facts.body).toMatch(/^"?Noto Sans TC Variable"?,/);
    // A loaded FontFace, not document.fonts.check() — check() can answer true for
    // a family that does not exist, which would make this pass hollow.
    expect(facts.loadedNoto).toBeGreaterThanOrEqual(1);
    expect(facts.errored).toEqual([]);

    const remote = fonts.filter((f) => !isLocal(f.url));
    expect(remote).toEqual([]);
    const noto = fonts.filter((f) => f.url.includes('noto-sans-tc-'));
    expect(noto.length).toBeGreaterThanOrEqual(1);
    const notoBytes = noto.reduce((sum, f) => sum + f.bytes, 0);
    console.log(`[font-sans] /library first load: ${noto.length} Noto files, ${notoBytes} bytes`);
    // A static server's SPA fallback answers a missing woff2 with 200 + index.html,
    // which would still have bytes > 0; only a real woff2 starts with "wOF2".
    for (const f of noto) {
      expect(f, f.url).toMatchObject({ status: 200, type: 'font/woff2', head: 'wOF2' });
    }
    expect(notoBytes).toBeLessThan(NOTO_BUDGET_BYTES);
  });
});
