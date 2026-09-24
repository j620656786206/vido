/**
 * The settings shell at real widths (dsr-3a — Component/SettingsTabStrip iiG1y,
 * page-header on C4-D / C4-M, 效能監控 on C14).
 *
 * @tags @e2e @settings @settings-shell
 *
 * jsdom has no layout, so the unit specs only prove the class tokens exist and
 * fake the strip's scroll metrics. THIS spec measures: the page title is 20px
 * on a phone and 24px from 640 up while the description stays 14px; the strip's
 * edge fades appear only on the side that really hides tabs; and at 1440 the
 * whole strip fits, so there is no fade at all. It measures both sides of the
 * 640 breakpoint (feedback_measure_the_breakpoint_you_return_to).
 *
 * dsr-3b〜3f append their page-level checks to this file.
 */
import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };
const DESKTOP = { width: 1440, height: 900 };

const fontSize = (page: Page, selector: string) =>
  page
    .locator(selector)
    .first()
    .evaluate((el) => window.getComputedStyle(el).fontSize);

async function openKeys(page: Page) {
  await page.goto('/settings/keys');
  await expect(page.getByRole('heading', { level: 1, name: '金鑰設定' })).toBeVisible({
    timeout: 15000,
  });
}

test.describe('Settings shell @settings @dsr-3a', () => {
  test('[P1] phone: title steps down to 20px, description stays 14px', async ({ page }) => {
    await page.setViewportSize(PHONE);
    await openKeys(page);
    expect(await fontSize(page, 'h1')).toBe('20px');
    const description = page.getByText('設定 Vido 使用的第三方服務 API 金鑰', { exact: false });
    expect(await description.evaluate((el) => window.getComputedStyle(el).fontSize)).toBe('14px');
  });

  test('[P1] 640 and 1440: title is 24px', async ({ page }) => {
    for (const size of [AT_BREAKPOINT, DESKTOP]) {
      await page.setViewportSize(size);
      await openKeys(page);
      expect(await fontSize(page, 'h1'), `h1 at ${size.width}`).toBe('24px');
    }
  });

  test('[P1] phone: the strip fades only on the side that hides tabs', async ({ page }) => {
    await page.setViewportSize(PHONE);
    // 外觀 is the first tab, so the strip starts scrolled to the left edge.
    await page.goto('/settings/appearance');
    const strip = page.getByTestId('settings-tabs-strip');
    await expect(strip).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('settings-tabs-fade')).toBeVisible();
    await expect(page.getByTestId('settings-tabs-fade-left')).toHaveCount(0);

    await strip.evaluate((el) => {
      el.scrollLeft = el.scrollWidth;
    });
    await expect(page.getByTestId('settings-tabs-fade-left')).toBeVisible();
    await expect(page.getByTestId('settings-tabs-fade')).toHaveCount(0);

    await strip.evaluate((el) => {
      el.scrollLeft = 200;
    });
    await expect(page.getByTestId('settings-tabs-fade-left')).toBeVisible();
    await expect(page.getByTestId('settings-tabs-fade')).toBeVisible();
  });

  test('[P1] phone: opening a later tab scrolls it into view with a left fade', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await page.goto('/settings/logs');
    const tab = page.getByTestId('settings-tab-logs');
    await expect(tab).toBeInViewport({ timeout: 15000 });
    await expect(page.getByTestId('settings-tabs-fade-left')).toBeVisible();
  });

  test('[P1] 1440: all twelve tabs fit, so there is no fade on either side', async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await openKeys(page);
    const strip = page.getByTestId('settings-tabs-strip');
    const { scrollWidth, clientWidth } = await strip.evaluate((el) => ({
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
    }));
    expect(scrollWidth, 'strip overflows its 1152px column at 1440').toBeLessThanOrEqual(
      clientWidth
    );
    await expect(page.getByTestId('settings-tabs-fade')).toHaveCount(0);
    await expect(page.getByTestId('settings-tabs-fade-left')).toHaveCount(0);
    await expect(page.getByTestId('settings-tab-performance')).toBeInViewport({ ratio: 1 });
  });

  // Rotation / a window resize changes what fits without any scroll event —
  // only the ResizeObserver hears it. A fade left behind is the false signal
  // this story removes.
  test('[P1] widening a phone window to 1440 clears the fade', async ({ page }) => {
    await page.setViewportSize(PHONE);
    await page.goto('/settings/appearance');
    await expect(page.getByTestId('settings-tabs-fade')).toBeVisible({ timeout: 15000 });
    await page.setViewportSize(DESKTOP);
    await expect(page.getByTestId('settings-tabs-fade')).toHaveCount(0);
    await expect(page.getByTestId('settings-tabs-fade-left')).toHaveCount(0);
  });

  test('[P1] every open tab has exactly one h1 that matches its label', async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    const tabs: [string, string][] = [
      ['/settings/appearance', '外觀'],
      ['/settings/connection', '連線設定'],
      ['/settings/keys', '金鑰設定'],
      ['/settings/status', '服務狀態'],
      ['/settings/scanner', '媒體庫掃描'],
      ['/settings/subtitle', '字幕設定'],
      ['/settings/homepage', '自訂首頁'],
      ['/settings/cache', '快取管理'],
      ['/settings/logs', '系統日誌'],
      ['/settings/backup', '備份與還原'],
      ['/settings/export', '匯出/匯入'],
    ];
    for (const [path, label] of tabs) {
      await page.goto(path);
      const h1 = page.locator('h1');
      await expect(h1, path).toHaveCount(1, { timeout: 15000 });
      await expect(h1, path).toHaveText(label);
    }
  });
});

// dsr-3b — 金鑰設定 (C7-D PWvEX / C7-M f8Fda). The key state is stubbed: one key
// of each source, the way C7 draws them.
test.describe('金鑰設定 layout @settings @dsr-3b', () => {
  const KEYS = {
    writable: true,
    keys: [
      { name: 'claude', configured: true, source: 'secret', masked: 'sk-ant…7f3a' },
      { name: 'tmdb', configured: true, source: 'env' },
      { name: 'openai', configured: false, source: 'none' },
    ],
  };
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/settings/keys', (route) =>
      route.request().method() === 'GET'
        ? route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ success: true, data: KEYS }),
          })
        : route.fallback()
    );
  });

  test('[P1] phone: every key is its own card and 儲存金鑰 spans the column', async ({ page }) => {
    await page.setViewportSize(PHONE);
    await page.goto('/settings/keys');
    const rows = ['claude', 'tmdb', 'openai'].map((k) => page.getByTestId(`key-row-${k}`));
    await expect(rows[0]).toBeVisible({ timeout: 15000 });
    for (const row of rows) {
      const bg = await row.evaluate((el) => window.getComputedStyle(el).backgroundColor);
      expect(bg, 'each row carries its own card surface').not.toBe('rgba(0, 0, 0, 0)');
    }
    const outer = await page
      .getByTestId('api-keys-card')
      .evaluate((el) => window.getComputedStyle(el).backgroundColor);
    expect(outer, 'the shared card dissolves on a phone').toBe('rgba(0, 0, 0, 0)');
    const save = await page.getByTestId('key-save').boundingBox();
    const column = await page.getByTestId('key-rows').boundingBox();
    expect(Math.abs((save?.width ?? 0) - (column?.width ?? 0))).toBeLessThanOrEqual(1);
  });

  test('[P1] 1440: one card holds every key', async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto('/settings/keys');
    await expect(page.getByTestId('key-row-claude')).toBeVisible({ timeout: 15000 });
    const outer = await page
      .getByTestId('api-keys-card')
      .evaluate((el) => window.getComputedStyle(el).backgroundColor);
    expect(outer).not.toBe('rgba(0, 0, 0, 0)');
    const row = await page
      .getByTestId('key-row-tmdb')
      .evaluate((el) => window.getComputedStyle(el).backgroundColor);
    expect(row, 'rows sit on the shared card, not on their own').toBe('rgba(0, 0, 0, 0)');
  });

  test('[P1] a typed Claude key shows its 測試 inside the box, clear of the text', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await page.route('**/api/v1/settings/keys', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            ...KEYS,
            keys: [{ name: 'claude', configured: false, source: 'none' }, ...KEYS.keys.slice(1)],
          },
        }),
      })
    );
    await page.goto('/settings/keys');
    // Hold the probe open so the button is in its WIDEST state (spinner + label).
    await page.route('**/api/v1/settings/keys/test', () => {
      /* never fulfilled — the busy state is what gets measured */
    });
    const input = page.getByLabel('Claude（翻譯）');
    await expect(input).toBeVisible({ timeout: 15000 });
    await page.getByTestId('key-test-claude').click();
    await expect(page.getByTestId('key-test-claude')).toBeDisabled();
    const box = await input.boundingBox();
    const btn = await page.getByTestId('key-test-claude').boundingBox();
    expect(btn && box).toBeTruthy();
    // Inside the input's box…
    expect(btn!.x).toBeGreaterThanOrEqual(box!.x);
    expect(btn!.x + btn!.width).toBeLessThanOrEqual(box!.x + box!.width);
    // …and the text area (content box) ends before the button starts, so a long
    // key never runs under it — measured with the button at its widest.
    const textRight = await input.evaluate(
      (el) =>
        el.getBoundingClientRect().right - parseFloat(window.getComputedStyle(el).paddingRight)
    );
    expect(textRight).toBeLessThanOrEqual(btn!.x + 1);
  });
});

test.describe('服務狀態 @settings @dsr-3c', () => {
  const svc = (name: string, display_name: string, status: string, extra = {}) => ({
    name,
    display_name,
    status,
    message: status,
    last_success_at: null,
    last_check_at: new Date().toISOString(),
    response_time_ms: 0,
    ...extra,
  });
  const SERVICES = {
    services: [
      svc('tmdb', 'TMDb API', 'connected', { response_time_ms: 142 }),
      svc('douban', 'Douban Scraper', 'unconfigured'),
      svc('qbittorrent', 'qBittorrent', 'error', {
        error_message: 'dial tcp 127.0.0.1:8080: connect: connection refused',
      }),
    ],
  };

  async function stubServices(page: Page, status: number, body: unknown) {
    await page.route('**/api/v1/settings/services', (route) =>
      route.request().method() === 'GET'
        ? route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
        : route.fallback()
    );
  }

  test('[P1] a broken qBittorrent gets Chinese advice whose link opens 連線設定', async ({
    page,
  }) => {
    await stubServices(page, 200, { success: true, data: SERVICES });
    await page.goto('/settings/status');
    const banner = page.getByTestId('service-error-banner');
    await expect(banner).toBeVisible({ timeout: 15000 });
    await expect(banner).toContainText(
      'qBittorrent：目前無法連線。請確認下載器已啟動，或到「連線設定」檢查位址。'
    );
    // The backend's English names are not on the page.
    await expect(page.getByText('Douban Scraper')).toHaveCount(0);
    await expect(page.getByText('豆瓣', { exact: true })).toBeVisible();

    await banner.getByRole('link', { name: '連線設定' }).click();
    await expect(page).toHaveURL(/\/settings\/connection$/);
  });

  test('[P1] phone keeps a 44px re-check button on every card; desktop is 36px', async ({
    page,
  }) => {
    await stubServices(page, 200, { success: true, data: SERVICES });
    await page.setViewportSize(PHONE);
    await page.goto('/settings/status');
    const btn = page.getByRole('button', { name: '重新檢查 豆瓣' });
    await expect(btn).toBeVisible({ timeout: 15000 });
    expect(await btn.boundingBox()).toMatchObject({ width: 44, height: 44 });
    await page.setViewportSize(DESKTOP);
    await expect.poll(async () => (await btn.boundingBox())?.width).toBe(36);
  });

  test('[P1] a failed load says so in plain words, offers 重試, and hides the backend error', async ({
    page,
  }) => {
    await stubServices(page, 500, {
      success: false,
      error: { code: 'INTERNAL_ERROR', message: 'sql: database is locked' },
    });
    await page.goto('/settings/status');
    // The app's QueryClient retries once (queryClient.ts `retry: 1`) before it gives up.
    await expect(page.getByText('無法載入服務狀態')).toBeVisible({ timeout: 20000 });
    await expect(page.getByRole('button', { name: '重試' })).toBeVisible();
    await expect(page.getByText('sql: database is locked')).toHaveCount(0);
  });
});

test.describe('備份與還原 @settings @dsr-3f', () => {
  const bk = (id: string, status: string, created: string) => ({
    id,
    filename: `vido-backup-${id}.db`,
    size_bytes: 54_000_000,
    schema_version: 17,
    checksum: '',
    status,
    created_at: created,
  });
  const LIST = {
    backups: [
      bk('b1', 'completed', '2026-09-11T03:00:00'),
      bk('b2', 'completed', '2026-09-10T03:00:00'),
      bk('b3', 'running', '2026-09-09T14:12:00'),
    ],
    total_size_bytes: 162_000_000,
  };

  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/settings/backups', (route) =>
      route.request().method() === 'GET'
        ? route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ success: true, data: LIST }),
          })
        : route.fallback()
    );
    await page.route('**/api/v1/settings/backups/schedule', (route) =>
      route.request().method() === 'GET'
        ? route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              success: true,
              data: { enabled: false, frequency: 'disabled', hour: 3, day_of_week: 0 },
            }),
          })
        : route.fallback()
    );
  });

  test('[P1] phone: every card’s restore and delete are on screen and tappable; running has no restore', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await page.goto('/settings/backup');
    await expect(page.getByTestId('backup-row-b1')).toBeVisible({ timeout: 15000 });
    for (const id of ['b1', 'b2']) {
      for (const btn of [`restore-btn-${id}`, `delete-btn-${id}`]) {
        const el = page.getByTestId(btn);
        await el.scrollIntoViewIfNeeded();
        await expect(el).toBeInViewport({ ratio: 1 });
        const box = await el.boundingBox();
        expect(box!.height).toBeGreaterThanOrEqual(44);
        expect(box!.x + box!.width).toBeLessThanOrEqual(PHONE.width);
      }
    }
    await expect(page.getByTestId('restore-btn-b3')).toHaveCount(0);
    await expect(page.getByTestId('delete-btn-b3')).toBeVisible();
  });

  test('[P1] 768 with the sidebar: the column is too narrow for the table, so it is cards and nothing is clipped', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/settings/backup');
    await expect(page.getByTestId('backup-row-b1')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('backup-list')).toHaveAttribute('data-layout', 'cards');
    const list = (await page.getByTestId('backup-table').boundingBox())!;
    for (const btn of ['restore-btn-b1', 'delete-btn-b1', 'download-btn-b1']) {
      const b = (await page.getByTestId(btn).boundingBox())!;
      expect(b.x + b.width).toBeLessThanOrEqual(list.x + list.width);
    }
  });

  for (const [label, vp] of [
    ['phone', PHONE],
    ['1440', DESKTOP],
  ] as const) {
    test(`[P1] ${label}: 還原 opens a real dialog; Esc closes it and focus returns to 還原`, async ({
      page,
    }) => {
      await page.setViewportSize(vp);
      await page.goto('/settings/backup');
      const restore = page.getByTestId('restore-btn-b2');
      await expect(restore).toBeVisible({ timeout: 15000 });
      await restore.click();
      const dialog = page.getByRole('dialog', { name: '確認還原' });
      await expect(dialog).toBeVisible();
      await expect(page.getByTestId('restore-cancel-btn')).toBeFocused();
      const box = (await dialog.boundingBox())!;
      if (label === 'phone') {
        // A bottom sheet: full width, pinned to the bottom edge.
        expect(Math.round(box.width)).toBe(PHONE.width);
        await expect
          .poll(async () =>
            Math.round((await dialog.boundingBox())!.y + (await dialog.boundingBox())!.height)
          )
          .toBe(PHONE.height);
      } else {
        // Centred.
        expect(Math.abs(box.x + box.width / 2 - DESKTOP.width / 2)).toBeLessThan(2);
      }
      await page.keyboard.press('Escape');
      await expect(dialog).toHaveCount(0);
      await expect(restore).toBeFocused();
    });
  }
});

test.describe('系統日誌 @settings @dsr-3e', () => {
  const LOGS = {
    logs: [
      {
        id: 1,
        level: 'ERROR',
        message: 'qBittorrent 連線遭拒（ECONNREFUSED 127.0.0.1:8080）',
        source: 'qbittorrent',
        created_at: '2026-09-11T09:42:18',
      },
      {
        id: 2,
        level: 'INFO',
        message: '掃描完成：1,247 個檔案，比對成功 1,198',
        source: 'scanner',
        created_at: '2026-09-11T09:40:02',
      },
    ],
    total: 2,
    page: 1,
    per_page: 50,
  };
  const EMPTY = { logs: [], total: 0, page: 1, per_page: 50 };

  async function stubLogs(page: Page) {
    await page.route('**/api/v1/settings/logs**', (route) => {
      if (route.request().method() !== 'GET') return route.fallback();
      const url = new URL(route.request().url());
      const filtered = url.searchParams.has('level') || url.searchParams.has('keyword');
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: filtered ? EMPTY : LOGS }),
      });
    });
  }

  test('[P1] phone: every message gets (nearly) the full width, not the ~80px left after the timestamp', async ({
    page,
  }) => {
    await stubLogs(page);
    await page.setViewportSize(PHONE);
    await page.goto('/settings/logs');
    const msgs = page.getByTestId('log-message');
    await expect(msgs.first()).toBeVisible({ timeout: 15000 });
    expect(await msgs.count()).toBe(2);
    for (let i = 0; i < 2; i++) {
      const box = (await msgs.nth(i).boundingBox())!;
      expect(box.width).toBeGreaterThanOrEqual(PHONE.width - 64 - 32);
    }
  });

  test('[P1] filtered to nothing: says so and 清除篩選 clears both the level and the keyword', async ({
    page,
  }) => {
    await stubLogs(page);
    await page.goto('/settings/logs');
    await expect(page.getByTestId('logs-count')).toHaveText('共 2 筆記錄', { timeout: 15000 });
    await page.getByTestId('log-filter-error').click();
    await page.getByTestId('log-keyword-input').fill('qbittorrent');
    await page.getByTestId('log-keyword-input').press('Enter');
    await expect(page.getByText('沒有符合條件的日誌記錄')).toBeVisible();
    await expect(page.getByTestId('logs-count')).toHaveText('符合條件 0 筆');
    await page.getByTestId('logs-clear-filters').click();
    await expect(page.getByTestId('logs-count')).toHaveText('共 2 筆記錄');
    await expect(page.getByTestId('log-keyword-input')).toHaveValue('');
    await expect(page.getByTestId('logs-empty')).toHaveCount(0);
  });
});
