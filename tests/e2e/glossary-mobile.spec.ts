/**
 * 名詞對照表 on a phone — the nested bottom sheet, measured in a REAL browser
 * (dsr-6f-2).
 *
 * The unit specs can only pin class tokens: jsdom evaluates no media queries, so
 * `max-sm:flex-col` proves nothing about what a 390px screen draws. And the one
 * thing this story is really about — does a glossary row still FIT once 編輯 is
 * back on every row (⚖️ 2026-09-17) — is a width question with about 9px of
 * slack. Only a browser can answer it.
 *
 * An in-page visual fixture cannot answer it either: /test/gallery's own `p-8`
 * would leave it 326 wide, not the 358 of a real sheet — so the only pixel
 * evidence is the portal fixture `glossary-panel-v2/seeded-mobile` and this
 * spec, which measures BOTH sides of the 640px breakpoint
 * (feedback_measure_the_breakpoint_you_return_to).
 *
 * The seeded movie has no file on disk and 管理字幕 is gated on `file_path`, so
 * the detail GET is patched on the way through; the estimate and the glossary
 * list are stubbed because a seeded row has neither.
 *
 * @tags @e2e @glossary-mobile
 */

import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { seedMovie, deleteMovies } from '../support/helpers/seed-helpers';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };

/** snake_case on the wire — Rule 18's snakeToCamel runs in the front end. */
const TERMS = [
  {
    id: 'g1',
    media_id: 'x',
    term_src: 'Demogorgon',
    term_zh: '魔王獸',
    language: 'zh-Hant',
    source: 'subtitle',
    confirmed: true,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
  },
  {
    // The widest line 2 there is: a 4-character source badge, 未確認, and all
    // three actions.
    id: 'g2',
    media_id: 'x',
    term_src: 'Hopper',
    term_zh: '霍普',
    language: 'zh-Hant',
    source: 'metadata',
    confirmed: false,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
  },
  {
    id: 'g3',
    media_id: 'x',
    term_src: 'Vecna',
    term_zh: '維克那',
    language: 'zh-Hant',
    source: 'manual',
    confirmed: false,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
  },
];

async function stubApis(page: Page, movieId: string) {
  await page.route(`**/api/v1/movies/${movieId}`, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback();
    const response = await route.fetch();
    const body = await response.json();
    body.data = { ...body.data, file_path: '/media/movies/e2e-dsr-6f-2.mkv' };
    // status + json only: passing `response` would keep the ORIGINAL
    // content-length on a body that just grew.
    await route.fulfill({ status: response.status(), json: body });
  });
  await page.route(`**/api/v1/movies/${movieId}/transcribe/estimate`, (route) =>
    route.fulfill({
      json: {
        success: true,
        data: {
          media_id: movieId,
          media_type: 'movie',
          plan: 'full',
          asr_available: true,
          self_hosted_asr: false,
          translation_configured: true,
          model_id: 'claude-sonnet-5',
          runtime_minutes: 120,
          runtime_known: true,
          runtime_source: 'ffprobe',
          estimated_usd: 0.42,
        },
      },
    })
  );
  // Anchored glob: this does NOT swallow …/glossary/confirm-all.
  await page.route('**/api/v1/media/*/glossary', (route) =>
    route.fulfill({ json: { success: true, data: { terms: TERMS } } })
  );
}

/** Opens 管理字幕, then the nested glossary sheet, and waits out both slides. */
async function openGlossary(page: Page, movieId: string) {
  await page.goto(`/media/movie/${movieId}`);
  await page.getByTestId('action-manage-subtitle').click();
  await expect(page.getByTestId('manage-subtitle-dialog-v2')).toBeVisible();
  await page.getByTestId('open-glossary').click();

  const panel = page.getByTestId('glossary-panel-v2');
  await expect(panel).toBeVisible();
  await expect(page.getByTestId('glossary-list')).toBeVisible();
  // e2e runs with motion ON: measuring mid-slide reads the sheet below the
  // screen. allSettled + explicit undefined — a cancelled animation must not
  // reject, and Animation objects are not serialisable back to Node.
  await panel.evaluate((el) =>
    Promise.allSettled(el.getAnimations().map((a) => a.finished)).then(() => undefined)
  );
  return panel;
}

const animationName = (page: Page) =>
  page.getByTestId('glossary-panel-v2').evaluate((el) => window.getComputedStyle(el).animationName);

test.describe('名詞對照表 — phone bottom sheet @e2e @glossary-mobile', () => {
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  test('[P0] 390px: slides up flush to the bottom, 44px ✕, every row keeps 編輯 on two lines', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 手機名詞表 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(PHONE);
    await stubApis(page, movie.id);

    const panel = await openGlossary(page, movie.id);

    expect(await animationName(page)).toBe('sheet-enter');

    const box = (await panel.boundingBox())!;
    // Math.round: a computed -0 is not Object.is-equal to 0.
    expect(Math.round(box.x)).toBe(0);
    expect(Math.round(box.width)).toBe(PHONE.width);
    expect(Math.round(box.y + box.height)).toBe(PHONE.height);

    await expect(page.getByTestId('glossary-sheet-grabber')).toBeVisible();

    const close = panel.getByRole('button', { name: 'Close' });
    const closeBox = (await close.boundingBox())!;
    expect(closeBox.width).toBeGreaterThanOrEqual(44);
    expect(closeBox.height).toBeGreaterThanOrEqual(44);

    // ⚖️ 2026-09-17: 編輯 is on EVERY row, confirmed ones included. The design
    // had switched it off on all four rows to win back width.
    for (const id of ['g1', 'g2', 'g3']) {
      await expect(page.getByTestId(`glossary-edit-${id}`)).toBeVisible();
    }

    // TWO lines — and exactly two. The delete button sits BELOW the source term…
    const row = page.getByTestId('glossary-row-g2');
    const rowBox = (await row.boundingBox())!;
    const srcBox = (await row.getByText('Hopper').boundingBox())!;
    const delBox = (await page.getByTestId('glossary-delete-g2').boundingBox())!;
    expect(delBox.y).toBeGreaterThan(srcBox.y + srcBox.height - 1);

    // …and line 2 has NOT wrapped to a third: the badge and the last action are
    // on the same baseline. A wrapping flex container never reports overflow —
    // it just grows — so an upper bound on the height and this same-line check
    // are the only things that can catch the ~9px of slack running out.
    expect(rowBox.height).toBeLessThanOrEqual(96);
    const badgeBox = (await page.getByTestId('glossary-source-g2').boundingBox())!;
    expect(
      Math.abs(delBox.y + delBox.height / 2 - (badgeBox.y + badgeBox.height / 2))
    ).toBeLessThan(4);
  });

  test('[P1] 390px: the delete confirm is a third sheet, flush and with a clear ✕', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 名詞表刪除 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(PHONE);
    await stubApis(page, movie.id);

    await openGlossary(page, movie.id);
    await page.getByTestId('glossary-delete-g2').click();

    const confirm = page.getByTestId('glossary-delete-dialog-g2');
    await expect(confirm).toBeVisible();
    await confirm.evaluate((el) =>
      Promise.allSettled(el.getAnimations().map((a) => a.finished)).then(() => undefined)
    );

    // Three sheets deep: 管理字幕 → 名詞對照表 → 刪除確認. The third one still
    // has to be a sheet, not a centred box (AC #5).
    const box = (await confirm.boundingBox())!;
    expect(Math.round(box.x)).toBe(0);
    expect(Math.round(box.width)).toBe(PHONE.width);
    expect(Math.round(box.y + box.height)).toBe(PHONE.height);
    await expect(page.getByTestId('glossary-delete-sheet-grabber')).toBeVisible();

    // AC #5 asked for this to be measured, not eyeballed: the ✕ must clear the
    // title, since this dialog has no title ROW to centre against.
    const close = confirm.getByRole('button', { name: 'Close' });
    const closeBox = (await close.boundingBox())!;
    expect(closeBox.width).toBeGreaterThanOrEqual(44);
    expect(closeBox.height).toBeGreaterThanOrEqual(44);
    // Measure the TEXT, not the block: DialogTitle is a full-width element, so
    // its bounding box runs under the ✕ in every p-6 dialog in the app. What
    // matters is whether the glyphs do.
    const titleInk = await confirm.getByText('刪除詞彙').evaluate((el) => {
      const range = document.createRange();
      range.selectNodeContents(el);
      const { x, y, width, height } = range.getBoundingClientRect();
      return { x, y, width, height };
    });
    const overlaps =
      closeBox.x < titleInk.x + titleInk.width &&
      titleInk.x < closeBox.x + closeBox.width &&
      closeBox.y < titleInk.y + titleInk.height &&
      titleInk.y < closeBox.y + closeBox.height;
    expect(overlaps).toBe(false);
  });

  test('[P0] 390px: 全部確認 / 新增詞彙 are full-width and sit BELOW the list', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 手機名詞表鈕 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(PHONE);
    await stubApis(page, movie.id);

    await openGlossary(page, movie.id);

    const lastRow = (await page.getByTestId('glossary-row-g3').boundingBox())!;
    for (const id of ['glossary-confirm-all', 'glossary-add-term']) {
      const btn = (await page.getByTestId(id).boundingBox())!;
      // 390 minus the sheet's 16px gutters.
      expect(Math.round(btn.width)).toBe(PHONE.width - 32);
      expect(btn.y).toBeGreaterThan(lastRow.y + lastRow.height - 1);
    }

    // The add form stacks: two full-width inputs, one above the other.
    await page.getByTestId('glossary-add-term').click();
    const src = (await page.getByTestId('glossary-add-src').boundingBox())!;
    const zh = (await page.getByTestId('glossary-add-zh').boundingBox())!;
    expect(src.width).toBeGreaterThan(250);
    expect(Math.round(zh.width)).toBe(Math.round(src.width));
    expect(zh.y).toBeGreaterThan(src.y + src.height - 1);
  });

  test('[P0] 640px — the other side of the breakpoint — is the untouched desktop dialog', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 名詞表斷點 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(AT_BREAKPOINT);
    await stubApis(page, movie.id);

    const panel = await openGlossary(page, movie.id);

    expect(await animationName(page)).toBe('dialog-enter');
    await expect(page.getByTestId('glossary-sheet-grabber')).toBeHidden();

    const box = (await panel.boundingBox())!;
    // Centred, with air underneath — not pinned to the bottom edge.
    expect(box.y + box.height).toBeLessThan(AT_BREAKPOINT.height);

    // One line again: the delete button is level with the source term.
    const srcBox = (await page.getByTestId('glossary-row-g2').getByText('Hopper').boundingBox())!;
    const delBox = (await page.getByTestId('glossary-delete-g2').boundingBox())!;
    expect(Math.abs(delBox.y + delBox.height / 2 - (srcBox.y + srcBox.height / 2))).toBeLessThan(4);

    // …and the action buttons are not stretched.
    const btn = (await page.getByTestId('glossary-add-term').boundingBox())!;
    expect(btn.width).toBeLessThan(box.width / 2);
  });
});
