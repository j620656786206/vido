import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { SpendHistoryView, type SpendHistoryViewProps } from './SpendHistoryPage';
import { daysInMonth } from './SpendDailyChart';
import type {
  RunHistoryDay,
  RunHistoryItem,
  RunHistoryPage,
} from '../../services/subtitleRunHistoryService';

function day(date: string, over: Partial<RunHistoryDay['byCategory']> = {}): RunHistoryDay {
  const byCategory = { chinese_track: 0, translate_english: 0, asr: 0, legacy: 0, ...over };
  const spent = Object.values(byCategory).reduce((a, b) => Math.round((a + b) * 100) / 100, 0);
  return { date, spentUsd: spent, runs: spent > 0 ? 1 : 0, byCategory };
}

function run(over: Partial<RunHistoryItem> = {}): RunHistoryItem {
  return {
    id: 'r1',
    startedAt: '2026-10-28T13:33:46Z',
    completedAt: '2026-10-28T13:38:25Z',
    status: 'completed',
    failureReason: '',
    media: { id: 'm1', type: 'episode', title: 'Lioness', season: 1, episode: 1 },
    route: 'asr',
    source: 'audio',
    category: 'asr',
    spentUsd: 0.52,
    estimateUsd: 0.5,
    translationModel: 'claude-sonnet-5',
    asrProvider: 'elevenlabs',
    asrModel: 'scribe_v2',
    batchId: null,
    triggeredBy: 'manual',
    ...over,
  };
}

function page(over: Partial<RunHistoryPage> = {}): RunHistoryPage {
  const items = over.items ?? [run()];
  return {
    month: '2026-10',
    summary: {
      spentUsd: 3.91,
      runs: 12,
      unpricedRuns: 0,
      estimateUsd: 3.27,
      estimatedRuns: 10,
      comparableRuns: 5,
      comparableSpentUsd: 1.2,
      comparableEstimateUsd: 1.35,
    },
    series: [day('2026-10-01'), day('2026-10-02', { translate_english: 0.81, asr: 3.1 })],
    items,
    page: 1,
    pageSize: 50,
    total: items.length,
    ...over,
  };
}

function props(over: Partial<SpendHistoryViewProps> = {}): SpendHistoryViewProps {
  const data = over.data === undefined && !('data' in over) ? page() : over.data;
  return {
    month: '2026-10',
    currentMonth: '2026-11',
    onMonthChange: vi.fn(),
    data,
    items: data?.items ?? [],
    isLoading: false,
    isError: false,
    onRetry: vi.fn(),
    category: undefined,
    onCategoryChange: vi.fn(),
    hasMore: false,
    isLoadingMore: false,
    onLoadMore: vi.fn(),
    ...over,
  };
}

/** The view renders Links, so it needs a router around it. */
function renderView(p: SpendHistoryViewProps) {
  const rootRoute = createRootRoute();
  const spend = createRoute({
    getParentRoute: () => rootRoute,
    path: '/activity/spend',
    component: () => <SpendHistoryView {...p} />,
  });
  const activity = createRoute({
    getParentRoute: () => rootRoute,
    path: '/activity',
    component: () => <div data-testid="activity-page-stub" />,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([spend, activity]),
    history: createMemoryHistory({ initialEntries: ['/activity/spend'] }),
  });
  return render(React.createElement(RouterProvider, { router } as never));
}

const table = () => screen.findByTestId('spend-history-table');

describe('SpendHistoryView — header and month switch', () => {
  it('[P1] shows the month and steps back a month with ‹', async () => {
    const p = props();
    renderView(p);
    expect(await screen.findByTestId('spend-history-month')).toHaveTextContent('2026 年 10 月');
    fireEvent.click(screen.getByTestId('spend-history-prev-month'));
    expect(p.onMonthChange).toHaveBeenCalledWith('2026-09');
  });

  it('[P1] › steps forward from a past month, across the year', async () => {
    const p = props({ month: '2025-12', currentMonth: '2026-11' });
    renderView(p);
    fireEvent.click(await screen.findByTestId('spend-history-next-month'));
    expect(p.onMonthChange).toHaveBeenCalledWith('2026-01');
  });

  it('[P0] cannot switch into the future: › is disabled on this month', async () => {
    const p = props({ month: '2026-11', currentMonth: '2026-11' });
    renderView(p);
    const next = await screen.findByTestId('spend-history-next-month');
    expect(next).toBeDisabled();
    fireEvent.click(next);
    expect(p.onMonthChange).not.toHaveBeenCalled();
  });

  it('[P2] ← 活動 goes back to the hub', async () => {
    renderView(props());
    fireEvent.click(await screen.findByTestId('spend-history-back'));
    expect(await screen.findByTestId('activity-page-stub')).toBeInTheDocument();
  });
});

describe('SpendHistoryView — summary', () => {
  it("[P1] the month's actual spend and its estimate side by side — no 「準度」 ratio", async () => {
    renderView(props());
    expect(await screen.findByTestId('spend-history-total')).toHaveTextContent(
      '這個月實際花費$3.91 · 12 次'
    );
    expect(screen.getByTestId('spend-history-estimate')).toHaveTextContent('這個月預估$3.27');
    expect(screen.getByTestId('spend-history-summary')).not.toHaveTextContent('準度');
    expect(screen.queryByTestId('spend-history-accuracy')).toBeNull();
    const note = screen.getByTestId('spend-history-note');
    expect(note).toHaveTextContent('依開始時間歸到這個月');
    expect(note).toHaveTextContent('失敗和中途停止的也算進去');
    expect(note).toHaveTextContent('另有 2 次沒有預估，沒算進預估');
  });

  it('[P1] no estimate recorded → 「沒有預估紀錄」, never $0.00', async () => {
    const data = page({
      summary: {
        spentUsd: 1,
        runs: 2,
        unpricedRuns: 1,
        estimateUsd: null,
        estimatedRuns: 0,
        comparableRuns: 0,
        comparableSpentUsd: null,
        comparableEstimateUsd: null,
      },
    });
    renderView(props({ data, items: data.items }));
    const est = await screen.findByTestId('spend-history-estimate');
    expect(est).toHaveTextContent('沒有預估紀錄');
    expect(est).not.toHaveTextContent('$');
    const note = screen.getByTestId('spend-history-note');
    expect(note).toHaveTextContent('另有 1 次沒記錄金額，沒算進實際花費');
    expect(note).not.toHaveTextContent('沒有預估，沒算進預估');
  });
});

describe('SpendHistoryView — list honesty rules', () => {
  it('[P0] an unrecorded amount reads 「沒記錄」, never $0.00', async () => {
    const data = page({ items: [run({ spentUsd: null, estimateUsd: null })] });
    renderView(props({ data, items: data.items }));
    const row = within(await table()).getByTestId('spend-history-row');
    expect(within(row).getByTestId('spend-run-actual-unrecorded')).toHaveTextContent('沒記錄');
    expect(within(row).getByTestId('spend-run-estimate-unrecorded')).toHaveTextContent('沒記錄');
    expect(row).not.toHaveTextContent('$0.00');
  });

  it('[P1] a real $0 (直接用片內繁中) still reads $0.00', async () => {
    const data = page({
      items: [
        run({
          route: 'deliver_direct',
          source: 'embedded_text',
          category: 'chinese_track',
          spentUsd: 0,
          estimateUsd: 0,
          translationModel: null,
          asrProvider: null,
          asrModel: null,
        }),
      ],
    });
    renderView(props({ data, items: data.items }));
    const row = within(await table()).getByTestId('spend-history-row');
    expect(row).toHaveTextContent('直接用片內繁中');
    expect(row).toHaveTextContent('$0.00');
    // No model recorded → 「—」.
    expect(row).toHaveTextContent('—');
  });

  it('[P0] actual more than 50% over the estimate wears ⚠ 比預估高 N%', async () => {
    const data = page({
      items: [
        run({ id: 'over', spentUsd: 0.8, estimateUsd: 0.5 }),
        run({ id: 'edge', spentUsd: 0.75, estimateUsd: 0.5 }),
      ],
    });
    renderView(props({ data, items: data.items }));
    const rows = within(await table()).getAllByTestId('spend-history-row');
    expect(within(rows[0]).getByTestId('spend-run-over-estimate')).toHaveTextContent(
      '⚠ 比預估高 60%'
    );
    // Exactly 1.5× is not MORE than 50% over.
    expect(within(rows[1]).queryByTestId('spend-run-over-estimate')).toBeNull();
  });

  it('[P0] failed and cancelled runs are listed with what they cost', async () => {
    const data = page({
      items: [
        run({ id: 'f', status: 'failed', failureReason: 'unauthorized', spentUsd: 0.12 }),
        run({ id: 'c', status: 'failed', failureReason: 'cancelled', spentUsd: 0.3 }),
      ],
    });
    renderView(props({ data, items: data.items }));
    const rows = within(await table()).getAllByTestId('spend-history-row');
    expect(within(rows[0]).getByTestId('spend-run-status')).toHaveTextContent('失敗');
    expect(within(rows[0]).getByTestId('spend-run-status')).toHaveTextContent('API 金鑰無效');
    expect(rows[0]).toHaveTextContent('$0.12');
    expect(within(rows[1]).getByTestId('spend-run-status')).toHaveTextContent('中途停止');
    expect(rows[1]).toHaveTextContent('$0.30');
  });

  it('[P1] models: translation model, then the ASR engine and model; 語音辨識 wording', async () => {
    renderView(props());
    const row = within(await table()).getByTestId('spend-history-row');
    expect(row).toHaveTextContent('claude-sonnet-5');
    expect(row).toHaveTextContent('ElevenLabs scribe_v2');
    expect(row).toHaveTextContent('語音辨識＋翻譯');
    expect(row).not.toHaveTextContent('聽寫');
    expect(row).toHaveTextContent('Lioness S1E1');
  });

  it('[P2] a deleted video reads 「影片已刪除」', async () => {
    const data = page({
      items: [run({ media: { id: 'x', type: 'movie', title: null, season: null, episode: null } })],
    });
    renderView(props({ data, items: data.items }));
    expect(within(await table()).getByTestId('spend-history-row')).toHaveTextContent('影片已刪除');
  });
});

describe('SpendHistoryView — phone: the same table, four columns', () => {
  it('[P1] time, 處理方式 and models are desktop-only columns; on a phone they sit under the title', async () => {
    const data = page({
      items: [run({ status: 'failed', failureReason: 'cancelled', spentUsd: 0.8 })],
    });
    renderView(props({ data, items: data.items }));
    const t = await table();
    // One table at every width — no phone-only card list any more.
    expect(screen.queryByTestId('spend-history-cards')).toBeNull();
    expect(t.parentElement).not.toHaveClass('hidden');
    const heads = Array.from(t.querySelectorAll('th'));
    const phone = heads
      .filter((th) => !th.classList.contains('hidden'))
      .map((th) => th.textContent);
    expect(phone).toEqual(['影片', '實際', '預估', '狀態']);
    for (const th of heads.filter((h) => h.classList.contains('hidden'))) {
      expect(th).toHaveClass('md:table-cell');
    }

    const row = within(t).getByTestId('spend-history-row');
    const compact = within(row).getByTestId('spend-run-compact');
    expect(compact).toHaveClass('md:hidden');
    expect(compact).toHaveTextContent('語音辨識＋翻譯');
    expect(compact).toHaveTextContent('claude-sonnet-5 ＋ ElevenLabs scribe_v2');
    expect(within(compact).getByTestId('spend-run-duration')).toBeInTheDocument();
    expect(row).toHaveTextContent('⚠ 比預估高 60%');
    expect(row).toHaveTextContent('$0.50');
    expect(within(row).getByTestId('spend-run-status')).toHaveTextContent('中途停止');
  });
});

describe('SpendHistoryView — how long each run took', () => {
  it('[P1] a 花了多久 column on the desktop and in the phone line; no end time → 沒記錄', async () => {
    const data = page({
      items: [
        run({ id: 'done', startedAt: '2026-10-28T05:33:00Z', completedAt: '2026-10-28T05:41:03Z' }),
        run({ id: 'open', startedAt: '2026-10-28T06:00:00Z', completedAt: null }),
      ],
    });
    renderView(props({ data, items: data.items }));
    const t = await table();
    const head = Array.from(t.querySelectorAll('th')).find((th) => th.textContent === '花了多久');
    expect(head).toHaveClass('hidden', 'md:table-cell');
    const [done, open] = within(t).getAllByTestId('spend-history-row');
    expect(within(done).getAllByTestId('spend-run-duration')[0]).toHaveTextContent('8 分 3 秒');
    expect(within(open).getAllByTestId('spend-run-duration-unrecorded')[0]).toHaveTextContent(
      '沒記錄'
    );
    expect(open).not.toHaveTextContent('0 秒');
  });
});

describe('SpendHistoryView — category chips and paging', () => {
  it('[P1] a chip filters by category; 全部 clears it', async () => {
    const p = props();
    renderView(p);
    fireEvent.click(await screen.findByTestId('spend-history-chip-asr'));
    expect(p.onCategoryChange).toHaveBeenCalledWith('asr');
    fireEvent.click(screen.getByTestId('spend-history-chip-all'));
    expect(p.onCategoryChange).toHaveBeenLastCalledWith(undefined);
    expect(screen.getByTestId('spend-history-chip-all')).toHaveAttribute('aria-pressed', 'true');
  });

  it('[P1] the selected chip is pressed; a filter with no runs keeps summary and chart', async () => {
    const data = page({ items: [], total: 0 });
    renderView(props({ data, items: [], category: 'legacy' }));
    expect(await screen.findByTestId('spend-history-chip-legacy')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    expect(screen.getByTestId('spend-history-filter-empty')).toBeInTheDocument();
    expect(screen.getByTestId('spend-history-summary')).toBeInTheDocument();
    expect(screen.getByTestId('spend-history-chart')).toBeInTheDocument();
    expect(screen.queryByTestId('spend-history-empty')).toBeNull();
  });

  it('[P1] 載入更多 asks for the next page', async () => {
    const p = props({ hasMore: true });
    renderView(p);
    fireEvent.click(await screen.findByTestId('spend-history-load-more'));
    expect(p.onLoadMore).toHaveBeenCalled();
  });
});

describe('SpendHistoryView — states', () => {
  it('[P1] loading: a skeleton per block', async () => {
    renderView(props({ data: undefined, items: [], isLoading: true }));
    expect(await screen.findByTestId('spend-history-summary-skeleton')).toBeInTheDocument();
    expect(screen.getByTestId('spend-history-chart-skeleton')).toBeInTheDocument();
    expect(screen.getByTestId('activity-skeleton')).toBeInTheDocument();
  });

  it('[P0] an untouched month: 「這個月還沒有生成過字幕」 with a way back; the switcher stays', async () => {
    const data = page({
      items: [],
      total: 0,
      summary: {
        spentUsd: 0,
        runs: 0,
        unpricedRuns: 0,
        comparableRuns: 0,
        comparableSpentUsd: null,
        comparableEstimateUsd: null,
      },
    });
    renderView(props({ data, items: [] }));
    expect(await screen.findByTestId('spend-history-empty')).toHaveTextContent(
      '這個月還沒有生成過字幕'
    );
    expect(screen.getByTestId('spend-history-empty-back')).toHaveAttribute('href', '/activity');
    expect(screen.getByTestId('spend-history-prev-month')).toBeInTheDocument();
    expect(screen.queryByTestId('spend-history-summary')).toBeNull();
  });

  it('[P0] error: 無法載入 + 重試 retries', async () => {
    const p = props({ data: undefined, items: [], isError: true });
    renderView(p);
    const err = await screen.findByTestId('spend-history-error');
    expect(err).toHaveTextContent('無法載入');
    fireEvent.click(within(err).getByTestId('activity-section-retry'));
    expect(p.onRetry).toHaveBeenCalled();
  });
});

describe('SpendDailyChart (inside the view)', () => {
  it('[P1] one column per day, a $0 day gets a baseline stub, legend has the four categories', async () => {
    renderView(props());
    const chart = await screen.findByTestId('spend-history-chart');
    expect(within(chart).getAllByTestId('spend-history-chart-day')).toHaveLength(2);
    expect(within(chart).getAllByTestId('spend-history-chart-zero')).toHaveLength(1);
    const legend = within(chart).getByTestId('spend-history-chart-legend');
    for (const word of ['片內中文字幕', '翻譯英文字幕', '語音辨識＋翻譯', '舊紀錄']) {
      expect(legend).toHaveTextContent(word);
    }
  });

  it('[P1] a slot for every day of the month — days still to come stay empty (disc-2026-11-spend-chart-current-month-one-bar)', async () => {
    // October has 31 days; the series (through "today") has only the 1st and 2nd.
    renderView(props());
    const chart = await screen.findByTestId('spend-history-chart');
    const axis = within(chart).getByTestId('spend-history-chart-axis');
    expect(axis.children).toHaveLength(31);
    expect(axis.style.gridTemplateColumns).toBe('repeat(31, minmax(0, 1fr))');
    expect(chart.querySelector('svg')).toHaveAttribute('viewBox', '0 0 310 100');
    // Only the two real days are drawn and hoverable.
    expect(within(chart).getAllByTestId('spend-history-chart-hit')).toHaveLength(2);
    const second = within(chart).getAllByTestId('spend-history-chart-day')[1];
    expect(second).toHaveAttribute('data-date', '2026-10-02');
    expect(second.querySelector('[data-category]')).toHaveAttribute('x', '12');
  });

  it('[P2] daysInMonth knows month lengths and refuses junk', () => {
    expect(daysInMonth('2026-11')).toBe(30);
    expect(daysInMonth('2026-02')).toBe(28);
    expect(daysInMonth('2028-02')).toBe(29);
    expect(daysInMonth('2026-12')).toBe(31);
    expect(daysInMonth('nope')).toBe(0);
  });

  it('[P1] hover shows the day total and each category; a tap pins it', async () => {
    renderView(props());
    const chart = await screen.findByTestId('spend-history-chart');
    const hits = within(chart).getAllByTestId('spend-history-chart-hit');
    const readout = within(chart).getByTestId('spend-history-chart-readout');

    fireEvent.mouseEnter(hits[1]);
    expect(readout).toHaveTextContent('10 月 2 日');
    expect(readout).toHaveTextContent('$3.91');
    expect(readout).toHaveTextContent('翻譯英文字幕$0.81');
    expect(readout).toHaveTextContent('語音辨識＋翻譯$3.10');
    expect(readout).not.toHaveTextContent('舊紀錄');

    // Leaving without a tap clears it; a tap (phone) keeps it.
    fireEvent.mouseLeave(chart.querySelector('svg') as Element);
    expect(readout).not.toHaveTextContent('10 月 2 日');
    fireEvent.click(hits[0]);
    expect(readout).toHaveTextContent('10 月 1 日');
    expect(readout).toHaveTextContent('這天沒有花費');
  });
});
