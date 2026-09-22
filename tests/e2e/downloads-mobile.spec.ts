/**
 * The downloads page on a phone (dsr-4b-1 — D1-M-v2 `uMDjw`, D10-M-v2 `JxMWL`).
 *
 * @tags @e2e @downloads-mobile
 *
 * jsdom evaluates no media queries, so the unit specs only prove the CSS tokens
 * exist; THIS spec is what sees a 390px screen: one chip row that scrolls
 * sideways, a 排序 button instead of the native select, and the sort sheet that
 * slides over the tab bar. It measures both sides of the 640px breakpoint
 * (feedback_measure_the_breakpoint_you_return_to).
 *
 * Horizontal positions are measured RELATIVE to the page root: from 640 up the app
 * sidebar appears and shifts the whole page right.
 */
import type { Locator, Page, TestInfo } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { presetDownloads } from '../support/fixtures/factories/download-factory';
import { stubCounts, stubList, stubQbtConfig } from '../support/helpers/downloads-stubs';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };
const GUTTER = 16;

const LIST = [
  presetDownloads.downloading,
  presetDownloads.paused,
  presetDownloads.completed,
  presetDownloads.seeding,
  presetDownloads.error,
];
// error > 0 so the 錯誤 chip renders too: six chips are wider than 358px.
const COUNTS = { all: 5, downloading: 1, paused: 1, completed: 1, seeding: 1, error: 1 };

/** Step-9 evidence, not baselines: only with `DSR_SHOTS=1`, into the test's own output dir. */
async function shot(page: Page, testInfo: TestInfo, name: string) {
  if (!process.env.DSR_SHOTS) return;
  await page.screenshot({ path: testInfo.outputPath(name), fullPage: true });
}

type Box = { x: number; y: number; width: number; height: number };
const box = async (l: Locator): Promise<Box> => {
  const b = await l.boundingBox();
  if (!b) throw new Error('no bounding box');
  return b;
};
const centreY = (b: Box) => b.y + b.height / 2;
const right = (b: Box) => b.x + b.width;
const bottom = (b: Box) => b.y + b.height;
// Base UI's sheet is a CSS transition. Its `data-starting-style` comes off one frame after
// mount, and until then there is no transition to wait for — so wait for the attribute to
// go first, then for whatever is running.
async function settleSheet(sheet: Locator) {
  await expect(sheet).not.toHaveAttribute('data-starting-style');
  await sheet.evaluate((el) =>
    Promise.allSettled(el.getAnimations({ subtree: true }).map((a) => a.finished)).then(
      () => undefined
    )
  );
}

async function openDownloads(page: Page, path = '/downloads') {
  await stubQbtConfig(page);
  const requests = await stubList(page, LIST);
  await stubCounts(page, COUNTS);
  await page.goto(path);
  await expect(page.getByTestId('downloads-browse-v2')).toBeVisible({ timeout: 15000 });
  await expect(page.locator('[data-testid^="download-card-v2-"]').first()).toBeVisible();
  return requests;
}

async function openSortSheet(page: Page) {
  await page.getByTestId('downloads-sort-btn').click();
  const sheet = page.getByTestId('download-sort-sheet');
  await expect(sheet).toBeVisible();
  await settleSheet(sheet);
  return sheet;
}

test.describe('下載 — phone sort sheet + chip row @e2e @downloads-mobile', () => {
  test.beforeEach(async ({ page: _page }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'chromium',
      'pixel-exact geometry — desktop chromium project only'
    );
  });

  test('[P0] 390 — the status chips are one bled row that really scrolls, focus ring intact', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    await openDownloads(page);

    const root = await box(page.getByTestId('downloads-browse-v2'));
    const row = page.getByRole('tablist', { name: '下載狀態篩選' });
    const chips = row.getByRole('tab');
    await expect(chips).toHaveCount(6);

    const boxes = await Promise.all((await chips.all()).map(box));
    const firstCentre = centreY(boxes[0]);
    for (const b of boxes) {
      expect(Math.abs(centreY(b) - firstCentre)).toBeLessThan(2);
      expect(Math.round(b.height)).toBeGreaterThanOrEqual(44);
    }

    // Bled to the screen edges, first chip on the 16px gutter.
    const rowB = await box(row);
    expect(Math.abs(rowB.x - root.x)).toBeLessThan(0.5);
    expect(Math.round(rowB.width)).toBe(PHONE.width);
    expect(Math.round(boxes[0].x - root.x)).toBe(GUTTER);

    // A scroller, not a clipper: overflow-x is auto and scrollLeft actually moves.
    const scroll = await row.evaluate((el) => {
      const before = { scrollWidth: el.scrollWidth, clientWidth: el.clientWidth };
      el.scrollLeft = 10_000;
      const moved = el.scrollLeft;
      el.scrollLeft = 0;
      return { ...before, moved, overflowX: window.getComputedStyle(el).overflowX };
    });
    expect(scroll.overflowX).toBe('auto');
    expect(scroll.scrollWidth).toBeGreaterThan(scroll.clientWidth);
    expect(scroll.moved).toBeGreaterThan(0);
    // The page itself never scrolls sideways — only the chip row does.
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      PHONE.width
    );

    // Tab (keyboard, so :focus-visible paints) to the first chip. The 2px outline + 2px
    // offset needs 4px of room on every side inside the scroller, or overflow clips it.
    for (let i = 0; i < 20; i++) {
      await page.keyboard.press('Tab');
      if (await chips.first().evaluate((el) => el === document.activeElement)) break;
    }
    await expect(chips.first()).toBeFocused();
    const ring = await chips.first().evaluate((el) => ({
      w: window.getComputedStyle(el).outlineWidth,
      o: window.getComputedStyle(el).outlineOffset,
    }));
    expect(ring).toEqual({ w: '2px', o: '2px' });
    const [chipB, rowNow] = await Promise.all([box(chips.first()), box(row)]);
    expect(chipB.y - rowNow.y).toBeGreaterThanOrEqual(4);
    expect(bottom(rowNow) - bottom(chipB)).toBeGreaterThanOrEqual(4);
    expect(chipB.x - rowNow.x).toBeGreaterThanOrEqual(4);

    await shot(page, testInfo, 'dsr-4b-1-390-chips.png');
  });

  test('[P1] 390 — an active chip that starts off-screen is scrolled into the row', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openDownloads(page, '/downloads?filter=seeding');
    const active = page.getByRole('tab', { name: /做種/ });
    await expect(active).toHaveAttribute('aria-selected', 'true');
    await expect
      .poll(async () => {
        const b = await box(active);
        return b.x >= 0 && right(b) <= PHONE.width;
      })
      .toBe(true);
  });

  test('[P0] 390 — 排序 opens the sheet over the tab bar; picking sorts, closes and returns focus', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    const requests = await openDownloads(page);

    const root = await box(page.getByTestId('downloads-browse-v2'));
    const sortBtn = page.getByTestId('downloads-sort-btn');
    await expect(sortBtn).toBeVisible();
    const btn = await box(sortBtn);
    expect(Math.round(btn.width)).toBeGreaterThanOrEqual(44);
    expect(Math.round(btn.height)).toBeGreaterThanOrEqual(44);
    expect(Math.abs(right(btn) - (right(root) - GUTTER))).toBeLessThanOrEqual(1);
    await expect(page.getByRole('combobox', { name: '排序方式' })).toBeHidden();

    const tabBar = await box(page.getByTestId('mobile-tab-bar'));

    const sheet = await openSortSheet(page);
    await expect(sortBtn).toHaveAttribute('aria-expanded', 'true');

    // Pinned to the bottom edge, full width. (Math.round turns -0.3 into -0, which
    // toBe(0) rejects — compare the magnitude instead.)
    const s = await box(sheet);
    expect(Math.abs(s.x)).toBeLessThan(0.5);
    expect(Math.round(s.width)).toBe(PHONE.width);
    expect(Math.round(bottom(s))).toBe(PHONE.height);
    // DESIGN.md: a sheet stops at 80% of the screen. (ui/Sheet's own cap is 85vh, so this
    // guards the content — eight rows must fit under the tighter limit.)
    expect(s.height).toBeLessThanOrEqual(PHONE.height * 0.8);

    const radios = sheet.getByRole('radio');
    await expect(radios).toHaveCount(8);
    // Every row keeps its icon slot, so all eight labels start at the same x —
    // the D10-M draft moved the checked row's label 32px right.
    const labelXs: number[] = [];
    for (const radio of await radios.all()) {
      const r = await box(radio);
      expect(Math.round(r.height)).toBeGreaterThanOrEqual(52);
      labelXs.push((await box(radio.locator('span').last())).x);
    }
    expect(Math.max(...labelXs) - Math.min(...labelXs)).toBeLessThanOrEqual(1);

    // The modal sheet covers the bottom tab bar (the scrim sits over the rest).
    const onTabBar = await page.evaluate(
      ({ x, y }) => {
        const el = document.elementFromPoint(x, y);
        const bar = document.querySelector('[data-testid="mobile-tab-bar"]');
        return Boolean(el && bar && bar.contains(el));
      },
      { x: tabBar.x + tabBar.width / 2, y: tabBar.y + tabBar.height / 2 }
    );
    expect(onTabBar).toBe(false);

    await shot(page, testInfo, 'dsr-4b-1-390-sort-sheet.png');

    await sheet.getByRole('radio', { name: '名稱（A–Z）' }).click();
    await expect(sheet).toHaveCount(0);
    await expect
      .poll(() => requests.some((u) => u.includes('sort=name') && u.includes('order=asc')))
      .toBe(true);
    await expect(sortBtn).toBeFocused();
    await expect(sortBtn).toHaveAttribute('aria-expanded', 'false');

    // Reopen: the choice stuck. Esc closes, focus back on 排序.
    await openSortSheet(page);
    await expect(sheet.getByRole('radio', { name: '名稱（A–Z）' })).toHaveAttribute(
      'aria-checked',
      'true'
    );
    await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0);
    await expect(sortBtn).toBeFocused();

    // Tapping the scrim closes it too.
    await openSortSheet(page);
    await page.mouse.click(PHONE.width / 2, 100);
    await expect(sheet).toHaveCount(0);
    await expect(sortBtn).toBeFocused();
  });

  test('[P1] 390 — focus returns to 排序 even when the tap never focused it (Safari)', async ({
    page,
  }) => {
    // Safari does not focus a button on tap, so Base UI's default "back to whatever had
    // focus" would land on <body>. A synthetic click reproduces that: it moves no focus.
    await page.setViewportSize(PHONE);
    await openDownloads(page);
    const sortBtn = page.getByTestId('downloads-sort-btn');
    await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
    await sortBtn.dispatchEvent('click');
    const sheet = page.getByTestId('download-sort-sheet');
    await expect(sheet).toBeVisible();
    await settleSheet(sheet);
    await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0);
    await expect(sortBtn).toBeFocused();
  });

  test('[P1] 390 → 640 — rotating past sm closes the sheet and hands focus to the select', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openDownloads(page);
    const sheet = await openSortSheet(page);
    await page.setViewportSize(AT_BREAKPOINT);
    await expect(sheet).toHaveCount(0);
    await expect(page.getByTestId('downloads-sort-btn')).toBeHidden();
    await expect(page.getByRole('combobox', { name: '排序方式' })).toBeFocused();
  });

  test('[P1] 640 — the other side of the breakpoint: no 排序 button, the select works, chips wrap', async ({
    page,
  }) => {
    await page.setViewportSize(AT_BREAKPOINT);
    const requests = await openDownloads(page);

    await expect(page.getByTestId('downloads-sort-btn')).toBeHidden();
    const select = page.getByRole('combobox', { name: '排序方式' });
    await expect(select).toBeVisible();
    await select.selectOption('name:asc');
    await expect
      .poll(() => requests.some((u) => u.includes('sort=name') && u.includes('order=asc')))
      .toBe(true);

    const row = page.getByRole('tablist', { name: '下載狀態篩選' });
    expect(await row.evaluate((el) => window.getComputedStyle(el).flexWrap)).toBe('wrap');
  });
});
