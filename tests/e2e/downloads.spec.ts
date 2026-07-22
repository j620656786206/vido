/**
 * Downloads Page E2E UI Tests (Story 4.2)
 *
 * Browser-based tests for the downloads page UI.
 * Uses route interception (network-first) for deterministic tests
 * without requiring a real qBittorrent connection.
 *
 * @tags @downloads @ui
 */

import { test, expect } from '../support/fixtures';

// For page.route() interception, use glob pattern that matches any origin
const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data
// =============================================================================

// =============================================================================
// Downloads Page UI Tests
// =============================================================================

test.describe('Downloads Page @downloads @ui', () => {
  // ux3-cutover-3: the legacy downloads page is deleted — list/expand/sort and
  // the NOT_CONFIGURED error display are covered on the v2 page by
  // downloads-v2.spec.ts (AC1/AC2 cards, D6 qbt-error fail-soft, sort/table).
  // Only shared-semantics tests remain here.
  // bugfix-10-2: useDownloads now gates on useQBittorrentConfig().data?.configured.
  // Without this default mock, fresh-DB CI environments resolve configured: false
  // and the gate suppresses the downloads requests these tests rely on, breaking
  // every assertion below. Individual tests can override via a per-test page.route
  // (last-registered wins).
  test.beforeEach(async ({ page }) => {
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            host: 'http://localhost:8080',
            username: 'admin',
            basePath: '',
            configured: true,
          },
        }),
      })
    );
  });

  test('[P2] should display empty state when no downloads exist', async ({ page }) => {
    // GIVEN: API returns empty download list
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { items: [], page: 1, pageSize: 100, totalItems: 0, totalPages: 0 },
        }),
      })
    );

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Empty state message is shown
    await expect(page.getByText('目前沒有下載任務')).toBeVisible();
    await expect(page.getByText('在 qBittorrent 中新增種子後會自動顯示')).toBeVisible();
  });
});

// =============================================================================
// qBittorrent Config Gate (bugfix-10-2 AC#7)
// =============================================================================
//
// Browser-level guard for the AC#7 contract: when useQBittorrentConfig reports
// configured !== true, the useDownloads / useDownloadCounts / useDownloadDetails
// hooks MUST NOT issue any /api/v1/downloads* request. Unit-level coverage
// already exists in apps/web/src/hooks/useDownloads.spec.ts; this E2E gives the
// same invariant a deterministic browser proof so future regressions surface in
// the Network tab the way they would in production.
// =============================================================================

test.describe('Downloads qBittorrent config gate (bugfix-10-2) @downloads @ui', () => {
  test('[P1] (AC#7) should NOT request /api/v1/downloads when qBittorrent is unconfigured', async ({
    page,
  }) => {
    // GIVEN: collect every /api/v1/downloads* request the page issues
    const downloadRequests: string[] = [];
    page.on('request', (req) => {
      const url = req.url();
      if (/\/api\/v1\/downloads(\b|\/|\?|$)/.test(url)) {
        downloadRequests.push(url);
      }
    });

    // GIVEN: gate signal resolves to configured: false (qBT not yet set up)
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { host: '', username: '', basePath: '', configured: false },
        }),
      })
    );

    // GIVEN: any downloads request that DOES leak through is captured by a
    //        catch-all route that records the URL and 503s — this surfaces gate
    //        leaks as both an array entry AND a visible error in the test trace.
    await page.route(`${ROUTE_API}/downloads*`, (route) => {
      downloadRequests.push(`LEAKED:${route.request().url()}`);
      return route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'QBITTORRENT_NOT_CONFIGURED', message: 'leak', suggestion: 'leak' },
        }),
      });
    });

    // WHEN: navigating to /downloads and waiting for the config response to land
    //       (synchronization barrier — TanStack Query has had its reactive cycle)
    const configResponse = page.waitForResponse('**/settings/qbittorrent');
    await page.goto('/downloads');
    await configResponse;

    // WHEN: page rendering settles (no in-flight requests for 500ms)
    await page.waitForLoadState('networkidle');

    // THEN: the v2 page rendered (proves React processed the closed-gate state)
    await expect(page.getByTestId('downloads-browse-v2')).toBeVisible();

    // THEN: zero /api/v1/downloads* requests were issued — gate held closed
    //       across mount + render + TanStack Query reactive cycle
    expect(downloadRequests).toEqual([]);
  });

  test('[P1] (AC#7) should request /api/v1/downloads after qBittorrent becomes configured', async ({
    page,
  }) => {
    // GIVEN: collect every /api/v1/downloads* request to prove the gate is reversible
    const downloadRequests: string[] = [];
    page.on('request', (req) => {
      if (/\/api\/v1\/downloads(\b|\/|\?|$)/.test(req.url())) {
        downloadRequests.push(req.url());
      }
    });

    // GIVEN: gate is OPEN (configured: true) — the inverse of the closed-gate test
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            host: 'http://localhost:8080',
            username: 'admin',
            basePath: '',
            configured: true,
          },
        }),
      })
    );

    // GIVEN: backend returns an empty list so the page reaches its rendered state
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { items: [], page: 1, pageSize: 100, totalItems: 0, totalPages: 0 },
        }),
      })
    );

    // WHEN: navigating to /downloads (gate open → hooks should fire)
    await page.goto('/downloads');
    await expect(page.getByText('目前沒有下載任務')).toBeVisible();

    // THEN: at least one /api/v1/downloads* request fired — gate transition works
    expect(downloadRequests.length).toBeGreaterThan(0);
  });
});
