/**
 * PosterCard Hover E2E Tests (bugfix-10-4)
 *
 * Browser-based proof of the in-card hover overlay mechanism. The unit
 * tests in apps/web/src/components/media/PosterCard.spec.tsx assert class
 * presence (hidden, lg:flex, lg:group-hover:opacity-100, …) but RTL cannot
 * fire CSS `:hover`, so they cannot prove the runtime mechanism actually
 * triggers the opacity transition. Sally signed off bugfix-10-4 via a
 * Playwright spike that captured this evidence — the spike was deleted
 * post-sign-off per spike convention. This file is the durable
 * replacement.
 *
 * Coverage gaps closed (vs. unit tests):
 *   - [P0] CSS :hover at lg: viewport actually drives opacity 0 → 1 on
 *     (the center hover-play-overlay was REMOVED by critique R2 — guards below)
 *   - [P0] Top-right badge cluster fades opacity 1 → 0 on hover (collision
 *     strategy from AC #10 — kebab takes over the corner)
 *   - [P0] Bottom-left title/year overlay must NOT be rendered — locks in
 *     the Party Mode 2026-05-08 (Sally + Alexyu) dev-time decision to drop
 *     the MQbvp-spec'd overlay because it duplicates the below-image title
 *     and has legibility issues against varying poster backgrounds. See
 *     PosterCard.tsx:209-213 for the inline rationale. The story doc, the
 *     comparison artifact, and the Rule 22 audit mirror were not updated
 *     to reflect this — this regression guard prevents silent re-add.
 *   - [P1] Mobile viewport (< lg) — `hidden lg:flex` keeps the overlay out
 *     of layout entirely, not just transparent
 *   - [P1] Click on card body navigates to /media/$type/$id (AC #5)
 *   - [P1] Click on the decorative center play icon ALSO navigates —
 *     proves the overlay does not capture clicks (AC #5 + AC #1)
 *
 * Out-of-scope (deferred):
 *   - Visual regression baseline → story 19-4 (Playwright toHaveScreenshot)
 *   - Cross-call-site sweep (library/search/TMDb-detail) → story 19-8
 *   - ESLint enforcement of Rule 21 header → story 19-3
 *
 * Design notes:
 *   - Network-first: every page.route() is installed BEFORE page.goto().
 *   - lg: breakpoint = 1024px (Tailwind default). Desktop Chrome project
 *     defaults to 1280x720 — already lg+. Mobile projects are skipped at
 *     the describe level; the < lg behavior is exercised by an explicit
 *     setViewportSize(375x667) inside the mobile test on Chromium.
 *   - Opacity assertions use `toHaveCSS('opacity', '1' | '0')` which polls
 *     up to expect.timeout (10s) so transition-duration (Tailwind default
 *     ~150ms) settles before the assertion fails.
 *
 * @tags @ui @poster-card @bugfix-10-4
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock payloads — snake_case wire format (frontend's fetchApi runs
// snakeToCamel on response per Rule 18)
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

/** An active (pending) request — what earns a card its 已請求 badge. */
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
});

async function stubHomepageBaseline(page: import('@playwright/test').Page) {
  // ux3-1-8: the hero reads the OWN library and the readout band reads
  // /home-summary. Both MUST be stubbed here — an unstubbed call reaches the
  // real API, and a homepage whose layout depends on live data breaks this
  // file's own network-first determinism rule (the click-coordinate test in
  // particular measures a boundingBox that a late-arriving hero would move).
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, page_size: 20, total_items: 0, total_pages: 0 }))
  );
  await page.route(`${ROUTE_API}/home-summary`, (route: Route) =>
    route.fulfill(
      jsonOk({
        coverage: { status: 'ok', covered: 0, total: 0 },
        processed_today: { status: 'ok', count: 0 },
        attention: { status: 'ok', failed_count: 0 },
        in_flight: { status: 'ok', count: 0 },
      })
    )
  );
  // No open requests by default — the two badge-cluster tests opt in.
  await page.route(`${ROUTE_API}/requests`, (route: Route) =>
    route.fulfill(jsonOk({ requests: [] }))
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
  content: typeof movieContent,
  // ux3-1-8: kept at its default by every caller now — an OWNED id would
  // delete the card from the explore row, and every test in this file needs
  // the card to exist. Left as a parameter so a future non-homepage surface
  // can reuse the helper.
  ownedIds: number[] = []
) {
  await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
    route.fulfill(jsonOk(defaultBlocks))
  );
  await page.route(`${ROUTE_API}/explore-blocks/b-movies/content`, (route: Route) =>
    route.fulfill(jsonOk(content))
  );
  await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
    route.fulfill(jsonOk({ owned_ids: ownedIds }))
  );
}

// =============================================================================
// Tests
// =============================================================================

test.describe('PosterCard Hover @ui @poster-card @bugfix-10-4', () => {
  // Hover overlay is desktop-only by design (lg: breakpoint). The mobile
  // < lg behavior is exercised explicitly via setViewportSize in the
  // mobile-specific test below — running there in mobile projects would
  // duplicate that case. Skip the whole describe on mobile projects.
  // The empty fixtures destructure `({}, testInfo)` is the Playwright pattern
  // for accessing only the second positional `testInfo`.
  // eslint-disable-next-line no-empty-pattern
  test.beforeEach(async ({}, testInfo) => {
    test.skip(
      testInfo.project.name.startsWith('mobile-'),
      'PosterCard hover overlay is lg:-only by design; mobile path is exercised on Chromium via setViewportSize'
    );
  });

  // Critique R2 P1 (2026-08-26) REMOVED the center ▶ overlay: it promised
  // playback the product does not have (the click lands on the detail page).
  // This guard fires if anyone re-introduces it without re-opening that
  // decision — the mirror of the bottom-left-title guard below.
  test('[P0] the lying center play overlay stays out of the DOM, hover included', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();
    await expect(card.getByTestId('hover-play-overlay')).toHaveCount(0);

    await card.hover();
    await expect(card.getByTestId('hover-play-overlay')).toHaveCount(0);
  });

  test('[P0] bottom-left title overlay is NOT rendered — Party Mode 2026-05-08 dev-time decision (regression guard)', async ({
    page,
  }) => {
    // The MQbvp .pen design originally specified a bottom-left title/year
    // overlay. Sally + Alexyu dropped it during bugfix-10-4 dev (Party Mode
    // 2026-05-08) because it duplicates the RusTY below-image title and
    // has legibility issues against varying poster backgrounds. See
    // PosterCard.tsx:209-213 for the inline rationale. This test fires if
    // anyone silently re-introduces the overlay without re-opening that
    // decision — the bugfix-10-4 story doc + comparison artifact + audit
    // mirror were never updated to reflect the drop, so the doc trail
    // alone cannot be trusted as the source of truth here.
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();

    // GIVEN: hover at lg viewport — would surface the overlay if it
    // existed (eliminates the alternative "it's hidden by display: none"
    // false-negative)
    await card.hover();

    // THEN: there is no `hover-title-overlay` element in the card subtree
    await expect(card.getByTestId('hover-title-overlay')).toHaveCount(0);
  });

  test('[P0] hover at lg: viewport fades top-right badge cluster (opacity 1 → 0) — AC #10 collision', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // The cluster needs a badge inside it or the opacity assertions are noise.
    // ux3-1-8 retired the OWNED anchor these two tests used to rely on: an
    // owned title is now deleted from the explore row, so the card it was
    // hovering cannot exist. 已請求 is the badge that IS reachable here —
    // requested-but-not-owned survives the filter by design — and the cluster's
    // hover behaviour is the same whichever badge fills it.
    await stubExploreBlocksWith(page, movieContent);
    await page.route(`${ROUTE_API}/requests`, (route: Route) =>
      route.fulfill(jsonOk({ requests: [activeMovieRequest(603, '駭客任務')] }))
    );

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();

    const anchorBadge = card.getByTestId('availability-badge-requested');
    await expect(anchorBadge).toBeVisible();

    const badgeCluster = anchorBadge.locator('xpath=..');

    // BEFORE hover: cluster fully opaque
    await expect(badgeCluster).toHaveCSS('opacity', '1');

    await card.hover();

    // AFTER hover: cluster fades so kebab can occupy the corner
    await expect(badgeCluster).toHaveCSS('opacity', '0');
  });

  test('[P2] hover at lg: viewport shrinks the top-right badge cluster (scale-95) as it fades — bugfix-10-7 AC #2 kinetic recede', async ({
    page,
  }) => {
    // bugfix-10-7 AC #2: the cluster wrapper picked up `transition-all` +
    // `origin-top-right` + `lg:group-hover:scale-95` so it RECEDES (opacity 1 → 0
    // is covered by the sibling test above) rather than just dissolving in place.
    // Unit tests assert the className; only a real browser can prove the CSS
    // `:hover` actually drives the transform. Browser-pixel verification of the
    // 300 ms timing/easing is deferred to NAS deploy per the story DoD — this is
    // the deterministic regression guard for "scale-95 fires on hover".
    // NOTE: Tailwind v4 compiles `scale-95` to the standalone CSS `scale`
    // property (`scale: 95% 95%`), NOT a `transform: matrix(...)`. Assert on
    // `scale` — `getComputedStyle().transform` stays `none` for `scale-*` in v4.
    await stubHomepageBaseline(page);
    // 已請求 anchor, same reason as the sibling opacity test above (ux3-1-8).
    await stubExploreBlocksWith(page, movieContent);
    await page.route(`${ROUTE_API}/requests`, (route: Route) =>
      route.fulfill(jsonOk({ requests: [activeMovieRequest(603, '駭客任務')] }))
    );

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();

    const anchorBadge = card.getByTestId('availability-badge-requested');
    await expect(anchorBadge).toBeVisible();
    const badgeCluster = anchorBadge.locator('xpath=..');

    // BEFORE hover: no scale applied — full size
    await expect(badgeCluster).toHaveCSS('scale', 'none');

    await card.hover();

    // AFTER hover: `scale-95` → computed `scale: 0.95` once the 300 ms
    // transition settles (toHaveCSS polls up to expect.timeout).
    await expect(badgeCluster).toHaveCSS('scale', '0.95');
  });

  test('[P1] mobile viewport (375x667) — no play overlay there either (AC #6 heir)', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();
    await expect(card.getByTestId('hover-play-overlay')).toHaveCount(0);

    await card.hover();
    await expect(card.getByTestId('hover-play-overlay')).toHaveCount(0);
  });

  test('[P1] click on card body navigates to /media/movie/$id (AC #5)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    // The detail page will fire its own data fetches once routed. Stub
    // them so the test does not race a real network call. Status of the
    // destination is not under test — only the URL change is.
    await page.route(`${ROUTE_API}/movies/**`, (route: Route) =>
      route.fulfill(jsonOk({ id: 603, title: '駭客任務' }))
    );
    await page.route(`${ROUTE_API}/tmdb/movies/**`, (route: Route) =>
      route.fulfill(jsonOk({ id: 603, title: '駭客任務' }))
    );

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();

    // WHEN: user clicks on the card image area (NOT the play overlay
    // center). Use the card image testid if present; fall back to a
    // top-of-card click via boundingBox so we land on the image, not the
    // below-image title row or the overlay.
    const cardBox = await card.boundingBox();
    expect(cardBox).not.toBeNull();
    // Click in the upper-left quadrant of the image — far from the
    // center play overlay and away from kebab/checkbox corners.
    await page.mouse.click(cardBox!.x + cardBox!.width * 0.25, cardBox!.y + cardBox!.height * 0.15);

    // THEN: TanStack Router navigates to the movie detail route
    await page.waitForURL(/\/media\/movie\/603(\?|$)/);
  });

  test('[P1] hovered poster center still navigates — nothing intercepts the card link (AC #1 + AC #5 heir)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubExploreBlocksWith(page, movieContent);

    await page.route(`${ROUTE_API}/movies/**`, (route: Route) =>
      route.fulfill(jsonOk({ id: 603, title: '駭客任務' }))
    );
    await page.route(`${ROUTE_API}/tmdb/movies/**`, (route: Route) =>
      route.fulfill(jsonOk({ id: 603, title: '駭客任務' }))
    );

    await page.goto('/');

    const card = page.getByTestId('poster-card').first();
    await expect(card).toBeVisible();

    // Click dead-center of the poster (where the removed ▶ used to sit)
    // while hovered — the card <Link> must receive it.
    await card.hover();
    await card.click();

    await page.waitForURL(/\/media\/movie\/603(\?|$)/);
  });
});
