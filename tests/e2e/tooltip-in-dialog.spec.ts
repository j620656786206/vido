/**
 * A tooltip opened from INSIDE a dialog must sit on top of that dialog
 * (disc-2026-10-tooltip-hidden-in-dialog).
 *
 * jsdom has no stacking contexts, and Playwright's toBeVisible() passes for an
 * element another layer covers — so both missed it when the tooltip's z-index
 * sat on the popup instead of its fixed positioner and every dialog (z-50)
 * buried it. This spec asks the browser which element is actually on top at
 * the tooltip's centre.
 *
 * Two dialogs, because they layer differently: 儲存篩選 on /discover
 * (SavePresetDialog, a hand-built z-50 overlay) and 產生字幕 (the shared
 * ui/Dialog — Radix, also z-50 — whose ✕ and reason badges #770/#771 added
 * tooltips to). Their backends are stubbed.
 *
 * @tags @e2e @tooltip @regression
 */

import type { Locator } from '@playwright/test';
import { test, expect, type Route } from '../support/fixtures';
import {
  stubPopulatedLibrary,
  openGenerationDialog,
} from '../support/helpers/batch-subtitle-stubs';

const ROUTE_API = '**/api/v1';

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

const emptyPage = { page: 1, results: [], total_pages: 1, total_results: 0 };

/** The open tooltip popup with this text — not the trigger's sr-only label and
 * not its positioner (Base UI marks both with data-side; the positioner is
 * role=presentation). */
const tooltipWith = (page: import('@playwright/test').Page, text: string) =>
  page.locator('[data-side]:not([role="presentation"])').filter({ hasText: text });

/** Is the tooltip what the browser paints on top at its own centre?
 * A modal Radix dialog sets pointer-events:none on <body>, and the tooltip,
 * portalled into body, inherits it — elementFromPoint skips such elements. So
 * hit-testing is switched back on for the tooltip and its positioner for the
 * one probe: that asks about PAINT order, which is the bug. */
async function isOnTop(tip: Locator) {
  return tip.evaluate((el: HTMLElement) => {
    const layers = [el, el.parentElement as HTMLElement];
    const before = layers.map((l) => l.style.pointerEvents);
    layers.forEach((l) => (l.style.pointerEvents = 'auto'));
    const r = el.getBoundingClientRect();
    const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
    layers.forEach((l, i) => (l.style.pointerEvents = before[i]));
    return hit !== null && (el === hit || el.contains(hit));
  });
}

test.describe('對話框裡的提示框 @e2e @tooltip', () => {
  test('[P1] the ✕ tooltip in a dialog is on top, not under the dialog', async ({
    page,
    isMobile,
  }) => {
    test.skip(isMobile, 'hover is a desktop interaction');
    await page.route(`${ROUTE_API}/tmdb/discover/*`, (route: Route) =>
      route.fulfill(jsonOk(emptyPage))
    );
    await page.route(`${ROUTE_API}/filter-presets`, (route: Route) =>
      route.fulfill(jsonOk({ presets: [] }))
    );

    await page.goto('/discover?genre=16');
    await page.getByTestId('save-preset-button').click();
    const dialog = page.getByTestId('save-preset-dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByRole('button', { name: '關閉視窗' }).hover();
    const tip = tooltipWith(page, '關閉視窗');
    await expect(tip).toBeVisible();
    expect(await isOnTop(tip), 'the tooltip is painted above the dialog').toBe(true);
  });

  test('[P1] the ✕ tooltip in the shared (Radix) dialog is on top', async ({ page, isMobile }) => {
    test.skip(isMobile, 'hover is a desktop interaction');
    await stubPopulatedLibrary(page);
    await openGenerationDialog(page);
    const dialog = page.getByTestId('generation-consent-view');

    await dialog.getByRole('button', { name: '關閉視窗' }).hover();
    const tip = tooltipWith(page, '關閉視窗');
    await expect(tip).toBeVisible();
    expect(await isOnTop(tip), 'the tooltip is painted above the dialog').toBe(true);
  });
});
