/**
 * Empty Library 3-State E2E Tests (bugfix-10-5)
 *
 * Browser-based proof of the 3-state empty-library classifier introduced
 * by bugfix-10-5. The unit suite (`emptyLibraryState.spec.ts` x8 cases)
 * proves the classifier is internally correct, and the route integration
 * suite (`library.spec.tsx` x3 Case A/B/C cases) proves the JSX wiring
 * within a mocked-hooks render. Neither proves:
 *   - TanStack Router actually navigates when a CTA `<Link>` is clicked
 *   - The real `useMediaLibraries`/`useQBittorrentConfig`/`useTriggerScan`
 *     wire shapes (snake_case + ApiResponse wrapper) hydrate the
 *     classifier inputs correctly through the data-fetch layer
 *   - The Case C scan button actually issues `POST /api/v1/scanner/scan`
 *     and surfaces the success notification on round-trip
 *   - The classifier loading short-circuit really keeps the empty-state
 *     hidden while queries are pending (vs racing the skeleton out)
 *   - Search-active branch still owns its empty-state independent of the
 *     3-state classifier (scope-boundary regression guard)
 *
 * The TA pass found a P0 integration bug en route to writing this file —
 * `library.tsx:653` was reading `mediaLibrariesQuery.data?.length`
 * against a `{ libraries: [...] }` wrapper, making Case C unreachable in
 * production despite all unit tests passing (the unit mocks shaped data
 * as a bare array, hiding the real wire contract). The fix lives in the
 * same commit as this file. These specs lock the post-fix behavior so the
 * mock/wire drift cannot recur silently.
 *
 * @tags @ui @library @bugfix-10-5
 */

import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

// =============================================================================
// Wire-format payloads — snake_case (frontend's fetchApi runs snakeToCamel
// on response per Rule 18). MediaLibrary fields kept camel here because
// mediaLibraryService doesn't transform — it returns the raw shape.
// =============================================================================

const qbtConnected = {
  host: 'http://localhost:8080',
  username: 'admin',
  basePath: '',
  configured: true,
};

const qbtDisconnected = {
  host: '',
  username: '',
  basePath: '',
  configured: false,
};

const oneLibrary = {
  libraries: [
    {
      id: 'lib-1',
      name: 'Movies',
      contentType: 'movie' as const,
      autoDetect: false,
      sortOrder: 0,
      createdAt: '2026-05-01T00:00:00Z',
      updatedAt: '2026-05-01T00:00:00Z',
      paths: [
        {
          id: 'path-1',
          libraryId: 'lib-1',
          path: '/data/movies',
          status: 'accessible' as const,
          lastCheckedAt: '2026-05-11T00:00:00Z',
          createdAt: '2026-05-01T00:00:00Z',
        },
      ],
      mediaCount: 0,
    },
  ],
};

const fiveLibraries = {
  libraries: Array.from({ length: 5 }, (_, i) => ({
    id: `lib-${i + 1}`,
    name: `Library ${i + 1}`,
    contentType: 'movie' as const,
    autoDetect: false,
    sortOrder: i,
    createdAt: '2026-05-01T00:00:00Z',
    updatedAt: '2026-05-01T00:00:00Z',
    paths: [],
    mediaCount: 0,
  })),
};

const emptyLibraryList = {
  items: [],
  page: 1,
  page_size: 24,
  total_items: 0,
  total_pages: 0,
};

const emptyMediaStats = { total: 0, unmatched_count: 0 };
const emptyLibraryStats = {
  year_min: 0,
  year_max: 0,
  movie_count: 0,
  tv_count: 0,
  total_count: 0,
};
const idleScanStatus = {
  is_scanning: false,
  files_scanned: 0,
  files_total: 0,
  current_path: '',
  started_at: null,
};

// =============================================================================
// Helpers
// =============================================================================

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

type Page = import('@playwright/test').Page;

/**
 * Install the baseline mocks every empty-library test needs. Per-test
 * overrides MUST be installed BEFORE this so the more-specific patterns
 * win the route-resolution race (Playwright matches in registration
 * order and runs the first matching handler).
 */
async function stubLibraryRouteBaseline(page: Page) {
  await page.route(`${ROUTE_API}/library/stats`, (route: Route) =>
    route.fulfill(jsonOk(emptyLibraryStats))
  );
  await page.route(`${ROUTE_API}/library/genres`, (route: Route) => route.fulfill(jsonOk([])));
  await page.route(`${ROUTE_API}/library/recent*`, (route: Route) =>
    route.fulfill(jsonOk(emptyLibraryList))
  );
  await page.route(`${ROUTE_API}/movies/stats`, (route: Route) =>
    route.fulfill(jsonOk(emptyMediaStats))
  );
  await page.route(`${ROUTE_API}/series/stats`, (route: Route) =>
    route.fulfill(jsonOk(emptyMediaStats))
  );
  await page.route(`${ROUTE_API}/scanner/status`, (route: Route) =>
    route.fulfill(jsonOk(idleScanStatus))
  );
  await page.route(`${ROUTE_API}/health/services*`, (route: Route) =>
    route.fulfill(jsonOk({ services: [] }))
  );
  // Catch-all empty list for /library and /library?page=...
  await page.route(`${ROUTE_API}/library*`, (route: Route, request) => {
    // Don't capture /library/stats, /library/genres, /library/recent*
    // — already routed above. Playwright matches first registered first,
    // so this only fires for /library and /library?... query variants.
    if (request.url().includes('/library/search')) {
      return route.fulfill(jsonOk({ results: [], total_count: 0 }));
    }
    return route.fulfill(jsonOk(emptyLibraryList));
  });
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Empty Library 3-State Classifier @ui @library @bugfix-10-5', () => {
  test('[P0] Case A — qBT disconnected renders EmptyNoQBT with correctly-routed CTAs', async ({
    page,
  }) => {
    // GIVEN: qBT not configured + no libraries + no items (the original bug
    // Alexyu identified at retro-10 walkthrough: qBT was DISCONNECTED but
    // the UI was telling the user to connect a folder)
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
      route.fulfill(jsonOk(qbtDisconnected))
    );
    await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
      route.fulfill(jsonOk({ libraries: [] }))
    );
    await stubLibraryRouteBaseline(page);

    // WHEN: user lands on /library
    await page.goto('/library');

    // THEN: Case A wins — EmptyNoQBT is the only empty-state visible
    const noQbt = page.getByTestId('empty-no-qbt');
    await expect(noQbt).toBeVisible();
    await expect(page.getByTestId('empty-no-folder')).toHaveCount(0);
    await expect(page.getByTestId('empty-ready-for-scan')).toHaveCount(0);

    // AND: CTAs route correctly
    const connectBtn = page.getByTestId('empty-no-qbt-connect-btn');
    const folderBtn = page.getByTestId('empty-no-qbt-folder-btn');
    await expect(connectBtn).toHaveAttribute('href', '/settings/qbittorrent');
    await expect(folderBtn).toHaveAttribute('href', '/settings/libraries');
    await expect(connectBtn).toHaveText('連接 qBittorrent');
    await expect(folderBtn).toHaveText('已有檔案？設定資料夾');

    // AND: clicking connect-btn really navigates (proves <Link> works inside
    // TanStack Router, not just that the href attribute is set). Note the
    // app redirects /settings/qbittorrent → /settings/connection (the
    // consolidated settings page) — the Link's `to` prop is correct per
    // AC #2, but landing target is the connection settings. Assert we
    // left /library and arrived inside /settings/*.
    await connectBtn.click();
    await expect(page).toHaveURL(/\/settings\//);
    await expect(page).not.toHaveURL(/\/library/);
  });

  test('[P0] Case B — qBT OK + zero libraries renders EmptyNoFolder', async ({ page }) => {
    // GIVEN: qBT connected but no media folders configured + no items
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
      route.fulfill(jsonOk(qbtConnected))
    );
    await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
      route.fulfill(jsonOk({ libraries: [] }))
    );
    await stubLibraryRouteBaseline(page);

    // WHEN: user lands on /library
    await page.goto('/library');

    // THEN: Case B wins — EmptyNoFolder is the only empty-state visible
    const noFolder = page.getByTestId('empty-no-folder');
    await expect(noFolder).toBeVisible();
    await expect(page.getByTestId('empty-no-qbt')).toHaveCount(0);
    await expect(page.getByTestId('empty-ready-for-scan')).toHaveCount(0);

    // AND: CTAs route correctly
    await expect(page.getByTestId('empty-no-folder-libraries-btn')).toHaveAttribute(
      'href',
      '/settings/libraries'
    );
    await expect(page.getByTestId('empty-no-folder-wizard-btn')).toHaveAttribute('href', '/setup');
  });

  test('[P0] Case C — qBT OK + 1 folder + 0 items renders EmptyReadyForScan, scan button POSTs /scanner/scan', async ({
    page,
  }) => {
    // GIVEN: full happy-path config but library is genuinely empty —
    // EmptyReadyForScan must be reachable. This is the case the wrapper-
    // vs-array bug hid until the TA-pass fix.
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
      route.fulfill(jsonOk(qbtConnected))
    );
    await page.route(`${ROUTE_API}/libraries`, (route: Route) => route.fulfill(jsonOk(oneLibrary)));
    let scanPostCount = 0;
    await page.route(`${ROUTE_API}/scanner/scan`, (route: Route) => {
      if (route.request().method() === 'POST') {
        scanPostCount += 1;
        return route.fulfill(jsonOk({ id: 'scan-task-1', status: 'started' }));
      }
      return route.continue();
    });
    await stubLibraryRouteBaseline(page);

    // WHEN: user lands on /library
    await page.goto('/library');

    // THEN: Case C wins — EmptyReadyForScan visible
    const readyForScan = page.getByTestId('empty-ready-for-scan');
    await expect(readyForScan).toBeVisible();
    await expect(page.getByTestId('empty-no-qbt')).toHaveCount(0);
    await expect(page.getByTestId('empty-no-folder')).toHaveCount(0);

    // AND: trigger button issues POST /api/v1/scanner/scan
    const triggerBtn = page.getByTestId('empty-ready-for-scan-trigger-btn');
    await expect(triggerBtn).toBeEnabled();
    await expect(triggerBtn).toHaveText('立即掃描');

    await Promise.all([
      page.waitForRequest(
        (req) => req.url().includes('/api/v1/scanner/scan') && req.method() === 'POST'
      ),
      triggerBtn.click(),
    ]);
    expect(scanPostCount).toBe(1);

    // AND: success notification surfaces
    const notification = page.getByTestId('empty-ready-for-scan-notification');
    await expect(notification).toBeVisible();
    await expect(notification).toHaveText(/掃描已啟動/);

    // AND: secondary CTA routes to /downloads
    await expect(page.getByTestId('empty-ready-for-scan-downloads-btn')).toHaveAttribute(
      'href',
      '/downloads'
    );
  });

  test('[P1] Loading — slow qBT + libraries queries keep empty-state hidden until they resolve', async ({
    page,
  }) => {
    // GIVEN: qBT and libraries respond slowly (1.2s) — the classifier must
    // return 'loading' (rendering null), letting the existing skeleton own
    // the frame instead of flashing an empty-state.
    let qbtResolve: (() => void) | undefined;
    let libsResolve: (() => void) | undefined;
    const qbtGate = new Promise<void>((r) => (qbtResolve = r));
    const libsGate = new Promise<void>((r) => (libsResolve = r));

    await page.route(`${ROUTE_API}/settings/qbittorrent`, async (route: Route) => {
      await qbtGate;
      await route.fulfill(jsonOk(qbtConnected));
    });
    await page.route(`${ROUTE_API}/libraries`, async (route: Route) => {
      await libsGate;
      await route.fulfill(jsonOk(oneLibrary));
    });
    await stubLibraryRouteBaseline(page);

    // WHEN: navigation starts; we don't await full load
    void page.goto('/library');

    // THEN: while the two gating queries are pending, NONE of the 3
    // empty-state slots may render. Poll for ~600ms.
    await page.waitForTimeout(600);
    await expect(page.getByTestId('empty-no-qbt')).toHaveCount(0);
    await expect(page.getByTestId('empty-no-folder')).toHaveCount(0);
    await expect(page.getByTestId('empty-ready-for-scan')).toHaveCount(0);

    // WHEN: the gating queries finally resolve
    qbtResolve!();
    libsResolve!();

    // THEN: the correct empty-state (Case C — qBT OK + 1 lib + 0 items)
    // appears, proving the hold-back was only transient.
    await expect(page.getByTestId('empty-ready-for-scan')).toBeVisible();
  });

  test('[P1] Case A absolute priority — qBT off + 5 folders + 0 items still renders EmptyNoQBT (overrides C)', async ({
    page,
  }) => {
    // GIVEN: the operator has 5 libraries already configured, but qBT
    // dropped offline. AC #1 priority order: A > B > C. EmptyNoQBT must
    // win over EmptyReadyForScan even though folders exist.
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
      route.fulfill(jsonOk(qbtDisconnected))
    );
    await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
      route.fulfill(jsonOk(fiveLibraries))
    );
    await stubLibraryRouteBaseline(page);

    // WHEN
    await page.goto('/library');

    // THEN: Case A absolute priority enforced
    await expect(page.getByTestId('empty-no-qbt')).toBeVisible();
    await expect(page.getByTestId('empty-no-folder')).toHaveCount(0);
    await expect(page.getByTestId('empty-ready-for-scan')).toHaveCount(0);
  });

  test('[P1] Search-active scope — typing a query renders EmptySearchResults, not the 3-state classifier', async ({
    page,
  }) => {
    // GIVEN: all 3 classifier conditions would resolve to Case A (qBT off
    // + no libs + no items) — but search is active. The classifier branch
    // must NOT fire; EmptySearchResults owns the search-empty UX.
    await page.route(`${ROUTE_API}/settings/qbittorrent`, (route: Route) =>
      route.fulfill(jsonOk(qbtDisconnected))
    );
    await page.route(`${ROUTE_API}/libraries`, (route: Route) =>
      route.fulfill(jsonOk({ libraries: [] }))
    );
    await page.route(`${ROUTE_API}/library/search*`, (route: Route) =>
      route.fulfill(jsonOk({ results: [], total_count: 0 }))
    );
    await stubLibraryRouteBaseline(page);

    // WHEN: navigate with ?q=xyz (>= 2 chars triggers isSearchActive)
    await page.goto('/library?q=xyz');

    // THEN: search-empty UX wins; classifier branch is bypassed
    await expect(page.getByTestId('empty-search-results')).toBeVisible();
    await expect(page.getByTestId('empty-no-qbt')).toHaveCount(0);
    await expect(page.getByTestId('empty-no-folder')).toHaveCount(0);
    await expect(page.getByTestId('empty-ready-for-scan')).toHaveCount(0);
  });
});
