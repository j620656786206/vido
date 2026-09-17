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
 * fresh, empty CI DB the old pattern silently self-skipped → false green.
 * dsr-2b-b re-armed the last skips (the no-metadata states).
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

test.describe('Media Detail - no metadata @e2e @media-detail @dsr-2b-b', () => {
  // dsr-2b-b: the v1 fallback panel (Story 5-11) never made it into the v2 detail
  // page; these four tests waited on it as honest skips. They now cover the v2
  // states, read from parse_status (never tmdb_id):
  //   - failed  → seeded by a real single re-match of a nonsense title (dsr-2b-a)
  //   - pending → seeded by a batch re-parse, which sets pending and runs nothing
  const movieIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteMovies(api, ...movieIds.splice(0));
  });

  async function seedFailedMovie(api: Parameters<typeof seedMovie>[0]) {
    const movie = await seedMovie(api, { title: `[E2E] zzqx nonsense ${Date.now()}` });
    movieIds.push(movie.id);
    const rematch = await api.reparseMovie(movie.id);
    expect(rematch.data?.parse_status, JSON.stringify(rematch.error)).toBe('failed');
    return movie;
  }

  test('[P0] a movie without a poster shows the gradient tile, initial skipping the bracket', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] 無資料電影 ${Date.now()}` });
    movieIds.push(movie.id);

    await page.goto(`/media/movie/${movie.id}`);

    const tile = page.getByTestId('detail-poster-fallback');
    await expect(tile).toBeVisible({ timeout: 15000 });
    await expect(tile).toHaveText('E');
  });

  test('[P0] a movie whose match failed shows the 比對失敗 block and its two ways back', async ({
    page,
    api,
  }) => {
    const movie = await seedFailedMovie(api);

    await page.goto(`/media/movie/${movie.id}`);

    const block = page.locator('[data-testid="detail-no-metadata"][data-variant="failed"]');
    await expect(block).toBeVisible({ timeout: 15000 });
    await expect(block.getByRole('heading', { name: '沒有找到這部電影的資料' })).toBeVisible();
    await expect(page.getByTestId('no-metadata-manual-match')).toBeVisible();
    await expect(page.getByTestId('no-metadata-rematch')).toBeVisible();
  });

  test('[P1] a pending movie shows 資料整理中 with 立即比對 and nothing spinning', async ({
    page,
    api,
  }) => {
    const movie = await seedMovie(api, { title: `[E2E] pending ${Date.now()}` });
    movieIds.push(movie.id);
    const batch = await api.batchReparse([movie.id], 'movie');
    expect(batch.success, JSON.stringify(batch.error)).toBe(true);
    // BatchReparse answers 200 even when an id failed; make sure this one was set.
    expect(batch.data?.success_count).toBe(1);

    await page.goto(`/media/movie/${movie.id}`);

    const block = page.locator('[data-testid="detail-no-metadata"][data-variant="pending"]');
    await expect(block).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('no-metadata-rematch')).toHaveText('立即比對');
    await expect(page.locator('.animate-spin')).toHaveCount(0);
  });

  test('[P1] 手動選片 opens the locked dialog prefilled with the cleaned file name', async ({
    page,
    api,
  }) => {
    const movie = await seedFailedMovie(api);

    await page.goto(`/media/movie/${movie.id}`);
    await page.getByTestId('no-metadata-manual-match').click();

    const dialog = page.getByTestId('manual-match-dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('searchbox')).toHaveValue(/^zzqx nonsense \d+$/);
    await expect(dialog.getByRole('combobox')).toHaveCount(0);
    await expect(dialog.getByRole('button', { name: '影集' })).toHaveCount(0);
  });

  test('[P1] picking and applying a TMDb match turns the page into a normal detail page', async ({
    page,
    api,
  }) => {
    const movie = await seedFailedMovie(api);

    await page.goto(`/media/movie/${movie.id}`);
    await page.getByTestId('no-metadata-manual-match').click();
    const dialog = page.getByTestId('manual-match-dialog');
    await dialog.getByRole('searchbox').fill('Fight Club');
    await dialog.getByTestId('manual-match-result').first().click({ timeout: 15000 });
    await dialog.getByTestId('manual-match-apply').click();

    await expect(dialog).toBeHidden({ timeout: 15000 });
    await expect(page.getByTestId('detail-no-metadata')).toHaveCount(0, { timeout: 15000 });
    await expect(page.getByTestId('detail-hero-v2')).not.toContainText('zzqx');
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

    // THEN: Should show the v2 not-found state — the same one a missing row shows
    // (dsr-2 AC #8 unified the route-level 「404 · 找不到該媒體內容」 into it).
    await page.waitForLoadState('networkidle');
    const notFound = page.getByTestId('detail-not-found');
    await expect(notFound).toBeVisible({ timeout: 15000 });
    await expect(notFound.getByText('找不到這部影片')).toBeVisible();
  });

  test('[P1] should show error for non-existent UUID', async ({ page }) => {
    // GIVEN: A UUID that doesn't exist in the database

    // WHEN: Navigating to a non-existent movie
    // Note: react-query retries once (queryClient.ts retry: 1) before isError=true —
    // even for a 404 (disc-2026-09-detail-404-still-retried)
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

    // WHEN: User clicks the back to library button. Scoped to the not-found panel:
    // DetailNotFoundV2 also has an icon back button with the same accessible name
    // (aria-label), which would make an unscoped role query a strict-mode violation.
    const backButton = page
      .getByTestId('detail-not-found')
      .getByRole('button', { name: '返回媒體庫' });
    await expect(backButton).toBeVisible({ timeout: 15000 });

    // THEN: Should navigate to library page
    await backButton.click();
    await expect(page).toHaveURL(/\/library/, { timeout: 10000 });
  });
});
