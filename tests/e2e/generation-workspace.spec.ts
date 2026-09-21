import { test, expect, type Route } from '@playwright/test';
import {
  ROUTE_API,
  ITEMS,
  jsonOk,
  snapshot,
  sseFrame,
  stubCommon,
} from '../support/helpers/generation-workspace-stubs';

/**
 * Generation WORKSPACE e2e (dsr-6d-c-1). The workspace had ZERO e2e coverage —
 * the existing batch-subtitle suite only drives the DIALOG from /library and
 * never renders this page at all.
 *
 * What this pins, in a real browser:
 *  - `/activity?view=generation` actually mounts the workspace;
 *  - a RUNNING batch draws real rows from the backend's `progress.items[]`;
 *  - a FINISHED batch still draws its queue — from the status probe's `last` —
 *    which is the regression dsr-6d-b introduced by removing the items cache;
 *  - a refused item says WHY, instead of being drawn as 完成;
 *  - (dsr-6d-c-2) the live log keeps one row per film per stage, names the film,
 *    and drops `subtitle_progress` that is not this batch's (search / download).
 */
test.describe('Generation Workspace @ui @generation-workspace', () => {
  test('[P0] a RUNNING batch draws the backend queue, and a refused item says WHY', async ({
    page,
  }) => {
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(jsonOk({ running: true, progress: snapshot(), last: null }))
    );

    await page.goto('/activity?view=generation');

    await expect(page.getByTestId('generation-workspace')).toBeVisible();
    await expect(
      page.getByTestId('workspace-queue-row-5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63')
    ).toBeVisible();

    // The whole point of dsr-6d-c-1: the refused item is NOT drawn as 完成.
    const refused = page.getByTestId('workspace-queue-row-8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57');
    await expect(refused).toHaveAttribute('data-state', 'failed');
    await expect(refused).toContainText('這部正在別處處理');
  });

  test('[P0] a FINISHED batch still shows its queue — from `last`, not the dialog cache', async ({
    page,
  }) => {
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(
        jsonOk({
          running: false,
          progress: null,
          last: snapshot({
            status: 'complete',
            success_count: 1,
            fail_count: 2,
            current_media_id: '',
            current_item: '',
            items: ITEMS.map((it) =>
              it.status === 'running' ? { ...it, status: 'failed', reason: 'skipped' } : it
            ),
          }),
        })
      )
    );

    await page.goto('/activity?view=generation');

    await expect(
      page.getByTestId('workspace-queue-row-5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63')
    ).toBeVisible();
    // Neutral verdict — a run with failures never claims 全部完成.
    await expect(page.getByTestId('workspace-terminal-label')).toHaveText('完成 1 部、失敗 2 部');
    // …and the way out exists.
    await expect(page.getByTestId('workspace-close')).toBeVisible();
  });

  test('[P1] no batch at all → the idle workspace with the episode-inclusive count', async ({
    page,
  }) => {
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(jsonOk({ running: false, progress: null, last: null }))
    );

    await page.goto('/activity?view=generation');

    const idle = page.getByTestId('workspace-idle');
    await expect(idle).toBeVisible();
    await expect(idle).toContainText('目前沒有進行中的生成');
    // sub-5-1 AC #7: 7 (incl. episodes), not the movies-only 2.
    await expect(idle).toContainText('7');
  });

  test('[P0] the live log: one row per film per stage, the film named, foreign events dropped', async ({
    page,
  }) => {
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(jsonOk({ running: true, progress: snapshot(), last: null }))
    );
    const running = '9ff0c000-dead-4bee-8f00-000000000999';
    const frames = [
      sseFrame('generation_batch_progress', {
        ...snapshot(),
        items: null,
        changed_item: ITEMS[2],
      }),
      sseFrame('transcription_extracting', { media_id: running, phase: 'extracting', title: '' }),
      ...[10, 20, 30].map((pct) =>
        sseFrame('translation_progress', {
          media_id: running,
          phase: 'translating',
          percentage: pct,
          title: '',
        })
      ),
      // The subtitle SEARCH engine reuses this event name, in English. Not ours.
      sseFrame('subtitle_progress', {
        media_id: '1a2b3c4d-0000-4000-8000-00000000abcd',
        media_type: 'movie',
        stage: 'searching',
        message: 'Searching subtitle providers...',
      }),
    ].join('');
    // The log's own EventSource connects first (on mount, before the status probe
    // answers). It gets the frames; every later connection is refused. A fulfilled
    // body ENDS the stream — a real SSE never does — and a replay on reconnect
    // would log the same events twice.
    let served = 0;
    await page.route(`${ROUTE_API}/events`, (route: Route) => {
      served += 1;
      if (served > 1) return route.abort();
      return route.fulfill({
        status: 200,
        headers: { 'content-type': 'text/event-stream', 'cache-control': 'no-cache' },
        body: frames,
      });
    });

    await page.goto('/activity?view=generation');

    const log = page.getByTestId('workspace-event-log');
    const translating = log.getByTestId('workspace-feed-row').filter({ hasText: '翻譯中' });
    await expect(translating).toHaveCount(1);
    await expect(translating).toContainText('30%');
    await expect(translating).toContainText('正在處理的電影');
    await expect(log).not.toContainText('Searching');
    await expect(log).not.toContainText(running);
  });

  test('[P1] a long log scrolls inside its pane and follows the newest row (CR H3)', async ({
    page,
  }) => {
    // The pane cap is `lg:`-only, so below 1024 the LIST never scrolled (the page
    // did) — and since dsr-6f-4 a phone starts with the log collapsed. The phone
    // behaviour is measured in generation-workspace-mobile.spec.ts. Width, not
    // project name: firefox (1280) is a valid gate for this test.
    test.skip((page.viewportSize()?.width ?? 1280) < 1024, 'pane cap is lg-only');
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(jsonOk({ running: true, progress: snapshot(), last: null }))
    );
    // 40 films finishing → 40 result rows: far more than one screen.
    const films = Array.from({ length: 40 }, (_, i) => ({
      media_id: `00000000-0000-4000-8000-${String(i).padStart(12, '0')}`,
      title: `第 ${i + 1} 部`,
      media_type: 'movie',
      series_title: '',
      reason: '',
    }));
    const frames = films
      .flatMap((film) => [
        sseFrame('generation_batch_progress', {
          ...snapshot(),
          items: null,
          changed_item: { ...film, status: 'running' },
        }),
        sseFrame('generation_batch_progress', {
          ...snapshot(),
          items: null,
          changed_item: { ...film, status: 'done' },
        }),
      ])
      .join('');
    let served = 0;
    await page.route(`${ROUTE_API}/events`, (route: Route) => {
      served += 1;
      if (served > 1) return route.abort();
      return route.fulfill({
        status: 200,
        headers: { 'content-type': 'text/event-stream', 'cache-control': 'no-cache' },
        body: frames,
      });
    });

    await page.goto('/activity?view=generation');

    const log = page.getByTestId('workspace-event-log');
    await expect(log.getByTestId('workspace-feed-row')).toHaveCount(40);
    const list = log.getByRole('list', { name: '生成事件日誌' });
    // The LIST scrolls (the pane is capped), and it is parked on the newest row.
    await expect
      .poll(() =>
        list.evaluate((el) => ({
          scrolls: el.scrollHeight > el.clientHeight,
          atBottom: el.scrollTop + el.clientHeight >= el.scrollHeight - 2,
        }))
      )
      .toEqual({ scrolls: true, atBottom: true });
    const newest = log.getByTestId('workspace-feed-row').last();
    await expect(newest).toContainText('第 40 部');
    // Scrolling the PAGE parks the pane under the app header (sticky, capped to the
    // viewport): the newest row is then on screen without touching the list.
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await expect(newest).toBeInViewport();
  });
});
