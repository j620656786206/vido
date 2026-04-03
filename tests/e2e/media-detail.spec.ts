/**
 * Media Detail Page E2E Tests
 *
 * Tests for the media detail functionality after bugfix-1 refactor:
 * - Full-page layout (no side panel)
 * - UUID-based routing (not TMDb IDs)
 * - Local API as primary data source
 * - Fallback UI for missing metadata
 *
 * Prerequisites:
 * - Frontend running on port 4200: npx nx serve web
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @e2e @media-detail @regression
 */

import { test, expect } from '../support/fixtures';

// =============================================================================
// Movie Detail Tests
// =============================================================================

test.describe('Media Detail - Movie @e2e @media-detail', () => {
  test('[P0] should display movie detail page via library navigation', async ({ page, api }) => {
    // GIVEN: Library has at least one movie
    const moviesRes = await api.listMovies({ page: 1, pageSize: 1 });
    const movie = moviesRes.data?.items?.[0];

    if (!movie) {
      // Create a movie via API if none exists
      const createRes = await api.createMovie({
        title: 'E2E Test Movie',
        releaseDate: '2024-01-01',
      });
      test.skip(!createRes.data, 'No movies available and cannot create one');
      return;
    }

    // WHEN: Navigate to movie detail page using UUID
    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Should display movie title
    await expect(page.getByText(movie.title).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should display back button on detail page', async ({ page, api }) => {
    // GIVEN: A movie exists
    const moviesRes = await api.listMovies({ page: 1, pageSize: 1 });
    const movie = moviesRes.data?.items?.[0];
    test.skip(!movie, 'No movies available');

    // WHEN: Navigate to detail page
    await page.goto(`/media/movie/${movie!.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Back button should be visible
    await expect(page.getByText('返回媒體庫').first()).toBeVisible({ timeout: 15000 });
  });
});

// =============================================================================
// TV Show Detail Tests
// =============================================================================

test.describe('Media Detail - TV Show @e2e @media-detail', () => {
  test('[P0] should display TV show detail page via library navigation', async ({ page, api }) => {
    // GIVEN: Library has at least one series
    const seriesRes = await api.listSeries({ page: 1, pageSize: 1 });
    const series = seriesRes.data?.items?.[0];

    if (!series) {
      test.skip(true, 'No series available in library');
      return;
    }

    // WHEN: Navigate to series detail page using UUID
    await page.goto(`/media/tv/${series.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Should display series title
    await expect(page.getByText(series.title).first()).toBeVisible({ timeout: 15000 });
  });
});

// =============================================================================
// Navigation Tests
// =============================================================================

test.describe('Media Detail - Navigation @e2e @media-detail', () => {
  test('[P1] should navigate back to library from detail', async ({ page, api }) => {
    // GIVEN: A movie exists and user is on its detail page
    const moviesRes = await api.listMovies({ page: 1, pageSize: 1 });
    const movie = moviesRes.data?.items?.[0];
    test.skip(!movie, 'No movies available');

    await page.goto(`/media/movie/${movie!.id}`);
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks back button
    const backButton = page.getByText('返回媒體庫').first();
    await expect(backButton).toBeVisible({ timeout: 15000 });
    await backButton.click();

    // THEN: Should navigate to library page
    await expect(page).toHaveURL(/\/library/, { timeout: 10000 });
  });

  test('[P1] should handle direct URL navigation', async ({ page, api }) => {
    // GIVEN: A movie exists
    const moviesRes = await api.listMovies({ page: 1, pageSize: 1 });
    const movie = moviesRes.data?.items?.[0];
    test.skip(!movie, 'No movies available');

    // WHEN: Navigating directly to the movie detail URL
    await page.goto(`/media/movie/${movie!.id}`);

    // THEN: Page should load correctly
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(movie!.title).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should navigate to detail from library grid click', async ({ page }) => {
    // GIVEN: User is on library page
    await page.goto('/library');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on a poster card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    if (!(await firstCard.isVisible({ timeout: 10000 }))) {
      test.skip(true, 'No media cards visible in library');
      return;
    }
    await firstCard.click();

    // THEN: Should navigate to a detail page with UUID
    await expect(page).toHaveURL(/\/media\/(movie|tv)\/[a-f0-9-]+/, { timeout: 10000 });
  });
});

// =============================================================================
// Fallback UI Tests (Story 5-11)
// =============================================================================

test.describe('Media Detail - Fallback UI @e2e @media-detail @story-5-11', () => {
  test('[P0] should display color placeholder for media without poster', async ({ page, api }) => {
    // GIVEN: Find a movie without TMDB metadata (tmdbId = 0)
    const moviesRes = await api.listMovies({ page: 1, pageSize: 50 });
    const noMetadataMovie = moviesRes.data?.items?.find(
      (m: { tmdb_id?: number }) => !m.tmdb_id || m.tmdb_id === 0
    );
    test.skip(!noMetadataMovie, 'No movies without metadata available');

    // WHEN: Navigate to the detail page
    await page.goto(`/media/movie/${noMetadataMovie!.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Color placeholder should be rendered (not the Film icon placeholder)
    const placeholder = page.getByTestId('color-placeholder');
    await expect(placeholder).toBeAttached({ timeout: 15000 });
  });

  test('[P0] should display failed state with file info and CTAs', async ({ page, api }) => {
    // GIVEN: Find a movie with failed/empty parse status
    const moviesRes = await api.listMovies({ page: 1, pageSize: 50 });
    const failedMovie = moviesRes.data?.items?.find(
      (m: { tmdb_id?: number; parse_status?: string }) =>
        (!m.tmdb_id || m.tmdb_id === 0) && (m.parse_status === 'failed' || m.parse_status === '')
    );
    test.skip(!failedMovie, 'No movies with failed parse status available');

    // WHEN: Navigate to the detail page
    await page.goto(`/media/movie/${failedMovie!.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Failed state UI should be visible
    await expect(page.getByTestId('fallback-failed')).toBeAttached({ timeout: 15000 });
    await expect(page.getByText('我們找不到這部電影的資料')).toBeVisible();
    await expect(page.getByText('檔案資訊')).toBeVisible();

    // AND: CTA buttons should be present
    await expect(page.getByTestId('cta-search-metadata')).toBeVisible();
    await expect(page.getByTestId('cta-manual-edit')).toBeVisible();
  });

  test('[P1] should display pending state with spinner', async ({ page, api }) => {
    // GIVEN: Find a movie with pending parse status
    const moviesRes = await api.listMovies({ page: 1, pageSize: 50 });
    const pendingMovie = moviesRes.data?.items?.find(
      (m: { tmdb_id?: number; parse_status?: string }) =>
        (!m.tmdb_id || m.tmdb_id === 0) && m.parse_status === 'pending'
    );
    test.skip(!pendingMovie, 'No movies with pending parse status available');

    // WHEN: Navigate to the detail page
    await page.goto(`/media/movie/${pendingMovie!.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Pending state UI should be rendered
    // Use toBeAttached() for animated elements per project gotcha
    await expect(page.getByTestId('fallback-pending')).toBeAttached({ timeout: 15000 });
    await expect(page.getByTestId('pending-spinner')).toBeAttached();
    await expect(page.getByText('正在搜尋電影資訊⋯')).toBeVisible();
    await expect(page.getByTestId('pending-progress')).toBeAttached();
  });

  test('[P1] search metadata CTA should navigate to search page', async ({ page, api }) => {
    // GIVEN: A movie with failed status on detail page
    const moviesRes = await api.listMovies({ page: 1, pageSize: 50 });
    const failedMovie = moviesRes.data?.items?.find(
      (m: { tmdb_id?: number; parse_status?: string }) =>
        (!m.tmdb_id || m.tmdb_id === 0) && (m.parse_status === 'failed' || m.parse_status === '')
    );
    test.skip(!failedMovie, 'No movies with failed parse status available');

    await page.goto(`/media/movie/${failedMovie!.id}`);
    await page.waitForLoadState('networkidle');

    // WHEN: Click the search metadata button
    const searchBtn = page.getByTestId('cta-search-metadata');
    await expect(searchBtn).toBeVisible({ timeout: 15000 });
    await searchBtn.click();

    // THEN: Should navigate to search page with query parameter
    await expect(page).toHaveURL(/\/search\?q=.+/, { timeout: 10000 });
  });
});

// =============================================================================
// Error Handling Tests
// =============================================================================

test.describe('Media Detail - Error Handling @e2e @media-detail', () => {
  test('[P1] should show 404 for invalid media type', async ({ page }) => {
    // GIVEN: User navigates with invalid media type

    // WHEN: Using invalid type parameter
    await page.goto('/media/invalid/12345');

    // THEN: Should show 404 page
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('404')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('找不到該媒體內容')).toBeVisible();
  });

  test('[P1] should show error for non-existent UUID', async ({ page }) => {
    // GIVEN: A UUID that doesn't exist in the database

    // WHEN: Navigating to a non-existent movie
    // Note: react-query retries 3x with exponential backoff before isError=true
    await page.goto('/media/movie/00000000-0000-0000-0000-000000000000');

    // THEN: Should eventually show 404/error state (after query retries exhaust)
    await expect(page.getByText('找不到該媒體內容')).toBeVisible({
      timeout: 30000,
    });
  });

  test('[P2] should provide navigation back from 404', async ({ page }) => {
    // GIVEN: User is on 404 page (invalid type triggers 404)
    await page.goto('/media/invalid/12345');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks the back to library button
    const backButton = page.getByRole('button', { name: '返回媒體庫' });
    await expect(backButton).toBeVisible({ timeout: 15000 });

    // THEN: Should navigate to library page
    await backButton.click();
    await expect(page).toHaveURL(/\/library/, { timeout: 10000 });
  });
});
