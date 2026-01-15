/**
 * Settings API E2E Tests
 *
 * Comprehensive API tests for the Settings CRUD endpoints.
 * Tests follow Given-When-Then format with priority tags.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @settings
 */

import { test, expect } from '@playwright/test';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Settings API Tests
// =============================================================================

test.describe('Settings API - CRUD Operations @api @settings', () => {
  const testKeys: string[] = [];

  test.afterEach(async ({ request }) => {
    // Cleanup: Delete all created test settings
    for (const key of testKeys) {
      await request.delete(`${API_BASE_URL}/settings/${key}`);
    }
    testKeys.length = 0;
  });

  // ---------------------------------------------------------------------------
  // SET Tests (Create/Update)
  // ---------------------------------------------------------------------------

  test('[P2] POST /settings - should set string setting', async ({ request }) => {
    // GIVEN: Valid string setting data
    const settingKey = `test_string_${Date.now()}`;
    const settingData = {
      key: settingKey,
      value: 'test-value-123',
      type: 'string',
    };
    testKeys.push(settingKey);

    // WHEN: Setting a string value
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 200 with saved setting
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.key).toBe(settingKey);
  });

  test('[P2] POST /settings - should set integer setting', async ({ request }) => {
    // GIVEN: Valid integer setting data
    const settingKey = `test_int_${Date.now()}`;
    const settingData = {
      key: settingKey,
      value: 42,
      type: 'int',
    };
    testKeys.push(settingKey);

    // WHEN: Setting an integer value
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 200 with saved setting
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.key).toBe(settingKey);
  });

  test('[P2] POST /settings - should set boolean setting', async ({ request }) => {
    // GIVEN: Valid boolean setting data
    const settingKey = `test_bool_${Date.now()}`;
    const settingData = {
      key: settingKey,
      value: true,
      type: 'bool',
    };
    testKeys.push(settingKey);

    // WHEN: Setting a boolean value
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 200 with saved setting
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.key).toBe(settingKey);
  });

  test('[P2] POST /settings - should return 400 for invalid type', async ({ request }) => {
    // GIVEN: Setting with invalid type
    const settingData = {
      key: 'test_invalid_type',
      value: 'some-value',
      type: 'invalid_type',
    };

    // WHEN: Attempting to set with invalid type
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P2] POST /settings - should return 400 for missing key', async ({ request }) => {
    // GIVEN: Setting without key
    const settingData = {
      value: 'test-value',
      type: 'string',
    };

    // WHEN: Attempting to set without key
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P2] POST /settings - should return 400 for type mismatch', async ({ request }) => {
    // GIVEN: String value with int type
    const settingData = {
      key: 'test_type_mismatch',
      value: 'not-a-number',
      type: 'int',
    };

    // WHEN: Attempting to set mismatched type
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: settingData,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  // ---------------------------------------------------------------------------
  // GET Tests
  // ---------------------------------------------------------------------------

  test('[P2] GET /settings - should list all settings', async ({ request }) => {
    // GIVEN: Settings exist in database
    const settingKey = `test_list_${Date.now()}`;
    await request.post(`${API_BASE_URL}/settings`, {
      data: { key: settingKey, value: 'list-test', type: 'string' },
    });
    testKeys.push(settingKey);

    // WHEN: Listing all settings
    const response = await request.get(`${API_BASE_URL}/settings`);

    // THEN: Should return list of settings
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeInstanceOf(Array);
  });

  test('[P2] GET /settings/:key - should return setting by key', async ({ request }) => {
    // GIVEN: A setting exists
    const settingKey = `test_get_${Date.now()}`;
    await request.post(`${API_BASE_URL}/settings`, {
      data: { key: settingKey, value: 'get-test-value', type: 'string' },
    });
    testKeys.push(settingKey);

    // WHEN: Getting setting by key
    const response = await request.get(`${API_BASE_URL}/settings/${settingKey}`);

    // THEN: Should return the setting
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.key).toBe(settingKey);
  });

  test('[P2] GET /settings/:key - should return 404 for non-existent setting', async ({ request }) => {
    // GIVEN: A non-existent setting key
    const fakeKey = 'non_existent_setting_key_12345';

    // WHEN: Getting setting by fake key
    const response = await request.get(`${API_BASE_URL}/settings/${fakeKey}`);

    // THEN: Should return 404 not found
    expect(response.status()).toBe(404);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('DB_NOT_FOUND');
  });

  // ---------------------------------------------------------------------------
  // DELETE Tests
  // ---------------------------------------------------------------------------

  test('[P2] DELETE /settings/:key - should delete setting', async ({ request }) => {
    // GIVEN: A setting exists
    const settingKey = `test_delete_${Date.now()}`;
    await request.post(`${API_BASE_URL}/settings`, {
      data: { key: settingKey, value: 'delete-test', type: 'string' },
    });

    // WHEN: Deleting the setting
    const response = await request.delete(`${API_BASE_URL}/settings/${settingKey}`);

    // THEN: Should return 204 No Content
    expect(response.status()).toBe(204);

    // Verify setting is deleted
    const getResponse = await request.get(`${API_BASE_URL}/settings/${settingKey}`);
    expect(getResponse.status()).toBe(404);
  });

  // ---------------------------------------------------------------------------
  // UPDATE (Overwrite) Tests
  // ---------------------------------------------------------------------------

  test('[P2] POST /settings - should overwrite existing setting', async ({ request }) => {
    // GIVEN: A setting exists
    const settingKey = `test_overwrite_${Date.now()}`;
    await request.post(`${API_BASE_URL}/settings`, {
      data: { key: settingKey, value: 'original-value', type: 'string' },
    });
    testKeys.push(settingKey);

    // WHEN: Setting the same key with new value
    const response = await request.post(`${API_BASE_URL}/settings`, {
      data: { key: settingKey, value: 'updated-value', type: 'string' },
    });

    // THEN: Should update the setting
    expect(response.status()).toBe(200);

    // Verify the update
    const getResponse = await request.get(`${API_BASE_URL}/settings/${settingKey}`);
    const body = await getResponse.json();
    expect(body.data.value).toBe('updated-value');
  });
});

// =============================================================================
// Response Format Validation
// =============================================================================

test.describe('Settings API - Response Format @api @settings', () => {
  test('[P2] should follow standard API response format', async ({ request }) => {
    // GIVEN: API is running

    // WHEN: Making any request
    const response = await request.get(`${API_BASE_URL}/settings`);

    // THEN: Response should follow standard format
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('success');
    expect(body.success).toBe(true);
    expect(body).toHaveProperty('data');
  });
});
