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
import {
  ROUTE_API,
  paginated,
  stubCounts,
  stubList,
  stubQbtConfig,
} from '../support/helpers/downloads-stubs';

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

// dsr-4b-2 — the card's ⋯ on a phone: the actions sheet, the detail sheet, and the one confirm.
test.describe('下載 — phone card sheets @e2e @downloads-mobile', () => {
  test.beforeEach(async ({ page: _page }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'chromium',
      'pixel-exact geometry — desktop chromium project only'
    );
  });

  // A long, space-free path: without `break-all` it overflows the 350px cell sideways, and it
  // pushes the content past the sheet's height so the scroll/pinned-bar checks have something
  // to scroll (a real NAS path looks like this).
  const A = {
    ...presetDownloads.downloading,
    name: 'Dune.Part.Two.2024.2160p.UHD.BluRay.Remux.HEVC.DV.HDR10Plus.TrueHD.Atmos.7.1-FraMeSToR.mkv',
    savePath:
      '/volume1/media/movies/Dune.Part.Two.2024.2160p.UHD.BluRay.Remux.HEVC.DV.HDR10Plus.TrueHD.Atmos.7.1-FraMeSToR/Extras/Featurettes/',
  };
  const moreBtn = (page: Page) => page.getByRole('button', { name: `更多動作：${A.name}` });

  /** The list stub flips A to paused once a pause POST has landed — a refetch right after the
   *  optimistic update must not bounce the pill back to 下載中 (removal likewise drops A). */
  async function stubMutations(page: Page) {
    const state = { paused: false, removed: false, deletes: [] as string[], pauses: 0, lists: 0 };
    await page.route(`${ROUTE_API}/downloads*`, (route) => {
      state.lists += 1;
      const a = state.paused ? { ...A, status: 'paused', downloadSpeed: 0 } : A;
      const items = state.removed ? [presetDownloads.seeding] : [a, presetDownloads.seeding];
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(paginated(items)),
      });
    });
    await stubCounts(page, { all: 2, downloading: 1, seeding: 1 });
    await page.route(/\/api\/v1\/downloads\/[^/?]+(\?.*)?$/, (route) => {
      if (route.request().method() === 'DELETE') {
        state.removed = true;
        state.deletes.push(route.request().url());
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: null }),
        });
      }
      return route.fallback();
    });
    await page.route(/\/api\/v1\/downloads\/[^/]+\/pause$/, (route) => {
      state.paused = true;
      state.pauses += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: null }),
      });
    });
    return state;
  }

  async function openPhone(page: Page) {
    await page.setViewportSize(PHONE);
    await stubQbtConfig(page);
    const state = await stubMutations(page);
    await page.goto('/downloads');
    await expect(page.getByTestId('downloads-browse-v2')).toBeVisible({ timeout: 15000 });
    await expect(moreBtn(page)).toBeVisible();
    return state;
  }

  async function openActionsSheet(page: Page) {
    await moreBtn(page).click();
    const sheet = page.getByTestId('download-actions-sheet');
    await expect(sheet).toBeVisible();
    await settleSheet(sheet);
    return sheet;
  }

  const activeIsBody = (page: Page) =>
    page.evaluate(() => document.activeElement === document.body);

  test('[P0] 390 — ⋯ opens the actions sheet pinned to the bottom; 保留檔案 removes at once', async ({
    page,
  }, testInfo) => {
    const state = await openPhone(page);
    const sheet = await openActionsSheet(page);
    expect(await page.getByRole('menu').count()).toBe(0);

    const s = await box(sheet);
    expect(Math.abs(s.x)).toBeLessThan(0.5);
    expect(Math.round(s.width)).toBe(PHONE.width);
    expect(Math.round(bottom(s))).toBe(PHONE.height);
    await expect(page.getByRole('dialog')).toHaveAccessibleName(A.name);

    const rows = sheet.getByRole('button');
    await expect(rows).toHaveCount(4);
    for (const r of await rows.all()) {
      const b = await box(r);
      expect(Math.round(b.height)).toBeGreaterThanOrEqual(52);
      expect(Math.abs(b.width - (PHONE.width - 16))).toBeLessThanOrEqual(1);
    }
    await shot(page, testInfo, 'dsr-4b-2-390-actions.png');

    await sheet.getByRole('button', { name: '移除（保留檔案）' }).click();
    await expect(sheet).toHaveCount(0);
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect.poll(() => state.deletes.length).toBe(1);
    expect(state.deletes[0]).not.toContain('deleteFiles=true');
    // The card is gone (optimistic) → focus lands on the heading, never <body>.
    await expect(page.locator(`[data-testid="download-card-v2-${A.hash}"]`)).toHaveCount(0);
    await expect.poll(() => activeIsBody(page)).toBe(false);
    await expect(page.getByRole('heading', { level: 1, name: '下載' })).toBeFocused();
  });

  test('[P1] 390 · reduced motion — 保留檔案 still lands focus on the heading (the 1ms exit is enough)', async ({
    browser,
  }) => {
    // The close-then-unmount fix rides on a transitionend; under reduced motion that transition
    // is 1ms, never 0 — this is the test that notices if someone makes it 0.
    const ctx = await browser.newContext({ reducedMotion: 'reduce', viewport: PHONE });
    const page = await ctx.newPage();
    try {
      await stubQbtConfig(page);
      const state = await stubMutations(page);
      await page.goto('/downloads');
      await expect(moreBtn(page)).toBeVisible({ timeout: 15000 });
      const sheet = await openActionsSheet(page);
      await sheet.getByRole('button', { name: '移除（保留檔案）' }).click();
      await expect(sheet).toHaveCount(0);
      await expect.poll(() => state.deletes.length).toBe(1);
      await expect.poll(() => activeIsBody(page)).toBe(false);
      await expect(page.getByRole('heading', { level: 1, name: '下載' })).toBeFocused();
    } finally {
      await ctx.close();
    }
  });

  test('[P0] 390 — 連同檔案刪除: sheet first, then the confirm; 取消 → ⋯; 刪除檔案 → DELETE + heading', async ({
    page,
  }) => {
    const state = await openPhone(page);
    let sheet = await openActionsSheet(page);
    await sheet.getByRole('button', { name: '移除（連同檔案刪除）' }).click();
    const confirm = page.getByRole('dialog', { name: '移除並刪除檔案？' });
    // 先關再開: right after the tap the sheet is still exiting (320ms) and the confirm must NOT
    // exist yet — otherwise it animates in UNDER the exiting popup (z-71) and its scrim.
    expect(await confirm.count()).toBe(0);
    await expect(sheet).toHaveCount(0);
    await expect(confirm).toBeVisible();
    expect(await page.getByTestId('download-actions-sheet').count()).toBe(0);
    // …and its buttons are actually on top: a hit test lands on the button itself.
    const del = await box(confirm.getByRole('button', { name: '刪除檔案' }));
    expect(
      await page.evaluate(
        ({ x, y }) => document.elementFromPoint(x, y)?.closest('button')?.textContent ?? '',
        { x: del.x + del.width / 2, y: del.y + del.height / 2 }
      )
    ).toContain('刪除檔案');
    expect(await confirm.evaluate((el) => el.contains(document.activeElement))).toBe(true);
    await expect(confirm).toContainText(A.name);

    await confirm.getByRole('button', { name: '取消' }).click();
    await expect(confirm).toHaveCount(0);
    await expect(moreBtn(page)).toBeFocused();
    expect(state.deletes).toEqual([]);

    sheet = await openActionsSheet(page);
    await sheet.getByRole('button', { name: '移除（連同檔案刪除）' }).click();
    await expect(confirm).toBeVisible();
    await confirm.getByRole('button', { name: '刪除檔案' }).click();
    await expect(confirm).toHaveCount(0);
    await expect.poll(() => state.deletes.length).toBe(1);
    expect(state.deletes[0]).toContain('deleteFiles=true');
    await expect.poll(() => activeIsBody(page)).toBe(false);
    await expect(page.getByRole('heading', { level: 1, name: '下載' })).toBeFocused();
  });

  test('[P0] 390 — 詳細資訊: one sheet at a time, hash/path do not overflow, bar stays put, 暫停 keeps it open', async ({
    page,
  }, testInfo) => {
    const state = await openPhone(page);
    // A short phone (iPhone SE height): the sheet hits its 85vh cap, so the content region has
    // to scroll while the action bar stays pinned — the case the sheet's layout exists for.
    const SHORT = { width: PHONE.width, height: 667 };
    await page.setViewportSize(SHORT);
    const actions = await openActionsSheet(page);
    await actions.getByRole('button', { name: '詳細資訊' }).click();
    const detail = page.getByTestId('download-detail-sheet');
    await expect(detail).toBeVisible();
    await settleSheet(detail);
    await expect(page.locator('[data-testid$="-sheet"]:visible')).toHaveCount(1);
    // 先關再開: the actions sheet is gone before the detail sheet opens, and the handoff leaves
    // focus INSIDE the new sheet — the old one must not pull it back to ⋯.
    await expect(actions).toHaveCount(0);
    expect(await detail.evaluate((el) => el.contains(document.activeElement))).toBe(true);

    const dds = detail.getByRole('definition');
    await expect(dds.nth(6)).toHaveText(A.hash);
    await expect(dds.nth(7)).toHaveText(A.savePath);
    for (const i of [6, 7]) {
      const m = await dds.nth(i).evaluate((el) => ({ sw: el.scrollWidth, cw: el.clientWidth }));
      expect(m.sw).toBeLessThanOrEqual(m.cw);
    }
    // The sheet never grows wider than the screen.
    expect(Math.round((await box(detail)).width)).toBe(PHONE.width);

    // Content scrolls INSIDE the sheet (the sheet itself is capped, the inner region overflows),
    // and the action bar is pinned: scroll to the bottom, the bar does not move.
    const bar = detail.getByRole('button', { name: `暫停 ${A.name}` });
    const before = await box(bar);
    const scroller = detail.locator('.overflow-y-auto').first();
    const sizes = await scroller.evaluate((el) => ({ sh: el.scrollHeight, ch: el.clientHeight }));
    expect(sizes.sh).toBeGreaterThan(sizes.ch);
    expect((await box(detail)).height).toBeLessThanOrEqual(SHORT.height * 0.85 + 1);
    expect(Math.round(bottom(await box(detail)))).toBe(SHORT.height);
    await scroller.evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });
    expect(await scroller.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
    const after = await box(bar);
    expect(Math.abs(after.y - before.y)).toBeLessThan(1);
    await shot(page, testInfo, 'dsr-4b-2-390-detail.png');

    // Primary button: pause → request goes, sheet STAYS, pill flips and stays flipped.
    const listsBefore = state.lists;
    await bar.click();
    await expect.poll(() => state.pauses).toBe(1);
    await expect(detail).toBeVisible();
    await expect(detail.getByTestId(`download-status-${A.hash}`)).toHaveText('已暫停');
    await expect(detail.getByRole('button', { name: `繼續 ${A.name}` })).toBeVisible();
    // …and after the refetch that follows the optimistic update it is still paused.
    await expect.poll(() => state.lists).toBeGreaterThan(listsBefore);
    await expect(detail.getByTestId(`download-status-${A.hash}`)).toHaveText('已暫停');

    // ⋯ hands back to the actions sheet; Esc from there returns focus to the card's ⋯.
    await detail.getByRole('button', { name: `更多動作：${A.name}` }).click();
    await expect(actions).toBeVisible();
    await expect(detail).toHaveCount(0);
    await settleSheet(actions);
    await page.keyboard.press('Escape');
    await expect(actions).toHaveCount(0);
    await expect(moreBtn(page)).toBeFocused();
  });

  test('[P1] 640 — the other side: ⋯ is the dropdown menu, no sheet, no 詳細資訊', async ({
    page,
  }) => {
    await page.setViewportSize(AT_BREAKPOINT);
    await stubQbtConfig(page);
    await stubMutations(page);
    await page.goto('/downloads');
    await expect(page.getByTestId('downloads-browse-v2')).toBeVisible({ timeout: 15000 });
    await moreBtn(page).click();
    await expect(page.getByRole('menu')).toBeVisible();
    expect(await page.locator('[data-testid$="-sheet"]').count()).toBe(0);
    expect(await page.getByRole('menuitem', { name: '詳細資訊' }).count()).toBe(0);
    await expect(page.getByRole('menuitem', { name: '移除（保留檔案）' })).toBeVisible();
  });
});
