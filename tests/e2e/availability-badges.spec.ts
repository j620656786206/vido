/**
 * Availability Badges E2E Tests (Story 10-4)
 *
 * Browser-based tests for the 已有 / 已請求 badges that overlay PosterCards,
 * plus the ownership wire they both ride on. DEV's unit tests mocked every
 * boundary — this suite exercises the full wire-level stack:
 *
 *   ExploreBlock / MediaGrid → useOwnedMedia → availabilityService
 *                                            → POST /media/check-owned
 *                                                        ↓
 *                                            AvailabilityHandler → Service → Repo
 *
 * SURFACE SPLIT (ux3-1-8): the homepage explore rows now DELETE owned titles
 * instead of badging them, so 已有 is unreachable there by construction. The
 * owned-badge coverage therefore runs on /discover (MediaGrid), which is the
 * surface that still legitimately renders it — the homepage keeps the honest
 * inverse guard (owned ⇒ no card, and no stray 已有 anywhere on the page).
 * 已請求 is untouched by the filter and stays covered on the homepage row.
 *
 * Coverage gaps this suite closes (vs. DEV's unit tests):
 *   - [P0] Real camelToSnake (tmdbIds → tmdb_ids) body transform in POST
 *   - [P0] Real snakeToCamel (owned_ids → ownedIds) response transform — /discover
 *   - [P0] Homepage: an owned title renders no card at all (ux3-1-8)
 *   - [P0] Mobile viewport — badge positioning (/discover)
 *   - [P1] Batching: exactly one POST regardless of number of visible cards
 *   - [P1] Empty visible cards → no POST fired (lazy enabled: false)
 *   - [P1] 500 error → ExploreBlock still renders every card (filter fails OPEN)
 *   - [P1] 已請求 still badges a non-owned title on the homepage row
 *
 * Design notes:
 *   - Route interception is installed BEFORE page.goto so the hook's initial
 *     fetch is intercepted (network-first pattern, knowledge/network-first.md).
 *   - Mock payloads are snake_case at the wire (fetchApi runs snakeToCamel).
 *   - POST request body is captured via a `postDataJSON()` snapshot to verify
 *     Rule 18 (camelToSnake on POST bodies) at the real network layer.
 *
 * @tags @ui @availability-badges @story-10-4 @ux3-1-8
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

// The same three movies served as a /discover result page. ux3-1-8 made the
// two surfaces behave differently on ownership, so they are fed identical data
// to keep the difference attributable to the surface, not the payload.
const discoverMovieResults = {
  page: 1,
  results: movieContent.movies,
  total_pages: 1,
  total_results: movieContent.movies.length,
};

const emptyTvResults = { page: 1, results: [], total_pages: 1, total_results: 0 };

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

/** An active (pending) request — the only state that earns the 已請求 badge. */
const activeMovieRequest = (tmdbId: number, title: string) => ({
  id: `req-${tmdbId}`,
  tmdb_id: tmdbId,
  media_type: 'movie',
  title,
  status: 'pending',
  fulfilment_source: null,
  external_id: null,
  seasons: null,
  episodes: null,
  error_message: null,
  requested_at: '2026-08-26T00:00:00Z',
  updated_at: '2026-08-26T00:00:00Z',
});

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
  // No open requests by default — otherwise every 已請求 count in this file
  // would depend on whatever the shared backend happens to be tracking.
  await page.route(`${ROUTE_API}/requests`, (route: Route) =>
    route.fulfill(jsonOk({ requests: [] }))
  );
  // ux3-1-8 put an own-library hero above the explore tail; it and the 最近新增
  // row share /library/recent, and ux3-1-7's band reads /home-summary. Neither
  // is under test here — pin them so the homepage DOM stays hermetic.
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 }))
  );
  await page.route(`${ROUTE_API}/home-summary`, (route: Route) =>
    route.fulfill(jsonOk(emptyHomeSummary))
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

/**
 * /discover baseline — the surface that still renders the 已有 badge after
 * ux3-1-8 (MediaGrid keeps owned titles on screen; only the homepage explore
 * rows drop them).
 */
async function stubDiscoverBaseline(page: import('@playwright/test').Page) {
  await page.route(`${ROUTE_API}/tmdb/discover/movies*`, (route: Route) =>
    route.fulfill(jsonOk(discoverMovieResults))
  );
  await page.route(`${ROUTE_API}/tmdb/discover/tv*`, (route: Route) =>
    route.fulfill(jsonOk(emptyTvResults))
  );
  await page.route(`${ROUTE_API}/tmdb/discover/facet-counts*`, (route: Route) =>
    route.fulfill(jsonOk({ counts: {}, partial: false }))
  );
  await page.route(`${ROUTE_API}/requests`, (route: Route) =>
    route.fulfill(jsonOk({ requests: [] }))
  );
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Availability Badges — Discover grid @ui @availability-badges @story-10-4', () => {
  // MOVED here from the homepage by ux3-1-8: an owned title no longer reaches a
  // homepage explore row, so /discover (MediaGrid) is the only surface left
  // where the badge's full wire roundtrip can be observed end-to-end.
  test('[P0] renders 已有 badge on owned cards after real wire roundtrip (AC #1)', async ({
    page,
  }) => {
    await stubDiscoverBaseline(page);

    // Owned set: first two IDs. This is the canonical success path — proves
    // the whole stack (card → hook → service → POST → handler mock → badge).
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603, 157336] }))
    );

    await page.goto('/discover');

    // GIVEN: the grid has rendered all three results — /discover does NOT drop
    // owned titles (that filter is the homepage row's job alone), so a missing
    // card here would be the ux3-1-8 filter leaking onto the wrong surface.
    await expect(page.getByTestId('media-grid')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('poster-card')).toHaveCount(3);

    // THEN: exactly two 已有 badges appear — one per owned card, zero for the
    // unowned one. Counting across the whole page doubles as a regression
    // check that no stray badge leaks into other surfaces.
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(2);
    await expect(page.getByTestId('availability-badge-requested')).toHaveCount(0);
  });

  test('[P1] mobile viewport (375x667) — badge still positioned top-right', async ({ page }) => {
    // Set BEFORE navigation so the first render is at mobile width.
    await page.setViewportSize({ width: 375, height: 667 });

    await stubDiscoverBaseline(page);
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603] }))
    );

    await page.goto('/discover');

    // Locate the card BY its badge — /discover sorts by vote count, and this
    // assertion is about geometry inside a card, not about ordering.
    const ownedCard = page
      .getByTestId('poster-card')
      .filter({ has: page.getByTestId('availability-badge-owned') })
      .first();
    const ownedBadge = ownedCard.getByTestId('availability-badge-owned');
    await expect(ownedBadge).toBeVisible();
    await expect(ownedBadge).toHaveText('已有');

    // Positional sanity: the badge is in the right-side cluster (the owned
    // badge is the LEFTMOST in the cluster, so its center may sit just left
    // of card-midpoint — but its right edge must be in the right 60% of the
    // card, and it must be in the top quarter).
    const cardBox = await ownedCard.boundingBox();
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

test.describe('Availability Badges — homepage explore rows @ui @availability-badges @story-10-4 @ux3-1-8', () => {
  test('[P0] an owned title renders NO card on the homepage row (ux3-1-8)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603, 157336] }))
    );

    await page.goto('/');

    // The badge's homepage coverage inverted rather than vanished: what used to
    // be「owned ⇒ 已有 badge」is now「owned ⇒ not on this surface at all」. The
    // href check pins WHICH cards were dropped, so a filter that removes the
    // wrong ones cannot pass on the count alone.
    const block = page.getByTestId('explore-block-b-movies');
    await expect(block).toBeVisible();
    await expect(block.getByTestId('poster-card')).toHaveCount(1);
    await expect(
      block.locator('[data-testid="poster-card"][href$="/media/movie/603"]')
    ).toHaveCount(0);
    await expect(
      block.locator('[data-testid="poster-card"][href$="/media/movie/157336"]')
    ).toHaveCount(0);
    await expect(block.getByTestId('poster-card')).toHaveAttribute(
      'href',
      /\/media\/movie\/999999$/
    );

    // Page-wide: 已有 is unreachable on the homepage now, so any occurrence is
    // a leak (e.g. a row rendering owned items again).
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(0);
  });

  test('[P1] a requested-but-not-owned title keeps its card AND its 已請求 badge', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);
    await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
      route.fulfill(jsonOk({ owned_ids: [603] }))
    );
    // 未擁有的電影 has an open request — requested is NOT ownership, so the
    // ux3-1-8 filter must not touch it (over-filtering would hide exactly the
    // titles the user is waiting for).
    await page.route(`${ROUTE_API}/requests`, (route: Route) =>
      route.fulfill(jsonOk({ requests: [activeMovieRequest(999999, '未擁有的電影')] }))
    );

    await page.goto('/');

    const block = page.getByTestId('explore-block-b-movies');
    await expect(block.getByTestId('poster-card')).toHaveCount(2);
    await expect(block.getByTestId('availability-badge-requested')).toHaveCount(1);
    await expect(block.getByTestId('availability-badge-owned')).toHaveCount(0);
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
    // Every id in the block payload must be asked about, INCLUDING the ones
    // ux3-1-8 will filter out of the row: the list is hoisted from the content
    // query, never from the rendered cards. Sourcing it from the rendered cards
    // would be circular — an owned title would never be asked about, so it
    // would never be recognised as owned, so it would never be filtered.
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

    // Settle signal: the owned card LEAVING the row (3 → 2) is what proves the
    // POST resolved and was applied. Pre-ux3-1-8 this waited on the 已有 badge,
    // which the row can no longer render.
    await expect(page.getByTestId('explore-block-b-movies').getByTestId('poster-card')).toHaveCount(
      2
    );

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

  test('[P1] 500 from check-owned → every card still renders (the ux3-1-8 filter fails OPEN)', async ({
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
    // ux3-1-8 raised the stakes on this path: unknown ownership must degrade to
    // "show everything", never to "hide everything" — a filter that failed
    // CLOSED would empty the entire discovery tail on a backend hiccup.
    await expect(block.getByTestId('poster-card')).toHaveCount(3);
    await expect(block.getByTestId('explore-block-empty')).toHaveCount(0);

    // No badges because ownership is unknown.
    await expect(page.getByTestId('availability-badge-owned')).toHaveCount(0);
    await expect(page.getByTestId('availability-badge-requested')).toHaveCount(0);
  });
});
