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
import { presetDownloads } from '../support/fixtures/factories/download-factory';

// For page.route() interception, use glob pattern that matches any origin
const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data
// =============================================================================

const mockDownloadList = [
  presetDownloads.downloading,
  presetDownloads.completed,
  presetDownloads.seeding,
  presetDownloads.paused,
];

const mockDownloadDetails = {
  ...presetDownloads.downloading,
  pieceSize: 4194304,
  comment: 'Downloaded via torrent',
  createdBy: 'qBittorrent v4.5.2',
  creationDate: '2026-01-10T08:00:00Z',
  totalWasted: 1024,
  timeElapsed: 3600,
  seedingTime: 0,
  avgDownSpeed: 8388608,
  avgUpSpeed: 262144,
};

// =============================================================================
// Downloads Page UI Tests
// =============================================================================

test.describe('Downloads Page @downloads @ui', () => {
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
          data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true },
        }),
      })
    );
  });

  test('[P1] should display torrent list with name, progress, speed, and status (AC1)', async ({
    page,
  }) => {
    // GIVEN: API returns a list of downloads
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            items: mockDownloadList,
            page: 1,
            pageSize: 100,
            totalItems: mockDownloadList.length,
            totalPages: 1,
          },
        }),
      })
    );

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Page title is displayed
    await expect(page.locator('h1')).toHaveText('下載管理');

    // THEN: Torrent names are displayed
    await expect(page.getByText('[SubGroup] Movie Name (2026) [1080p].mkv')).toBeVisible();
    await expect(page.getByText('Series S01 Complete [720p]')).toBeVisible();
    await expect(page.getByText('Documentary (2025) [4K]')).toBeVisible();

    // THEN: Progress bars are rendered
    const progressBars = page.getByRole('progressbar');
    await expect(progressBars).toHaveCount(4);

    // THEN: Item count is shown
    await expect(page.getByText('4 個項目')).toBeVisible();
  });

  test('[P1] should expand torrent details on click (AC4)', async ({ page }) => {
    // GIVEN: API returns downloads and detail data
    await page.route(`${ROUTE_API}/downloads?*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            items: mockDownloadList,
            page: 1,
            pageSize: 100,
            totalItems: mockDownloadList.length,
            totalPages: 1,
          },
        }),
      })
    );

    const detailHash = presetDownloads.downloading.hash;
    await page.route(`${ROUTE_API}/downloads/${detailHash}`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockDownloadDetails }),
      })
    );

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');
    await expect(page.getByText('[SubGroup] Movie Name (2026) [1080p].mkv')).toBeVisible();

    // WHEN: Clicking on the first download item
    await page.getByText('[SubGroup] Movie Name (2026) [1080p].mkv').click();

    // THEN: Detail fields are displayed (AC4)
    await expect(page.getByTestId('download-details')).toBeVisible();
    await expect(page.getByText('儲存路徑')).toBeVisible();
    await expect(page.getByText('/downloads/movies')).toBeVisible();
    await expect(page.getByText('做種數')).toBeVisible();
    await expect(page.getByText('平均下載速度')).toBeVisible();
  });

  test('[P2] should sort torrents when sort dropdown changes (AC5)', async ({ page }) => {
    // GIVEN: Track API calls to verify sort parameters
    const apiCalls: string[] = [];

    await page.route(`${ROUTE_API}/downloads*`, (route) => {
      apiCalls.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            items: mockDownloadList,
            page: 1,
            pageSize: 100,
            totalItems: mockDownloadList.length,
            totalPages: 1,
          },
        }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');
    await expect(page.getByText('4 個項目')).toBeVisible();

    // WHEN: Changing sort dropdown to "name"
    await page.selectOption('#sort-select', 'name');

    // THEN: API is called with new sort parameter
    await expect(async () => {
      const nameCall = apiCalls.find((url) => url.includes('sort=name'));
      expect(nameCall).toBeTruthy();
    }).toPass({ timeout: 5000 });
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

  test('[P2] should display error when qBittorrent returns NOT_CONFIGURED 503 (bugfix-10-2 [@contract-v1] AC#1, #3)', async ({
    page,
  }) => {
    // GIVEN: qBT config gate is open (configured: true) — required after bugfix-10-2
    //        so the useDownloads hook actually fires and exercises the error path
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true },
        }),
      })
    );

    // GIVEN: Backend returns NOT_CONFIGURED with the new contract status (503, not 400)
    //        and the SETUP_REQUIRED suffix on suggestion (AC#3 programmatic frontend marker)
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: {
            code: 'QBITTORRENT_NOT_CONFIGURED',
            message: 'qBittorrent 尚未設定',
            suggestion: '請先設定 qBittorrent 連線。SETUP_REQUIRED',
          },
        }),
      })
    );

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Error message is displayed
    await expect(page.getByText('無法載入下載清單')).toBeVisible();
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

    // THEN: page header rendered (proves React processed the closed-gate state)
    await expect(page.locator('h1')).toHaveText('下載管理');

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
          data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true },
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
