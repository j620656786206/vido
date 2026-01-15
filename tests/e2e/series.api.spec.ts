/**
 * Series API E2E Tests
 *
 * Comprehensive API tests for the TV Series CRUD endpoints.
 * Tests follow Given-When-Then format with priority tags.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @series
 */

import { test, expect } from '@playwright/test';
import { faker } from '@faker-js/faker';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Test Data Factories
// =============================================================================

interface SeriesInput {
  title: string;
  firstAirDate: string;
  originalTitle?: string;
  genres?: string[];
  overview?: string;
  posterPath?: string;
  numberOfSeasons?: number;
  numberOfEpisodes?: number;
  tmdbId?: number;
  imdbId?: string;
}

function createSeriesInput(overrides: Partial<SeriesInput> = {}): SeriesInput {
  return {
    title: faker.lorem.words(3),
    firstAirDate: faker.date.past({ years: 5 }).toISOString().split('T')[0],
    originalTitle: faker.lorem.words(3),
    genres: [faker.helpers.arrayElement(['Drama', 'Comedy', 'Thriller', 'Sci-Fi'])],
    overview: faker.lorem.paragraph(),
    posterPath: `/posters/${faker.string.alphanumeric(10)}.jpg`,
    numberOfSeasons: faker.number.int({ min: 1, max: 10 }),
    numberOfEpisodes: faker.number.int({ min: 6, max: 100 }),
    tmdbId: faker.number.int({ min: 100000, max: 999999 }),
    ...overrides,
  };
}

// =============================================================================
// Series API Tests
// =============================================================================

test.describe('Series API - CRUD Operations @api @series', () => {
  let createdSeriesId: string;

  test.afterEach(async ({ request }) => {
    // Cleanup: Delete created series if exists
    if (createdSeriesId) {
      await request.delete(`${API_BASE_URL}/series/${createdSeriesId}`);
      createdSeriesId = '';
    }
  });

  // ---------------------------------------------------------------------------
  // CREATE Tests
  // ---------------------------------------------------------------------------

  test('[P1] POST /series - should create series with required fields', async ({ request }) => {
    // GIVEN: Valid series data with required fields only
    const seriesData = {
      title: '測試劇集 ' + Date.now(),
      firstAirDate: '2024-01-15',
    };

    // WHEN: Creating a series via POST
    const response = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });

    // THEN: Should return 201 with created series
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.id).toBeDefined();
    expect(body.data.title).toBe(seriesData.title);
    expect(body.data.firstAirDate).toBe(seriesData.firstAirDate);

    // Store for cleanup
    createdSeriesId = body.data.id;
  });

  test('[P1] POST /series - should create series with all optional fields', async ({ request }) => {
    // GIVEN: Complete series data with all fields
    const seriesData = createSeriesInput({
      title: '完整測試劇集 ' + Date.now(),
    });

    // WHEN: Creating a series via POST
    const response = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });

    // THEN: Should return 201 with all fields populated
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.title).toBe(seriesData.title);
    expect(body.data.genres).toEqual(seriesData.genres);

    createdSeriesId = body.data.id;
  });

  test('[P1] POST /series - should return 400 for missing required fields', async ({ request }) => {
    // GIVEN: Series data missing required 'title' field
    const invalidData = {
      firstAirDate: '2024-01-15',
      // title is missing
    };

    // WHEN: Attempting to create series
    const response = await request.post(`${API_BASE_URL}/series`, {
      data: invalidData,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  // ---------------------------------------------------------------------------
  // READ Tests
  // ---------------------------------------------------------------------------

  test('[P1] GET /series - should list series with pagination', async ({ request }) => {
    // GIVEN: Series exist in database

    // First, create a series to ensure data exists
    const seriesData = createSeriesInput({ title: '列表測試劇集 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    createdSeriesId = created.data.id;

    // WHEN: Listing series with pagination
    const response = await request.get(`${API_BASE_URL}/series?page=1&page_size=10`);

    // THEN: Should return paginated list
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.items).toBeInstanceOf(Array);
    expect(body.data.page).toBe(1);
    expect(body.data.pageSize).toBe(10);
    expect(body.data).toHaveProperty('totalItems');
    expect(body.data).toHaveProperty('totalPages');
  });

  test('[P1] GET /series/:id - should return series by ID', async ({ request }) => {
    // GIVEN: A series exists
    const seriesData = createSeriesInput({ title: 'ID查詢測試劇集 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    createdSeriesId = created.data.id;

    // WHEN: Getting series by ID
    const response = await request.get(`${API_BASE_URL}/series/${createdSeriesId}`);

    // THEN: Should return the series
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.id).toBe(createdSeriesId);
    expect(body.data.title).toBe(seriesData.title);
  });

  test('[P1] GET /series/:id - should return 404 for non-existent series', async ({ request }) => {
    // GIVEN: A non-existent series ID
    const fakeId = 'non-existent-series-id-12345';

    // WHEN: Getting series by fake ID
    const response = await request.get(`${API_BASE_URL}/series/${fakeId}`);

    // THEN: Should return 404 not found
    expect(response.status()).toBe(404);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('DB_NOT_FOUND');
  });

  // ---------------------------------------------------------------------------
  // SEARCH Tests
  // ---------------------------------------------------------------------------

  test('[P1] GET /series/search - should search series by title', async ({ request }) => {
    // GIVEN: A series with known title exists
    const uniqueTitle = `搜尋測試劇集_${Date.now()}`;
    const seriesData = createSeriesInput({ title: uniqueTitle });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    createdSeriesId = created.data.id;

    // WHEN: Searching for series by title
    const response = await request.get(`${API_BASE_URL}/series/search?q=${encodeURIComponent(uniqueTitle)}`);

    // THEN: Should return matching series
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.items).toBeInstanceOf(Array);
    expect(body.data.items.length).toBeGreaterThan(0);
    expect(body.data.items[0].title).toContain('搜尋測試');
  });

  test('[P1] GET /series/search - should return 400 when query is missing', async ({ request }) => {
    // GIVEN: No search query provided

    // WHEN: Searching without query parameter
    const response = await request.get(`${API_BASE_URL}/series/search`);

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P2] GET /series/search - should return empty results for no matches', async ({ request }) => {
    // GIVEN: A search query that matches nothing

    // WHEN: Searching for non-existent series
    const response = await request.get(`${API_BASE_URL}/series/search?q=xyznonexistent99999`);

    // THEN: Should return empty array
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.items).toBeInstanceOf(Array);
    expect(body.data.items).toHaveLength(0);
  });

  // ---------------------------------------------------------------------------
  // UPDATE Tests
  // ---------------------------------------------------------------------------

  test('[P1] PUT /series/:id - should update series', async ({ request }) => {
    // GIVEN: An existing series
    const seriesData = createSeriesInput({ title: '更新前標題劇集 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    createdSeriesId = created.data.id;

    // WHEN: Updating the series
    const updateData = {
      title: '更新後標題劇集 ' + Date.now(),
      rating: 9.0,
      numberOfSeasons: 5,
    };
    const response = await request.put(`${API_BASE_URL}/series/${createdSeriesId}`, {
      data: updateData,
    });

    // THEN: Should return updated series
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.title).toBe(updateData.title);
  });

  test('[P1] PUT /series/:id - should return 404 for non-existent series', async ({ request }) => {
    // GIVEN: A non-existent series ID
    const fakeId = 'non-existent-series-id-update';

    // WHEN: Attempting to update non-existent series
    const response = await request.put(`${API_BASE_URL}/series/${fakeId}`, {
      data: { title: '新標題' },
    });

    // THEN: Should return 404
    expect(response.status()).toBe(404);
  });

  // ---------------------------------------------------------------------------
  // DELETE Tests
  // ---------------------------------------------------------------------------

  test('[P1] DELETE /series/:id - should delete series', async ({ request }) => {
    // GIVEN: An existing series
    const seriesData = createSeriesInput({ title: '將被刪除劇集 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    const seriesId = created.data.id;

    // WHEN: Deleting the series
    const response = await request.delete(`${API_BASE_URL}/series/${seriesId}`);

    // THEN: Should return 204 No Content
    expect(response.status()).toBe(204);

    // Verify series is deleted
    const getResponse = await request.get(`${API_BASE_URL}/series/${seriesId}`);
    expect(getResponse.status()).toBe(404);

    // Clear cleanup since already deleted
    createdSeriesId = '';
  });
});

// =============================================================================
// Series-Specific Field Tests
// =============================================================================

test.describe('Series API - TV-Specific Fields @api @series', () => {
  let createdSeriesId: string;

  test.afterEach(async ({ request }) => {
    if (createdSeriesId) {
      await request.delete(`${API_BASE_URL}/series/${createdSeriesId}`);
      createdSeriesId = '';
    }
  });

  test('[P2] should handle numberOfSeasons and numberOfEpisodes', async ({ request }) => {
    // GIVEN: Series with TV-specific fields
    const seriesData = createSeriesInput({
      title: '季數測試 ' + Date.now(),
      numberOfSeasons: 3,
      numberOfEpisodes: 24,
    });

    // WHEN: Creating series
    const response = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });

    // THEN: Should store and return TV-specific fields
    expect(response.status()).toBe(201);

    const body = await response.json();
    createdSeriesId = body.data.id;

    // Verify by GET
    const getResponse = await request.get(`${API_BASE_URL}/series/${createdSeriesId}`);
    const getData = await getResponse.json();
    expect(getData.data.numberOfSeasons).toBeDefined();
    expect(getData.data.numberOfEpisodes).toBeDefined();
  });

  test('[P2] should handle inProduction status update', async ({ request }) => {
    // GIVEN: An existing series
    const seriesData = createSeriesInput({ title: '製作狀態測試 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/series`, {
      data: seriesData,
    });
    const created = await createResponse.json();
    createdSeriesId = created.data.id;

    // WHEN: Updating production status
    const updateResponse = await request.put(`${API_BASE_URL}/series/${createdSeriesId}`, {
      data: {
        inProduction: false,
        status: 'Ended',
      },
    });

    // THEN: Should update status fields
    expect(updateResponse.status()).toBe(200);

    const body = await updateResponse.json();
    expect(body.success).toBe(true);
  });
});

// =============================================================================
// Response Format Validation
// =============================================================================

test.describe('Series API - Response Format @api @series', () => {
  test('[P1] should follow standard API response format', async ({ request }) => {
    // GIVEN: API is running

    // WHEN: Making any request
    const response = await request.get(`${API_BASE_URL}/series`);

    // THEN: Response should follow standard format
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('success');
    expect(body.success).toBe(true);
    expect(body).toHaveProperty('data');
  });
});
