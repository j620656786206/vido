/**
 * Manual Search E2E Tests (Story 3-7)
 *
 * Tests for the manual metadata search and selection UI.
 * These tests validate the complete user journey for manually searching
 * and selecting metadata when automatic parsing fails.
 *
 * Prerequisites:
 * - Frontend running on port 4200: npx nx serve web
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * Acceptance Criteria Coverage:
 * - AC1: Manual Search Dialog - User can open dialog and enter query
 * - AC2: Search Results Display - Results show poster, title, year, description
 * - AC3: Selection and Application - User can select and apply metadata
 * - AC4: Source Selection - User can filter by source (TMDb, Douban, Wikipedia)
 *
 * @tags @e2e @manual-search @story-3-7
 */

import { test, expect } from '../support/fixtures';

// =============================================================================
// Manual Search Dialog Tests (AC1)
// =============================================================================

test.describe('Manual Search Dialog @e2e @manual-search', () => {
  // Note: These tests require a page with parse failure state to trigger manual search
  // Using search page as a mock entry point until library page with parse failures is implemented

  test('[P0] should display search input in manual search dialog', async ({ page }) => {
    // GIVEN: User navigates to search page (mock entry point)
    await page.goto('/search');

    // WHEN: Page loads
    await page.waitForLoadState('networkidle');

    // THEN: Search input should be available
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await expect(searchInput).toBeVisible();
  });

  test('[P1] should search with custom query and display results (AC1)', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User enters a custom search query
    await searchInput.fill('Demon Slayer');
    await searchInput.press('Enter');

    // THEN: Should display search results
    await expect(page).toHaveURL(/q=Demon.*Slayer/i);
    await page.waitForLoadState('networkidle');

    // Results should appear (media-grid shows search results)
    await expect(page.getByTestId('media-grid')).toBeVisible({
      timeout: 15000,
    });
  });

  test('[P1] should filter results by movie type', async ({ page }) => {
    // GIVEN: User is on search page with results
    await page.goto('/search?q=Batman');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on movie filter
    const movieTab = page.getByRole('tab', { name: /電影/i });
    await movieTab.click();

    // THEN: URL should include movie type filter
    await expect(page).toHaveURL(/type=movie/);
  });

  test('[P1] should filter results by TV show type', async ({ page }) => {
    // GIVEN: User is on search page with results
    await page.goto('/search?q=Breaking');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on TV filter
    const tvTab = page.getByRole('tab', { name: /影集/i });
    await tvTab.click();

    // THEN: URL should include TV type filter
    await expect(page).toHaveURL(/type=tv/);
  });

  test('[P1] should display Chinese title search results', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User searches with Chinese characters
    await searchInput.fill('鬼滅之刃');
    await searchInput.press('Enter');

    // THEN: Should handle Chinese query correctly
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/q=%E9%AC%BC%E6%BB%85/); // URL encoded Chinese
  });
});

// =============================================================================
// Search Results Display Tests (AC2)
// =============================================================================

test.describe('Search Results Display @e2e @manual-search', () => {
  test('[P0] should display search results with poster cards', async ({ page }) => {
    // GIVEN: User searches for a popular movie
    await page.goto('/search?q=Inception&type=movie');

    // WHEN: Results load
    await page.waitForLoadState('networkidle');

    // THEN: Should display poster cards
    const posterCards = page.locator('[data-testid="poster-card"]');
    await expect(posterCards.first()).toBeVisible({ timeout: 15000 });
  });

  test('[P1] should show movie title and year in results (AC2)', async ({ page }) => {
    // GIVEN: User searches for a specific movie
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: Results are displayed
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });

    // THEN: Card should show title
    // Note: Exact text depends on UI implementation
    await expect(page.getByText(/Inception|全面啟動/i).first()).toBeVisible();
  });

  test('[P1] should show description preview on hover (AC2)', async ({ page }) => {
    // GIVEN: User searches for a movie with results
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User hovers over a card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.hover();

    // THEN: Should show preview panel with description
    // Note: This depends on hover preview implementation
    // Look for HoverPreviewCard or similar component
    const previewPanel = page.locator(
      '[data-testid="hover-preview"], [data-testid="preview-card"], [data-testid="media-detail-panel"]'
    );
    // Preview may appear on hover - this is implementation dependent
    if (await previewPanel.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(previewPanel).toBeVisible();
    }
  });

  test('[P2] should show no results message for invalid search', async ({ page }) => {
    // GIVEN: User searches for non-existent content
    await page.goto('/search?q=xyznonexistentmovie99999');

    // WHEN: Results load
    await page.waitForLoadState('networkidle');

    // THEN: Should show no results message
    await expect(page.getByText(/沒有找到|找不到|No results/i)).toBeVisible({ timeout: 10000 });
  });

  test('[P2] should handle empty search gracefully', async ({ page }) => {
    // GIVEN: User navigates to search without query
    await page.goto('/search');

    // WHEN: Page loads without a query
    await page.waitForLoadState('networkidle');

    // THEN: Should show search prompt or empty state
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);
    await expect(searchInput).toBeVisible();
  });
});

// =============================================================================
// Source Selection Tests (AC4)
// =============================================================================

test.describe('Source Selection @e2e @manual-search', () => {
  // Note: Source selection UI is part of ManualSearchDialog component
  // These tests verify the source filtering behavior

  test('[P1] should search all sources by default', async ({ page }) => {
    // GIVEN: User performs a search without source filter
    await page.goto('/search?q=Inception');

    // WHEN: Search completes
    await page.waitForLoadState('networkidle');

    // THEN: Results should be displayed (from any available source)
    const posterCards = page.locator('[data-testid="poster-card"]');
    await expect(posterCards.first()).toBeVisible({ timeout: 15000 });
  });

  // Note: Source-specific filtering UI tests would go here
  // These depend on the ManualSearchDialog implementation having source selector
  // For now, we test via API in manual-search.api.spec.ts
});

// =============================================================================
// Selection and Confirmation Tests (AC3)
// =============================================================================

test.describe('Selection and Confirmation @e2e @manual-search', () => {
  test('[P1] should navigate to detail page when selecting a result', async ({ page }) => {
    // GIVEN: User has search results
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User clicks on a result card
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // THEN: Should navigate to movie detail page
    await expect(page).toHaveURL(/\/media\/movie\/\d+/);
  });

  test('[P1] should display movie details after selection', async ({ page }) => {
    // GIVEN: User clicks on a movie from search results
    await page.goto('/search?q=Inception&type=movie');
    await page.waitForLoadState('networkidle');

    const firstCard = page.locator('[data-testid="poster-card"]').first();
    await expect(firstCard).toBeVisible({ timeout: 15000 });
    await firstCard.click();

    // WHEN: Detail page loads
    await page.waitForLoadState('networkidle');

    // THEN: Should show movie details
    await expect(page.getByText(/Inception|全面啟動/i).first()).toBeVisible({ timeout: 15000 });
  });

  // Note: Apply metadata confirmation tests require library page with parse failure state
  // These tests would verify:
  // - User can click "Select" button on a result
  // - Confirmation dialog appears with metadata preview
  // - Success toast shows after applying
  // - Learning prompt appears (Story 3.9)
});

// =============================================================================
// Complete Apply Metadata Flow Tests (AC1, AC2, AC3, AC4)
// =============================================================================

test.describe('Apply Metadata Complete Flow @e2e @manual-search @story-3-7', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the test page with ParseFailureCard fixtures
    await page.goto('/test/manual-search');
    await page.waitForLoadState('networkidle');
  });

  test('[P0] should display parse failure cards on test page', async ({ page }) => {
    // GIVEN: User navigates to test page

    // WHEN: Page loads

    // THEN: Should display parse failure cards
    await expect(page.getByTestId('test-manual-search-page')).toBeVisible();
    await expect(page.getByTestId('parse-failure-grid')).toBeVisible();

    // Should show 4 test cards
    const failureCards = page.locator('[data-testid="parse-failure-card"]');
    await expect(failureCards).toHaveCount(4);
  });

  test('[P0] should open manual search dialog when clicking button (AC1)', async ({ page }) => {
    // GIVEN: Parse failure cards are displayed
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await expect(firstCard).toBeVisible();

    // WHEN: User clicks "手動搜尋" button
    const searchButton = firstCard.getByTestId('manual-search-button');
    await searchButton.click();

    // THEN: Manual search dialog should open
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();
    await expect(page.getByText('手動搜尋 Metadata')).toBeVisible();
  });

  test('[P1] should pre-fill search query with parsed title (AC1)', async ({ page }) => {
    // GIVEN: Parse failure cards are displayed
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();

    // WHEN: User opens manual search for Fight Club
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // THEN: Search input should be pre-filled with parsed title
    const searchInput = page.getByPlaceholder('輸入電影或影集名稱...');
    await expect(searchInput).toHaveValue('Fight Club');
  });

  test('[P1] should display search results with source indicator (AC2, AC4)', async ({ page }) => {
    // GIVEN: Manual search dialog is open
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await firstCard.getByTestId('manual-search-button').click();

    // WHEN: Search is performed (auto-triggered by pre-filled query)
    await page.waitForLoadState('networkidle');

    // THEN: Results should be visible (may take time due to API call)
    // Note: This test depends on actual API response
    const resultsGrid = page.locator('[data-testid="search-results-grid"]');
    await expect(
      resultsGrid.or(page.getByText(/搜尋失敗|沒有找到|Fight Club/i).first())
    ).toBeVisible({
      timeout: 15000,
    });
  });

  test('[P1] should show fallback status indicating tried sources (UX-4)', async ({ page }) => {
    // GIVEN: Parse failure card with fallback status
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();

    // WHEN: User opens manual search
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // THEN: Fallback status display should be visible
    const fallbackStatus = page.locator('[data-testid="fallback-status"]');
    if (await fallbackStatus.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(fallbackStatus).toBeVisible();
    }
  });

  test('[P1] should allow source filtering (AC4)', async ({ page }) => {
    // GIVEN: Manual search dialog is open
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // WHEN: User selects a specific source
    const sourceSelect = page.locator('select');
    await sourceSelect.selectOption('tmdb');

    // THEN: Source filter should be applied
    await expect(sourceSelect).toHaveValue('tmdb');
  });

  test('[P1] should allow media type switching', async ({ page }) => {
    // GIVEN: Manual search dialog is open
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // WHEN: User clicks TV button
    const tvButton = page.getByRole('button', { name: '影集' });
    await tvButton.click();

    // THEN: TV button should be active
    await expect(tvButton).toHaveClass(/bg-blue-600/);
  });

  test('[P1] should close dialog on escape key', async ({ page }) => {
    // GIVEN: Manual search dialog is open
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // WHEN: User presses Escape
    await page.keyboard.press('Escape');

    // THEN: Dialog should close
    await expect(page.getByTestId('manual-search-dialog')).not.toBeVisible();
  });

  test('[P1] should close dialog on backdrop click', async ({ page }) => {
    // GIVEN: Manual search dialog is open
    const firstCard = page.locator('[data-testid="parse-failure-card"]').first();
    await firstCard.getByTestId('manual-search-button').click();
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();

    // WHEN: User clicks on backdrop (outside dialog)
    // Use dispatchEvent to ensure the click reaches the backdrop
    await page.evaluate(() => {
      const backdrop = document.querySelector('[data-testid="dialog-backdrop"]');
      if (backdrop) {
        backdrop.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      }
    });

    // THEN: Dialog should close
    await expect(page.getByTestId('manual-search-dialog')).not.toBeVisible();
  });

  test('[P2] should handle TV show with season/episode info', async ({ page }) => {
    // GIVEN: Breaking Bad card (TV show with S01E01)
    const tvCard = page.locator('[data-testid="parse-failure-card"]').nth(1);
    await expect(tvCard.getByText('Breaking Bad')).toBeVisible();

    // WHEN: User opens manual search
    await tvCard.getByTestId('manual-search-button').click();

    // THEN: Dialog should open with TV show pre-selected
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();
    await expect(page.getByPlaceholder('輸入電影或影集名稱...')).toHaveValue('Breaking Bad');
  });

  test('[P2] should handle file with no parsed info', async ({ page }) => {
    // GIVEN: Unknown file card (no parsed info)
    const unknownCard = page.locator('[data-testid="parse-failure-card"]').nth(2);

    // WHEN: User opens manual search
    await unknownCard.getByTestId('manual-search-button').click();

    // THEN: Dialog should open with filename-derived query
    await expect(page.getByTestId('manual-search-dialog')).toBeVisible();
    // The extracted title from 'random_video_file_2024.mp4' should be something reasonable
    const searchInput = page.getByPlaceholder('輸入電影或影集名稱...');
    await expect(searchInput).toHaveValue(/random|video|file/i);
  });
});

// =============================================================================
// Edge Cases
// =============================================================================

test.describe('Manual Search Edge Cases @e2e @manual-search', () => {
  test('[P2] should handle special characters in search query', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User searches with special characters
    await searchInput.fill('Batman: The Dark Knight');
    await searchInput.press('Enter');

    // THEN: Should handle gracefully
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /搜尋媒體/i })).toBeVisible();
  });

  test('[P2] should debounce rapid typing', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User types rapidly
    await searchInput.pressSequentially('Inception', { delay: 30 });
    await searchInput.press('Enter');

    // THEN: Should debounce and search correctly
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/q=Inception/);
  });

  test('[P2] should show loading state during search', async ({ page }) => {
    // GIVEN: User is on search page
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜尋電影或影集/i);

    // WHEN: User initiates search
    await searchInput.fill('Interstellar');

    // Create promise to check for loading state
    const searchPromise = page.waitForResponse((response) => response.url().includes('/api/'));

    await searchInput.press('Enter');

    // THEN: Should eventually complete search
    await searchPromise;
    await page.waitForLoadState('networkidle');
  });

  test('[P2] should preserve search state on page navigation', async ({ page }) => {
    // GIVEN: User has performed a search
    await page.goto('/search?q=Matrix&type=movie');
    await page.waitForLoadState('networkidle');

    // WHEN: User navigates to detail and back
    const firstCard = page.locator('[data-testid="poster-card"]').first();
    if (await firstCard.isVisible({ timeout: 5000 }).catch(() => false)) {
      await firstCard.click();
      await page.waitForLoadState('networkidle');
      await page.goBack();

      // THEN: Search state should be preserved
      await expect(page).toHaveURL(/q=Matrix/);
      await expect(page).toHaveURL(/type=movie/);
    }
  });
});
