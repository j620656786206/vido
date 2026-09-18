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
 *  - a refused item says WHY, instead of being drawn as 完成.
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
});
