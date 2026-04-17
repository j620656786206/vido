/**
 * Availability API Tests (Story 10-4)
 *
 * Backend contract tests for POST /api/v1/media/check-owned. These complement
 * the Go unit tests (handler + service + repo) by proving the endpoint is
 * actually mounted, accepts snake_case input, and returns the
 * `{success, data: {owned_ids: []}}` envelope end-to-end.
 *
 * Pure API tests — no browser. Uses Playwright's `request` fixture.
 *
 * Prerequisites:
 *   - Backend running on port 8080 (global-setup ensures this).
 *   - Library can be empty — an empty library still returns `owned_ids: []`,
 *     which validates the contract without requiring seeded DB data.
 *
 * @tags @api @availability @story-10-4
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
const ENDPOINT = `${API_BASE_URL}/media/check-owned`;

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

interface CheckOwnedResponse {
  owned_ids: number[];
}

test.describe('POST /api/v1/media/check-owned @api @availability @story-10-4', () => {
  test('[P1] returns owned_ids array for unknown TMDb IDs (contract shape)', async ({
    request,
  }) => {
    // Use astronomically unlikely TMDb IDs so the test is insensitive to
    // whatever the local dev library happens to contain.
    const response = await request.post(ENDPOINT, {
      data: { tmdb_ids: [9000001, 9000002, 9000003] },
    });

    expect(response.status()).toBe(200);
    const body = (await response.json()) as ApiResponse<CheckOwnedResponse>;

    expect(body.success).toBe(true);
    expect(body.error).toBeUndefined();
    expect(body.data).toBeDefined();
    // Critical contract guarantees:
    //   1. owned_ids is always an array (never null — handler normalises)
    //   2. snake_case key at the wire (Rule 18 backend format)
    expect(Array.isArray(body.data!.owned_ids)).toBe(true);
    // Nothing we queried can match (IDs are out-of-range for any real TMDb
    // record), so the result must be empty.
    expect(body.data!.owned_ids).toEqual([]);
  });

  test('[P1] returns 200 + owned_ids: [] for empty array input', async ({ request }) => {
    // DEV's handler short-circuits empty input server-side. Contract check:
    // no error, empty array back.
    const response = await request.post(ENDPOINT, {
      data: { tmdb_ids: [] },
    });

    expect(response.status()).toBe(200);
    const body = (await response.json()) as ApiResponse<CheckOwnedResponse>;
    expect(body.success).toBe(true);
    expect(body.data!.owned_ids).toEqual([]);
  });

  test('[P1] returns 400 when tmdb_ids field is missing', async ({ request }) => {
    const response = await request.post(ENDPOINT, { data: {} });

    expect(response.status()).toBe(400);
    const body = (await response.json()) as ApiResponse<CheckOwnedResponse>;
    expect(body.success).toBe(false);
    expect(body.error?.code).toBe('VALIDATION_INVALID_FORMAT');
  });

  test('[P2] rejects requests with more than 500 IDs (over-limit guard)', async ({ request }) => {
    // The handler caps at 500. Build a 501-element payload.
    const overLimit = Array.from({ length: 501 }, (_, i) => 10000000 + i);

    const response = await request.post(ENDPOINT, {
      data: { tmdb_ids: overLimit },
    });

    expect(response.status()).toBe(400);
    const body = (await response.json()) as ApiResponse<CheckOwnedResponse>;
    expect(body.success).toBe(false);
    // Body message should mention the cap so a caller learns the limit.
    expect(body.error?.message.toLowerCase()).toContain('500');
  });
});
