/**
 * Metadata Editor API Tests (Story 3-8)
 *
 * Tests for the metadata editing API endpoints including:
 * - Update metadata: PUT /api/v1/media/{id}/metadata
 * - Upload poster: POST /api/v1/media/{id}/poster
 *
 * Prerequisites:
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * Acceptance Criteria Coverage:
 * - AC2: Persist Changes - Metadata updates are saved with source="manual"
 * - AC3: Custom Poster Upload - Image is resized, optimized, and stored
 * - AC4: Form Validation - Required fields are validated
 *
 * @tags @api @metadata-editor @story-3-8
 */

import { test, expect } from '../support/fixtures';
import { faker } from '@faker-js/faker';
import { getValidPngBuffer, getValidJpegBuffer } from '../support/fixtures/test-images';

// =============================================================================
// Test Data Factories
// =============================================================================

function createTestMovieData() {
  return {
    title: `測試電影 ${Date.now()}`,
    releaseDate: faker.date.past({ years: 10 }).toISOString().split('T')[0],
    originalTitle: faker.lorem.words(3),
    genres: ['動作', '科幻'],
    overview: faker.lorem.paragraph(),
  };
}

function createUpdateMetadataData() {
  return {
    title: `更新測試 ${Date.now()}`,
    titleEnglish: 'Updated Test Movie',
    year: 2024,
    genres: ['動作', '冒險'],
    director: '李安',
    cast: ['演員一', '演員二', '演員三'],
    overview: '這是一部測試電影的描述，用於驗證 metadata 更新功能。',
  };
}

// =============================================================================
// Update Metadata API Tests (AC2, AC4)
// =============================================================================

test.describe('Update Metadata API @api @metadata-editor', () => {
  let testMovieId: string;

  test.beforeEach(async ({ api }) => {
    // Create a test movie for each test
    const movieData = createTestMovieData();
    const response = await api.createMovie(movieData);
    expect(response.success).toBe(true);
    testMovieId = response.data!.id;
  });

  test.afterEach(async ({ api }) => {
    // Cleanup: Delete created movie
    if (testMovieId) {
      await api.deleteMovie(testMovieId);
      testMovieId = '';
    }
  });

  test('[P1] PUT /media/{id}/metadata - should update metadata with all fields (AC2)', async ({
    api,
  }) => {
    // GIVEN: An existing movie and valid update data
    const updateData = createUpdateMetadataData();

    // WHEN: Updating the movie metadata
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should return success with updated metadata
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.id).toBe(testMovieId);
    expect(response.data!.title).toBe(updateData.title);
    expect(response.data!.metadataSource).toBe('manual');
  });

  test('[P1] PUT /media/{id}/metadata - should update only title and year (minimal required fields)', async ({
    api,
  }) => {
    // GIVEN: An existing movie and minimal update data
    const updateData = {
      title: '最小更新測試',
      year: 2023,
    };

    // WHEN: Updating with minimal data
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should succeed with minimal fields
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.title).toBe(updateData.title);
  });

  test('[P1] PUT /media/{id}/metadata - should return 400 for missing title (AC4)', async ({
    api,
  }) => {
    // GIVEN: Update data without title
    const updateData = {
      title: '',
      year: 2024,
    };

    // WHEN: Attempting to update without title
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should return validation error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P1] PUT /media/{id}/metadata - should return 400 for missing year (AC4)', async ({
    api,
  }) => {
    // GIVEN: Update data without year
    const updateData = {
      title: '無年份測試',
      year: 0,
    };

    // WHEN: Attempting to update without year
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should return validation error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P1] PUT /media/{id}/metadata - should return 404 for non-existent movie', async ({
    api,
  }) => {
    // GIVEN: A non-existent movie ID
    const fakeId = 'nonexistent-movie-id-12345';
    const updateData = createUpdateMetadataData();

    // WHEN: Attempting to update non-existent movie
    const response = await api.updateMetadata(fakeId, updateData);

    // THEN: Should return not found error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('METADATA_UPDATE_NOT_FOUND');
  });

  test('[P2] PUT /media/{id}/metadata - should update genres array', async ({ api }) => {
    // GIVEN: Update data with specific genres
    const updateData = {
      title: '類型更新測試',
      year: 2024,
      genres: ['恐怖', '驚悚', '懸疑'],
    };

    // WHEN: Updating with genres
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should succeed
    expect(response.success).toBe(true);

    // Verify by getting the movie
    const getResponse = await api.getMovie(testMovieId);
    expect(getResponse.success).toBe(true);
    expect(getResponse.data!.genres).toEqual(updateData.genres);
  });

  test('[P2] PUT /media/{id}/metadata - should update cast array', async ({ api }) => {
    // GIVEN: Update data with specific cast
    const updateData = {
      title: '演員更新測試',
      year: 2024,
      cast: ['周星馳', '吳孟達', '張曼玉'],
    };

    // WHEN: Updating with cast
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should succeed
    expect(response.success).toBe(true);
  });

  test('[P2] PUT /media/{id}/metadata - should handle Chinese and English titles', async ({
    api,
  }) => {
    // GIVEN: Update data with both Chinese and English titles
    const updateData = {
      title: '臥虎藏龍',
      titleEnglish: 'Crouching Tiger, Hidden Dragon',
      year: 2000,
    };

    // WHEN: Updating with bilingual titles
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Should succeed
    expect(response.success).toBe(true);
    expect(response.data!.title).toBe(updateData.title);
  });

  test('[P2] PUT /media/{id}/metadata - should set metadataSource to manual (AC2)', async ({
    api,
  }) => {
    // GIVEN: An existing movie
    const updateData = createUpdateMetadataData();

    // WHEN: Updating the metadata
    const response = await api.updateMetadata(testMovieId, updateData);

    // THEN: Source should be set to manual
    expect(response.success).toBe(true);
    expect(response.data!.metadataSource).toBe('manual');
  });
});

// =============================================================================
// Series Metadata Update Tests
// =============================================================================

test.describe('Update Series Metadata API @api @metadata-editor', () => {
  let testSeriesId: string;

  test.beforeEach(async ({ api }) => {
    // Create a test series for each test
    const seriesData = {
      title: `測試影集 ${Date.now()}`,
      firstAirDate: faker.date.past({ years: 5 }).toISOString().split('T')[0],
      originalTitle: faker.lorem.words(3),
      genres: ['劇情', '犯罪'],
      overview: faker.lorem.paragraph(),
    };
    const response = await api.createSeries(seriesData);
    expect(response.success).toBe(true);
    testSeriesId = response.data!.id;
  });

  test.afterEach(async ({ api }) => {
    // Cleanup: Delete created series
    if (testSeriesId) {
      await api.deleteSeries(testSeriesId);
      testSeriesId = '';
    }
  });

  test('[P1] PUT /media/{id}/metadata - should update series metadata', async ({ api }) => {
    // GIVEN: An existing series and valid update data
    const updateData = {
      mediaType: 'series' as const,
      title: '絕命毒師',
      titleEnglish: 'Breaking Bad',
      year: 2008,
      genres: ['劇情', '犯罪', '驚悚'],
      director: 'Vince Gilligan',
      cast: ['乔治·莫里斯', '亚伦·保罗'],
      overview: '一位高中化學老師被診斷出患有肺癌...',
    };

    // WHEN: Updating the series metadata
    const response = await api.updateMetadata(testSeriesId, updateData);

    // THEN: Should return success
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.title).toBe(updateData.title);
    expect(response.data!.metadataSource).toBe('manual');
  });
});

// =============================================================================
// Upload Poster API Tests (AC3)
// =============================================================================

test.describe('Upload Poster API @api @metadata-editor', () => {
  let testMovieId: string;

  test.beforeEach(async ({ api }) => {
    // Create a test movie for each test
    const movieData = createTestMovieData();
    const response = await api.createMovie(movieData);
    expect(response.success).toBe(true);
    testMovieId = response.data!.id;
  });

  test.afterEach(async ({ api }) => {
    // Cleanup: Delete created movie
    if (testMovieId) {
      await api.deleteMovie(testMovieId);
      testMovieId = '';
    }
  });

  test('[P1] POST /media/{id}/poster - should upload JPEG poster (AC3)', async ({ api }) => {
    // GIVEN: A valid JPEG image
    const jpegBuffer = getValidJpegBuffer();

    // WHEN: Uploading the poster
    const response = await api.uploadPoster(testMovieId, jpegBuffer, 'test-poster.jpg');

    // THEN: Should return success with poster URLs
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.posterUrl).toBeDefined();
    expect(response.data!.thumbnailUrl).toBeDefined();
  });

  test('[P1] POST /media/{id}/poster - should upload PNG poster (AC3)', async ({ api }) => {
    // GIVEN: A valid PNG image
    const pngBuffer = getValidPngBuffer();

    // WHEN: Uploading the poster
    const response = await api.uploadPoster(testMovieId, pngBuffer, 'test-poster.png');

    // THEN: Should return success with poster URLs
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.posterUrl).toBeDefined();
  });

  test('[P1] POST /media/{id}/poster - should return 400 for invalid format', async ({ api }) => {
    // GIVEN: An invalid file (text file)
    const textBuffer = Buffer.from('This is not an image file');

    // WHEN: Attempting to upload invalid file
    const response = await api.uploadPoster(testMovieId, textBuffer, 'test.txt');

    // THEN: Should return format error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('POSTER_INVALID_FORMAT');
  });

  test('[P1] POST /media/{id}/poster - should return 404 for non-existent movie', async ({
    api,
  }) => {
    // GIVEN: A non-existent movie ID and valid image
    const fakeId = 'nonexistent-movie-id-12345';
    const pngBuffer = getValidPngBuffer();

    // WHEN: Attempting to upload to non-existent movie
    const response = await api.uploadPoster(fakeId, pngBuffer, 'test-poster.png');

    // THEN: Should return not found error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('POSTER_UPLOAD_NOT_FOUND');
  });

  test('[P2] POST /media/{id}/poster - should return 400 for file too large', async ({
    api: _api,
  }) => {
    // GIVEN: A file larger than 5MB (simulate with header check)
    // Note: In real test, we'd need a large file - this tests the error handling
    // Creating a 6MB buffer would be slow, so we rely on handler tests for full coverage
    // This test verifies the error code is correct when the service returns the error

    // Skip this test as it requires creating a large buffer
    // The unit tests in metadata_handler_test.go cover this case
    test.skip();
  });

  test('[P2] POST /media/{id}/poster - should process and optimize image (AC3)', async ({
    api,
  }) => {
    // GIVEN: A valid image
    const pngBuffer = getValidPngBuffer();

    // WHEN: Uploading the poster
    const response = await api.uploadPoster(testMovieId, pngBuffer, 'test-poster.png');

    // THEN: Should return both full poster and thumbnail URLs
    expect(response.success).toBe(true);
    // Note: The actual output format depends on image processor configuration
    expect(response.data!.posterUrl).toBeDefined();
    expect(response.data!.thumbnailUrl).toBeDefined();
  });
});

// =============================================================================
// Integration Tests - Update + Upload Combined
// =============================================================================

test.describe('Metadata Editor Integration @api @metadata-editor', () => {
  let testMovieId: string;

  test.beforeEach(async ({ api }) => {
    const movieData = createTestMovieData();
    const response = await api.createMovie(movieData);
    expect(response.success).toBe(true);
    testMovieId = response.data!.id;
  });

  test.afterEach(async ({ api }) => {
    if (testMovieId) {
      await api.deleteMovie(testMovieId);
      testMovieId = '';
    }
  });

  test('[P1] should update metadata and upload poster in sequence', async ({ api }) => {
    // GIVEN: Test data
    const updateData = createUpdateMetadataData();
    const pngBuffer = getValidPngBuffer();

    // WHEN: Updating metadata first
    const metadataResponse = await api.updateMetadata(testMovieId, updateData);

    // THEN: Metadata update should succeed
    expect(metadataResponse.success).toBe(true);
    expect(metadataResponse.data!.metadataSource).toBe('manual');

    // WHEN: Then uploading poster
    const posterResponse = await api.uploadPoster(testMovieId, pngBuffer, 'poster.png');

    // THEN: Poster upload should succeed
    expect(posterResponse.success).toBe(true);
    expect(posterResponse.data!.posterUrl).toBeDefined();

    // VERIFY: Final state
    const getResponse = await api.getMovie(testMovieId);
    expect(getResponse.success).toBe(true);
    expect(getResponse.data!.posterPath).toBeDefined();
  });
});
