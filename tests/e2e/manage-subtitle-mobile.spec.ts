/**
 * 管理字幕 on a phone — the bottom sheet, measured in a REAL browser (dsr-6f-1).
 *
 * The unit specs can only pin class tokens: jsdom evaluates no media queries, so
 * `max-sm:w-full` proves nothing about what a 390px screen draws. And the dialog's
 * generating / failed views cannot be visual fixtures at all — they are internal
 * state fed by a POST and an SSE stream. So this spec is the layer that measures:
 *
 *   · BOTH sides of the 640px breakpoint (feedback_measure_the_breakpoint_you_return_to):
 *     below it the dialog slides up (`sheet-enter`) and has no footer; from it the
 *     dialog is the centred desktop one (`dialog-enter`) with its footer back.
 *   · the generating view, by stubbing the trigger POST and the SSE stream.
 *
 * The seeded movie has no file on disk, and 管理字幕 is gated on `file_path`, so
 * the detail GET is patched on the way through. The estimate is stubbed because
 * a seeded row has nothing for ffprobe to measure.
 *
 * @tags @e2e @manage-subtitle-mobile
 */

import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { seedMovie, deleteMovies } from '../support/helpers/seed-helpers';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };

async function stubMovie(page: Page, movieId: string) {
  await page.route(`**/api/v1/movies/${movieId}`, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback();
    const response = await route.fetch();
    const body = await response.json();
    body.data = { ...body.data, file_path: '/media/movies/e2e-dsr-6f-1.mkv' };
    // status + json only: passing `response` would keep the ORIGINAL content-length
    // on a body that just grew.
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
}

async function openDialog(page: Page, movieId: string) {
  await page.goto(`/media/movie/${movieId}`);
  await page.getByTestId('action-manage-subtitle').click();
  const sheet = page.getByTestId('manage-subtitle-dialog-v2');
  await expect(sheet).toBeVisible();
  // e2e runs with motion ON: measuring mid-slide reads the sheet 200px below the
  // screen. Wait for the enter animation to land before any geometry.
  // allSettled + an explicit undefined: a cancelled animation must not reject the
  // wait, and Animation objects are not something to serialise back to Node.
  await sheet.evaluate((el) =>
    Promise.allSettled(el.getAnimations().map((a) => a.finished)).then(() => undefined)
  );
  return sheet;
}

const animationName = (page: Page) =>
  page
    .getByTestId('manage-subtitle-dialog-v2')
    .evaluate((el) => window.getComputedStyle(el).animationName);

test.describe('管理字幕 — phone bottom sheet @e2e @manage-subtitle-mobile', () => {
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  test('[P0] 390px: slides up, no footer, 44px ✕, full-width 生成字幕, search toggle in the body', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 手機字幕 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(PHONE);
    await stubMovie(page, movie.id);

    const sheet = await openDialog(page, movie.id);

    expect(await animationName(page)).toBe('sheet-enter');

    const box = (await sheet.boundingBox())!;
    expect(Math.round(box.x)).toBe(0); // Math.round: `toBe` is Object.is, and -0 is not 0
    expect(box.width).toBe(PHONE.width);
    expect(Math.round(box.y + box.height)).toBe(PHONE.height); // pinned to the bottom edge

    await expect(page.getByTestId('manage-sheet-grabber')).toBeVisible();

    // No footer on the idle phone sheet: 關閉 is the ✕, and it must be a real target.
    await expect(page.getByTestId('dialog-close')).toBeHidden();
    const close = (await sheet.getByRole('button', { name: 'Close' }).boundingBox())!;
    expect(close.width).toBeGreaterThanOrEqual(44);
    expect(close.height).toBeGreaterThanOrEqual(44);

    const cta = page.getByTestId('action-generate-subtitle');
    await expect(cta).toContainText('$0.42');
    const ctaBox = (await cta.boundingBox())!;
    expect(Math.round(ctaBox.width)).toBe(PHONE.width - 32); // 16px gutters

    // The toggle moved into the body, and its panel opens BELOW it.
    await expect(page.getByTestId('toggle-fetch')).toBeHidden();
    const toggle = page.getByTestId('toggle-fetch-mobile');
    await expect(toggle).toBeVisible();
    await toggle.click();
    const panel = (await page.getByTestId('fetch-section').boundingBox())!;
    expect(panel.y).toBeGreaterThan((await toggle.boundingBox())!.y);

    await page.keyboard.press('Escape');
    await expect(sheet).toBeHidden();
  });

  test('[P0] 640px — the other side of the breakpoint — is the untouched desktop dialog', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 斷點字幕 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(AT_BREAKPOINT);
    await stubMovie(page, movie.id);

    const dialog = await openDialog(page, movie.id);

    expect(await animationName(page)).toBe('dialog-enter');
    await expect(page.getByTestId('manage-sheet-grabber')).toBeHidden();
    await expect(page.getByTestId('dialog-close')).toBeVisible();
    await expect(page.getByTestId('toggle-fetch')).toBeVisible();
    await expect(page.getByTestId('toggle-fetch-mobile')).toBeHidden();

    // Centred, not pinned: there is air under it.
    const box = (await dialog.boundingBox())!;
    expect(box.y + box.height).toBeLessThan(AT_BREAKPOINT.height);

    // The generate button is NOT stretched here.
    const cta = (await page.getByTestId('action-generate-subtitle').boundingBox())!;
    expect(cta.width).toBeLessThan(box.width / 2);
  });

  test('[P1] 390px generating: 關閉 fills the sheet and the hint sits UNDER it', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 手機生成 ${Date.now()}` });
    movieIds.push(movie.id);
    await page.setViewportSize(PHONE);
    await stubMovie(page, movie.id);
    // `?translate=true` rides the real URL — a glob without the `*` never matches it.
    await page.route(`**/api/v1/movies/${movie.id}/transcribe?*`, (route) =>
      route.request().method() === 'POST'
        ? route.fulfill({
            status: 202,
            json: { success: true, data: { job_id: 'e2e-job', message: 'started' } },
          })
        : route.fallback()
    );
    // One canned frame is enough to put the run in 轉錄中; the stream then ends,
    // and the hook's reconnect just replays the same frame.
    await page.route('**/api/v1/events', (route) =>
      route.fulfill({
        status: 200,
        headers: { 'content-type': 'text/event-stream', 'cache-control': 'no-cache' },
        body: `event: transcription_progress\ndata: ${JSON.stringify({
          id: 'e2e',
          type: 'transcription_progress',
          data: { media_id: movie.id, media_type: 'movie' },
        })}\n\n`,
      })
    );

    const sheet = await openDialog(page, movie.id);
    await page.getByTestId('action-generate-subtitle').click();
    await expect(sheet.getByText(/^生成字幕 — /)).toBeVisible();
    // Proves the SSE stub was actually CONSUMED: the run starts in 提取音訊, and only
    // the canned `transcription_progress` frame moves it on. Without this, every
    // assertion below also holds in the initial phase and a dead stub stays green.
    await expect(page.getByTestId('gen-stage-轉錄中')).toHaveAttribute('data-state', 'active');

    const close = page.getByTestId('dialog-close');
    await expect(close).toBeVisible();
    await expect(close).toHaveText('關閉');
    const closeBox = (await close.boundingBox())!;
    expect(Math.round(closeBox.width)).toBe(PHONE.width - 32);

    const hint = sheet.getByText('關閉後生成會在背景繼續');
    await expect(hint).toBeVisible();
    expect((await hint.boundingBox())!.y).toBeGreaterThan(closeBox.y + closeBox.height - 1);

    // The vertical stepper: Body 14 labels.
    const label = page.getByTestId('gen-stage-轉錄中').getByText('轉錄中');
    expect(await label.evaluate((el) => window.getComputedStyle(el).fontSize)).toBe('14px');
  });
});
