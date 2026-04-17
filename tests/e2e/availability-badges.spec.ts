/**
 * Availability Badges E2E Tests (Story 10-4)
 *
 * Browser-based tests for the homepage 已有 / 已請求 badges that overlay
 * PosterCards in ExploreBlocks. DEV's unit tests mocked every boundary —
 * this suite exercises the full wire-level stack:
 *
 *   ExploreBlock → useOwnedMedia → availabilityService → POST /media/check-owned
 *                                                        ↓
 *                                            AvailabilityHandler → Service → Repo
 *
 * Coverage gaps this suite closes (vs. DEV's unit tests):
 *   - [P0] Real camelToSnake (tmdbIds → tmdb_ids) body transform in POST
 *   - [P0] Real snakeToCamel (owned_ids → ownedIds) response transform
 *   - [P0] Mobile viewport (flow-g-homepage-mobile) — badge positioning
 *   - [P1] Batching: exactly one POST regardless of number of visible cards
 *   - [P1] Empty visible cards → no POST fired (lazy enabled: false)
 *   - [P1] 500 error → ExploreBlock still renders, just without badges
 *
 * Design notes:
 *   - Route interception is installed BEFORE page.goto so the hook's initial
 *     fetch is intercepted (network-first pattern, knowledge/network-first.md).
 *   - Mock payloads are snake_case at the wire (fetchApi runs snakeToCamel).
 *   - POST request body is captured via a `postDataJSON()` snapshot to verify
 *     Rule 18 (camelToSnake on POST bodies) at the real network layer.
 *
 * @tags @ui @availability-badges @story-10-4
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
  ],
};

// Three movies — two owned, one not. The owned IDs (603, 157336) come back
// from the stubbed /media/check-owned response.
const movieContent = {
  block_id: 'b-movies',
  content_type: 'movie',
  movies: [
    {
      id: 603,
      title: '駭客任務',
      original_title: 'The Matrix',
      overview: '',
      release_date: '1999-03-31',
      poster_path: '/matrix.jpg',
      backdrop_path: null,
      vote_average: 8.7,
      vote_count: 22000,
      genre_ids: [28],
    },
    {
      id: 157336,
      title: '星際效應',
      original_title: 'Interstellar',
      overview: '',
      release_date: '2014-11-07',
      poster_path: '/interstellar.jpg',
      backdrop_path: null,
      vote_average: 8.6,
      vote_count: 32000,
      genre_ids: [18],
    },
    {
      id: 999999,
      title: '未擁有的電影',
      original_title: 'Unowned Movie',
      overview: '',
      release_date: '2024-01-01',
      poster_path: '/unowned.jpg',
      backdrop_path: null,
      vote_average: 5.0,
      vote_count: 10,
      genre_ids: [28],
    },
  ],
  total_items: 3,
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

const jsonError = (status: number, code: string, message: string) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify({ success: false, error: { code, message } }),
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

async function stubExploreBlocksWith(
  page: import('@playwright/test').Page,
  content: typeof movieContent
) {
  await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
    route.fulfill(jsonOk(defaultBlocks))
  );
  await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
    route.fulfill(jsonOk(content))
  );
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Availability Badges @ui @availability-badges @story-10-4', () => {
  test('[P0] renders 已有 badge on owned cards after real wire roundtrip (AC #1)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    // Owned set: first two IDs. This is the canonical success path — proves
    // the whole stack (card → hook → service → POST → handler mock → badge).
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603, 157336] }))
    );

    await page.goto('/');

    // GIVEN: ExploreBlock has rendered all three cards
    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();

    // THEN: exactly two 已有 badges appear — one per owned card, zero for the
    // unowned one. Counting across the whole page doubles as a regression
    // check that no stray badge leaks into other surfaces.
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(2);
    await expect(page.getByTestId('availability-badge-requested')).toHaveCount(0);
  });

  test('[P0] POST body uses snake_case tmdb_ids — Rule 18 wire contract (AC #4)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    let capturedBody: unknown = null;
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) => {
      capturedBody = route.request().postDataJSON();
      return route.fulfill(jsonOk({ owned_ids: [] }));
    });

    await page.goto('/');

    // WAIT for the card to render so the hook has fired.
    await expect(page.getByTestId('explore-block-b-movies')).toBeVisible();
    // Poll for the body to land. waitFor is preferred over hard wait.
    await expect.poll(() => capturedBody).not.toBeNull();

    // THEN: body is snake_case per Rule 18 — NOT tmdbIds.
    expect(capturedBody).toHaveProperty('tmdb_ids');
    expect((capturedBody as { tmdb_ids: number[] }).tmdb_ids).toEqual(
      expect.arrayContaining([603, 157336, 999999])
    );
    // Negative assertion: camelCase key must NOT be present (catches a
    // regression where camelToSnake is removed from the service).
    expect(capturedBody).not.toHaveProperty('tmdbIds');
  });

  test('[P1] fires exactly one POST regardless of visible card count (AC #4 batching)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    let postCount = 0;
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) => {
      postCount += 1;
      return route.fulfill(jsonOk({ owned_ids: [603] }));
    });

    await page.goto('/');

    // Wait for the first badge to guarantee the POST has resolved.
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(1);

    // N+1 guard: even though there are 3 cards, there is exactly ONE POST.
    expect(postCount).toBe(1);
  });

  test('[P1] empty block → no POST fired (lazy enabled)', async ({ page }) => {
    await stubHomepageBaseline(page);
    // Empty movie list.
    await stubExploreBlocksWith(page, { ...movieContent, movies: [], total_items: 0 });

    let postCount = 0;
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) => {
      postCount += 1;
      return route.fulfill(jsonOk({ owned_ids: [] }));
    });

    await page.goto('/');

    // Empty-state message must be visible — proves the page settled.
    await expect(page.getByTestId('explore-block-empty')).toBeVisible();

    // THEN: no badge request ever fired — AC #4 efficiency extends to the
    // empty case. The hook's `enabled: false` guard must hold.
    expect(postCount).toBe(0);
  });

  test('[P1] 500 from check-owned → cards still render without badges (graceful degradation)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    // Simulate check-owned backend failure.
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonError(500, 'INTERNAL_ERROR', 'server boom'))
    );

    await page.goto('/');

    // ExploreBlock still renders — the ownership failure must not brick the
    // discovery surface. (DEV's ExploreBlock has no try/catch around the hook
    // because useOwnedMedia returns { owned: Set<number>(), error } instead
    // of throwing.)
    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();
    await expect(block.getByTestId('poster-card')).toHaveCount(3);

    // No badges because ownership is unknown.
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(0);
    await expect(page.getByTestId('availability-badge-requested')).toHaveCount(0);
  });

  test('[P1] mobile viewport (375x667) — badge still positioned top-right (flow-g mobile)', async ({
    page,
  }) => {
    // Set BEFORE navigation so the first render is at mobile width.
    await page.setViewportSize({ width: 375, height: 667 });

    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603] }))
    );

    await page.goto('/');

    const ownedBadge = page.getByTestId('availability-badge-owned').first();
    await expect(ownedBadge).toBeVisible();
    await expect(ownedBadge).toHaveText('已有');

    // Positional sanity: the badge is in the right-side cluster (the owned
    // badge is the LEFTMOST in the cluster, so its center may sit just left
    // of card-midpoint — but its right edge must be in the right 60% of the
    // card, and it must be in the top quarter).
    const card = page.getByTestId('poster-card').first();
    const cardBox = await card.boundingBox();
    const badgeBox = await ownedBadge.boundingBox();
    expect(cardBox).not.toBeNull();
    expect(badgeBox).not.toBeNull();

    const badgeRightEdge = badgeBox!.x + badgeBox!.width;
    const cardRightEdgeThreshold = cardBox!.x + 0.4 * cardBox!.width;
    expect(badgeRightEdge).toBeGreaterThan(cardRightEdgeThreshold);

    const cardQuarterY = cardBox!.y + cardBox!.height / 4;
    expect(badgeBox!.y).toBeLessThan(cardQuarterY);
  });
});
