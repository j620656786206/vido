/**
 * Season list: each episode's "has Chinese subtitles" mark and its tooltip,
 * in a REAL browser (disc-2026-10-episode-list-subtitle-badge-b, J11-D).
 *
 * jsdom cannot tell whether a tooltip actually opens on hover, on Tab, on a
 * tap, or closes on Esc and on a tap elsewhere — Base UI decides that from
 * real pointer and focus events. This spec is the layer that does.
 *
 * A seeded series has no seasons and no episode files, so the two season
 * endpoints are stubbed with the real shapes of 《末日光明》(See) S01.
 *
 * @tags @e2e @episode-subtitle-badge
 */

import type { Page } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { seedSeries, deleteSeries } from '../support/helpers/seed-helpers';

const episode = (n: number, extra: Record<string, unknown>) => ({
  episode_number: n,
  name: `第 ${n} 集`,
  air_date: '2019-11-01',
  runtime: 56,
  vote_average: 7,
  has_local_file: true,
  episode_id: `see-s01e0${n}`,
  file_path: `/media/tv/See/Season 01/See.S01E0${n}.mkv`,
  subtitle_status: 'not_searched',
  embedded_subtitles_read: true,
  chinese_subtitle_sources: [],
  ...extra,
});

async function stubSeasons(page: Page, seriesId: string) {
  await page.route(`**/api/v1/series/${seriesId}/seasons`, (route) =>
    route.fulfill({
      json: {
        success: true,
        data: [{ id: 1, season_number: 1, name: '第 1 季', episode_count: 3 }],
      },
    })
  );
  await page.route(`**/api/v1/series/${seriesId}/seasons/1/episodes`, (route) =>
    route.fulfill({
      json: {
        success: true,
        data: {
          season: { id: 1, season_number: 1, name: '第 1 季', episode_count: 3 },
          episodes: [
            episode(1, { name: '神之火', chinese_subtitle: 'none' }),
            episode(2, {
              name: '瓶中信',
              chinese_subtitle: 'zh_hant',
              chinese_subtitle_sources: [
                { kind: 'sidecar', language: 'zh-Hant', label: 'zh-TW' },
                { kind: 'vido', language: 'zh-Hant', label: 'zh-Hant' },
              ],
            }),
            episode(3, {
              name: '新的血脈',
              chinese_subtitle: 'unknown',
              embedded_subtitles_read: false,
            }),
          ],
        },
      },
    })
  );
}

async function openSeason(page: Page, seriesId: string) {
  await page.goto(`/media/tv/${seriesId}`);
  await page.getByTestId('season-header-1').click();
  await expect(page.getByTestId('episode-list')).toBeVisible();
}

const mark = (page: Page, code: string) =>
  page.locator(`[data-testid="episode-subtitle-indicator"][data-episode="${code}"]`);
const tooltip = (page: Page) => page.getByTestId('episode-subtitle-tooltip');

test.describe('季清單字幕圖示 @e2e @episode-subtitle-badge', () => {
  const seriesIds: string[] = [];

  test.afterEach(async ({ api }) => {
    await deleteSeries(api, ...seriesIds.splice(0));
  });

  test('[P0] hover opens at once, Tab opens, Esc closes; the icon does not open 管理字幕', async ({
    page,
    api,
    isMobile,
  }) => {
    test.skip(isMobile, 'hover and Tab are desktop interactions');
    const series = await seedSeries(api, {
      title: `[E2E] 季清單字幕 ${Date.now()}`,
      tmdbId: 85421,
    });
    seriesIds.push(series.id);
    await stubSeasons(page, series.id);
    await openSeason(page, series.id);

    await expect(mark(page, 'S01E02')).toHaveAccessibleName(
      '有繁中字幕，旁邊的 zh-TW 檔・Vido 生成'
    );
    await expect(mark(page, 'S01E01')).not.toHaveAttribute('title');

    // Hover: no browser-title wait (J11-D rule 4).
    await mark(page, 'S01E02').hover();
    await expect(tooltip(page)).toBeVisible({ timeout: 1500 });
    await expect(tooltip(page)).toContainText('有繁中字幕');
    await expect(tooltip(page)).toContainText('旁邊的 zh-TW 檔・Vido 生成');

    // Keyboard: Tab onto the next mark shows its tooltip, Esc closes it.
    await page.mouse.move(0, 0);
    await expect(tooltip(page)).toBeHidden();
    await mark(page, 'S01E01').focus();
    await page.keyboard.press('Shift+Tab');
    await page.keyboard.press('Tab');
    await expect(tooltip(page)).toBeVisible();
    await expect(tooltip(page)).toContainText('缺中文字幕');
    await page.keyboard.press('Escape');
    await expect(tooltip(page)).toBeHidden();

    // Pressing the icon is not the 管理字幕 action.
    await mark(page, 'S01E01').click();
    await expect(page.getByTestId('manage-subtitle-dialog-v2')).toHaveCount(0);
  });

  test('[P0] phone: a tap opens it, a tap elsewhere closes it', async ({ page, api, isMobile }) => {
    // The mobile projects (touch + phone viewport) are the ones that can tap.
    test.skip(!isMobile, 'touch only');
    const series = await seedSeries(api, {
      title: `[E2E] 季清單字幕手機 ${Date.now()}`,
      tmdbId: 85421,
    });
    seriesIds.push(series.id);
    await stubSeasons(page, series.id);
    await openSeason(page, series.id);

    await mark(page, 'S01E03').tap();
    await expect(tooltip(page)).toBeVisible();
    await expect(tooltip(page)).toContainText('還沒檢查這一集的字幕');

    await page.getByTestId('season-accordion').tap({ position: { x: 5, y: 5 } });
    await expect(tooltip(page)).toBeHidden();
  });
});
