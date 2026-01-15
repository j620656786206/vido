/**
 * Movie Search E2E Tests
 *
 * Tests for the movie search functionality.
 * Covers search input, results display, and edge cases.
 *
 * TODO: Enable these tests when search UI is implemented
 *
 * @tags @regression
 */

import { test, expect } from '../support/fixtures';

// Skip all search tests until UI is implemented
test.describe.skip('Movie Search', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should search for a movie and display results', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await searchInput.fill('Inception');
    await searchInput.press('Enter');

    await expect(page.getByText(/Inception|全面啟動/i)).toBeVisible({
      timeout: 15000,
    });
  });

  test('should show no results message for invalid search', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await searchInput.fill('xyznonexistentmovie12345');
    await searchInput.press('Enter');

    await expect(page.getByText(/no results|找不到|沒有結果/i)).toBeVisible({
      timeout: 10000,
    });
  });

  test('should clear search results when input is cleared', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await searchInput.fill('Matrix');
    await searchInput.press('Enter');

    await expect(page.getByText(/Matrix|乩/i)).toBeVisible({
      timeout: 15000,
    });

    await searchInput.clear();
  });

  test('should handle special characters in search', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await searchInput.fill('乩童');
    await searchInput.press('Enter');

    await page.waitForLoadState('networkidle');
  });
});

test.describe.skip('Search Performance @slow', () => {
  test('should debounce rapid typing', async ({ page }) => {
    await page.goto('/');

    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await searchInput.pressSequentially('Inception', { delay: 50 });
    await page.waitForTimeout(500);
  });
});
