/**
 * Homepage Layout E2E Tests (Story 10-5)
 *
 * Story: _bmad-output/implementation-artifacts/10-5-homepage-layout-responsive.md
 *
 * DEV's Vitest suite covers structural class tokens and mocked hook returns
 * inside jsdom. These real-browser scenarios close the gaps that jsdom can't
 * observe:
 *
 *   - AC #1: section order as rendered by the real router/AppShell (not
 *     stubbed child components)
 *   - AC #2: Intersection-Observer lazy-load is a NETWORK-level assertion —
 *     the 3rd block's content endpoint must NOT be hit until the user
 *     scrolls it into view. Vitest can't observe this because its
 *     IntersectionObserver is an inert stub.
 *   - AC #2: route-loader prefetch on Link hover — fires the own-library
 *     recently-added request BEFORE navigation (requires a real router + a
 *     real Link to hover), cannot be exercised from an isolated component
 *     spec. (ux3-1-8 moved that prefetch off TMDb trending onto
 *     library/recent, the hero's new source.)
 *   - AC #3: hero height is a Tailwind arbitrary class (`h-[250px]` /
 *     `md:h-[400px]`) — the only way to verify it compiles and yields the
 *     right pixel height is a real browser at viewport 390/1440.
 *   - AC #3: explore blocks scroll horizontally on mobile — again, a
 *     real-DOM `scrollWidth > clientWidth` check, not a class assertion.
 *   - AC #4: per-block skeleton is visible WHILE the content request is
 *     inflight (not just on the first mount). Deterministic via a deferred
 *     route fulfillment.
 *   - AC #5: panels truly removed from the DOM when data is empty — jsdom
 *     tests stub the panels entirely, so this proves the real branch in
 *     production code paths.
 *
 * Network-first: every route is intercepted BEFORE page.goto() so the
 * homepage never touches live TMDb / local backend services.
 *
 * ux3-1-8 (Home v3 identity flip) rewrites what "hero" means here: the hero
 * is now the OWN library's newest backdrop-bearing items (GET
 * /library/recent?limit=20), it is STATIC (no autoplay/pause), and it sits
 * ABOVE the own-content row while the TMDb explore rows retreat to the tail.
 * Everything in this file that used to be phrased in TMDb-trending terms is
 * re-anchored on that source.
 *
 * @tags @ui @homepage @story-10-5 @story-ux3-1-8
 */

import { test, expect } from '../support/fixtures';
import type { Route } from '@playwright/test';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data — snake_case wire format (frontend fetchApi runs snakeToCamel)
// =============================================================================

const mockQBConfig = {
  host: 'http://localhost:8080',
  username: 'admin',
  basePath: '',
  configured: true,
};

// ux3-1-8: the hero's ONLY source. Two items carry a backdrop (hero-eligible,
// so the dots pill renders too); the third has none — it feeds the 最近新增
// row but must never reach the hero. Wire shape = LibraryListResponse.
const mockLibraryRecent = {
  items: [
    {
      type: 'movie',
      movie: {
        id: '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63',
        title: '駭客任務',
        release_date: '1999-03-31',
        genres: ['動作', '科幻'],
        poster_path: '/matrix.jpg',
        backdrop_path: '/matrix-bg.jpg',
        vote_average: 8.7,
        parse_status: 'parsed',
        created_at: '2026-08-20T00:00:00Z',
        updated_at: '2026-08-20T00:00:00Z',
      },
    },
    {
      type: 'series',
      series: {
        id: '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57',
        title: '黑鏡',
        first_air_date: '2011-12-04',
        genres: ['劇情', '科幻'],
        poster_path: '/black-mirror.jpg',
        backdrop_path: '/black-mirror-bg.jpg',
        vote_average: 8.3,
        parse_status: 'parsed',
        created_at: '2026-08-19T00:00:00Z',
        updated_at: '2026-08-19T00:00:00Z',
      },
    },
    {
      type: 'movie',
      movie: {
        id: 'c1d7f4b2-9a3e-4c6d-8b0f-5e2a7c9d1b48',
        title: '沒有劇照的片',
        release_date: '2020-01-01',
        genres: ['紀錄'],
        poster_path: '/no-backdrop.jpg',
        parse_status: 'parsed',
        created_at: '2026-08-18T00:00:00Z',
        updated_at: '2026-08-18T00:00:00Z',
      },
    },
  ],
  page: 1,
  page_size: 20,
  total_items: 3,
  total_pages: 1,
};

const emptyLibraryRecent = {
  items: [],
  page: 1,
  page_size: 20,
  total_items: 0,
  total_pages: 0,
};

// Home v3 readout band (ux3-1-7) — the first section of the page. Stubbed so
// the band is a deterministic DOM node in the order assertion instead of a
// fail-soft blank that would silently pass any ordering.
const mockHomeSummary = {
  coverage: { status: 'ok', covered: 2, total: 3 },
  processed_today: { status: 'ok', count: 1 },
  attention: { status: 'ok', failed_count: 0 },
  in_flight: { status: 'ok', count: 0 },
};

const mockBlocks = {
  blocks: [
    {
      id: 'b1',
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
      id: 'b2',
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
    {
      id: 'b3',
      name: '動作片',
      content_type: 'movie',
      genre_ids: '28',
      language: '',
      region: '',
      sort_by: 'popularity.desc',
      max_items: 20,
      sort_order: 2,
      created_at: '2026-04-15T00:00:00Z',
      updated_at: '2026-04-15T00:00:00Z',
    },
    {
      id: 'b4',
      name: '劇情片',
      content_type: 'movie',
      genre_ids: '18',
      language: '',
      region: '',
      sort_by: 'popularity.desc',
      max_items: 20,
      sort_order: 3,
      created_at: '2026-04-15T00:00:00Z',
      updated_at: '2026-04-15T00:00:00Z',
    },
  ],
};

const blockContent = (id: string) => ({
  block_id: id,
  content_type: 'movie',
  movies: [
    {
      id: 1000 + Number(id.slice(1)),
      title: `${id} 片 A`,
      original_title: `${id} Movie A`,
      overview: '',
      release_date: '2024-01-01',
      poster_path: `/poster-${id}.jpg`,
      backdrop_path: null,
      vote_average: 8,
      vote_count: 100,
      genre_ids: [28],
    },
  ],
  total_items: 1,
});

const mockRecentMedia = [
  {
    id: 'movie-1',
    title: '測試電影',
    year: 2024,
    posterUrl: '',
    mediaType: 'movie',
    justAdded: true,
    addedAt: '2026-04-17T10:00:00Z',
  },
];

const mockDownloads = {
  items: [
    {
      hash: 'abc123',
      name: '正在下載的影片',
      size: 4294967296,
      progress: 0.5,
      downloadSpeed: 1048576,
      uploadSpeed: 0,
      eta: 300,
      status: 'downloading',
      addedOn: '2026-04-17T10:00:00Z',
      seeds: 5,
      peers: 3,
      downloaded: 2147483648,
      uploaded: 0,
      ratio: 0,
      savePath: '/downloads',
    },
  ],
  page: 1,
  pageSize: 100,
  totalItems: 1,
  totalPages: 1,
};

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

// =============================================================================
// Baseline stub — all homepage-side API endpoints. Tests compose this then
// override individual routes for scenario-specific behavior.
// =============================================================================

async function stubHomepageBaseline(page: import('@playwright/test').Page) {
  // ux3-1-8: own-library recently-added feeds BOTH the hero and the 最近新增
  // row (one query, deduped). Uses jsonOk — the frontend's fetchApi unwraps
  // `{success,data}` and throws when `.success` is absent.
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk(mockLibraryRecent))
  );
  // Home v3 readout band (ux3-1-7) — first section of the page.
  await page.route(`${ROUTE_API}/home-summary`, (route: Route) =>
    route.fulfill(jsonOk(mockHomeSummary))
  );
  // TMDb image CDN — the hero's backdrops are own-library rows but the paths
  // are still TMDb-hosted. Stub with a 1x1 PNG so `<img onError>` never fires
  // and `imageBroken` never unmounts the backdrop in CI (where image.tmdb.org
  // is unreachable).
  await page.route(/image\.tmdb\.org\/.*/, (route: Route) =>
    route.fulfill({
      status: 200,
      contentType: 'image/png',
      body: Buffer.from(
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
        'base64'
      ),
    })
  );
  // Explore blocks (list + content for all 4 blocks)
  await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
    route.fulfill(jsonOk(mockBlocks))
  );
  for (const id of ['b1', 'b2', 'b3', 'b4']) {
    await page.route(`${ROUTE_API}/explore-blocks/${id}/content`, (route: Route) =>
      route.fulfill(jsonOk(blockContent(id)))
    );
  }
  // Availability hoisted lookup (Story 10-4 contract)
  await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
    route.fulfill(jsonOk({ owned_tmdb_ids: [], requested_tmdb_ids: [] }))
  );
  // Recent media / downloads / QB / health
  await page.route(`${ROUTE_API}/media/recent*`, (route: Route) =>
    route.fulfill(jsonOk(mockRecentMedia))
  );
  await page.route(`${ROUTE_API}/downloads*`, (route: Route) =>
    route.fulfill(jsonOk(mockDownloads))
  );
  await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
    route.fulfill(jsonOk(mockQBConfig))
  );
  await page.route(`${ROUTE_API}/health/services*`, (route: Route) =>
    route.fulfill(jsonOk({ services: [] }))
  );
}

// =============================================================================
// AC #1 — Section order in the real DOM (readout → hero → own row → explore)
// =============================================================================

test.describe('Homepage section order @ui @homepage @story-10-5 @story-ux3-1-8', () => {
  // ux3-1-8 (Home v3): D3's own-above-external law still holds and gets
  // STRONGER — the hero is own content now, so readout band + hero + 最近新增
  // are all yours and the TMDb explore rows are the tail. The v2 order
  // (own-content ABOVE hero) is the regression this test catches.
  test('[P0] ux3-1-8 — 讀數帶 → 自有 Hero → 最近新增 → TMDb 探索尾巴（依此 DOM 順序）', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);

    await page.goto('/');
    await expect(page.getByTestId('home-v2-root')).toBeVisible();
    await expect(page.getByTestId('home-readout-band')).toBeVisible();
    await expect(page.getByTestId('hero-banner')).toBeVisible();
    await expect(page.getByTestId('home-own-content')).toBeVisible();
    await expect(page.getByTestId('explore-blocks-list')).toBeVisible();

    // ONE querySelectorAll, not one query per selector: querySelectorAll
    // returns document order, so the array is the real DOM order. (Mapping
    // over a selector list — the pre-ux3-1-8 shape of this test — echoed the
    // selector array back and would have passed on ANY ordering.)
    const order = await page
      .getByTestId('home-v2-root')
      .evaluate((el) =>
        Array.from(
          el.querySelectorAll(
            '[data-testid="home-readout-band"],[data-testid="hero-banner"],[data-testid="home-own-content"],[data-testid="explore-blocks-list"]'
          )
        ).map((n) => n.getAttribute('data-testid'))
      );
    expect(order).toEqual([
      'home-readout-band',
      'hero-banner',
      'home-own-content',
      'explore-blocks-list',
    ]);

    // The hero on this page is the OWN-library one: 最新入庫 is copy no TMDb
    // trending hero ever carried, so a revert to the trending source fails here
    // even if the section order survived. `.first()` = the active slide (every
    // slide carries its own eyebrow).
    await expect(page.getByTestId('hero-banner-eyebrow').first()).toHaveText('最新入庫');

    // D3 guardrail #3 (ux3-1-4): Epic-4 dashboard remnants stay off the home.
    await expect(page.getByTestId('download-panel')).toHaveCount(0);
    await expect(page.getByTestId('recent-media-panel')).toHaveCount(0);
  });

  test('[P0] ux3-1-8 — 首頁不再向 TMDb trending 要任何資料', async ({ page }) => {
    await stubHomepageBaseline(page);
    // The retired hero source. Counting (and answering) it rather than leaving
    // it unrouted: an un-stubbed hit would fall through to the real backend and
    // pass silently, which is exactly the regression — a TMDb hero creeping
    // back in — this guard exists to catch.
    // RegExp, not a glob: `*` stops at `/`, so `**/tmdb/trending*` would sail
    // straight past `/tmdb/trending/movies` — the very URL this guard is for.
    let trendingHits = 0;
    await page.route(/\/api\/v1\/tmdb\/trending/, (route: Route) => {
      trendingHits += 1;
      return route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }));
    });

    await page.goto('/');
    await expect(page.getByTestId('hero-banner')).toBeVisible();
    await expect(page.getByTestId('explore-blocks-list')).toBeVisible();
    await page.waitForLoadState('networkidle');

    expect(trendingHits).toBe(0);
  });
});

// =============================================================================
// AC #3 — Responsive hero heights (real pixel measurement at 390 / 1440)
// =============================================================================

test.describe('Homepage responsive hero height @ui @homepage @story-10-5', () => {
  // ux3-1-8: these only measure anything while the hero HAS content, and the
  // hero's content is now own-library rows with a backdrop_path — the baseline
  // stub's mockLibraryRecent is what keeps them from silently measuring nothing.
  test('[P0] AC #3 — hero is 250px tall at mobile (390×844 iPhone)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');

    const hero = page.getByTestId('hero-banner');
    await expect(hero).toBeVisible();
    const height = await hero.evaluate((el) => Math.round(el.getBoundingClientRect().height));
    // `h-[250px]` is a Tailwind arbitrary class — this is the only way to
    // verify it actually resolved to 250 pixels in the built CSS.
    expect(height).toBe(250);
  });

  test('[P0] AC #3 — hero is 400px tall at desktop (1440×900)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/');

    const hero = page.getByTestId('hero-banner');
    await expect(hero).toBeVisible();
    const height = await hero.evaluate((el) => Math.round(el.getBoundingClientRect().height));
    // `md:h-[400px]` (md = ≥768px).
    expect(height).toBe(400);
  });

  test('[P1] AC #3 — explore block scroller is horizontally scrollable on mobile', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // Override content to return enough cards that the row overflows.
    await page.route(`${ROUTE_API}/explore-blocks/b1/content`, (route: Route) =>
      route.fulfill(
        jsonOk({
          block_id: 'b1',
          content_type: 'movie',
          movies: Array.from({ length: 10 }, (_, i) => ({
            id: 2000 + i,
            title: `電影 ${i}`,
            original_title: `Movie ${i}`,
            overview: '',
            release_date: '2024-01-01',
            poster_path: `/m-${i}.jpg`,
            backdrop_path: null,
            vote_average: 8,
            vote_count: 100,
            genre_ids: [28],
          })),
          total_items: 10,
        })
      )
    );

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');

    const scroller = page.getByTestId('explore-block-b1').getByTestId('explore-block-scroller');
    await expect(scroller).toBeVisible();
    const { scrollWidth, clientWidth } = await scroller.evaluate((el) => ({
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
    }));
    expect(scrollWidth).toBeGreaterThan(clientWidth);
  });
});

// =============================================================================
// AC #2 — Intersection-Observer lazy-load (network-level proof)
// =============================================================================

test.describe('Homepage lazy-load @ui @homepage @story-10-5', () => {
  test('[P0] AC #2 — below-the-fold block content is NOT fetched until scrolled into view', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // Track hits per block id.
    const hits: Record<string, number> = { b1: 0, b2: 0, b3: 0, b4: 0 };
    for (const id of ['b1', 'b2', 'b3', 'b4']) {
      await page.route(`${ROUTE_API}/explore-blocks/${id}/content`, (route: Route) => {
        hits[id] += 1;
        return route.fulfill(jsonOk(blockContent(id)));
      });
    }

    // Give below-the-fold blocks enough height to sit outside the initial
    // viewport. Narrow height amplifies the gap.
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/');

    // Wait for above-the-fold eagers (index < EAGER_BLOCK_COUNT=2) to settle.
    await expect(page.getByTestId('explore-block-b1')).toBeVisible();
    // Tiny settle so any racing fetches actually fire.
    await page.waitForLoadState('networkidle');

    // Eager blocks (index 0, 1) hit exactly once.
    expect(hits.b1).toBe(1);
    expect(hits.b2).toBe(1);
    // Lazy blocks (index 2, 3) have NOT been requested yet — they render as
    // skeletons while the observer waits for intersection.
    expect(hits.b3).toBe(0);
    expect(hits.b4).toBe(0);

    // Scrolling b3 into view triggers its content fetch.
    await page.getByTestId('explore-block-b3').scrollIntoViewIfNeeded();
    await expect.poll(() => hits.b3).toBe(1);
    // (b4 behavior is not asserted — rootMargin + variable block height make
    // it non-deterministic. b3's fetch-on-scroll is the authoritative proof.)
  });

  test('[P1] AC #2 — route loader prefetches the own-library hero BEFORE navigation when a Link to "/" is hovered', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // ux3-1-8: the index loader seeds libraryKeys.recent(20) now, not the
    // retired TMDb trending query. Watching the wrong endpoint here would let
    // the prefetch rot away unnoticed (the page still works, it just gets
    // slower), so the counter follows the hero's real source.
    let recentHits = 0;
    await page.route(`${ROUTE_API}/library/recent*`, (route: Route) => {
      recentHits += 1;
      return route.fulfill(jsonOk(mockLibraryRecent));
    });

    // Start on a different route so the homepage loader has not yet run —
    // /library does not mount useRecentlyAdded, so the recent query is cold
    // and the prefetch cannot be masked by a fresh (staleTime 30s) cache entry.
    await page.goto('/library');
    await page.waitForLoadState('networkidle');
    const baselineHits = recentHits;

    // AppShell houses a logo Link to "/" (text content "vido"). Hover fires
    // the router's intent-preload, which runs the index route loader
    // (prefetchQuery for the recently-added hero source). The request should
    // appear before we navigate.
    // v2 shell: the sidebar logo Link to "/" (accessible name may include the
    // wordmark styling) — match any link pointing home.
    const homeLink = page.locator('a[href="/"]').first();
    await expect(homeLink).toBeVisible();
    await homeLink.hover();

    await expect
      .poll(() => recentHits, {
        message: 'Route loader prefetch should fire library/recent on Link hover',
        timeout: 3000,
      })
      .toBeGreaterThan(baselineHits);
  });
});

// =============================================================================
// AC #4 — Per-block skeleton visible while content is inflight
// =============================================================================

test.describe('Homepage per-block skeleton @ui @homepage @story-10-5', () => {
  test('[P0] AC #4 — per-block skeleton renders while each block content is inflight, then swaps to cards', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // Defer b1 content so the test can observe its skeleton.
    let releaseB1: (() => void) | null = null;
    const b1Ready = new Promise<void>((resolve) => {
      releaseB1 = resolve;
    });
    await page.route(`${ROUTE_API}/explore-blocks/b1/content`, async (route: Route) => {
      await b1Ready;
      await route.fulfill(jsonOk(blockContent('b1')));
    });

    await page.goto('/');
    const block = page.getByTestId('explore-block-b1');
    await expect(block).toBeVisible();
    // While content is deferred, a non-zero number of skeletons render inside
    // this specific block (ExploreBlockSkeleton emits 6 by default).
    const skeletonsBefore = await block.getByTestId('explore-block-skeleton').count();
    expect(skeletonsBefore).toBeGreaterThan(0);

    // Release — content arrives, skeletons swap for real poster cards.
    releaseB1?.();
    await expect(block.getByTestId('poster-card').first()).toBeVisible();
    await expect(block.getByTestId('explore-block-skeleton')).toHaveCount(0);
  });
});

// =============================================================================
// AC #5 — Empty-section hide behavior in a real browser
// =============================================================================

test.describe('Homepage empty-section hide @ui @homepage @story-10-5 @story-ux3-1-8', () => {
  // ux3-1-8 rewrites WHY the hero can be absent: it is no longer "TMDb gave us
  // nothing" but "no own item can dress a hero". An empty library therefore
  // takes the hero with it, while the own-content zone stays (最近新增 shows its
  // quiet 尚無最近新增 note instead of vanishing, ux3-1-2 H5 sparse state) and
  // the dashboard panels are always absent (D3 guardrail, ux3-1-4).
  test('[P0] 空片庫 + 空 blocks：Hero/Explore 缺席，最近新增留下清淡提示，無 dashboard 殘骸', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [] }))
    );
    await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
      route.fulfill(jsonOk(emptyLibraryRecent))
    );

    await page.goto('/');
    await expect(page.getByTestId('home-v2-root')).toBeVisible();

    // The sparse note proves the recent query RESOLVED — without it, asserting
    // hero-banner count 0 would also pass while the hero was still a skeleton.
    await expect(page.getByTestId('home-recent-empty')).toBeVisible();
    await expect(page.getByTestId('hero-banner-skeleton')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
    await expect(page.getByTestId('explore-blocks-list')).toHaveCount(0);
    await expect(page.getByTestId('download-panel')).toHaveCount(0);
    await expect(page.getByTestId('recent-media-panel')).toHaveCount(0);
    await expect(page.getByText('目前沒有下載任務')).toHaveCount(0);
  });

  // Replaces the retired「TMDb trending 空 → Hero 隱藏」premise with the rule
  // that actually governs absence now. This is the honest half of the identity
  // flip: fresh items with no artwork must NOT produce an empty hero frame,
  // and must NOT take the 最近新增 row down with them.
  test('[P0] ux3-1-8 — 最近入庫都沒有 backdrop：Hero 缺席，但最近新增照常列出', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
      route.fulfill(
        jsonOk({
          ...emptyLibraryRecent,
          items: [mockLibraryRecent.items[2]],
          total_items: 1,
          total_pages: 1,
        })
      )
    );

    await page.goto('/');
    await expect(page.getByTestId('home-v2-root')).toBeVisible();

    // The row rendering the very item the hero rejected = the query resolved,
    // so the hero's absence below is a decision, not a pending state.
    await expect(page.getByTestId('home-recent-row')).toBeVisible();
    await expect(page.getByTestId('home-recent-row').getByText('沒有劇照的片')).toBeVisible();
    await expect(page.getByTestId('hero-banner-skeleton')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
  });

  test('[P1] populated downloads still render NO download panel on home (D3)', async ({ page }) => {
    await stubHomepageBaseline(page);

    await page.goto('/');
    await expect(page.getByTestId('home-v2-root')).toBeVisible();
    await expect(page.getByTestId('download-panel')).toHaveCount(0);
  });
});
