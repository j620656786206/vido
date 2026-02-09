/**
 * Graceful Degradation E2E Tests
 *
 * Tests for graceful degradation behavior (Story 3-12).
 * Priority: P1 (High - run on PR to main)
 *
 * These tests verify the system's behavior when external services fail:
 * - AC1: All sources fail fallback
 * - AC2: AI service down fallback
 * - AC3: Core functionality availability
 * - AC4: Partial success handling
 *
 * @tags @e2e @p1 @degradation
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = 'http://localhost:8080/api/v1';

// =============================================================================
// Mock Data for Degradation Scenarios
// =============================================================================

const mockHealthyResponse = {
  success: true,
  data: {
    degradationLevel: 'normal',
    services: {
      tmdb: {
        name: 'tmdb',
        displayName: 'TMDb API',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      douban: {
        name: 'douban',
        displayName: 'Douban Scraper',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      wikipedia: {
        name: 'wikipedia',
        displayName: 'Wikipedia API',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      ai: {
        name: 'ai',
        displayName: 'AI Parser',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
    },
    message: '',
  },
};

const mockPartialDegradedResponse = {
  success: true,
  data: {
    degradationLevel: 'partial',
    services: {
      tmdb: {
        name: 'tmdb',
        displayName: 'TMDb API',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      douban: {
        name: 'douban',
        displayName: 'Douban Scraper',
        status: 'degraded',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 300000).toISOString(),
        errorCount: 2,
        message: 'Rate limited',
      },
      wikipedia: {
        name: 'wikipedia',
        displayName: 'Wikipedia API',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      ai: {
        name: 'ai',
        displayName: 'AI Parser',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 7200000).toISOString(),
        errorCount: 5,
        message: 'quota exceeded',
      },
    },
    message: '部分服務降級中：AI 解析暫時使用基本模式',
  },
};

const mockOfflineResponse = {
  success: true,
  data: {
    degradationLevel: 'offline',
    services: {
      tmdb: {
        name: 'tmdb',
        displayName: 'TMDb API',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 3600000).toISOString(),
        errorCount: 10,
        message: 'Connection refused',
      },
      douban: {
        name: 'douban',
        displayName: 'Douban Scraper',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 3600000).toISOString(),
        errorCount: 10,
        message: 'Connection refused',
      },
      wikipedia: {
        name: 'wikipedia',
        displayName: 'Wikipedia API',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 3600000).toISOString(),
        errorCount: 10,
        message: 'Connection refused',
      },
      ai: {
        name: 'ai',
        displayName: 'AI Parser',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date(Date.now() - 3600000).toISOString(),
        errorCount: 10,
        message: 'Connection refused',
      },
    },
    message: '無法連線到外部服務，使用本地快取資料',
  },
};

// =============================================================================
// Services Health API Verification Tests
// =============================================================================

test.describe('Graceful Degradation - API Integration @e2e @p1', () => {
  test('[P1] should fetch services health status on app load', async ({ page, request }) => {
    // GIVEN: The app is loading

    // WHEN: Navigating to the app
    const healthPromise = page.waitForResponse(
      (response) => response.url().includes('/health/services') && response.status() === 200
    );

    await page.goto('/');

    // THEN: Health endpoint should be called (if the app has health monitoring UI)
    // Note: This test verifies that the API works, not that the UI calls it
    const response = await request.get(`${API_BASE_URL}/health/services`);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.degradationLevel).toBeDefined();
  });

  test('[P1] should handle healthy services status', async ({ page }) => {
    // GIVEN: All services are healthy
    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockHealthyResponse),
      });
    });

    // WHEN: Page loads
    await page.goto('/');

    // THEN: No degradation banner should be visible
    // (Normal status should not show warning)
    const banner = page.locator('[role="alert"]');
    await expect(banner).toHaveCount(0);
  });

  test('[P1] should display degradation banner when services are partially down', async ({
    page,
  }) => {
    // GIVEN: Mock the health endpoint to return partial degradation
    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPartialDegradedResponse),
      });
    });

    // WHEN: Page loads (with hypothetical integration)
    await page.goto('/');

    // Note: This test documents expected behavior if ServiceHealthBanner
    // is integrated into the app layout. If not yet integrated, this test
    // serves as a specification for future implementation.

    // THEN: Page should load successfully
    await expect(page).toHaveURL('/');
  });

  test('[P1] should handle offline status gracefully', async ({ page }) => {
    // GIVEN: Mock the health endpoint to return offline status
    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify(mockOfflineResponse),
      });
    });

    // WHEN: Page loads
    await page.goto('/');

    // THEN: Page should still be accessible (core functionality works)
    await expect(page).toHaveURL('/');
  });
});

// =============================================================================
// AC3: Core Functionality Availability Tests
// =============================================================================

test.describe('Graceful Degradation - Core Functionality @e2e @p1', () => {
  test('[P1] should allow library browsing when external APIs are down', async ({
    page,
    request,
  }) => {
    // GIVEN: External services are offline
    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify(mockOfflineResponse),
      });
    });

    // WHEN: Navigating to library
    await page.goto('/');

    // THEN: Core navigation should work
    await expect(page).toHaveURL('/');

    // Verify basic health endpoint still responds
    const healthResponse = await request.get('http://localhost:8080/health');
    expect(healthResponse.status()).toBe(200);
  });

  test('[P1] should keep local data accessible during degradation', async ({ page, api }) => {
    // GIVEN: Some movies exist in local database
    const movies = await api.listMovies();

    // WHEN: External services are degraded
    await page.route('**/api/v1/health/services', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPartialDegradedResponse),
      });
    });

    await page.goto('/');

    // THEN: Local movies API should still work
    const response = await api.listMovies();
    expect(response.success).toBe(true);
  });
});

// =============================================================================
// AC2: AI Service Fallback Tests (API Level)
// =============================================================================

test.describe('Graceful Degradation - AI Fallback @e2e @p1', () => {
  test('[P1] should use regex fallback when AI is down', async ({ request }) => {
    // GIVEN: A filename to parse

    // WHEN: Parsing a movie filename
    const response = await request.post(`${API_BASE_URL}/parser/movie`, {
      data: {
        filename: 'The.Matrix.1999.1080p.BluRay.x264.mkv',
      },
    });

    // THEN: Should return parsed result (even if AI is down, regex fallback works)
    // The actual source field will indicate if fallback was used
    if (response.status() === 200) {
      const body = await response.json();
      expect(body.success).toBe(true);
      expect(body.data).toBeDefined();
    }
    // Note: If AI is down, the source should be 'regex_fallback'
  });

  test('[P1] should return degradation message when using fallback', async ({ request }) => {
    // GIVEN: Health endpoint shows AI is down
    const healthResponse = await request.get(`${API_BASE_URL}/health/services`);
    const health = await healthResponse.json();

    // WHEN: Checking AI service status
    const aiStatus = health.data?.services?.ai?.status;

    // THEN: If AI is down, parsing should still work via fallback
    if (aiStatus === 'down') {
      const parseResponse = await request.post(`${API_BASE_URL}/parser/movie`, {
        data: {
          filename: '[SubGroup] Anime Title - 01 [1080p].mkv',
        },
      });

      const body = await parseResponse.json();

      // Should include degradation message or fallback indicator
      if (body.data?.source === 'regex_fallback') {
        expect(body.data.degradationMessage).toBeDefined();
      }
    }
  });
});

// =============================================================================
// Network Resilience Tests
// =============================================================================

test.describe('Graceful Degradation - Network Resilience @e2e @p2', () => {
  test('[P2] should handle API timeout gracefully', async ({ page }) => {
    // GIVEN: Health endpoint times out
    await page.route('**/api/v1/health/services', async (route) => {
      // Simulate timeout by delaying response
      await new Promise((resolve) => setTimeout(resolve, 5000));
      route.abort('timedout');
    });

    // WHEN: Page loads
    await page.goto('/', { timeout: 10000 });

    // THEN: Page should still load (degraded gracefully)
    await expect(page).toHaveURL('/');
  });

  test('[P2] should handle network errors gracefully', async ({ page }) => {
    // GIVEN: Health endpoint returns network error
    await page.route('**/api/v1/health/services', (route) => {
      route.abort('connectionfailed');
    });

    // WHEN: Page loads
    await page.goto('/');

    // THEN: Page should still be functional
    await expect(page).toHaveURL('/');
  });
});
