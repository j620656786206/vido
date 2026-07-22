/**
 * Per-component visual-regression baselines — story 19-4 (+ 19-4b Task 0 / Task 4 harness fixes).
 *
 * Runs only under the Playwright `visual` project (`playwright.config.ts`):
 *   pnpm run test:visual              # verify against committed baselines
 *   pnpm run test:visual:update       # (re)generate baselines — see tests/visual/README.md
 *
 * Drives the dev-only gallery route (`apps/web/src/routes/test/gallery.tsx`). The spec
 * navigates first to `/test/gallery?manifest=1` to discover fixture IDs, then to
 * `/test/gallery?fixture=<id>` per fixture — each snapshot happens with ONLY that
 * fixture mounted, so `fixed inset-0` overlay components (ui/Dialog, ui/SidePanel,
 * 10 custom dialogs) can no longer intercept pointer events globally and break every
 * other fixture's hover/focus. For every `<section data-gallery-id>` and each owned
 * `<div data-gallery-state>` (`default` / `hover` / `focus` / `open`), the spec applies
 * the state and asserts `toHaveScreenshot(['components', <id>, <state>.png])`. The
 * worklist is derived from the live manifest DOM, so adding a component = adding a
 * fixture entry in `-gallery.fixtures.tsx`.
 *
 * **19-4b Task 0 (Sally 2026-05-12 follow-ups, all three landed):**
 *   - **Fix A (`:focus-visible`):** each state div is preceded by a hidden
 *     `[data-gallery-sentinel="pre"]` focusable button in the gallery route. For
 *     `focus` state the spec focuses that sentinel and presses `Tab` — Chromium then
 *     flags input modality as keyboard so the subsequent focus inside the state div
 *     triggers `:focus-visible` rules. Programmatic `locator.focus()` did not.
 *   - **Fix B (router-state-dependent fixtures):** the gallery route wraps fixtures
 *     declaring `routePath` (e.g. `dashboard-recent-media-panel` → `/library`) in a
 *     nested memory `RouterProvider`. `useRouterState()` inside the component reports
 *     the stub path. No spec change needed — this is gallery-side.
 *   - **Fix C (interactive `open` state):** fixtures setting `openTrigger` get an
 *     extra `<div data-gallery-state="open" data-gallery-open-trigger="<selector>">`
 *     block; the spec clicks that selector inside the state div before screenshotting,
 *     capturing e.g. `library/SortSelector`'s open `SortDropdown 955EZ` panel.
 *
 * **19-4b Task 4 (single-fixture-per-page isolation):**
 *   The original DOM-driven discovery iterated all fixtures inside one page load.
 *   That broke when fixtures with `fixed inset-0` overlays were added in Task 2/3
 *   (ui-dialog Radix portal, ui-side-panel, 10 custom dialogs) — their overlays
 *   stacked on the viewport and intercepted hover/focus on every neighbour. Task 4
 *   keeps the DOM-driven contract but isolates each fixture to its own navigation:
 *     1. `goto('/test/gallery?manifest=1')` → scrape `[data-gallery-id]` ids.
 *     2. For each id: `goto('/test/gallery?fixture=<id>')` → screenshot all states.
 *   Per-fixture isolation is the right level of granularity: gallery DOM structure
 *   stays identical (same state-div CSS), so existing baselines remain byte-stable.
 *
 * Snapshot tolerance, reduced-motion, viewport, dark colour scheme: configured on the
 * `visual` project + `expect.toHaveScreenshot` in `playwright.config.ts`.
 *
 * @tags @visual @story-19-4 @story-19-4b
 */
import { test, expect, type Locator, type Page } from '@playwright/test';
import { withFixedClock } from './clock-mock';

const FOCUSABLE =
  ':is(a[href], button, input, select, textarea, [tabindex]):not([tabindex="-1"]):not([disabled])';

async function stubSetupStatus(page: Page) {
  // `__root.tsx` redirects to /setup when setup status reports needsSetup — pin it so the
  // gallery is reachable regardless of the dev backend's data-dir state.
  await page.route('**/api/v1/setup/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ needsSetup: false }),
    })
  );
}

async function abortTmdbImages(page: Page) {
  // 19-4b Task 4 (burn-in flake fix): the Task 3 Q-bucket fixtures (homepage-explore-block,
  // dashboard-recent-media-panel, homepage-hero-banner, ...) reference TMDB-hosted poster
  // paths (e.g. `/p-101.jpg`) which the PosterCard composes into
  // `https://image.tmdb.org/t/p/w342/...`. The fetch races the screenshot — slow runs paint
  // the loading placeholder, fast runs paint the error fallback. Abort the fetch immediately
  // so every run paints the deterministic error-fallback icon.
  await page.route('**://image.tmdb.org/**', (route) => route.abort());
}

test.describe('@visual @story-19-4 component visual baselines', () => {
  test.beforeEach(async ({ page }) => {
    await stubSetupStatus(page);
    await abortTmdbImages(page);
  });

  test('every gallery component matches its baseline (default / hover / focus / open)', async ({
    page,
  }) => {
    // 19-4b Task 4: per-fixture navigation × ~123 fixtures × ~1-2s each blows past
    // the default 60s test budget. Extend to 10 min — covers the full bulk-fill set
    // with comfortable headroom. The visual project still runs serially (1 worker)
    // so this does not increase the wallclock for parallel CI shards.
    test.setTimeout(10 * 60 * 1000);

    // 19-4b Task 4: discover fixture ids from the manifest endpoint (no components mounted).
    await page.goto('/test/gallery?manifest=1');
    await page.waitForSelector('[data-testid="component-gallery-manifest"]', { state: 'visible' });
    // 19-9 AC #4: harvest each fixture's optional clockTime alongside its id so we can
    // pin the in-page wall clock (Rule 23) BEFORE the per-fixture goto. Fixtures
    // without clockTime fall through the helper-not-called branch — backward-compat.
    const fixtures = await page.locator('li[data-gallery-id]').evaluateAll((els) =>
      els
        .map((el) => ({
          id: el.getAttribute('data-gallery-id'),
          clockTime: el.getAttribute('data-gallery-clock-time'),
        }))
        .filter(
          (f): f is { id: string; clockTime: string | null } =>
            typeof f.id === 'string' && f.id.length > 0
        )
    );
    expect(fixtures.length, 'manifest returned at least one fixture id').toBeGreaterThan(0);

    for (const { id, clockTime } of fixtures) {
      // 19-9 AC #4: Rule 23 clock-mock. Install BEFORE goto so `page.clock` init
      // scripts run before any time-dependent JS in the fixture page evaluates.
      if (clockTime) {
        await withFixedClock(page, clockTime);
      }
      // 19-4b Task 4: per-fixture isolated page load — only one fixture is mounted,
      // so `fixed inset-0` overlay components can no longer block neighbour fixtures.
      // `waitUntil: 'domcontentloaded'` is faster than the default `'load'` — Vite
      // continues fetching chunks past DOMContentLoaded for components with deep
      // import graphs; we don't need them all loaded before our waitForSelector
      // detects the gallery page render. 60s budget tolerates slow Vite recompiles
      // after many sequential navigations within one test (~123 fixtures total).
      await page.goto(`/test/gallery?fixture=${encodeURIComponent(id)}`, {
        timeout: 60_000,
        waitUntil: 'domcontentloaded',
      });
      await page.waitForSelector('[data-testid="component-gallery-page"]', {
        state: 'visible',
        timeout: 30_000,
      });

      const section = page.locator(`section[data-gallery-id="${id}"]`);
      // The manifest is derived from the same `GALLERY_FIXTURES` array that the
      // `?fixture=<id>` filter applies to, so the section is always rendered for
      // any id surfaced by the manifest. Wait for visibility to ensure the section
      // has committed before screenshotting; `waitForLoadState('networkidle')` is
      // NOT usable — app-shell SSE / long-poll never reach idle.
      await section.waitFor({ state: 'visible', timeout: 30_000 });
      // Settle web fonts before screenshot — deterministic and short-lived.
      await page.evaluate(() => document.fonts.ready);

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

        // 19-4b Task 4: detect overlay/portal fixtures whose visible content escapes
        // the state-div (Radix `Dialog.Portal` → document.body; `position: fixed`
        // children → removed from inline-block flow → state-div is 0×0). For those
        // we capture a viewport screenshot so the overlay paint is still recorded.
        const bbox = await stateDiv.boundingBox();
        const isZeroSize = !bbox || bbox.width < 4 || bbox.height < 4;

        if (!isZeroSize) {
          await stateDiv.scrollIntoViewIfNeeded();
        }

        if (state === 'hover' && !isZeroSize) {
          await stateDiv.hover();
        } else if (state === 'hover' && isZeroSize) {
          // No flow content to hover — keep the default-state viewport screenshot.
          test.info().annotations.push({
            type: 'gallery-skip-interaction',
            description: `${id}:hover skipped (zero-size state div — overlay/portal fixture)`,
          });
        } else if (state === 'focus') {
          // 19-4b Task 0 Fix A: focus a hidden sentinel before the state div, then
          // press Tab to enter it. Chromium flags the resulting focus as keyboard
          // modality so `:focus-visible` rules paint correctly. Programmatic
          // `locator.focus()` does not trigger `:focus-visible`.
          // The gallery route ALWAYS renders the sentinel before each state-div,
          // so the sentinel-missing branch was dead code (dropped in 19-4b Task 6
          // CR fix). Zero-size fixtures fall through to the viewport screenshot.
          const focusable: Locator = stateDiv.locator(FOCUSABLE).first();
          if ((await focusable.count()) > 0 && !isZeroSize) {
            const sentinel: Locator = stateDiv.locator(
              'xpath=preceding-sibling::*[@data-gallery-sentinel="pre"][1]'
            );
            await sentinel.focus();
            await page.keyboard.press('Tab');
          } else if (!isZeroSize) {
            // No focusable descendant — focus state is identical to default; still capture it.
            await stateDiv.evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
          }
          // Zero-size fixtures: focus state is captured as a viewport screenshot identical to default.
        } else if (state === 'open' && !isZeroSize) {
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

        if (isZeroSize) {
          // 19-4b Task 4: overlay/portal fixtures — capture viewport instead so the
          // dialog/sidepanel paint is recorded. Page screenshot includes the app
          // shell, but the overlay is the visually-dominant content (centered
          // dialog box + dark backdrop). This is the documented capture strategy
          // for the 12 `fixed inset-0` / Radix-portal fixtures (see story 19-4b).
          await expect(page).toHaveScreenshot(['components', id, `${state}.png`]);
        } else {
          await expect(stateDiv).toHaveScreenshot(['components', id, `${state}.png`]);
        }
      }
    }
  });
});
