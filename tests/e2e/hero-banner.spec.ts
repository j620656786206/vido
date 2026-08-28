/**
 * Hero Banner E2E Tests (Story ux3-1-8 — own-library hero)
 *
 * ux3-1-8 flipped the homepage's largest surface from TMDb trending to the
 * user's OWN shelf. This suite is the regression fence around that flip: every
 * assertion here fails loudly if the retired carousel ever grows back.
 *
 * What the hero is now (HeroBanner.tsx, design ref H1-D-v3 / H2-M-v3 / H7-D-v3):
 *   - Content = GET /api/v1/library/recent?limit=20, keeping only items that
 *     carry a backdrop, capped at 5. Same query the 最近新增 row reads.
 *   - STATIC (⚖️ R3 靜止＋手動): no interval, no autoplay, no hover/focus pause,
 *     no pause button, no 觀看預告片 button. Only the user moves it.
 *   - Manual switching = prev/next chevrons + dots, all inside the dots pill,
 *     which only exists when there is more than one dressed item.
 *   - Links carry the LIBRARY id (a UUID string), never a TMDb numeric id.
 *   - No dressed item (or a failed request) → the section is ABSENT, and the
 *     page below stays complete (例外訊號原則).
 *
 * Coverage:
 *   - 最新入庫 identity: eyebrow / title / year / type / subtitle badge / CTA
 *   - the STATIC guard: dead controls are gone AND virtual time moves nothing
 *   - manual switching via dots + chevrons
 *   - backdrop filter + single-item (no dots pill)
 *   - absent-hero paths (empty library, failing endpoint) with a complete page
 *   - mobile fixed height (250px)
 *
 * Design notes:
 *   - `page.clock.install()` is set BEFORE `page.goto()` so ANY timer the
 *     component might register would bind to virtual time. `fastForward` then
 *     proves nothing is registered — the old suite used the same machinery to
 *     prove the opposite (8s rotation), which is exactly why it belongs here.
 *   - Mocked API payloads are snake_case at the wire level; the frontend
 *     `fetchApi` runs `snakeToCamel` on `data.data` (see services/libraryService.ts).
 *   - Library ids are UUID STRINGS (the [@contract-v2] media-id convention) —
 *     never invent numeric ids for library rows.
 *
 * NOTE: the TrailerModal is no longer mounted anywhere on the homepage, so its
 * open/close/empty journeys are NOT re-homed here. They live in
 * apps/web/src/components/homepage/TrailerModal.spec.tsx.
 *
 * @tags @ui @hero-banner @ux3-1-8
 */

import { test, expect } from '../support/fixtures';
import type { Page, Route } from '@playwright/test';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Mock Data — snake_case wire format (transformed by snakeToCamel in fetchApi)
// =============================================================================

// UUID string ids — the library's real id shape. A numeric id here would let a
// TMDb-id regression pass unnoticed (classifyId routes /^\d+$/ to the TMDb
// detail path), so the fixture deliberately cannot be confused for one.
const MOVIE_A_ID = '5c2a9d3e-1f4b-4a8c-9d2e-3f5a7b9c1d63';
const SERIES_B_ID = '8e4b2c6a-7d1f-4e3a-b5c9-2a6d8f0e4b57';
const MOVIE_C_ID = 'b7d1e4f2-3a6c-4d8e-9f0b-1c2d3e4f5a6b';
const NO_BACKDROP_ID = 'c9f2a1b4-5d7e-4c3a-8b6f-0e1d2c3b4a59';

const movieA = {
  type: 'movie',
  movie: {
    id: MOVIE_A_ID,
    title: '沙丘：第二部',
    release_date: '2024-03-01',
    genres: ['科幻'],
    poster_path: '/dune2.jpg',
    backdrop_path: '/dune2-bg.jpg',
    vote_average: 8.4,
    // Authoritative subtitle-engine verdict → deriveSubtitleStatus returns the
    // 繁中 steady state, which the hero (unlike the exception-only grid badge)
    // spells out as 繁中字幕 ✓ 已就緒.
    subtitle_status: 'found',
    subtitle_language: 'zh-Hant',
    parse_status: 'success',
    created_at: '2026-08-20T00:00:00Z',
    updated_at: '2026-08-20T00:00:00Z',
  },
};

const seriesB = {
  type: 'series',
  series: {
    id: SERIES_B_ID,
    title: '幕府將軍',
    first_air_date: '2024-02-27',
    genres: ['劇情'],
    poster_path: '/shogun.jpg',
    backdrop_path: '/shogun-bg.jpg',
    vote_average: 8.7,
    number_of_seasons: 1,
    parse_status: 'success',
    created_at: '2026-08-19T00:00:00Z',
    updated_at: '2026-08-19T00:00:00Z',
  },
};

const movieC = {
  type: 'movie',
  movie: {
    id: MOVIE_C_ID,
    title: '奧本海默',
    release_date: '2023-07-21',
    genres: ['劇情'],
    poster_path: '/oppenheimer.jpg',
    backdrop_path: '/oppenheimer-bg.jpg',
    vote_average: 8.1,
    parse_status: 'success',
    created_at: '2026-08-18T00:00:00Z',
    updated_at: '2026-08-18T00:00:00Z',
  },
};

// Newest item of all, but it cannot dress a hero — it must never take the
// first slide (or any slide).
const movieNoBackdrop = {
  type: 'movie',
  movie: {
    id: NO_BACKDROP_ID,
    title: '沒有劇照的新片',
    release_date: '2025-01-05',
    genres: ['紀錄'],
    poster_path: '/no-backdrop.jpg',
    backdrop_path: null,
    parse_status: 'success',
    created_at: '2026-08-25T00:00:00Z',
    updated_at: '2026-08-25T00:00:00Z',
  },
};

/** Three dressed items, newest first — the default hero cast. */
const dressedLibrary = [movieA, seriesB, movieC];

const mockHomeSummary = {
  coverage: { status: 'ok', covered: 2, total: 3 },
  processed_today: { status: 'ok', count: 1 },
  attention: { status: 'ok', failed_count: 0 },
  in_flight: { status: 'ok', count: 0 },
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

const jsonErr = (status: number, code: string, message: string) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify({ success: false, error: { code, message } }),
});

/** GET /library/recent envelope — libraryService reads `.items` off it. */
const recentPayload = (items: unknown[]) => ({
  items,
  page: 1,
  page_size: 20,
  total_items: items.length,
  total_pages: 1,
});

/**
 * Stubs the non-HeroBanner homepage dependencies (readout band, explore tail,
 * ownership lookup, downloads, qBittorrent settings, health services) so the
 * `/` route loads deterministically regardless of HeroBanner state.
 * `/library/recent` is deliberately NOT stubbed here — it IS the hero's data,
 * so every test states its own.
 */
async function stubHomepageBaseline(page: Page) {
  // ux3-1-7 readout band — first section on the page. It fails soft, but a stub
  // keeps the DOM (and the network) deterministic.
  await page.route(`${ROUTE_API}/home-summary`, (route: Route) =>
    route.fulfill(jsonOk(mockHomeSummary))
  );
  // ux3-1-8 tail: the TMDb rows are irrelevant to the hero — an empty block
  // list unmounts the whole group so no explore request can race these tests.
  await page.route(`${ROUTE_API}/explore-blocks`, (route: Route) =>
    route.fulfill(jsonOk({ blocks: [] }))
  );
  await page.route(`${ROUTE_API}/media/check-owned`, (route: Route) =>
    route.fulfill(jsonOk({ owned_tmdb_ids: [], requested_tmdb_ids: [] }))
  );
  await page.route(`${ROUTE_API}/downloads*`, (route: Route) =>
    route.fulfill(jsonOk({ items: [], page: 1, pageSize: 100, totalItems: 0, totalPages: 1 }))
  );
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

/** The hero's own data source — one place so no test hand-rolls the envelope. */
async function stubRecent(page: Page, items: unknown[]) {
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk(recentPayload(items)))
  );
}

const activeSlide = (page: Page) =>
  page.locator('[data-testid="hero-banner-slide"][data-active="true"]');

// =============================================================================
// Identity: the hero sells the OWN shelf
// =============================================================================

test.describe('HeroBanner own-library identity @ui @hero-banner @ux3-1-8', () => {
  test('[P0] renders the newest OWN item: 最新入庫 eyebrow, title, year, type, 查看詳情 CTA', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);

    await page.goto('/');

    const banner = page.getByTestId('hero-banner');
    await expect(banner).toBeVisible();

    // Newest dressed item takes the first slide.
    const first = activeSlide(page);
    await expect(first).toHaveCount(1);
    // 最新入庫, not 熱門 — the eyebrow is the identity claim of the whole story.
    await expect(first.getByTestId('hero-banner-eyebrow')).toHaveText('最新入庫');
    await expect(first.getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');
    await expect(first.getByTestId('hero-banner-year')).toHaveText('2024');
    await expect(first.getByTestId('hero-banner-type')).toHaveText('電影');
    await expect(first.getByTestId('hero-banner-rating')).toContainText('8.4');
    await expect(first.getByTestId('hero-banner-detail-link')).toHaveText('查看詳情');

    // The hero speaks the library's vocabulary now: a subtitle-status badge
    // (deriveSubtitleStatus, 繁中 spelled out) instead of a TMDb synopsis. The
    // retired overview line must not come back.
    await expect(first.getByTestId('hero-banner-subtitle-badge')).toHaveText('繁中字幕 ✓ 已就緒');
    await expect(page.getByTestId('hero-banner-overview')).toHaveCount(0);

    // w780 is the src baseline (handsets must never pay the desktop image);
    // srcset still offers the upgrade ladder.
    await expect(first.getByTestId('hero-banner-backdrop')).toHaveAttribute(
      'src',
      /\/t\/p\/w780\/dune2-bg\.jpg$/
    );
    await expect(first.getByTestId('hero-banner-backdrop')).toHaveAttribute(
      'srcset',
      /\/t\/p\/w1280\/dune2-bg\.jpg 1280w/
    );
  });

  test('[P1] the single detail door addresses the LIBRARY uuid, not a TMDb numeric id', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);
    // The detail page reads the LOCAL row for a UUID (classifyId → local-uuid);
    // stubbing it keeps the click-through off the shared backend.
    await page.route(`${ROUTE_API}/movies/${MOVIE_A_ID}`, (route: Route) =>
      route.fulfill(jsonOk(movieA.movie))
    );

    await page.goto('/');

    const first = activeSlide(page);
    // The gold CTA is the one explicit detail door for the library row.
    const cta = first.getByTestId('hero-banner-detail-link');
    await expect(cta).toHaveAttribute('href', `/media/movie/${MOVIE_A_ID}`);

    await cta.click();
    await page.waitForURL(`**/media/movie/${MOVIE_A_ID}`);
    expect(page.url()).toContain(`/media/movie/${MOVIE_A_ID}`);
  });
});

// =============================================================================
// The STATIC guard — the retired carousel must stay dead
// =============================================================================

test.describe('HeroBanner is static @ui @hero-banner @ux3-1-8', () => {
  test('[P0] no auto-rotation: the active slide survives a 30s virtual fast-forward', async ({
    page,
  }) => {
    // Installed BEFORE navigation so any setInterval/setTimeout the component
    // registered would bind to virtual time — fast-forwarding then PROVES
    // there is none (the retired hero rotated every 8s here).
    await page.clock.install({ time: new Date('2026-08-26T00:00:00Z') });

    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);

    await page.goto('/');
    // Give React/TanStack Query a beat to flush the data (uses setTimeout).
    await page.clock.runFor(500);

    await expect(activeSlide(page).getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');

    // 30s > the retired 8s interval AND > useRecentlyAdded's 30s
    // refetchInterval: neither a rotation nor a data refresh may move the slide.
    await page.clock.fastForward(30_000);
    await page.clock.runFor(1_000);

    await expect(activeSlide(page).getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');
    await expect(page.getByTestId('hero-banner-dot-0')).toHaveAttribute('aria-current', 'true');
  });

  test('[P0] the carousel remnants are gone: no pause button, no 觀看預告片 button', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);

    await page.goto('/');
    await expect(page.getByTestId('hero-banner')).toBeVisible();

    // Both controls only made sense for an autoplaying TMDb carousel. If either
    // reappears, the 靜止＋手動 ruling has been reverted.
    await expect(page.getByTestId('hero-banner-pause')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner-play-trailer')).toHaveCount(0);
  });
});

// =============================================================================
// Manual switching — the only thing that may move the hero
// =============================================================================

test.describe('HeroBanner manual switching @ui @hero-banner @ux3-1-8', () => {
  test('[P1] dots and prev/next chevrons move the active slide', async ({ page }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);

    await page.goto('/');

    const dots = page.getByTestId('hero-banner-dots');
    await expect(dots).toBeVisible();
    const title = () => activeSlide(page).getByTestId('hero-banner-title');
    await expect(title()).toHaveText('沙丘：第二部');

    // Dot jump — index 1 is the series, so this also proves 影集 rows reach the
    // hero (the retired carousel interleaved TMDb movie/tv lists instead).
    await page.getByTestId('hero-banner-dot-1').click();
    await expect(title()).toHaveText('幕府將軍');
    await expect(page.getByTestId('hero-banner-dot-1')).toHaveAttribute('aria-current', 'true');

    await page.getByTestId('hero-banner-next').click();
    await expect(title()).toHaveText('奧本海默');

    await page.getByTestId('hero-banner-prev').click();
    await expect(title()).toHaveText('幕府將軍');
  });

  test('[P1] a single dressed item renders no dots pill (nothing to switch, no controls)', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, [movieNoBackdrop, movieA]);

    await page.goto('/');

    // The newest item has no backdrop — it is filtered out, not blank-slotted.
    await expect(page.getByTestId('hero-banner')).toBeVisible();
    await expect(page.getByTestId('hero-banner-slide')).toHaveCount(1);
    await expect(activeSlide(page).getByTestId('hero-banner-title')).toHaveText('沙丘：第二部');

    // The chevrons live inside the pill, so the whole control cluster goes.
    await expect(page.getByTestId('hero-banner-dots')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner-next')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner-prev')).toHaveCount(0);
  });
});

// =============================================================================
// 例外訊號原則 — absent hero, complete page
// =============================================================================

test.describe('HeroBanner absent states @ui @hero-banner @ux3-1-8', () => {
  test('[P0] no backdrop-bearing item: hero is absent, the page below stays complete', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    // Library is not empty — it just holds nothing that can dress a hero. The
    // section must be ABSENT, never an empty frame or a lingering skeleton.
    await stubRecent(page, [movieNoBackdrop]);

    await page.goto('/');

    await expect(page.getByTestId('home-v2-root')).toBeVisible();
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
    await expect(page.getByTestId('hero-banner-skeleton')).toHaveCount(0);
    // The page stays complete: 最近新增 still shows the backdrop-less item.
    await expect(page.getByTestId('home-recent-row')).toBeVisible();
  });

  test('[P0] /library/recent failure: hero is absent, 最近新增 degrades on its own', async ({
    page,
  }) => {
    await stubHomepageBaseline(page);
    await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
      route.fulfill(jsonErr(500, 'LIBRARY_ERROR', 'library unavailable'))
    );

    await page.goto('/');

    // Fail-soft (F3): the hero disappears, the section below owns its own error
    // banner, and the page never hard-fails.
    await expect(page.getByTestId('home-v2-root')).toBeVisible();
    await expect(page.getByTestId('hero-banner')).toHaveCount(0);
    await expect(page.getByTestId('home-recent-error')).toBeVisible();
  });
});

// =============================================================================
// Mobile layout
// =============================================================================

test.describe('HeroBanner Mobile Layout @ui @hero-banner @ux3-1-8', () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test('[P1] hero is 250px tall on a 375px viewport', async ({ page }) => {
    await stubHomepageBaseline(page);
    await stubRecent(page, dressedLibrary);

    await page.goto('/');

    const banner = page.getByTestId('hero-banner');
    await expect(banner).toBeVisible();

    // `h-[250px] md:h-[400px]` are Tailwind arbitrary classes — a real browser
    // measurement is the only proof they compiled.
    const box = await banner.boundingBox();
    expect(box).not.toBeNull();
    expect(Math.round(box!.height)).toBe(250);
  });
});
