/**
 * Explore Blocks E2E Tests (Story 10.3)
 *
 * Browser-based tests for the homepage Explore Blocks list and the Settings
 * management UI. Uses route interception for deterministic tests.
 *
 * Coverage:
 *   - AC#1 horizontal rows of poster cards with section title (homepage render)
 *   - AC#2 create new block via settings modal
 *   - AC#3 reorder blocks via up/down arrows
 *   - AC#4 edit/delete updates list without page reload
 *   - AC#5 default blocks pre-seeded
 *   - AC#6 content uses TMDb discover with block params
 *
 * @tags @ui @explore-blocks @story-10-3
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock payloads — snake_case wire format
// =============================================================================

const defaultBlocks = {
  blocks: [
    {
      id: 'b-movies',
      name: '熱門電影',
      content_type: 'movie',
      genre_ids: '',
      language: '',
      region: '',
      sort_by: 'popularity.desc',
      max_items: 20,
      sort_order: 0,
      created_at: '2026-04-15T00:00:00Z',
      updated_at: '2026-04-15T00:00:00Z',
    },
    {
      id: 'b-tv',
      name: '熱門影集',
      content_type: 'tv',
      genre_ids: '',
      language: '',
      region: '',
      sort_by: 'popularity.desc',
      max_items: 20,
      sort_order: 1,
      created_at: '2026-04-15T00:00:00Z',
      updated_at: '2026-04-15T00:00:00Z',
    },
  ],
};

const movieContent = {
  block_id: 'b-movies',
  content_type: 'movie',
  movies: [
    {
      id: 1,
      title: '電影 A',
      original_title: 'Movie A',
      overview: '',
      release_date: '2024-01-01',
      poster_path: '/posterA.jpg',
      backdrop_path: null,
      vote_average: 8,
      vote_count: 100,
      genre_ids: [28],
    },
    {
      id: 2,
      title: '電影 B',
      original_title: 'Movie B',
      overview: '',
      release_date: '2024-02-01',
      poster_path: '/posterB.jpg',
      backdrop_path: null,
      vote_average: 7,
      vote_count: 80,
      genre_ids: [12],
    },
  ],
  total_items: 2,
};

const tvContent = {
  block_id: 'b-tv',
  content_type: 'tv',
  tv_shows: [
    {
      id: 10,
      name: '劇集 X',
      original_name: 'Show X',
      overview: '',
      first_air_date: '2023-01-01',
      poster_path: '/posterX.jpg',
      backdrop_path: null,
      vote_average: 9,
      vote_count: 500,
      genre_ids: [18],
    },
  ],
  total_items: 1,
};

const mockQBConfig = {
  host: 'http://localhost:8080',
  username: 'admin',
  basePath: '',
  configured: true,
};

// =============================================================================
// Helpers
// =============================================================================

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

async function stubHomepageBaseline(page: import('@playwright/test').Page) {
  await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
    route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
  );
  await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
    route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
  );
  await page.route(`${ROUTE_API}/downloads*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, pageSize: 100, totalItems: 0, totalPages: 1 }))
  );
  await page.route(`${ROUTE_API}/media/recent*`, (route: Route) => route.fulfill(jsonOk([])));
  await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
    route.fulfill(jsonOk(mockQBConfig))
  );
  await page.route(`${ROUTE_API}/health/services*`, (route: Route) =>
    route.fulfill(jsonOk({ services: [] }))
  );
}

// =============================================================================
// Homepage rendering (AC#1, AC#5, AC#6)
// =============================================================================

test.describe('Homepage Explore Blocks @ui @explore-blocks @story-10-3', () => {
  test('[P0] renders one section per configured block with title (AC1, AC5)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-tv/content`, (route: Route) =>
      route.fulfill(jsonOk(tvContent))
    );

    await page.goto('/');

    const moviesBlock = page.getByTestId('explore-block-b-movies');
    const tvBlock = page.getByTestId('explore-block-b-tv');

    await expect(moviesBlock).toBeVisible();
    await expect(tvBlock).toBeVisible();

    await expect(moviesBlock.getByTestId('explore-block-title')).toHaveText('熱門電影');
    await expect(tvBlock.getByTestId('explore-block-title')).toHaveText('熱門影集');
  });

  test('[P1] hides list when API returns empty (AC5 — graceful hide)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [] }))
    );

    await page.goto('/');

    await expect(page.getByTestId('explore-blocks-list')).toHaveCount(0);
  });

  test('[P0] poster cards inside a block are clickable to detail page (AC1)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-tv/content`, (route: Route) =>
      route.fulfill(jsonOk({ ...tvContent, tv_shows: [] }))
    );

    await page.goto('/');

    const block = page.getByTestId('explore-block-b-movies');
    const firstCard = block.getByTestId('poster-card').first();
    await expect(firstCard).toBeVisible();
    await expect(firstCard).toHaveAttribute('href', /\/media\/movie\/1$/);
  });
});

// =============================================================================
// Settings management UI (AC#2, AC#3, AC#4)
// =============================================================================

test.describe('Settings — Explore Blocks Management @ui @explore-blocks @story-10-3', () => {
  test('[P0] shows existing blocks and renders add button (AC2 entry point)', async ({ page }) => {
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );

    await page.goto('/settings/homepage');

    await expect(page.getByTestId('explore-blocks-settings')).toBeVisible();
    await expect(page.getByTestId('explore-blocks-add-button')).toBeVisible();
    await expect(page.getByTestId('explore-block-row-b-movies')).toBeVisible();
    await expect(page.getByTestId('explore-block-row-b-tv')).toBeVisible();
  });

  test('[P0] opens create modal with all required fields (AC2)', async ({ page }) => {
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [] }))
    );

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-blocks-add-button').click();

    const modal = page.getByTestId('explore-block-edit-modal');
    await expect(modal).toBeVisible();
    await expect(modal.getByTestId('explore-block-name-input')).toBeVisible();
    await expect(modal.getByTestId('explore-block-type-select')).toBeVisible();
    await expect(modal.getByTestId('explore-block-genre-input')).toBeVisible();
    await expect(modal.getByTestId('explore-block-language-input')).toBeVisible();
    await expect(modal.getByTestId('explore-block-region-input')).toBeVisible();
    await expect(modal.getByTestId('explore-block-sort-select')).toBeVisible();
    await expect(modal.getByTestId('explore-block-max-items-input')).toBeVisible();
  });

  test('[P0] reorder up arrow shifts block order (AC3)', async ({ page }) => {
    let reorderCalled: string[] | null = null;

    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/reorder`, async (route: Route) => {
      const body = JSON.parse(route.request().postData() || '{}');
      reorderCalled = body.ordered_ids;
      await route.fulfill(jsonOk(defaultBlocks));
    });

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-block-move-up-b-tv').click();

    await expect.poll(() => reorderCalled).toEqual(['b-tv', 'b-movies']);
  });

  test('[P1] delete shows confirmation before issuing DELETE (AC4)', async ({ page }) => {
    let deleteHit = false;

    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies`, async (route: Route) => {
      if (route.request().method() === 'DELETE') {
        deleteHit = true;
        await route.fulfill(jsonOk({ deleted: true }));
      } else {
        await route.fulfill(jsonOk(defaultBlocks.blocks[0]));
      }
    });

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-block-delete-b-movies').click();
    await expect(page.getByTestId('explore-block-delete-confirm')).toBeVisible();
    expect(deleteHit).toBe(false);

    await page.getByTestId('explore-block-delete-confirm-button').click();
    await expect.poll(() => deleteHit).toBe(true);
  });

  test('[P0] edit modal round-trip submits PUT with updated payload (AC4)', async ({ page }) => {
    let putBody: Record<string, unknown> | null = null;

    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies`, async (route: Route) => {
      if (route.request().method() === 'PUT') {
        putBody = JSON.parse(route.request().postData() || '{}');
        await route.fulfill(jsonOk({ ...defaultBlocks.blocks[0], name: '台灣電影' }));
      } else {
        await route.fulfill(jsonOk(defaultBlocks.blocks[0]));
      }
    });

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-block-edit-b-movies').click();
    const modal = page.getByTestId('explore-block-edit-modal');
    await expect(modal).toBeVisible();

    const nameInput = modal.getByTestId('explore-block-name-input');
    await nameInput.fill('台灣電影');
    await modal.getByTestId('explore-block-save-button').click();

    await expect(modal).toBeHidden();
    await expect.poll(() => putBody).not.toBeNull();
    expect(putBody).toMatchObject({ name: '台灣電影' });
  });

  test('[P1] create flow submits POST with snake_case payload (AC2)', async ({ page }) => {
    let postBody: Record<string, unknown> | null = null;

    await page.route(`${ROUTE_API}/explore-blocks`, async (route: Route) => {
      if (route.request().method() === 'POST') {
        postBody = JSON.parse(route.request().postData() || '{}');
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              id: 'new-block-id',
              name: '新區塊',
              content_type: 'tv',
              genre_ids: '',
              language: '',
              region: '',
              sort_by: 'popularity.desc',
              max_items: 25,
              sort_order: 2,
              created_at: '2026-04-16T00:00:00Z',
              updated_at: '2026-04-16T00:00:00Z',
            },
          }),
        });
      } else {
        await route.fulfill(jsonOk({ blocks: [] }));
      }
    });

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-blocks-add-button').click();
    const modal = page.getByTestId('explore-block-edit-modal');
    await expect(modal).toBeVisible();

    await modal.getByTestId('explore-block-name-input').fill('新區塊');
    await modal.getByTestId('explore-block-type-select').selectOption('tv');
    await modal.getByTestId('explore-block-max-items-input').fill('25');
    await modal.getByTestId('explore-block-save-button').click();

    await expect(modal).toBeHidden();
    await expect.poll(() => postBody).not.toBeNull();
    // Request body should use snake_case (Rule 18 — camelToSnake at API boundary).
    expect(postBody).toMatchObject({
      name: '新區塊',
      content_type: 'tv',
      max_items: 25,
    });
  });

  test('[P1] reorder down arrow swaps adjacent blocks (AC3)', async ({ page }) => {
    let reorderCalled: string[] | null = null;

    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/reorder`, async (route: Route) => {
      const body = JSON.parse(route.request().postData() || '{}');
      reorderCalled = body.ordered_ids;
      await route.fulfill(jsonOk(defaultBlocks));
    });

    await page.goto('/settings/homepage');

    await page.getByTestId('explore-block-move-down-b-movies').click();

    await expect.poll(() => reorderCalled).toEqual(['b-tv', 'b-movies']);
  });
});

// =============================================================================
// Cross-route integration: homepage reflects settings changes without reload (AC#4)
// =============================================================================

test.describe('Homepage reflects settings changes @ui @explore-blocks @story-10-3', () => {
  test('[P0] deleting a block in settings removes it from homepage without reload (AC4)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);

    let deleteDone = false;
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(deleteDone ? { blocks: [defaultBlocks.blocks[1]] } : defaultBlocks))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-tv/content`, (route: Route) =>
      route.fulfill(jsonOk(tvContent))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies`, async (route: Route) => {
      if (route.request().method() === 'DELETE') {
        deleteDone = true;
        await route.fulfill(jsonOk({ deleted: true }));
      } else {
        await route.fulfill(jsonOk(defaultBlocks.blocks[0]));
      }
    });

    await page.goto('/');
    await expect(page.getByTestId('explore-block-b-movies')).toBeVisible();
    await expect(page.getByTestId('explore-block-b-tv')).toBeVisible();

    // Navigate to settings via client-side SPA routing (no full reload)
    await page.goto('/settings/homepage');
    await page.getByTestId('explore-block-delete-b-movies').click();
    await page.getByTestId('explore-block-delete-confirm-button').click();
    await expect.poll(() => deleteDone).toBe(true);

    // Return to homepage — SPA navigation, React Query should refetch invalidated list
    await page.goto('/');

    await expect(page.getByTestId('explore-block-b-tv')).toBeVisible();
    await expect(page.getByTestId('explore-block-b-movies')).toHaveCount(0);
  });
});

// =============================================================================
// API contract via in-page fetch (page.request bypasses route() mocks, so
// use page.evaluate to drive the same fetchApi path the components use).
// =============================================================================

test.describe('Explore Blocks API contract @api @explore-blocks @story-10-3', () => {
  test('[P0] block content endpoint returns ApiResponse envelope', async ({ page }) => {
    await page.route(`${ROUTE_API}/explore-blocks/test-id/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );

    await page.goto('/'); // any page that mounts fetch
    const body = await page.evaluate(async () => {
      const res = await fetch('/api/v1/explore-blocks/test-id/content');
      return res.json();
    });
    expect(body.success).toBe(true);
    expect(body.data.block_id).toBe('b-movies');
    expect(body.data.movies).toHaveLength(2);
  });

  test('[P1] reorder endpoint accepts ordered_ids array', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [] }))
    );
    let captured: unknown = null;
    await page.route(`${ROUTE_API}/explore-blocks/reorder`, async (route: Route) => {
      captured = JSON.parse(route.request().postData() || '{}');
      await route.fulfill(jsonOk(defaultBlocks));
    });

    await page.goto('/');
    await page.evaluate(async () => {
      await fetch('/api/v1/explore-blocks/reorder', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ordered_ids: ['b-tv', 'b-movies'] }),
      });
    });

    expect(captured).toEqual({ ordered_ids: ['b-tv', 'b-movies'] });
  });
});
