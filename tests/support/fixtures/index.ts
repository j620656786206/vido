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

import { test as base, expect } from '@playwright/test';
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
// ux3-cutover-3: the cutover-1 legacy-pin page fixture is deleted — the suite
// now runs against the real v2 shell (new_shell_enabled is forced ON by the
// backend, and the migrated routes no longer carry a legacy branch).
export const test = base.extend<TestFixtures>({
  api: async ({ request }, use) => {
    const helpers = apiHelpers(request);
    await use(helpers);
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
