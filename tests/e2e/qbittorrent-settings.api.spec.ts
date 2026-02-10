/**
 * qBittorrent Settings API E2E Tests (Story 4.1)
 *
 * Tests for qBittorrent connection configuration endpoints.
 * Tests follow Given-When-Then format.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @qbittorrent @settings
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// qBittorrent Settings API Tests
// =============================================================================

test.describe('qBittorrent Settings API @api @qbittorrent', () => {
  // ---------------------------------------------------------------------------
  // GET /settings/qbittorrent
  // ---------------------------------------------------------------------------

  test('[P1] GET /settings/qbittorrent - should return config (AC1)', async ({ request }) => {
    // WHEN: Fetching qBittorrent configuration
    const response = await request.get(`${API_BASE_URL}/settings/qbittorrent`);

    // THEN: Should return success with config data
    expect(response.ok()).toBe(true);
    const json = await response.json();
    expect(json.success).toBe(true);
    expect(json.data).toHaveProperty('host');
    expect(json.data).toHaveProperty('username');
    expect(json.data).toHaveProperty('basePath');
    expect(json.data).toHaveProperty('configured');
    // Password should NEVER be in response (AC2)
    expect(json.data).not.toHaveProperty('password');
  });

  // ---------------------------------------------------------------------------
  // PUT /settings/qbittorrent
  // ---------------------------------------------------------------------------

  test('[P1] PUT /settings/qbittorrent - should save configuration (AC5)', async ({ request }) => {
    // GIVEN: Valid qBittorrent configuration
    const config = {
      host: 'http://192.168.1.100:8080',
      username: 'admin',
      password: 'test-password-123',
      basePath: '/qbt',
    };

    // WHEN: Saving the configuration
    const response = await request.put(`${API_BASE_URL}/settings/qbittorrent`, {
      data: config,
    });

    // THEN: Should save successfully
    expect(response.ok()).toBe(true);
    const json = await response.json();
    expect(json.success).toBe(true);

    // THEN: Should be retrievable (without password)
    const getResponse = await request.get(`${API_BASE_URL}/settings/qbittorrent`);
    const getJson = await getResponse.json();
    expect(getJson.data.host).toBe('http://192.168.1.100:8080');
    expect(getJson.data.username).toBe('admin');
    expect(getJson.data.basePath).toBe('/qbt');
    expect(getJson.data.configured).toBe(true);
    expect(getJson.data).not.toHaveProperty('password');
  });

  test('[P1] PUT /settings/qbittorrent - should reject missing required fields', async ({
    request,
  }) => {
    // GIVEN: Config missing required host
    const config = {
      username: 'admin',
      password: 'secret',
    };

    // WHEN: Attempting to save
    const response = await request.put(`${API_BASE_URL}/settings/qbittorrent`, {
      data: config,
    });

    // THEN: Should return 400
    expect(response.status()).toBe(400);
    const json = await response.json();
    expect(json.success).toBe(false);
  });

  // ---------------------------------------------------------------------------
  // POST /settings/qbittorrent/test
  // ---------------------------------------------------------------------------

  test('[P1] POST /settings/qbittorrent/test - should fail when not configured (AC3)', async ({
    request,
  }) => {
    // GIVEN: qBittorrent is not configured (or points to invalid host)
    // WHEN: Testing connection
    const response = await request.post(`${API_BASE_URL}/settings/qbittorrent/test`);

    // THEN: Should return error
    expect(response.status()).toBe(400);
    const json = await response.json();
    expect(json.success).toBe(false);
    expect(json.error).toBeTruthy();
    expect(json.error.code).toBe('QB_CONNECTION_FAILED');
  });

  test('[P2] POST /settings/qbittorrent/test - should return error details for invalid credentials (AC3)', async ({
    request,
  }) => {
    // GIVEN: Config pointing to non-existent host
    await request.put(`${API_BASE_URL}/settings/qbittorrent`, {
      data: {
        host: 'http://127.0.0.1:1',
        username: 'admin',
        password: 'wrong',
      },
    });

    // WHEN: Testing connection
    const response = await request.post(`${API_BASE_URL}/settings/qbittorrent/test`);

    // THEN: Should return connection error with details
    expect(response.status()).toBe(400);
    const json = await response.json();
    expect(json.success).toBe(false);
    expect(json.error.code).toBe('QB_CONNECTION_FAILED');
    expect(json.error.message).toBeTruthy();
    expect(json.error.suggestion).toBeTruthy();
  });
});
