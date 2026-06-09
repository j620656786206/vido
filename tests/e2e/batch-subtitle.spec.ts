/**
 * Batch Subtitle Search UI E2E Tests (Story 8-11)
 *
 * Browser-based tests for the library batch-subtitle trigger + dialog. DEV's
 * unit/component tests (subtitleService.spec, useSubtitleBatchProgress.spec,
 * BatchSubtitleDialog.spec, SelectionToolbar.spec — 42 specs) mocked every
 * boundary. This suite exercises the wire-level integration those mocks hide:
 *
 *   /library → SelectionToolbar → BatchSubtitleDialog → subtitleService
 *                                                       ↓
 *                          POST /api/v1/subtitles/batch  (202 / 409 / cancel)
 *
 * Coverage gaps this suite closes (vs. DEV's unit tests):
 *   - [P0] Real selection-toolbar wiring: 批次字幕搜尋 button reachable on /library  (AC#1)
 *   - [P0] Dialog opens with library scope default; season scope absent on /library  (AC#2)
 *   - [P0] Real POST body {scope:"library"} (Rule 18 camelToSnake) + 202 → processing (AC#3)
 *   - [P1] 409 conflict recovers to the in-progress snapshot, no error surfaced     (AC#7)
 *   - [P1] Lazy SSE: idle dialog opens NO EventSource on mount; page hits networkidle (AC#8)
 *   - [P2] Cancel confirm fires the real POST /api/v1/subtitles/batch/cancel         (AC#5)
 *
 * DELIBERATELY OUT OF SCOPE — the live SSE event stream (AC#4 progress increments,
 * AC#6 complete-summary + 「查看未找到項目」 deep-link, AC#5 terminal `cancelled`
 * reflection). This repo never mocks the /api/v1/events stream in E2E (the lazy-SSE
 * pattern exists precisely so `networkidle` works), and `startTracking()` flips to
 * the processing state OPTIMISTICALLY — so a 202 alone is enough to assert the
 * processing UI without any SSE events. The live journey is covered at the hook +
 * component level (useSubtitleBatchProgress.spec drives SSE_UPDATE; BatchSubtitleDialog.spec
 * drives the full terminal state machine) and is the proper home for a future
 * TestSprite journey case (real backend) — see story 8-11 Dev Notes + the
 * disc-2026-06-batch-subtitle-frontend-ui triage. AC#6's deep-link target is also
 * blocked by the deferred backend subtitle_status filter (backlog
 * disc-2026-06-library-subtitle-status-filter).
 *
 * Design notes:
 *   - Route interception is installed BEFORE page.goto (network-first, knowledge/network-first.md).
 *   - Mock payloads are snake_case at the wire (fetchApi runs snakeToCamel).
 *   - POST bodies are captured via postDataJSON() to verify Rule 18 at the real network layer.
 *   - /api/v1/events is aborted so EventSource never depends on a live backend; the
 *     optimistic 'running' state is already dispatched before the stream matters.
 *
 * @tags @ui @batch-subtitle @story-8-11
 */

import { test, expect, type Route } from '../support/fixtures';
import type { Page, Request } from '@playwright/test';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock payloads — snake_case wire format
// =============================================================================

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

const jsonStatus = <T>(status: number, body: T, success = true) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify({ success, data: body }),
});

// A populated two-item library so the grid renders and `enter-selection-btn` shows.
const populatedLibrary = {
  items: [
    {
      type: 'movie',
      movie: {
        id: 'm-603',
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
        id: 'm-157336',
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

const libraryStats = { total_count: 2, movie_count: 2, series_count: 0 };
const mediaStats = { total_count: 2, matched_count: 2, unmatched_count: 0 };
const qbtConnected = {
  host: 'http://localhost:8080',
  username: 'admin',
  base_path: '',
  configured: true,
};

// =============================================================================
// Baseline stubs — a populated /library so selection mode is reachable
// =============================================================================

async function stubPopulatedLibrary(page: Page) {
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
}

/** Enter selection mode and open the batch-subtitle dialog from /library. */
async function openBatchDialog(page: Page) {
  await page.goto('/library');
  await page.getByTestId('enter-selection-btn').click();
  await expect(page.getByTestId('selection-toolbar')).toBeVisible();
  await page.getByTestId('batch-subtitle-btn').click();
  await expect(page.getByTestId('batch-subtitle-dialog')).toBeVisible();
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Batch Subtitle Search UI @ui @batch-subtitle @story-8-11', () => {
  test.beforeEach(async ({ page }) => {
    await stubPopulatedLibrary(page);
  });

  test('[P0] 批次字幕搜尋 trigger is reachable in the selection toolbar (AC#1)', async ({
    page,
  }) => {
    // GIVEN: a populated library
    await page.goto('/library');

    // WHEN: the user enters selection mode
    await page.getByTestId('enter-selection-btn').click();

    // THEN: the batch-subtitle action is visible alongside the other batch actions
    await expect(page.getByTestId('selection-toolbar')).toBeVisible();
    await expect(page.getByTestId('batch-subtitle-btn')).toBeVisible();
  });

  test('[P0] dialog opens with library scope default; season scope absent on /library (AC#2)', async ({
    page,
  }) => {
    // GIVEN/WHEN: the user opens the batch-subtitle dialog
    await openBatchDialog(page);

    // THEN: the start button + library scope (pre-selected) render
    await expect(page.getByTestId('batch-subtitle-start-btn')).toBeVisible();
    await expect(page.getByTestId('batch-subtitle-scope-library')).toBeChecked();

    // AND: the season scope is not offered without a seasonId context
    await expect(page.getByTestId('batch-subtitle-scope-season')).toHaveCount(0);
  });

  test('[P0] start sends POST {scope:"library"} and 202 transitions to processing (AC#3)', async ({
    page,
  }) => {
    // GIVEN: the batch endpoint accepts the request (202 Accepted, 42 items)
    let captured: Request | null = null;
    await page.route(`${ROUTE_API}/subtitles/batch`, (route: Route) => {
      captured = route.request();
      return route.fulfill(jsonStatus(202, { batch_id: 'batch-abc', total_items: 42 }));
    });
    await openBatchDialog(page);

    // WHEN: the user starts the batch with the default library scope
    await page.getByTestId('batch-subtitle-start-btn').click();

    // THEN: the panel transitions to the processing state. The counter is the
    // visible signal; the progress bar at 0/42 has width:0% (zero-width → not
    // "visible") so it is asserted with toBeAttached (Rule 16).
    await expect(page.getByTestId('batch-subtitle-counter')).toHaveText('0 / 42');
    await expect(page.getByTestId('batch-subtitle-progress-bar')).toBeAttached();

    // AND: the wire body is snake_case {scope:"library"} with no season_id (Rule 18)
    expect(captured).not.toBeNull();
    expect(captured!.postDataJSON()).toEqual({ scope: 'library' });
  });

  test('[P1] 409 conflict recovers to the in-progress batch without erroring (AC#7)', async ({
    page,
  }) => {
    // GIVEN: a batch is already running — the server answers 409 with its snapshot
    await page.route(`${ROUTE_API}/subtitles/batch`, (route: Route) =>
      route.fulfill(
        jsonStatus(
          409,
          {
            batch_id: 'batch-existing',
            total_items: 42,
            current_index: 10,
            current_item: '正在處理的電影',
            success_count: 7,
            fail_count: 3,
            status: 'running',
          },
          false
        )
      )
    );
    await openBatchDialog(page);

    // WHEN: the user attempts to start another batch
    await page.getByTestId('batch-subtitle-start-btn').click();

    // THEN: the panel shows the in-progress snapshot instead of an error
    await expect(page.getByTestId('batch-subtitle-counter')).toHaveText('10 / 42');
    await expect(page.getByTestId('batch-subtitle-found')).toHaveText('找到 7');
    await expect(page.getByTestId('batch-subtitle-notfound')).toHaveText('未找到 3');
    await expect(page.getByTestId('batch-subtitle-error')).toHaveCount(0);
  });

  test('[P1] lazy SSE — idle dialog opens no EventSource and the page reaches networkidle (AC#8)', async ({
    page,
  }) => {
    // GIVEN: every request is observed
    const sseRequests: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/events')) sseRequests.push(req.url());
    });

    // WHEN: the dialog is opened and left in the idle state
    await openBatchDialog(page);
    await expect(page.getByTestId('batch-subtitle-start-btn')).toBeVisible();

    // THEN: no EventSource connection was attempted on mount (lazy pattern, project-context §8)
    expect(sseRequests).toHaveLength(0);

    // AND: with no open SSE stream, the page settles to networkidle (the exact
    // property eager SSE would break). This is the load-bearing lazy-SSE assertion.
    await page.waitForLoadState('networkidle');
  });

  test('[P2] cancel confirmation fires the real POST /subtitles/batch/cancel (AC#5)', async ({
    page,
  }) => {
    // GIVEN: a batch is started (202 → processing)
    await page.route(`${ROUTE_API}/subtitles/batch`, (route: Route) =>
      route.fulfill(jsonStatus(202, { batch_id: 'batch-abc', total_items: 42 }))
    );
    let cancelHit = false;
    await page.route(`${ROUTE_API}/subtitles/batch/cancel`, (route: Route) => {
      cancelHit = true;
      return route.fulfill(jsonOk({ cancelled: true }));
    });
    await openBatchDialog(page);
    await page.getByTestId('batch-subtitle-start-btn').click();
    await expect(page.getByTestId('batch-subtitle-cancel-btn')).toBeVisible();

    // WHEN: the user cancels and confirms
    await page.getByTestId('batch-subtitle-cancel-btn').click();
    await expect(page.getByTestId('batch-subtitle-cancel-confirm')).toBeVisible();
    await page.getByTestId('batch-subtitle-cancel-confirm-btn').click();

    // THEN: the cancel endpoint was called (the terminal `cancelled` status arrives
    // via SSE — covered at the hook level, not here).
    await expect.poll(() => cancelHit).toBe(true);
  });
});
