/**
 * Services Health API Tests
 *
 * Tests for the Vido API services health endpoint (Story 3-12).
 * Priority: P1 (High - run on PR to main)
 *
 * @tags @api @p1 @degradation
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = 'http://localhost:8080/api/v1';

// =============================================================================
// Type Definitions
// =============================================================================

interface ServiceHealth {
  name: string;
  display_name: string;
  status: 'healthy' | 'degraded' | 'down';
  last_check: string;
  last_success: string;
  error_count: number;
  message?: string;
}

interface ServicesHealth {
  tmdb: ServiceHealth;
  douban: ServiceHealth;
  wikipedia: ServiceHealth;
  ai: ServiceHealth;
}

interface HealthStatusResponse {
  degradation_level: 'normal' | 'partial' | 'minimal' | 'offline';
  services: ServicesHealth;
  message: string;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

// =============================================================================
// Services Health API Tests
// =============================================================================

test.describe('Services Health API @api @p1', () => {
  test('[P1] should return services health status with all services', async ({ request }) => {
    // GIVEN: The API server is running with health monitoring

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Should return 200 with success response
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
  });

  test('[P1] should include degradation level in response', async ({ request }) => {
    // GIVEN: The API server is running

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Should include valid degradation level
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();
    expect(body.data?.degradation_level).toMatch(/^(normal|partial|minimal|offline)$/);
  });

  test('[P1] should include all four external services', async ({ request }) => {
    // GIVEN: The API server is running with health monitoring

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Should include TMDb, Douban, Wikipedia, and AI services
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();
    const services = body.data?.services;

    expect(services).toBeDefined();
    expect(services?.tmdb).toBeDefined();
    expect(services?.douban).toBeDefined();
    expect(services?.wikipedia).toBeDefined();
    expect(services?.ai).toBeDefined();
  });

  test('[P1] should include service health details', async ({ request }) => {
    // GIVEN: The API server is running

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Each service should have required health fields
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();
    const tmdb = body.data?.services?.tmdb;

    expect(tmdb?.name).toBe('tmdb');
    expect(tmdb?.display_name).toBeDefined();
    expect(tmdb?.status).toMatch(/^(healthy|degraded|down)$/);
    expect(tmdb?.last_check).toBeDefined();
    expect(tmdb?.last_success).toBeDefined();
    expect(typeof tmdb?.error_count).toBe('number');
  });

  test('[P1] should follow API response format', async ({ request }) => {
    // GIVEN: The API server is running

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Should follow standard API response format
    const body = await response.json();

    // Following project-context.md Rule 3: API Response Format
    expect(body).toHaveProperty('success');
    expect(typeof body.success).toBe('boolean');

    if (body.success) {
      expect(body).toHaveProperty('data');
    } else {
      expect(body).toHaveProperty('error');
      expect(body.error).toHaveProperty('code');
      expect(body.error).toHaveProperty('message');
    }
  });

  test('[P1] should include status message when degraded', async ({ request }) => {
    // GIVEN: The services health endpoint

    // WHEN: Requesting health status
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Response should have message field
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();

    // Message field should exist (may be empty if all healthy)
    expect(body.data).toHaveProperty('message');
    expect(typeof body.data?.message).toBe('string');

    // If degraded, message should contain affected service info
    if (body.data?.degradation_level !== 'normal') {
      expect(body.data?.message.length).toBeGreaterThan(0);
    }
  });

  test('[P2] should return ISO 8601 timestamps', async ({ request }) => {
    // GIVEN: The API server is running

    // WHEN: Requesting the services health endpoint
    const response = await request.get(`${API_BASE_URL}/health/services`);

    // THEN: Timestamps should be in ISO 8601 format
    expect(response.status()).toBe(200);

    const body: ApiResponse<HealthStatusResponse> = await response.json();
    const tmdb = body.data?.services?.tmdb;

    // ISO 8601 regex pattern
    const isoPattern = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/;

    if (tmdb?.last_check) {
      expect(tmdb.last_check).toMatch(isoPattern);
    }
    if (tmdb?.last_success) {
      expect(tmdb.last_success).toMatch(isoPattern);
    }
  });
});

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

test.describe('Services Health API - Edge Cases @api @p2', () => {
  test('[P2] should handle concurrent requests', async ({ request }) => {
    // GIVEN: Multiple concurrent requests

    // WHEN: Sending multiple requests simultaneously
    const requests = Array(5)
      .fill(null)
      .map(() => request.get(`${API_BASE_URL}/health/services`));

    const responses = await Promise.all(requests);

    // THEN: All requests should succeed
    responses.forEach((response) => {
      expect(response.status()).toBe(200);
    });
  });

  test('[P2] should have reasonable response time', async ({ request }) => {
    // GIVEN: Health endpoint should be fast

    // WHEN: Measuring response time
    const startTime = Date.now();
    const response = await request.get(`${API_BASE_URL}/health/services`);
    const duration = Date.now() - startTime;

    // THEN: Should respond within 2 seconds
    expect(response.status()).toBe(200);
    expect(duration).toBeLessThan(2000);
  });
});
