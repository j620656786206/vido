/**
 * Shared network stubs for the 產生字幕 flow (consent F15/F16 → batch F8).
 *
 * Lifted verbatim out of tests/e2e/batch-subtitle.spec.ts (dsr-6f-3) so the phone
 * spec drives the exact same fake backend — one definition of what a ready
 * candidates snapshot and a started batch look like on the wire.
 */

import { expect, type Route } from '../fixtures';
import type { Page } from '@playwright/test';

export const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock payloads — snake_case wire format
// =============================================================================

export const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

export const jsonStatus = <T>(status: number, body: T, success = true) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify({ success, data: body }),
});

// A populated two-movie library so the grid renders and `enter-selection-btn`
// shows. UUID string ids — the [@contract-v2] media_id contract (9R-18).
export const populatedLibrary = {
  items: [
    {
      type: 'movie',
      movie: {
        id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
        title: '駭客任務',
        release_date: '1999-03-31',
        genres: ['動作', '科幻'],
        poster_path: '/matrix.jpg',
        parse_status: 'parsed',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    },
    {
      type: 'movie',
      movie: {
        id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57',
        title: '星際效應',
        release_date: '2014-11-07',
        genres: ['劇情', '科幻'],
        poster_path: '/interstellar.jpg',
        parse_status: 'parsed',
        created_at: '2026-05-02T00:00:00Z',
        updated_at: '2026-05-02T00:00:00Z',
      },
    },
  ],
  page: 1,
  page_size: 24,
  total_items: 2,
  total_pages: 1,
};

export const libraryStats = { total_count: 2, movie_count: 2, series_count: 0 };
export const mediaStats = { total_count: 2, matched_count: 2, unmatched_count: 0 };
export const qbtConnected = {
  host: 'http://localhost:8080',
  username: 'admin',
  base_path: '',
  configured: true,
};

export const startedBatchItems = [
  {
    media_id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
    title: '駭客任務',
    media_type: 'movie',
    series_title: '',
  },
  {
    media_id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57',
    title: '星際效應',
    media_type: 'movie',
    series_title: '',
  },
];

/**
 * 202 body since dsr-6d-a: the enumerated queue PLUS the started batch's own
 * snapshot, so the dialog paints the queue without waiting for an SSE event.
 */
export const startedBatch = {
  batch_id: 'gb-e2e-1',
  total_items: 2,
  items: startedBatchItems,
  progress: {
    batch_id: 'gb-e2e-1',
    total_items: 2,
    current_index: 1,
    current_media_id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
    current_item: '駭客任務',
    success_count: 0,
    fail_count: 0,
    paused_count: 0,
    status: 'running',
    spent_usd: 0,
    budget_usd: 5,
    items: startedBatchItems.map((it, i) => ({
      ...it,
      status: i === 0 ? 'running' : 'queued',
      reason: '',
    })),
  },
};

// sub-4-3: a READY candidates snapshot (both library movies, extract route,
// honest small translation fees — §5-sexies). Default selection = extract-all.
export const readyCandidates = {
  status: 'ready',
  analyzed: 2,
  total: 2,
  analyzed_at: '2026-08-11T00:00:00Z',
  result: {
    candidates: [
      {
        media_id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
        media_type: 'movie',
        title: '駭客任務',
        route: 'extract',
        runtime_minutes: 136,
        runtime_known: true,
        estimated_usd: 0.05,
      },
      {
        media_id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57',
        media_type: 'movie',
        title: '星際效應',
        route: 'extract',
        runtime_minutes: 169,
        runtime_known: true,
        estimated_usd: 0.04,
      },
    ],
    summary: {
      extract_count: 2,
      asr_count: 0,
      skipped_count: 0,
      estimated_total_usd: 0.09,
      self_hosted_asr: false,
    },
  },
};

// =============================================================================
// Baseline stubs — a populated /library so selection mode is reachable
// =============================================================================

export async function stubPopulatedLibrary(page: Page) {
  // Abort the SSE stream: EventSource must never depend on a live backend. The
  // optimistic 'running' state is dispatched before the stream matters, and the
  // lazy-SSE test asserts the stream is not even requested while idle.
  await page.route(`${ROUTE_API}/events`, (route: Route) => route.abort());

  await page.route(`${ROUTE_API}/library/stats`, (route: Route) =>
    route.fulfill(jsonOk(libraryStats))
  );
  await page.route(`${ROUTE_API}/library/genres`, (route: Route) => route.fulfill(jsonOk([])));
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, page_size: 20, total_items: 0, total_pages: 0 }))
  );
  await page.route(`${ROUTE_API}/movies/stats`, (route: Route) =>
    route.fulfill(jsonOk(mediaStats))
  );
  await page.route(`${ROUTE_API}/series/stats`, (route: Route) =>
    route.fulfill(jsonOk(mediaStats))
  );
  await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
    route.fulfill(jsonOk(qbtConnected))
  );
  await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
    route.fulfill(jsonOk({ libraries: [{ id: 1, name: '電影', content_type: 'movie' }] }))
  );
  await page.route(`${ROUTE_API}/health/services*`, (route: Route) =>
    route.fulfill(jsonOk({ services: [] }))
  );
  await page.route(`${ROUTE_API}/scanner/status`, (route: Route) =>
    route.fulfill(jsonOk({ status: 'idle', progress: 0 }))
  );
  // Empty search — registered BEFORE the /library* catch-all (specific-first wins).
  await page.route(`${ROUTE_API}/library/search*`, (route: Route) =>
    route.fulfill(jsonOk({ results: [], total_count: 0 }))
  );
  // Catch-all populated list for bare /library and /library?page=... variants.
  await page.route(`${ROUTE_API}/library*`, (route: Route) =>
    route.fulfill(jsonOk(populatedLibrary))
  );

  // Generation-batch dialog on-open calls (9R-16): the recovery status probe
  // (nothing running) + the 缺字幕 preview count.
  await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
    // dsr-6d-a: `last` is the remembered terminal snapshot — null here means the
    // dialog opens straight into the consent flow.
    route.fulfill(jsonOk({ running: false, progress: null, last: null }))
  );
  await page.route(`${ROUTE_API}/subtitles/generation-batch/preview*`, (route: Route) =>
    route.fulfill(jsonOk({ total_items: 2 }))
  );
  // sub-4-3 consent flow: the candidates snapshot is READY — the flow renders
  // F15 directly (no analyze POST, no SSE — the lazy-SSE test relies on this).
  await page.route(`${ROUTE_API}/subtitles/generation-candidates`, (route: Route) =>
    route.fulfill(jsonOk(readyCandidates))
  );
}

/** Walk the consent flow to a started batch: F15 → 開始產生 → 確認並開始. */
export async function confirmAndStart(page: Page) {
  await page.getByTestId('consent-start-btn').click();
  await expect(page.getByTestId('consent-confirm-dialog')).toBeVisible();
  await page.getByTestId('consent-confirm-start').click();
}

/** Enter selection mode and open the consent flow from /library (sub-4-3:
 * the dialog's idle phase IS the consent view — F15 renders from the ready
 * snapshot after the recovery probe settles). */
export async function openGenerationDialog(page: Page) {
  await page.goto('/library');
  await page.getByTestId('enter-selection-btn').click();
  await expect(page.getByTestId('selection-toolbar')).toBeVisible();
  await page.getByTestId('batch-subtitle-btn').click();
  await expect(page.getByTestId('generation-consent-view')).toBeVisible();
  await expect(page.getByTestId('consent-candidate-list')).toBeVisible();
}
