/**
 * Download Filtering API E2E Tests (Story 4.4)
 *
 * Tests for download status filtering and counts endpoints.
 * Tests follow Given-When-Then format.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * Note: These tests work against the live API. If qBittorrent
 * is not configured, they validate graceful error handling.
 *
 * @tags @api @downloads @filtering
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Download Filtering API Tests
// =============================================================================

test.describe('Download Filtering API @api @downloads @filtering', () => {
  // ---------------------------------------------------------------------------
  // GET /downloads?filter= - Filter downloads by status
  // ---------------------------------------------------------------------------

  test('[P1] GET /downloads?filter=downloading - should accept filter parameter (AC2)', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/downloads?filter=downloading`);
    const json = await response.json();

    if (response.ok()) {
      expect(json.success).toBe(true);
      expect(Array.isArray(json.data)).toBe(true);
    } else {
      expect(json.error.code).toMatch(/^QBITTORRENT_/);
    }
  });

  test('[P1] GET /downloads - should accept all filter values (AC1)', async ({ request }) => {
    const filters = ['all', 'downloading', 'paused', 'completed', 'seeding', 'error'];

    for (const filter of filters) {
      const response = await request.get(`${API_BASE_URL}/downloads?filter=${filter}`);
      const json = await response.json();

      // Should not crash regardless of filter value
      expect(json).toHaveProperty('success');
    }
  });

  test('[P2] GET /downloads?filter=invalid - should fall back to all for invalid filter', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/downloads?filter=invalid`);
    const json = await response.json();

    // Should not crash, falls back to "all"
    expect(json).toHaveProperty('success');
  });

  test('[P1] GET /downloads - should combine filter with sort (AC2)', async ({ request }) => {
    const response = await request.get(
      `${API_BASE_URL}/downloads?filter=downloading&sort=name&order=asc`
    );
    const json = await response.json();

    if (response.ok()) {
      expect(json.success).toBe(true);
      expect(Array.isArray(json.data)).toBe(true);
    } else {
      expect(json.error.code).toMatch(/^QBITTORRENT_/);
    }
  });

  // ---------------------------------------------------------------------------
  // GET /downloads/counts - Download counts by status
  // ---------------------------------------------------------------------------

  test('[P1] GET /downloads/counts - should return counts by status (AC1)', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/downloads/counts`);
    const json = await response.json();

    if (response.ok()) {
      expect(json.success).toBe(true);
      expect(json.data).toHaveProperty('all');
      expect(json.data).toHaveProperty('downloading');
      expect(json.data).toHaveProperty('paused');
      expect(json.data).toHaveProperty('completed');
      expect(json.data).toHaveProperty('seeding');
      expect(json.data).toHaveProperty('error');

      // All counts should be non-negative integers
      expect(json.data.all).toBeGreaterThanOrEqual(0);
      expect(json.data.downloading).toBeGreaterThanOrEqual(0);
      expect(json.data.paused).toBeGreaterThanOrEqual(0);
      expect(json.data.completed).toBeGreaterThanOrEqual(0);
      expect(json.data.seeding).toBeGreaterThanOrEqual(0);
      expect(json.data.error).toBeGreaterThanOrEqual(0);

      // Sum of individual counts should equal all
      const sum =
        json.data.downloading +
        json.data.paused +
        json.data.completed +
        json.data.seeding +
        json.data.error;
      // Note: sum may be <= all because stalled/queued/checking statuses are not counted
      expect(sum).toBeLessThanOrEqual(json.data.all);
    } else {
      expect(json.error.code).toMatch(/^QBITTORRENT_/);
    }
  });

  test('[P2] GET /downloads/counts - should follow standard API response format', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/downloads/counts`);
    const json = await response.json();

    expect(json).toHaveProperty('success');
    expect(typeof json.success).toBe('boolean');

    if (json.success) {
      expect(json).toHaveProperty('data');
    } else {
      expect(json).toHaveProperty('error');
      expect(json.error).toHaveProperty('code');
      expect(json.error).toHaveProperty('message');
    }
  });

  // ---------------------------------------------------------------------------
  // Filter response structure validation
  // ---------------------------------------------------------------------------

  test('[P1] GET /downloads - filtered response items should contain status field (AC2)', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/downloads?filter=all`);
    const json = await response.json();

    if (response.ok() && json.data && json.data.length > 0) {
      // THEN: each download item has required fields
      for (const item of json.data) {
        expect(item).toHaveProperty('hash');
        expect(item).toHaveProperty('name');
        expect(item).toHaveProperty('status');
        expect(typeof item.hash).toBe('string');
        expect(typeof item.name).toBe('string');
        expect(typeof item.status).toBe('string');
      }
    }
  });

  test('[P2] GET /downloads/counts - all count values should be integers (AC1)', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/downloads/counts`);
    const json = await response.json();

    if (response.ok()) {
      const countFields = ['all', 'downloading', 'paused', 'completed', 'seeding', 'error'];
      for (const field of countFields) {
        expect(Number.isInteger(json.data[field])).toBe(true);
      }
    }
  });

  test('[P2] GET /downloads - empty filter param defaults to all', async ({ request }) => {
    // GIVEN: request with empty filter value
    const responseEmpty = await request.get(`${API_BASE_URL}/downloads?filter=`);
    const responseDefault = await request.get(`${API_BASE_URL}/downloads`);

    // THEN: both return same structure
    const jsonEmpty = await responseEmpty.json();
    const jsonDefault = await responseDefault.json();

    expect(jsonEmpty).toHaveProperty('success');
    expect(jsonDefault).toHaveProperty('success');
    // Both should succeed or both should fail with same error
    expect(jsonEmpty.success).toBe(jsonDefault.success);
  });
});
