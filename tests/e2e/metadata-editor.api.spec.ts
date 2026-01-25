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
import * as fs from 'fs';
import * as path from 'path';

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

  test('[P1] PUT /media/{id}/metadata - should update metadata with all fields (AC2)', async ({ api }) => {
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

  test('[P1] PUT /media/{id}/metadata - should update only title and year (minimal required fields)', async ({ api }) => {
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

  test('[P1] PUT /media/{id}/metadata - should return 400 for missing title (AC4)', async ({ api }) => {
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

  test('[P1] PUT /media/{id}/metadata - should return 400 for missing year (AC4)', async ({ api }) => {
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

  test('[P1] PUT /media/{id}/metadata - should return 404 for non-existent movie', async ({ api }) => {
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

  test('[P2] PUT /media/{id}/metadata - should handle Chinese and English titles', async ({ api }) => {
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

  test('[P2] PUT /media/{id}/metadata - should set metadataSource to manual (AC2)', async ({ api }) => {
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
    // GIVEN: A test JPEG image (1x1 pixel red JPEG)
    // Minimal valid JPEG file
    const jpegBuffer = Buffer.from([
      0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01,
      0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43,
      0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
      0x09, 0x08, 0x0a, 0x0c, 0x14, 0x0d, 0x0c, 0x0b, 0x0b, 0x0c, 0x19, 0x12,
      0x13, 0x0f, 0x14, 0x1d, 0x1a, 0x1f, 0x1e, 0x1d, 0x1a, 0x1c, 0x1c, 0x20,
      0x24, 0x2e, 0x27, 0x20, 0x22, 0x2c, 0x23, 0x1c, 0x1c, 0x28, 0x37, 0x29,
      0x2c, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1f, 0x27, 0x39, 0x3d, 0x38, 0x32,
      0x3c, 0x2e, 0x33, 0x34, 0x32, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0x00, 0x01,
      0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xff, 0xc4, 0x00, 0x1f, 0x00, 0x00,
      0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
      0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
      0x09, 0x0a, 0x0b, 0xff, 0xc4, 0x00, 0xb5, 0x10, 0x00, 0x02, 0x01, 0x03,
      0x03, 0x02, 0x04, 0x03, 0x05, 0x05, 0x04, 0x04, 0x00, 0x00, 0x01, 0x7d,
      0x01, 0x02, 0x03, 0x00, 0x04, 0x11, 0x05, 0x12, 0x21, 0x31, 0x41, 0x06,
      0x13, 0x51, 0x61, 0x07, 0x22, 0x71, 0x14, 0x32, 0x81, 0x91, 0xa1, 0x08,
      0x23, 0x42, 0xb1, 0xc1, 0x15, 0x52, 0xd1, 0xf0, 0x24, 0x33, 0x62, 0x72,
      0x82, 0x09, 0x0a, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x25, 0x26, 0x27, 0x28,
      0x29, 0x2a, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x43, 0x44, 0x45,
      0x46, 0x47, 0x48, 0x49, 0x4a, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59,
      0x5a, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x73, 0x74, 0x75,
      0x76, 0x77, 0x78, 0x79, 0x7a, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89,
      0x8a, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0xa2, 0xa3,
      0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6,
      0xb7, 0xb8, 0xb9, 0xba, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9,
      0xca, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xe1, 0xe2,
      0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xf1, 0xf2, 0xf3, 0xf4,
      0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01,
      0x00, 0x00, 0x3f, 0x00, 0xfb, 0xd5, 0xdb, 0x20, 0xa8, 0xf1, 0x1c, 0xff,
      0xd9,
    ]);

    // WHEN: Uploading the poster
    const response = await api.uploadPoster(testMovieId, jpegBuffer, 'test-poster.jpg');

    // THEN: Should return success with poster URLs
    expect(response.success).toBe(true);
    expect(response.data).toBeDefined();
    expect(response.data!.posterUrl).toBeDefined();
    expect(response.data!.thumbnailUrl).toBeDefined();
  });

  test('[P1] POST /media/{id}/poster - should upload PNG poster (AC3)', async ({ api }) => {
    // GIVEN: A minimal valid PNG image (1x1 pixel)
    const pngBuffer = Buffer.from([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
      0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
      0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
      0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0x0f, 0x00, 0x00,
      0x01, 0x01, 0x00, 0x05, 0x18, 0xd8, 0x4d, 0x00, 0x00, 0x00, 0x00, 0x49,
      0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
    ]);

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

  test('[P1] POST /media/{id}/poster - should return 404 for non-existent movie', async ({ api }) => {
    // GIVEN: A non-existent movie ID and valid image
    const fakeId = 'nonexistent-movie-id-12345';
    const pngBuffer = Buffer.from([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
      0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
      0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
      0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0x0f, 0x00, 0x00,
      0x01, 0x01, 0x00, 0x05, 0x18, 0xd8, 0x4d, 0x00, 0x00, 0x00, 0x00, 0x49,
      0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
    ]);

    // WHEN: Attempting to upload to non-existent movie
    const response = await api.uploadPoster(fakeId, pngBuffer, 'test-poster.png');

    // THEN: Should return not found error
    expect(response.success).toBe(false);
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe('POSTER_UPLOAD_NOT_FOUND');
  });

  test('[P2] POST /media/{id}/poster - should return 400 for file too large', async ({ api }) => {
    // GIVEN: A file larger than 5MB (simulate with header check)
    // Note: In real test, we'd need a large file - this tests the error handling
    // Creating a 6MB buffer would be slow, so we rely on handler tests for full coverage
    // This test verifies the error code is correct when the service returns the error

    // Skip this test as it requires creating a large buffer
    // The unit tests in metadata_handler_test.go cover this case
    test.skip();
  });

  test('[P2] POST /media/{id}/poster - should process and optimize image (AC3)', async ({ api }) => {
    // GIVEN: A valid image
    const pngBuffer = Buffer.from([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
      0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
      0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
      0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0x0f, 0x00, 0x00,
      0x01, 0x01, 0x00, 0x05, 0x18, 0xd8, 0x4d, 0x00, 0x00, 0x00, 0x00, 0x49,
      0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
    ]);

    // WHEN: Uploading the poster
    const response = await api.uploadPoster(testMovieId, pngBuffer, 'test-poster.png');

    // THEN: Should return both full poster and thumbnail URLs
    expect(response.success).toBe(true);
    expect(response.data!.posterUrl).toContain('.webp');
    expect(response.data!.thumbnailUrl).toContain('-thumb.webp');
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
    const pngBuffer = Buffer.from([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
      0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
      0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
      0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0x0f, 0x00, 0x00,
      0x01, 0x01, 0x00, 0x05, 0x18, 0xd8, 0x4d, 0x00, 0x00, 0x00, 0x00, 0x49,
      0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
    ]);

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
