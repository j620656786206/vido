/**
 * Downloads API E2E Tests (Story 4.2)
 *
 * Tests for real-time download status monitoring endpoints.
 * Tests follow Given-When-Then format.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * Note: These tests work against the live API. If qBittorrent
 * is not configured, they validate graceful error handling.
 *
 * @tags @api @downloads
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Downloads API Tests
// =============================================================================

test.describe('Downloads API @api @downloads', () => {
  // ---------------------------------------------------------------------------
  // GET /downloads - List all downloads
  // ---------------------------------------------------------------------------

  test('[P1] GET /downloads - should return download list or not-configured error (AC1)', async ({
    request,
  }) => {
    // WHEN: Fetching download list
    const response = await request.get(`${API_BASE_URL}/downloads`);

    const json = await response.json();

    if (response.ok()) {
      // THEN: Should return success with array of torrents
      expect(json.success).toBe(true);
      expect(Array.isArray(json.data)).toBe(true);

      // Validate torrent structure if any exist
      if (json.data.length > 0) {
        const torrent = json.data[0];
        expect(torrent).toHaveProperty('hash');
        expect(torrent).toHaveProperty('name');
        expect(torrent).toHaveProperty('size');
        expect(torrent).toHaveProperty('progress');
        expect(torrent).toHaveProperty('downloadSpeed');
        expect(torrent).toHaveProperty('uploadSpeed');
        expect(torrent).toHaveProperty('eta');
        expect(torrent).toHaveProperty('status');
        expect(torrent).toHaveProperty('addedOn');
        expect(torrent).toHaveProperty('seeds');
        expect(torrent).toHaveProperty('peers');
        expect(torrent).toHaveProperty('savePath');

        // Validate status is one of expected values
        expect([
          'downloading',
          'paused',
          'seeding',
          'completed',
          'stalled',
          'error',
          'queued',
          'checking',
        ]).toContain(torrent.status);
      }
    } else {
      // THEN: Should return QB_NOT_CONFIGURED if qBittorrent is not set up
      expect(json.success).toBe(false);
      expect(json.error).toBeDefined();
      expect(json.error.code).toMatch(/^QB_/);
    }
  });

  test('[P2] GET /downloads - should accept sort parameters (AC5)', async ({ request }) => {
    // WHEN: Fetching downloads with sort parameters
    const response = await request.get(`${API_BASE_URL}/downloads?sort=name&order=asc`);

    const json = await response.json();

    // THEN: Should accept sort params without error (or return QB_NOT_CONFIGURED)
    if (response.ok()) {
      expect(json.success).toBe(true);
      expect(Array.isArray(json.data)).toBe(true);
    } else {
      expect(json.error.code).toMatch(/^QB_/);
    }
  });

  test('[P2] GET /downloads - should accept all sort fields (AC5)', async ({ request }) => {
    const sortFields = ['added_on', 'name', 'progress', 'size'];

    for (const sort of sortFields) {
      const response = await request.get(`${API_BASE_URL}/downloads?sort=${sort}&order=desc`);
      const json = await response.json();

      // Should not crash regardless of sort field
      expect(json).toHaveProperty('success');
    }
  });

  // ---------------------------------------------------------------------------
  // GET /downloads/:hash - Get download details
  // ---------------------------------------------------------------------------

  test('[P1] GET /downloads/:hash - should return 404 for nonexistent hash (AC4)', async ({
    request,
  }) => {
    // WHEN: Fetching details for a nonexistent torrent
    const response = await request.get(`${API_BASE_URL}/downloads/nonexistent_hash_12345`);

    const json = await response.json();

    // THEN: Should return error (either not found or not configured)
    expect(json.success).toBe(false);
    expect(json.error).toBeDefined();
    // Could be QB_TORRENT_NOT_FOUND or QB_NOT_CONFIGURED
    expect(json.error.code).toBeTruthy();
  });

  test('[P1] GET /downloads/:hash - should return details if torrent exists (AC4)', async ({
    request,
  }) => {
    // GIVEN: Get list of downloads first
    const listResponse = await request.get(`${API_BASE_URL}/downloads`);
    const listJson = await listResponse.json();

    // Skip if qBittorrent is not configured or no torrents exist
    if (!listResponse.ok() || !listJson.data || listJson.data.length === 0) {
      test.skip();
      return;
    }

    const firstHash = listJson.data[0].hash;

    // WHEN: Fetching details for the first torrent
    const response = await request.get(`${API_BASE_URL}/downloads/${firstHash}`);

    // THEN: Should return success with detailed torrent info
    expect(response.ok()).toBe(true);
    const json = await response.json();
    expect(json.success).toBe(true);
    expect(json.data).toHaveProperty('hash', firstHash);
    expect(json.data).toHaveProperty('name');
    expect(json.data).toHaveProperty('pieceSize');
    expect(json.data).toHaveProperty('creationDate');
    expect(json.data).toHaveProperty('timeElapsed');
    expect(json.data).toHaveProperty('avgDownSpeed');
    expect(json.data).toHaveProperty('avgUpSpeed');
  });

  // ---------------------------------------------------------------------------
  // Response Format Validation
  // ---------------------------------------------------------------------------

  test('[P2] Downloads API - should follow standard API response format', async ({ request }) => {
    // WHEN: Making any downloads API call
    const response = await request.get(`${API_BASE_URL}/downloads`);
    const json = await response.json();

    // THEN: Should follow standard response format
    expect(json).toHaveProperty('success');
    expect(typeof json.success).toBe('boolean');

    if (json.success) {
      expect(json).toHaveProperty('data');
      expect(json).not.toHaveProperty('error');
    } else {
      expect(json).toHaveProperty('error');
      expect(json.error).toHaveProperty('code');
      expect(json.error).toHaveProperty('message');
    }
  });
});
