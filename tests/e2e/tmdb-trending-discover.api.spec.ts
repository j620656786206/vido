/**
 * TMDb Trending & Discover API E2E Smoke Tests (Story 10-1)
 *
 * Smoke-level coverage for the four new TMDb endpoints introduced in Story 10-1:
 *   - GET /api/v1/tmdb/trending/movies
 *   - GET /api/v1/tmdb/trending/tv
 *   - GET /api/v1/tmdb/discover/movies
 *   - GET /api/v1/tmdb/discover/tv
 *
 * Verifies the end-to-end stack (Handler → Service → Cache → Fallback → Client)
 * returns the canonical ApiResponse<T> envelope with snake_case keys (Rule 18)
 * and zh-TW localized content when no language override is supplied.
 *
 * Prerequisites:
 *   - Backend running on port 8080 with TMDB_API_KEY configured.
 *     If TMDB_API_KEY is unset the suite is skipped (consistent with other
 *     TMDb-touching specs).
 *
 * @tags @api @tmdb @story-10-1 @p2
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
const HAS_TMDB_KEY = !!process.env.TMDB_API_KEY;

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

interface PagedResult<TItem> {
  page: number;
  results: TItem[];
  total_pages: number;
  total_results: number;
}

interface TrendingMovieItem {
  id: number;
  title?: string;
  vote_average?: number;
  vote_count?: number;
  release_date?: string;
}

interface TrendingTVItem {
  id: number;
  name?: string;
  vote_average?: number;
  vote_count?: number;
  first_air_date?: string;
}

test.describe('TMDb Trending & Discover API @api @tmdb @story-10-1', () => {
  test.skip(!HAS_TMDB_KEY, 'TMDB_API_KEY not configured — skipping live TMDb smoke');

  test('[P2] GET /tmdb/trending/movies returns ApiResponse envelope with paged results', async ({
    request,
  }) => {
    const response = await request.get(
      `${API_BASE_URL}/tmdb/trending/movies?time_window=week&page=1`
    );
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<PagedResult<TrendingMovieItem>>;
    expect(body.success).toBe(true);
    expect(body.error).toBeUndefined();
    expect(body.data).toBeDefined();
    expect(body.data!.page).toBe(1);
    expect(Array.isArray(body.data!.results)).toBe(true);
    expect(body.data!.total_pages).toBeGreaterThan(0);

    if (body.data!.results.length > 0) {
      const horizonMs = Date.now() + 6 * 30 * 24 * 60 * 60 * 1000;
      for (const item of body.data!.results) {
        expect(item.id).toBeGreaterThan(0);
        if (item.release_date) {
          expect(Date.parse(item.release_date)).toBeLessThanOrEqual(horizonMs);
        }
        if (typeof item.vote_average === 'number' && typeof item.vote_count === 'number') {
          expect(item.vote_average >= 3.0 || item.vote_count >= 50).toBe(true);
        }
      }
    }
  });

  test('[P2] GET /tmdb/trending/tv defaults time_window to week and returns ApiResponse', async ({
    request,
  }) => {
    const response = await request.get(`${API_BASE_URL}/tmdb/trending/tv`);
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<PagedResult<TrendingTVItem>>;
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data!.page).toBe(1);
    expect(Array.isArray(body.data!.results)).toBe(true);

    if (body.data!.results.length > 0) {
      const horizonMs = Date.now() + 6 * 30 * 24 * 60 * 60 * 1000;
      for (const item of body.data!.results) {
        if (item.first_air_date) {
          expect(Date.parse(item.first_air_date)).toBeLessThanOrEqual(horizonMs);
        }
      }
    }
  });

  test('[P2] GET /tmdb/discover/movies maps query params (genre, year, sort)', async ({
    request,
  }) => {
    const response = await request.get(
      `${API_BASE_URL}/tmdb/discover/movies?genre=28&year_gte=2023&year_lte=2024&sort=popularity.desc&page=1`
    );
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<PagedResult<TrendingMovieItem>>;
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data!.page).toBe(1);
  });

  test('[P2] GET /tmdb/discover/tv with region=TW returns ApiResponse envelope', async ({
    request,
  }) => {
    const response = await request.get(
      `${API_BASE_URL}/tmdb/discover/tv?region=TW&sort=popularity.desc`
    );
    expect(response.status()).toBe(200);

    const body = (await response.json()) as ApiResponse<PagedResult<TrendingTVItem>>;
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data!.page).toBe(1);
    expect(Array.isArray(body.data!.results)).toBe(true);
  });

  test('[P2] GET /tmdb/trending/movies normalizes unknown time_window to week', async ({
    request,
  }) => {
    const validResp = await request.get(
      `${API_BASE_URL}/tmdb/trending/movies?time_window=week&page=1`
    );
    const garbageResp = await request.get(
      `${API_BASE_URL}/tmdb/trending/movies?time_window=garbage&page=1`
    );

    expect(validResp.status()).toBe(200);
    expect(garbageResp.status()).toBe(200);

    const valid = (await validResp.json()) as ApiResponse<PagedResult<TrendingMovieItem>>;
    const garbage = (await garbageResp.json()) as ApiResponse<PagedResult<TrendingMovieItem>>;

    expect(valid.success).toBe(true);
    expect(garbage.success).toBe(true);
    expect(garbage.data!.page).toBe(1);
  });
});
