/**
 * Static Serving & Unified Docker API Tests
 *
 * Integration tests for retro-8-D1: Single Docker Image.
 * Validates static file serving, SPA fallback, cache headers,
 * security headers, and gzip compression from HTTP client perspective.
 *
 * Two test groups:
 * - "Unified Server" tests require static files deployed (Docker/production).
 *   Auto-skipped in dev mode when Go server has no public dir.
 * - "Server Middleware" tests work in all environments.
 *
 * Priority: P1 (High - run on PR to main)
 *
 * @tags @api @p1
 */

import { test, expect } from '../support/fixtures';

const API_URL = process.env.API_URL?.replace('/api/v1', '') || 'http://localhost:8080';

// =============================================================================
// Unified Server Tests (require static files — auto-skip in dev mode)
// =============================================================================

test.describe('Static Serving & SPA Fallback @api @deployment', () => {
  let hasStaticFiles: boolean;

  test.beforeAll(async ({ request }) => {
    // Detect if static files are deployed by probing the root path.
    // In dev mode, Go server has no public dir → NoRoute not registered → 404.
    // In unified mode, root path serves index.html → 200.
    try {
      const response = await request.get(`${API_URL}/`);
      hasStaticFiles = response.status() === 200;
    } catch {
      hasStaticFiles = false;
    }
  });

  test.beforeEach(async () => {
    test.skip(
      !hasStaticFiles,
      'Static files not deployed — skipping deployment tests (run in Docker)'
    );
  });

  test('[P1] SPA fallback: /library returns 200 with HTML content', async ({ request }) => {
    // GIVEN: The unified server is running with static file serving

    // WHEN: Requesting a client-side route that has no server-side handler
    const response = await request.get(`${API_URL}/library`);

    // THEN: Should return 200 with HTML content (index.html via SPA fallback)
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('<!DOCTYPE html');
  });

  test('[P1] SPA deep route: /library/movies/123 returns 200 with HTML', async ({ request }) => {
    // GIVEN: The unified server serves SPA for all non-API routes

    // WHEN: Requesting a deep nested client-side route
    const response = await request.get(`${API_URL}/library/movies/123`);

    // THEN: Should return 200 with index.html (not 404)
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('<!DOCTYPE html');
  });

  test('[P1] API 404: /api/v1/nonexistent returns JSON error, not HTML', async ({ request }) => {
    // GIVEN: The server distinguishes API routes from SPA routes

    // WHEN: Requesting a non-existent API endpoint
    const response = await request.get(`${API_URL}/api/v1/nonexistent`);

    // THEN: Should return JSON 404 with error structure, not index.html
    expect(response.status()).toBe(404);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('NOT_FOUND');
  });

  test('[P1] Cache-Control: root path returns no-store header', async ({ request }) => {
    // GIVEN: index.html must never be cached for instant SPA updates

    // WHEN: Requesting the root path
    const response = await request.get(`${API_URL}/`);

    // THEN: Should have no-cache Cache-Control header
    expect(response.status()).toBe(200);
    const cacheControl = response.headers()['cache-control'];
    expect(cacheControl).toContain('no-store');
    expect(cacheControl).toContain('no-cache');
  });
});

// =============================================================================
// Server Middleware Tests (work in all environments)
// =============================================================================

test.describe('Server Middleware @api', () => {
  test('[P1] Security headers present on API responses', async ({ request }) => {
    // GIVEN: Go middleware replicates Nginx security headers

    // WHEN: Making a request to the health endpoint (always available)
    const response = await request.get(`${API_URL}/health`);

    // THEN: Standard security headers should be present
    const headers = response.headers();
    expect(headers['x-frame-options']).toBe('SAMEORIGIN');
    expect(headers['x-content-type-options']).toBe('nosniff');
    expect(headers['x-xss-protection']).toBe('1; mode=block');
    expect(headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
  });

  test('[P1] Gzip: JSON responses compressed with Accept-Encoding', async ({ request }) => {
    // GIVEN: Gzip middleware is enabled for text-based responses

    // WHEN: Requesting a JSON API endpoint with Accept-Encoding: gzip
    const response = await request.get(`${API_URL}/health`, {
      headers: {
        'Accept-Encoding': 'gzip, deflate',
      },
    });

    // THEN: Response should be compressed with gzip
    expect(response.status()).toBe(200);
    const headers = response.headers();
    expect(headers['content-encoding']).toBe('gzip');
    // Playwright auto-decompresses, so body should still be valid JSON
    const body = await response.json();
    expect(body).toHaveProperty('status');
    expect(body).toHaveProperty('service');
  });

  test('[P1] Health endpoint works after middleware changes', async ({ request }) => {
    // GIVEN: Health endpoint must work after adding gzip + security middleware

    // WHEN: Requesting the health endpoint
    const response = await request.get(`${API_URL}/health`);

    // THEN: Should return 200 with expected health structure
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.status).toBe('healthy');
    expect(body.service).toBe('vido-api');
  });

  test('[P2] Health endpoint returns complete schema with database info', async ({ request }) => {
    // GIVEN: Health endpoint provides database health details

    // WHEN: Requesting the health endpoint
    const response = await request.get(`${API_URL}/health`);

    // THEN: Should include database health information
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.database).toBeDefined();
    expect(body.database).toHaveProperty('status');
    expect(body.database).toHaveProperty('latency');
    expect(body.database).toHaveProperty('walEnabled');
  });
});
