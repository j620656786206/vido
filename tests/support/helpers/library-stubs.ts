import type { Page } from '@playwright/test';

/**
 * Stubs for the phone library e2e (dsr-1b-b library-mobile.spec.ts), in the same
 * wire shapes empty-library.spec.ts stubs (snake_case, ApiResponse-wrapped). The list
 * stub records every request URL so a test can assert the filter that went over the
 * wire — `subtitle_status=not_found` is the whole point of the deep-link test.
 */
export const ROUTE_API = '**/api/v1';

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

/** A library movie item as GET /library returns it (snake_case, wrapped in `{type, movie}`). */
export function movieItem(id: string, title: string, subtitleStatus = 'not_found') {
  return {
    type: 'movie',
    movie: {
      id,
      title,
      original_title: title,
      release_date: '2020-01-01',
      runtime: 120,
      genres: ['動畫'],
      parse_status: 'success',
      subtitle_status: subtitleStatus,
      subtitle_tracks: '[]',
      vote_average: 7.5,
      poster_path: null,
      tmdb_id: 1,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  };
}

export function paginated(items: unknown[], totalItems = items.length) {
  return {
    items,
    page: 1,
    page_size: 36,
    total_items: totalItems,
    total_pages: totalItems ? 1 : 0,
  };
}

/**
 * Everything the /library page asks for besides the list: qBittorrent config (so the
 * empty-state classifier does not think qBT is missing), `opts.libraries` media
 * libraries (default 1; 0 → the EmptyNoFolder state), genres, stats.
 *
 * Playwright tries routes newest-first (`page.route` unshifts), so to override one of
 * these register your own route AFTER calling this helper.
 */
export async function stubLibraryBaseline(
  page: Page,
  genres = ['動畫', '科幻', '劇情'],
  opts: { libraries?: number } = {}
) {
  const libraryCount = opts.libraries ?? 1;
  await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
    route.fulfill(
      jsonOk({ host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true })
    )
  );
  await page.route(`${ROUTE_API}/libraries`, (route) =>
    route.fulfill(
      jsonOk({
        libraries: Array.from({ length: libraryCount }, (_, i) => ({
          id: `lib-${i + 1}`,
          name: 'Movies',
          contentType: 'movie',
          autoDetect: false,
          sortOrder: 0,
          createdAt: '2026-05-01T00:00:00Z',
          updatedAt: '2026-05-01T00:00:00Z',
          paths: [],
          mediaCount: 3,
        })),
      })
    )
  );
  await page.route(`${ROUTE_API}/library/genres`, (route) => route.fulfill(jsonOk(genres)));
  await page.route(`${ROUTE_API}/library/stats`, (route) =>
    route.fulfill(
      jsonOk({ year_min: 1990, year_max: 2026, movie_count: 3, tv_count: 0, total_count: 3 })
    )
  );
  await page.route(`${ROUTE_API}/movies/stats`, (route) =>
    route.fulfill(jsonOk({ total: 3, unmatched_count: 1 }))
  );
  await page.route(`${ROUTE_API}/series/stats`, (route) =>
    route.fulfill(jsonOk({ total: 0, unmatched_count: 0 }))
  );
  await page.route(`${ROUTE_API}/library/recent*`, (route) => route.fulfill(jsonOk(paginated([]))));
  await page.route(`${ROUTE_API}/health/services*`, (route) =>
    route.fulfill(jsonOk({ services: [] }))
  );
  await page.route(`${ROUTE_API}/scanner/status`, (route) =>
    route.fulfill(
      jsonOk({
        is_scanning: false,
        files_scanned: 0,
        files_total: 0,
        current_path: '',
        started_at: null,
      })
    )
  );
}

/**
 * Serves the list. `pick` decides what each request gets from its URL (so a
 * `subtitle_status=not_found` request can return fewer rows than the unfiltered one).
 * Returns the recorded request URLs. Both `/library?…` and bare `/library` are covered:
 * a glob `*` stops at `/`, so neither pattern can swallow `/library/genres` etc.
 */
export async function stubLibraryList(
  page: Page,
  pick: (url: URL) => { items: unknown[]; totalItems?: number }
): Promise<string[]> {
  const requests: string[] = [];
  await page.route(`${ROUTE_API}/library?*`, (route) => {
    const url = new URL(route.request().url());
    requests.push(url.toString());
    const { items, totalItems } = pick(url);
    return route.fulfill(jsonOk(paginated(items, totalItems)));
  });
  await page.route(`${ROUTE_API}/library`, (route) => {
    const url = new URL(route.request().url());
    requests.push(url.toString());
    const { items, totalItems } = pick(url);
    return route.fulfill(jsonOk(paginated(items, totalItems)));
  });
  return requests;
}
