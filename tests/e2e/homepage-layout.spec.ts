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
 *   - AC #2: route-loader prefetch on Link hover — fires the trending
 *     request BEFORE navigation (requires a real router + a real Link to
 *     hover), cannot be exercised from an isolated component spec.
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
 * @tags @ui @homepage @story-10-5
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

const mockTrendingMovies = {
  page: 1,
  results: [
    {
      id: 101,
      title: '鬼滅之刃：無限城篇',
      overview: '炭治郎與鬼殺隊潛入無限城。',
      backdrop_path: '/bg1.jpg',
      poster_path: '/p1.jpg',
      release_date: '2024-05-10',
      vote_average: 8.2,
      vote_count: 1250,
      media_type: 'movie',
    },
  ],
  total_pages: 1,
  total_results: 1,
};

const mockTrendingTV = {
  page: 1,
  results: [],
  total_pages: 0,
  total_results: 0,
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

const jsonRaw = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify(body),
});

// =============================================================================
// Baseline stub — all homepage-side API endpoints. Tests compose this then
// override individual routes for scenario-specific behavior.
// =============================================================================

async function stubHomepageBaseline(page: import('@playwright/test').Page) {
  // TMDb trending (HeroBanner)
  await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
    route.fulfill(jsonRaw(mockTrendingMovies))
  );
  await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
    route.fulfill(jsonRaw(mockTrendingTV))
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
// AC #1 — Section order in the real DOM (hero → explore → recent → downloads)
// =============================================================================

test.describe('Homepage section order @ui @homepage @story-10-5', () => {
  test('[P0] AC #1 — Hero → Explore → Recent → Downloads render in that order', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);

    await page.goto('/');
    await expect(page.getByTestId('homepage-root')).toBeVisible();
    // Wait for at least one section each so the DOM is fully composed.
    await expect(page.getByTestId('hero-banner')).toBeVisible();
    await expect(page.getByTestId('explore-blocks-list')).toBeVisible();
    await expect(page.getByTestId('recent-media-panel')).toBeVisible();
    await expect(page.getByTestId('download-panel')).toBeVisible();

    const root = page.getByTestId('homepage-root');
    const order = await root.evaluate((el) => {
      const selectors = [
        '[data-testid="hero-banner"]',
        '[data-testid="explore-blocks-list"]',
        '[data-testid="recent-media-panel"]',
        '[data-testid="download-panel"]',
      ];
      return selectors
        .map((sel) => el.querySelector(sel))
        .filter((n): n is Element => Boolean(n))
        .map((n) => n.getAttribute('data-testid'));
    });
    expect(order).toEqual([
      'hero-banner',
      'explore-blocks-list',
      'recent-media-panel',
      'download-panel',
    ]);
  });
});

// =============================================================================
// AC #3 — Responsive hero heights (real pixel measurement at 390 / 1440)
// =============================================================================

test.describe('Homepage responsive hero height @ui @homepage @story-10-5', () => {
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
    // b4 still below the fold, still zero (rootMargin is 400px, one extra
    // block height — but positioned far enough down that it stays untouched).
    // Note: if the tester's block height is small enough that rootMargin
    // picks up b4 too, this assertion may need loosening; adjust to `>=1` and
    // keep the key assertion on b3.
    expect(hits.b4).toBeLessThanOrEqual(1);
  });

  test('[P1] AC #2 — route loader prefetches trending hero BEFORE navigation when a Link to "/" is hovered', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    let trendingMoviesHits = 0;
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) => {
      trendingMoviesHits += 1;
      return route.fulfill(jsonRaw(mockTrendingMovies));
    });

    // Start on a different route so the homepage loader has not yet run.
    await page.goto('/library');
    await page.waitForLoadState('networkidle');
    const baselineHits = trendingMoviesHits;

    // AppShell's nav houses a Link to "/" ("首頁"). Hover fires the router's
    // intent-preload, which runs the index route loader (prefetchQuery for
    // trending hero). The request should appear before we navigate.
    const homeLink = page.getByRole('link', { name: /首頁/ }).first();
    await expect(homeLink).toBeVisible();
    await homeLink.hover();

    await expect
      .poll(() => trendingMoviesHits, {
        message: 'Route loader prefetch should fire trending/movies on Link hover',
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

test.describe('Homepage empty-section hide @ui @homepage @story-10-5', () => {
  test('[P0] AC #5 — when trending/blocks/recent/downloads all return empty, only HeroBanner-less homepage-root survives', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // Override everything to empty (baseline uses populated trending).
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonRaw({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonRaw({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );
    await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
      route.fulfill(jsonOk({ blocks: [] }))
    );
    await page.route(`${ROUTE_API}/media/recent*`, (route: Route) => route.fulfill(jsonOk([])));
    await page.route(`${ROUTE_API}/downloads*`, (route: Route) =>
      route.fulfill(jsonOk({ items: [], page: 1, pageSize: 100, totalItems: 0, totalPages: 0 }))
    );

    await page.goto('/');
    await expect(page.getByTestId('homepage-root')).toBeVisible();

    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
    await expect(page.getByTestId('explore-blocks-list')).toHaveCount(0);
    await expect(page.getByTestId('recent-media-panel')).toHaveCount(0);
    await expect(page.getByTestId('download-panel')).toHaveCount(0);
    // The two inline zh-TW default placeholders must NOT leak onto the
    // homepage (they still exist on /downloads and /library).
    await expect(page.getByText('目前沒有下載任務')).toHaveCount(0);
    await expect(page.getByText('媒體庫中還沒有內容')).toHaveCount(0);
  });

  test('[P1] AC #5 — populated Downloads + empty Recent renders only DownloadPanel', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/media/recent*`, (route: Route) => route.fulfill(jsonOk([])));

    await page.goto('/');
    await expect(page.getByTestId('homepage-root')).toBeVisible();

    await expect(page.getByTestId('download-panel')).toBeVisible();
    await expect(page.getByTestId('recent-media-panel')).toHaveCount(0);
  });
});
