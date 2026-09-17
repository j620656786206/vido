/**
 * Manual Search API Tests (Story 3-7)
 *
 * Tests for the manual metadata search and apply functionality.
 * These tests validate the API endpoints directly without UI interaction.
 *
 * Prerequisites:
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @api @metadata @story-3-7
 */

import { test, expect } from '../support/fixtures';
import { seedMovie, seedSeries, deleteMovies, deleteSeries } from '../support/helpers/seed-helpers';

// The library API serializes rows in snake_case; the helper's Movie type is
// camelCase, so read the fields a contract is about by their wire names.
const wire = (row: unknown) => row as Record<string, unknown>;

// =============================================================================
// Manual Search API Tests (AC1, AC4)
// =============================================================================

test.describe('Manual Search API @api @metadata', () => {
  test('[P1] POST /metadata/manual-search - should search all sources', async ({ api }) => {
    // GIVEN: A valid search request for all sources
    const searchRequest = {
      query: 'Inception',
      media_type: 'movie' as const,
      source: 'all' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return success with results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.results).toBeDefined();
    expect(response.data!.searched_sources).toContain('tmdb');
  });

  test('[P1] POST /metadata/manual-search - should search specific source (TMDb)', async ({
    api,
  }) => {
    // GIVEN: A search request for TMDb only
    const searchRequest = {
      query: 'Fight Club',
      media_type: 'movie' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return results from TMDb only
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.searched_sources).toEqual(['tmdb']);

    // All results should be from TMDb
    if (response.data!.results.length > 0) {
      response.data!.results.forEach((result) => {
        expect(result.source).toBe('tmdb');
      });
    }
  });

  test('[P1] POST /metadata/manual-search - should search TV shows', async ({ api }) => {
    // GIVEN: A search request for TV shows
    const searchRequest = {
      query: 'Breaking Bad',
      media_type: 'tv' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return TV show results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();

    if (response.data!.results.length > 0) {
      const firstResult = response.data!.results[0];
      expect(firstResult.media_type).toBe('tv');
    }
  });

  test('[P1] POST /metadata/manual-search - should filter by year', async ({ api }) => {
    // GIVEN: A search request with year filter
    const searchRequest = {
      query: 'Matrix',
      media_type: 'movie' as const,
      year: 1999,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
  });

  test('[P2] POST /metadata/manual-search - should return empty results for non-existent query', async ({
    api,
  }) => {
    // GIVEN: A search request for non-existent content
    const searchRequest = {
      query: 'xyznonexistentmovie99999abcdef',
      media_type: 'movie' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return success with empty results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.results).toEqual([]);
    expect(response.data!.total_count).toBe(0);
  });

  test('[P1] POST /metadata/manual-search - should return error for missing query', async ({
    api,
  }) => {
    // GIVEN: A search request without query
    const searchRequest = {
      query: '',
      media_type: 'movie' as const,
      source: 'all' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('MANUAL_SEARCH_INVALID_REQUEST');
  });

  test('[P2] POST /metadata/manual-search - should default to movie media type', async ({
    api,
  }) => {
    // GIVEN: A search request without media_type
    const searchRequest = {
      query: 'Inception',
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should succeed (defaults to movie)
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
  });

  test('[P2] POST /metadata/manual-search - should default to all sources', async ({ api }) => {
    // GIVEN: A search request without source
    const searchRequest = {
      query: 'Inception',
      media_type: 'movie' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should search all sources
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    // Should have searched at least TMDb
    expect(response.data!.searched_sources.length).toBeGreaterThanOrEqual(1);
  });

  test('[P1] POST /metadata/manual-search - should include source indicator in results (AC4)', async ({
    api,
  }) => {
    // GIVEN: A search request
    const searchRequest = {
      query: 'Inception',
      media_type: 'movie' as const,
      source: 'all' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Each result should have a source indicator
    expect(response.success).toBe(true);
    if (response.data!.results.length > 0) {
      response.data!.results.forEach((result) => {
        expect(result.source).toBeDefined();
        expect(['tmdb', 'douban', 'wikipedia']).toContain(result.source);
      });
    }
  });

  test('[P1] POST /metadata/manual-search - results should include required fields (AC2)', async ({
    api,
  }) => {
    // GIVEN: A search request that should return results
    const searchRequest = {
      query: 'Inception',
      media_type: 'movie' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Results should include poster, title, year, and description
    expect(response.success).toBe(true);
    if (response.data!.results.length > 0) {
      const firstResult = response.data!.results[0];
      expect(firstResult.id).toBeDefined();
      expect(firstResult.title).toBeDefined();
      expect(firstResult.year).toBeDefined();
      expect(firstResult.source).toBeDefined();
      // posterUrl and overview may be optional but should be defined if available
    }
  });
});

// =============================================================================
// Apply Metadata API Tests (AC3)
// =============================================================================

test.describe('Apply Metadata API @api @metadata', () => {
  // dsr-2b-a: until this story apply was a silent no-op in the running backend
  // (the service's updaters were never wired — it answered success with the
  // title "Unknown" and wrote nothing), so these tests could only check the
  // envelope. Apply now writes the picked TMDb match onto the row, so the
  // happy paths READ THE ROW BACK. They call real TMDb (CI's e2e backend has
  // TMDB_API_KEY).
  const movieIds: string[] = [];
  const seriesIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
    await deleteSeries(api, ...seriesIds.splice(0));
  });

  test('[P1] POST /metadata/apply - writes the picked movie match onto the row (AC3, dsr-2b-a AC #1)', async ({
    api,
  }) => {
    // GIVEN: An unmatched movie
    const movie = await seedMovie(api, { title: `[E2E] fc ${Date.now()}.mkv` });
    movieIds.push(movie.id);

    // WHEN: Applying the TMDb result the user picked
    const response = await api.applyMetadata({
      media_id: movie.id,
      media_type: 'movie',
      selected_item: { id: 'tmdb-550', source: 'tmdb', media_type: 'movie' },
    });

    // THEN: The response carries what was written…
    expect(response.success).toBe(true);
    expect(response.data!.media_id).toBe(movie.id);
    expect(response.data!.source).toBe('tmdb');
    expect(response.data!.tmdb_id).toBe(550);
    expect(response.data!.parse_status).toBe('success');
    expect(response.data!.title).not.toBe('Unknown');

    // …and the row really holds it.
    const row = wire((await api.getMovie(movie.id)).data);
    expect(row.tmdb_id).toBe(550);
    expect(row.parse_status).toBe('success');
    expect(row.metadata_source).toBe('manual');
    expect(row.title).toBe(response.data!.title);
  });

  test('[P1] POST /metadata/apply - writes the picked series match onto the row', async ({
    api,
  }) => {
    const series = await seedSeries(api, { title: `[E2E] bb ${Date.now()}` });
    seriesIds.push(series.id);

    const response = await api.applyMetadata({
      media_id: series.id,
      media_type: 'series',
      selected_item: { id: 'tmdb-1396', source: 'tmdb', media_type: 'tv' },
    });

    expect(response.success).toBe(true);
    expect(response.data!.media_type).toBe('series');
    expect(response.data!.tmdb_id).toBe(1396);
    const row = wire((await api.getSeries(series.id)).data);
    expect(row.tmdb_id).toBe(1396);
    expect(row.parse_status).toBe('success');
  });

  test('[P1] POST /metadata/apply - should return error for missing media_id', async ({ api }) => {
    const response = await api.applyMetadata({
      media_id: '',
      media_type: 'movie',
      selected_item: { id: 'tmdb-550', source: 'tmdb', media_type: 'movie' },
    });

    expect(response.success).toBe(false);
    expect(response.error!.code).toBe('APPLY_METADATA_INVALID_REQUEST');
  });

  test('[P1] POST /metadata/apply - rejects a TV result for a movie (dsr-2b-a AC #1)', async ({
    api,
  }) => {
    // tmdb-550 exists on BOTH sides and is a different work on each.
    const movie = await seedMovie(api, { title: `[E2E] mismatch ${Date.now()}.mkv` });
    movieIds.push(movie.id);

    const response = await api.applyMetadata({
      media_id: movie.id,
      media_type: 'movie',
      selected_item: { id: 'tmdb-550', source: 'tmdb', media_type: 'tv' },
    });

    expect(response.success).toBe(false);
    expect(response.error!.code).toBe('APPLY_METADATA_INVALID_REQUEST');
    expect(wire((await api.getMovie(movie.id)).data).tmdb_id).toBeFalsy();
  });

  test('[P1] POST /metadata/apply - should return error for non-existent media', async ({
    api,
  }) => {
    // Un-skipped by dsr-2b-a: the nil-updater branch that answered success for
    // any id is gone (was wire-set-media-updaters-test-harness).
    const response = await api.applyMetadata({
      media_id: 'nonexistent-media-id-12345',
      media_type: 'movie',
      selected_item: { id: 'tmdb-550', source: 'tmdb', media_type: 'movie' },
    });

    expect(response.success).toBe(false);
    expect(response.error!.code).toBe('APPLY_METADATA_NOT_FOUND');
  });

  test('[P2] POST /metadata/apply - should accept learnPattern flag for Story 3.9', async ({
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] learn ${Date.now()}.mkv` });
    movieIds.push(movie.id);

    const response = await api.applyMetadata({
      media_id: movie.id,
      media_type: 'movie',
      selected_item: { id: 'tmdb-550', source: 'tmdb', media_type: 'movie' },
      learn_pattern: true,
    });

    // learn_pattern is accepted but still does nothing (Story 3.9 TODO).
    expect(response.success).toBe(true);
  });
});

// =============================================================================
// Single-item re-match (dsr-2b-a AC #2)
// =============================================================================

test.describe('Library re-match API @api @metadata @dsr-2b-a', () => {
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  test('[P1] POST /library/movies/:id/reparse - ran but found nothing is a 200 carrying failed', async ({
    api,
  }) => {
    // This is also how dsr-2b-b seeds a "match failed" item for its detail-page e2e.
    const movie = await seedMovie(api, { title: `zzqx-e2e-nonsense-${Date.now()}` });
    movieIds.push(movie.id);

    const response = await api.reparseMovie(movie.id);

    expect(response.success).toBe(true);
    expect(response.data!.id).toBe(movie.id);
    expect(response.data!.parse_status).toBe('failed');
    expect(wire((await api.getMovie(movie.id)).data).parse_status).toBe('failed');
  });
});
