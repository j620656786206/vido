/**
 * Hero Banner E2E Tests (Story 10-2)
 *
 * Browser-based tests for the homepage HeroBanner carousel and TrailerModal.
 * Uses route interception for deterministic tests (no live TMDb calls).
 *
 * Coverage:
 *   - AC#1 display backdrop/title/year/rating/overview
 *   - AC#2 auto-rotation via page.clock (virtual time, no hard waits)
 *   - AC#3 click-through to media detail page
 *   - AC#4 mobile responsive layout
 *   - AC#5 graceful hide on empty/error
 *   - AC#6 trailer modal open + Escape close
 *
 * Design notes:
 *   - `page.clock.install()` is set BEFORE `page.goto()` so the component's
 *     `setInterval` registers against virtual time. `fastForward(ms)` advances
 *     time deterministically without wall-clock waits (avoids the flaky
 *     `waitForTimeout(8000)` anti-pattern).
 *   - Mocked API payloads are snake_case at the wire level; the frontend
 *     `fetchApi` runs `snakeToCamel` on `data.data` (see services/tmdb.ts).
 *
 * @tags @ui @hero-banner @story-10-2
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data — snake_case wire format (transformed by snakeToCamel in fetchApi)
// =============================================================================

const mockTrendingMovies = {
  page: 1,
  results: [
    {
      id: 101,
      title: '鬼滅之刃：無限城篇',
      overview: '炭治郎與鬼殺隊潛入無限城，與上弦之鬼展開最終決戰。',
      backdrop_path: '/movie1.jpg',
      poster_path: '/movie1p.jpg',
      release_date: '2024-05-10',
      vote_average: 8.2,
      vote_count: 1250,
      media_type: 'movie',
    },
    {
      id: 102,
      title: '沙丘：第二部',
      overview: '保羅·亞崔迪與弗瑞曼人聯手對抗哈肯能家族。',
      backdrop_path: '/movie2.jpg',
      poster_path: '/movie2p.jpg',
      release_date: '2023-11-20',
      vote_average: 7.5,
      vote_count: 890,
      media_type: 'movie',
    },
  ],
  total_pages: 1,
  total_results: 2,
};

const mockTrendingTV = {
  page: 1,
  results: [
    {
      id: 201,
      name: '進擊的巨人最終季',
      overview: '艾連發動地鳴，世界陷入毀滅危機。',
      backdrop_path: '/tv1.jpg',
      poster_path: '/tv1p.jpg',
      first_air_date: '2024-01-15',
      vote_average: 9.0,
      vote_count: 2100,
      media_type: 'tv',
    },
  ],
  total_pages: 1,
  total_results: 1,
};

const mockVideosWithTrailer = {
  id: 101,
  results: [
    {
      id: 'v-old',
      iso_639_1: 'en',
      iso_3166_1: 'US',
      name: 'Old Teaser',
      key: 'teaser_xyz_00',
      site: 'YouTube',
      size: 1080,
      type: 'Teaser',
      official: false,
      published_at: '2024-01-01T00:00:00.000Z',
    },
    {
      id: 'v-official',
      iso_639_1: 'en',
      iso_3166_1: 'US',
      name: 'Official Trailer',
      key: 'official_abc_01',
      site: 'YouTube',
      size: 1080,
      type: 'Trailer',
      official: true,
      published_at: '2024-04-01T00:00:00.000Z',
    },
  ],
};

const mockVideosEmpty = { id: 999, results: [] };

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

const jsonErr = (status: number, code: string, message: string) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify({ success: false, error: { code, message } }),
});

/**
 * Stubs the non-HeroBanner homepage dependencies (downloads, recent media,
 * qBittorrent settings, health services) so the `/` route loads cleanly
 * under test regardless of HeroBanner state.
 */
async function stubHomepageBaseline(page: import('@playwright/test').Page) {
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
  // TMDb image CDN — stub with a 1x1 PNG so `<img onError>` never fires and
  // `imageBroken` never unmounts the backdrop in CI (image.tmdb.org
  // unreachable on runners).
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
}

// =============================================================================
// AC#1 + AC#5: Display & Graceful Hide
// =============================================================================

test.describe('HeroBanner Display @ui @hero-banner @story-10-2', () => {
  test('[P0] should display banner with title, year, rating, overview when trending data available (AC1)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingTV))
    );

    await page.goto('/');

    const banner = page.getByTestId('hero-banner');
    await expect(banner).toBeVisible();

    // First slide (movie #101) is active on mount.
    const firstSlide = banner.locator('[data-testid="hero-banner-slide"][data-active="true"]');
    await expect(firstSlide).toHaveCount(1);
    await expect(firstSlide.getByTestId('hero-banner-title')).toHaveText('鬼滅之刃：無限城篇');
    await expect(firstSlide.getByTestId('hero-banner-year')).toHaveText('2024');
    await expect(firstSlide.getByTestId('hero-banner-rating')).toContainText('8.2');
    await expect(firstSlide.getByTestId('hero-banner-overview')).toContainText('炭治郎');
    // H1 fix: w1280 is the safe baseline; srcset upgrades to original on
    // wide viewports and downgrades to w780 on mobile.
    await expect(firstSlide.getByTestId('hero-banner-backdrop')).toHaveAttribute(
      'src',
      /\/t\/p\/w1280\/movie1\.jpg$/
    );
    await expect(firstSlide.getByTestId('hero-banner-backdrop')).toHaveAttribute(
      'srcset',
      /\/t\/p\/w780\/movie1\.jpg 780w/
    );
  });

  test('[P0] should hide banner section when trending API returns empty (AC5)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );

    await page.goto('/');

    // Homepage root still mounts; banner section and skeleton are both absent.
    // Story 10-5 replaced the dashboard-layout grid with the homepage-root
    // vertical flex stack.
    await expect(page.getByTestId('homepage-root')).toBeVisible();
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner-skeleton')).toHaveCount(0);
  });

  test('[P0] should hide banner section when trending API fails (AC5)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonErr(500, 'TMDB_ERROR', 'upstream failure'))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonErr(500, 'TMDB_ERROR', 'upstream failure'))
    );

    await page.goto('/');

    // Homepage still renders independently (AC5 graceful degradation); Story
    // 10-5 swapped `dashboard-layout` for `homepage-root`.
    await expect(page.getByTestId('homepage-root')).toBeVisible();
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
  });

  test('[P1] should filter out items missing backdrop_path (render guard)', async ({ page }) => {
    await stubHomepageBaseline(page);
    const missingBackdrop = {
      page: 1,
      results: [
        { ...mockTrendingMovies.results[0], backdrop_path: null, id: 301 },
        { ...mockTrendingMovies.results[1], id: 302 },
      ],
      total_pages: 1,
      total_results: 2,
    };
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(missingBackdrop))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );

    await page.goto('/');

    // Only the item with a backdrop survives the filter.
    await expect(page.getByTestId('hero-banner')).toBeVisible();
    await expect(page.getByTestId('hero-banner-slide')).toHaveCount(1);
    // Single-item carousel: no dot indicators rendered.
    await expect(page.getByTestId('hero-banner-dots')).toHaveCount(0);
  });
});

// =============================================================================
// AC#2: Auto-rotation (via page.clock — virtual time, deterministic)
// =============================================================================

test.describe('HeroBanner Auto-Rotation @ui @hero-banner @story-10-2', () => {
  test('[P1] should auto-rotate to next slide after 8s via virtual clock (AC2)', async ({
    page,
  }) => {
    // Install virtual clock BEFORE navigation so setInterval binds to it.
    await page.clock.install({ time: new Date('2026-04-15T00:00:00Z') });

    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingTV))
    );

    await page.goto('/');

    // Give React/TanStack Query a beat to flush the data (uses setTimeout).
    await page.clock.runFor(500);

    // Initial active slide = movie #101.
    const activeSlide = () => page.locator('[data-testid="hero-banner-slide"][data-active="true"]');
    await expect(activeSlide().getByTestId('hero-banner-title')).toHaveText('鬼滅之刃：無限城篇');

    // Advance virtual time by the full rotation interval.
    await page.clock.fastForward(8000);

    // Now second slide (movie #102) should be active.
    await expect(activeSlide().getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');
  });

  test('[P1] should navigate to a specific slide via dot indicator (AC2)', async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingTV))
    );

    await page.goto('/');

    // Dots rendered (3 items: movie+movie+tv interleaved → M0, T0, M1).
    await expect(page.getByTestId('hero-banner-dots')).toBeVisible();

    // Jump to the 3rd dot.
    await page.getByTestId('hero-banner-dot-2').click();

    const activeSlide = page.locator('[data-testid="hero-banner-slide"][data-active="true"]');
    await expect(activeSlide.getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');
    await expect(page.getByTestId('hero-banner-dot-2')).toHaveAttribute('aria-current', 'true');
  });
});

// =============================================================================
// AC#3: Click-through to detail page
// =============================================================================

test.describe('HeroBanner Navigation @ui @hero-banner @story-10-2', () => {
  test('[P1] should navigate to media detail page when 查看詳情 link is clicked (AC3)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );
    // Detail page pulls TMDb details — stub minimally so navigation resolves.
    await page.route(`${ROUTE_API}/tmdb/movies/101`, (route: Route) =>
      route.fulfill(
        jsonOk({
          id: 101,
          title: '鬼滅之刃：無限城篇',
          overview: '炭治郎…',
          backdrop_path: '/movie1.jpg',
          poster_path: '/movie1p.jpg',
          release_date: '2024-05-10',
          vote_average: 8.2,
          vote_count: 1250,
          runtime: 120,
          genres: [],
          production_countries: [],
        })
      )
    );
    await page.route(`${ROUTE_API}/tmdb/movies/101/credits`, (route: Route) =>
      route.fulfill(jsonOk({ cast: [], crew: [] }))
    );

    await page.goto('/');

    const detailLink = page.getByTestId('hero-banner-detail-link').first();
    await expect(detailLink).toHaveAttribute('href', '/media/movie/101');

    await detailLink.click();
    await page.waitForURL(/\/media\/movie\/101/);
    expect(page.url()).toMatch(/\/media\/movie\/101/);
  });
});

// =============================================================================
// AC#4: Mobile responsive layout
// =============================================================================

test.describe('HeroBanner Mobile Layout @ui @hero-banner @story-10-2', () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test('[P1] should render banner with mobile-sized backdrop and truncated overview (AC4)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );

    await page.goto('/');

    const banner = page.getByTestId('hero-banner');
    await expect(banner).toBeVisible();

    // Story 10-5 replaced the vh-based mobile height with a fixed
    // `h-[250px] md:h-[400px]`. Mobile viewport (<768px) → 250px.
    const box = await banner.boundingBox();
    expect(box).not.toBeNull();
    expect(Math.round(box!.height)).toBe(250);

    // Overview has the mobile line-clamp (2 lines via `line-clamp-2` utility).
    const overview = page.getByTestId('hero-banner-overview').first();
    await expect(overview).toHaveClass(/line-clamp-2/);
  });
});

// =============================================================================
// AC#6: Trailer modal
// =============================================================================

test.describe('HeroBanner Trailer Modal @ui @hero-banner @story-10-2', () => {
  test.beforeEach(async ({ page }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/tmdb/trending/movies*`, (route: Route) =>
      route.fulfill(jsonOk(mockTrendingMovies))
    );
    await page.route(`${ROUTE_API}/tmdb/trending/tv*`, (route: Route) =>
      route.fulfill(jsonOk({ page: 1, results: [], total_pages: 0, total_results: 0 }))
    );
  });

  test('[P0] should open trailer modal with YouTube iframe on play button click (AC6)', async ({
    page,
  }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonOk(mockVideosWithTrailer))
    );

    await page.goto('/');

    // Modal is not in the DOM before the button click.
    await expect(page.getByTestId('trailer-modal')).toHaveCount(0);

    await page.getByTestId('hero-banner-play-trailer').first().click();

    const modal = page.getByTestId('trailer-modal');
    await expect(modal).toBeVisible();

    // Iframe renders with the OFFICIAL trailer key (pickBestTrailer: official > newest).
    const iframe = modal.getByTestId('trailer-modal-iframe');
    await expect(iframe).toHaveAttribute(
      'src',
      /youtube-nocookie\.com\/embed\/official_abc_01\?autoplay=1/
    );
  });

  test('[P0] should close modal on Escape key (AC6)', async ({ page }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonOk(mockVideosWithTrailer))
    );

    await page.goto('/');
    await page.getByTestId('hero-banner-play-trailer').first().click();
    await expect(page.getByTestId('trailer-modal')).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(page.getByTestId('trailer-modal')).toHaveCount(0);
  });

  test('[P1] should close modal on backdrop click (AC6)', async ({ page }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonOk(mockVideosWithTrailer))
    );

    await page.goto('/');
    await page.getByTestId('hero-banner-play-trailer').first().click();
    const modal = page.getByTestId('trailer-modal');
    await expect(modal).toBeVisible();

    // Click the backdrop (the outer dialog surface, not the inner iframe container).
    // The backdrop discriminator is `e.target === e.currentTarget`; click at a corner
    // that is guaranteed to hit the outer container.
    await modal.click({ position: { x: 5, y: 5 } });

    await expect(page.getByTestId('trailer-modal')).toHaveCount(0);
  });

  test('[P1] should close modal on close button click (AC6)', async ({ page }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonOk(mockVideosWithTrailer))
    );

    await page.goto('/');
    await page.getByTestId('hero-banner-play-trailer').first().click();
    await expect(page.getByTestId('trailer-modal')).toBeVisible();

    await page.getByTestId('trailer-modal-close').click();

    await expect(page.getByTestId('trailer-modal')).toHaveCount(0);
  });

  test('[P1] should show empty state when no embeddable trailer exists (AC6 fallback)', async ({
    page,
  }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonOk(mockVideosEmpty))
    );

    await page.goto('/');
    await page.getByTestId('hero-banner-play-trailer').first().click();

    const modal = page.getByTestId('trailer-modal');
    await expect(modal).toBeVisible();
    await expect(page.getByTestId('trailer-modal-empty')).toHaveText('找不到預告片');
    await expect(page.getByTestId('trailer-modal-iframe')).toHaveCount(0);
  });

  test('[P1] should show empty state when videos endpoint fails (AC6 graceful)', async ({
    page,
  }) => {
    await page.route(`${ROUTE_API}/tmdb/movies/101/videos`, (route: Route) =>
      route.fulfill(jsonErr(500, 'TMDB_ERROR', 'upstream failure'))
    );

    await page.goto('/');
    await page.getByTestId('hero-banner-play-trailer').first().click();

    // Modal opens but renders the empty state instead of crashing.
    await expect(page.getByTestId('trailer-modal')).toBeVisible();
    await expect(page.getByTestId('trailer-modal-empty')).toBeVisible();
  });
});
