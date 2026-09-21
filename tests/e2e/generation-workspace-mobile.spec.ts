/**
 * Generation WORKSPACE on a phone (dsr-6f-4 — F11-M-v2 `PXB0z`).
 *
 * @tags @e2e @generation-workspace-mobile
 *
 * The workspace is a PAGE, not a Portal sheet, so it has no honest phone visual
 * fixture: an in-page 390 fixture is only 326 wide (the gallery's p-8) and an
 * element taller than the window photographs the fixed tab bar with it (dsr-6f-2
 * CR M7 / dsr-6f-1 CR L8). jsdom evaluates no media queries either — so THIS spec
 * is the only thing that sees what a 390px screen draws. It measures both sides of
 * the 640px breakpoint (feedback_measure_the_breakpoint_you_return_to).
 *
 * Every horizontal position is measured RELATIVE to the workspace root: from 640
 * up the app sidebar appears and shifts the whole page right.
 */
import type { Locator, Page, Route, TestInfo } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import {
  ROUTE_API,
  ITEMS,
  jsonOk,
  snapshot,
  stubCommon,
  stubEventsOnce,
  finishedFilmsFrames,
} from '../support/helpers/generation-workspace-stubs';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };
const GUTTER = 16;

/**
 * Step-9 evidence, not baselines: only when asked (`DSR_SHOTS=1`), and into the
 * test's own output dir — never an unconditional artefact on every CI run.
 */
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
const settle = (l: Locator) =>
  l.evaluate((el) =>
    Promise.allSettled(el.getAnimations({ subtree: true }).map((a) => a.finished)).then(
      () => undefined
    )
  );

async function openRunning(page: Page, frames?: string) {
  await stubCommon(page);
  await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
    route.fulfill(jsonOk({ running: true, progress: snapshot(), last: null }))
  );
  if (frames) await stubEventsOnce(page, frames);
  await page.goto('/activity?view=generation');
  await expect(page.getByTestId('workspace-overall')).toBeVisible();
}

test.describe('生成工作區 — phone page @e2e @generation-workspace-mobile', () => {
  test.beforeEach(async ({ page: _page }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'chromium',
      'pixel-exact geometry — desktop chromium project only'
    );
  });

  test('[P0] 390 — back button, title row, 16px gutters, overall strip in three rows', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    await openRunning(page);

    const root = await box(page.getByTestId('generation-workspace'));
    const overall = await box(page.getByTestId('workspace-overall'));
    expect(Math.round(overall.x - root.x)).toBe(GUTTER);
    expect(Math.round(overall.width)).toBe(PHONE.width - 2 * GUTTER);

    // Header: [‹ title ……… pill] then the breadcrumb underneath.
    const back = page.getByTestId('workspace-back');
    const h1 = page.getByRole('heading', { level: 1 });
    const pill = page.getByTestId('workspace-status-pill');
    const crumb = page.getByRole('navigation', { name: '麵包屑' });
    const [backB, h1B, pillB, crumbB, glyphB] = await Promise.all([
      box(back),
      box(h1),
      box(pill),
      box(crumb),
      box(back.locator('svg')),
    ]);
    expect(backB.width).toBeGreaterThanOrEqual(44);
    expect(backB.height).toBeGreaterThanOrEqual(44);
    expect(Math.round(glyphB.x - root.x)).toBe(GUTTER);
    expect(Math.round(h1B.x - right(glyphB))).toBe(10);
    expect(Math.abs(centreY(backB) - centreY(h1B))).toBeLessThan(2);
    expect(await h1.evaluate((el) => window.getComputedStyle(el).fontSize)).toBe('18px');
    expect(Math.abs(centreY(pillB) - centreY(h1B))).toBeLessThan(2);
    expect(Math.round(right(pillB) - root.x)).toBe(PHONE.width - GUTTER);
    expect(crumbB.y).toBeGreaterThan(bottom(h1B) - 1);
    // F11-M `e6nOTG` gap 2 between the title row and the breadcrumb; 14 from the
    // header to the body (`PXB0z` gap md-plus).
    const titleRowB = await box(page.getByTestId('workspace-title-row'));
    expect(Math.abs(crumbB.y - bottom(titleRowB) - 2)).toBeLessThanOrEqual(1);
    const queueHeading = await box(page.getByRole('heading', { level: 2, name: '生成佇列' }));
    // …and the breadcrumb is still ABOVE the strip (an `order` on the nav would
    // have dropped it to the end of the header column).
    expect(bottom(crumbB)).toBeLessThan(overall.y + 1);
    expect(await crumb.evaluate((el) => window.getComputedStyle(el).fontSize)).toBe('12px');

    // Overall strip: count + chip / full-width bar / cost on one line.
    const strip = page.getByTestId('workspace-overall');
    const bar = await box(strip.getByRole('progressbar', { name: '整批生成進度' }));
    const chip = await box(strip.getByTestId('workspace-sse-chip'));
    const count = await box(strip.getByText('已完成'));
    const costLabel = await box(strip.getByText('本次用量'));
    const spent = await box(strip.getByText('$0.42'));
    // 358 − 2 (border) − 28 (px-3.5)
    expect(Math.abs(bar.width - 328)).toBeLessThanOrEqual(1);
    expect(Math.abs(centreY(chip) - centreY(count))).toBeLessThan(6);
    expect(Math.abs(right(chip) - (right(overall) - 1 - 14))).toBeLessThanOrEqual(1);
    expect(bar.y).toBeGreaterThan(bottom(count) - 1);
    expect(costLabel.y).toBeGreaterThan(bottom(bar) - 1);
    expect(Math.abs(bottom(costLabel) - bottom(spent))).toBeLessThan(6);
    expect(costLabel.x).toBeLessThan(spent.x);
    // The 全部取消 row is 44 tall and centres the 24px heading: (44 − 24) / 2 = 10.
    expect(Math.abs(queueHeading.y - bottom(overall) - 14 - 10)).toBeLessThanOrEqual(2);

    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      PHONE.width
    );

    await shot(page, testInfo, '390-running.png');

    await back.click();
    await expect(page).not.toHaveURL(/view=generation/);
    await expect(page.getByTestId('generation-workspace')).toHaveCount(0);
  });

  test('[P0] 390 — the live log starts collapsed; open, it scrolls inside itself and follows the newest row', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    await openRunning(page, finishedFilmsFrames(40));

    const log = page.getByTestId('workspace-event-log');
    const toggle = page.getByTestId('workspace-log-toggle');
    const list = log.getByRole('list', { name: '生成事件日誌' });
    await expect(log.getByTestId('workspace-feed-row')).toHaveCount(40);

    await expect(toggle).toBeVisible();
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
    const toggleB = await box(toggle);
    expect(toggleB.width).toBeGreaterThanOrEqual(44);
    expect(toggleB.height).toBeGreaterThanOrEqual(44);
    await expect(list).toBeHidden();
    await expect(page.getByTestId('workspace-log-hint')).toBeVisible();
    await expect(log.getByText('僅狀態事件，不含逐字內容')).toBeHidden();
    // (No chip assertion: the stubbed stream ENDS after its frames, so the log is
    // honestly not `connected` here — the chip's presence is covered in the unit spec.)
    // Collapsed or not, results are still announced from the one live region.
    await expect(page.getByTestId('workspace-log-announcer')).toHaveCount(1);

    // The WHOLE header is the hit area: a click on the far-left title lands on the
    // toggle's stretched ::after (boundingBox cannot see a pseudo-element).
    const title = log.getByText('即時活動', { exact: true });
    // Centre it first: elementFromPoint takes VIEWPORT coordinates, and with 40 rows
    // of queue above it the header otherwise sits under the fixed tab bar.
    await title.evaluate((el) => el.scrollIntoView({ block: 'center' }));
    const titleB = await box(title);
    expect(
      await page.evaluate(
        ([x, y]) => document.elementFromPoint(x, y)?.getAttribute('data-testid'),
        [titleB.x + 4, centreY(titleB)]
      )
    ).toBe('workspace-log-toggle');
    // A raw mouse click: locator.click() would (rightly) report that the toggle's
    // ::after intercepts the title — which is exactly the design.
    await page.mouse.click(titleB.x + 4, centreY(titleB));
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    await settle(log);

    await expect(list).toBeVisible();
    await expect(page.getByTestId('workspace-log-hint')).toBeHidden();
    await expect(log.getByText('僅狀態事件，不含逐字內容')).toBeVisible();
    const listB = await box(list);
    expect(listB.height).toBeLessThanOrEqual(320 + 1);
    await expect
      .poll(() =>
        list.evaluate((el) => ({
          scrolls: el.scrollHeight > el.clientHeight,
          atBottom: el.scrollTop + el.clientHeight >= el.scrollHeight - 2,
          // Has teeth: a log row is a no-wrap flex line — overflow WIDENS it.
          fits: el.scrollWidth <= el.clientWidth,
        }))
      )
      .toEqual({ scrolls: true, atBottom: true, fits: true });
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      PHONE.width
    );
    await shot(page, testInfo, '390-log-open.png');

    // Scroll up to read, collapse, re-open → back on the newest row.
    await list.evaluate((el) => {
      el.scrollTop = 0;
    });
    await toggle.click();
    await expect(list).toBeHidden();
    await toggle.click();
    await expect
      .poll(() => list.evaluate((el) => el.scrollTop + el.clientHeight >= el.scrollHeight - 2))
      .toBe(true);
  });

  test('[P1] 390 — cancel-confirm: the sentence gets its own line, the two buttons share one', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openRunning(page);
    const root = await box(page.getByTestId('generation-workspace'));

    await page.getByTestId('workspace-cancel-all').click();
    const confirm = page.getByTestId('workspace-cancel-confirm');
    const sentence = await box(confirm.getByText('確定要取消整個批次嗎？已完成的字幕會保留。'));
    const keep = await box(confirm.getByRole('button', { name: '繼續生成' }));
    const go = await box(page.getByTestId('workspace-cancel-confirm-btn'));
    expect(keep.y).toBeGreaterThan(bottom(sentence) - 1);
    expect(Math.abs(centreY(keep) - centreY(go))).toBeLessThan(2);
    expect(Math.abs(keep.width - go.width)).toBeLessThanOrEqual(1);
    expect(Math.round(keep.x - root.x)).toBeGreaterThanOrEqual(GUTTER);
    expect(Math.round(right(go) - root.x)).toBeLessThanOrEqual(PHONE.width - GUTTER);
    expect(keep.height).toBeGreaterThanOrEqual(44);
  });

  test('[P0] 390 — budget ceiling: equal footer buttons, clear of the tab bar; the stop line shows while collapsed', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(PHONE);
    await stubCommon(page);
    await page.route(`${ROUTE_API}/subtitles/generation-batch/status`, (route: Route) =>
      route.fulfill(
        jsonOk({
          running: false,
          progress: null,
          last: snapshot({
            status: 'budget_ceiling',
            success_count: 1,
            fail_count: 1,
            paused_count: 1,
            spent_usd: 5,
            current_media_id: '',
            current_item: '',
            items: ITEMS.map((it) => (it.status === 'running' ? { ...it, status: 'paused' } : it)),
          }),
        })
      )
    );
    await page.goto('/activity?view=generation');

    const close = page.getByTestId('workspace-close');
    const resume = page.getByTestId('workspace-resume');
    await expect(resume).toBeVisible();
    await expect(page.getByTestId('workspace-event-log')).toContainText('已停止（達預算上限）');
    await expect(page.getByTestId('workspace-log-toggle')).toHaveAttribute(
      'aria-expanded',
      'false'
    );

    // Scroll the PAGE to its end. scrollIntoViewIfNeeded would park the footer at
    // the window's bottom edge — under the fixed tab bar — and fail a correct page.
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    const [closeB, resumeB, footerB, tabB] = await Promise.all([
      box(close),
      box(resume),
      box(page.getByTestId('workspace-footer')),
      box(page.getByTestId('mobile-tab-bar')),
    ]);
    expect(Math.abs(centreY(closeB) - centreY(resumeB))).toBeLessThan(2);
    expect(Math.abs(closeB.width - resumeB.width)).toBeLessThanOrEqual(1);
    expect(Math.abs(closeB.width - (PHONE.width - 2 * GUTTER - 12) / 2)).toBeLessThanOrEqual(1);
    expect(bottom(footerB)).toBeLessThanOrEqual(tabB.y + 1);

    // No SSE chip in a terminal mode: the count keeps row 1 to itself and the bar
    // still gets its own full-width row.
    const strip = page.getByTestId('workspace-overall');
    await expect(strip.getByTestId('workspace-sse-chip')).toHaveCount(0);
    const bar = await box(strip.getByRole('progressbar', { name: '整批生成進度' }));
    const count = await box(strip.getByText('已完成'));
    expect(bar.y).toBeGreaterThan(bottom(count) - 1);
    expect(Math.abs(bar.width - 328)).toBeLessThanOrEqual(1);
    await shot(page, testInfo, '390-budget-ceiling.png');
  });

  test('[P0] 640 — the other side of the breakpoint — is the untouched desktop page', async ({
    page,
  }) => {
    await page.setViewportSize(AT_BREAKPOINT);
    await openRunning(page, finishedFilmsFrames(3));

    await expect(page.getByTestId('workspace-back')).toBeHidden();
    await expect(page.getByTestId('workspace-log-toggle')).toBeHidden();
    await expect(page.getByTestId('workspace-log-hint')).toBeHidden();
    const log = page.getByTestId('workspace-event-log');
    // Never opened — and still visible: the collapse acts through max-sm: only.
    await expect(log.getByRole('list', { name: '生成事件日誌' })).toBeVisible();
    await expect(log.getByText('僅狀態事件，不含逐字內容')).toBeVisible();

    const root = await box(page.getByTestId('generation-workspace'));
    const overall = await box(page.getByTestId('workspace-overall'));
    expect(Math.round(overall.x - root.x)).toBe(32);
    expect(Math.round(root.width - overall.width)).toBe(64);
    const bar = await box(page.getByRole('progressbar', { name: '整批生成進度' }));
    expect(Math.round(bar.width)).toBe(180);

    const h1 = page.getByRole('heading', { level: 1 });
    expect(await h1.evaluate((el) => window.getComputedStyle(el).fontSize)).toBe('20px');
    const crumb = await box(page.getByRole('navigation', { name: '麵包屑' }));
    const h1B = await box(h1);
    expect(bottom(crumb)).toBeLessThan(h1B.y + 1);
    // The pill hugs the title on desktop (gap-2), it is not pushed to the far edge.
    const pill = await box(page.getByTestId('workspace-status-pill'));
    expect(Math.round(pill.x - right(h1B))).toBe(8);
  });
});
