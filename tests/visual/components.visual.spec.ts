/**
 * Per-component visual-regression baselines — story 19-4.
 *
 * Runs only under the Playwright `visual` project (`playwright.config.ts`):
 *   pnpm run test:visual              # verify against committed baselines
 *   pnpm run test:visual:update       # (re)generate baselines — see tests/visual/README.md
 *
 * Drives the dev-only gallery route (`apps/web/src/routes/test/gallery.tsx`): for every
 * `<section data-gallery-id>` and each owned `<div data-gallery-state>` (default/hover/focus),
 * applies the state and asserts `toHaveScreenshot(['components', <id>, <state>.png])`. The
 * worklist is derived from the live DOM, so adding a component = adding a fixture entry in
 * `gallery.fixtures.tsx` — nothing changes here.
 *
 * Snapshot tolerance, reduced-motion, viewport, dark colour scheme: configured on the `visual`
 * project + `expect.toHaveScreenshot` in `playwright.config.ts`.
 *
 * @tags @visual @story-19-4
 */
import { test, expect, type Locator } from '@playwright/test';

const FOCUSABLE =
  ':is(a[href], button, input, select, textarea, [tabindex]):not([tabindex="-1"]):not([disabled])';

test.describe('@visual @story-19-4 component visual baselines', () => {
  test.beforeEach(async ({ page }) => {
    // `__root.tsx` redirects to /setup when setup status reports needsSetup — pin it so the
    // gallery is reachable regardless of the dev backend's data-dir state.
    await page.route('**/api/v1/setup/status', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ needsSetup: false }),
      })
    );
    await page.goto('/test/gallery');
    await page.waitForSelector('[data-testid="component-gallery-page"]', { state: 'visible' });
    // Let lazy chunks + fonts settle before any capture.
    await page.waitForLoadState('networkidle');
  });

  test('every gallery component matches its baseline (default / hover / focus)', async ({
    page,
  }) => {
    const sections = page.locator('section[data-gallery-id]');
    const count = await sections.count();
    expect(count, 'gallery rendered at least one component section').toBeGreaterThan(0);

    for (let i = 0; i < count; i++) {
      const section = sections.nth(i);
      const id = (await section.getAttribute('data-gallery-id')) as string;
      const stateDivs = section.locator('[data-gallery-state]');
      const stateCount = await stateDivs.count();

      for (let j = 0; j < stateCount; j++) {
        const stateDiv = stateDivs.nth(j);
        const state = (await stateDiv.getAttribute('data-gallery-state')) as string;

        // Skip fixtures that rendered the error placeholder — an error state is not a valid baseline.
        if ((await stateDiv.locator('[data-gallery-error]').count()) > 0) {
          test.info().annotations.push({
            type: 'gallery-skip',
            description: `${id}:${state} (fixture error)`,
          });
          continue;
        }

        await stateDiv.scrollIntoViewIfNeeded();

        if (state === 'hover') {
          await stateDiv.hover();
        } else if (state === 'focus') {
          const focusable: Locator = stateDiv.locator(FOCUSABLE).first();
          if ((await focusable.count()) > 0) {
            await focusable.focus();
          } else {
            // No focusable descendant — focus state is identical to default; still capture it.
            await stateDiv.evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
          }
        }

        await expect(stateDiv).toHaveScreenshot(['components', id, `${state}.png`]);
      }
    }
  });
});
