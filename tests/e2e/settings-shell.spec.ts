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
