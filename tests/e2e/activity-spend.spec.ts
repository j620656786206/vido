import { test, expect, type Route } from '@playwright/test';
import { ROUTE_API, jsonOk, stubCommon } from '../support/helpers/generation-workspace-stubs';

/**
 * 本月 AI 花費 on /activity (sub-7-6c AC #1/#4/#6, K5-D xgYKA / K5-M ptNai).
 *
 * In a real browser, with the two endpoints stubbed independently:
 *  - a month with money in it draws the card — and, with nothing running, the
 *    card is the page (the calm empty state must NOT win);
 *  - the honesty marks come through end to end: 「—」 for an unmeasured cache,
 *    the word 估算 on every saving, ≈ only when a runtime was assumed;
 *  - a broken spend endpoint degrades to its own 無法載入 while the rest of
 *    the hub renders on (fail-soft, F3).
 */
const SPEND_MONTH = {
  period: 'month',
  from: '2026-10-01T00:00:00+08:00',
  to: '2026-11-01T00:00:00+08:00',
  translated_runs: 12,
  translated_usd: 3.48,
  asr_runs: 2,
  asr_usd: 1.9,
  unpriced_runs: 0,
  unrouted_runs: 0,
  unrouted_usd: 0,
  skipped_deliver_count: 8,
  skipped_saved_usd_estimate: 2.14,
  skipped_saved_runtime_assumed: true,
  cache_hit_cues: 0,
  cache_measured_runs: 0,
  cache_saved_usd_estimate: null,
  by_model: [
    { model_id: 'claude-sonnet-5', runs: 9, usd: 4.2 },
    { model_id: 'claude-haiku-4-5', runs: 5, usd: 1.18 },
  ],
};

/** GET /activity in the backend's own shape (snake_case, the four sections). */
const ACTIVITY_IDLE = {
  active_jobs: { status: 'ok', jobs: [] },
  pending: { status: 'ok', parse_count: 0 },
  downloads: { status: 'ok', downloading: 0, queued: 0, errored: 0, paused: 0, total: 0 },
  recent: { status: 'ok', events: [] },
};

test.describe('Activity hub — 本月 AI 花費 @ui @activity @story-sub-7-6c', () => {
  test('[P0] a month with spend draws the card with its honesty marks; nothing running → the card is the page', async ({
    page,
  }) => {
    await stubCommon(page);
    // stubCommon's /activity is the WORKSPACE view's shape; the hub body reads
    // the full four-section contract, so it is overridden here (last route wins).
    await page.route(`${ROUTE_API}/activity`, (route: Route) =>
      route.fulfill(jsonOk(ACTIVITY_IDLE))
    );
    await page.route(`${ROUTE_API}/subtitles/spend*`, (route: Route) =>
      route.fulfill(jsonOk(SPEND_MONTH))
    );

    await page.goto('/activity');

    const card = page.getByTestId('activity-spend-card');
    await expect(card).toBeVisible();
    await expect(page.getByTestId('activity-empty')).toHaveCount(0);
    await expect(page.getByTestId('activity-spend-month')).toHaveText('10 月');
    await expect(page.getByTestId('activity-spend-translated')).toContainText('12 次 · $3.48');
    await expect(page.getByTestId('activity-spend-asr')).toContainText('2 次 · $1.90');
    // ≈ because the backend flagged an assumed runtime; the word 估算 regardless.
    await expect(page.getByTestId('activity-spend-skipped')).toHaveText(
      '略過 8 集，估算省下 ≈ $2.14'
    );
    // No run measured its cache → 「—」, never $0.
    await expect(page.getByTestId('activity-spend-cache')).toHaveText('快取估算省下 —');
    await expect(page.getByTestId('activity-spend-model-row')).toHaveCount(2);
    await expect(page.getByTestId('activity-spend-model-row').first()).toContainText(
      'claude-sonnet-5'
    );
  });

  test('[P1] a broken spend endpoint degrades to its own 無法載入; the hub renders on', async ({
    page,
  }) => {
    await stubCommon(page);
    await page.route(`${ROUTE_API}/activity`, (route: Route) =>
      route.fulfill(jsonOk({ ...ACTIVITY_IDLE, pending: { status: 'ok', parse_count: 4 } }))
    );
    await page.route(`${ROUTE_API}/subtitles/spend*`, (route: Route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'INTERNAL', message: '無法讀取字幕花費紀錄，請稍後再試。' },
        }),
      })
    );

    await page.goto('/activity');

    await expect(page.getByTestId('activity-spend-error')).toBeVisible();
    await expect(page.getByTestId('activity-pending-row')).toBeVisible();
    await expect(page.getByTestId('activity-page-error')).toHaveCount(0);
    await expect(page.getByTestId('activity-empty')).toHaveCount(0);
  });
});
