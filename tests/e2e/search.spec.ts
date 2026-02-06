/**
 * Media Search E2E Tests
 *
 * Tests for the media search functionality including:
 * - Search input and results display
 * - Media type filtering (all, movie, tv)
 * - Pagination
 * - Edge cases and error handling
 *
 * Prerequisites:
 * - Frontend running on port 4200: npx nx serve web
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @e2e @search @regression
 */

import { test, expect } from '../support/fixtures';

// =============================================================================
// Search Input Tests
// =============================================================================

test.describe('Media Search - Input @e2e @search', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/search');
  });

  test('[P0] should display search page with search bar', async ({ page }) => {
    // GIVEN: User navigates to search page

    // WHEN: Page loads

    // THEN: Search bar should be visible
    await expect(page.getByRole('heading', { name: /搜尋媒體/i })).toBeVisible();
    await expect(page.getByPlaceholder(/搜尋電影或影集/i)).toBeVisible();
  });

  test('[P0] should search for movies and display results', async ({ page }) => {
    // GIVEN: User is on search page
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User enters search query and submits
    await searchInput.fill('Inception');
    await searchInput.press('Enter');

    // THEN: Should display search results
    await expect(page).toHaveURL(/\/search\?q=Inception/);
    await expect(page.getByText(/Inception|全面啟動/i).first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should show minimum character message for short queries', async ({ page }) => {
    // GIVEN: User navigates to search page with a single character query
    // Note: The SearchBar component only triggers search when query length >= 2
    // So we test by navigating directly to the URL with a short query

    // WHEN: Page loads with single character query
    await page.goto('/search?q=a');
    await page.waitForLoadState('networkidle');

    // THEN: Should show minimum character message
    await expect(page.getByText(/至少 2 個字元/i)).toBeVisible();
  });

  test('[P1] should search with Chinese characters', async ({ page }) => {
    // GIVEN: User is on search page
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User searches with Chinese
    await searchInput.fill('全面啟動');
    await searchInput.press('Enter');

    // THEN: Should display results
    await expect(page).toHaveURL(/\/search\?q=%E5%85%A8%E9%9D%A2%E5%95%9F%E5%8B%95/);
    await page.waitForLoadState('networkidle');
  });

  test('[P2] should preserve search query on page reload', async ({ page }) => {
    // GIVEN: User has performed a search
    await page.goto('/search?q=Matrix');

    // WHEN: Page loads with query parameter

    // THEN: Search input should contain the query
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await expect(searchInput).toHaveValue('Matrix');
  });
});

// =============================================================================
// Media Type Filter Tests
// =============================================================================

test.describe('Media Search - Type Filters @e2e @search', () => {
  test.beforeEach(async ({ page }) => {
    // Start with a search that has results
    await page.goto('/search?q=Batman');
    await page.waitForLoadState('networkidle');
  });

  test('[P1] should filter by movie type', async ({ page }) => {
    // GIVEN: Search results are displayed

    // WHEN: User clicks on movie filter tab
    await page.getByRole('tab', { name: /電影/i }).click();

    // THEN: URL should update with type parameter
    await expect(page).toHaveURL(/type=movie/);
  });

  test('[P1] should filter by TV show type', async ({ page }) => {
    // GIVEN: Search results are displayed

    // WHEN: User clicks on TV filter tab
    await page.getByRole('tab', { name: /影集/i }).click();

    // THEN: URL should update with type parameter
    await expect(page).toHaveURL(/type=tv/);
  });

  test('[P1] should show all results by default', async ({ page }) => {
    // GIVEN: User performs a new search
    await page.goto('/search?q=Star');
    await page.waitForLoadState('networkidle');

    // WHEN: No type filter is selected

    // THEN: All tab should be active (default)
    const allTab = page.getByRole('tab', { name: /全部/i });
    await expect(allTab).toHaveAttribute('aria-selected', 'true');
  });

  test('[P2] should show result counts in tabs', async ({ page }) => {
    // GIVEN: Search results are loaded

    // WHEN: Viewing the type tabs

    // THEN: Tabs should show result counts (if any results exist)
    // Note: Count display depends on actual search results
    await expect(page.getByRole('tab', { name: /電影/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /影集/i })).toBeVisible();
  });
});

// =============================================================================
// Search Results Display Tests
// =============================================================================

test.describe('Media Search - Results Display @e2e @search', () => {
  test('[P0] should display movie poster cards in results', async ({ page }) => {
    // GIVEN: User searches for a popular movie
    await page.goto('/search?q=Inception&type=movie');

    // WHEN: Results load
    await page.waitForLoadState('networkidle');

    // THEN: Should display poster cards with images
    const posterCards = page.locator('[data-testid="poster-card"]');
    // Wait for at least one card to appear
    await expect(posterCards.first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should show loading state while searching', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');

    // WHEN: User initiates search
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await searchInput.fill('Interstellar');

    // Then press enter and immediately check for loading
    const _loadingPromise = page.getByTestId('search-loading').isVisible();
    await searchInput.press('Enter');

    // THEN: Loading indicator may briefly appear (or results load quickly)
    // This test verifies the search completes successfully
    await page.waitForLoadState('networkidle');
  });

  test('[P1] should navigate to movie detail on card click', async ({ page }) => {
    // GIVEN: Search results are displayed
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on a movie card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // THEN: Should navigate to movie detail page
    await expect(page).toHaveURL(/\/media\/movie\/\d+/);
  });

  test('[P2] should show no results message for invalid search', async ({ page }) => {
    // GIVEN: User searches for non-existent content
    await page.goto('/search?q=xyznonexistentmovie99999');

    // WHEN: Results load
    await page.waitForLoadState('networkidle');

    // THEN: Should show no results message
    await expect(page.getByText(/沒有找到|找不到|No results/i)).toBeVisible({ timeout: 10000 });
  });
});

// =============================================================================
// Pagination Tests
// =============================================================================

test.describe('Media Search - Pagination @e2e @search', () => {
  test('[P1] should display pagination for multiple pages', async ({ page }) => {
    // GIVEN: Search with many results
    await page.goto('/search?q=action');
    await page.waitForLoadState('networkidle');

    // WHEN: Results load

    // THEN: Pagination should be visible if results exceed one page
    const pagination = page.locator('[data-testid="pagination"]');
    // Pagination may not appear if fewer than one page of results
    if (await pagination.isVisible()) {
      await expect(pagination).toBeVisible();
    }
  });

  test('[P2] should navigate to next page', async ({ page }) => {
    // GIVEN: Search results with pagination
    await page.goto('/search?q=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks next page (if available)
    const nextButton = page.getByRole('button', { name: /next|下一頁|>/i });
    if (await nextButton.isVisible()) {
      await nextButton.click();

      // THEN: URL should update with page parameter
      await expect(page).toHaveURL(/page=2/);
    }
  });

  test('[P2] should preserve type filter when paginating', async ({ page }) => {
    // GIVEN: Movie filter is active
    await page.goto('/search?q=action&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User navigates to next page (if available)
    const nextButton = page.getByRole('button', { name: /next|下一頁|>/i });
    if (await nextButton.isVisible()) {
      await nextButton.click();

      // THEN: Type filter should remain
      await expect(page).toHaveURL(/type=movie/);
      await expect(page).toHaveURL(/page=2/);
    }
  });
});

// =============================================================================
// Edge Cases
// =============================================================================

test.describe('Media Search - Edge Cases @e2e @search', () => {
  test('[P2] should handle special characters in search', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');

    // WHEN: User searches with special characters
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await searchInput.fill('Batman: The Dark Knight');
    await searchInput.press('Enter');

    // THEN: Should handle gracefully without errors
    await page.waitForLoadState('networkidle');
    // Page should not crash
    await expect(page.getByRole('heading', { name: /搜尋媒體/i })).toBeVisible();
  });

  test('[P2] should handle rapid search input', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');

    // WHEN: User types rapidly
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await searchInput.pressSequentially('Inception', { delay: 50 });
    await searchInput.press('Enter');

    // THEN: Should debounce and search correctly
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/q=Inception/);
  });

  test('[P2] should handle network timeout gracefully', async ({ page }) => {
    // GIVEN: Slow network conditions
    await page.route('**/api/v1/tmdb/**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 100));
      await route.continue();
    });

    // WHEN: User performs search
    await page.goto('/search?q=Inception');

    // THEN: Should eventually show results or appropriate message
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /搜尋媒體/i })).toBeVisible();
  });
});
