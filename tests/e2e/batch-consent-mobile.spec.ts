/**
 * 產生字幕 on a phone — the three sheets (挑片 F15-M → 確認 F16-M → 批次 F8-M),
 * measured in a REAL browser (dsr-6f-3).
 *
 * jsdom evaluates no media queries, so the unit specs can only pin class tokens.
 * And the consent view cannot be a visual fixture at all: its phase and its
 * candidates are internal state fed by direct service calls. So this spec is the
 * only layout evidence the 挑片 sheet has, and it measures BOTH sides of the
 * 640px breakpoint (feedback_measure_the_breakpoint_you_return_to).
 *
 * What it deliberately measures, because a class string cannot:
 *   · the ✕ is vertically centred on the 44px title row — that is what decides
 *     whether `top-4` (was `top-[22px]` on the old 56px row) is right;
 *   · 本次用量 sits BELOW the scrolling body, i.e. in the fixed footer;
 *   · F16-M's footer pair stays right-aligned and NOT full-width (the design's
 *     own `justifyContent: end` — easy to "fix" by mistake).
 *
 * Same fake backend as tests/e2e/batch-subtitle.spec.ts (shared stubs). The SSE
 * stream is aborted there; the started batch's first item is `running`, so the
 * six-step stepper renders without it.
 *
 * @tags @e2e @batch-consent-mobile
 */

import type { Locator, Page } from '@playwright/test';
import { test, expect, type Route } from '../support/fixtures';
import {
  ROUTE_API,
  jsonStatus,
  startedBatch,
  stubPopulatedLibrary,
  openGenerationDialog,
} from '../support/helpers/batch-subtitle-stubs';

const PHONE = { width: 390, height: 844 };
const AT_BREAKPOINT = { width: 640, height: 844 };
const GUTTER = 16;

/** e2e runs with motion ON — measuring mid-slide reads the sheet below the
 * screen. allSettled + explicit undefined: a cancelled animation must not
 * reject, and Animation objects are not serialisable back to Node. */
const settle = (sheet: Locator) =>
  sheet.evaluate((el) =>
    Promise.allSettled(el.getAnimations().map((a) => a.finished)).then(() => undefined)
  );

const animationName = (sheet: Locator) =>
  sheet.evaluate((el) => window.getComputedStyle(el).animationName);

const centreY = (box: { y: number; height: number }) => box.y + box.height / 2;

async function expectFlushSheet(sheet: Locator) {
  const box = (await sheet.boundingBox())!;
  // Math.round: a computed -0 is not Object.is-equal to 0.
  expect(Math.round(box.x)).toBe(0);
  expect(Math.round(box.width)).toBe(PHONE.width);
  expect(Math.round(box.y + box.height)).toBe(PHONE.height);
}

/** 44px title row with the 44×44 ✕ centred on it. */
async function expectTitleBar(sheet: Locator, bar: Locator) {
  const barBox = (await bar.boundingBox())!;
  expect(Math.round(barBox.height)).toBe(44);
  const closeBox = (await sheet.getByRole('button', { name: 'Close' }).boundingBox())!;
  expect(closeBox.width).toBeGreaterThanOrEqual(44);
  expect(closeBox.height).toBeGreaterThanOrEqual(44);
  expect(Math.abs(centreY(closeBox) - centreY(barBox))).toBeLessThan(2);
}

async function stubStart(page: Page) {
  await page.route(`${ROUTE_API}/subtitles/generation-batch`, (route: Route) =>
    route.fulfill(jsonStatus(202, startedBatch))
  );
}

test.describe('產生字幕 — phone sheets @e2e @batch-consent-mobile', () => {
  // Exact-pixel geometry at a viewport this spec sets itself: the device
  // projects (Pixel 5 / iPhone 13 — other scale factors, isMobile) would measure
  // something else. CI runs it on chromium; keep a local `test:e2e` honest too.
  test.beforeEach(async ({ page }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'chromium',
      'pixel-exact geometry — desktop chromium project only'
    );
    await stubPopulatedLibrary(page);
    await stubStart(page);
  });

  test('[P0] 390px 挑片 (F15-M): flush sheet, 44px title row, budget box fills the row', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openGenerationDialog(page);

    const sheet = page.getByTestId('generation-consent-view');
    await settle(sheet);
    expect(await animationName(sheet)).toBe('sheet-enter');
    await expectFlushSheet(sheet);
    await expect(page.getByTestId('consent-sheet-grabber')).toBeVisible();
    await expectTitleBar(sheet, page.getByTestId('consent-title-bar'));

    // The framed budget box — not the <input> inside it — runs to the gutter.
    const budget = (await page.getByTestId('consent-budget-box').boundingBox())!;
    expect(Math.round(budget.x + budget.width)).toBe(PHONE.width - GUTTER);
    expect(budget.width).toBeGreaterThan(200);

    // 開始產生 already stretched across the footer before this story (guard); what
    // is new is the footer's 16px gutter, which makes that width 358, not 342.
    const start = (await page.getByTestId('consent-start-btn').boundingBox())!;
    expect(Math.round(start.width)).toBe(PHONE.width - GUTTER * 2);

    // The list takes the 16px gutter, not the desktop 24.
    const list = (await page.getByTestId('consent-candidate-list').boundingBox())!;
    expect(Math.round(list.x)).toBe(GUTTER);
  });

  test('[P0] 390px 確認 (F16-M): 44px title row; the footer pair stays right-aligned, NOT full-width', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openGenerationDialog(page);
    await page.getByTestId('consent-start-btn').click();

    const sheet = page.getByTestId('consent-confirm-dialog');
    await expect(sheet).toBeVisible();
    await settle(sheet);
    await expectFlushSheet(sheet);
    await expectTitleBar(sheet, page.getByTestId('consent-confirm-title-bar'));

    const cancel = (await page.getByTestId('consent-confirm-cancel').boundingBox())!;
    const confirm = (await page.getByTestId('consent-confirm-start').boundingBox())!;
    expect(Math.abs(centreY(cancel) - centreY(confirm))).toBeLessThan(2);
    expect(Math.round(confirm.x + confirm.width)).toBe(PHONE.width - GUTTER);
    expect(confirm.width).toBeLessThan(200);
    expect(cancel.width).toBeLessThan(200);
    expect(cancel.x).toBeGreaterThan(PHONE.width / 3);
  });

  test('[P0] 390px 批次 (F8-M): 本次用量 lives in the fixed footer, 全部取消 is full-width, stepper is vertical', async ({
    page,
  }) => {
    await page.setViewportSize(PHONE);
    await openGenerationDialog(page);
    await page.getByTestId('consent-start-btn').click();
    await page.getByTestId('consent-confirm-start').click();

    const sheet = page.getByTestId('generation-batch-dialog-v2');
    await expect(page.getByTestId('gen-batch-counter')).toBeVisible();
    await settle(sheet);
    await expectFlushSheet(sheet);
    await expectTitleBar(sheet, page.getByTestId('gen-batch-title-bar'));

    // One copy per breakpoint: the body one is gone, the footer one is on.
    await expect(page.getByTestId('gen-batch-cost-line')).toBeHidden();
    const cost = page.getByTestId('gen-batch-cost-line-mobile');
    await expect(cost).toBeVisible();
    await expect(page.getByTestId('gen-batch-sse-chip-mobile')).toBeVisible();

    // FIXED footer: the cost row starts where the scrolling body ends.
    const body = (await page.getByTestId('gen-batch-body').boundingBox())!;
    const costBox = (await cost.boundingBox())!;
    expect(costBox.y).toBeGreaterThanOrEqual(body.y + body.height - 1);

    const cancelAll = (await page.getByTestId('gen-batch-cancel-all').boundingBox())!;
    expect(Math.round(cancelAll.width)).toBe(PHONE.width - GUTTER * 2);
    expect(cancelAll.y).toBeGreaterThan(costBox.y);

    // ⚖️ 2026-09-21: the six-step stepper is VERTICAL on a phone (same as F3-M).
    const first = (await page.getByTestId('gen-stage-提取音訊').boundingBox())!;
    const second = (await page.getByTestId('gen-stage-轉錄中').boundingBox())!;
    expect(second.y).toBeGreaterThan(first.y + first.height - 1);
    expect(Math.abs(second.x - first.x)).toBeLessThan(2);

    // Cancel-confirm: the sentence takes its own line, the pair splits the row.
    await page.getByTestId('gen-batch-cancel-all').click();
    const keep = (await page.getByRole('button', { name: '繼續生成' }).boundingBox())!;
    const sure = (await page.getByTestId('gen-batch-cancel-confirm-btn').boundingBox())!;
    expect(Math.abs(centreY(keep) - centreY(sure))).toBeLessThan(2);
    expect(Math.abs(keep.width - sure.width)).toBeLessThan(2);
    expect(Math.round(sure.x + sure.width)).toBe(PHONE.width - GUTTER);
  });

  test('[P0] 640px — the other side of the breakpoint — is the untouched desktop dialog', async ({
    page,
  }) => {
    await page.setViewportSize(AT_BREAKPOINT);
    await openGenerationDialog(page);

    const consent = page.getByTestId('generation-consent-view');
    await settle(consent);
    expect(await animationName(consent)).toBe('dialog-enter');
    await expect(page.getByTestId('consent-sheet-grabber')).toBeHidden();
    const bar = (await page.getByTestId('consent-title-bar').boundingBox())!;
    expect(Math.round(bar.height)).toBe(56);
    // Desktop: the fixed 96px budget box.
    const budget = (await page.getByTestId('consent-budget-box').boundingBox())!;
    expect(Math.round(budget.width)).toBe(96);

    await page.getByTestId('consent-start-btn').click();
    await page.getByTestId('consent-confirm-start').click();
    await expect(page.getByTestId('gen-batch-counter')).toBeVisible();
    const batch = page.getByTestId('generation-batch-dialog-v2');
    await settle(batch);

    // The batch dialog's own title row is the 56px desktop one too, with its ✕
    // back to the small desktop target.
    const batchBar = (await page.getByTestId('gen-batch-title-bar').boundingBox())!;
    expect(Math.round(batchBar.height)).toBe(56);
    await expect(page.getByTestId('gen-batch-drag-handle')).toBeHidden();
    const batchClose = (await batch.getByRole('button', { name: 'Close' }).boundingBox())!;
    expect(batchClose.width).toBeLessThan(44);

    await expect(page.getByTestId('gen-batch-cost-line')).toBeVisible();
    await expect(page.getByTestId('gen-batch-cost-line-mobile')).toBeHidden();
    const box = (await batch.boundingBox())!;
    const cancelAll = (await page.getByTestId('gen-batch-cancel-all').boundingBox())!;
    expect(cancelAll.width).toBeLessThan(box.width / 2);
    // …and the stepper is horizontal again.
    const first = (await page.getByTestId('gen-stage-提取音訊').boundingBox())!;
    const second = (await page.getByTestId('gen-stage-轉錄中').boundingBox())!;
    expect(second.x).toBeGreaterThan(first.x + first.width - 1);
  });
});
