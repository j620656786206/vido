/**
 * Connection Health Monitoring E2E Tests (Story 4.6)
 *
 * Tests for qBittorrent connection health status display,
 * disconnection detection, history panel, and integration
 * with the existing health monitoring system.
 *
 * @tags @e2e @p1 @health
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// For page.route() interception, use glob pattern that matches any origin
const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data
// =============================================================================

function buildServicesHealth(
  qbStatus: 'healthy' | 'degraded' | 'down' = 'healthy',
  opts: { lastSuccess?: string; errorCount?: number; message?: string } = {}
) {
  const now = new Date().toISOString();
  return {
    success: true,
    data: {
      degradationLevel: qbStatus === 'healthy' ? 'normal' : 'partial',
      services: {
        tmdb: {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'healthy',
          lastCheck: now,
          lastSuccess: now,
          errorCount: 0,
        },
        douban: {
          name: 'douban',
          displayName: 'Douban Scraper',
          status: 'healthy',
          lastCheck: now,
          lastSuccess: now,
          errorCount: 0,
        },
        wikipedia: {
          name: 'wikipedia',
          displayName: 'Wikipedia API',
          status: 'healthy',
          lastCheck: now,
          lastSuccess: now,
          errorCount: 0,
        },
        ai: {
          name: 'ai',
          displayName: 'AI Parser',
          status: 'healthy',
          lastCheck: now,
          lastSuccess: now,
          errorCount: 0,
        },
        qbittorrent: {
          name: 'qbittorrent',
          displayName: 'qBittorrent',
          status: qbStatus,
          lastCheck: now,
          lastSuccess: opts.lastSuccess ?? now,
          errorCount: opts.errorCount ?? 0,
          message: opts.message ?? '',
        },
      },
      message: '',
    },
  };
}

const mockConnectionHistory = {
  success: true,
  data: [
    {
      id: 'evt-1',
      service: 'qbittorrent',
      eventType: 'disconnected',
      status: 'down',
      message: 'connection refused',
      createdAt: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    },
    {
      id: 'evt-2',
      service: 'qbittorrent',
      eventType: 'connected',
      status: 'healthy',
      message: '',
      createdAt: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    },
    {
      id: 'evt-3',
      service: 'qbittorrent',
      eventType: 'recovered',
      status: 'healthy',
      message: '',
      createdAt: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    },
  ],
};

const mockEmptyHistory = {
  success: true,
  data: [],
};

// Common mock to silence dashboard API calls
async function mockDashboardAPIs(page: import('@playwright/test').Page) {
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
      body: JSON.stringify({
        success: true,
        data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true },
      }),
    })
  );
}

// =============================================================================
// AC1: Status Indicator Display
// =============================================================================

// [Downgraded to unit] status indicator display → QBStatusIndicator.spec.tsx

// =============================================================================
// AC2: Disconnection Detection
// =============================================================================

test.describe('Connection Health - Disconnection Detection @e2e @p1', () => {
  // [Downgraded to unit] disconnected status display → QBStatusIndicator.spec.tsx

  test('[P1] should update indicator when status changes from healthy to down (AC2)', async ({
    page,
  }) => {
    await mockDashboardAPIs(page);

    // Start with healthy
    let callCount = 0;
    await page.route('**/api/v1/health/services', (route) => {
      callCount++;
      const isDown = callCount > 1;
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          isDown
            ? buildServicesHealth('down', {
                lastSuccess: new Date(Date.now() - 60 * 1000).toISOString(),
                errorCount: 3,
                message: 'connection refused',
              })
            : buildServicesHealth('healthy')
        ),
      });
    });

    await page.goto('/');

    // Initially healthy
    await expect(page.getByRole('button', { name: 'qBittorrent 已連線' })).toBeVisible();

    // Wait for re-fetch (polling interval is 30s, but we can trigger re-fetch by navigating)
    // Force re-fetch by evaluating window
    await page.evaluate(() => {
      window.dispatchEvent(new Event('focus'));
    });

    // Wait for the status to change
    await expect(page.getByRole('button', { name: 'qBittorrent 未連線' })).toBeVisible({
      timeout: 35000,
    });
  });
});

// =============================================================================
// AC3: Auto-Recovery
// =============================================================================

test.describe('Connection Health - Auto-Recovery @e2e @p1', () => {
  test('[P1] should update indicator from down to healthy on recovery (AC3)', async ({ page }) => {
    await mockDashboardAPIs(page);

    // GIVEN: Start with qBittorrent down
    let callCount = 0;
    await page.route('**/api/v1/health/services', (route) => {
      callCount++;
      const isRecovered = callCount > 1;
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          isRecovered
            ? buildServicesHealth('healthy')
            : buildServicesHealth('down', {
                lastSuccess: new Date(Date.now() - 3 * 60 * 1000).toISOString(),
                errorCount: 5,
                message: 'connection refused',
              })
        ),
      });
    });

    await page.goto('/');

    // THEN: Initially shows disconnected
    await expect(page.getByRole('button', { name: 'qBittorrent 未連線' })).toBeVisible();

    // WHEN: Trigger re-fetch (simulate window focus to trigger refetchOnWindowFocus)
    await page.evaluate(() => {
      window.dispatchEvent(new Event('focus'));
    });

    // THEN: Indicator recovers to healthy
    await expect(page.getByRole('button', { name: 'qBittorrent 已連線' })).toBeVisible({
      timeout: 35000,
    });
  });
});

// =============================================================================
// AC4: Connection History Panel
// =============================================================================

test.describe('Connection Health - History Panel @e2e @p1', () => {
  test('[P1] should open history panel when clicking status indicator (AC4)', async ({ page }) => {
    await mockDashboardAPIs(page);

    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildServicesHealth('healthy')),
      });
    });

    await page.route('**/api/v1/health/services/qbittorrent/history*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockConnectionHistory),
      });
    });

    await page.goto('/');

    // WHEN: Click the status indicator
    const indicator = page.getByRole('button', { name: 'qBittorrent 已連線' });
    await expect(indicator).toBeVisible();
    await indicator.click();

    // THEN: Side panel opens with title
    await expect(page.getByText('qBittorrent 連線記錄')).toBeVisible();
  });

  test('[P1] should display connection history events (AC4)', async ({ page }) => {
    await mockDashboardAPIs(page);

    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildServicesHealth('healthy')),
      });
    });

    await page.route('**/api/v1/health/services/qbittorrent/history*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockConnectionHistory),
      });
    });

    await page.goto('/');
    const indicator = page.getByRole('button', { name: 'qBittorrent 已連線' });
    await expect(indicator).toBeVisible();
    await indicator.click();

    // THEN: History events are displayed
    await expect(page.getByText('connection refused')).toBeVisible();

    // Filter buttons are visible
    await expect(page.getByRole('button', { name: '全部' })).toBeVisible();
  });

  test('[P1] should filter connection history by event type (AC4)', async ({ page }) => {
    await mockDashboardAPIs(page);

    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildServicesHealth('healthy')),
      });
    });

    await page.route('**/api/v1/health/services/qbittorrent/history*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockConnectionHistory),
      });
    });

    await page.goto('/');
    const indicator = page.getByRole('button', { name: 'qBittorrent 已連線' });
    await expect(indicator).toBeVisible();
    await indicator.click();
    await expect(page.getByText('qBittorrent 連線記錄')).toBeVisible();

    // WHEN: Click "已斷線" filter button (in filter group)
    const filterGroup = page.getByRole('group', { name: '篩選事件類型' });
    await filterGroup.getByText('已斷線').click();

    // THEN: Only disconnected events remain
    const listItems = page.getByRole('listitem');
    await expect(listItems).toHaveCount(1);
    await expect(page.getByText('connection refused')).toBeVisible();
  });

  // [Downgraded to unit] empty history state → ConnectionHistoryPanel.spec.tsx

  test('[P1] should close history panel via close button (AC4)', async ({ page }) => {
    await mockDashboardAPIs(page);

    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildServicesHealth('healthy')),
      });
    });

    await page.route('**/api/v1/health/services/qbittorrent/history*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockConnectionHistory),
      });
    });

    await page.goto('/');
    const indicator = page.getByRole('button', { name: 'qBittorrent 已連線' });
    await expect(indicator).toBeVisible();
    await indicator.click();
    await expect(page.getByText('qBittorrent 連線記錄')).toBeVisible();

    // WHEN: Click close button
    await page.getByTestId('side-panel-close').click();

    // THEN: Panel closes
    await expect(page.getByText('qBittorrent 連線記錄')).not.toBeVisible();
  });
});

// =============================================================================
// AC5: Integration with Existing Health System (API Tests)
// =============================================================================

test.describe('Connection Health - API Integration @e2e @p1', () => {
  test('[P1] should include qBittorrent in services health response (AC5)', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/health/services`);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.services.qbittorrent).toBeDefined();
    expect(body.data.services.qbittorrent.name).toBe('qbittorrent');
    expect(body.data.services.qbittorrent.displayName).toBe('qBittorrent');
    expect(body.data.services.qbittorrent.status).toMatch(/^(healthy|degraded|down)$/);
  });

  test('[P1] should return connection history for qBittorrent (AC5)', async ({ request }) => {
    const response = await request.get(
      `${API_BASE_URL}/health/services/qbittorrent/history?limit=20`
    );
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('[P1] should include required fields in qBittorrent health (AC5)', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/health/services`);
    expect(response.status()).toBe(200);

    const body = await response.json();
    const qb = body.data.services.qbittorrent;

    expect(qb).toHaveProperty('status');
    expect(qb).toHaveProperty('lastCheck');
    expect(qb).toHaveProperty('lastSuccess');
    expect(typeof qb.errorCount).toBe('number');
  });

  test('[P2] should respect limit parameter for connection history', async ({ request }) => {
    const response = await request.get(
      `${API_BASE_URL}/health/services/qbittorrent/history?limit=5`
    );
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.length).toBeLessThanOrEqual(5);
  });
});
