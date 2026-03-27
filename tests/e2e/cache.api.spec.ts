/**
 * Cache Management API E2E Tests (Story 6.2)
 *
 * Tests for cache statistics and cleanup endpoints:
 * - GET /api/v1/settings/cache — view cache stats by type (AC1)
 * - DELETE /api/v1/settings/cache?older_than_days=N — clear old cache (AC2)
 * - DELETE /api/v1/settings/cache/:type — clear specific cache type (AC3)
 * - DELETE /api/v1/settings/cache — clear all cache types
 *
 * Prerequisites:
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @api @cache @story-6-2
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// GET /api/v1/settings/cache — Cache Stats (AC1)
// =============================================================================

test.describe('Cache Stats API @api @cache @story-6-2', () => {
  test('[P0] GET /settings/cache should return cache stats with all cache types (AC1)', async ({
    request,
  }) => {
    // GIVEN: Backend is running with cache tables available

    // WHEN: Requesting cache statistics
    const response = await request.get(`${API_BASE_URL}/settings/cache`);

    // THEN: Should return 200 with cache stats
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toHaveProperty('cache_types');
    expect(body.data).toHaveProperty('total_size_bytes');
    expect(Array.isArray(body.data.cache_types)).toBe(true);
  });

  test('[P1] GET /settings/cache should include all 5 cache types (AC1)', async ({ request }) => {
    // GIVEN: Backend is running

    // WHEN: Requesting cache statistics
    const response = await request.get(`${API_BASE_URL}/settings/cache`);

    // THEN: Should contain all 5 cache types
    const body = await response.json();
    const types = body.data.cache_types.map((ct: { type: string }) => ct.type);
    expect(types).toContain('image');
    expect(types).toContain('ai');
    expect(types).toContain('metadata');
    expect(types).toContain('douban');
    expect(types).toContain('wikipedia');
  });

  test('[P1] GET /settings/cache cache type entries should have required fields (AC1)', async ({
    request,
  }) => {
    // GIVEN: Backend is running

    // WHEN: Requesting cache statistics
    const response = await request.get(`${API_BASE_URL}/settings/cache`);

    // THEN: Each cache type should have type, label, size_bytes, entry_count
    const body = await response.json();
    for (const ct of body.data.cache_types) {
      expect(ct).toHaveProperty('type');
      expect(ct).toHaveProperty('label');
      expect(ct).toHaveProperty('size_bytes');
      expect(ct).toHaveProperty('entry_count');
      expect(typeof ct.size_bytes).toBe('number');
      expect(typeof ct.entry_count).toBe('number');
      expect(ct.size_bytes).toBeGreaterThanOrEqual(0);
      expect(ct.entry_count).toBeGreaterThanOrEqual(0);
    }
  });

  test('[P1] GET /settings/cache total_size_bytes should be sum of all types (AC1)', async ({
    request,
  }) => {
    // GIVEN: Backend is running

    // WHEN: Requesting cache statistics
    const response = await request.get(`${API_BASE_URL}/settings/cache`);

    // THEN: total_size_bytes should match sum
    const body = await response.json();
    const summedSize = body.data.cache_types.reduce(
      (sum: number, ct: { size_bytes: number }) => sum + ct.size_bytes,
      0
    );
    expect(body.data.total_size_bytes).toBe(summedSize);
  });

  test('[P2] GET /settings/cache should include Chinese labels for each type', async ({
    request,
  }) => {
    // GIVEN: Backend configured with zh-TW

    // WHEN: Requesting cache statistics
    const response = await request.get(`${API_BASE_URL}/settings/cache`);

    // THEN: Labels should be in Chinese
    const body = await response.json();
    const labels = body.data.cache_types.map((ct: { label: string }) => ct.label);
    expect(labels).toContain('圖片快取');
    expect(labels).toContain('AI 解析快取');
  });
});

// =============================================================================
// DELETE /api/v1/settings/cache?older_than_days=N — Clear by Age (AC2)
// =============================================================================

test.describe('Cache Clear by Age API @api @cache @story-6-2', () => {
  test('[P1] DELETE /settings/cache?older_than_days=30 should clear old cache entries (AC2)', async ({
    request,
  }) => {
    // GIVEN: Cache may contain old entries

    // WHEN: Clearing cache older than 30 days
    const response = await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=30`);

    // THEN: Should return 200 with cleanup result
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toHaveProperty('type');
    expect(body.data).toHaveProperty('entries_removed');
    expect(body.data).toHaveProperty('bytes_reclaimed');
    expect(typeof body.data.entries_removed).toBe('number');
    expect(typeof body.data.bytes_reclaimed).toBe('number');
  });

  test('[P1] DELETE /settings/cache?older_than_days=abc should return 400 (AC2)', async ({
    request,
  }) => {
    // GIVEN: Invalid days parameter

    // WHEN: Sending non-numeric days
    const response = await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=abc`);

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_INVALID_FORMAT');
  });

  test('[P1] DELETE /settings/cache?older_than_days=-5 should return 400 (AC2)', async ({
    request,
  }) => {
    // GIVEN: Negative days parameter

    // WHEN: Sending negative days
    const response = await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=-5`);

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);
  });

  test('[P2] DELETE /settings/cache?older_than_days=0 should return 400 (AC2)', async ({
    request,
  }) => {
    // GIVEN: Zero days parameter

    // WHEN: Sending zero days
    const response = await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=0`);

    // THEN: Should return 400 (0 is not positive)
    expect(response.status()).toBe(400);
  });

  test('[P2] DELETE /settings/cache?older_than_days= should fall through to clear all', async ({
    request,
  }) => {
    // GIVEN: Empty string days parameter

    // WHEN: Sending empty days value
    const response = await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=`);

    // THEN: Empty param treated as absent — falls through to clear all
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data.type).toBe('all');
  });
});

// =============================================================================
// DELETE /api/v1/settings/cache/:type — Clear by Type (AC3)
// =============================================================================

test.describe('Cache Clear by Type API @api @cache @story-6-2', () => {
  test('[P1] DELETE /settings/cache/metadata should clear metadata cache (AC3)', async ({
    request,
  }) => {
    // GIVEN: Metadata cache exists

    // WHEN: Clearing metadata cache
    const response = await request.delete(`${API_BASE_URL}/settings/cache/metadata`);

    // THEN: Should return 200 with result
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.type).toBe('metadata');
    expect(typeof body.data.entries_removed).toBe('number');
  });

  test('[P1] DELETE /settings/cache/ai should clear AI cache (AC3)', async ({ request }) => {
    // GIVEN: AI cache exists

    // WHEN: Clearing AI cache
    const response = await request.delete(`${API_BASE_URL}/settings/cache/ai`);

    // THEN: Should return 200 with result
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data.type).toBe('ai');
  });

  test('[P1] DELETE /settings/cache/image should clear image cache (AC3)', async ({ request }) => {
    // GIVEN: Image cache exists

    // WHEN: Clearing image cache
    const response = await request.delete(`${API_BASE_URL}/settings/cache/image`);

    // THEN: Should return 200 with bytes_reclaimed
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data.type).toBe('image');
    expect(body.data).toHaveProperty('bytes_reclaimed');
  });

  test('[P1] DELETE /settings/cache/bogus should return 400 CACHE_TYPE_INVALID (AC3)', async ({
    request,
  }) => {
    // GIVEN: Invalid cache type

    // WHEN: Clearing non-existent cache type
    const response = await request.delete(`${API_BASE_URL}/settings/cache/bogus`);

    // THEN: Should return 400 with CACHE_TYPE_INVALID error code
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.error.code).toBe('CACHE_TYPE_INVALID');
  });

  test('[P2] DELETE /settings/cache/IMAGE should return 400 for case-sensitive type', async ({
    request,
  }) => {
    // GIVEN: Uppercase cache type (API is case-sensitive)

    // WHEN: Clearing with uppercase type
    const response = await request.delete(`${API_BASE_URL}/settings/cache/IMAGE`);

    // THEN: Should return 400 (types are lowercase)
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.error.code).toBe('CACHE_TYPE_INVALID');
  });
});

// =============================================================================
// DELETE /api/v1/settings/cache — Clear All (no param)
// =============================================================================

test.describe('Cache Clear All API @api @cache @story-6-2', () => {
  test('[P1] DELETE /settings/cache should clear all cache types', async ({ request }) => {
    // GIVEN: Cache contains entries

    // WHEN: Clearing all cache (no older_than_days param)
    const response = await request.delete(`${API_BASE_URL}/settings/cache`);

    // THEN: Should return 200 with aggregated result
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.type).toBe('all');
    expect(typeof body.data.entries_removed).toBe('number');
    expect(typeof body.data.bytes_reclaimed).toBe('number');
  });
});

// =============================================================================
// Response Format Validation
// =============================================================================

test.describe('Cache API - Response Format @api @cache @story-6-2', () => {
  test('[P1] error responses should include code and message', async ({ request }) => {
    // GIVEN: An invalid request

    // WHEN: Sending invalid cache type
    const response = await request.delete(`${API_BASE_URL}/settings/cache/invalid_type`);

    // THEN: Error should follow standard API format
    expect(response.status()).toBe(400);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toHaveProperty('code');
    expect(body.error).toHaveProperty('message');
    expect(typeof body.error.code).toBe('string');
    expect(typeof body.error.message).toBe('string');
  });
});
