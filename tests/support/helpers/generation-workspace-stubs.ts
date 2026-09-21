import type { Page, Route } from '@playwright/test';

/**
 * Shared stubs for the generation WORKSPACE e2e specs (`/activity?view=generation`):
 * generation-workspace.spec.ts (desktop, dsr-6d-c-1/2) and
 * generation-workspace-mobile.spec.ts (phone, dsr-6f-4). Moved here verbatim from
 * the desktop spec — the shapes are the backend's (`progress.items[]` / `last`).
 */
export const ROUTE_API = '**/api/v1';

export const jsonOk = (data: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data }),
});

export const ITEMS = [
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

export const snapshot = (over: Record<string, unknown> = {}) => ({
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

export async function stubCommon(page: Page) {
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
export const sseFrame = (type: string, data: unknown) =>
  `event: ${type}\ndata: ${JSON.stringify({ type, data })}\n\n`;

/**
 * Serve `frames` to the FIRST EventSource connection only. A fulfilled body ENDS
 * the stream (a real SSE never does); replaying it on the hook's reconnect would
 * log every event twice, so later connections are refused.
 */
export async function stubEventsOnce(page: Page, frames: string) {
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
}

/** `n` films each going running → done: `n` result rows in the live log. */
export function finishedFilmsFrames(n: number): string {
  return Array.from({ length: n }, (_, i) => ({
    media_id: `00000000-0000-4000-8000-${String(i).padStart(12, '0')}`,
    title: `第 ${i + 1} 部`,
    media_type: 'movie',
    series_title: '',
    reason: '',
  }))
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
}
