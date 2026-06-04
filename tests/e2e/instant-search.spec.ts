/**
 * Instant Search E2E Tests (Story 11-3)
 *
 * System-level coverage for the GLOBAL instant-search dropdown in the AppShell
 * toolbar. Focuses on the cross-layer journey that unit/component specs cannot
 * exercise end-to-end:
 *   - AppShell toolbar input -> 300ms debounce -> GET /api/v1/search ->
 *     suggestions dropdown (電影/影集/人物) -> navigation
 *   - Click a suggestion -> /media/$type/$id (AC #4)
 *   - Enter (no highlight) -> /search?q=... full results page (AC #1, #6 bridge)
 *   - Arrow-key + Enter -> highlighted result (AC #1, #4)
 *   - Mobile full-screen search view (AC #5)
 *
 * Already covered by unit/component specs (intentionally NOT duplicated here —
 * avoid-duplicate-coverage):
 *   - apps/web/src/components/search/SearchSuggestions.spec.tsx (section render,
 *     person department label, active highlight, empty/loading states)
 *   - apps/web/src/components/search/InstantSearchBar.spec.tsx (debounce timing,
 *     <2-char gating, keyboard index math, clear button)
 *   - apps/web/src/services/tmdb.spec.ts (unifiedSearch URL + camelCase boundary)
 *   - apps/api/internal/services/search_service_test.go (dual-language merge +
 *     zh-TW boost + dedup + graceful degradation)
 *
 * The /api/v1/search endpoint is mocked, so this suite is hermetic and does NOT
 * require TMDB_API_KEY.
 *
 * Prerequisites:
 *   - Frontend running on port 4200: npx nx serve web
 *   - Backend running on port 8080 (AppShell shell requests)
 *
 * @tags @e2e @search @instant-search @story-11-3 @regression
 */

import { test, expect } from '../support/fixtures';
import type { Route, Page } from '@playwright/test';

const ROUTE_API = '**/api/v1';

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

// Backend wire shape (snake_case) — the frontend snakeToCamel-transforms it.
const unifiedResult = {
  query: '你的名字',
  page: 1,
  movies: [
    {
      id: 1,
      title: '你的名字',
      original_title: 'Your Name',
      overview: '',
      release_date: '2016-08-26',
      poster_path: '/posterMovie.jpg',
      backdrop_path: null,
      vote_average: 8.4,
      vote_count: 100,
      genre_ids: [16],
    },
  ],
  tv_shows: [
    {
      id: 2,
      name: '進擊的巨人',
      original_name: 'Attack on Titan',
      overview: '',
      first_air_date: '2013-04-07',
      poster_path: '/posterTv.jpg',
      backdrop_path: null,
      vote_average: 8.6,
      vote_count: 50,
      genre_ids: [16],
    },
  ],
  people: [
    {
      id: 3,
      name: '新海誠',
      original_name: 'Makoto Shinkai',
      profile_path: null,
      known_for_department: 'Directing',
      popularity: 12,
      gender: 2,
    },
  ],
};

/**
 * Stub the unified search endpoint and capture every requested URL so a test can
 * assert the query + page propagate to the backend (AC #1).
 */
async function stubSearch(page: Page): Promise<string[]> {
  const requested: string[] = [];
  await page.route(`${ROUTE_API}/search*`, (route: Route) => {
    requested.push(route.request().url());
    return route.fulfill(jsonOk(unifiedResult));
  });
  return requested;
}

// =============================================================================
// Desktop — global toolbar instant search (AC #1, #4, #6)
// =============================================================================

test.describe('Instant Search — Desktop @e2e @instant-search', () => {
  test('[P0] typing >=2 chars opens a dropdown with 電影 / 影集 / 人物 sections (AC #1)', async ({
    page,
  }) => {
    // GIVEN: the app shell has loaded with its global search input
    const requested = await stubSearch(page);
    await page.goto('/search');
    const input = page.getByTestId('instant-search-input');
    await expect(input).toBeVisible();

    // WHEN: a single character is typed → still below the 2-char threshold
    await input.fill('你');
    await expect(page.getByTestId('search-suggestions')).toBeHidden();

    // WHEN: the query reaches >= 2 characters
    await input.fill('你的名字');

    // THEN: the suggestions dropdown renders all three categories
    const dropdown = page.getByTestId('search-suggestions');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText('電影')).toBeVisible();
    await expect(dropdown.getByText('影集')).toBeVisible();
    await expect(dropdown.getByText('人物')).toBeVisible();
    await expect(dropdown.getByText('你的名字')).toBeVisible();
    await expect(dropdown.getByText('進擊的巨人')).toBeVisible();
    await expect(dropdown.getByText('新海誠')).toBeVisible();

    // AND: the request carried the query + page to the backend
    expect(requested.some((u) => /\/api\/v1\/search\?/.test(u) && /page=1/.test(u))).toBe(true);
  });

  test('[P1] clicking a suggestion navigates to the media detail page (AC #4)', async ({
    page,
  }) => {
    // GIVEN: the dropdown is open with results
    await stubSearch(page);
    await page.goto('/search');
    const input = page.getByTestId('instant-search-input');
    await input.fill('你的名字');
    await expect(page.getByTestId('search-suggestions').getByText('你的名字')).toBeVisible();

    // WHEN: the first (movie) suggestion is clicked
    await page.getByTestId('search-suggestion-item').first().click();

    // THEN: the router navigates to that movie's detail page
    await expect(page).toHaveURL(/\/media\/movie\/1$/);
  });

  test('[P1] pressing Enter with no highlight opens the full /search results page (AC #6 bridge)', async ({
    page,
  }) => {
    // GIVEN: a query has been typed into the global search
    await stubSearch(page);
    await page.goto('/');
    const input = page.getByTestId('instant-search-input');
    await input.fill('你的名字');

    // WHEN: Enter is pressed without highlighting a suggestion
    await input.press('Enter');

    // THEN: the legacy full-results page opens with the query preserved
    await expect(page).toHaveURL(/\/search\?q=%E4%BD%A0%E7%9A%84%E5%90%8D%E5%AD%97/);
  });

  test('[P1] arrow-key navigation + Enter opens the highlighted result (AC #1, #4)', async ({
    page,
  }) => {
    // GIVEN: the dropdown is open (movie first, TV second in nav order)
    await stubSearch(page);
    await page.goto('/search');
    const input = page.getByTestId('instant-search-input');
    await input.fill('你的名字');
    await expect(page.getByTestId('search-suggestions').getByText('進擊的巨人')).toBeVisible();

    // WHEN: ArrowDown twice (movie → TV) then Enter
    await input.press('ArrowDown');
    await input.press('ArrowDown');
    await input.press('Enter');

    // THEN: navigates to the highlighted TV show's detail page
    await expect(page).toHaveURL(/\/media\/tv\/2$/);
  });
});

// =============================================================================
// Mobile — full-screen dedicated search view (AC #5)
// =============================================================================

test.describe('Instant Search — Mobile @e2e @instant-search', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('[P2] mobile search toggle opens a full-screen view with live suggestions (AC #5)', async ({
    page,
  }) => {
    // GIVEN: the mobile app shell with the search toggle button
    await stubSearch(page);
    await page.goto('/search');
    await page.getByTestId('mobile-search-toggle').click();

    // THEN: a dedicated full-screen search overlay opens
    const overlay = page.getByTestId('mobile-search-overlay');
    await expect(overlay).toBeVisible();

    // WHEN: a query is typed in the overlay's input
    await overlay.getByTestId('instant-search-input').fill('你的名字');

    // THEN: suggestions render inside the overlay
    const dropdown = overlay.getByTestId('search-suggestions');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText('你的名字')).toBeVisible();
    await expect(dropdown.getByText('新海誠')).toBeVisible();
  });
});
