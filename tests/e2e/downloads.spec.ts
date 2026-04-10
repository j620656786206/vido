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

  test('[P2] should display error when qBittorrent is not configured', async ({ page }) => {
    // GIVEN: API returns QB_NOT_CONFIGURED error
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: {
            code: 'QB_NOT_CONFIGURED',
            message: 'qBittorrent 尚未設定',
            suggestion: '請先前往設定頁面設定 qBittorrent 連線。',
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
