/**
 * System Logs API E2E Tests (Story 6.3)
 *
 * Tests for system log viewing and clearing endpoints:
 * - GET /api/v1/settings/logs — paginated logs with filters (AC1, AC2, AC3)
 * - DELETE /api/v1/settings/logs — clear logs older than N days (AC2)
 *
 * Prerequisites:
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @api @logs @story-6-3
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// GET /api/v1/settings/logs — Paginated Logs (AC1, AC2, AC3)
// =============================================================================

test.describe('System Logs API @api @logs @story-6-3', () => {
  test('[P0] GET /settings/logs should return paginated logs response (AC1, AC2)', async ({
    request,
  }) => {
    // GIVEN: Backend is running with system_logs table

    // WHEN: Requesting logs without filters
    const response = await request.get(`${API_BASE_URL}/settings/logs`);

    // THEN: Should return 200 with paginated response
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toHaveProperty('logs');
    expect(body.data).toHaveProperty('total');
    expect(body.data).toHaveProperty('page');
    expect(body.data).toHaveProperty('perPage');
    expect(Array.isArray(body.data.logs)).toBe(true);
    expect(typeof body.data.total).toBe('number');
  });

  test('[P1] GET /settings/logs log entries should have required fields (AC1)', async ({
    request,
  }) => {
    // GIVEN: Backend has generated some logs during startup

    // WHEN: Requesting logs
    const response = await request.get(`${API_BASE_URL}/settings/logs`);
    const body = await response.json();

    // THEN: If logs exist, each entry should have required fields
    if (body.data.logs.length > 0) {
      const log = body.data.logs[0];
      expect(log).toHaveProperty('id');
      expect(log).toHaveProperty('level');
      expect(log).toHaveProperty('message');
      expect(log).toHaveProperty('createdAt');
      expect(['ERROR', 'WARN', 'INFO', 'DEBUG']).toContain(log.level);
      expect(typeof log.message).toBe('string');
    }
  });

  test('[P1] GET /settings/logs should return newest logs first (AC2)', async ({ request }) => {
    // GIVEN: Multiple logs exist

    // WHEN: Requesting logs
    const response = await request.get(`${API_BASE_URL}/settings/logs`);
    const body = await response.json();

    // THEN: Logs should be in descending chronological order
    if (body.data.logs.length >= 2) {
      const first = new Date(body.data.logs[0].createdAt).getTime();
      const second = new Date(body.data.logs[1].createdAt).getTime();
      expect(first).toBeGreaterThanOrEqual(second);
    }
  });

  test('[P1] GET /settings/logs?level=ERROR should filter by level (AC3)', async ({ request }) => {
    // GIVEN: Logs of various levels exist

    // WHEN: Filtering by ERROR level
    const response = await request.get(`${API_BASE_URL}/settings/logs?level=ERROR`);

    // THEN: Should return only ERROR logs
    expect(response.status()).toBe(200);
    const body = await response.json();
    for (const log of body.data.logs) {
      expect(log.level).toBe('ERROR');
    }
  });

  test('[P1] GET /settings/logs?keyword=search should filter by keyword (AC3)', async ({
    request,
  }) => {
    // GIVEN: Backend has logged some messages

    // WHEN: Searching for a keyword
    const response = await request.get(`${API_BASE_URL}/settings/logs?keyword=initialized`);

    // THEN: Should return 200 (may or may not have results)
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
  });

  test('[P1] GET /settings/logs?page=1&per_page=5 should paginate (AC2)', async ({ request }) => {
    // GIVEN: Multiple logs exist

    // WHEN: Requesting page 1 with 5 per page
    const response = await request.get(`${API_BASE_URL}/settings/logs?page=1&per_page=5`);

    // THEN: Should return at most 5 logs
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data.logs.length).toBeLessThanOrEqual(5);
    expect(body.data.page).toBe(1);
    expect(body.data.perPage).toBe(5);
  });

  test('[P1] GET /settings/logs?level=INVALID should return 400 (AC3)', async ({ request }) => {
    // GIVEN: Invalid log level

    // WHEN: Filtering with invalid level
    const response = await request.get(`${API_BASE_URL}/settings/logs?level=INVALID`);

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_INVALID_FORMAT');
  });
});

// =============================================================================
// DELETE /api/v1/settings/logs — Clear Logs (AC2)
// =============================================================================

test.describe('System Logs Clear API @api @logs @story-6-3', () => {
  test('[P1] DELETE /settings/logs?older_than_days=30 should clear old logs', async ({
    request,
  }) => {
    // GIVEN: System logs may contain old entries

    // WHEN: Clearing logs older than 30 days
    const response = await request.delete(`${API_BASE_URL}/settings/logs?older_than_days=30`);

    // THEN: Should return 200 with clear result
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toHaveProperty('entriesRemoved');
    expect(body.data).toHaveProperty('days');
    expect(typeof body.data.entriesRemoved).toBe('number');
    expect(body.data.days).toBe(30);
  });

  test('[P1] DELETE /settings/logs without older_than_days should return 400', async ({
    request,
  }) => {
    // GIVEN: Missing required parameter

    // WHEN: Deleting without days parameter
    const response = await request.delete(`${API_BASE_URL}/settings/logs`);

    // THEN: Should return 400 requiring the parameter
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.error.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P1] DELETE /settings/logs?older_than_days=-1 should return 400', async ({ request }) => {
    // GIVEN: Negative days parameter

    // WHEN: Sending negative days
    const response = await request.delete(`${API_BASE_URL}/settings/logs?older_than_days=-1`);

    // THEN: Should return 400
    expect(response.status()).toBe(400);
  });

  test('[P1] DELETE /settings/logs?older_than_days=abc should return 400', async ({ request }) => {
    // GIVEN: Non-numeric days parameter

    // WHEN: Sending non-numeric days
    const response = await request.delete(`${API_BASE_URL}/settings/logs?older_than_days=abc`);

    // THEN: Should return 400
    expect(response.status()).toBe(400);
  });
});

// =============================================================================
// Response Format (AC1, AC4)
// =============================================================================

test.describe('System Logs API - Response Format @api @logs @story-6-3', () => {
  test('[P1] error responses should follow standard API format', async ({ request }) => {
    // GIVEN: An invalid request

    // WHEN: Sending invalid level
    const response = await request.get(`${API_BASE_URL}/settings/logs?level=BOGUS`);

    // THEN: Error should follow standard API format
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toHaveProperty('code');
    expect(body.error).toHaveProperty('message');
    expect(typeof body.error.code).toBe('string');
    expect(typeof body.error.message).toBe('string');
  });

  test('[P2] ERROR log entries should include hint field when available (AC4)', async ({
    request,
  }) => {
    // GIVEN: Backend may have ERROR logs with known error codes

    // WHEN: Requesting ERROR logs
    const response = await request.get(`${API_BASE_URL}/settings/logs?level=ERROR`);
    const body = await response.json();

    // THEN: ERROR entries with known error codes should have hints
    // Note: hints are enriched server-side based on context.error_code
    for (const log of body.data.logs) {
      if (log.context?.error_code) {
        // Known error codes should have hints
        const knownCodes = [
          'TMDB_TIMEOUT',
          'AI_QUOTA_EXCEEDED',
          'DB_QUERY_FAILED',
          'QBT_CONNECTION',
        ];
        if (knownCodes.includes(log.context.error_code)) {
          expect(log.hint).toBeTruthy();
        }
      }
    }
  });
});
