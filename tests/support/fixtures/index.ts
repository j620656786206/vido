/**
 * Vido Test Fixtures
 *
 * Composable fixture system using Playwright's mergeTests pattern.
 * Each fixture provides one isolated capability with auto-cleanup.
 *
 * Usage:
 *   import { test, expect } from '../support/fixtures';
 *
 * @see https://playwright.dev/docs/test-fixtures
 */

import { test as base, expect, type Route } from '@playwright/test';
import { apiHelpers, type ApiHelpers } from '../helpers/api-helpers';

// =============================================================================
// Type Definitions
// =============================================================================

type TestFixtures = {
  /**
   * API helpers for direct backend interaction
   * Use for test setup/teardown instead of UI actions
   */
  api: ApiHelpers;
};

// =============================================================================
// Extended Test with Fixtures
// =============================================================================

/**
 * Main test export with all fixtures composed
 *
 * Available fixtures:
 * - page: Playwright Page object
 * - request: Playwright APIRequestContext
 * - api: Custom API helpers for Vido backend
 */
export const test = base.extend<TestFixtures>({
  api: async ({ request }, use) => {
    const helpers = apiHelpers(request);
    await use(helpers);
  },

  // ux3-cutover-1 INTERIM (delete with the legacy shell in ux3-cutover-3): the
  // backend now forces `new_shell_enabled` ON at startup, but most specs in this
  // suite assert the LEGACY shell. Pin the flag OFF at the shared-page chokepoint
  // (localStorage cache seed + settings-route intercept) so they keep rendering
  // legacy until cutover-3 rewrites/removes them. v2 specs are unaffected: their
  // own addInitScript runs after this one and their later-registered route wins.
  page: async ({ page }, use) => {
    await page.addInitScript(() => {
      try {
        localStorage.setItem('vido:flag:new_shell_enabled', 'false');
      } catch {
        // ignore — storage unavailable
      }
    });
    await page.route('**/api/v1/settings/new_shell_enabled', (route: Route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { key: 'new_shell_enabled', value: 'false' },
        }),
      })
    );
    await use(page);
  },
});

export { expect };

// =============================================================================
// Test Annotations
// =============================================================================

/**
 * Tag tests for selective execution
 *
 * Usage:
 *   test('feature @smoke', async ({ page }) => { ... });
 *
 * Run:
 *   npx playwright test --grep @smoke
 *   npx playwright test --grep-invert @slow
 */
export const tags = {
  smoke: '@smoke',
  regression: '@regression',
  slow: '@slow',
  api: '@api',
  auth: '@auth',
} as const;
