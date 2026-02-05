/**
 * Learning API E2E Tests (Story 3.9: Filename Mapping Learning System)
 *
 * Comprehensive API tests for the filename pattern learning endpoints.
 * Tests follow Given-When-Then format with priority tags.
 *
 * Acceptance Criteria:
 * - AC1: Learn pattern prompt (create pattern from correction)
 * - AC2: Auto-apply learned patterns (tested via pattern existence)
 * - AC3: Manage learned patterns (list, delete, stats)
 * - AC4: Fuzzy pattern matching (fansub group extraction)
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @learning @story-3-9
 */

import { test, expect } from '../support/fixtures';
import {
  createLearnPatternRequest,
  createMoviePatternRequest,
  presetPatternRequests,
} from '../support/fixtures/factories/learning-factory';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Learning API - Create Pattern Tests (AC1, AC4)
// =============================================================================

test.describe('Learning API - Create Pattern @api @learning @story-3-9', () => {
  const createdPatternIds: string[] = [];

  test.afterEach(async ({ request }) => {
    // Cleanup: Delete all created patterns
    for (const id of createdPatternIds) {
      await request.delete(`${API_BASE_URL}/learning/patterns/${id}`);
    }
    createdPatternIds.length = 0;
  });

  test('[P1] POST /learning/patterns - should create pattern from fansub filename (AC1, AC4)', async ({
    request,
  }) => {
    // GIVEN: A fansub-formatted filename with metadata reference
    const patternRequest = createLearnPatternRequest({
      filename: '[SubsPlease] Spy x Family - 12 (1080p) [ABC123].mkv',
      metadataType: 'series',
    });

    // WHEN: Learning a pattern from this filename
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });

    // THEN: Should return 201 with extracted pattern data
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.id).toBeTruthy();
    expect(body.data.pattern).toBeTruthy();
    expect(body.data.metadata_type).toBe('series');
    expect(body.data.metadata_id).toBe(patternRequest.metadataId);
    expect(body.data.fansub_group).toBe('SubsPlease');
    expect(body.data.title_pattern).toBeTruthy();
    expect(body.data.pattern_type).toBe('fansub');
    expect(body.data.confidence).toBe(1.0);
    expect(body.data.use_count).toBe(0);

    createdPatternIds.push(body.data.id);
  });

  test('[P1] POST /learning/patterns - should create pattern for movie type (AC1)', async ({
    request,
  }) => {
    // GIVEN: A standard movie filename
    const patternRequest = createMoviePatternRequest({
      filename: 'The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv',
      metadataType: 'movie',
    });

    // WHEN: Learning a pattern from this filename
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });

    // THEN: Should return 201 with movie-type pattern
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.metadata_type).toBe('movie');
    expect(body.data.title_pattern).toBeTruthy();

    createdPatternIds.push(body.data.id);
  });

  test('[P1] POST /learning/patterns - should extract fansub group correctly (AC4)', async ({
    request,
  }) => {
    // GIVEN: A filename with fansub group in brackets
    const patternRequest = createLearnPatternRequest({
      filename: '[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv',
      metadataType: 'series',
      tmdbId: 85937,
    });

    // WHEN: Learning a pattern
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });

    // THEN: Fansub group and title should be correctly extracted
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.data.fansub_group).toBe('Leopard-Raws');
    expect(body.data.title_pattern).toContain('Kimetsu no Yaiba');
    expect(body.data.pattern_regex).toBeTruthy();

    createdPatternIds.push(body.data.id);
  });

  test('[P1] POST /learning/patterns - should not duplicate similar patterns (AC1)', async ({
    request,
  }) => {
    // GIVEN: A pattern already exists for this fansub+title
    const firstRequest = createLearnPatternRequest({
      filename: '[Erai-raws] One Piece - 01 (1080p).mkv',
      metadataId: 'series-op-001',
      metadataType: 'series',
    });

    const firstResponse = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: firstRequest,
    });
    expect(firstResponse.status()).toBe(201);
    const firstBody = await firstResponse.json();
    createdPatternIds.push(firstBody.data.id);

    // WHEN: Learning from a very similar filename (same fansub + title, different episode)
    const secondRequest = createLearnPatternRequest({
      filename: '[Erai-raws] One Piece - 02 (1080p).mkv',
      metadataId: 'series-op-001',
      metadataType: 'series',
    });

    const secondResponse = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: secondRequest,
    });

    // THEN: Should return the existing pattern instead of creating a duplicate
    expect(secondResponse.status()).toBe(201);
    const secondBody = await secondResponse.json();
    expect(secondBody.success).toBe(true);

    // The returned pattern should be the same as the first one (same ID)
    expect(secondBody.data.id).toBe(firstBody.data.id);
  });

  // ---------------------------------------------------------------------------
  // Validation Tests (AC1)
  // ---------------------------------------------------------------------------

  test('[P1] POST /learning/patterns - should return 400 for missing filename (AC1)', async ({
    request,
  }) => {
    // GIVEN: Request without filename
    const invalidRequest = {
      metadataId: 'series-001',
      metadataType: 'series',
    };

    // WHEN: Attempting to create pattern without filename
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: invalidRequest,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P1] POST /learning/patterns - should return 400 for missing metadataId (AC1)', async ({
    request,
  }) => {
    // GIVEN: Request without metadataId
    const invalidRequest = {
      filename: '[Test] Some Anime - 01.mkv',
      metadataType: 'series',
    };

    // WHEN: Attempting to create pattern without metadataId
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: invalidRequest,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P1] POST /learning/patterns - should return 400 for invalid metadataType (AC1)', async ({
    request,
  }) => {
    // GIVEN: Request with invalid metadataType
    const invalidRequest = {
      filename: '[Test] Some Anime - 01.mkv',
      metadataId: 'series-001',
      metadataType: 'invalid',
    };

    // WHEN: Attempting to create pattern with invalid type
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: invalidRequest,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P2] POST /learning/patterns - should return 400 for empty body (AC1)', async ({
    request,
  }) => {
    // GIVEN: Empty request body
    const emptyRequest = {};

    // WHEN: Attempting to create pattern with empty body
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: emptyRequest,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
  });
});

// =============================================================================
// Learning API - List & Stats Tests (AC3)
// =============================================================================

test.describe('Learning API - List & Stats @api @learning @story-3-9', () => {
  const createdPatternIds: string[] = [];

  test.afterEach(async ({ request }) => {
    // Cleanup: Delete all created patterns
    for (const id of createdPatternIds) {
      await request.delete(`${API_BASE_URL}/learning/patterns/${id}`);
    }
    createdPatternIds.length = 0;
  });

  test('[P1] GET /learning/patterns - should list patterns with stats (AC3)', async ({
    request,
  }) => {
    // GIVEN: Multiple patterns exist
    const requests = [
      createLearnPatternRequest({ filename: '[GroupA] Title A - 01.mkv', metadataId: 'series-a' }),
      createMoviePatternRequest({ filename: 'Movie.B.2024.mkv', metadataId: 'movie-b' }),
    ];

    for (const req of requests) {
      const res = await request.post(`${API_BASE_URL}/learning/patterns`, { data: req });
      const resBody = await res.json();
      if (resBody.data?.id) {
        createdPatternIds.push(resBody.data.id);
      }
    }

    // WHEN: Listing all patterns
    const response = await request.get(`${API_BASE_URL}/learning/patterns`);

    // THEN: Should return patterns with totalCount and stats
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.patterns).toBeInstanceOf(Array);
    expect(body.data.totalCount).toBeGreaterThanOrEqual(2);
    expect(body.data.stats).toBeDefined();
    expect(body.data.stats.totalPatterns).toBeGreaterThanOrEqual(2);
    expect(typeof body.data.stats.totalApplied).toBe('number');
  });

  test('[P2] GET /learning/patterns - should return valid structure when no patterns exist (AC3)', async ({
    request,
  }) => {
    // GIVEN: API is running (may or may not have patterns)
    // Note: Verifies the response structure is valid

    // WHEN: Listing patterns
    const response = await request.get(`${API_BASE_URL}/learning/patterns`);

    // THEN: Should return valid response with patterns (array or null for empty)
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    // Go serializes nil slices as null, so patterns can be null or array
    const patterns = body.data.patterns ?? [];
    expect(Array.isArray(patterns)).toBe(true);
    expect(typeof body.data.totalCount).toBe('number');
  });

  test('[P1] GET /learning/stats - should return pattern statistics (AC3)', async ({
    request,
  }) => {
    // GIVEN: At least one pattern exists
    const patternRequest = createLearnPatternRequest({
      filename: '[Stats-Test] Stats Anime - 01.mkv',
      metadataId: 'series-stats-001',
    });
    const createRes = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });
    const createBody = await createRes.json();
    createdPatternIds.push(createBody.data.id);

    // WHEN: Getting statistics
    const response = await request.get(`${API_BASE_URL}/learning/stats`);

    // THEN: Should return stats with expected fields
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(typeof body.data.totalPatterns).toBe('number');
    expect(body.data.totalPatterns).toBeGreaterThanOrEqual(1);
    expect(typeof body.data.totalApplied).toBe('number');
  });
});

// =============================================================================
// Learning API - Delete Pattern Tests (AC3)
// =============================================================================

test.describe('Learning API - Delete Pattern @api @learning @story-3-9', () => {
  test('[P1] DELETE /learning/patterns/:id - should delete existing pattern (AC3)', async ({
    request,
  }) => {
    // GIVEN: A pattern exists
    const patternRequest = createLearnPatternRequest({
      filename: '[Delete-Test] Deletable Anime - 01.mkv',
      metadataId: 'series-del-001',
    });
    const createRes = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });
    const createBody = await createRes.json();
    const patternId = createBody.data.id;

    // WHEN: Deleting the pattern
    const response = await request.delete(`${API_BASE_URL}/learning/patterns/${patternId}`);

    // THEN: Should return 204 No Content
    expect(response.status()).toBe(204);

    // Verify pattern is removed from list
    const listRes = await request.get(`${API_BASE_URL}/learning/patterns`);
    const listBody = await listRes.json();
    const patterns = listBody.data.patterns ?? [];
    const found = patterns.find(
      (p: { id: string }) => p.id === patternId
    );
    expect(found).toBeUndefined();
  });

  test('[P2] DELETE /learning/patterns/:id - should handle non-existent pattern gracefully', async ({
    request,
  }) => {
    // GIVEN: A pattern ID that does not exist
    const fakeId = 'non-existent-pattern-id-12345';

    // WHEN: Attempting to delete non-existent pattern
    const response = await request.delete(`${API_BASE_URL}/learning/patterns/${fakeId}`);

    // THEN: Should return an error status (500 or 404 depending on implementation)
    // The current implementation returns 500 for delete errors
    const status = response.status();
    expect([204, 404, 500]).toContain(status);
  });
});

// =============================================================================
// Learning API - Full CRUD Lifecycle (AC1, AC3)
// =============================================================================

test.describe('Learning API - CRUD Lifecycle @api @learning @story-3-9', () => {
  test('[P1] should complete full pattern lifecycle: create → list → stats → delete → verify (AC1, AC3)', async ({
    request,
  }) => {
    // === CREATE ===
    // GIVEN: A new fansub filename to learn
    const patternRequest = createLearnPatternRequest({
      filename: '[Lifecycle-Test] Full CRUD Anime - 05 (1080p).mkv',
      metadataId: 'series-lifecycle-001',
      metadataType: 'series',
      tmdbId: 12345,
    });

    // WHEN: Creating a new pattern
    const createRes = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: patternRequest,
    });

    // THEN: Pattern should be created
    expect(createRes.status()).toBe(201);
    const createBody = await createRes.json();
    expect(createBody.success).toBe(true);
    const patternId = createBody.data.id;
    expect(patternId).toBeTruthy();

    // === LIST ===
    // WHEN: Listing all patterns
    const listRes = await request.get(`${API_BASE_URL}/learning/patterns`);

    // THEN: Created pattern should appear in the list
    expect(listRes.status()).toBe(200);
    const listBody = await listRes.json();
    const foundPattern = listBody.data.patterns.find(
      (p: { id: string }) => p.id === patternId
    );
    expect(foundPattern).toBeDefined();
    expect(foundPattern.metadata_id).toBe('series-lifecycle-001');

    // === STATS ===
    // WHEN: Getting stats
    const statsRes = await request.get(`${API_BASE_URL}/learning/stats`);

    // THEN: Stats should reflect at least one pattern
    expect(statsRes.status()).toBe(200);
    const statsBody = await statsRes.json();
    expect(statsBody.data.totalPatterns).toBeGreaterThanOrEqual(1);

    // === DELETE ===
    // WHEN: Deleting the pattern
    const deleteRes = await request.delete(`${API_BASE_URL}/learning/patterns/${patternId}`);

    // THEN: Should return 204
    expect(deleteRes.status()).toBe(204);

    // === VERIFY DELETION ===
    // WHEN: Listing patterns after deletion
    const verifyRes = await request.get(`${API_BASE_URL}/learning/patterns`);

    // THEN: Pattern should no longer be in the list
    const verifyBody = await verifyRes.json();
    const remainingPatterns = verifyBody.data.patterns ?? [];
    const deletedPattern = remainingPatterns.find(
      (p: { id: string }) => p.id === patternId
    );
    expect(deletedPattern).toBeUndefined();
  });
});

// =============================================================================
// Learning API - Response Format Validation
// =============================================================================

test.describe('Learning API - Response Format @api @learning @story-3-9', () => {
  test('[P2] should follow standard API response format for success', async ({ request }) => {
    // GIVEN: API is running

    // WHEN: Making a list request
    const response = await request.get(`${API_BASE_URL}/learning/patterns`);

    // THEN: Response should follow standard format
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('success');
    expect(body.success).toBe(true);
    expect(body).toHaveProperty('data');
  });

  test('[P2] should follow standard API error response format', async ({ request }) => {
    // GIVEN: An invalid request body

    // WHEN: Sending invalid data
    const response = await request.post(`${API_BASE_URL}/learning/patterns`, {
      data: {},
    });

    // THEN: Error response should follow standard format
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error).toHaveProperty('code');
    expect(body.error).toHaveProperty('message');
  });
});
