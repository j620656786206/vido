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

const mockQBConfig = {
  host: 'http://localhost:8080',
  username: 'admin',
  basePath: '',
  configured: true,
};

// =============================================================================

// =============================================================================

// =============================================================================

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

// =============================================================================

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
// =============================================================================
// ux3-cutover-3: the legacy dashboard-home describes (Layout / Disconnected /
// Mobile / Quick Actions / Panel Independence / Empty State) are DELETED — the
// legacy home no longer exists and D3 (ux3-1-4) moved the dashboard remnants
// off the homepage by design. v2 home coverage: HomeBrowseV2.spec (D3 ordering,
// remnant-absence) + homepage-layout.spec E2E (v2 assertions).
// =============================================================================

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
