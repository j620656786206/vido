import type { Page } from '@playwright/test';

/**
 * Stubs for the phone downloads e2e (dsr-4b-1 downloads-mobile.spec.ts), in the
 * same shapes downloads-v2.spec.ts stubs: the qBittorrent config, the paginated
 * list, and the per-status counts. No real qBittorrent is involved.
 */
export const ROUTE_API = '**/api/v1';

export async function stubQbtConfig(page: Page, configured = true): Promise<void> {
  await page.route(`${ROUTE_API}/settings/qbittorrent`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured },
      }),
    })
  );
}

export function paginated(items: unknown[]) {
  return {
    success: true,
    data: {
      items,
      page: 1,
      pageSize: 100,
      totalItems: items.length,
      totalPages: items.length ? 1 : 0,
    },
  };
}

/**
 * Serves `items` for every list request and records each request URL, so a test
 * can assert the sort that went over the wire.
 */
export async function stubList(page: Page, items: unknown[]): Promise<string[]> {
  const requests: string[] = [];
  await page.route(`${ROUTE_API}/downloads*`, (route) => {
    requests.push(route.request().url());
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(paginated(items)),
    });
  });
  return requests;
}

// Registration order does not matter: the list route's `/downloads*` glob does not
// match `/downloads/counts` (`*` stops at `/`), so the two never compete.
export async function stubCounts(
  page: Page,
  counts: Partial<Record<string, number>> & { all: number }
): Promise<void> {
  await page.route(`${ROUTE_API}/downloads/counts`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        // bugfix-e [@contract-v1]: `queued` is part of the shape — keep the stub in
        // step with the real endpoint (same note as downloads-v2.spec.ts).
        data: {
          downloading: 0,
          paused: 0,
          queued: 0,
          completed: 0,
          seeding: 0,
          error: 0,
          ...counts,
        },
      }),
    })
  );
}
