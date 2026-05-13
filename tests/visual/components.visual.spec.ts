/**
 * Per-component visual-regression baselines — story 19-4 (+ 19-4b Task 0 harness fixes).
 *
 * Runs only under the Playwright `visual` project (`playwright.config.ts`):
 *   pnpm run test:visual              # verify against committed baselines
 *   pnpm run test:visual:update       # (re)generate baselines — see tests/visual/README.md
 *
 * Drives the dev-only gallery route (`apps/web/src/routes/test/gallery.tsx`): for every
 * `<section data-gallery-id>` and each owned `<div data-gallery-state>`
 * (`default` / `hover` / `focus` / `open`), applies the state and asserts
 * `toHaveScreenshot(['components', <id>, <state>.png])`. The worklist is derived from
 * the live DOM, so adding a component = adding a fixture entry in `-gallery.fixtures.tsx`.
 *
 * **19-4b Task 0 (Sally 2026-05-12 follow-ups, all three landed):**
 *   - **Fix A (`:focus-visible`):** each state div is preceded by a hidden
 *     `[data-gallery-sentinel="pre"]` focusable button in the gallery route. For
 *     `focus` state the spec focuses that sentinel and presses `Tab` — Chromium then
 *     flags input modality as keyboard so the subsequent focus inside the state div
 *     triggers `:focus-visible` rules. Programmatic `locator.focus()` did not.
 *   - **Fix B (router-state-dependent fixtures):** the gallery route wraps fixtures
 *     declaring `routePath` (e.g. `shell-tab-navigation` → `/library`) in a nested
 *     memory `RouterProvider`. `useRouterState()` inside the component reports the
 *     stub path. No spec change needed — this is gallery-side.
 *   - **Fix C (interactive `open` state):** fixtures setting `openTrigger` get an
 *     extra `<div data-gallery-state="open" data-gallery-open-trigger="<selector>">`
 *     block; the spec clicks that selector inside the state div before screenshotting,
 *     capturing e.g. `library/SortSelector`'s open `SortDropdown 955EZ` panel.
 *
 * Snapshot tolerance, reduced-motion, viewport, dark colour scheme: configured on the
 * `visual` project + `expect.toHaveScreenshot` in `playwright.config.ts`.
 *
 * @tags @visual @story-19-4 @story-19-4b
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
      const id = await section.getAttribute('data-gallery-id');
      if (!id) {
        test.info().annotations.push({
          type: 'gallery-skip',
          description: `section[${i}] missing data-gallery-id`,
        });
        continue;
      }
      const stateDivs = section.locator('[data-gallery-state]');
      const stateCount = await stateDivs.count();

      for (let j = 0; j < stateCount; j++) {
        const stateDiv = stateDivs.nth(j);
        const state = await stateDiv.getAttribute('data-gallery-state');
        if (!state) {
          test.info().annotations.push({
            type: 'gallery-skip',
            description: `${id}: state div[${j}] missing data-gallery-state`,
          });
          continue;
        }

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
          // 19-4b Task 0 Fix A: focus a hidden sentinel before the state div, then
          // press Tab to enter it. Chromium flags the resulting focus as keyboard
          // modality so `:focus-visible` rules paint correctly. Programmatic
          // `locator.focus()` does not trigger `:focus-visible`.
          const sentinel: Locator = stateDiv.locator(
            'xpath=preceding-sibling::*[@data-gallery-sentinel="pre"][1]'
          );
          const focusable: Locator = stateDiv.locator(FOCUSABLE).first();
          if ((await focusable.count()) > 0 && (await sentinel.count()) > 0) {
            await sentinel.focus();
            await page.keyboard.press('Tab');
          } else if ((await focusable.count()) > 0) {
            // Sentinel missing — fall back to programmatic focus (not :focus-visible).
            await focusable.focus();
          } else {
            // No focusable descendant — focus state is identical to default; still capture it.
            await stateDiv.evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
          }
        } else if (state === 'open') {
          // 19-4b Task 0 Fix C: click the fixture-declared trigger selector inside
          // the state div to open the interactive sub-UI (dropdown / menu / modal).
          // After click, wait for the most common popup role to be visible so the
          // screenshot doesn't race the popup paint. .catch keeps the wait
          // tolerant for openers whose popup doesn't expose a standard role.
          const trigger = await stateDiv.getAttribute('data-gallery-open-trigger');
          if (trigger) {
            await stateDiv.locator(trigger).first().click();
            await stateDiv
              .locator(':is([role="listbox"], [role="menu"], [role="dialog"])')
              .first()
              .waitFor({ state: 'visible', timeout: 1000 })
              .catch(() => {
                /* no role-bearing popup — screenshot whatever opened */
              });
          }
        }

        await expect(stateDiv).toHaveScreenshot(['components', id, `${state}.png`]);
      }
    }
  });
});
