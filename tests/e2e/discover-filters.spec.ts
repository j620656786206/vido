/**
 * Discover / Persistent Filter Chip UI E2E Tests (Story 11-2)
 *
 * Integration-level coverage for the /discover browse flow. Focuses on the
 * cross-layer behaviour that unit/component specs cannot exercise:
 *   - URL <-> filter state <-> discover API <-> chip bar <-> results grid
 *   - Browser back/forward preserving filter state via URL (AC #4)
 *   - Mobile filter bottom sheet apply flow (AC #6)
 *
 * Component-level interactions (chip render/remove logic, panel toggles, bottom
 * sheet draft/apply, URL serialization, param mapping) are already covered by:
 *   - apps/web/src/lib/discoverFilters.spec.ts
 *   - apps/web/src/hooks/useFilterState.spec.tsx
 *   - apps/web/src/hooks/useDiscoverResults.spec.tsx
 *   - apps/web/src/components/search/{FilterChipBar,FilterPanel,FilterBottomSheet}.spec.tsx
 * — intentionally NOT duplicated here (avoid-duplicate-coverage).
 *
 * The TMDb discover endpoints are mocked, so this suite is hermetic and does
 * NOT require TMDB_API_KEY (unlike the api smoke specs).
 *
 * Prerequisites:
 *   - Frontend running on port 4200: npx nx serve web
 *   - Backend running on port 8080 (AppShell shell requests)
 *
 * @tags @e2e @discover @filter @story-11-2 @regression
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

// The router's round-trip-safe stringifier writes a single string value as
// genre="16" (URL-encoded %2216%22) to preserve its type, while a clean
// deep-link genre=16 (number) also round-trips via the route's coercion. Match
// both encodings — the functional contract (filter active) is identical.
const GENRE_16 = /genre=(?:16|%2216%22)/;

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

const movieResults = {
  page: 1,
  results: [
    {
      id: 1,
      title: '電影 A',
      original_title: 'Movie A',
      overview: '',
      release_date: '2024-01-01',
      poster_path: '/posterA.jpg',
      backdrop_path: null,
      vote_average: 8,
      vote_count: 120,
      genre_ids: [16],
    },
    {
      id: 2,
      title: '電影 B',
      original_title: 'Movie B',
      overview: '',
      release_date: '2023-06-01',
      poster_path: '/posterB.jpg',
      backdrop_path: null,
      vote_average: 7.5,
      vote_count: 90,
      genre_ids: [28],
    },
  ],
  total_pages: 1,
  total_results: 2,
};

const tvResults = {
  page: 1,
  results: [
    {
      id: 10,
      name: '劇集 X',
      original_name: 'Show X',
      overview: '',
      first_air_date: '2022-03-01',
      poster_path: '/posterX.jpg',
      backdrop_path: null,
      vote_average: 9,
      vote_count: 300,
      genre_ids: [16],
    },
  ],
  total_pages: 1,
  total_results: 1,
};

/**
 * Stub both discover endpoints and capture every requested URL so the test can
 * assert that the active filters propagate to the backend query (AC #1/#5).
 */
async function stubDiscover(page: import('@playwright/test').Page): Promise<string[]> {
  const requested: string[] = [];
  await page.route(`${ROUTE_API}/tmdb/discover/movies*`, (route: Route) => {
    requested.push(route.request().url());
    return route.fulfill(jsonOk(movieResults));
  });
  await page.route(`${ROUTE_API}/tmdb/discover/tv*`, (route: Route) => {
    requested.push(route.request().url());
    return route.fulfill(jsonOk(tvResults));
  });
  return requested;
}

/**
 * Stub the facet-counts endpoint (Story ux3-discover-facet-aggregation-fe). Keeps
 * the v2 rail hermetic now that it fires a per-facet counts query on load.
 */
async function stubFacetCounts(
  page: import('@playwright/test').Page,
  body: { counts: Record<string, Record<string, number>>; partial: boolean } = {
    counts: {},
    partial: false,
  }
): Promise<void> {
  await page.route(`${ROUTE_API}/tmdb/discover/facet-counts*`, (route: Route) =>
    route.fulfill(jsonOk(body))
  );
}

/** Stub the facet-counts endpoint as unavailable (AC6 fallback path). */
async function stubFacetCountsUnavailable(page: import('@playwright/test').Page): Promise<void> {
  await page.route(`${ROUTE_API}/tmdb/discover/facet-counts*`, (route: Route) =>
    route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'TMDB_TIMEOUT', message: '無法連線到 TMDb API' },
      }),
    })
  );
}

// =============================================================================
// Desktop — apply / persist / remove / clear (AC #1-5)
// =============================================================================

test.describe('Discover Filters — Desktop @e2e @discover', () => {
  test('[P0] selecting a genre adds a chip, updates the URL, and re-queries with the filter (AC #1, #2, #5)', async ({
    page,
  }) => {
    // GIVEN: the discover page has loaded its initial (unfiltered) results
    const requested = await stubDiscover(page);
    await page.goto('/discover');
    await expect(page.locator('[data-testid="poster-card"]').first()).toBeVisible({
      timeout: 15000,
    });

    // WHEN: the user selects the 動畫 genre in the desktop sidebar
    await page.getByTestId('filter-genre-16').click();

    // THEN: a removable chip appears and the URL carries the filter
    await expect(page).toHaveURL(GENRE_16);
    await expect(page.getByTestId('filter-chip-genre-16')).toHaveText(/類型: 動畫/);

    // AND: the discover endpoint was re-queried with the genre param
    await expect.poll(() => requested.some((u) => /[?&]genre=16(&|$)/.test(u))).toBe(true);
  });

  test('[P0] browser back skips intermediate filter toggles (replace semantics, ux3-3-2 AC #5)', async ({
    page,
  }) => {
    // ux3-cutover-3: filter toggles REPLACE the history entry unconditionally —
    // composing genre → rating must not push half-built combos onto the stack,
    // so Back leaves the pre-filter entry instead of stepping through toggles.
    await stubDiscover(page);
    await page.goto('/discover');
    await page.getByTestId('filter-genre-16').click();
    await expect(page).toHaveURL(GENRE_16);

    await page.getByTestId('filter-rating-8').click();
    await expect(page).toHaveURL(/rating_gte=8/);
    await expect(page.getByTestId('filter-chip-rating')).toBeVisible();

    // WHEN: the user presses the browser back button
    await page.goBack();

    // THEN: back does NOT land on the intermediate genre-only state — it
    // returns to the pre-filter /discover entry.
    await expect(page).not.toHaveURL(/rating_gte/);
    await expect(page).not.toHaveURL(/genre=16/);
    await expect(page.getByTestId('filter-chip-rating')).toHaveCount(0);
  });

  test('[P1] removing a chip drops its URL param (AC #2)', async ({ page }) => {
    // GIVEN: a deep-linked page with an active genre filter
    await stubDiscover(page);
    await page.goto('/discover?genre=16');
    await expect(page.getByTestId('filter-chip-genre-16')).toBeVisible();

    // WHEN: the user clicks the chip's remove button
    await page.getByRole('button', { name: '移除類型: 動畫篩選' }).click();

    // THEN: the chip disappears and the param leaves the URL
    await expect(page.getByTestId('filter-chip-genre-16')).toHaveCount(0);
    await expect(page).not.toHaveURL(/genre=16/);
  });

  test('[P1] 清除全部 removes all chips at once when ≥2 filters are active (AC #3)', async ({
    page,
  }) => {
    // GIVEN: a deep-linked page with two active filters
    await stubDiscover(page);
    await page.goto('/discover?genre=16&rating_gte=8');
    await expect(page.getByTestId('clear-all-filters')).toBeVisible();

    // WHEN: the user clicks 清除全部
    await page.getByTestId('clear-all-filters').click();

    // THEN: all chips are removed and the filter params leave the URL
    await expect(page.getByTestId('filter-chip-bar')).toHaveCount(0);
    await expect(page).not.toHaveURL(/genre=16/);
    await expect(page).not.toHaveURL(/rating_gte/);
  });

  test('[P2] a deep link renders the chips for the URL filter state (AC #4)', async ({ page }) => {
    // GIVEN/WHEN: the user lands directly on a filtered discover URL
    await stubDiscover(page);
    await page.goto('/discover?genre=16&region=JP');

    // THEN: chips reflect the URL state and the sidebar marks the genre selected
    await expect(page.getByTestId('filter-chip-genre-16')).toHaveText(/類型: 動畫/);
    await expect(page.getByTestId('filter-chip-region')).toHaveText(/地區: 日本/);
    await expect(page.getByTestId('filter-genre-16')).toHaveAttribute('aria-pressed', 'true');
  });
});

// =============================================================================
// v2 shell — persistent instant filter rail (ux3-3-2 AC #1/#2/#3/#4/#7)
// =============================================================================

/**
 * Enable the v2 shell for these tests: seed the flag's localStorage mirror (the
 * useNewShellEnabled hook reads it as `initialData`, so the rail renders on first
 * paint with no flag→shell flash) AND stub the flag endpoint so the confirming
 * query agrees. No data-dependent self-skip — the discover endpoints are stubbed
 * (the TMDb upstream is external, so mocking is the hermetic equivalent of seeding).
 */
async function enableV2Shell(page: import('@playwright/test').Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      localStorage.setItem('vido:flag:new_shell_enabled', 'true');
    } catch {
      /* ignore */
    }
  });
  await page.route('**/api/v1/settings/new_shell_enabled', (route: Route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { key: 'new_shell_enabled', value: 'true' },
      }),
    })
  );
}

test.describe('Discover v2 rail — Desktop @e2e @discover @ux3-3-2', () => {
  test('[P1] renders the persistent rail with a single live total + instant categorical apply (AC #2/#3/#4/#7)', async ({
    page,
  }) => {
    // GIVEN: the v2 shell is enabled and discover has loaded
    await enableV2Shell(page);
    const requested = await stubDiscover(page);
    await stubFacetCounts(page);
    await page.goto('/discover');
    await expect(page.getByTestId('discover-filter-rail')).toBeVisible({ timeout: 15000 });

    // THEN: the rail shows ONE live total (AC #3 — no per-facet over-fetch)
    await expect(page.getByTestId('discover-rail-count')).toHaveText(/符合 \d+ 部|計算中…/);

    // WHEN: a categorical chip is toggled in the rail
    await page.getByTestId('discover-filter-rail').getByTestId('filter-genre-16').click();

    // THEN: it applies instantly (URL + re-query) and the demoted chip-bar summary reflects it
    await expect(page).toHaveURL(GENRE_16);
    await expect(page.getByTestId('filter-chip-genre-16')).toHaveText(/類型: 動畫/);
    await expect.poll(() => requested.some((u) => /[?&]genre=16(&|$)/.test(u))).toBe(true);
  });

  test('[P2] collapsing the rail hides it and surfaces the re-open control (AC #2)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubDiscover(page);
    await stubFacetCounts(page);
    await page.goto('/discover');
    await expect(page.getByTestId('discover-filter-rail')).toBeVisible({ timeout: 15000 });

    // WHEN: the rail is collapsed
    await page.getByTestId('discover-rail-collapse').click();

    // THEN: the rail is gone (grid reclaims the width) and the 篩選 re-open button shows
    await expect(page.getByTestId('discover-filter-rail')).toHaveCount(0);
    await expect(page.getByTestId('discover-rail-expand')).toBeVisible();
  });

  test('[P1] the rail shows per-chip contextual counts; a 0-result chip is dimmed yet still navigates (AC1/AC2)', async ({
    page,
  }) => {
    // GIVEN: the v2 rail with facet-counts where 動畫(16)=42 and 動作(28)=0 (dead-end)
    await enableV2Shell(page);
    await stubDiscover(page);
    await stubFacetCounts(page, { counts: { genre: { '16': 42, '28': 0 } }, partial: false });
    await page.goto('/discover');
    const rail = page.getByTestId('discover-filter-rail');
    await expect(rail).toBeVisible({ timeout: 15000 });

    // THEN: the resolved facet shows its contextual count
    await expect(rail.getByTestId('facet-count-genre-16')).toHaveText('42');

    // AND: the 0-result chip is dimmed (opacity-70) but NOT disabled (AC2)
    const deadEnd = rail.getByTestId('filter-genre-28');
    await expect(deadEnd).toHaveClass(/opacity-70/);
    await expect(deadEnd).toBeEnabled();

    // AND: the user can still switch to the dead-end facet (it navigates on click)
    await deadEnd.click();
    await expect(page).toHaveURL(/genre=(?:28|%2228%22)/);
  });

  test('[P2] facet-counts unavailable → rail falls back to the single total, no per-chip counts (AC6)', async ({
    page,
  }) => {
    // GIVEN: the counts endpoint errors
    await enableV2Shell(page);
    await stubDiscover(page);
    await stubFacetCountsUnavailable(page);
    await page.goto('/discover');
    const rail = page.getByTestId('discover-filter-rail');
    await expect(rail).toBeVisible({ timeout: 15000 });

    // THEN: the single-total footer still renders (page never hard-fails)
    await expect(page.getByTestId('discover-rail-count')).toHaveText(/符合 \d+ 部|計算中…/);

    // AND: no per-chip counts are rendered (chips degrade to their count-less form)
    await expect(rail.getByTestId('facet-count-genre-16')).toHaveCount(0);
  });
});

// =============================================================================
// Mobile — filter bottom sheet (AC #6)
// =============================================================================

test.describe('Discover Filters — Mobile bottom sheet @e2e @discover', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('[P1] opening the sheet, selecting a genre and applying updates chips + URL (AC #6)', async ({
    page,
  }) => {
    // GIVEN: the discover page on a mobile viewport
    await stubDiscover(page);
    await page.goto('/discover');
    await expect(page.locator('[data-testid="poster-card"]').first()).toBeVisible({
      timeout: 15000,
    });

    // WHEN: the user opens the bottom sheet, picks a genre and applies
    // (scope to the sheet — the hidden desktop sidebar carries the same testids)
    await page.getByTestId('open-filter-sheet').click();
    const sheet = page.getByTestId('filter-bottom-sheet');
    await expect(sheet).toBeVisible();
    await sheet.getByTestId('filter-genre-16').click();
    await page.getByTestId('filter-sheet-apply').click();

    // THEN: the sheet closes, the chip appears and the URL carries the filter
    await expect(page.getByTestId('filter-bottom-sheet')).toHaveCount(0);
    await expect(page.getByTestId('filter-chip-genre-16')).toHaveText(/類型: 動畫/);
    await expect(page).toHaveURL(GENRE_16);
  });
});
