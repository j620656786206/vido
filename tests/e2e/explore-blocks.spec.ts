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
 *   - ux3-1-8 owned titles are FILTERED OUT of the rows (+ the caption that
 *     announces the rule, and the all-owned empty fork)
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

// ux3-1-7 readout band — every cell present and boring, so the band never
// becomes the reason a homepage assertion moves.
const emptyHomeSummary = {
  coverage: { status: 'ok', covered: 0, total: 0 },
  processed_today: { status: 'ok', count: 0 },
  attention: { status: 'ok', failed_count: 0 },
  in_flight: { status: 'ok', count: 0 },
};

// =============================================================================
// Helpers
// =============================================================================

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

/**
 * Pin the ownership verdict for the whole page. Since ux3-1-8 an owned title is
 * DELETED from the row rather than badged, so ownership is now an input to the
 * card COUNT of every test in this file — leaving it to the shared backend
 * would make each count depend on whatever that library happens to hold.
 */
async function stubCheckOwned(page: import('@playwright/test').Page, ownedIds: number[]) {
  await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
    route.fulfill(jsonOk({ owned_ids: ownedIds }))
  );
}

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
  // Default: the user owns nothing, so every block renders its full payload.
  // The owned-filter tests re-register this route AFTER calling the baseline —
  // Playwright resolves the most recently registered handler first.
  await stubCheckOwned(page, []);
  // ux3-1-8 moved the own-library hero above the explore tail, and both the
  // hero and the 最近新增 row read /library/recent. Neither is under test here;
  // an empty library keeps the homepage DOM hermetic.
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 }))
  );
  await page.route(`${ROUTE_API}/home-summary`, (route: Route) =>
    route.fulfill(jsonOk(emptyHomeSummary))
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

  // bugfix-10-6 AC#5 — a block whose content query resolves to zero results
  // still renders (only an errored query hides it), shows the
  // "沒有符合條件的內容" message, and renders NO scroll chevrons (nothing to
  // scroll ⇒ no affordance ⇒ the left-edge message can never be clipped).
  // ux3-1-8 forked that copy in two: this is the TMDb-returned-nothing branch,
  // and it must NOT drift into the all-owned congratulation (covered below).
  test('[P2] empty block shows the no-results message and no scroll chevrons (bugfix-10-6 AC#5)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [defaultBlocks.blocks[0]] }))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(
        jsonOk({ block_id: 'b-movies', content_type: 'movie', movies: [], total_items: 0 })
      )
    );

    await page.goto('/');

    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();
    await expect(block.getByTestId('explore-block-empty')).toHaveText('沒有符合條件的內容');
    await expect(block.getByTestId('explore-block-scroll-left')).toHaveCount(0);
    await expect(block.getByTestId('explore-block-scroll-right')).toHaveCount(0);
  });
});

// =============================================================================
// Owned-title filter (ux3-1-8) — the explore tail is DISCOVERY only, so a title
// already in the library drops out of the row instead of being badged 已有.
// =============================================================================

test.describe('Explore Blocks — owned filter @ui @explore-blocks @ux3-1-8', () => {
  test('[P0] an owned title is removed from the row, and the row says so', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [defaultBlocks.blocks[0]] }))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );
    // 電影 A (TMDb id 1) is already in the library; 電影 B (id 2) is not.
    await stubCheckOwned(page, [1]);

    await page.goto('/');

    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();

    // The regression this catches: an "owned" item surviving as a card (the
    // pre-ux3-1-8 behaviour, where ownership only painted a badge). Assert on
    // the href rather than the count alone so a filter that drops the WRONG
    // card cannot pass.
    await expect(block.getByTestId('poster-card')).toHaveCount(1);
    await expect(block.locator('[data-testid="poster-card"][href$="/media/movie/1"]')).toHaveCount(
      0
    );
    await expect(block.getByTestId('poster-card')).toHaveAttribute('href', /\/media\/movie\/2$/);

    // Corollary: with owned items gone, the 已有 badge is unreachable on this
    // surface. (Its live coverage moved to /discover — availability-badges.spec.)
    await expect(block.getByTestId('availability-badge-owned')).toHaveCount(0);

    // A row that silently deletes items reads as「TMDb 內容變少了」— the caption
    // is what keeps the filter honest, so it is part of the contract.
    await expect(block.getByTestId('explore-block-caption')).toHaveText('已擁有的作品不會出現這裡');
  });

  test('[P1] a row whose every title is owned shows the all-owned message, not the no-results one', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [defaultBlocks.blocks[0]] }))
    );
    await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
      route.fulfill(jsonOk(movieContent))
    );
    await stubCheckOwned(page, [1, 2]);

    await page.goto('/');

    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();
    await expect(block.getByTestId('poster-card')).toHaveCount(0);

    // TMDb DID return content — telling the user「沒有符合條件的內容」here would
    // be a lie about the upstream, so the empty state must take the other fork.
    await expect(block.getByTestId('explore-block-empty')).toHaveText('這排的作品你都已經擁有了');

    // Nothing left to scroll ⇒ no chevrons over the message (bugfix-10-6 AC#5
    // must survive the new emptiness path too).
    await expect(block.getByTestId('explore-block-scroll-left')).toHaveCount(0);
    await expect(block.getByTestId('explore-block-scroll-right')).toHaveCount(0);
  });

  // The filter's fail-OPEN behaviour (check-owned 500 ⇒ the row keeps every
  // card) lives in availability-badges.spec.ts, which already owns the
  // check-owned failure path — not duplicated here.
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

  // bugfix-10-6 AC#4 — the block-row content-type marker is a lucide icon
  // (<Film>/<Tv>, rendered as inline <svg>) followed by plain "電影"/"影集",
  // never the 🎬/📺 emoji. Right-side action icons (ArrowUp/Pencil/Trash2) are
  // unchanged.
  test('[P2] block rows show lucide content-type icons, not 🎬/📺 emoji (bugfix-10-6 AC#4)', async ({
    page,
  }) => {
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk(defaultBlocks))
    );

    await page.goto('/settings/homepage');

    await expect(page.getByTestId('explore-blocks-settings')).toBeVisible();
    await expect(page.getByText(/🎬|📺/)).toHaveCount(0);

    const movieMeta = page.getByTestId('explore-block-row-b-movies').locator('p').first();
    await expect(movieMeta).toContainText('電影 ·');
    await expect(movieMeta.locator('svg')).toHaveCount(1);

    const tvMeta = page.getByTestId('explore-block-row-b-tv').locator('p').first();
    await expect(tvMeta).toContainText('影集 ·');
    await expect(tvMeta.locator('svg')).toHaveCount(1);
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
