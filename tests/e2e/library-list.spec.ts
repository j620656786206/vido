/**
 * The library list as a 修字幕清單 (disc-2026-10-library-list-triage-b1 — A4p-D `b1H71g`,
 * A4p-M `EkB8g`).
 *
 * @tags @e2e @library-list
 *
 * The list picks its layout from ITS OWN width (`@container`), not the window: the
 * sidebar (240) and the filter rail (264 + 24 gap) take a different bite at every window
 * size. jsdom evaluates no container queries, so this spec is what proves the three
 * tiers — and it measures the real list width at each window it claims, both sides of
 * each threshold (feedback_measure_the_breakpoint_you_return_to):
 *   390 phone → 358 (three-line) · 768 → 480 · 1024 rail open → 448 (three-line)
 *   1200 rail open → 624 (table, no 規格) · 1280 rail open → 704 (full table)
 *   1440 rail collapsed → 1152 (full table, the A4p-D frame)
 */
import type { Locator, Page, TestInfo } from '@playwright/test';
import { test, expect } from '../support/fixtures';
import { movieItem, stubLibraryBaseline, stubLibraryList } from '../support/helpers/library-stubs';

const FAILED_FILE = '[ItachiUchiha] Chainsaw Man – The Movie – Reze-hen.mkv';
const UHD = { video_resolution: '3840x2160', video_codec: 'H.265' };

const ITEMS = [
  movieItem('r1', '執法記錄', 'none', { ...UHD, audio_codec: 'DTS', vote_average: 6 }),
  movieItem('r2', '鏢人：風起大漠', 'unknown', { vote_average: 8.1 }),
  movieItem('r3', '[ItachiUchiha] Chainsaw Man', 'unknown', {
    parse_status: 'failed',
    file_path: `/media/movies/${FAILED_FILE}`,
    vote_average: 0,
  }),
  movieItem('r4', '牧師的最後誘惑', 'zh', { ...UHD, audio_codec: 'DTS', vote_average: 7 }),
  movieItem('r5', '阿凡達：火與燼：一個很長很長很長很長很長很長很長很長很長很長的片名', 'zh_hant', {
    ...UHD,
    audio_codec: 'E-AC-3',
    vote_average: 7.6,
  }),
  movieItem('r6', 'F1電影', 'zh_hant', {
    video_resolution: '1920x800',
    video_codec: 'H.265',
    audio_codec: 'AAC',
    vote_average: 7.8,
  }),
];
const IDS = ITEMS.map((i) => i.movie.id);

type Box = { x: number; y: number; width: number; height: number };
const box = async (l: Locator): Promise<Box> => {
  const b = await l.boundingBox();
  if (!b) throw new Error('no bounding box');
  return b;
};
const right = (b: Box) => b.x + b.width;

async function openList(
  page: Page,
  viewport: { width: number; height: number },
  opts: { railCollapsed?: boolean } = {}
) {
  await page.setViewportSize(viewport);
  await page.addInitScript((collapsed) => {
    localStorage.setItem('vido:library:view', 'list');
    localStorage.setItem('vido:library:rail-collapsed', collapsed ? '1' : '0');
  }, !!opts.railCollapsed);
  await stubLibraryBaseline(page);
  await stubLibraryList(page, () => ({ items: ITEMS }));
  await page.goto('/library');
  const frame = page.getByTestId('library-list-v2');
  await expect(frame).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('list-row-v2-r6')).toBeVisible();
  return frame;
}

/** Step-9 evidence, not baselines: only with `DSR_SHOTS=1`, into the test's own output dir. */
async function shot(page: Page, testInfo: TestInfo, name: string) {
  if (!process.env.DSR_SHOTS) return;
  await page.screenshot({ path: testInfo.outputPath(name) });
}

/**
 * The list's width is the page's client width minus what sits beside it: the 16px phone
 * gutters (32), or from sm the 240 sidebar + 24px gutters (288), plus the 264 rail and its
 * 24 gap (288) when the rail is open. Read off clientWidth, not the viewport, so a classic
 * scrollbar (Linux CI) does not move the goalposts — the tier margins below absorb 15px.
 */
async function expectListWidth(page: Page, frame: Locator, beside: number, nominal: number) {
  const cw = await page.evaluate(() => document.documentElement.clientWidth);
  const width = (await box(frame)).width;
  expect(width).toBe(cw - beside);
  // `nominal` is the width with an overlay scrollbar — the number the story's tiers use.
  expect(nominal - width).toBeLessThanOrEqual(16);
}

async function expectNoSideScroll(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth
  );
  expect(overflow).toBe(0);
}

/** Every row's 字幕 cell starts at the same x, and that x is the header's 字幕 x. */
async function expectSubtitleColumnAligned(page: Page) {
  const xs = await Promise.all(
    IDS.map(async (id) => (await box(page.getByTestId(`list-row-subtitle-${id}`))).x)
  );
  for (const x of xs) expect(x).toBeCloseTo(xs[0], 0);
  const head = await box(page.getByTestId('library-list-header').getByText('字幕'));
  expect(head.x).toBeCloseTo(xs[0], 0);
  return xs[0];
}

async function expectThreeLineRows(page: Page) {
  await expect(page.getByTestId('library-list-header')).toBeHidden();
  for (const id of IDS) {
    await expect(page.getByTestId(`list-row-subtitle-${id}`)).toBeVisible();
    await expect(page.getByTestId(`list-row-spec-${id}`)).toBeHidden();
    // Third line: the subtitle state sits under the title, not beside it.
    const title = await box(
      page.getByTestId(`list-row-v2-${id}`).locator('span.font-semibold').first()
    );
    const sub = await box(page.getByTestId(`list-row-subtitle-${id}`));
    expect(sub.y).toBeGreaterThan(title.y + title.height - 1);
  }
}

test.describe('Library list — 修字幕清單 @e2e @library-list', () => {
  test('390 phone: three-line rows in a 358 frame, every row shows its subtitle state (AC #3)', async ({
    page,
  }, testInfo) => {
    const frame = await openList(page, { width: 390, height: 844 });
    await shot(page, testInfo, 'list-390.png');
    await expectListWidth(page, frame, 32, 358);
    await expectThreeLineRows(page);
    await expectNoSideScroll(page);

    // Nothing inside a row runs past the frame.
    const f = await box(frame);
    for (const id of IDS) {
      const row = await box(page.getByTestId(`list-row-v2-${id}`));
      expect(right(row)).toBeLessThanOrEqual(right(f) + 0.5);
    }
    // Failed row: B10p's sentence, 失敗 on the third line, the file name keeps its tail.
    const failed = page.getByTestId('list-row-v2-r3');
    await expect(failed.getByText('沒有找到這部電影的資料')).toBeVisible();
    await expect(failed.getByText('失敗').filter({ visible: true })).toHaveCount(1);
    await expect(failed.getByText('Reze-hen.mkv', { exact: true })).toBeVisible();
    const name = await box(failed.getByTestId('list-row-filename'));
    expect(right(name)).toBeLessThanOrEqual(right(f));
  });

  test('768 and 1024 (rail open): the list is under 576 → three-line rows', async ({
    page,
  }, testInfo) => {
    let frame = await openList(page, { width: 768, height: 900 });
    await shot(page, testInfo, 'list-768.png');
    await expectListWidth(page, frame, 288, 480);
    await expectThreeLineRows(page);
    await expectNoSideScroll(page);

    frame = await openList(page, { width: 1024, height: 900 });
    await shot(page, testInfo, 'list-1024-rail.png');
    await expectListWidth(page, frame, 576, 448);
    await expectThreeLineRows(page);
    await expectNoSideScroll(page);
  });

  test('1200 rail open (list 624): the table without 規格, columns aligned (AC #2)', async ({
    page,
  }, testInfo) => {
    const frame = await openList(page, { width: 1200, height: 900 });
    await shot(page, testInfo, 'list-1200-rail.png');
    await expectListWidth(page, frame, 576, 624);
    await expect(page.getByTestId('library-list-header')).toBeVisible();
    await expect(page.getByTestId('library-list-header-spec')).toBeHidden();
    for (const id of IDS) await expect(page.getByTestId(`list-row-spec-${id}`)).toBeHidden();
    await expectSubtitleColumnAligned(page);
    await expectNoSideScroll(page);
  });

  test('1280 rail open (list 704): the full table (AC #1 tier)', async ({ page }, testInfo) => {
    const frame = await openList(page, { width: 1280, height: 900 });
    await shot(page, testInfo, 'list-1280-rail.png');
    await expectListWidth(page, frame, 576, 704);
    await expect(page.getByTestId('library-list-header-spec')).toBeVisible();
    for (const id of IDS) await expect(page.getByTestId(`list-row-spec-${id}`)).toBeVisible();
    const subX = await expectSubtitleColumnAligned(page);
    // 片名 gave way first; 規格 kept its 120.
    const spec = await box(page.getByTestId('library-list-header-spec'));
    expect(spec.width).toBeGreaterThanOrEqual(120);
    const title = await box(page.getByTestId('library-list-header').getByText('片名'));
    expect(subX - title.x).toBeLessThanOrEqual(360 + 16);
    await expectNoSideScroll(page);
  });

  test('1440 rail collapsed: A4p-D — the empty space is only after 規格 (AC #1)', async ({
    page,
  }, testInfo) => {
    const frame = await openList(page, { width: 1440, height: 900 }, { railCollapsed: true });
    await shot(page, testInfo, 'list-1440.png');
    await expectListWidth(page, frame, 288, 1152);
    const header = page.getByTestId('library-list-header');
    const subX = await expectSubtitleColumnAligned(page);
    const title = await box(header.getByText('片名'));
    expect(title.width).toBe(360);
    // 片名 → 字幕 is exactly one 16px gap — no space between the title and its state.
    expect(subX - title.x).toBeLessThanOrEqual(360 + 16);
    expect(subX - right(title)).toBeCloseTo(16, 0);
    // The 規格 column runs right up to the › column (one 16 gap + half of 20−16) and its
    // text is short — so the only slack in the row sits after the spec text.
    const spec = await box(header.getByText('規格'));
    const chevron = await box(
      page.getByTestId('list-row-v2-r1').locator('svg.lucide-chevron-right')
    );
    expect(chevron.x - right(spec)).toBeCloseTo(18, 0);
    const specText = page.getByTestId('list-row-spec-r1');
    await expect(specText).toHaveText('4K · H.265 · DTS');
    const textWidth = await specText.evaluate((el) => {
      const r = document.createRange();
      r.selectNodeContents(el);
      return r.getBoundingClientRect().width;
    });
    expect(textWidth).toBeLessThan((await box(specText)).width - 100);
    // A row with no rating (r3, failed) keeps the 評分 column: its 規格 starts at the same x.
    expect((await box(page.getByTestId('list-row-spec-r3'))).x).toBeCloseTo(
      (await box(specText)).x,
      0
    );
    // A 1920x800 scope encode reads as 1080p.
    await expect(page.getByTestId('list-row-spec-r6')).toHaveText('1080p · H.265 · AAC');
    // A long title truncates inside its 360, it does not push 字幕.
    const long = page.getByTestId('list-row-v2-r5').locator('span.font-semibold').first();
    expect((await box(long)).width).toBeLessThanOrEqual(360);
    await expectNoSideScroll(page);
  });

  test('each threshold, both sides: list 575 / 576 / 671 / 672 (no rail below lg)', async ({
    page,
  }) => {
    const frame = await openList(page, { width: 863, height: 900 });
    const header = page.getByTestId('library-list-header');
    const headSpec = page.getByTestId('library-list-header-spec');
    const seen: number[] = [];
    // list = client width − 240 sidebar − 48 gutters; the layout follows the list, live.
    // Sized off the client width so a classic scrollbar cannot move us off the edges.
    const scrollbar = await page.evaluate(
      () => window.innerWidth - document.documentElement.clientWidth
    );
    for (const list of [575, 576, 671, 672]) {
      await page.setViewportSize({ width: list + 288 + scrollbar, height: 900 });
      const w = (await box(frame)).width;
      seen.push(w);
      if (w < 576) {
        await expect(header).toBeHidden();
        await expect(page.getByTestId('list-row-spec-r1')).toBeHidden();
      } else if (w < 672) {
        await expect(header).toBeVisible();
        await expect(headSpec).toBeHidden();
        await expect(page.getByTestId('list-row-spec-r1')).toBeHidden();
        await expectSubtitleColumnAligned(page);
      } else {
        await expect(headSpec).toBeVisible();
        await expect(page.getByTestId('list-row-spec-r1')).toBeVisible();
        await expectSubtitleColumnAligned(page);
      }
      for (const id of IDS) await expect(page.getByTestId(`list-row-subtitle-${id}`)).toBeVisible();
      await expectNoSideScroll(page);
    }
    expect(seen).toEqual([575, 576, 671, 672]);
  });

  test('keyboard focus ring is drawn inside the clipped frame (AC #9)', async ({ page }) => {
    await openList(page, { width: 1440, height: 900 }, { railCollapsed: true });
    await page.getByTestId('list-row-v2-r1').focus();
    await page.keyboard.press('Tab');
    const second = page.getByTestId('list-row-v2-r2');
    await expect(second).toBeFocused();
    const style = await second.evaluate((el) => {
      const s = window.getComputedStyle(el);
      return { offset: s.outlineOffset, width: s.outlineWidth, style: s.outlineStyle };
    });
    expect(style.offset).toBe('-2px');
    // The rows at the frame's corners round with it, so their ring follows the curve.
    const lastRadius = await page
      .getByTestId('list-row-v2-r6')
      .evaluate((el) => window.getComputedStyle(el).borderBottomLeftRadius);
    expect(parseFloat(lastRadius)).toBeGreaterThan(0);
    expect(style.style).toBe('solid');
    await expect(second).toHaveAttribute(
      'aria-label',
      /^鏢人：風起大漠，.*字幕：不知道，評分 8\.1$/
    );
  });
});
