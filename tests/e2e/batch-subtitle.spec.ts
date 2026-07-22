/**
 * Batch Subtitle GENERATION UI E2E Tests (Story ux3-subtitle-v2-batch)
 *
 * Re-pointed from the Story 8-11 fetch-batch suite: /library's selection
 * toolbar now opens `GenerationBatchDialogV2` (Route C generation, 9R-16
 * endpoints) — `BatchSubtitleDialog` is superseded and no longer mounted, so
 * the old journeys are unreachable. Unit/component specs
 * (subtitleService.spec, useGenerationBatchProgress.spec,
 * GenerationBatchDialogV2.spec — mocked at every boundary) are complemented
 * here by the wire-level integration those mocks hide:
 *
 *   /library → SelectionToolbar → GenerationBatchDialogV2 → subtitleService
 *                                                           ↓
 *              POST /api/v1/subtitles/generation-batch  (202 / 409 / cancel)
 *
 * Coverage (vs the unit specs):
 *   - [P0] Real selection-toolbar wiring: 批次生成字幕 reachable on /library   (AC#5)
 *   - [P0] Dialog opens idle: scope=missing preselected + real preview count  (AC#1)
 *   - [P0] Real POST body {scope:"missing"} (Rule 18) + 202 items[] → running (AC#1/3)
 *   - [P0] Selection ACTUALLY flows: media_ids on the wire, movies only       (AC#5)
 *   - [P1] 409 TRANSCRIPTION_BATCH_RUNNING recovers to the snapshot           (AC#1)
 *   - [P1] Lazy SSE: idle dialog opens NO EventSource; page hits networkidle  (§8)
 *   - [P2] 全部取消 confirm fires POST /subtitles/generation-batch/cancel     (AC#1)
 *
 * DELIBERATELY OUT OF SCOPE — the live `generation_batch_progress` SSE stream
 * (running increments, budget_ceiling F9, terminal close). This repo never
 * mocks /api/v1/events in E2E (the lazy-SSE pattern exists precisely so
 * `networkidle` works) and `startTracking()` enters the running state
 * OPTIMISTICALLY from the 202 — the SSE state machine is covered at the hook
 * (useGenerationBatchProgress.spec terminal-close matrix) and component
 * (GenerationBatchDialogV2.spec state matrix incl. the batch-status-
 * authoritative race) levels.
 *
 * Design notes:
 *   - Route interception installed BEFORE page.goto (network-first).
 *   - Mock payloads are snake_case at the wire (fetchApi runs snakeToCamel).
 *   - POST bodies captured via postDataJSON() to verify Rule 18 at the network layer.
 *   - Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
 *     mirror the prod creation path (uuid.New().String()); do NOT invent
 *     numeric ids. media_ids ride the wire UNCONVERTED ([@contract-v2];
 *     disc-2026-07-movie-id-int64-contract-mismatch CLOSED by 9R-18).
 *
 * @tags @ui @batch-subtitle @ux3-subtitle-v2-batch
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

// A populated two-movie library so the grid renders and `enter-selection-btn`
// shows. UUID string ids — the [@contract-v2] media_id contract (9R-18).
const populatedLibrary = {
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

const libraryStats = { total_count: 2, movie_count: 2, series_count: 0 };
const mediaStats = { total_count: 2, matched_count: 2, unmatched_count: 0 };
const qbtConnected = {
  host: 'http://localhost:8080',
  username: 'admin',
  base_path: '',
  configured: true,
};

const startedBatch = {
  batch_id: 'gb-e2e-1',
  total_items: 2,
  items: [
    { media_id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63', title: '駭客任務' },
    { media_id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57', title: '星際效應' },
  ],
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

  // Generation-batch dialog on-open calls (9R-16): the recovery status probe
  // (nothing running) + the 缺字幕 preview count.
  await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
    route.fulfill(jsonOk({ running: false, progress: null }))
  );
  await page.route(`${ROUTE_API}/subtitles/generation-batch/preview*`, (route: Route) =>
    route.fulfill(jsonOk({ total_items: 2 }))
  );
}

/** Enter selection mode and open the generation-batch dialog from /library. */
async function openGenerationDialog(page: Page) {
  await page.goto('/library');
  await page.getByTestId('enter-selection-btn').click();
  await expect(page.getByTestId('selection-toolbar')).toBeVisible();
  await page.getByTestId('batch-subtitle-btn').click();
  await expect(page.getByTestId('generation-batch-dialog-v2')).toBeVisible();
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Batch Subtitle Generation UI @ui @batch-subtitle @ux3-subtitle-v2-batch', () => {
  test.beforeEach(async ({ page }) => {
    await stubPopulatedLibrary(page);
  });

  test('[P0] 批次生成字幕 trigger is reachable in the selection toolbar (AC#5)', async ({
    page,
  }) => {
    // GIVEN: a populated library
    await page.goto('/library');

    // WHEN: the user enters selection mode
    await page.getByTestId('enter-selection-btn').click();

    // THEN: the re-pointed batch action is visible with the new label
    await expect(page.getByTestId('selection-toolbar')).toBeVisible();
    const btn = page.getByTestId('batch-subtitle-btn');
    await expect(btn).toBeVisible();
    await expect(btn).toHaveAttribute('aria-label', '批次生成字幕');
  });

  test('[P0] dialog opens idle with scope=missing preselected and the real preview count (AC#1)', async ({
    page,
  }) => {
    // GIVEN/WHEN: the user opens the dialog without any selection
    await openGenerationDialog(page);

    // THEN: the 缺字幕的項目 segment is preselected and carries the preview count
    const missing = page.getByTestId('gen-batch-scope-missing');
    await expect(missing).toHaveAttribute('aria-pressed', 'true');
    await expect(missing).toContainText('2');

    // AND: no selection → the 已選項目 segment is absent (AC#1 presence rule)
    await expect(page.getByTestId('gen-batch-scope-selected')).toHaveCount(0);
  });

  test('[P0] start sends POST {scope:"missing"} and 202 items[] transitions to running (AC#1/3)', async ({
    page,
  }) => {
    // GIVEN: the start endpoint accepts the request (202, 2 items)
    let captured: Request | null = null;
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) => {
      captured = route.request();
      return route.fulfill(jsonStatus(202, startedBatch));
    });
    await openGenerationDialog(page);

    // WHEN: the user starts the batch with the default missing scope
    await page.getByTestId('gen-batch-start-btn').click();

    // THEN: the panel enters the running state — counter + queue rows from items[]
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('0 / 2');
    await expect(
      page.getByTestId('gen-batch-row-5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63')
    ).toBeVisible();
    await expect(
      page.getByTestId('gen-batch-row-8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57')
    ).toBeVisible();

    // AND: the wire body is snake_case {scope:"missing"} with no media_ids (Rule 18)
    expect(captured).not.toBeNull();
    expect(captured!.postDataJSON()).toEqual({ scope: 'missing' });
  });

  test('[P0] library selection ACTUALLY flows — media_ids on the wire (AC#5)', async ({ page }) => {
    // GIVEN: the start endpoint accepts the request
    let captured: Request | null = null;
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) => {
      captured = route.request();
      return route.fulfill(jsonStatus(202, startedBatch));
    });

    // WHEN: the user selects both movies and opens the dialog
    await page.goto('/library');
    await page.getByTestId('enter-selection-btn').click();
    // ux3-cutover-3: v2 poster cards (selection mode shipped in ux3-cutover-2)
    const cards = page.locator('[data-testid^="poster-v2-"]');
    await cards.nth(0).click();
    await cards.nth(1).click();
    await page.getByTestId('batch-subtitle-btn').click();
    await expect(page.getByTestId('generation-batch-dialog-v2')).toBeVisible();

    // THEN: the 已選項目 segment renders preselected with the selection count
    const selected = page.getByTestId('gen-batch-scope-selected');
    await expect(selected).toHaveAttribute('aria-pressed', 'true');
    await expect(selected).toContainText('2');

    // AND: starting sends the selected MOVIE ids as UUID string media_ids
    // (Rule 18 + [@contract-v2])
    await page.getByTestId('gen-batch-start-btn').click();
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('0 / 2');
    expect(captured).not.toBeNull();
    expect(captured!.postDataJSON()).toEqual({
      scope: 'selected',
      media_ids: ['5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63', '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57'],
    });
  });

  test('[P1] 409 TRANSCRIPTION_BATCH_RUNNING recovers to the in-progress snapshot without erroring (AC#1)', async ({
    page,
  }) => {
    // GIVEN: a batch is already running — 409 with the snapshot riding the error body
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) =>
      route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: {
            code: 'TRANSCRIPTION_BATCH_RUNNING',
            message: '已有一個字幕生成批次正在執行',
          },
          data: {
            batch_id: 'gb-existing',
            total_items: 38,
            current_index: 12,
            current_media_id: '9ff0c000-dead-4bee-8f00-000000000999',
            current_item: '正在處理的電影',
            success_count: 11,
            fail_count: 1,
            paused_count: 0,
            status: 'running',
            spent_usd: 0.42,
            budget_usd: 5,
          },
        }),
      })
    );
    await openGenerationDialog(page);

    // WHEN: the user attempts to start another batch
    await page.getByTestId('gen-batch-start-btn').click();

    // THEN: the panel attaches to the running batch (processed = success + fail)
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('12 / 38');
    // The status probe has no items[] — the in-flight fallback card renders.
    await expect(
      page.getByTestId('gen-batch-row-9ff0c000-dead-4bee-8f00-000000000999')
    ).toBeVisible();
    await expect(
      page.getByTestId('gen-batch-row-9ff0c000-dead-4bee-8f00-000000000999')
    ).toContainText('正在處理的電影');
    await expect(page.getByTestId('gen-batch-start-error')).toHaveCount(0);
  });

  test('[P1] lazy SSE — idle dialog opens no EventSource and the page reaches networkidle (§8)', async ({
    page,
  }) => {
    // GIVEN: every request is observed
    const sseRequests: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/events')) sseRequests.push(req.url());
    });

    // WHEN: the dialog is opened and left in the idle state
    await openGenerationDialog(page);
    await expect(page.getByTestId('gen-batch-start-btn')).toBeVisible();

    // THEN: no EventSource connection was attempted on mount (lazy pattern, §8)
    expect(sseRequests).toHaveLength(0);

    // AND: with no open SSE stream, the page settles to networkidle (the exact
    // property eager SSE would break). This is the load-bearing lazy-SSE assertion.
    await page.waitForLoadState('networkidle');
  });

  test('[P2] 全部取消 confirmation fires the real POST /subtitles/generation-batch/cancel (AC#1)', async ({
    page,
  }) => {
    // GIVEN: a batch is started (202 → running)
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) =>
      route.fulfill(jsonStatus(202, startedBatch))
    );
    let cancelHit = false;
    await page.route(`${ROUTE_API}/subtitles/generation-batch/cancel`, (route: Route) => {
      cancelHit = true;
      return route.fulfill(jsonOk({ cancelled: true, running: false }));
    });
    await openGenerationDialog(page);
    await page.getByTestId('gen-batch-start-btn').click();
    await expect(page.getByTestId('gen-batch-cancel-all')).toBeVisible();

    // WHEN: the user cancels and confirms inline
    await page.getByTestId('gen-batch-cancel-all').click();
    await expect(page.getByTestId('gen-batch-cancel-confirm')).toBeVisible();
    await page.getByTestId('gen-batch-cancel-confirm-btn').click();

    // THEN: the cancel endpoint was called (the terminal `cancelled` status
    // arrives via SSE — covered at the hook level, not here).
    await expect.poll(() => cancelHit).toBe(true);
  });
});
