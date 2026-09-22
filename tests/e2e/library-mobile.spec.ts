/**
 * The library page on a phone (dsr-1b-b — A3p-M `h1v1U6` entry button, A6p-M `Bz0YN`
 * sort + filter sheet).
 *
 * @tags @e2e @library-mobile
 *
 * jsdom evaluates no media queries, so the unit specs only prove the CSS tokens exist;
 * THIS spec is what sees a 390px screen: the header 篩選 icon button, the sheet pinned
 * to the bottom with a footer that stays put while the middle scrolls, the chip row
 * that scrolls sideways — and the 8-11 deep link `?subtitleStatus=not_found` finally
 * putting `subtitle_status=not_found` on the wire. It measures both sides of the 640px
 * breakpoint and the 1024 rail (feedback_measure_the_breakpoint_you_return_to).
 */
import type { Locator, Page, TestInfo } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { movieItem, stubLibraryBaseline, stubLibraryList } from '../support/helpers/library-stubs';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };
const AT_RAIL = { width: 1024, height: 844 };
const GUTTER = 16;

const ALL = [
  movieItem('m1', '你的名字', 'found'),
  movieItem('m2', '寄生上流', 'not_found'),
  movieItem('m3', '天氣之子', 'not_found'),
];
// What the stub returns depends on the filter on the wire — so a green test PROVES the
// param reached the backend; an unfiltered request always gets all three.
const pick = (url: URL) => {
  const status = url.searchParams.get('subtitle_status');
  const items = status
    ? ALL.filter((i) => status.split(',').includes(i.movie.subtitle_status))
    : ALL;
  // The sheet's footer preview asks with page_size=1 — same filter, same total.
  return url.searchParams.get('page_size') === '1'
    ? { items: items.slice(0, 1), totalItems: items.length }
    : { items };
};

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
async function settleSheet(sheet: Locator) {
  await expect(sheet).not.toHaveAttribute('data-starting-style');
  await sheet.evaluate((el) =>
    Promise.allSettled(el.getAnimations({ subtree: true }).map((a) => a.finished)).then(
      () => undefined
    )
  );
}

async function openLibrary(page: Page, path = '/library') {
  await stubLibraryBaseline(page);
  const requests = await stubLibraryList(page, pick);
  await page.goto(path);
  await expect(page.getByTestId('library-page-title')).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('library-grid-v2')).toBeVisible();
  return requests;
}

async function openSheet(page: Page, opener: Locator) {
  await opener.click();
  const sheet = page.getByTestId('library-sort-filter-sheet');
  await expect(sheet).toBeVisible();
  await settleSheet(sheet);
  return sheet;
}

const hasParam = (u: string, k: string, v: string) => new URL(u).searchParams.get(k) === v;

test.describe('媒體庫 — phone sort+filter sheet @e2e @library-mobile', () => {
  test.beforeEach(async ({ page: _page }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'chromium',
      'pixel-exact geometry — desktop chromium project only'
    );
  });

  test('[P0] 390 — the header 篩選 button opens the sheet pinned to the bottom, footer fixed, over the tab bar', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    await openLibrary(page);

    const root = await box(
      page
        .getByTestId('library-page-title')
        .locator('xpath=ancestor::div[contains(@class,"px-4")][1]')
    );
    const opener = page.getByTestId('library-filter-open-phone');
    await expect(opener).toBeVisible();
    await expect(page.getByTestId('library-filter-open')).toBeHidden();
    const btn = await box(opener);
    expect(Math.round(btn.width)).toBeGreaterThanOrEqual(44);
    expect(Math.round(btn.height)).toBeGreaterThanOrEqual(44);
    expect(Math.abs(right(btn) - (right(root) - GUTTER))).toBeLessThanOrEqual(1);

    const tabBar = await box(page.getByTestId('mobile-tab-bar'));
    const sheet = await openSheet(page, opener);
    await expect(opener).toHaveAttribute('aria-expanded', 'true');

    const s = await box(sheet);
    expect(Math.abs(s.x)).toBeLessThan(0.5);
    expect(Math.round(s.width)).toBe(PHONE.width);
    expect(Math.round(bottom(s))).toBe(PHONE.height);
    expect(s.height).toBeLessThanOrEqual(PHONE.height * 0.8);

    // 重設 is a 44×44 target; the footer does not move when the middle scrolls.
    const reset = await box(page.getByTestId('library-filter-reset'));
    expect(Math.round(reset.width)).toBeGreaterThanOrEqual(44);
    expect(Math.round(reset.height)).toBeGreaterThanOrEqual(44);
    const apply = page.getByTestId('library-filter-apply');
    const before = await box(apply);
    const scroller = sheet.locator('.overflow-y-auto').first();
    await scroller.evaluate((el) => {
      el.scrollTop = 200;
    });
    const after = await box(apply);
    expect(Math.abs(after.y - before.y)).toBeLessThan(0.5);
    expect(bottom(after)).toBeLessThanOrEqual(PHONE.height);

    const onTabBar = await page.evaluate(
      ({ x, y }) => {
        const el = document.elementFromPoint(x, y);
        const bar = document.querySelector('[data-testid="mobile-tab-bar"]');
        return Boolean(el && bar && bar.contains(el));
      },
      { x: tabBar.x + tabBar.width / 2, y: tabBar.y + tabBar.height / 2 }
    );
    expect(onTabBar).toBe(false);

    await shot(page, testInfo, 'dsr-1b-b-390-sheet.png');

    // Esc closes; focus lands back on the opener.
    await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0);
    await expect(opener).toBeFocused();
    await expect(opener).toHaveAttribute('aria-expanded', 'false');
  });

  test('[P0] 390 — pick 缺字幕, read the count, apply: the wire, the URL, the chip and the badge all say so', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    const requests = await openLibrary(page);
    const opener = page.getByTestId('library-filter-open-phone');
    const sheet = await openSheet(page, opener);

    await sheet.getByTestId('filter-subtitle-not_found').click();
    // The footer preview is a real (stubbed) query: 2 of 3 are not_found.
    await expect(sheet.getByTestId('library-filter-apply')).toHaveText('套用篩選 · 2 部');
    await sheet.getByTestId('library-filter-apply').click();
    await expect(sheet).toHaveCount(0);

    await expect
      .poll(() =>
        requests.some(
          (u) => hasParam(u, 'subtitle_status', 'not_found') && !hasParam(u, 'page_size', '1')
        )
      )
      .toBe(true);
    await expect(page).toHaveURL(/subtitleStatus=not_found/);
    await expect(page.getByRole('button', { name: '移除缺字幕篩選' })).toBeVisible();
    await expect(page.getByTestId('library-filter-open-phone-count')).toHaveText('1');
    await expect(page.getByTestId('library-result-count')).toHaveText('2 部');
    await expect(opener).toBeFocused();

    // Remove the chip: the param leaves the URL and the unfiltered list (already in the
    // 30s query cache, so no new request) comes back — 3 部 again.
    await page.getByRole('button', { name: '移除缺字幕篩選' }).click();
    await expect(page).not.toHaveURL(/subtitleStatus/);
    await expect(page.getByTestId('library-result-count')).toHaveText('3 部');
  });

  test('[P0] 390 — the 8-11 deep link ?subtitleStatus=not_found filters from the very first request', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    const requests = await openLibrary(page, '/library?subtitleStatus=not_found');
    const listRequests = requests.filter((u) => !hasParam(u, 'page_size', '1'));
    expect(listRequests.length).toBeGreaterThan(0);
    expect(listRequests.every((u) => hasParam(u, 'subtitle_status', 'not_found'))).toBe(true);
    await expect(page.getByRole('button', { name: '移除缺字幕篩選' })).toBeVisible();
    await expect(page.getByTestId('library-result-count')).toHaveText('2 部');
  });

  test('[P0] 390 — sort chips: pick 標題 (asc), apply; pick it again (desc), apply', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    const requests = await openLibrary(page);
    const opener = page.getByTestId('library-filter-open-phone');

    let sheet = await openSheet(page, opener);
    const radios = sheet.getByRole('radio');
    await expect(radios).toHaveCount(4);
    await sheet.getByTestId('library-sort-title').click();
    await sheet.getByTestId('library-filter-apply').click();
    await expect(sheet).toHaveCount(0);
    await expect
      .poll(() =>
        requests.some(
          (u) =>
            hasParam(u, 'sort_by', 'title') &&
            hasParam(u, 'sort_order', 'asc') &&
            !hasParam(u, 'page_size', '1')
        )
      )
      .toBe(true);

    sheet = await openSheet(page, opener);
    await expect(sheet.getByTestId('library-sort-title')).toHaveAttribute('aria-checked', 'true');
    await sheet.getByTestId('library-sort-title').click();
    await expect(sheet.getByTestId('library-sort-title')).toHaveAttribute('data-order', 'desc');
    await sheet.getByTestId('library-filter-apply').click();
    await expect
      .poll(() =>
        requests.some(
          (u) =>
            hasParam(u, 'sort_by', 'title') &&
            hasParam(u, 'sort_order', 'desc') &&
            !hasParam(u, 'page_size', '1')
        )
      )
      .toBe(true);
  });

  test('[P0] 390 — the active-filter chips are one bled row that really scrolls, focus ring intact', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    // Five facets → wider than 358px.
    await openLibrary(
      page,
      '/library?genres=%E5%8B%95%E7%95%AB%2C%E7%A7%91%E5%B9%BB&yearMin=2010&yearMax=2019&subtitleStatus=not_found%2Cfound&unmatched=true'
    );
    const remove = page.getByRole('button', { name: '移除缺字幕篩選' });
    await expect(remove).toBeVisible();
    const row = remove.locator('xpath=ancestor::span[1]/..');
    const chips = row.locator(':scope > span');
    const count = await chips.count();
    expect(count).toBeGreaterThanOrEqual(5);

    const boxes = await Promise.all((await chips.all()).map(box));
    const firstCentre = centreY(boxes[0]);
    for (const b of boxes) expect(Math.abs(centreY(b) - firstCentre)).toBeLessThan(2);

    const scroll = await row.evaluate((el) => {
      const before = { scrollWidth: el.scrollWidth, clientWidth: el.clientWidth };
      el.scrollLeft = 10_000;
      const moved = el.scrollLeft;
      el.scrollLeft = 0;
      return {
        ...before,
        moved,
        overflowX: window.getComputedStyle(el).overflowX,
        wrap: window.getComputedStyle(el).flexWrap,
      };
    });
    expect(scroll.overflowX).toBe('auto');
    expect(scroll.wrap).toBe('nowrap');
    expect(scroll.scrollWidth).toBeGreaterThan(scroll.clientWidth);
    expect(scroll.moved).toBeGreaterThan(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      PHONE.width
    );

    // Focus ring room: the first remove button sits ≥4px inside the scroller on every side.
    const firstBtn = chips.first().getByRole('button');
    await firstBtn.focus();
    const [chipB, rowB] = await Promise.all([box(chips.first()), box(row)]);
    expect(chipB.y - rowB.y).toBeGreaterThanOrEqual(4);
    expect(bottom(rowB) - bottom(chipB)).toBeGreaterThanOrEqual(4);
  });

  test('[P1] 640 — the other side of the breakpoint: toolbar 篩選 opens the same sheet, chips wrap', async ({
    page,
  }) => {
    await page.setViewportSize(AT_BREAKPOINT);
    await openLibrary(page, '/library?subtitleStatus=not_found');
    await expect(page.getByTestId('library-filter-open-phone')).toBeHidden();
    const toolbarBtn = page.getByTestId('library-filter-open');
    await expect(toolbarBtn).toBeVisible();
    const sheet = await openSheet(page, toolbarBtn);
    await expect(sheet.getByTestId('filter-subtitle-not_found')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0);
    await expect(toolbarBtn).toBeFocused();

    const row = page
      .getByRole('button', { name: '移除缺字幕篩選' })
      .locator('xpath=ancestor::span[1]/..');
    expect(await row.evaluate((el) => window.getComputedStyle(el).flexWrap)).toBe('wrap');
  });

  test('[P1] 1024 — the rail: no sheet openers, and the rail now has the 字幕 chips', async ({
    page,
  }) => {
    await page.setViewportSize(AT_RAIL);
    await openLibrary(page);
    await expect(page.getByTestId('library-filter-open-phone')).toBeHidden();
    await expect(page.getByTestId('library-filter-open')).toBeHidden();
    const rail = page.getByTestId('filter-panel');
    await expect(rail).toBeVisible();
    await expect(rail.getByTestId('filter-subtitle-found')).toHaveText('有字幕');
    await expect(rail.getByTestId('filter-subtitle-not_found')).toHaveText('缺字幕');
    await expect(rail.getByTestId('filter-subtitle-not_searched')).toHaveText('還沒搜尋');
  });
});
