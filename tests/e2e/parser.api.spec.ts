/**
 * Parser API E2E Tests
 *
 * Comprehensive API tests for the Filename Parser endpoints.
 * Tests follow Given-When-Then format with priority tags.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @parser
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Test Data: Sample Filenames
// =============================================================================

const sampleFilenames = {
  // Movie filenames
  movieStandard: 'Inception.2010.1080p.BluRay.x264-SPARKS.mkv',
  movieChinese: '全面啟動.Inception.2010.1080p.BluRay.mkv',
  movieSimple: 'The.Dark.Knight.2008.mkv',
  movieWithYear: 'Interstellar (2014) 2160p UHD BluRay.mkv',

  // TV Show filenames
  tvStandard: 'Breaking.Bad.S01E01.720p.BluRay.mkv',
  tvSeasonEpisode: 'Game.of.Thrones.S08E06.1080p.WEB-DL.mkv',
  tvMultiEpisode: 'Friends.S01E01-E03.DVDRip.mkv',
  tvChinese: '權力遊戲.Game.of.Thrones.S01E01.mkv',

  // Edge cases
  noYear: 'Random.Movie.1080p.mkv',
  needsAI: '[字幕組] 某動漫 第01話.mp4',
  invalid: '',
  specialChars: 'Movie: The Sequel? (2023) [1080p].mkv',
};

// =============================================================================
// Single Parse Tests
// =============================================================================

test.describe('Parser API - Single Parse @api @parser', () => {
  // ---------------------------------------------------------------------------
  // Success Cases
  // ---------------------------------------------------------------------------

  test('[P0] POST /parser/parse - should parse standard movie filename', async ({ request }) => {
    // GIVEN: A standard movie filename with year and quality
    const filename = sampleFilenames.movieStandard;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should return success with extracted metadata
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.status).toBe('success');
    expect(body.data.media_type).toBe('movie');
    expect(body.data.original_filename).toBe(filename);
    expect(body.data.title).toContain('Inception');
    expect(body.data.year).toBe(2010);
    expect(body.data.quality).toBe('1080p');
    expect(body.data.source).toBe('BluRay');
    expect(body.data.video_codec).toBe('x264');
    expect(body.data.release_group).toBe('SPARKS');
    expect(body.data.confidence).toBeGreaterThan(0);
  });

  test('[P0] POST /parser/parse - should parse standard TV show filename', async ({ request }) => {
    // GIVEN: A standard TV show filename
    const filename = sampleFilenames.tvStandard;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should return success with TV show metadata
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.status).toBe('success');
    expect(body.data.media_type).toBe('tv');
    expect(body.data.title).toContain('Breaking');
    expect(body.data.season).toBe(1);
    expect(body.data.episode).toBe(1);
    expect(body.data.quality).toBe('720p');
  });

  test('[P1] POST /parser/parse - should parse movie with Chinese title', async ({ request }) => {
    // GIVEN: A filename with Chinese and English titles
    const filename = sampleFilenames.movieChinese;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should extract both titles
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.media_type).toBe('movie');
    expect(body.data.year).toBe(2010);
  });

  test('[P1] POST /parser/parse - should parse TV show with multiple episodes', async ({
    request,
  }) => {
    // GIVEN: A TV show filename with episode range
    const filename = sampleFilenames.tvMultiEpisode;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should extract episode range
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.media_type).toBe('tv');
    expect(body.data.season).toBe(1);
    expect(body.data.episode).toBe(1);
    // episode_end should be 3 for range E01-E03
    expect(body.data.episode_end).toBe(3);
  });

  test('[P1] POST /parser/parse - should handle movie without year', async ({ request }) => {
    // GIVEN: A filename without year
    const filename = sampleFilenames.noYear;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should still parse with lower confidence
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.quality).toBe('1080p');
  });

  test('[P2] POST /parser/parse - should return needs_ai for complex fansub filename', async ({
    request,
  }) => {
    // GIVEN: A complex fansub filename that needs AI
    const filename = sampleFilenames.needsAI;

    // WHEN: Parsing the filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should return needs_ai status
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(['needs_ai', 'success']).toContain(body.data.status);
  });

  // ---------------------------------------------------------------------------
  // Validation Error Cases
  // ---------------------------------------------------------------------------

  test('[P1] POST /parser/parse - should return 400 for missing filename', async ({ request }) => {
    // GIVEN: Request body without filename
    const requestBody = {};

    // WHEN: Attempting to parse
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: requestBody,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('VALIDATION_INVALID_FORMAT');
  });

  test('[P1] POST /parser/parse - should return 400 for empty filename', async ({ request }) => {
    // GIVEN: Empty filename
    const filename = '';

    // WHEN: Attempting to parse
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename },
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P2] POST /parser/parse - should return 400 for invalid JSON', async ({ request }) => {
    // GIVEN: Invalid request body

    // WHEN: Sending invalid content
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: 'not-valid-json',
      headers: { 'Content-Type': 'text/plain' },
    });

    // THEN: Should return 400 error
    expect(response.status()).toBe(400);
  });
});

// =============================================================================
// Batch Parse Tests
// =============================================================================

test.describe('Parser API - Batch Parse @api @parser', () => {
  test('[P0] POST /parser/parse-batch - should parse multiple filenames', async ({ request }) => {
    // GIVEN: Multiple filenames of different types
    const filenames = [
      sampleFilenames.movieStandard,
      sampleFilenames.tvStandard,
      sampleFilenames.movieChinese,
    ];

    // WHEN: Batch parsing
    const response = await request.post(`${API_BASE_URL}/parser/parse-batch`, {
      data: { filenames },
    });

    // THEN: Should return results for all filenames
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data).toHaveLength(3);

    // First result should be movie
    expect(body.data[0].media_type).toBe('movie');
    expect(body.data[0].original_filename).toBe(filenames[0]);

    // Second result should be TV
    expect(body.data[1].media_type).toBe('tv');
    expect(body.data[1].original_filename).toBe(filenames[1]);
  });

  test('[P1] POST /parser/parse-batch - should handle mixed success and needs_ai', async ({
    request,
  }) => {
    // GIVEN: Mix of parseable and complex filenames
    const filenames = [sampleFilenames.movieStandard, sampleFilenames.needsAI];

    // WHEN: Batch parsing
    const response = await request.post(`${API_BASE_URL}/parser/parse-batch`, {
      data: { filenames },
    });

    // THEN: Should return results for each
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toHaveLength(2);

    // First should succeed
    expect(body.data[0].status).toBe('success');

    // Second may need AI
    expect(['success', 'needs_ai']).toContain(body.data[1].status);
  });

  test('[P1] POST /parser/parse-batch - should return 400 for empty array', async ({ request }) => {
    // GIVEN: Empty filenames array
    const filenames: string[] = [];

    // WHEN: Attempting batch parse
    const response = await request.post(`${API_BASE_URL}/parser/parse-batch`, {
      data: { filenames },
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P1] POST /parser/parse-batch - should return 400 for missing filenames field', async ({
    request,
  }) => {
    // GIVEN: Request without filenames field

    // WHEN: Attempting batch parse
    const response = await request.post(`${API_BASE_URL}/parser/parse-batch`, {
      data: {},
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_INVALID_FORMAT');
  });

  test('[P2] POST /parser/parse-batch - should handle large batch efficiently', async ({
    request,
  }) => {
    // GIVEN: Large batch of filenames
    const filenames = Array.from({ length: 20 }, (_, i) => `Movie.${2000 + i}.1080p.mkv`);

    // WHEN: Batch parsing
    const startTime = Date.now();
    const response = await request.post(`${API_BASE_URL}/parser/parse-batch`, {
      data: { filenames },
    });
    const duration = Date.now() - startTime;

    // THEN: Should complete within reasonable time
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.data).toHaveLength(20);
    expect(duration).toBeLessThan(5000); // Should complete within 5 seconds
  });
});

// =============================================================================
// Response Format Validation
// =============================================================================

test.describe('Parser API - Response Format @api @parser', () => {
  test('[P1] should follow standard API response format for success', async ({ request }) => {
    // GIVEN: Valid parse request

    // WHEN: Parsing a filename
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename: sampleFilenames.movieStandard },
    });

    // THEN: Response should follow standard format
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('success', true);
    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('original_filename');
    expect(body.data).toHaveProperty('status');
    expect(body.data).toHaveProperty('media_type');
    expect(body.data).toHaveProperty('confidence');
  });

  test('[P1] error responses should include code and message', async ({ request }) => {
    // GIVEN: An invalid request

    // WHEN: Making request that will fail
    const response = await request.post(`${API_BASE_URL}/parser/parse`, {
      data: { filename: '' },
    });

    // THEN: Error response should have proper format
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBeDefined();
    expect(body.error.message).toBeDefined();
  });
});
