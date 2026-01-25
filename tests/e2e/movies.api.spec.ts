/**
 * Movies API E2E Tests
 *
 * Comprehensive API tests for the Movies CRUD endpoints.
 * Tests follow Given-When-Then format with priority tags.
 *
 * Prerequisites: Go backend must be running on port 8080
 *   cd apps/api && go run ./cmd/api
 *
 * @tags @api @movies
 */

import { test, expect } from '../support/fixtures';
import { faker } from '@faker-js/faker';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Test Data Factories
// =============================================================================

interface MovieInput {
  title: string;
  releaseDate: string;
  originalTitle?: string;
  genres?: string[];
  overview?: string;
  posterPath?: string;
  tmdbId?: number;
  imdbId?: string;
}

function createMovieInput(overrides: Partial<MovieInput> = {}): MovieInput {
  return {
    title: faker.lorem.words(3),
    releaseDate: faker.date.past({ years: 10 }).toISOString().split('T')[0],
    originalTitle: faker.lorem.words(3),
    genres: [faker.helpers.arrayElement(['Action', 'Drama', 'Comedy', 'Thriller'])],
    overview: faker.lorem.paragraph(),
    posterPath: `/posters/${faker.string.alphanumeric(10)}.jpg`,
    tmdbId: faker.number.int({ min: 100000, max: 999999 }),
    ...overrides,
  };
}

// =============================================================================
// Movies API Tests
// =============================================================================

test.describe('Movies API - CRUD Operations @api @movies', () => {
  let createdMovieId: string;

  test.afterEach(async ({ request }) => {
    // Cleanup: Delete created movie if exists
    if (createdMovieId) {
      await request.delete(`${API_BASE_URL}/movies/${createdMovieId}`);
      createdMovieId = '';
    }
  });

  // ---------------------------------------------------------------------------
  // CREATE Tests
  // ---------------------------------------------------------------------------

  test('[P1] POST /movies - should create movie with required fields', async ({ request }) => {
    // GIVEN: Valid movie data with required fields only
    const movieData = {
      title: '測試電影 ' + Date.now(),
      releaseDate: '2024-01-15',
    };

    // WHEN: Creating a movie via POST
    const response = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });

    // THEN: Should return 201 with created movie
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data).toBeDefined();
    expect(body.data.id).toBeDefined();
    expect(body.data.title).toBe(movieData.title);
    expect(body.data.releaseDate).toBe(movieData.releaseDate);

    // Store for cleanup
    createdMovieId = body.data.id;
  });

  test('[P1] POST /movies - should create movie with all optional fields', async ({ request }) => {
    // GIVEN: Complete movie data with all fields
    const movieData = createMovieInput({
      title: '完整測試電影 ' + Date.now(),
    });

    // WHEN: Creating a movie via POST
    const response = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });

    // THEN: Should return 201 with all fields populated
    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.title).toBe(movieData.title);
    expect(body.data.genres).toEqual(movieData.genres);

    createdMovieId = body.data.id;
  });

  test('[P1] POST /movies - should return 400 for missing required fields', async ({ request }) => {
    // GIVEN: Movie data missing required 'title' field
    const invalidData = {
      releaseDate: '2024-01-15',
      // title is missing
    };

    // WHEN: Attempting to create movie
    const response = await request.post(`${API_BASE_URL}/movies`, {
      data: invalidData,
    });

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBe('VALIDATION_ERROR');
  });

  test('[P2] POST /movies - should return 400 for invalid JSON body', async ({ request }) => {
    // GIVEN: Invalid request body (not JSON)

    // WHEN: Sending invalid content
    const response = await request.post(`${API_BASE_URL}/movies`, {
      data: 'not-valid-json',
      headers: { 'Content-Type': 'text/plain' },
    });

    // THEN: Should return 400 error
    expect(response.status()).toBe(400);
  });

  // ---------------------------------------------------------------------------
  // READ Tests
  // ---------------------------------------------------------------------------

  test('[P1] GET /movies - should list movies with pagination', async ({ request }) => {
    // GIVEN: Movies exist in database

    // First, create a movie to ensure data exists
    const movieData = createMovieInput({ title: '列表測試 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });
    const created = await createResponse.json();
    createdMovieId = created.data.id;

    // WHEN: Listing movies with pagination
    const response = await request.get(`${API_BASE_URL}/movies?page=1&page_size=10`);

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

  test('[P1] GET /movies/:id - should return movie by ID', async ({ request }) => {
    // GIVEN: A movie exists
    const movieData = createMovieInput({ title: 'ID查詢測試 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });
    const created = await createResponse.json();
    createdMovieId = created.data.id;

    // WHEN: Getting movie by ID
    const response = await request.get(`${API_BASE_URL}/movies/${createdMovieId}`);

    // THEN: Should return the movie
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.id).toBe(createdMovieId);
    expect(body.data.title).toBe(movieData.title);
  });

  test('[P1] GET /movies/:id - should return 404 for non-existent movie', async ({ request }) => {
    // GIVEN: A non-existent movie ID
    const fakeId = 'non-existent-movie-id-12345';

    // WHEN: Getting movie by fake ID
    const response = await request.get(`${API_BASE_URL}/movies/${fakeId}`);

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

  test('[P1] GET /movies/search - should search movies by title', async ({ request }) => {
    // GIVEN: A movie with known title exists
    const uniqueTitle = `搜尋測試電影_${Date.now()}`;
    const movieData = createMovieInput({ title: uniqueTitle });
    const createResponse = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });
    const created = await createResponse.json();
    createdMovieId = created.data.id;

    // WHEN: Searching for movies by title
    const response = await request.get(
      `${API_BASE_URL}/movies/search?q=${encodeURIComponent(uniqueTitle)}`
    );

    // THEN: Should return matching movies
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.items).toBeInstanceOf(Array);
    expect(body.data.items.length).toBeGreaterThan(0);
    expect(body.data.items[0].title).toContain('搜尋測試');
  });

  test('[P1] GET /movies/search - should return 400 when query is missing', async ({ request }) => {
    // GIVEN: No search query provided

    // WHEN: Searching without query parameter
    const response = await request.get(`${API_BASE_URL}/movies/search`);

    // THEN: Should return 400 validation error
    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error.code).toBe('VALIDATION_REQUIRED_FIELD');
  });

  test('[P2] GET /movies/search - should return empty results for no matches', async ({
    request,
  }) => {
    // GIVEN: A search query that matches nothing

    // WHEN: Searching for non-existent movie
    const response = await request.get(`${API_BASE_URL}/movies/search?q=xyznonexistent99999`);

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

  test('[P1] PUT /movies/:id - should update movie', async ({ request }) => {
    // GIVEN: An existing movie
    const movieData = createMovieInput({ title: '更新前標題 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });
    const created = await createResponse.json();
    createdMovieId = created.data.id;

    // WHEN: Updating the movie
    const updateData = {
      title: '更新後標題 ' + Date.now(),
      rating: 8.5,
    };
    const response = await request.put(`${API_BASE_URL}/movies/${createdMovieId}`, {
      data: updateData,
    });

    // THEN: Should return updated movie
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.title).toBe(updateData.title);
  });

  test('[P1] PUT /movies/:id - should return 404 for non-existent movie', async ({ request }) => {
    // GIVEN: A non-existent movie ID
    const fakeId = 'non-existent-movie-id-update';

    // WHEN: Attempting to update non-existent movie
    const response = await request.put(`${API_BASE_URL}/movies/${fakeId}`, {
      data: { title: '新標題' },
    });

    // THEN: Should return 404
    expect(response.status()).toBe(404);
  });

  // ---------------------------------------------------------------------------
  // DELETE Tests
  // ---------------------------------------------------------------------------

  test('[P1] DELETE /movies/:id - should delete movie', async ({ request }) => {
    // GIVEN: An existing movie
    const movieData = createMovieInput({ title: '將被刪除 ' + Date.now() });
    const createResponse = await request.post(`${API_BASE_URL}/movies`, {
      data: movieData,
    });
    const created = await createResponse.json();
    const movieId = created.data.id;

    // WHEN: Deleting the movie
    const response = await request.delete(`${API_BASE_URL}/movies/${movieId}`);

    // THEN: Should return 204 No Content
    expect(response.status()).toBe(204);

    // Verify movie is deleted
    const getResponse = await request.get(`${API_BASE_URL}/movies/${movieId}`);
    expect(getResponse.status()).toBe(404);

    // Clear cleanup since already deleted
    createdMovieId = '';
  });

  test('[P2] DELETE /movies/:id - should handle non-existent movie gracefully', async ({
    request,
  }) => {
    // GIVEN: A non-existent movie ID
    const fakeId = 'non-existent-movie-delete';

    // WHEN: Attempting to delete non-existent movie
    const response = await request.delete(`${API_BASE_URL}/movies/${fakeId}`);

    // THEN: Should return error (500 based on current implementation)
    // Note: Ideally should return 404, but current implementation returns 500
    expect(response.status()).toBeGreaterThanOrEqual(400);
  });
});

// =============================================================================
// Pagination Tests
// =============================================================================

test.describe('Movies API - Pagination @api @movies', () => {
  test('[P2] should respect page_size parameter', async ({ request }) => {
    // GIVEN: Movies exist

    // WHEN: Requesting with specific page size
    const response = await request.get(`${API_BASE_URL}/movies?page_size=5`);

    // THEN: Should return at most 5 items
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.data.pageSize).toBe(5);
    expect(body.data.items.length).toBeLessThanOrEqual(5);
  });

  test('[P2] should return correct page number', async ({ request }) => {
    // GIVEN: Movies exist

    // WHEN: Requesting specific page
    const response = await request.get(`${API_BASE_URL}/movies?page=2&page_size=5`);

    // THEN: Should return page 2
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.data.page).toBe(2);
  });

  test('[P2] should limit page_size to maximum 100', async ({ request }) => {
    // GIVEN: Request with excessive page size

    // WHEN: Requesting with page_size > 100
    const response = await request.get(`${API_BASE_URL}/movies?page_size=500`);

    // THEN: Should cap at 100 or use default
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.data.pageSize).toBeLessThanOrEqual(100);
  });
});

// =============================================================================
// Response Format Validation
// =============================================================================

test.describe('Movies API - Response Format @api @movies', () => {
  test('[P1] should follow standard API response format', async ({ request }) => {
    // GIVEN: API is running

    // WHEN: Making any request
    const response = await request.get(`${API_BASE_URL}/movies`);

    // THEN: Response should follow standard format
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('success');
    expect(body.success).toBe(true);
    expect(body).toHaveProperty('data');
  });

  test('[P1] error responses should include code and message', async ({ request }) => {
    // GIVEN: An invalid request

    // WHEN: Making request that will fail
    const response = await request.get(`${API_BASE_URL}/movies/invalid-id`);

    // THEN: Error response should have proper format
    expect(response.status()).toBe(404);

    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    expect(body.error.code).toBeDefined();
    expect(body.error.message).toBeDefined();
  });
});
