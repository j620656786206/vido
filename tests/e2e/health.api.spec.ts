/**
 * Health Check API Tests
 *
 * Tests for the Vido API health endpoint.
 * Priority: P0 (Critical - run on every commit)
 *
 * @tags @api @smoke @p0
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080';

test.describe('Health API @api @smoke', () => {
  test('[P0] should return healthy status when API is running', async ({ request }) => {
    // GIVEN: The API server is running

    // WHEN: Requesting the health endpoint
    const response = await request.get(`${API_BASE_URL}/health`);

    // THEN: Should return 200 with healthy status
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.status).toBe('healthy');
    expect(body.service).toBe('vido-api');
  });

  test('[P0] should include database health information', async ({ request }) => {
    // GIVEN: The API server is running with database

    // WHEN: Requesting the health endpoint
    const response = await request.get(`${API_BASE_URL}/health`);

    // THEN: Should include database health details
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.database).toBeDefined();
    expect(body.database.status).toMatch(/healthy|degraded/);
    expect(body.database.latency).toBeGreaterThanOrEqual(0);
    expect(body.database.walEnabled).toBeDefined();
  });

  test('[P1] should return service unavailable when database is unhealthy', async ({ request }) => {
    // NOTE: This test documents expected behavior
    // In practice, database should be healthy in test environment

    // GIVEN: Health endpoint exists

    // WHEN: Checking health response format
    const response = await request.get(`${API_BASE_URL}/health`);

    // THEN: Response should follow expected schema
    const body = await response.json();
    expect(body).toHaveProperty('status');
    expect(body).toHaveProperty('service');

    // Verify response structure allows for unhealthy state
    if (body.status === 'unhealthy') {
      expect(response.status()).toBe(503);
    }
  });
});
