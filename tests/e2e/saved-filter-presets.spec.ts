/**
 * Saved Filter Presets E2E Tests (Story 11-4)
 *
 * Integration-level coverage for the /discover saved-preset journey. Focuses on
 * the cross-layer behaviour unit/component specs cannot exercise:
 *   - save dialog -> POST -> preset chip appears (AC #1, #2)
 *   - clicking a preset chip restores the saved filter combination + URL (AC #3)
 *   - presets persist across a full page reload via the DB-backed API (AC #5)
 *   - right-click a preset chip -> confirm -> DELETE removes it (AC #4)
 *
 * Component-level interactions (dialog validation, chip parse/apply logic,
 * delete-confirm toggling) are covered by:
 *   - apps/web/src/components/search/{SavePresetDialog,PresetChips,FilterChipBar}.spec.tsx
 * — intentionally NOT duplicated here (avoid-duplicate-coverage).
 *
 * Both the TMDb discover endpoints AND the filter-presets endpoints are mocked
 * with an in-memory store, so this suite is hermetic. The in-memory store is the
 * stand-in for the SQLite table: a page reload re-fetches GET /filter-presets and
 * must still show the saved preset — proving persistence is server-side, not
 * localStorage (AC #5).
 *
 * Prerequisites:
 *   - Frontend running on port 4200: npx nx serve web
 *   - Backend running on port 8080 (AppShell shell requests)
 *
 * @tags @e2e @discover @preset @story-11-4 @regression
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

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
  ],
  total_pages: 1,
  total_results: 1,
};

const tvResults = { page: 1, results: [], total_pages: 1, total_results: 0 };

interface StoredPreset {
  id: string;
  name: string;
  filters: string;
  sort_order: number;
  created_at: string;
}

/** Stub discover + a stateful in-memory filter-presets store (the DB stand-in). */
async function stubAll(page: import('@playwright/test').Page): Promise<StoredPreset[]> {
  const store: StoredPreset[] = [];
  let seq = 0;

  await page.route(`${ROUTE_API}/tmdb/discover/movies*`, (route: Route) =>
    route.fulfill(jsonOk(movieResults))
  );
  await page.route(`${ROUTE_API}/tmdb/discover/tv*`, (route: Route) =>
    route.fulfill(jsonOk(tvResults))
  );

  // GET list / POST create / DELETE by id — backed by `store`.
  await page.route(`${ROUTE_API}/filter-presets`, (route: Route) => {
    const method = route.request().method();
    if (method === 'POST') {
      const body = JSON.parse(route.request().postData() || '{}');
      const preset: StoredPreset = {
        id: `p${++seq}`,
        name: body.name,
        filters: body.filters,
        sort_order: store.length,
        created_at: '2026-06-04T00:00:00Z',
      };
      store.push(preset);
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: preset }),
      });
    }
    return route.fulfill(jsonOk({ presets: store }));
  });

  await page.route(`${ROUTE_API}/filter-presets/*`, (route: Route) => {
    if (route.request().method() === 'DELETE') {
      const id = route.request().url().split('/').pop();
      const idx = store.findIndex((p) => p.id === id);
      if (idx >= 0) store.splice(idx, 1);
      return route.fulfill(jsonOk({ deleted: true }));
    }
    return route.continue();
  });

  return store;
}

test.describe('Saved Filter Presets — Desktop @e2e @discover @preset', () => {
  test('[P0] save current filters as a preset; chip appears and persists across reload (AC #1, #2, #5)', async ({
    page,
  }) => {
    // GIVEN: the discover page with an active genre filter
    await stubAll(page);
    await page.goto('/discover?genre=16');
    await expect(page.getByTestId('filter-chip-genre-16')).toBeVisible();

    // WHEN: the user saves the current filters as a named preset
    await page.getByTestId('save-preset-button').click();
    await expect(page.getByTestId('save-preset-dialog')).toBeVisible();
    await page.getByTestId('preset-name-input').fill('我的動畫');
    await page.getByTestId('save-preset-confirm').click();

    // THEN: the dialog closes and the preset appears as a quick-access chip
    await expect(page.getByTestId('save-preset-dialog')).toHaveCount(0);
    await expect(page.getByTestId('preset-chips')).toContainText('我的動畫');

    // AND: after a full reload the preset is still present (server-side persistence, AC #5)
    await page.reload();
    await expect(page.getByTestId('preset-chips')).toContainText('我的動畫');
  });

  test('[P0] clicking a saved preset restores its filters and URL (AC #3)', async ({ page }) => {
    // GIVEN: a saved preset exists (created via the UI)
    await stubAll(page);
    await page.goto('/discover?genre=16');
    await page.getByTestId('save-preset-button').click();
    await page.getByTestId('preset-name-input').fill('我的動畫');
    await page.getByTestId('save-preset-confirm').click();
    await expect(page.getByTestId('preset-chips')).toContainText('我的動畫');

    // AND: filters are cleared away
    await page.getByRole('button', { name: '移除類型: 動畫篩選' }).click();
    await expect(page).not.toHaveURL(/genre=16/);

    // WHEN: the user clicks the saved preset chip
    await page.getByTestId('preset-chips').getByText('我的動畫').click();

    // THEN: the genre filter is restored to the URL + chip bar
    await expect(page).toHaveURL(/genre=(?:16|%2216%22)/);
    await expect(page.getByTestId('filter-chip-genre-16')).toBeVisible();
  });

  test('[P1] right-click a preset chip, confirm, and it is deleted (AC #4)', async ({ page }) => {
    // GIVEN: a saved preset
    await stubAll(page);
    await page.goto('/discover?genre=16');
    await page.getByTestId('save-preset-button').click();
    await page.getByTestId('preset-name-input').fill('待刪除');
    await page.getByTestId('save-preset-confirm').click();
    const chip = page.getByTestId('preset-chips').getByText('待刪除');
    await expect(chip).toBeVisible();

    // WHEN: the user right-clicks the chip and confirms deletion
    await chip.click({ button: 'right' });
    await expect(page.getByTestId('preset-delete-dialog')).toBeVisible();
    await page.getByTestId('preset-delete-confirm').click();

    // THEN: the preset chip is removed
    await expect(page.getByTestId('preset-delete-dialog')).toHaveCount(0);
    await expect(page.getByTestId('preset-chips')).toHaveCount(0);
  });
});
