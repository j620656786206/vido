import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { MineResult, MineStatus } from '../../services/glossaryMineService';

const h = vi.hoisted(() => ({
  status: undefined as MineStatus | undefined,
  start: vi.fn(),
}));

vi.mock('../../services/glossaryMineService', async (orig) => ({
  ...(await orig<typeof import('../../services/glossaryMineService')>()),
  glossaryMineService: {
    status: () => Promise.resolve(h.status),
    startSweep: (...args: unknown[]) => h.start(...args),
  },
}));

import {
  OfficialSubtitleMiningCard,
  describeResult,
  formatRunTime,
  statusLine,
} from './OfficialSubtitleMiningCard';

const SAB: MineResult = {
  seriesId: 's1',
  title: 'Shadow and Bone',
  scope: 'tmdb:tv:75006',
  episodesTotal: 8,
  episodesUsed: 8,
  fansubSkipped: 0,
  termsFound: 68,
  termsInserted: 68,
  startedAt: '2026-10-01T06:29:00Z',
  finishedAt: '2026-10-01T06:30:00Z',
};
const SCORPION: MineResult = {
  ...SAB,
  seriesId: 's2',
  title: 'Scorpion',
  episodesTotal: 22,
  episodesUsed: 0,
  fansubSkipped: 6,
  termsFound: 0,
  termsInserted: 0,
};

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <OfficialSubtitleMiningCard />
    </QueryClientProvider>
  );
}

describe('OfficialSubtitleMiningCard — copy rules (C9-D note ①–④)', () => {
  it('① never run', () => {
    expect(statusLine(undefined)).toBe('還沒跑過');
    expect(statusLine({ running: false, results: [] })).toBe('還沒跑過');
  });

  it('③ one sentence per show, joined by 、; fan-group skips are said out loud', () => {
    const line = statusLine({
      running: false,
      lastRunAt: '2026-10-01T06:30:00Z',
      results: [SAB, SCORPION],
    });
    expect(line).toContain('Shadow and Bone 學到 68 個詞（8 集）、Scorpion 跳過 6 個字幕組檔');
    expect(line.startsWith(`上次 ${formatRunTime('2026-10-01T06:30:00Z')} · `)).toBe(true);
  });

  it('④ a run that found no partial show says so', () => {
    expect(statusLine({ running: false, lastRunAt: '2026-10-01T06:30:00Z', results: [] })).toMatch(
      /片庫裡沒有「部分集有官方字幕」的影集$/
    );
  });

  it('a show that learned AND skipped says both; a failed show says 沒學成', () => {
    expect(describeResult({ ...SAB, fansubSkipped: 2 })).toBe(
      'Shadow and Bone 學到 68 個詞（8 集），跳過 2 個字幕組檔'
    );
    expect(describeResult({ ...SAB, error: 'boom' })).toBe('Shadow and Bone 沒學成');
    expect(describeResult({ ...SAB, termsReplaced: 1 })).toContain('其中 1 個修正了聽寫猜的寫法');
    expect(describeResult({ ...SAB, termsPruned: 20 })).toContain('清掉 20 個聽寫猜錯的詞');
    expect(describeResult({ ...SAB, termsSplit: 11 })).toContain('其中 11 個詞各季寫法不同');
    expect(describeResult({ ...SAB, episodesUsed: 0 })).toBe('Shadow and Bone 沒有可用的集');
  });

  it('formats in local time as YYYY-MM-DD HH:mm, and refuses garbage', () => {
    const iso = new Date(2026, 9, 1, 14, 30).toISOString();
    expect(formatRunTime(iso)).toBe('2026-10-01 14:30');
    expect(formatRunTime('nope')).toBe('');
  });
});

describe('OfficialSubtitleMiningCard — behaviour', () => {
  beforeEach(() => {
    h.start.mockReset();
  });

  it('idle: the button starts a sweep and the line shows the last run', async () => {
    h.status = { running: false, lastRunAt: '2026-10-01T06:30:00Z', results: [SAB] };
    h.start.mockResolvedValue({ started: true });
    renderCard();
    await waitFor(() =>
      expect(screen.getByTestId('official-subtitle-mining-status')).toHaveTextContent(
        'Shadow and Bone 學到 68 個詞（8 集）'
      )
    );
    const btn = screen.getByTestId('official-subtitle-mining-start');
    expect(btn).toHaveTextContent('重新從官方字幕學習');
    expect(btn).toBeEnabled();
    // Secondary on desktop (C9-D, $0 — never ButtonCost); the gold fill is
    // phone-only (C9-M's Touch button), so every accent class is max-sm: scoped.
    expect(btn.className).toContain('bg-[var(--bg-tertiary)]');
    expect(btn.className).toContain('max-sm:bg-[var(--accent-primary)]');
    expect(btn.className).not.toMatch(/(^|\s)bg-\[var\(--accent-primary\)\]/);
    expect(btn.querySelector('[data-cost-status]')).toBeNull();
    fireEvent.click(btn);
    await waitFor(() => expect(h.start).toHaveBeenCalledTimes(1));
  });

  it('② running: button disabled with 學習中…, and the line explains it costs nothing', async () => {
    h.status = { running: true, runningFor: 'partial', results: [] };
    renderCard();
    await waitFor(() =>
      expect(screen.getByTestId('official-subtitle-mining-start')).toHaveTextContent('學習中…')
    );
    expect(screen.getByTestId('official-subtitle-mining-start')).toBeDisabled();
    expect(screen.getByTestId('official-subtitle-mining-status')).toHaveTextContent('不花錢');
  });

  it('a refused start (already running) surfaces the server message', async () => {
    h.status = { running: false, results: [] };
    h.start.mockRejectedValue(new Error('正在從官方字幕學習中。'));
    renderCard();
    fireEvent.click(await screen.findByTestId('official-subtitle-mining-start'));
    expect(await screen.findByTestId('official-subtitle-mining-error')).toHaveTextContent(
      '正在從官方字幕學習中。'
    );
  });
});
