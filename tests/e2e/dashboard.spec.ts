/**
 * Dashboard E2E Tests (Story 4.3)
 *
 * Browser-based tests for the unified download dashboard.
 * Uses route interception for deterministic tests
 * without requiring real qBittorrent or media database.
 *
 * @tags @dashboard @ui
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// For page.route() interception, use glob pattern that matches any origin
// (frontend uses relative URLs like /api/v1/... which resolve to the dev server origin)
const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data
// =============================================================================

const mockDownloads = [
  {
    hash: 'abc123',
    name: '[SubGroup] Movie Name (2024) [1080p]',
    size: 4294967296,
    progress: 0.85,
    downloadSpeed: 1048576,
    uploadSpeed: 0,
    eta: 300,
    status: 'downloading',
    addedOn: '2026-02-10T10:00:00Z',
    seeds: 5,
    peers: 3,
    downloaded: 3650722201,
    uploaded: 0,
    ratio: 0,
    savePath: '/downloads',
  },
  {
    hash: 'def456',
    name: 'Another Download [720p]',
    size: 1073741824,
    progress: 1.0,
    downloadSpeed: 0,
    uploadSpeed: 0,
    eta: 0,
    status: 'completed',
    addedOn: '2026-02-10T08:00:00Z',
    seeds: 0,
    peers: 0,
    downloaded: 1073741824,
    uploaded: 0,
    ratio: 0,
    savePath: '/downloads',
  },
];

const mockRecentMedia = [
  {
    id: 'movie-1',
    title: '測試電影',
    year: 2024,
    posterUrl: '',
    mediaType: 'movie',
    justAdded: true,
    addedAt: '2026-02-10T10:00:00Z',
  },
  {
    id: 'series-1',
    title: '測試影集',
    year: 2023,
    posterUrl: '',
    mediaType: 'tv',
    justAdded: false,
    addedAt: '2026-02-10T09:00:00Z',
  },
];

const mockQBConfig = {
  host: 'http://localhost:8080',
  username: 'admin',
  basePath: '',
  configured: true,
};

// =============================================================================
// Dashboard Layout Tests
// =============================================================================

test.describe('Dashboard Layout @dashboard @ui', () => {
  test.beforeEach(async ({ page }) => {
    // Mock all API endpoints
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockDownloads }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockRecentMedia }),
      })
    );

    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );
  });

  test('[P1] should display dashboard with download panel and recent media (AC1)', async ({
    page,
  }) => {
    await page.goto('/');

    // THEN: Dashboard layout is visible
    await expect(page.getByTestId('dashboard-layout')).toBeVisible();
    await expect(page.getByTestId('download-panel')).toBeVisible();
    await expect(page.getByTestId('recent-media-panel')).toBeVisible();
    await expect(page.getByTestId('search-form')).toBeVisible();
  });

  test('[P1] should display downloads in compact view (AC1)', async ({ page }) => {
    await page.goto('/');

    // THEN: Download items are shown
    await expect(page.getByText('[SubGroup] Movie Name (2024) [1080p]')).toBeVisible();
    await expect(page.getByText('Another Download [720p]')).toBeVisible();
  });

  test('[P1] should display recent media with titles (AC1)', async ({ page }) => {
    await page.goto('/');

    // THEN: Recent media items are shown
    await expect(page.getByText('測試電影')).toBeVisible();
    await expect(page.getByText('測試影集')).toBeVisible();
  });

  // [Downgraded to unit] badges, links → RecentMediaPanel.spec.tsx, DownloadPanel.spec.tsx
});

// =============================================================================
// Disconnected State Tests
// =============================================================================

test.describe('Dashboard Disconnected State @dashboard @ui', () => {
  test('[P1] should show disconnected state when qBittorrent not configured (AC3)', async ({
    page,
  }) => {
    // Mock qBittorrent as not configured
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

    // Mock downloads to fail
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    // Mock recent media to still work
    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockRecentMedia }),
      })
    );

    await page.goto('/');

    // THEN: Download panel shows disconnected, but recent media still works
    await expect(page.getByText('qBittorrent 未連線')).toBeVisible();
    await expect(page.getByText('測試電影')).toBeVisible();
  });
});

// =============================================================================
// Mobile Responsive Tests
// =============================================================================

test.describe('Dashboard Mobile Layout @dashboard @ui', () => {
  test.beforeEach(async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 812 });

    // Mock all API endpoints
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockDownloads }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockRecentMedia }),
      })
    );

    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );
  });

  test('[P1] should stack panels vertically on mobile (AC4)', async ({ page }) => {
    await page.goto('/');

    // THEN: Both panels should be visible (stacked)
    await expect(page.getByTestId('download-panel')).toBeVisible();
    await expect(page.getByTestId('recent-media-panel')).toBeVisible();

    // Grid should be single column (stacked)
    const grid = page.getByTestId('dashboard-grid');
    const gridStyle = await grid.evaluate((el) => window.getComputedStyle(el).gridTemplateColumns);
    // On mobile, grid-cols-1 means single column
    expect(gridStyle).not.toContain('400px');
  });
});

// =============================================================================
// Quick Search Tests
// =============================================================================

test.describe('Dashboard Quick Search @dashboard @ui', () => {
  test.beforeEach(async ({ page }) => {
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );
  });

  // [Downgraded to unit] search input, placeholder → QuickSearchBar.spec.tsx
});

// =============================================================================
// Quick Actions Hover Tests (AC5)
// =============================================================================

test.describe('Dashboard Quick Actions @dashboard @ui', () => {
  test.beforeEach(async ({ page }) => {
    // Network-first: intercept all routes BEFORE navigation
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockDownloads }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockRecentMedia }),
      })
    );

    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );
  });

  test('[P1] should render detail link for each download item (AC5)', async ({ page }) => {
    // GIVEN: Dashboard is loaded with downloads
    await page.goto('/');
    await expect(page.getByText('[SubGroup] Movie Name (2024) [1080p]')).toBeVisible();

    // THEN: Quick action link exists with correct aria-label and destination
    const detailLink = page.getByLabel('查看 [SubGroup] Movie Name (2024) [1080p] 詳情');
    await expect(detailLink).toBeAttached();
    await expect(detailLink).toHaveAttribute('href', /\/downloads/);
  });

  test('[P1] should show media card as a clickable link (AC5)', async ({ page }) => {
    // GIVEN: Dashboard is loaded with recent media
    await page.goto('/');
    await expect(page.getByText('測試電影')).toBeVisible();

    // THEN: Media title is within a link
    const mediaLink = page.getByText('測試電影').locator('xpath=ancestor::a');
    await expect(mediaLink).toBeVisible();
  });
});

// =============================================================================
// Panel Error Independence Tests (AC3 / NFR-R12)
// =============================================================================

test.describe('Dashboard Panel Independence @dashboard @ui', () => {
  test('[P1] should show downloads when recent media API fails (AC3)', async ({ page }) => {
    // Network-first: intercept all routes BEFORE navigation
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );

    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockDownloads }),
      })
    );

    // Mock recent media API to return error
    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'DB_QUERY_FAILED', message: 'Database error' },
        }),
      })
    );

    // GIVEN: Dashboard loads
    await page.goto('/');

    // THEN: Download panel still works independently
    await expect(page.getByText('[SubGroup] Movie Name (2024) [1080p]')).toBeVisible();
    await expect(page.getByTestId('download-panel')).toBeVisible();
  });
});

// =============================================================================
// Quick Search Navigation Tests
// =============================================================================

test.describe('Dashboard Search Navigation @dashboard @ui', () => {
  test.beforeEach(async ({ page }) => {
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );
  });

  test('[P1] should navigate to search page on form submit', async ({ page }) => {
    // GIVEN: User is on the dashboard
    await page.goto('/');
    const input = page.getByPlaceholder('搜尋媒體庫...');
    await expect(input).toBeVisible();

    // WHEN: User types a query and presses Enter
    await input.fill('鬼滅之刃');
    await input.press('Enter');

    // THEN: URL changes to search page with query parameter
    await page.waitForURL(/\/search/);
    expect(page.url()).toContain('/search');
  });
});

// =============================================================================
// Empty Dashboard State Tests
// =============================================================================

test.describe('Dashboard Empty State @dashboard @ui', () => {
  test('[P2] should show empty states when connected but no data (AC1, AC3)', async ({ page }) => {
    // Network-first: intercept BEFORE navigation
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockQBConfig }),
      })
    );

    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.route(`${ROUTE_API}/media/recent*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    );

    await page.goto('/');

    // THEN: Both panels show their empty states
    await expect(page.getByText('目前沒有下載任務')).toBeVisible();
    await expect(page.getByText('媒體庫中還沒有內容')).toBeVisible();
    // Navigation links still available
    await expect(page.getByText('查看全部下載 →')).toBeVisible();
    await expect(page.getByText('查看全部媒體庫 →')).toBeVisible();
  });
});

// =============================================================================
// Connection Status Badge Tests
// =============================================================================

// [Downgraded to unit] connection badge → DownloadPanel.spec.tsx

// =============================================================================
// Recent Media API Tests
// =============================================================================

test.describe('Recent Media API @dashboard @api', () => {
  test('[P1] GET /api/v1/media/recent should return recent media items', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/media/recent`);
    expect(response.status()).toBe(200);

    const data = await response.json();
    expect(data.success).toBe(true);
    expect(Array.isArray(data.data)).toBe(true);
  });

  test('[P2] GET /api/v1/media/recent should respect limit parameter', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/media/recent?limit=5`);
    expect(response.status()).toBe(200);

    const data = await response.json();
    expect(data.success).toBe(true);
    expect(data.data.length).toBeLessThanOrEqual(5);
  });

  test('[P2] GET /api/v1/media/recent should default limit for invalid values', async ({
    request,
  }) => {
    // GIVEN: limit=0 (invalid)
    const response = await request.get(`${API_BASE_URL}/media/recent?limit=0`);

    // THEN: Returns 200 with default limit behavior
    expect(response.status()).toBe(200);
    const data = await response.json();
    expect(data.success).toBe(true);
  });

  test('[P2] GET /api/v1/media/recent should cap limit at 50', async ({ request }) => {
    // GIVEN: limit=100 (exceeds max)
    const response = await request.get(`${API_BASE_URL}/media/recent?limit=100`);

    // THEN: Returns 200, doesn't exceed 50 items
    expect(response.status()).toBe(200);
    const data = await response.json();
    expect(data.success).toBe(true);
    expect(data.data.length).toBeLessThanOrEqual(50);
  });
});
