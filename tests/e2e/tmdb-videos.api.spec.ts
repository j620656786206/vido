/**
 * TMDb Videos API E2E Smoke Tests (Story 10-2)
 *
 * Contract-level coverage for the two new videos endpoints introduced in
 * Story 10-2 Task 3.3 (Go client existed; HTTP route was added):
 *   - GET /api/v1/tmdb/movies/:id/videos
 *   - GET /api/v1/tmdb/tv/:id/videos
 *
 * Exercises the full stack (Handler → Service → Client → TMDb API) and
 * verifies the canonical `ApiResponse<T>` envelope with snake_case keys
 * (Rule 18). These endpoints bypass cache (small ephemeral payload) so we
 * don't assert cache-hit behaviour — only shape and validation.
 *
 * Prerequisites:
 *   - Backend running on port 8080 with TMDB_API_KEY configured.
 *     If TMDB_API_KEY is unset the suite is skipped (same pattern as
 *     tmdb-trending-discover.api.spec.ts).
 *
 * @tags @api @tmdb @hero-banner @story-10-2 @p2
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
const HAS_TMDB_KEY = !!process.env.TMDB_API_KEY;

// Known-stable TMDb IDs used across the codebase for smoke tests.
// Fight Club — decades-old, unlikely to disappear from TMDb.
const MOVIE_ID = 550;
// Breaking Bad.
const TV_ID = 1396;

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

interface Video {
  id: string;
  iso_639_1?: string;
  iso_3166_1?: string;
  name: string;
  key: string;
  site: string;
  size?: number;
  type: string;
  official: boolean;
  published_at?: string;
}

interface VideosResponse {
  id: number;
  results: Video[];
}

test.describe('TMDb Videos API @api @tmdb @story-10-2', () => {
  test.skip(!HAS_TMDB_KEY, 'TMDB_API_KEY not configured — skipping live TMDb smoke');

  test('[P1] GET /tmdb/movies/:id/videos returns ApiResponse envelope with snake_case fields', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/tmdb/movies/${MOVIE_ID}/videos`);
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<VideosResponse>;
    expect(body.success).toBe(true);
    expect(body.error).toBeUndefined();
    expect(body.data).toBeDefined();
    expect(body.data!.id).toBe(MOVIE_ID);
    expect(Array.isArray(body.data!.results)).toBe(true);

    // Rule 18: wire format stays snake_case — `published_at`, not `publishedAt`.
    if (body.data!.results.length > 0) {
      const v = body.data!.results[0];
      expect(typeof v.key).toBe('string');
      expect(typeof v.site).toBe('string');
      expect(typeof v.official).toBe('boolean');
      // Either snake_case key exists or is absent (TMDb can omit published_at on
      // older rows); what we MUST NOT see is a camelCase leak.
      expect(Object.prototype.hasOwnProperty.call(v, 'publishedAt')).toBe(false);
    }
  });

  test('[P1] GET /tmdb/tv/:id/videos returns ApiResponse envelope', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/tmdb/tv/${TV_ID}/videos`);
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<VideosResponse>;
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data!.id).toBe(TV_ID);
    expect(Array.isArray(body.data!.results)).toBe(true);
  });

  test('[P2] GET /tmdb/movies/:id/videos rejects invalid id (0)', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/tmdb/movies/0/videos`);
    // Service returns BadRequestError for id ≤ 0 (see tmdb_service.go:GetMovieVideos).
    expect(response.status()).toBe(400);

    const body = (await response.json()) as ApiResponse<never>;
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
  });

  test('[P2] GET /tmdb/movies/:id/videos rejects non-numeric id', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/tmdb/movies/not-a-number/videos`);
    // Gin route parsing: non-numeric :id fails parameter binding before service.
    expect(response.status()).toBeGreaterThanOrEqual(400);
    expect(response.status()).toBeLessThan(500);
  });

  test('[P2] GET /tmdb/movies/:id/videos returns 404 or empty results for unknown id', async ({
    request,
  }) => {
    // An impossibly-large ID that TMDb will not resolve.
    const response = await request.get(`${API_BASE_URL}/tmdb/movies/999999999/videos`);

    // TMDb may return 404 (upstream) or 200 with empty results depending on how
    // the service wraps the error. Accept either — the contract is: don't
    // explode and return an ApiResponse envelope.
    expect([200, 404]).toContain(response.status());
    const body = (await response.json()) as ApiResponse<VideosResponse>;
    expect(typeof body.success).toBe('boolean');
    if (body.success) {
      expect(Array.isArray(body.data!.results)).toBe(true);
      expect(body.data!.results.length).toBe(0);
    }
  });
});
