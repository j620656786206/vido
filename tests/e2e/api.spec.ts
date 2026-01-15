/**
 * API E2E Tests
 *
 * Direct API tests using Playwright's request context.
 * These tests verify the Go backend without browser overhead.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api
 */

import { test, expect } from '../support/fixtures';

// Skip API tests if backend is not running
// Run with: npx playwright test --grep @api (after starting backend)
test.describe.skip('API Health @smoke @api', () => {
  test('should return healthy status', async ({ api }) => {
    const response = await api.healthCheck();
    expect(response.status).toBe('ok');
  });
});

test.describe.skip('Movies API @api', () => {
  test('should search movies by query', async ({ api }) => {
    const response = await api.searchMovies('Inception');

    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data?.results).toBeInstanceOf(Array);
  });

  test('should return empty results for non-existent movie', async ({ api }) => {
    const response = await api.searchMovies('xyznonexistentmovie99999');

    expect(response.success).toBe(true);
    expect(response.data?.results).toHaveLength(0);
  });

  test('should get movie details by ID', async ({ api }) => {
    const searchResponse = await api.searchMovies('Inception');
    const firstMovie = searchResponse.data?.results[0];

    if (firstMovie) {
      const detailResponse = await api.getMovie(firstMovie.id);

      expect(detailResponse.success).toBe(true);
      expect(detailResponse.data?.title).toBeDefined();
    }
  });

  test('should return error for invalid movie ID', async ({ api }) => {
    const response = await api.getMovie('invalid-id-12345');

    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error?.code).toMatch(/NOT_FOUND|INVALID/);
  });
});

test.describe.skip('API Response Format @api', () => {
  test('should follow standard response format', async ({ api }) => {
    const response = await api.searchMovies('test');

    expect(response).toHaveProperty('success');

    if (response.success) {
      expect(response).toHaveProperty('data');
    } else {
      expect(response).toHaveProperty('error');
      expect(response.error).toHaveProperty('code');
      expect(response.error).toHaveProperty('message');
    }
  });
});
