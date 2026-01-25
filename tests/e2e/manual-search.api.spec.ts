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

// =============================================================================
// Manual Search API Tests (AC1, AC4)
// =============================================================================

test.describe('Manual Search API @api @metadata', () => {
  test('[P1] POST /metadata/manual-search - should search all sources', async ({ api }) => {
    // GIVEN: A valid search request for all sources
    const searchRequest = {
      query: 'Inception',
      mediaType: 'movie' as const,
      source: 'all' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return success with results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.results).toBeDefined();
    expect(response.data!.searchedSources).toContain('tmdb');
  });

  test('[P1] POST /metadata/manual-search - should search specific source (TMDb)', async ({ api }) => {
    // GIVEN: A search request for TMDb only
    const searchRequest = {
      query: 'Fight Club',
      mediaType: 'movie' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return results from TMDb only
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.searchedSources).toEqual(['tmdb']);

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
      mediaType: 'tv' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return TV show results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();

    if (response.data!.results.length > 0) {
      const firstResult = response.data!.results[0];
      expect(firstResult.mediaType).toBe('tv');
    }
  });

  test('[P1] POST /metadata/manual-search - should filter by year', async ({ api }) => {
    // GIVEN: A search request with year filter
    const searchRequest = {
      query: 'Matrix',
      mediaType: 'movie' as const,
      year: 1999,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
  });

  test('[P2] POST /metadata/manual-search - should return empty results for non-existent query', async ({ api }) => {
    // GIVEN: A search request for non-existent content
    const searchRequest = {
      query: 'xyznonexistentmovie99999abcdef',
      mediaType: 'movie' as const,
      source: 'tmdb' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return success with empty results
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.results).toEqual([]);
    expect(response.data!.totalCount).toBe(0);
  });

  test('[P1] POST /metadata/manual-search - should return error for missing query', async ({ api }) => {
    // GIVEN: A search request without query
    const searchRequest = {
      query: '',
      mediaType: 'movie' as const,
      source: 'all' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should return error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('MANUAL_SEARCH_INVALID_REQUEST');
  });

  test('[P2] POST /metadata/manual-search - should default to movie media type', async ({ api }) => {
    // GIVEN: A search request without mediaType
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
      mediaType: 'movie' as const,
    };

    // WHEN: Calling the manual search API
    const response = await api.manualSearch(searchRequest);

    // THEN: Should search all sources
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    // Should have searched at least TMDb
    expect(response.data!.searchedSources.length).toBeGreaterThanOrEqual(1);
  });

  test('[P1] POST /metadata/manual-search - should include source indicator in results (AC4)', async ({ api }) => {
    // GIVEN: A search request
    const searchRequest = {
      query: 'Inception',
      mediaType: 'movie' as const,
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

  test('[P1] POST /metadata/manual-search - results should include required fields (AC2)', async ({ api }) => {
    // GIVEN: A search request that should return results
    const searchRequest = {
      query: 'Inception',
      mediaType: 'movie' as const,
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
  test.skip('[P1] POST /metadata/apply - should apply metadata to movie (AC3)', async ({ api }) => {
    // GIVEN: A movie exists and we have selected metadata
    // Note: This test requires a movie to exist in the database
    // Skip until we have proper test data seeding

    const applyRequest = {
      mediaId: 'test-movie-id',
      mediaType: 'movie' as const,
      selectedItem: {
        id: 'tmdb-550',
        source: 'tmdb',
      },
    };

    // WHEN: Applying the metadata
    const response = await api.applyMetadata(applyRequest);

    // THEN: Should return success
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.mediaId).toBe('test-movie-id');
    expect(response.data!.source).toBe('tmdb');
  });

  test.skip('[P1] POST /metadata/apply - should apply metadata to series', async ({ api }) => {
    // GIVEN: A series exists and we have selected metadata
    // Note: This test requires a series to exist in the database
    // Skip until we have proper test data seeding

    const applyRequest = {
      mediaId: 'test-series-id',
      mediaType: 'series' as const,
      selectedItem: {
        id: 'tmdb-1396',
        source: 'tmdb',
      },
    };

    // WHEN: Applying the metadata
    const response = await api.applyMetadata(applyRequest);

    // THEN: Should return success
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.mediaType).toBe('series');
  });

  test('[P1] POST /metadata/apply - should return error for missing mediaId', async ({ api }) => {
    // GIVEN: A request without mediaId
    const applyRequest = {
      mediaId: '',
      mediaType: 'movie' as const,
      selectedItem: {
        id: 'tmdb-550',
        source: 'tmdb',
      },
    };

    // WHEN: Applying the metadata
    const response = await api.applyMetadata(applyRequest);

    // THEN: Should return error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('APPLY_METADATA_INVALID_REQUEST');
  });

  test('[P1] POST /metadata/apply - should return error for non-existent media', async ({ api }) => {
    // GIVEN: A request for non-existent media
    const applyRequest = {
      mediaId: 'nonexistent-media-id-12345',
      mediaType: 'movie' as const,
      selectedItem: {
        id: 'tmdb-550',
        source: 'tmdb',
      },
    };

    // WHEN: Applying the metadata
    const response = await api.applyMetadata(applyRequest);

    // THEN: Should return not found error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('APPLY_METADATA_NOT_FOUND');
  });

  test.skip('[P2] POST /metadata/apply - should accept learnPattern flag for Story 3.9', async ({ api }) => {
    // GIVEN: A valid apply request with learnPattern flag
    // Note: This test requires a movie to exist in the database
    // Skip until we have proper test data seeding

    const applyRequest = {
      mediaId: 'test-movie-id',
      mediaType: 'movie' as const,
      selectedItem: {
        id: 'tmdb-550',
        source: 'tmdb',
      },
      learnPattern: true,
    };

    // WHEN: Applying the metadata with learn pattern
    const response = await api.applyMetadata(applyRequest);

    // THEN: Should succeed (learning is triggered in background)
    expect(response.success).toBe(true);
  });
});
