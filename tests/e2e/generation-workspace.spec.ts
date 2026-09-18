import { test, expect, type Page, type Route } from '@playwright/test';

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
const ROUTE_API = '**/api/v1';

const jsonOk = (data: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data }),
});

const ITEMS = [
  {
    media_id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
    title: '駭客任務',
    media_type: 'movie',
    series_title: '',
    status: 'done',
    reason: '',
  },
  {
    media_id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57',
    title: '星際效應',
    media_type: 'movie',
    series_title: '',
    status: 'failed',
    reason: 'busy_elsewhere',
  },
  {
    media_id: '9ff0c000-dead-4bee-8f00-000000000999',
    title: '正在處理的電影',
    media_type: 'movie',
    series_title: '',
    status: 'running',
    reason: '',
  },
];

const snapshot = (over: Record<string, unknown> = {}) => ({
  batch_id: 'gb-ws-1',
  total_items: 3,
  current_index: 3,
  current_media_id: '9ff0c000-dead-4bee-8f00-000000000999',
  current_item: '正在處理的電影',
  success_count: 1,
  fail_count: 1,
  paused_count: 0,
  status: 'running',
  spent_usd: 0.42,
  budget_usd: 5,
  items: ITEMS,
  ...over,
});

async function stubCommon(page: Page) {
  await page.route(`${ROUTE_API}/activity`, (route: Route) =>
    route.fulfill(
      jsonOk({
        activeJobs: { status: 'ok', jobs: [] },
        pending: { status: 'ok', parseCount: 0 },
        downloads: { status: 'ok', downloading: 0, queued: 0, errored: 0 },
        recent: { status: 'ok', items: [] },
      })
    )
  );
  await page.route(`${ROUTE_API}/subtitles/generation-batch/preview*`, (route: Route) =>
    route.fulfill(jsonOk({ total_items: 2, total_items_including_episodes: 7 }))
  );
}

/** One SSE frame in the hub's wire shape: the whole Event struct as `data:`. */
const sseFrame = (type: string, data: unknown) =>
  `event: ${type}\ndata: ${JSON.stringify({ type, data })}\n\n`;

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
});
