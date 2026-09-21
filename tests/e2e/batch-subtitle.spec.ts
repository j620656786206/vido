/**
 * Batch Subtitle GENERATION UI E2E Tests (Story ux3-subtitle-v2-batch;
 * re-pointed by sub-4-3 to the COST-CONSENT flow)
 *
 * sub-4-3 replaced the dialog's idle branch: opening 產生字幕 now enters the
 * consent flow (F14 analyze → F15 candidate list → F16/F19 confirm) and the
 * batch ALWAYS starts as {scope:"selected", media_ids, budget_usd} — the old
 * scope=missing idle journeys are unreachable. This suite covers the
 * wire-level integration the unit mocks hide:
 *
 *   /library → SelectionToolbar → GenerationConsentView → ConfirmGenerationDialog
 *                                                           ↓
 *              GET /api/v1/subtitles/generation-candidates (state envelope)
 *              POST /api/v1/subtitles/generation-batch  (202 / 409 / cancel)
 *
 * Coverage (vs the unit specs):
 *   - [P0] Real selection-toolbar wiring: 批次生成字幕 reachable on /library
 *   - [P0] Consent list from a ready snapshot: default extract-only selection,
 *          honest per-item amounts (§5-sexies — no 免費)
 *   - [P0] Confirm sends {scope:"selected", media_ids, budget_usd} (Rule 18)
 *          and the 202 items[] transitions to the F8 running panel
 *   - [P0] Selection ACTUALLY flows: preselected ∩ candidates on the wire
 *   - [P1] 409 TRANSCRIPTION_BATCH_RUNNING recovers to the snapshot
 *   - [P1] Lazy SSE: a ready-snapshot consent open needs NO EventSource;
 *          page hits networkidle (§8)
 *   - [P2] 全部取消 confirm fires POST /subtitles/generation-batch/cancel
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
import type { Request } from '@playwright/test';
import {
  ROUTE_API,
  jsonOk,
  jsonStatus,
  startedBatch,
  stubPopulatedLibrary,
  confirmAndStart,
  openGenerationDialog,
} from '../support/helpers/batch-subtitle-stubs';

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

  test('[P0] opening 產生字幕 renders the consent list with default extract selection and honest amounts (sub-4-3 AC#2)', async ({
    page,
  }) => {
    // GIVEN/WHEN: the user opens the flow without any selection
    await openGenerationDialog(page);

    // THEN: both extract candidates render with their VERBATIM estimates
    // (§5-sexies: the word 免費 must not appear anywhere)
    await expect(
      page.getByTestId('consent-row-usd-5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63')
    ).toHaveText('$0.05');
    await expect(
      page.getByTestId('consent-row-usd-8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57')
    ).toHaveText('$0.04');
    await expect(page.getByTestId('generation-consent-view')).not.toContainText('免費');

    // AND: default selection = every extract candidate → summary/footer agree
    await expect(page.getByTestId('consent-summary-usd')).toHaveText('$0.09');
    await expect(page.getByTestId('consent-footer-usd')).toHaveText('$0.09');
  });

  test('[P0] confirm sends POST {scope:"selected", media_ids, budget_usd} and 202 items[] transitions to running (sub-4-3 AC#4)', async ({
    page,
  }) => {
    // GIVEN: the start endpoint accepts the request (202, 2 items)
    let captured: Request | null = null;
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) => {
      captured = route.request();
      return route.fulfill(jsonStatus(202, startedBatch));
    });
    await openGenerationDialog(page);

    // WHEN: the user walks the consent flow with the defaults (extract-only
    // selection, prefilled $5.00 ceiling) and confirms
    await confirmAndStart(page);

    // THEN: the F8 panel enters the running state — counter + queue rows
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('0 / 2');
    await expect(
      page.getByTestId('gen-batch-row-5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63')
    ).toBeVisible();
    await expect(
      page.getByTestId('gen-batch-row-8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57')
    ).toBeVisible();

    // AND: clicking OUTSIDE does NOT close a running batch (dsr-6d-b AC #7 —
    // only a real browser dispatches Radix's outside-pointer event, so this
    // assertion cannot live in the jsdom spec).
    await page.mouse.click(5, 5);
    await expect(page.getByTestId('generation-batch-dialog-v2')).toBeVisible();

    // AND: the wire body is snake_case — CONSENTED ids in list order + the
    // WYSIWYG on-screen ceiling (Rule 18 + 9R-16 AC#1 [@contract-v3])
    expect(captured).not.toBeNull();
    expect(captured!.postDataJSON()).toEqual({
      scope: 'selected',
      media_ids: ['5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63', '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57'],
      budget_usd: 5,
    });
  });

  test('[P0] library selection ACTUALLY flows — media_ids on the wire (AC#5)', async ({ page }) => {
    // GIVEN: the start endpoint accepts the request
    let captured: Request | null = null;
    await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) => {
      captured = route.request();
      return route.fulfill(jsonStatus(202, startedBatch));
    });

    // WHEN: the user selects both movies and opens the consent flow
    await page.goto('/library');
    await page.getByTestId('enter-selection-btn').click();
    // ux3-cutover-3: v2 poster cards (selection mode shipped in ux3-cutover-2)
    const cards = page.locator('[data-testid^="poster-v2-"]');
    await cards.nth(0).click();
    await cards.nth(1).click();
    await page.getByTestId('batch-subtitle-btn').click();
    await expect(page.getByTestId('consent-candidate-list')).toBeVisible();

    // THEN: the preselected ∩ candidates selection is checked (both rows)
    await expect(page.getByTestId('consent-summary-usd')).toHaveText('$0.09');

    // AND: confirming sends the consented UUID string media_ids + budget_usd
    // (Rule 18 + [@contract-v3])
    await confirmAndStart(page);
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('0 / 2');
    expect(captured).not.toBeNull();
    expect(captured!.postDataJSON()).toEqual({
      scope: 'selected',
      media_ids: ['5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63', '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57'],
      budget_usd: 5,
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
          // dsr-6d-a: the 409 body carries the running batch's OWN queue, so
          // the panel renders real rows instead of the degraded single card.
          // Counts and items[] must agree — the backend's snapshot always does.
          data: {
            batch_id: 'gb-existing',
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
            items: [
              {
                media_id: '7aa1c000-0000-4bee-8f00-000000000001',
                title: '已完成的電影',
                media_type: 'movie',
                series_title: '',
                status: 'done',
                reason: '',
              },
              {
                media_id: '7aa1c000-0000-4bee-8f00-000000000002',
                title: '別處在處理的電影',
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
            ],
          },
        }),
      })
    );
    await openGenerationDialog(page);

    // WHEN: the user walks the consent flow into a start attempt
    await confirmAndStart(page);

    // THEN: the panel attaches to the running batch (processed = success + fail)
    await expect(page.getByTestId('gen-batch-counter')).toHaveText('2 / 3');
    // dsr-6d-b: the row now comes from the 409 body's items[].
    await expect(
      page.getByTestId('gen-batch-row-9ff0c000-dead-4bee-8f00-000000000999')
    ).toBeVisible();
    await expect(
      page.getByTestId('gen-batch-row-9ff0c000-dead-4bee-8f00-000000000999')
    ).toContainText('正在處理的電影');
    // dsr-6d-b: the refused row says WHY, instead of being drawn as 完成.
    await expect(
      page.getByTestId('gen-batch-row-7aa1c000-0000-4bee-8f00-000000000002')
    ).toContainText('這部正在別處處理');
    await expect(page.getByTestId('consent-start-error')).toHaveCount(0);
  });

  test('[P1] lazy SSE — idle dialog opens no EventSource and the page reaches networkidle (§8)', async ({
    page,
  }) => {
    // GIVEN: every request is observed
    const sseRequests: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/events')) sseRequests.push(req.url());
    });

    // WHEN: the consent flow is opened from a READY snapshot (no analyzing
    // phase → no SSE need) and left on the F15 list
    await openGenerationDialog(page);
    await expect(page.getByTestId('consent-start-btn')).toBeVisible();

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
    await confirmAndStart(page);
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
