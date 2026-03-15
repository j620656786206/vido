/**
 * Media Detail Page E2E Tests
 *
 * Tests for the media detail functionality including:
 * - Movie detail display
 * - TV Show detail display
 * - Navigation and routing
 * - Error handling for invalid IDs
 *
 * Prerequisites:
 * - Frontend running on port 4200: npx nx serve web
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @e2e @media-detail @regression
 */

import { test, expect } from '../support/fixtures';

// =============================================================================
// Test Data: Known TMDb IDs
// =============================================================================

const testData = {
  // Well-known movie for consistent testing
  movie: {
    id: 27205, // Inception
    title: /Inception|全面啟動/i,
    year: '2010',
  },
  // Well-known TV show
  tvShow: {
    id: 1399, // Game of Thrones
    title: /Game of Thrones|乩|權力遊戲/i,
    year: '2011',
  },
  // Invalid IDs for error testing
  invalidMovieId: 99999999,
  invalidTvId: 99999999,
};

// =============================================================================
// Movie Detail Tests
// =============================================================================

test.describe('Media Detail - Movie @e2e @media-detail', () => {
  test('[P0] should display movie detail page', async ({ page }) => {
    // GIVEN: User navigates to a valid movie detail page
    await page.goto(`/media/movie/${testData.movie.id}`);

    // WHEN: Page loads
    await page.waitForLoadState('networkidle');

    // THEN: Should display movie details
    await expect(page.getByText(testData.movie.title).first()).toBeVisible({ timeout: 15000 });
  });

  // [Downgraded to unit] poster, overview, year, genres, rating → MediaDetailPanel.spec.tsx
});

// =============================================================================
// TV Show Detail Tests
// =============================================================================

test.describe('Media Detail - TV Show @e2e @media-detail', () => {
  test('[P0] should display TV show detail page', async ({ page }) => {
    // GIVEN: User navigates to a valid TV show detail page
    await page.goto(`/media/tv/${testData.tvShow.id}`);

    // WHEN: Page loads
    await page.waitForLoadState('networkidle');

    // THEN: Should display TV show details
    await expect(page.getByText(testData.tvShow.title).first()).toBeVisible({ timeout: 15000 });
  });

  // [Downgraded to unit] TV poster, TV info, air date → MediaDetailPanel.spec.tsx, TVShowInfo.spec.tsx
});

// =============================================================================
// Credits Section Tests
// =============================================================================

// [Downgraded to unit] cast members, director → CreditsSection.spec.tsx

// =============================================================================
// Navigation Tests
// =============================================================================

test.describe('Media Detail - Navigation @e2e @media-detail', () => {
  test('[P1] should navigate back to search from detail', async ({ page }) => {
    // GIVEN: User is on movie detail page after search
    await page.goto('/search?q=Inception');
    await page.waitForLoadState('networkidle');

    // Navigate to detail
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();
    await page.waitForURL(/\/media\//);

    // WHEN: User clicks close button
    const closeButton = page.locator('[data-testid="side-panel-close"]');
    await expect(closeButton).toBeVisible({ timeout: 5000 });
    await closeButton.click();

    // THEN: Should return to search page
    await expect(page).toHaveURL(/\/search/);
  });

  test('[P1] should handle direct URL navigation', async ({ page }) => {
    // GIVEN: User directly navigates to a movie detail URL

    // WHEN: Navigating directly
    await page.goto(`/media/movie/${testData.movie.id}`);

    // THEN: Page should load correctly
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(testData.movie.title).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P2] should update URL when opening detail panel', async ({ page }) => {
    // GIVEN: User is on search results
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on a movie card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    if (await firstCard.isVisible({ timeout: 10000 })) {
      await firstCard.click();

      // THEN: URL should update to include movie ID
      await expect(page).toHaveURL(/\/media\/movie\/\d+/);
    }
  });
});

// =============================================================================
// Error Handling Tests
// =============================================================================

test.describe('Media Detail - Error Handling @e2e @media-detail', () => {
  // Note: Non-existent TMDb IDs (e.g., 99999999) pass route validation but fail API call
  // Only invalid types or ID formats trigger the 404 page

  test('[P1] should show 404 for invalid media type', async ({ page }) => {
    // GIVEN: User navigates with invalid media type

    // WHEN: Using invalid type parameter
    await page.goto('/media/invalid/12345');

    // THEN: Should show 404 page
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('404')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('找不到該媒體內容')).toBeVisible();
  });

  test('[P1] should show 404 for invalid ID format', async ({ page }) => {
    // GIVEN: User navigates with non-numeric ID

    // WHEN: Using non-numeric ID
    await page.goto('/media/movie/invalid-id');

    // THEN: Should show 404 page
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('404')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('找不到該媒體內容')).toBeVisible();
  });

  test('[P2] should provide navigation back from 404', async ({ page }) => {
    // GIVEN: User is on 404 page (invalid type triggers 404)
    await page.goto('/media/invalid/12345');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks the back to search button
    const backButton = page.getByRole('button', { name: '返回搜尋' });
    await expect(backButton).toBeVisible({ timeout: 15000 });

    // THEN: Should navigate back to search page
    await backButton.click();
    await expect(page).toHaveURL(/\/search/);
  });
});

// =============================================================================
// Side Panel Tests
// =============================================================================

test.describe('Media Detail - Side Panel @e2e @media-detail', () => {
  test('[P1] should open detail in side panel from search', async ({ page }) => {
    // GIVEN: User is on search results
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on a movie card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // THEN: Side panel should open with movie details
    const sidePanel = page.locator('[data-testid="side-panel"]');
    await expect(sidePanel).toBeVisible({ timeout: 10000 });
  });

  test('[P2] should close side panel on X button click', async ({ page }) => {
    // GIVEN: Side panel is open
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    const sidePanel = page.locator('[data-testid="side-panel"]');
    await expect(sidePanel).toBeVisible({ timeout: 10000 });

    // WHEN: User clicks close button
    const closeButton = page.locator('[data-testid="side-panel-close"]');
    await closeButton.click();

    // THEN: Side panel should close (navigates to search page)
    await expect(page).toHaveURL(/\/search/);
  });

  test('[P2] should close side panel on escape key', async ({ page }) => {
    // GIVEN: Side panel is open
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // Wait for panel to be visible
    const sidePanel = page.locator('[data-testid="side-panel"]');
    await expect(sidePanel).toBeVisible({ timeout: 10000 });

    // WHEN: User presses Escape
    await page.keyboard.press('Escape');

    // THEN: Side panel should close (navigates to search page)
    await expect(page).toHaveURL(/\/search/, { timeout: 5000 });
  });
});
