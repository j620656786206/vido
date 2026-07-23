/**
 * Scan Progress Card — SSE-driven E2E (migrated out of TestSprite round-2)
 *
 * WHY THIS FILE EXISTS
 * --------------------
 * Three TestSprite cases (TC064 / TC067 / TC070) exercised the transient,
 * SSE-driven `ScanProgressCard`. On the local seeded env they were flaky-red:
 * a scan of the 15-byte stub files completes in <100 ms, so the card never
 * stays on screen long enough for the cloud browser to observe. That is a
 * TIMING artifact of the seed, NOT a product defect — the right home is a
 * Playwright spec where the scan-progress SSE stream is mocked and every
 * frame is driven deterministically. This file is that home.
 *
 * WHAT IS MOCKED  (the exact mechanism the app consumes)
 * ------------------------------------------------------
 * `useScanProgress` (apps/web/src/hooks/useScanProgress.ts) opens a native
 * `EventSource` at GET /api/v1/events (via `scannerService.getSSEUrl()`) and
 * listens for the NAMED SSE events: `connected`, `scan_progress`,
 * `scan_complete`, `scan_cancelled`, `ping`. Each `scan_progress` frame's
 * `e.data` is `JSON.parse`-d, then `event.data ?? event` is run through
 * `snakeToCamel`, then dispatched into the card's reducer.
 *
 * Rather than fight `route.fulfill`'s "whole body at once" limitation (which
 * makes progress→complete sequencing impossible), we replace `window.EventSource`
 * with a controllable mock via `addInitScript` and expose a `window.__scanSSE`
 * bridge. The test drives the exact event timeline through `page.evaluate`.
 * Wire payloads are snake_case (fetchApi/snakeToCamel converts them).
 *
 * WIRING (fixed in this change — bugfix-scan-progress-sse-unwired)
 * ---------------------------------------------------------------
 * The shell's `<ScanProgress>` mounts `useScanProgress()` but the SSE is opened
 * lazily via `startTracking()`. The 掃描媒體庫 trigger lives in a SEPARATE hook
 * instance (ScannerSettings) that cannot reach the shell instance directly, so
 * on scan-trigger success it broadcasts a module-level signal
 * (`requestScanTracking()`); the shell's `<ScanProgress>` subscribes
 * (`subscribeScanTracking`) and opens its SSE in response. This stays lazy (no
 * connect-on-mount → the app's networkidle-based E2E stability is preserved) and
 * is what `waitForScanSSE()` below asserts. Before this fix the card was
 * unreachable from the UI (the trigger never signalled the shell hook).
 *
 * Real testids used (from ScanProgressCard.tsx):
 *   scan-progress-card · scan-progress-pill · scan-minimize-btn · scan-close-btn
 *   scan-cancel-btn · cancel-confirm-dialog · cancel-continue-btn ·
 *   cancel-confirm-btn · scan-dismiss-btn · view-unmatched-link · scan-progress-bar
 *   (trigger: scan-trigger-button on /settings/scanner, label 掃描媒體庫)
 *
 * @tags @e2e @scanner @scan-progress @migrated-testsprite
 */

import { test, expect, type Route } from '../support/fixtures';
import type { Page } from '@playwright/test';

const ROUTE_API = '**/api/v1';

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

// ScanStatus wire shape (snake_case) — the page's loading gate needs status +
// schedule + libraries resolved before `scanner-settings` renders. is_scanning
// stays false: the card's visibility is driven by SSE, never by this poll.
const idleStatus = {
  is_scanning: false,
  files_found: 0,
  files_processed: 0,
  current_file: '',
  percent_done: 0,
  error_count: 0,
  estimated_time: '',
  last_scan_at: '2026-07-20T09:00:00Z',
  last_scan_files: 128,
  last_scan_duration: '2m 14s',
};

const scanResult = { files_found: 0, files_new: 0, errors: 0, duration: '0s' };

// =============================================================================
// Controllable scan-progress SSE — installs a mock EventSource + a test bridge.
// Must be registered (addInitScript) BEFORE navigation so the app's first
// `new EventSource(...)` gets the mock.
// =============================================================================
async function installScanSSE(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const instances: any[] = [];
    class MockEventSource {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSED = 2;
      url: string;
      readyState = 0;
      onopen: ((e: unknown) => void) | null = null;
      onerror: ((e: unknown) => void) | null = null;
      private listeners: Record<string, Array<(e: unknown) => void>> = {};
      constructor(url: string) {
        this.url = String(url);
        instances.push(this);
        // Attach happens synchronously right after construction in connectSSE();
        // flip to OPEN on the next microtask to mirror a real handshake.
        void Promise.resolve().then(() => {
          this.readyState = 1;
          this.onopen?.({});
        });
      }
      addEventListener(type: string, cb: (e: unknown) => void) {
        (this.listeners[type] ||= []).push(cb);
      }
      removeEventListener(type: string, cb: (e: unknown) => void) {
        this.listeners[type] = (this.listeners[type] || []).filter((f) => f !== cb);
      }
      close() {
        this.readyState = 2;
      }
      __dispatch(type: string, data: unknown) {
        const evt = { type, data: typeof data === 'string' ? data : JSON.stringify(data) };
        (this.listeners[type] || []).forEach((cb) => cb(evt));
        const inline = (this as unknown as Record<string, unknown>)['on' + type];
        if (typeof inline === 'function') (inline as (e: unknown) => void)(evt);
      }
    }
    // @ts-expect-error — override the platform EventSource for the whole page.
    window.EventSource = MockEventSource;
    const live = () =>
      [...instances].reverse().find((e) => e.url.includes('/events') && e.readyState !== 2);
    (window as unknown as Record<string, unknown>).__scanSSE = {
      hasConnection: () => Boolean(live()),
      emit: (type: string, data: unknown) => {
        const es = live();
        if (!es) throw new Error('no open scan EventSource — startTracking() never fired');
        es.__dispatch(type, data);
      },
    };
  });
}

/** Emit a NAMED scan SSE event on the live connection, from the browser context. */
async function emitScan(
  page: Page,
  type: string,
  data: Record<string, unknown> = {}
): Promise<void> {
  await page.evaluate(
    ([t, d]) =>
      (
        window as unknown as { __scanSSE: { emit: (t: string, d: unknown) => void } }
      ).__scanSSE.emit(t as string, { data: d }),
    [type, data] as const
  );
}

/** Block until the app has opened the scan SSE (the wiring-gated step, see header). */
async function waitForScanSSE(page: Page): Promise<void> {
  await expect
    .poll(
      () =>
        page.evaluate(
          () =>
            (
              window as unknown as { __scanSSE?: { hasConnection: () => boolean } }
            ).__scanSSE?.hasConnection() ?? false
        ),
      {
        message:
          'scan SSE never opened — the scan trigger must call startTracking() (see file header: known wiring gap)',
        timeout: 10_000,
      }
    )
    .toBe(true);
}

// =============================================================================
// Baseline stubs so /settings/scanner renders past its loading gate.
// =============================================================================
async function stubScannerPage(page: Page): Promise<void> {
  await page.route(`${ROUTE_API}/setup/status`, (route: Route) =>
    route.fulfill(jsonOk({ needs_setup: false }))
  );
  await page.route(`${ROUTE_API}/scanner/status`, (route: Route) =>
    route.fulfill(jsonOk(idleStatus))
  );
  await page.route(`${ROUTE_API}/scanner/schedule`, (route: Route) =>
    route.fulfill(jsonOk({ frequency: 'manual' }))
  );
  await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
    route.fulfill(jsonOk({ libraries: [] }))
  );
  await page.route(`${ROUTE_API}/scanner/scan`, (route: Route) =>
    route.fulfill(jsonOk(scanResult))
  );
  await page.route(`${ROUTE_API}/scanner/cancel`, (route: Route) =>
    route.fulfill(jsonOk({ cancelled: true }))
  );
}

/** Navigate to the scanner settings, trigger a scan, and bring the card up via SSE. */
async function startScanAndShowCard(page: Page): Promise<void> {
  await page.goto('/settings/scanner');
  await expect(page.getByTestId('scanner-settings')).toBeVisible({ timeout: 15_000 });

  // TC064 step 1 — trigger a manual scan.
  await page.getByTestId('scan-trigger-button').click();

  // The scan flow must open the SSE (wiring gate — see header).
  await waitForScanSSE(page);

  // Server handshake + first progress frame → the card becomes visible.
  await emitScan(page, 'connected', {});
  await emitScan(page, 'scan_progress', {
    files_found: 640,
    files_processed: 128,
    current_file: '/media/movies/Interstellar.2014.mkv',
    percent_done: 20,
    error_count: 0,
    estimated_time: '3 分',
  });
}

// =============================================================================
// Tests
// =============================================================================
test.describe('Scan Progress Card — SSE-driven @e2e @scanner @scan-progress', () => {
  test.beforeEach(async ({ page }) => {
    await installScanSSE(page);
    await stubScannerPage(page);
  });

  test('[P0] TC064 — triggering a manual scan shows the progress card (SSE-driven)', async ({
    page,
  }) => {
    await startScanAndShowCard(page);

    // THEN: the floating card is on screen with the live progress frame.
    const card = page.getByTestId('scan-progress-card');
    await expect(card).toBeVisible();
    await expect(card).toContainText('媒體庫掃描中');
    await expect(card).toContainText('20%');
    await expect(page.getByTestId('scan-progress-bar')).toHaveAttribute('style', /width:\s*20%/);

    // AND: a later frame updates the same card in place (no reload).
    await emitScan(page, 'scan_progress', {
      files_found: 640,
      files_processed: 384,
      current_file: '/media/tv/Foundation.S01E03.mkv',
      percent_done: 60,
      error_count: 0,
      estimated_time: '1 分',
    });
    await expect(card).toContainText('60%');
  });

  test('[P1] TC065-analog — minimize collapses to the pill and expanding restores the card', async ({
    page,
  }) => {
    await startScanAndShowCard(page);
    await expect(page.getByTestId('scan-progress-card')).toBeVisible();

    // WHEN: the user minimizes the card
    await page.getByTestId('scan-minimize-btn').click();

    // THEN: it collapses to the pill (which carries the live percentage) and the
    // full card is gone.
    const pill = page.getByTestId('scan-progress-pill');
    await expect(pill).toBeVisible();
    await expect(pill).toContainText('掃描中 20%');
    await expect(page.getByTestId('scan-progress-card')).toHaveCount(0);

    // AND: clicking the pill expands back to the full card.
    await pill.click();
    await expect(page.getByTestId('scan-progress-card')).toBeVisible();
    await expect(page.getByTestId('scan-progress-pill')).toHaveCount(0);
  });

  test('[P0] TC067 — cancel-confirm can be dismissed to continue scanning', async ({ page }) => {
    await startScanAndShowCard(page);
    const card = page.getByTestId('scan-progress-card');
    await expect(card).toBeVisible();

    // WHEN: the user opens the cancel-confirmation
    await page.getByTestId('scan-cancel-btn').click();
    const confirm = page.getByTestId('cancel-confirm-dialog');
    await expect(confirm).toBeVisible();
    await expect(confirm).toContainText('確定要取消掃描嗎');

    // AND: dismisses it with 繼續掃描
    await page.getByTestId('cancel-continue-btn').click();

    // THEN: the confirmation is gone and the scan card is still in the active
    // scanning state (dismiss did NOT cancel — no completion/cancelled summary).
    await expect(page.getByTestId('cancel-confirm-dialog')).toHaveCount(0);
    await expect(card).toBeVisible();
    await expect(card).toContainText('媒體庫掃描中');
    await expect(card).not.toContainText('掃描已取消');

    // AND: further progress still flows into the same card.
    await emitScan(page, 'scan_progress', {
      files_found: 640,
      files_processed: 512,
      current_file: '/media/movies/Dune.2021.mkv',
      percent_done: 80,
      error_count: 0,
      estimated_time: '30 秒',
    });
    await expect(card).toContainText('80%');
  });

  test('[P0] TC070 — scan completion shows the summary card with results', async ({ page }) => {
    await startScanAndShowCard(page);
    await expect(page.getByTestId('scan-progress-card')).toBeVisible();

    // WHEN: the backend signals completion over SSE
    await emitScan(page, 'scan_complete', { files_found: 640, error_count: 2 });

    // THEN: the card flips to the completion summary (same testid, new content)
    const card = page.getByTestId('scan-progress-card');
    await expect(card).toContainText('掃描完成');
    await expect(card).toContainText('640');
    await expect(page.getByTestId('view-unmatched-link')).toBeVisible();

    // AND: the auto-dismiss affordance + manual dismiss are present.
    await expect(page.getByTestId('auto-dismiss-bar')).toBeVisible();
    await page.getByTestId('scan-dismiss-btn').click();
    await expect(page.getByTestId('scan-progress-card')).toHaveCount(0);
  });
});
