/**
 * Downloads v2 deep page E2E (ux3-4-3a — shell-gate + restyle + states)
 *
 * Network-first (route interception) — no real qBittorrent, mirroring the legacy
 * downloads.spec.ts pattern. Enables the v2 shell so /downloads renders
 * DownloadsBrowseV2. NO data-dependent self-skip (Epic 20 lesson): every state is
 * driven by a deterministic mocked response.
 *
 * @tags @downloads @ui @ux3-4-3
 */

import { test, expect } from '../support/fixtures';
import type { Route } from '@playwright/test';
import { presetDownloads } from '../support/fixtures/factories/download-factory';

const ROUTE_API = '**/api/v1';

/**
 * Enable the v2 shell: seed the flag's localStorage mirror (read as initialData so the
 * shell renders on first paint with no flag→shell flash) AND stub the confirming flag
 * endpoint (mirrors discover-filters.spec.ts enableV2Shell).
 */
async function enableV2Shell(page: import('@playwright/test').Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      localStorage.setItem('vido:flag:new_shell_enabled', 'true');
    } catch {
      /* ignore */
    }
  });
  await page.route('**/api/v1/settings/new_shell_enabled', (route: Route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { key: 'new_shell_enabled', value: 'true' } }),
    })
  );
}

async function stubQbtConfig(
  page: import('@playwright/test').Page,
  configured: boolean
): Promise<void> {
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

const mockList = [
  presetDownloads.downloading,
  presetDownloads.completed,
  presetDownloads.seeding,
  presetDownloads.paused,
];

function paginated(items: unknown[]) {
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

// counts must be stubbed AFTER the general /downloads* route so the later, more-specific
// registration wins for /downloads/counts (Playwright: last matching route wins).
async function stubCounts(page: import('@playwright/test').Page, all: number): Promise<void> {
  await page.route(`${ROUTE_API}/downloads/counts`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { all, downloading: 1, paused: 1, completed: 1, seeding: 1, error: 0 },
      }),
    })
  );
}

test.describe('Downloads v2 deep page @downloads @ui @ux3-4-3', () => {
  test('[P1] v2 shell renders the DownloadsBrowseV2 deep page with restyled cards + status toolbar (AC1/AC2)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated(mockList)),
      })
    );
    await stubCounts(page, mockList.length);

    await page.goto('/downloads');

    const browse = page.getByTestId('downloads-browse-v2');
    await expect(browse).toBeVisible({ timeout: 15000 });
    // v2 header (scoped to the browse container — the sidebar also has a 下載 nav label)
    await expect(browse.getByRole('heading', { name: '下載' })).toBeVisible();
    // status-filter toolbar (AC2)
    await expect(browse.getByRole('tab', { name: /全部/ })).toBeVisible();
    // restyled DownloadCard-v2 rows with an accessible progressbar + a status token pill
    await expect(browse.locator('[data-testid^="download-card-v2-"]').first()).toBeVisible();
    await expect(browse.locator('[data-testid^="download-status-"]').first()).toBeVisible();
    await expect(browse.getByRole('progressbar').first()).toBeVisible();
    // legacy header is gone under v2
    await expect(page.getByText('下載管理')).toHaveCount(0);
  });

  test('[P2] empty list renders the distinct no-downloads state + 前往探索 (AC6/D5)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated([])),
      })
    );
    await stubCounts(page, 0);

    await page.goto('/downloads');

    await expect(page.getByTestId('downloads-empty-v2')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('目前沒有下載任務')).toBeVisible();
    await expect(page.getByRole('link', { name: '前往探索' })).toBeVisible();
  });

  test('[P2] qBittorrent 503 renders the per-section fail-soft with 重試 + 前往設定 (AC6/D6)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: {
            code: 'QBITTORRENT_NOT_CONFIGURED',
            message: '無法連線到 qBittorrent',
            suggestion: '請先設定 qBittorrent 連線。SETUP_REQUIRED',
          },
        }),
      })
    );
    await stubCounts(page, 0);

    await page.goto('/downloads');

    const failSoft = page.getByTestId('downloads-qbt-error-v2');
    await expect(failSoft).toBeVisible({ timeout: 15000 });
    await expect(failSoft).toContainText('無法連線到 qBittorrent');
    await expect(failSoft.getByRole('button', { name: '重試' })).toBeVisible();
    await expect(failSoft.getByRole('link', { name: '前往設定' })).toBeVisible();
    // fail-soft: the shell + nav still render (page never hard-fails)
    await expect(page.getByTestId('downloads-browse-v2')).toBeVisible();
  });
});

test.describe('Downloads v2 actions + batch @downloads @ui @ux3-4-3', () => {
  test('[P1] a card 暫停 action POSTs to /downloads/:hash/pause (AC3)', async ({ page }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);

    const pauseHits: string[] = [];
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated([presetDownloads.downloading])),
      })
    );
    await stubCounts(page, 1);
    await page.route(/\/api\/v1\/downloads\/[^/]+\/pause$/, (route) => {
      pauseHits.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: null }),
      });
    });

    await page.goto('/downloads');
    const card = page.locator('[data-testid^="download-card-v2-"]').first();
    await expect(card).toBeVisible({ timeout: 15000 });

    await card.getByRole('button', { name: /暫停/ }).click();
    await expect.poll(() => pauseHits.length).toBeGreaterThan(0);
  });

  test('[P1] remove opens a confirm dialog; 連同檔案刪除 DELETEs with deleteFiles=true (AC3)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);

    const deleteHits: string[] = [];
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated([presetDownloads.downloading])),
      })
    );
    await stubCounts(page, 1);
    await page.route(/\/api\/v1\/downloads\/[^/?]+(\?.*)?$/, (route) => {
      if (route.request().method() === 'DELETE') {
        deleteHits.push(route.request().url());
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: null }),
        });
      }
      return route.fallback();
    });

    await page.goto('/downloads');
    const card = page.locator('[data-testid^="download-card-v2-"]').first();
    await expect(card).toBeVisible({ timeout: 15000 });

    await card.getByRole('button', { name: /移除/ }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('button', { name: '移除（連同檔案刪除）' }).click();

    await expect.poll(() => deleteHits.length).toBeGreaterThan(0);
    expect(deleteHits[0]).toContain('deleteFiles=true');
  });

  test('[P2] batch select → 批次暫停 fires one pause request per selected hash (AC5)', async ({
    page,
  }) => {
    await enableV2Shell(page);
    await stubQbtConfig(page, true);

    const pauseHits: string[] = [];
    const list = [presetDownloads.downloading, presetDownloads.seeding];
    await page.route(`${ROUTE_API}/downloads*`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated(list)),
      })
    );
    await stubCounts(page, list.length);
    await page.route(/\/api\/v1\/downloads\/[^/]+\/pause$/, (route) => {
      pauseHits.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: null }),
      });
    });

    await page.goto('/downloads');
    const browse = page.getByTestId('downloads-browse-v2');
    await expect(browse).toBeVisible({ timeout: 15000 });

    // enter select mode → select all → batch pause
    await browse.getByRole('button', { name: '選取' }).click();
    await browse.getByRole('button', { name: '全選' }).click();
    await browse.getByRole('button', { name: '批次暫停' }).click();

    await expect.poll(() => pauseHits.length).toBe(list.length);
  });
});
