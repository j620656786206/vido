/**
 * Home Page E2E Tests
 *
 * Tests for the Vido home page and basic navigation.
 * These are smoke tests to verify the application loads correctly.
 *
 * @tags @smoke
 */

import { test, expect } from '../support/fixtures';

test.describe('Home Page @smoke', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should load the home page without errors', async ({ page }) => {
    // Verify page loads (basic smoke test)
    // Update title expectation when Vido UI is implemented
    await expect(page).toHaveTitle(/.+/); // Any non-empty title
  });

  test('should render the root component', async ({ page }) => {
    // Verify the React app mounts successfully
    await expect(page.locator('body')).not.toBeEmpty();
  });

  // TODO: Enable these tests when Vido UI is implemented
  test.skip('should display main navigation', async ({ page }) => {
    await expect(page.getByRole('navigation')).toBeVisible();
  });

  test.skip('should have search functionality visible', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search|搜尋/i);
    await expect(searchInput).toBeVisible();
  });
});

test.describe('Navigation @smoke', () => {
  test.skip('should navigate between main sections', async ({ page }) => {
    await page.goto('/');
    // TODO: Implement when routes are added
    // await page.click('[data-testid="nav-library"]');
    // await expect(page).toHaveURL(/.*library/);
  });
});
