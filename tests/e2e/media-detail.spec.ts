/**
 * Media Detail Page E2E Tests
 *
 * Tests for the media detail functionality after bugfix-1 refactor:
 * - Full-page layout (no side panel)
 * - UUID-based routing (not TMDb IDs)
 * - Local API as primary data source
 * - Fallback UI for missing metadata
 *
 * Story 20-2: every test now SEEDS the row it needs via the API and cleans up
 * afterwards, instead of `test.skip(!movie, 'No movies available')`. On the
 * fresh, empty CI DB the old pattern silently self-skipped → false green. The
 * only remaining skip is the `pending` fallback (the create endpoint can't set
 * `parse_status` — see the comment there).
 *
 * Prerequisites:
 * - Frontend running on port 4200: npx nx serve web
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @e2e @media-detail @regression
 */

import { test, expect } from '../support/fixtures';
import { seedMovie, seedSeries, deleteMovies, deleteSeries } from '../support/helpers/seed-helpers';

// A real TMDb id keeps `hasMetadata` true so the full local detail view (with
// the title) renders instead of the no-metadata fallback. Any TMDb enrichment
// that fails (e.g. no TMDB_API_KEY in CI) degrades soft — the title comes from
// local data regardless.
const SEED_TMDB_ID = 27205; // Inception

// =============================================================================
// Movie Detail Tests
// =============================================================================

test.describe('Media Detail - Movie @e2e @media-detail', () => {
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  test('[P0] should display movie detail page via library navigation', async ({ page, api }) => {
    // GIVEN: A movie exists in the library
    const movie = await seedMovie(api, {
      title: `E2E 詳情電影 ${Date.now()}`,
      tmdbId: SEED_TMDB_ID,
      posterPath: '/seed-poster.jpg',
    });
    movieIds.push(movie.id);

    // WHEN: Navigate to movie detail page using UUID
    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Should display movie title
    await expect(page.getByText(movie.title).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should display back button on detail page', async ({ page, api }) => {
    // GIVEN: A movie exists
    const movie = await seedMovie(api, { tmdbId: SEED_TMDB_ID });
    movieIds.push(movie.id);

    // WHEN: Navigate to detail page
    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Back button should be visible (v2 hero back affordance, ux3-cutover-3)
    await expect(page.getByTestId('detail-back').first()).toBeVisible({ timeout: 15000 });
  });
});

// =============================================================================
// TV Show Detail Tests
// =============================================================================

test.describe('Media Detail - TV Show @e2e @media-detail', () => {
  const seriesIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteSeries(api, ...seriesIds.splice(0));
  });

  test('[P0] should display TV show detail page via library navigation', async ({ page, api }) => {
    // GIVEN: A series exists in the library
    const series = await seedSeries(api, {
      title: `E2E 詳情影集 ${Date.now()}`,
      tmdbId: 1396, // Breaking Bad
      numberOfSeasons: 5,
      numberOfEpisodes: 62,
    });
    seriesIds.push(series.id);

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
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  test('[P1] should navigate back to library from detail', async ({ page, api }) => {
    // GIVEN: A movie exists and user is on its detail page
    const movie = await seedMovie(api, { tmdbId: SEED_TMDB_ID });
    movieIds.push(movie.id);

    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks the v2 hero back button
    const backButton = page.getByTestId('detail-back').first();
    await expect(backButton).toBeVisible({ timeout: 15000 });
    await backButton.click();

    // THEN: Should navigate to library page
    await expect(page).toHaveURL(/\/library/, { timeout: 10000 });
  });

  test('[P1] should handle direct URL navigation', async ({ page, api }) => {
    // GIVEN: A movie exists
    const movie = await seedMovie(api, { tmdbId: SEED_TMDB_ID });
    movieIds.push(movie.id);

    // WHEN: Navigating directly to the movie detail URL
    await page.goto(`/media/movie/${movie.id}`);

    // THEN: Page should load correctly
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(movie.title).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should navigate to detail from library grid click', async ({ page, api }) => {
    // GIVEN: At least one media card exists in the library
    const movie = await seedMovie(api, { tmdbId: SEED_TMDB_ID, posterPath: '/seed-poster.jpg' });
    movieIds.push(movie.id);

    await page.goto('/library');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on the first poster card
    const firstCard = page.locator('[data-testid^="poster-v2-"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // THEN: Should navigate to a detail page with UUID
    await expect(page).toHaveURL(/\/media\/(movie|tv)\/[a-f0-9-]+/, { timeout: 10000 });
  });
});

// =============================================================================
// Fallback UI Tests (Story 5-11)
// =============================================================================

test.describe('Media Detail - Fallback UI @e2e @media-detail @story-5-11', () => {
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  // A movie created via POST /movies with no tmdb_id → tmdb_id=0, parse_status=''
  // → hasMetadata=false, not 'pending' → ColorPlaceholder + FallbackFailed render
  // (see routes/media/$type.$id.tsx). No poster_path → ColorPlaceholder.
  async function seedNoMetadataMovie(api: Parameters<typeof seedMovie>[0]) {
    const movie = await seedMovie(api, { title: `E2E 無資料電影 ${Date.now()}` });
    movieIds.push(movie.id);
    return movie;
  }

  // ux3-cutover-3: LocalDetailV2 does NOT port the Story 5-11 fallback UX
  // (color placeholder testid / FallbackFailed panel / 搜尋中繼資料+手動編輯 CTAs) —
  // filed as disc-2026-07-v2-detail-fallback-states in sprint-status. Skipped
  // honestly (not deleted): they re-arm when the v2 fallback states land.
  test.skip('[P0] should display color placeholder for media without poster', async ({
    page,
    api,
  }) => {
    // GIVEN: A movie without TMDB metadata (tmdb_id = 0) and no poster
    const movie = await seedNoMetadataMovie(api);

    // WHEN: Navigate to the detail page
    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Color placeholder should be rendered (not the Film icon placeholder)
    await expect(page.getByTestId('color-placeholder')).toBeAttached({ timeout: 15000 });
  });

  test.skip('[P0] should display failed state with file info and CTAs', async ({ page, api }) => {
    // GIVEN: A movie with empty/failed parse status (no metadata)
    const movie = await seedNoMetadataMovie(api);

    // WHEN: Navigate to the detail page
    await page.goto(`/media/movie/${movie.id}`);
    await page.waitForLoadState('networkidle');

    // THEN: Failed state UI should be visible
    await expect(page.getByTestId('fallback-failed')).toBeAttached({ timeout: 15000 });
    await expect(page.getByText('我們找不到這部電影的資料')).toBeVisible();
    await expect(page.getByText('檔案資訊')).toBeVisible();

    // AND: CTA buttons should be present
    await expect(page.getByTestId('cta-search-metadata')).toBeVisible();
    await expect(page.getByTestId('cta-manual-edit')).toBeVisible();
  });

  // The 'pending' fallback requires parse_status='pending', which the create
  // endpoint (CreateMovieRequest) cannot set — only the scanner/parse_queue
  // pipeline produces that state. Seeding a 'pending' movie needs either a new
  // create field or the Tier-2 test-only seed endpoint. Tracked as
  // sprint-status `story-20-4-season-accordion-e2e` follow-up (parse-status
  // seeding). Kept skipped honestly rather than self-skipping at runtime.
  test.skip('[P1] should display pending state with spinner — needs parse_status seeding', () => {
    // Intentionally empty: see comment above.
  });

  test.skip('[P1] search metadata CTA should navigate to search page', async ({ page, api }) => {
    // GIVEN: A no-metadata movie on its (failed-state) detail page
    const movie = await seedNoMetadataMovie(api);

    await page.goto(`/media/movie/${movie.id}`);
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

    // THEN: v2 DetailNotFoundV2 renders after query retries exhaust (ux3-cutover-3)
    await expect(page.getByTestId('detail-not-found')).toBeVisible({
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
